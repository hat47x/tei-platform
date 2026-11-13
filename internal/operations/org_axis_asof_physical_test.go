package operations

import (
    "context"
    "database/sql"
    "database/sql/driver"
    "fmt"
    "path/filepath"
    "reflect"
    "sort"
    "strings"
    "testing"
    "time"

    postgresquery "tei/internal/driver/queryruntime/postgres"
    "tei/internal/driver/queryresult"
    "tei/internal/workspace"
)

// TestOrgAxisAsOfQueryPhysicalBinding exercises the product Loader, real
// PostgreSQL parameter binder and result mapper with an instrumented SQL
// connection. It does not simulate SQL WHERE semantics or run a real DB.
func TestOrgAxisAsOfQueryPhysicalBinding(t *testing.T) {
    root := filepath.Join("..", "..", "reference", "org-staged-axis")
    loaded, err := workspace.Load(root)
    if err != nil {
        t.Fatalf("load native org axis workspace: %v", err)
    }
    if len(loaded.World.Queries) != 1 || len(loaded.World.Applications) != 1 {
        t.Fatalf("unexpected query/application count: q=%d app=%d", len(loaded.World.Queries), len(loaded.World.Applications))
    }
    q := loaded.World.Queries[0]
    if q.ID != "org_axes_view.as_of_membership" || q.Datasource != "org_read" {
        t.Fatalf("wrong native query: %+v", q)
    }
    app := loaded.World.Applications[0]
    ds, ok := app.DatabaseByAliasExact(q.Datasource)
    if !ok || ds.TargetDatabase != "org_store" {
        t.Fatalf("native datasource must bind uniquely to org_store: %+v / %v", ds, ok)
    }
    required := []string{
        "FROM public.org_memberships m",
        "m.valid_from <= :valid_from",
        ":valid_from < m.valid_to",
        "m.known_from <= :known_from",
        ":known_from < m.known_to",
        "m.allocation >= :allocation_min",
    }
    for _, token := range required {
        if !strings.Contains(q.Source, token) {
            t.Fatalf("native as-of Query missing required predicate/table %q: %s", token, q.Source)
        }
    }
    // The result intentionally arrives in a different physical column order
    // to ensure the Query's explicit Semantic mapping remains the authority.
    probe := &orgCaseQueryProbe{
        columns: []string{"allocation", "org_unit_id", "membership_id"},
        rows: [][]driver.Value{{"0.60", "D1", "A2"}},
    }
    db := sql.OpenDB(orgCaseQueryConnector{probe: probe})
    defer db.Close()
    executor := postgresquery.Executor{DB: db, Database: ds.TargetDatabase}
    validAt := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
    knownAt := time.Date(2027, 7, 1, 0, 0, 0, 0, time.UTC)
    got, err := executor.ExecuteBrowseQuery(context.Background(), q, ds, map[string]any{
        "valid_from": validAt, "known_from": knownAt, "allocation_min": "0.40",
    })
    if err != nil {
        t.Fatalf("native Query failed before instrumented DB read: %v", err)
    }
    for _, input := range []struct{name string; value any}{
        {"valid_from", validAt}, {"known_from", knownAt}, {"allocation_min", "0.40"},
    } {
        seen := false
        for _, arg := range probe.args {
            if reflect.DeepEqual(arg.Value, input.value) { seen = true }
        }
        if !seen {
            t.Fatalf("bound SQL did not forward %s=%v: %+v", input.name, input.value, probe.args)
        }
    }
    if strings.Contains(probe.statement, ":valid_from") ||
        strings.Contains(probe.statement, ":known_from") ||
        strings.Contains(probe.statement, ":allocation_min") {
        t.Fatalf("named SQL parameters were not bound: %s", probe.statement)
    }
    mapped, err := queryresult.Map(q, got.Raw)
    if err != nil {
        t.Fatalf("native query explicit result map: %v", err)
    }
    if len(mapped.Rows) != 1 || len(mapped.Rows[0]) != 3 ||
        mapped.Rows[0][0] != "A2" || mapped.Rows[0][1] != "D1" {
        t.Fatalf("semantic result must map by column name, not input order: %+v", mapped)
    }
    // Wrong database and unknown parameter cannot dispatch to SQL.
    recorded := probe.statement
    if _, err := (postgresquery.Executor{DB: db, Database: "wrong_store"}).
        ExecuteBrowseQuery(context.Background(), q, ds, map[string]any{
            "valid_from": validAt, "known_from": knownAt, "allocation_min": "0.40",
        }); err == nil {
        t.Fatal("wrong target database must be rejected")
    }
    if _, err := executor.ExecuteBrowseQuery(context.Background(), q, ds, map[string]any{
        "valid_from": validAt, "known_from": knownAt, "allocation_min": "0.40", "not_declared": 1,
    }); err == nil {
        t.Fatal("unknown physical parameter must be rejected")
    }
    if probe.statement != recorded {
        t.Fatal("rejected input must not reach the SQL driver")
    }
    surface, err := SurfaceWorkspace(root, "org_axes_view")
    if err != nil {
        t.Fatal(err)
    }
    if surface.Effect != "read-only" || len(surface.AuthorityLimits) == 0 {
        t.Fatalf("native Query existence must not imply Host Grant: %+v", surface)
    }
}

// deterministic IDs for independent literal comparisons in the optional
// real-PostgreSQL test. Observed is from the native Query/Executor/Mapping.
func orgAxisResultIDs(rows [][]any) []string {
    ids := make([]string, 0, len(rows))
    for _, row := range rows {
        if len(row) != 3 {
            return nil
        }
        ids = append(ids, fmt.Sprint(row[0]))
    }
    sort.Strings(ids)
    return ids
}
