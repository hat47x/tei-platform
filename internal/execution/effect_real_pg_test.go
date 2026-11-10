package execution_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/effectlog"
	"tei/internal/execution"
	"tei/internal/runlog"
	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/commandadapter"
	"tei/internal/security/effectcommit/postgres"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
)

// These tests connect execution.RunEffect, Compensate and RunPlan to a real
// PostgreSQL Command: commandadapter.Executor over effectcommit.Runner and the
// trusted-owner routine provider. They add no execution API and re-implement no
// commit logic; the only new code is the test-side adapter that maps a capability
// to one fixed Command and an observer that reads the data back over a second
// connection pool.
//
// They run only when a PostgreSQL DSN is supplied through
// TEI_EFFECT_POSTGRES_DSN (or TEI_TEST_POSTGRES_DSN). Without it they SKIP, and a
// SKIP is not evidence. The schema is throwaway and the Owner routine exists only
// in it, so a pass does not qualify a production Provider, the Host's current
// Grant, or any real Provider receipt.

const (
	rpgApplication = "sales"
	rpgSubject     = "editor"
	rpgAction      = "write"
)

// rpgRoute is one fixed Command: the Owner effect a capability performs.
type rpgRoute struct {
	capability core.CanonicalID
	resource   string
	operation  string
	expected   uint64
	kind       string
	units      int
}

func (r rpgRoute) payload() []byte { return []byte(fmt.Sprintf("%s:%d", r.kind, r.units)) }

var rpgRoutes = []rpgRoute{
	{"sales.stock.reserve_a", "order-a", "op-reserve-a", 0, "reserve", 2},
	{"sales.stock.reserve_b", "order-b", "op-reserve-b", 0, "reserve", 5},
	{"sales.stock.reserve_d", "order-d", "op-reserve-d", 0, "reserve", 1},
	{"sales.stock.release_a", "order-a", "op-release-a", 1, "release", 2},
	{"sales.stock.release_c", "order-c", "op-release-c", 0, "release", 1},
	{"sales.audit.note_b", "order-b", "op-note-b", 0, "note", 0},
}

func rpgRule(id, onCapability, stage, via, onFailure string) semantic.EffectRule {
	return semantic.EffectRule{
		ID: core.CanonicalID(id), OnCapability: core.CanonicalID(onCapability), Stage: stage,
		EffectKind: "update", EffectData: "sales.stock", ViaCapability: core.CanonicalID(via), OnFailure: onFailure,
	}
}

// rpgRules declares the effects. The compensations answer a failure of
// fulfil_b; undo_a restores what fulfil_a reserved, undo_c cannot succeed (nothing
// was reserved on order-c), and audit_b only writes a note.
func rpgRules() []semantic.EffectRule {
	return []semantic.EffectRule{
		rpgRule("sales.fulfil.reserve_a", "sales.order.fulfil_a", "after-commit", "sales.stock.reserve_a", "reject"),
		rpgRule("sales.fulfil.reserve_b", "sales.order.fulfil_b", "after-commit", "sales.stock.reserve_b", "compensate"),
		rpgRule("sales.fulfil.reserve_d", "sales.order.fulfil_d", "after-commit", "sales.stock.reserve_d", "reject"),
		rpgRule("sales.fulfil.undo_a", "sales.order.fulfil_b", "compensation", "sales.stock.release_a", ""),
		rpgRule("sales.fulfil.undo_c", "sales.order.fulfil_b", "compensation", "sales.stock.release_c", ""),
		rpgRule("sales.fulfil.audit_b", "sales.order.fulfil_b", "compensation", "sales.audit.note_b", ""),
	}
}

func rpgRuleByID(rules []semantic.EffectRule, id string) semantic.EffectRule {
	for _, rule := range rules {
		if string(rule.ID) == id {
			return rule
		}
	}
	panic("unknown effect rule " + id)
}

// rpgInvoker is the execution.Invoker adapter: it runs the Command that the
// capability names and reports a business denial or version conflict as an error,
// which is what commandadapter.Executor does. It adds no Command logic.
type rpgInvoker struct {
	provider postgres.Provider
	grants   map[core.CanonicalID]legacyauthorization.AuthorizationGrant
	routes   map[core.CanonicalID]rpgRoute
	// after runs once the Command has returned, from outside its transaction.
	after map[core.CanonicalID]func()
	calls []core.CanonicalID
}

