package operations

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/workspace"
)

// This probe uses the REAL native TEI Query loaded in the existing CaseRecord
// characterization. The SQL connection below is an instrumented test double;
// no physical PostgreSQL, Function Data Context, current Grant or Consumer runs.
type orgCaseQueryProbe struct {
	statement string
	args      []sqldriver.NamedValue
	columns   []string
	rows      [][]sqldriver.Value
}

type orgCaseQueryConnector struct{ probe *orgCaseQueryProbe }

func (c orgCaseQueryConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &orgCaseQueryConn{probe: c.probe}, nil
}
func (orgCaseQueryConnector) Driver() sqldriver.Driver { return orgCaseQueryDriver{} }

type orgCaseQueryDriver struct{}

func (orgCaseQueryDriver) Open(string) (sqldriver.Conn, error) {
	return nil, errors.New("use the scoped test connector")
}

type orgCaseQueryConn struct{ probe *orgCaseQueryProbe }

func (*orgCaseQueryConn) Prepare(string) (sqldriver.Stmt, error) {
	return nil, sqldriver.ErrSkip
}
func (*orgCaseQueryConn) Close() error                 { return nil }
func (*orgCaseQueryConn) Begin() (sqldriver.Tx, error) { return nil, sqldriver.ErrSkip }
func (c *orgCaseQueryConn) QueryContext(_ context.Context, statement string, args []sqldriver.NamedValue) (sqldriver.Rows, error) {
	c.probe.statement = statement
	c.probe.args = append([]sqldriver.NamedValue(nil), args...)
	rows := make([][]sqldriver.Value, len(c.probe.rows))
	for i := range c.probe.rows {
		rows[i] = append([]sqldriver.Value(nil), c.probe.rows[i]...)
	}
	return &orgCaseQueryRows{columns: append([]string(nil), c.probe.columns...), rows: rows}, nil
}

var _ sqldriver.QueryerContext = (*orgCaseQueryConn)(nil)

type orgCaseQueryRows struct {
	columns []string
	rows    [][]sqldriver.Value
	offset  int
}

func (r *orgCaseQueryRows) Columns() []string { return r.columns }
func (r *orgCaseQueryRows) Close() error      { return nil }
func (r *orgCaseQueryRows) Next(dest []sqldriver.Value) error {
	if r.offset == len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.offset])
	r.offset++
	return nil
}

// verifyNativeCaseQuerySQLProbe is invoked only with the real Loader result
// created inside TestOrgCaseExplicitPersistenceEnablesReadCandidateOnly.
// It adds no duplicate native .tei/SQL fixture and no new runtime API.
func verifyNativeCaseQuerySQLProbe(t *testing.T, loaded workspace.Result) {
	t.Helper()
	if len(loaded.World.Queries) != 1 {
		t.Fatalf("need exactly one native Query for the probe: %+v", loaded.World.Queries)
	}
	q := loaded.World.Queries[0]
	if q.ID != "org_view.case_read" || len(loaded.World.Applications) != 1 {
		t.Fatalf("unexpected native Query/Application: %s / %d", q.ID, len(loaded.World.Applications))
	}
	app := loaded.World.Applications[0]
	if app.ID != "org_view" {
		t.Fatalf("unexpected application: %s", app.ID)
	}
	// DatabaseByAliasExact rejects duplicate alias candidates: no first-wins.
	ds, ok := app.DatabaseByAliasExact(q.Datasource)
	if !ok || ds.TargetDatabase != core.CanonicalID("org_store") {
		t.Fatalf("unresolved or unexpected actual authored datasource: %+v, %v", ds, ok)
	}
	probe := &orgCaseQueryProbe{
		// Deliberate physical column reordering to test explicit result mapping.
		columns: []string{"handling_unit_id", "case_id"},
		rows:    [][]sqldriver.Value{{"D1", "C1"}},
	}
	db := sql.OpenDB(orgCaseQueryConnector{probe: probe})
	t.Cleanup(func() { _ = db.Close() })
	executor := postgresquery.Executor{DB: db, Database: ds.TargetDatabase}
	got, err := executor.ExecuteBrowseQuery(context.Background(), q, ds, map[string]any{"case_id": "C1"})
	if err != nil {
		t.Fatalf("actual native query was not sent to the existing PostgreSQL executor: %v", err)
	}
	const wantSQL = "select case_id, handling_unit_id from public.org_case_records where case_id = $1"
	if strings.TrimSpace(probe.statement) != wantSQL {
		t.Fatalf("wrong actual named-to-positional SQL: %q", probe.statement)
	}
	if len(probe.args) != 1 || probe.args[0].Value != "C1" {
		t.Fatalf("Case business identity must bind as a parameter, got %+v", probe.args)
	}
	physical, ok := got.Payload.(postgresquery.Result)
	if !ok || !reflect.DeepEqual(physical.ParameterNames, []string{"case_id"}) {
		t.Fatalf("physical parameter evidence names (not values) lost: %+v", got.Payload)
	}
	mapped, err := queryresult.Map(q, got.Raw)
	if err != nil {
		t.Fatalf("explicit Semantic mapping rejected the native physical result: %v", err)
	}
	if len(mapped.Columns) != 2 ||
		mapped.Columns[0].Attribute != core.CanonicalID("organization.case_record.case_id") ||
		mapped.Columns[1].Attribute != core.CanonicalID("organization.case_record.handling_unit_id") ||
		!reflect.DeepEqual(mapped.Rows, [][]any{{"C1", "D1"}}) {
		t.Fatalf("Case/OrgUnit relation identity was mapped by inference/order: %+v", mapped)
	}
	// A mismatched connection or extra/unmapped parameter must reject before
	// dispatch. These are provider *selection* negative controls, not RLS.
	statement := probe.statement
	if _, err := (postgresquery.Executor{DB: db, Database: "other_store"}).
		ExecuteBrowseQuery(context.Background(), q, ds, map[string]any{"case_id": "C1"}); err == nil {
		t.Fatal("database instance mismatch did not reject")
	}
	if _, err := executor.ExecuteBrowseQuery(context.Background(), q, ds,
		map[string]any{"case_id": "C1", "extra": "not-declared"}); err == nil {
		t.Fatal("unmapped physical parameter did not reject")
	}
	if probe.statement != statement {
		t.Fatal("rejected probes must not reach the physical SQL connection")
	}
}
