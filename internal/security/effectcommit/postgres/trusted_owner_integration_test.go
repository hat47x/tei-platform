package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
)

// The trusted function exists only in the isolated PostgreSQL test schema.
// Business rules are NOT embedded in effectcommit or resolved from a Plugin.
func installTrustedOwnerFixture(t *testing.T, f *postgresFixture) {
	t.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		"CREATE TABLE " + f.schema + ".shared_inventory (bucket TEXT PRIMARY KEY, available INTEGER NOT NULL CHECK (available >= 0))",
		"INSERT INTO " + f.schema + ".shared_inventory VALUES ('shared',3)",
		"CREATE TABLE " + f.schema + ".owner_reservations (resource TEXT PRIMARY KEY, units INTEGER NOT NULL)",
		"CREATE TABLE " + f.schema + ".owner_history (operation_id TEXT PRIMARY KEY, resource TEXT NOT NULL)",
		"CREATE TABLE " + f.schema + ".owner_outbox (operation_id TEXT PRIMARY KEY, resource TEXT NOT NULL)",
	} {
		if _, err := f.pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	// The routine is a fixed, schema-qualified DB-owner installation. Its
	// resource CAS, shared-owner reservation, history and DB outbox are part
	// of the pgx.Tx that holds the authority lock and later saves receipt.
	ddl := fmt.Sprintf(`CREATE FUNCTION %[1]s.apply_owner_effect(
		p_application TEXT, p_resource TEXT, p_expected BIGINT, p_payload BYTEA, p_operation TEXT)
		RETURNS BIGINT LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, pg_temp AS $owner$
		DECLARE v_version BIGINT; v_units INTEGER;
		BEGIN
			v_units := convert_from(p_payload, 'UTF8')::INTEGER;
			IF v_units <= 0 THEN RETURN -2; END IF;
			UPDATE %[1]s.objects SET value=p_payload, version=version+1
				WHERE application=p_application AND resource=p_resource AND version=p_expected
				RETURNING version INTO v_version;
			IF NOT FOUND THEN RETURN 0; END IF;
			UPDATE %[1]s.shared_inventory SET available=available-v_units
				WHERE bucket='shared' AND available >= v_units;
			IF NOT FOUND THEN RETURN -1; END IF;
			INSERT INTO %[1]s.owner_reservations(resource,units) VALUES (p_resource,v_units);
			INSERT INTO %[1]s.owner_history(operation_id,resource) VALUES (p_operation,p_resource);
			INSERT INTO %[1]s.owner_outbox(operation_id,resource) VALUES (p_operation,p_resource);
			RETURN v_version;
		END $owner$`, f.schema)
	if _, err := f.pool.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
}

func ownerCounts(t *testing.T, f *postgresFixture) (available, reservations, history, outbox, receipts int) {
	t.Helper()
	ctx := context.Background()
	for i, query := range []string{
		"SELECT available FROM " + f.schema + ".shared_inventory WHERE bucket='shared'",
		"SELECT count(*) FROM " + f.schema + ".owner_reservations",
		"SELECT count(*) FROM " + f.schema + ".owner_history",
		"SELECT count(*) FROM " + f.schema + ".owner_outbox",
		"SELECT count(*) FROM " + f.schema + ".receipts",
	} {
		var v int
		if err := f.pool.QueryRow(ctx, query).Scan(&v); err != nil {
			t.Fatal(err)
		}
		switch i {
		case 0:
			available = v
		case 1:
			reservations = v
		case 2:
			history = v
		case 3:
			outbox = v
		case 4:
			receipts = v
		}
	}
	return
}