func (i *rpgInvoker) InvokeCapability(ctx context.Context, applicationID, capabilityID core.CanonicalID, _ driver.BindingAdapterResolver, _ any) (driver.CapabilityInvocation, error) {
	i.calls = append(i.calls, capabilityID)
	route, ok := i.routes[capabilityID]
	if !ok {
		return driver.CapabilityInvocation{}, fmt.Errorf("no Command is bound to %s", capabilityID)
	}
	executor := commandadapter.Executor{
		Runner:          effectcommit.Runner{Provider: i.provider},
		Grant:           i.grants[capabilityID],
		OperationID:     route.operation,
		ExpectedVersion: route.expected,
		Payload:         route.payload(),
		Effect:          &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.stock"},
	}
	_, err := executor.Execute(ctx, command.Request{Application: applicationID, Capability: capabilityID})
	if hook := i.after[capabilityID]; hook != nil {
		hook()
	}
	return driver.CapabilityInvocation{}, err
}

// rpgPIDs is a concurrency-safe set of PostgreSQL backend pids.
type rpgPIDs struct {
	mu   sync.Mutex
	seen map[int]bool
}

func (p *rpgPIDs) add(pid int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.seen == nil {
		p.seen = map[int]bool{}
	}
	p.seen[pid] = true
}

func (p *rpgPIDs) snapshot() map[int]bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := map[int]bool{}
	for pid := range p.seen {
		out[pid] = true
	}
	return out
}

// rpgWriteTracer records the backend pid of every session that runs the Owner
// routine, that is, of the connections the Command actually wrote through.
type rpgWriteTracer struct{ pids *rpgPIDs }

func (t rpgWriteTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if strings.Contains(data.SQL, "apply_owner_effect") {
		t.pids.add(int(conn.PgConn().PID()))
	}
	return ctx
}

func (rpgWriteTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// rpgQueryer runs each observer query on one acquired connection and records that
// connection's pg_backend_pid(), so a test can assert which session the Observer
// really read through.
type rpgQueryer struct {
	pool *pgxpool.Pool
	pids *rpgPIDs
}

func (q rpgQueryer) QueryRowContext(ctx context.Context, query string, args ...any) execution.RowScanner {
	conn, err := q.pool.Acquire(ctx)
	if err != nil {
		return rpgErrRow{err}
	}
	defer conn.Release()
	var pid int
	if err := conn.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		return rpgErrRow{err}
	}
	q.pids.add(pid)
	// Scan the row before the connection goes back to the pool.
	var count int64
	if err := conn.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return rpgErrRow{err}
	}
	return rpgCountRow{count}
}

type rpgErrRow struct{ err error }

func (r rpgErrRow) Scan(...any) error { return r.err }

type rpgCountRow struct{ count int64 }

func (r rpgCountRow) Scan(dest ...any) error {
	if len(dest) != 1 {
		return errors.New("rpgCountRow: one destination expected")
	}
	p, ok := dest[0].(*int64)
	if !ok {
		return errors.New("rpgCountRow: *int64 expected")
	}
	*p = r.count
	return nil
}

// rpgObserver is the execution.EffectObserver: it reads the Owner tables over a
// pool that is separate from the one the Command used. It checks only the rules it
// has a check for; Covers reports which.
//
// The product has no Checks type. "A required check is missing" is modelled here,
// in the test, by Covers and the coverage precondition below. The product's
// EffectPreconditions only tests whether the observer is nil, so blocking on a
// missing check is NOT a product guarantee; this test only shows that RunPlan
// blocks, and does not run the step, when a precondition reports a reason.
type rpgObserver struct {
	inner  execution.SQLEffectObserver
	checks map[string]rpgRoute
}

func (o rpgObserver) ObserveEffect(ctx context.Context, record effectlog.Record) (bool, string, error) {
	return o.inner.ObserveEffect(ctx, record)
}

func (o rpgObserver) Covers(ruleID string) bool { _, ok := o.checks[ruleID]; return ok }

type rpgHarness struct {
	pool         *pgxpool.Pool
	observerPool *pgxpool.Pool
	writePIDs    *rpgPIDs // sessions that ran the Command's Owner routine
	readPIDs     *rpgPIDs // sessions the Observer read through
	schema       string
	invoker      *rpgInvoker
	observer     rpgObserver
	effects      effectlog.File
	runs         runlog.File
	rules        []semantic.EffectRule
	routeByRule  map[string]rpgRoute
}

