//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	effectpostgres "tei/internal/security/effectcommit/postgres"
	"tei/internal/security/legacyauthorization"
	"tei/internal/workspace"
)

// Real PostgreSQL tests for runnerOrderSubmitStore. Gated by
// TEI_TEST_POSTGRES_DSN. The authority table, receipts table, lock_authority
// and apply_owner_effect below are TEST-ONLY DDL installed in a uniquely named
// disposable schema and dropped afterwards; they are not an Authority and the
// grant is synthetic. A green run is a local reference result, not proof of
// production atomicity or durability.

const runnerTestSchema = "tei_wf_g1_runner"

func runnerTestDDL() string {
	s := runnerTestSchema
	return `
CREATE SCHEMA ` + s + `;
CREATE TABLE ` + s + `.authority (
  application text NOT NULL, capability text NOT NULL, subject text NOT NULL, action text NOT NULL, resource text NOT NULL,
  source text NOT NULL, subject_link bigint NOT NULL, membership bigint NOT NULL, policy bigint NOT NULL, enabled boolean NOT NULL,
  PRIMARY KEY (application,capability,subject,action,resource));
CREATE TABLE ` + s + `.receipts (
  operation_id text PRIMARY KEY, application text NOT NULL, capability text NOT NULL, subject text NOT NULL, action text NOT NULL,
  resource text NOT NULL, request_hash text NOT NULL, resource_version bigint NOT NULL);
INSERT INTO ` + s + `.authority VALUES ('order_entry','order_entry.submit_order','legacy-order-editor','execute','order_entry.submit_order','test-authority',1,1,1,TRUE);
CREATE FUNCTION ` + s + `.lock_authority(p_application text, p_capability text, p_subject text, p_action text, p_resource text,
  p_source text, p_subject_link bigint, p_membership bigint, p_policy bigint)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $f$
DECLARE v_source text; v_link bigint; v_m bigint; v_p bigint; v_e boolean;
BEGIN
  SELECT source,subject_link,membership,policy,enabled INTO v_source,v_link,v_m,v_p,v_e FROM ` + s + `.authority
   WHERE application=p_application AND capability=p_capability AND subject=p_subject AND action=p_action AND resource=p_resource FOR UPDATE;
  IF NOT FOUND THEN RETURN FALSE; END IF;
  RETURN v_e AND v_source=p_source AND v_link=p_subject_link AND v_m=p_membership AND v_p=p_policy;
END $f$;
CREATE FUNCTION ` + s + `.apply_owner_effect(p_application text, p_resource text, p_expected bigint, p_payload bytea, p_operation text)
RETURNS bigint LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, pg_temp AS $f$
DECLARE v_order bigint; v_doc jsonb;
BEGIN
  BEGIN
    v_doc := convert_from(p_payload,'UTF8')::jsonb;
    v_order := (v_doc->>'orderId')::bigint;
  EXCEPTION WHEN others THEN RETURN -2;
  END;
  IF v_order IS NULL OR v_doc->>'to' <> 'submitted' THEN RETURN -2; END IF;
  IF p_expected <> 0 THEN RETURN 0; END IF;
  UPDATE public.orders SET status='submitted' WHERE order_id=v_order AND status='draft';
  IF NOT FOUND THEN RETURN -1; END IF;
  RETURN 1;
END $f$;`
}

type runnerPGEnv struct {
	dsn   string
	db    *sql.DB
	world ir.World
}

func newRunnerPGEnv(t *testing.T) *runnerPGEnv {
	t.Helper()
	dsn := orderSubmitTestDSN(t)
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	db := openDeclaredOrderFixtureDB(t, dsn, loaded.World)
	if _, err := db.Exec(`DROP SCHEMA IF EXISTS ` + runnerTestSchema + ` CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(runnerTestDDL()); err != nil {
		t.Fatalf("test DDL: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DROP SCHEMA IF EXISTS ` + runnerTestSchema + ` CASCADE`) })
	return &runnerPGEnv{dsn: dsn, db: db, world: loaded.World}
}