func TestTrustedOwnerRoutineConcurrentSharedInventory(t *testing.T) {
	f := openFixture(t)
	installTrustedOwnerFixture(t, f)
	ctx := context.Background()
	first := f.req
	first.Payload = []byte("3")
	grantFirst := f.grant(t)
	second := first
	second.Scope.Resource = "order-2"
	second.OperationID = "operation-2"
	if _, err := f.pool.Exec(ctx, "INSERT INTO "+f.schema+".objects(application,resource,version,value) VALUES($1,$2,0,'original')",
		second.Scope.Application, second.Scope.Resource); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, "INSERT INTO "+f.schema+".authority(application,capability,subject,action,resource,source,subject_link,membership,policy,enabled) "+
		"VALUES($1,$2,$3,$4,$5,'postgres-authority',1,1,1,TRUE)",
		second.Scope.Application, second.Scope.Capability, second.Scope.Subject,
		second.Scope.Action, second.Scope.Resource); err != nil {
		t.Fatal(err)
	}
	f.req = second
	grantSecond := f.grant(t)
	f.req = first
	p, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	runner := effectcommit.Runner{Provider: p}
	type attempt struct {
		result effectcommit.Result
		err    error
	}
	var wg sync.WaitGroup
	results := make(chan attempt, 2)
	begin := make(chan struct{})
	for _, item := range []struct {
		req   effectcommit.Request
		grant legacyauthorization.AuthorizationGrant
	}{{first, grantFirst}, {second, grantSecond}} {
		wg.Add(1)
		go func(req effectcommit.Request, grant legacyauthorization.AuthorizationGrant) {
			defer wg.Done()
			<-begin
			r, err := runner.Execute(ctx, req, grant)
			results <- attempt{r, err}
		}(item.req, item.grant)
	}
	close(begin)
	wg.Wait()
	close(results)
	committed, denied := 0, 0
	for x := range results {
		if x.err != nil {
			t.Fatalf("concurrent transaction error: %v", x.err)
		}
		switch x.result.Status {
		case effectcommit.StatusCommitted:
			committed++
		case effectcommit.StatusDenied:
			denied++
		default:
			t.Fatalf("unexpected outcome: %+v", x.result)
		}
	}
	if committed != 1 || denied != 1 {
		t.Fatalf("owner oversubscribed: committed=%d denied=%d", committed, denied)
	}
	a, reservations, history, outbox, receipts := ownerCounts(t, f)
	if a != 0 || reservations != 1 || history != 1 || outbox != 1 || receipts != 1 {
		t.Fatalf("partial or duplicate commit: available=%d reservations=%d history=%d outbox=%d receipts=%d",
			a, reservations, history, outbox, receipts)
	}
	// Exactly one object transitioned; the loser retained its old version.
	var advanced int
	if err := f.pool.QueryRow(ctx, "SELECT count(*) FROM "+f.schema+".objects WHERE version=1").Scan(&advanced); err != nil {
		t.Fatal(err)
	}
	if advanced != 1 {
		t.Fatalf("resource CAS not atomic with owner effects: advanced=%d", advanced)
	}
}