func rpgDSN() string {
	if dsn := os.Getenv("TEI_EFFECT_POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return os.Getenv("TEI_TEST_POSTGRES_DSN")
}

type rpgSubjects struct{ h *rpgHarness }

func (s rpgSubjects) FindBySubject(ctx context.Context, subject string) (legacyauth.User, bool, error) {
	var user legacyauth.User
	err := s.h.pool.QueryRow(ctx, "SELECT subject,role,disabled FROM "+s.h.schema+".subjects WHERE subject=$1", subject).
		Scan(&user.SubjectID, &user.RoleCode, &user.Disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return legacyauth.User{}, false, nil
	}
	return user, err == nil, err
}

type rpgEpochs struct {
	h          *rpgHarness
	capability string
}

func (e rpgEpochs) ReadAuthorizationEpochs(ctx context.Context, input legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
	var out legacyauthorization.AuthorityEpochs
	var link, membership, policy int64
	var enabled bool
	err := e.h.pool.QueryRow(ctx, "SELECT source,subject_link,membership,policy,enabled FROM "+e.h.schema+
		".authority WHERE subject=$1 AND action=$2 AND resource=$3 AND application=$4 AND capability=$5",
		input.SubjectID, input.Action, input.Resource, rpgApplication, e.capability).
		Scan(&out.Source, &link, &membership, &policy, &enabled)
	if err != nil {
		return legacyauthorization.AuthorityEpochs{}, err
	}
	if !enabled {
		return legacyauthorization.AuthorityEpochs{}, legacyauthorization.ErrAuthorityInactive
	}
	out.SubjectLink, out.Membership, out.Policy = uint64(link), uint64(membership), uint64(policy)
	return out, nil
}

type rpgRole struct{}

func (rpgRole) Evaluate(_ context.Context, user legacyauth.User, _, _ string) (bool, string, error) {
	return user.RoleCode == "EDITOR", "role_denied", nil
}

// rpgOpen builds a throwaway schema with the authority table, the receipts table
// and a trusted-owner routine, and wires the Invoker and the Observer to it.
func rpgOpen(t *testing.T) *rpgHarness {
	t.Helper()
	dsn := rpgDSN()
	if dsn == "" {
		t.Skip("実PostgreSQLの検証はTEI_EFFECT_POSTGRES_DSN(またはTEI_TEST_POSTGRES_DSN)がある場合のみ実行。SKIPは証拠にならない")
	}
	ctx := context.Background()
	writePIDs, readPIDs := &rpgPIDs{}, &rpgPIDs{}
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	poolConfig.ConnConfig.Tracer = rpgWriteTracer{writePIDs}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	observerPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observerPool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("PostgreSQLに接続できません: %v", err)
	}
	hash := sha256.Sum256([]byte(t.Name()))
	schema := "tei_exec_" + hex.EncodeToString(hash[:9])
	if _, err := pool.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
			t.Errorf("試験スキーマの削除に失敗: %v", err)
		}
	})
	routine := fmt.Sprintf(`CREATE FUNCTION %[1]s.apply_owner_effect(
		p_application TEXT, p_resource TEXT, p_expected BIGINT, p_payload BYTEA, p_operation TEXT)
		RETURNS BIGINT LANGUAGE plpgsql SECURITY INVOKER SET search_path = pg_catalog, pg_temp AS $owner$
		DECLARE v_version BIGINT; v_kind TEXT; v_units INTEGER;
		BEGIN
			v_kind := split_part(convert_from(p_payload, 'UTF8'), ':', 1);
			v_units := split_part(convert_from(p_payload, 'UTF8'), ':', 2)::INTEGER;
			IF v_kind NOT IN ('reserve','release','note') THEN RETURN -2; END IF;
			IF v_kind <> 'note' AND v_units <= 0 THEN RETURN -2; END IF;
			UPDATE %[1]s.objects SET value=p_payload, version=version+1
				WHERE application=p_application AND resource=p_resource AND version=p_expected
				RETURNING version INTO v_version;
			IF NOT FOUND THEN RETURN 0; END IF;
			IF v_kind = 'reserve' THEN
				UPDATE %[1]s.inventory SET available=available-v_units WHERE bucket='shared' AND available >= v_units;
				IF NOT FOUND THEN RETURN -1; END IF;
				INSERT INTO %[1]s.reservations(resource,units) VALUES (p_resource,v_units);
			ELSIF v_kind = 'release' THEN
				DELETE FROM %[1]s.reservations WHERE resource=p_resource AND units=v_units;
				IF NOT FOUND THEN RETURN -1; END IF;
				UPDATE %[1]s.inventory SET available=available+v_units WHERE bucket='shared';
			END IF;
			INSERT INTO %[1]s.history(operation_id,resource,kind) VALUES (p_operation,p_resource,v_kind);
			RETURN v_version;
		END $owner$`, schema)
	for _, statement := range []string{
		"CREATE SCHEMA " + schema,
		"CREATE TABLE " + schema + ".subjects (subject TEXT PRIMARY KEY, role TEXT NOT NULL, disabled BOOLEAN NOT NULL DEFAULT FALSE)",
		"CREATE TABLE " + schema + ".authority (" +
			"application TEXT NOT NULL, capability TEXT NOT NULL, subject TEXT NOT NULL, action TEXT NOT NULL, resource TEXT NOT NULL, " +
			"source TEXT NOT NULL, subject_link BIGINT NOT NULL CHECK (subject_link > 0), membership BIGINT NOT NULL CHECK (membership > 0), " +
			"policy BIGINT NOT NULL CHECK (policy > 0), enabled BOOLEAN NOT NULL, " +
			"PRIMARY KEY (application,capability,subject,action,resource))",
		"CREATE TABLE " + schema + ".objects (application TEXT NOT NULL, resource TEXT NOT NULL, version BIGINT NOT NULL, value BYTEA NOT NULL, PRIMARY KEY(application,resource))",
		"CREATE TABLE " + schema + ".receipts (" +
			"operation_id TEXT PRIMARY KEY, application TEXT NOT NULL, capability TEXT NOT NULL, subject TEXT NOT NULL, action TEXT NOT NULL, " +
			"resource TEXT NOT NULL, request_hash TEXT NOT NULL, resource_version BIGINT NOT NULL)",
		"CREATE TABLE " + schema + ".inventory (bucket TEXT PRIMARY KEY, available INTEGER NOT NULL CHECK (available >= 0))",
		"INSERT INTO " + schema + ".inventory VALUES ('shared',3)",
		"CREATE TABLE " + schema + ".reservations (resource TEXT PRIMARY KEY, units INTEGER NOT NULL)",
		"CREATE TABLE " + schema + ".history (seq BIGSERIAL PRIMARY KEY, operation_id TEXT NOT NULL UNIQUE, resource TEXT NOT NULL, kind TEXT NOT NULL)",
		// The definer function is the only way for the effect writer to lock authority.
		"CREATE FUNCTION " + schema + ".lock_authority(" +
			"p_application TEXT, p_capability TEXT, p_subject TEXT, p_action TEXT, p_resource TEXT, " +
			"p_source TEXT, p_subject_link BIGINT, p_membership BIGINT, p_policy BIGINT) " +
			"RETURNS BOOLEAN LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS " + "$" + "$" + " " +
			"DECLARE v_source TEXT; v_link BIGINT; v_membership BIGINT; v_policy BIGINT; v_enabled BOOLEAN; " +
			"BEGIN " +
			"SELECT source,subject_link,membership,policy,enabled INTO v_source,v_link,v_membership,v_policy,v_enabled FROM " + schema +
			".authority WHERE application=p_application AND capability=p_capability AND subject=p_subject " +
			"AND action=p_action AND resource=p_resource FOR UPDATE; " +
			"IF NOT FOUND THEN RETURN FALSE; END IF; " +
			"RETURN v_enabled AND v_source=p_source AND v_link=p_subject_link AND " +
			"v_membership=p_membership AND v_policy=p_policy; END " + "$" + "$",
		"REVOKE ALL ON FUNCTION " + schema +
			".lock_authority(TEXT,TEXT,TEXT,TEXT,TEXT,TEXT,BIGINT,BIGINT,BIGINT) FROM PUBLIC",
		routine,
		"INSERT INTO " + schema + ".subjects(subject,role,disabled) VALUES ('" + rpgSubject + "','EDITOR',FALSE)",
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("試験用DB定義が不正です: %v / %s", err, statement)
		}
	}
	provider, err := postgres.NewWithTrustedOwnerRoutine(pool, schema)
	if err != nil {
		t.Fatal(err)
	}

	h := &rpgHarness{
		pool: pool, observerPool: observerPool, schema: schema, writePIDs: writePIDs, readPIDs: readPIDs,
		effects:     effectlog.File{Path: filepath.Join(t.TempDir(), "effects.jsonl")},
		runs:        runlog.File{Path: filepath.Join(t.TempDir(), "runs.jsonl")},
		rules:       rpgRules(),
		routeByRule: map[string]rpgRoute{},
	}
	h.invoker = &rpgInvoker{
		provider: provider,
		grants:   map[core.CanonicalID]legacyauthorization.AuthorizationGrant{},
		routes:   map[core.CanonicalID]rpgRoute{},
		after:    map[core.CanonicalID]func(){},
	}
	seeded := map[string]bool{}
	for _, route := range rpgRoutes {
		if !seeded[route.resource] {
			seeded[route.resource] = true
			if _, err := pool.Exec(ctx, "INSERT INTO "+schema+".objects(application,resource,version,value) VALUES ($1,$2,0,'original')",
				rpgApplication, route.resource); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := pool.Exec(ctx, "INSERT INTO "+schema+".authority(application,capability,subject,action,resource,source,subject_link,membership,policy,enabled) "+
			"VALUES ($1,$2,$3,$4,$5,'postgres-authority',1,1,1,TRUE)",
			rpgApplication, string(route.capability), rpgSubject, rpgAction, route.resource); err != nil {
			t.Fatal(err)
		}
		adapter := legacyauthorization.Adapter{Store: rpgSubjects{h}, EpochReader: rpgEpochs{h, string(route.capability)}, Rule: rpgRole{}}
		got, err := adapter.Invoke(ctx, legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
			SubjectID: rpgSubject, Action: rpgAction, Resource: route.resource,
		})
		if err != nil || got.Outcome != "allowed" {
			t.Fatalf("security.authorize: %+v %v", got, err)
		}
		grant, ok := got.Payload.(legacyauthorization.AuthorizationGrant)
		if !ok || grant.Epochs == nil {
			t.Fatal("認可版がありません")
		}
		h.invoker.grants[route.capability] = grant
		h.invoker.routes[route.capability] = route
	}
	for _, rule := range h.rules {
		h.routeByRule[string(rule.ID)] = h.invoker.routes[rule.ViaCapability]
	}
	checks := map[string]rpgRoute{}
	for id, route := range h.routeByRule {
		checks[id] = route
	}
	h.observer = rpgObserver{checks: checks}
	h.observer.inner = execution.SQLEffectObserver{
		Queryer: rpgQueryer{observerPool, readPIDs},
		Query: "SELECT count(*) FROM " + schema + ".history h JOIN " + schema + ".receipts rc ON rc.operation_id=h.operation_id " +
			"WHERE h.operation_id=$1::text AND h.kind=$2::text AND rc.resource=$3::text AND CASE $2::text " +
			"WHEN 'reserve' THEN EXISTS (SELECT 1 FROM " + schema + ".reservations r WHERE r.resource=$3::text AND r.units=$4::int) " +
			"WHEN 'release' THEN NOT EXISTS (SELECT 1 FROM " + schema + ".reservations r WHERE r.resource=$3::text) " +
			"ELSE TRUE END",
		Args: func(record effectlog.Record) []any {
			route, ok := checks[record.RuleID]
			if !ok {
				return nil
			}
			return []any{route.operation, route.kind, route.resource, int32(route.units)}
		},
	}
	return h
}