func (e *runnerPGEnv) store(t *testing.T) *runnerOrderSubmitStore {
	t.Helper()
	s, err := newPostgresRunnerOrderSubmitStore(context.Background(), e.dsn, e.dsn, runnerTestSchema, e.world,
		func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
			return syntheticSubmitGrant(), nil
		})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func (e *runnerPGEnv) receipts(t *testing.T) int {
	t.Helper()
	var n int
	if err := e.db.QueryRow(`SELECT count(*) FROM ` + runnerTestSchema + `.receipts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRunnerOrderSubmitRealPostgresCommitsWithOneReceipt(t *testing.T) {
	env := newRunnerPGEnv(t)
	store := env.store(t)
	result, err := newOrderSubmitRunner(t, store).RunCommand(context.Background(), orderSubmitInput)
	if err != nil || result.Stage != "completed" || result.Observation == nil || result.Observation.Result != "submitted" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if got := orderStatus(t, env.db, 1001); got != "submitted" {
		t.Fatalf("status=%q", got)
	}
	if got := orderStatus(t, env.db, 1002); got != "draft" {
		t.Fatalf("1002=%q", got)
	}
	if env.receipts(t) != 1 {
		t.Fatalf("receipts=%d", env.receipts(t))
	}
	// A second submit is refused (Replayed -> not submittable); row and receipts unchanged.
	if err := store.Submit(context.Background(), 1001); !errors.Is(err, errOrderNotSubmittable) {
		t.Fatalf("second submit err=%v", err)
	}
	if orderStatus(t, env.db, 1001) != "submitted" || env.receipts(t) != 1 {
		t.Fatal("second submit changed state")
	}
	// Same OperationID, different payload: Conflict.
	grant := syntheticSubmitGrant()
	req, _ := runnerSubmitRequest(grant, 1001)
	req.Payload = []byte(`{"orderId":1001,"to":"cancelled"}`)
	res, err := store.runner.Execute(context.Background(), req, grant)
	if err != nil || res.Status != effectcommit.StatusConflict || env.receipts(t) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestRunnerOrderSubmitRealPostgresAlreadyNonDraftIsRefusedWithoutReceipt(t *testing.T) {
	env := newRunnerPGEnv(t)
	if _, err := env.db.Exec(`UPDATE public.orders SET status='cancelled' WHERE order_id=1001`); err != nil {
		t.Fatal(err)
	}
	err := env.store(t).Submit(context.Background(), 1001)
	if !errors.Is(err, errOrderNotSubmittable) || orderStatus(t, env.db, 1001) != "cancelled" || env.receipts(t) != 0 {
		t.Fatalf("err=%v", err)
	}
}

func TestRunnerOrderSubmitRealPostgresAuthorityBumpOrRevokeLeavesDraft(t *testing.T) {
	for name, stmt := range map[string]string{
		"epoch-bump": `UPDATE ` + runnerTestSchema + `.authority SET policy=policy+1`,
		"revoked":    `UPDATE ` + runnerTestSchema + `.authority SET enabled=FALSE`,
	} {
		t.Run(name, func(t *testing.T) {
			env := newRunnerPGEnv(t)
			if _, err := env.db.Exec(stmt); err != nil {
				t.Fatal(err)
			}
			err := env.store(t).Submit(context.Background(), 1001)
			if !errors.Is(err, errOrderNotSubmittable) || orderStatus(t, env.db, 1001) != "draft" || env.receipts(t) != 0 {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

// ackLossProvider performs the real Commit and then reports a failure.
type ackLossProvider struct{ inner effectcommit.Provider }

type ackLossTx struct{ effectcommit.Transaction }

func (p ackLossProvider) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	tx, err := p.inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return ackLossTx{tx}, nil
}

func (t ackLossTx) Commit(ctx context.Context) error {
	if err := t.Transaction.Commit(ctx); err != nil {
		return err
	}
	return errors.New("postgres://lost-ack-secret@internal-db: connection reset")
}

func TestRunnerOrderSubmitRealPostgresUnknownCommitThenLookupReplayed(t *testing.T) {
	env := newRunnerPGEnv(t)
	pool, err := pgxpool.New(context.Background(), env.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	provider, err := effectpostgres.NewWithTrustedOwnerRoutine(pool, runnerTestSchema)
	if err != nil {
		t.Fatal(err)
	}
	lossy, err := newRunnerOrderSubmitStore(effectcommit.Runner{Provider: ackLossProvider{provider}},
		func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
			return syntheticSubmitGrant(), nil
		},
		nil, &statusReader{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := newOrderSubmitRunner(t, lossy).RunCommand(context.Background(), orderSubmitInput)
	if !errors.Is(err, errOrderSubmitCommitUnconfirmed) || result.Stage != "command-commit-unconfirmed" ||
		strings.Contains(err.Error(), "lost-ack-secret") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	// The write really committed; recovery uses the SAME OperationID.
	if orderStatus(t, env.db, 1001) != "submitted" || env.receipts(t) != 1 {
		t.Fatal("commit not durable")
	}
	look, err := env.store(t).LookupSubmit(context.Background(), 1001)
	if err != nil || look.Status != effectcommit.StatusReplayed || look.Receipt.OperationID != "order-submit/1001" {
		t.Fatalf("lookup=%+v err=%v", look, err)
	}
	// An order never submitted is NotRecorded.
	miss, err := env.store(t).LookupSubmit(context.Background(), 1002)
	if err != nil || miss.Status != effectcommit.StatusNotRecorded {
		t.Fatalf("miss=%+v err=%v", miss, err)
	}
}

func TestRunnerOrderSubmitRealPostgresSchemaNotReadyRefusesBeforeWrite(t *testing.T) {
	env := newRunnerPGEnv(t)
	store := env.store(t)
	if _, err := env.db.Exec(`ALTER TABLE public.orders ADD COLUMN extra integer`); err != nil {
		t.Fatal(err)
	}
	err := store.Submit(context.Background(), 1001)
	if !errors.Is(err, errPhysicalSchemaNotReady) || orderStatus(t, env.db, 1001) != "draft" || env.receipts(t) != 0 {
		t.Fatalf("err=%v", err)
	}
}

func TestRunnerOrderSubmitRealPostgresHostGrantDeniedLeavesDraft(t *testing.T) {
	env := newRunnerPGEnv(t)
	store := env.store(t)
	cr := &countingRunner{inner: store.runner}
	store.runner = cr
	runner := newOrderSubmitRunner(t, store)
	runner.hostEffect = func(context.Context, string) error { return errors.New("revoked") }
	result, err := runner.RunCommand(context.Background(), orderSubmitInput)
	if err == nil || result.Stage != "host-effect-grant-denied" || cr.executes != 0 ||
		orderStatus(t, env.db, 1001) != "draft" || env.receipts(t) != 0 {
		t.Fatalf("result=%+v err=%v executes=%d", result, err, cr.executes)
	}
}