func TestTrustedOwnerRoutineReceiptFailureRollsBackAndMissingRoutineFailsClosed(t *testing.T) {
	f := openFixture(t)
	installTrustedOwnerFixture(t, f)
	ctx := context.Background()
	req := f.req
	req.Payload = []byte("3")
	grant := f.grant(t)
	p, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	runner := effectcommit.Runner{Provider: p}

	if _, err := f.pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s.reject_receipt() RETURNS trigger LANGUAGE plpgsql AS $body$
		BEGIN RAISE EXCEPTION 'receipt persistence failure'; END $body$`, f.schema)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, "CREATE TRIGGER reject_receipt BEFORE INSERT ON "+f.schema+
		".receipts FOR EACH ROW EXECUTE FUNCTION "+f.schema+".reject_receipt()"); err != nil {
		t.Fatal(err)
	}
	if result, err := runner.Execute(ctx, req, grant); err == nil || result != (effectcommit.Result{}) {
		t.Fatalf("receipt persistence failure committed: %+v err=%v", result, err)
	}
	a, reservations, history, outbox, receipts := ownerCounts(t, f)
	value, version := f.readObject(t)
	if a != 3 || reservations != 0 || history != 0 || outbox != 0 || receipts != 0 ||
		value != "original" || version != 0 {
		t.Fatalf("owner effect survived receipt rollback: available=%d reservations=%d history=%d outbox=%d receipts=%d object=%s version=%d",
			a, reservations, history, outbox, receipts, value, version)
	}
	if _, err := f.pool.Exec(ctx, "DROP TRIGGER reject_receipt ON "+f.schema+".receipts"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, "DROP FUNCTION "+f.schema+".apply_owner_effect(TEXT,TEXT,BIGINT,BYTEA,TEXT)"); err != nil {
		t.Fatal(err)
	}
	if result, err := runner.Execute(ctx, req, grant); err == nil || result != (effectcommit.Result{}) {
		t.Fatalf("missing trusted routine was not an execution error: %+v err=%v", result, err)
	}
	a, reservations, history, outbox, receipts = ownerCounts(t, f)
	if a != 3 || reservations != 0 || history != 0 || outbox != 0 || receipts != 0 {
		t.Fatalf("missing routine mutated owner: %d %d %d %d %d", a, reservations, history, outbox, receipts)
	}
}

func TestTrustedOwnerRoutineDistinguishesVersionConflictOwnerDenialAndInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name            string
		expectedVersion uint64
		units           string
		available       int
		wantStatus      effectcommit.Status
		wantInvalid     bool
		wantCommitted   bool
	}{
		{name: "matching-version-stock-shortage", expectedVersion: 0, units: "3", available: 0, wantStatus: effectcommit.StatusDenied},
		{name: "stale-version-stock-available", expectedVersion: 1, units: "3", available: 3, wantStatus: effectcommit.StatusConflict},
		{name: "matching-version-invalid-units", expectedVersion: 0, units: "0", available: 3, wantInvalid: true},
		{name: "matching-version-stock-available", expectedVersion: 0, units: "3", available: 3, wantStatus: effectcommit.StatusCommitted, wantCommitted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := openFixture(t)
			installTrustedOwnerFixture(t, f)
			ctx := context.Background()
			if _, err := f.pool.Exec(ctx, "UPDATE "+f.schema+".shared_inventory SET available=$1 WHERE bucket='shared'", tc.available); err != nil {
				t.Fatal(err)
			}
			req := f.req
			req.ExpectedVersion = tc.expectedVersion
			req.Payload = []byte(tc.units)
			grant := f.grant(t)
			provider, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
			if err != nil {
				t.Fatal(err)
			}
			got, err := (effectcommit.Runner{Provider: provider}).Execute(ctx, req, grant)
			if tc.wantInvalid {
				if !errors.Is(err, effectcommit.ErrInvalidRequest) || got != (effectcommit.Result{}) {
					t.Fatalf("invalid owner input wrongly mapped to conflict or denial: %+v err=%v", got, err)
				}
			} else if err != nil || got.Status != tc.wantStatus {
				t.Fatalf("unexpected classification for version=%d available=%d units=%s: %+v err=%v",
					tc.expectedVersion, tc.available, tc.units, got, err)
			}
			available, reservations, history, outbox, receipts := ownerCounts(t, f)
			value, version := f.readObject(t)
			if tc.wantCommitted {
				if got.Receipt.ResourceVersion != 1 || available != 0 ||
					reservations != 1 || history != 1 || outbox != 1 || receipts != 1 ||
					value != "3" || version != 1 {
					t.Fatalf("normal commit lost effects: result=%+v, inventory=%d, owner=%d/%d/%d, receipts=%d, object=%q version=%d",
						got, available, reservations, history, outbox, receipts, value, version)
				}
				return
			}
			if got.Receipt != (effectcommit.Receipt{}) || available != tc.available ||
				reservations != 0 || history != 0 || outbox != 0 || receipts != 0 ||
				value != "original" || version != 0 {
				t.Fatalf("non-commit changed DB: result=%+v, inventory=%d, owner=%d/%d/%d, receipts=%d, object=%q version=%d",
					got, available, reservations, history, outbox, receipts, value, version)
			}
		})
	}
}