// assertObserverIndependent requires that the Observer ran queries, that the
// Command wrote, and that no session served both: the Observer's reads did not go
// through a connection the Command used.
func (h *rpgHarness) assertObserverIndependent(t *testing.T) {
	t.Helper()
	reads, writes := h.readPIDs.snapshot(), h.writePIDs.snapshot()
	if len(reads) == 0 || len(writes) == 0 {
		t.Fatalf("no sessions were recorded: observer=%v command=%v", reads, writes)
	}
	for pid := range reads {
		if writes[pid] {
			t.Fatalf("the Observer read through backend %d, which the Command wrote through (observer=%v command=%v)", pid, reads, writes)
		}
	}
}

func (h *rpgHarness) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	if err := h.pool.QueryRow(context.Background(), strings.ReplaceAll(query, "@", h.schema+"."), args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (h *rpgHarness) version(t *testing.T, resource string) int {
	t.Helper()
	return h.count(t, "SELECT version FROM @objects WHERE application=$1 AND resource=$2", rpgApplication, resource)
}

func (h *rpgHarness) available(t *testing.T) int {
	t.Helper()
	return h.count(t, "SELECT available FROM @inventory WHERE bucket='shared'")
}

func (h *rpgHarness) historyOrder(t *testing.T) []string {
	t.Helper()
	rows, err := h.pool.Query(context.Background(), "SELECT operation_id FROM "+h.schema+".history ORDER BY seq")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var op string
		if err := rows.Scan(&op); err != nil {
			t.Fatal(err)
		}
		out = append(out, op)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// reconcile checks the effect journal against the Owner tables and the receipts
// table: a record is Succeeded exactly when its receipt and its history row
// exist, Observed implies the history row, the receipts hold nothing the journal
// does not claim, and the history shows the Succeeded effects in journal order.
func (h *rpgHarness) reconcile(t *testing.T, records []effectlog.Record) {
	t.Helper()
	var succeeded []string
	for _, record := range records {
		route, ok := h.routeByRule[record.RuleID]
		if !ok {
			t.Fatalf("journal names an unknown rule: %+v", record)
		}
		receipt := h.count(t, "SELECT count(*) FROM @receipts WHERE operation_id=$1", route.operation)
		history := h.count(t, "SELECT count(*) FROM @history WHERE operation_id=$1", route.operation)
		if record.Succeeded != (receipt == 1) || record.Succeeded != (history == 1) {
			t.Errorf("journal and DB disagree for %s: record=%+v receipts=%d history=%d", record.RuleID, record, receipt, history)
		}
		if record.Observed && !record.Succeeded {
			t.Errorf("an effect that did not succeed cannot be observed: %+v", record)
		}
		// Observed is also checked against the data, not only against Succeeded.
		// A Succeeded effect recorded as not observed must really be absent now: the
		// same read, made over the primary pool, finds nothing. (The reverse is not
		// asserted here because a later compensation legitimately removes an effect
		// that was observed when it was recorded.)
		if record.Succeeded && !record.Observed {
			args := []any{route.operation, route.kind, route.resource, int32(route.units)}
			if got := h.count(t, h.observer.inner.Query, args...); got != 0 {
				t.Errorf("%s is recorded as not observed but the data shows the effect (%d row)", record.RuleID, got)
			}
		}
		if record.Succeeded {
			succeeded = append(succeeded, route.operation)
		}
	}
	if got := h.count(t, "SELECT count(*) FROM @receipts"); got != len(succeeded) {
		t.Errorf("receipts hold %d commits but the journal claims %d", got, len(succeeded))
	}
	if got := h.historyOrder(t); !reflect.DeepEqual(got, succeeded) {
		t.Errorf("the Owner history order %v differs from the journal order %v", got, succeeded)
	}
}

func rpgRuleIDs(records []effectlog.Record) []string {
	out := make([]string, 0, len(records))
	for _, record := range records {
		out = append(out, record.RuleID)
	}
	return out
}

// step runs the effect rule bound to the operation and reports it as a plan step.
func (h *rpgHarness) step(observer execution.EffectObserver, failed *semantic.EffectRule) execution.StepFunc {
	return func(ctx context.Context, operation core.CanonicalID) execution.StepResult {
		for _, rule := range h.rules {
			if rule.OnCapability != operation || rule.Stage != "after-commit" {
				continue
			}
			record, err := execution.RunEffect(ctx, h.invoker, observer, h.effects, rpgApplication, rule, nil, nil)
			result := execution.StepResult{Operation: string(operation), Detail: record.Detail}
			switch {
			case err != nil:
				result.Status, result.Detail = "error", err.Error()
			case !record.Succeeded:
				result.Status = "failed"
			case !record.Observed:
				result.Status = "unobserved"
			default:
				result.Status = "resolved"
			}
			if result.Status != "resolved" && failed != nil {
				*failed = rule
			}
			return result
		}
		return execution.StepResult{Operation: string(operation), Status: "failed", Detail: "no effect rule is bound"}
	}
}

// coverage is the unattended precondition for the checks an observer must have:
// an effect whose rule the observer has no check for cannot be confirmed even when
// an observer is present.
func (h *rpgHarness) coverage(covers func(string) bool) execution.PreconditionFunc {
	return func(capabilityID core.CanonicalID) string {
		for _, rule := range h.rules {
			if rule.OnCapability != capabilityID || (rule.Stage != "after-commit" && rule.Stage != "compensation") {
				continue
			}
			if !covers(string(rule.ID)) {
				return "the check for effect " + string(rule.ID) + " is missing"
			}
		}
		return ""
	}
}

func rpgPlan(autonomy string, operations ...core.CanonicalID) semantic.RunPlan {
	return semantic.RunPlan{ID: "sales.fulfil", Operations: operations, Key: []core.CanonicalID{"sales.period.month"}, Resume: "none", Autonomy: autonomy}
}

// Succeeded is the Command's result; Observed is a second connection pool's read
// of the Owner tables. They are recorded separately, so an effect the Command
// committed but that is no longer visible stays Succeeded without being Observed,
// and a denied Command is neither.
func TestRealPGEffectSucceededAndIndependentObservedAreRecordedSeparately(t *testing.T) {
	h := rpgOpen(t)
	ctx := context.Background()

	// The observer's pool is a different database session from the Command's.
	cmdConn, err := h.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cmdConn.Release()
	obsConn, err := h.observerPool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer obsConn.Release()
	var cmdPID, obsPID int
	if err := cmdConn.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&cmdPID); err != nil {
		t.Fatal(err)
	}
	if err := obsConn.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&obsPID); err != nil {
		t.Fatal(err)
	}
	if cmdPID == obsPID {
		t.Fatalf("the observer shares a database session with the Command: %d", cmdPID)
	}
	cmdConn.Release()
	obsConn.Release()

	// 1. Committed and visible over the second pool.
	committed, err := execution.RunEffect(ctx, h.invoker, h.observer, h.effects, rpgApplication, rpgRuleByID(h.rules, "sales.fulfil.reserve_a"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !committed.Succeeded || !committed.Observed || !strings.Contains(committed.Detail, "1 row(s)") {
		t.Fatalf("a committed, visible effect must be Succeeded and Observed: %+v", committed)
	}

	// 2. Committed, then the Owner row disappears before the independent read.
	h.invoker.after["sales.stock.reserve_d"] = func() {
		if _, err := h.pool.Exec(ctx, "DELETE FROM "+h.schema+".reservations WHERE resource='order-d'"); err != nil {
			t.Error(err)
		}
	}
	vanished, err := execution.RunEffect(ctx, h.invoker, h.observer, h.effects, rpgApplication, rpgRuleByID(h.rules, "sales.fulfil.reserve_d"), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !vanished.Succeeded || vanished.Observed {
		t.Fatalf("Succeeded must not imply Observed: %+v", vanished)
	}
	if !strings.Contains(vanished.Detail, "not visible") {
		t.Fatalf("the missing observation must be explained: %+v", vanished)
	}
	if h.count(t, "SELECT count(*) FROM @receipts WHERE operation_id='op-reserve-d'") != 1 {
		t.Fatal("the Command's receipt must still exist")
	}

	// 3. Denied by the Owner: a recorded failure, not an error, and not observed.
	denied, err := execution.RunEffect(ctx, h.invoker, h.observer, h.effects, rpgApplication, rpgRuleByID(h.rules, "sales.fulfil.reserve_b"), nil, nil)
	if err != nil {
		t.Fatalf("a failed invocation is recorded, not returned: %v", err)
	}
	if denied.Succeeded || denied.Observed || !strings.Contains(denied.Detail, effectcommit.ErrBusinessDenied.Error()) {
		t.Fatalf("a denied Command must record the denial and nothing observed: %+v", denied)
	}
	if h.version(t, "order-b") != 0 || h.available(t) != 0 {
		t.Fatalf("the denied effect left a trace: version=%d available=%d", h.version(t, "order-b"), h.available(t))
	}

	records, err := h.effects.Records()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"sales.fulfil.reserve_a", "sales.fulfil.reserve_d", "sales.fulfil.reserve_b"}; !reflect.DeepEqual(rpgRuleIDs(records), want) {
		t.Fatalf("journal order=%v want %v", rpgRuleIDs(records), want)
	}
	if !records[0].Observed || records[1].Observed || records[2].Succeeded {
		t.Fatalf("the journal lost the separation: %+v", records)
	}
	h.reconcile(t, records)
	h.assertObserverIndependent(t)
}

// A later step fails: the compensations run in declared order, one of them fails
// and is recorded without stopping the others, and the journal, the Owner tables
// and the receipts agree.
func TestRealPGLaterStepFailureCompensatesInDeclaredOrderAndRecordsCompensationFailure(t *testing.T) {
	h := rpgOpen(t)
	ctx := context.Background()
	var failed semantic.EffectRule
	preconditions := execution.CombinePreconditions(execution.EffectPreconditions(h.rules, h.observer), h.coverage(h.observer.Covers))

	decision, results, err := execution.RunPlan(ctx, rpgPlan("unattended", "sales.order.fulfil_a", "sales.order.fulfil_b"),
		[]string{"2026-11"}, h.runs, h.step(h.observer, &failed), preconditions)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != runlog.Start || len(results) != 2 || results[0].Status != "resolved" || results[1].Status != "failed" {
		t.Fatalf("decision=%+v results=%+v", decision, results)
	}
	if failed.ID != "sales.fulfil.reserve_b" {
		t.Fatalf("the failed effect was not captured: %+v", failed)
	}
	runs, err := h.runs.Records()
	if err != nil {
		t.Fatal(err)
	}
	if last := runs[len(runs)-1]; last.Status != runlog.Failed || !reflect.DeepEqual(last.Steps, []string{"sales.order.fulfil_a"}) {
		t.Fatalf("a later failure must end the run as failed with the finished step kept: %+v", last)
	}
	// Before the compensation the first step's effect is in place and the stock is short.
	if h.available(t) != 1 || h.count(t, "SELECT count(*) FROM @reservations") != 1 {
		t.Fatalf("the first step's effect is missing: available=%d", h.available(t))
	}

	compensations, err := execution.Compensate(ctx, h.invoker, h.observer, h.effects, rpgApplication, h.rules, failed, nil, nil)
	if err != nil {
		t.Fatalf("a compensation that fails is recorded, not returned: %v", err)
	}
	if want := []string{"sales.fulfil.undo_a", "sales.fulfil.undo_c", "sales.fulfil.audit_b"}; !reflect.DeepEqual(rpgRuleIDs(compensations), want) {
		t.Fatalf("compensations ran out of the declared order or were skipped: %v want %v", rpgRuleIDs(compensations), want)
	}
	if !compensations[0].Succeeded || !compensations[0].Observed {
		t.Fatalf("undo_a must be committed and observed: %+v", compensations[0])
	}
	if compensations[1].Succeeded || compensations[1].Observed || !strings.Contains(compensations[1].Detail, effectcommit.ErrBusinessDenied.Error()) {
		t.Fatalf("undo_c fails (nothing was reserved) and the failure must be recorded: %+v", compensations[1])
	}
	if !compensations[2].Succeeded || !compensations[2].Observed {
		t.Fatalf("a failed compensation must not stop the next one: %+v", compensations[2])
	}

	records, err := h.effects.Records()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sales.fulfil.reserve_a", "sales.fulfil.reserve_b", "sales.fulfil.undo_a", "sales.fulfil.undo_c", "sales.fulfil.audit_b"}
	if !reflect.DeepEqual(rpgRuleIDs(records), want) {
		t.Fatalf("journal order=%v want %v", rpgRuleIDs(records), want)
	}
	h.reconcile(t, records)
	// The stock is restored, the failed step left nothing, and the failed compensation wrote nothing.
	if h.available(t) != 3 || h.count(t, "SELECT count(*) FROM @reservations") != 0 {
		t.Fatalf("the compensation did not restore the stock: available=%d", h.available(t))
	}
	if a, b, c := h.version(t, "order-a"), h.version(t, "order-b"), h.version(t, "order-c"); a != 2 || b != 1 || c != 0 {
		t.Fatalf("resource versions a=%d b=%d c=%d want 2/1/0", a, b, c)
	}
	h.assertObserverIndependent(t)
}

// An unattended run stops as blocked, not failed, when no observer is given or
// when an observer lacks a check an effect needs. Blocked means the Command was
// not attempted for that step.
func TestRealPGUnattendedRunIsBlockedNotFailedWithoutObserverOrChecks(t *testing.T) {
	t.Run("no observer", func(t *testing.T) {
		h := rpgOpen(t)
		ctx := context.Background()
		decision, results, err := execution.RunPlan(ctx, rpgPlan("unattended", "sales.order.fulfil_a", "sales.order.fulfil_b"),
			[]string{"2026-11"}, h.runs, h.step(nil, nil), execution.EffectPreconditions(h.rules, nil))
		if err != nil {
			t.Fatal(err)
		}
		if decision.Action != runlog.Start || len(results) != 1 || results[0].Status != "blocked" || !strings.Contains(results[0].Detail, "cannot be observed") {
			t.Fatalf("decision=%+v results=%+v", decision, results)
		}
		runs, _ := h.runs.Records()
		if last := runs[len(runs)-1]; last.Status != runlog.Blocked || len(last.Steps) != 0 {
			t.Fatalf("an unobservable effect must end the run as blocked, not failed: %+v", last)
		}
		if len(h.invoker.calls) != 0 || h.count(t, "SELECT count(*) FROM @receipts") != 0 || h.available(t) != 3 {
			t.Fatalf("blocked must not reach the Command: calls=%v", h.invoker.calls)
		}
		if records, _ := h.effects.Records(); len(records) != 0 {
			t.Fatalf("blocked must not leave an effect record: %+v", records)
		}
	})

	t.Run("observer lacks a required check", func(t *testing.T) {
		h := rpgOpen(t)
		ctx := context.Background()
		covers := func(id string) bool { return id != "sales.fulfil.reserve_b" && h.observer.Covers(id) }
		preconditions := execution.CombinePreconditions(execution.EffectPreconditions(h.rules, h.observer), h.coverage(covers))
		_, results, err := execution.RunPlan(ctx, rpgPlan("unattended", "sales.order.fulfil_a", "sales.order.fulfil_b"),
			[]string{"2026-11"}, h.runs, h.step(h.observer, nil), preconditions)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 || results[0].Status != "resolved" || results[1].Status != "blocked" ||
			!strings.Contains(results[1].Detail, "check for effect sales.fulfil.reserve_b is missing") {
			t.Fatalf("results=%+v", results)
		}
		runs, _ := h.runs.Records()
		if last := runs[len(runs)-1]; last.Status != runlog.Blocked || !reflect.DeepEqual(last.Steps, []string{"sales.order.fulfil_a"}) {
			t.Fatalf("a missing check must end the run as blocked with the earlier step kept: %+v", last)
		}
		if !reflect.DeepEqual(h.invoker.calls, []core.CanonicalID{"sales.stock.reserve_a"}) {
			t.Fatalf("only the covered step may reach the Command: %v", h.invoker.calls)
		}
		records, _ := h.effects.Records()
		if !reflect.DeepEqual(rpgRuleIDs(records), []string{"sales.fulfil.reserve_a"}) {
			t.Fatalf("the blocked step must leave no record: %v", rpgRuleIDs(records))
		}
		h.reconcile(t, records)
		if h.version(t, "order-b") != 0 {
			t.Fatal("the blocked step reached its resource")
		}
	})

	t.Run("covered run completes", func(t *testing.T) {
		h := rpgOpen(t)
		preconditions := execution.CombinePreconditions(execution.EffectPreconditions(h.rules, h.observer), h.coverage(h.observer.Covers))
		_, results, err := execution.RunPlan(context.Background(), rpgPlan("unattended", "sales.order.fulfil_a"),
			[]string{"2026-11"}, h.runs, h.step(h.observer, nil), preconditions)
		if err != nil {
			t.Fatal(err)
		}
		runs, _ := h.runs.Records()
		if len(results) != 1 || results[0].Status != "resolved" || runs[len(runs)-1].Status != runlog.Completed {
			t.Fatalf("the gate must not block a covered, observable effect: %+v %+v", results, runs)
		}
		records, _ := h.effects.Records()
		h.reconcile(t, records)
	})
}
