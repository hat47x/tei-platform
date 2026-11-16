package operations

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/workspace"
)

// This is the EXISTING production Resolver -> BrowseQueryExecutor path, not a
// newly invented Organization Function/Context Engine. It uses a fake
// database/sql Connector and therefore does not prove real physical scope,
// Function Data Context, consumer delivery, or Host authorization.
func verifyNativeCasePatternReadProbe(t *testing.T, loaded workspace.Result) {
	t.Helper()
	probe := &orgCaseQueryProbe{
		columns: []string{"handling_unit_id", "case_id"}, // reversed raw result
		rows:    [][]sqldriver.Value{{"D1", "C1"}},
	}
	db := sql.OpenDB(orgCaseQueryConnector{probe: probe})
	t.Cleanup(func() { _ = db.Close() })
	resolver := driver.NewResolver(loaded.World)
	const patternID = "org_view.case_browse"
	exec := postgresquery.Executor{DB: db, Database: "org_store"}

	// The Browse path validates a declared input's runtime representation before
	// any provider is called (driver/query_execution.go, c9596c5e2). A domain
	// type such as organization.case_record_id has no binding — the gate accepts
	// std.datetime only — so a call is refused rather than coerced from a string.
	// The fixture declares the domain type, so the refusal is what happens today
	// and it is asserted first: a refusal that silently reached the connector
	// would be the worse defect.
	if _, err := resolver.ExecuteBrowseQuery(context.Background(), "org_view",
		patternID, exec, map[string]any{"case_id": "C1"}); err == nil {
		t.Fatal("a domain-typed input was accepted without a runtime binding")
	} else if !strings.Contains(err.Error(), "unsupported runtime binding") {
		t.Fatalf("the refusal must name the missing binding: %v", err)
	}
	if probe.statement != "" {
		t.Fatalf("the refused call reached the SQL connector: %q", probe.statement)
	}

	// The path this probe exists for — Resolver -> Executor -> Mapper — needs a
	// declared input the gate can bind. std.datetime is the only one, and it is
	// what the driver's own tests use. Binding the fixture that way keeps the
	// remaining assertions about the Pattern/Query/Database/Data resolution and
	// the raw-to-mapped projection under test.
	boundWorld := loaded.World
	for i := range boundWorld.Queries {
		for j := range boundWorld.Queries[i].Inputs {
			if boundWorld.Queries[i].Inputs[j].Name == "case_id" {
				boundWorld.Queries[i].Inputs[j].Type = "std.datetime"
			}
		}
	}
	boundResolver := driver.NewResolver(boundWorld)
	at := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	result, err := boundResolver.ExecuteBrowseQuery(context.Background(), "org_view",
		patternID, exec, map[string]any{"case_id": at})
	if err != nil {
		t.Fatalf("native browse Pattern did not call the actual Query/Executor/Mapper: %v", err)
	}
	if result.Resolution.Pattern.ID != patternID || result.Resolution.Data == nil ||
		result.Resolution.Data.ID != "organization.case_record" ||
		result.Resolution.Query == nil || result.Resolution.Query.ID != "org_view.case_read" ||
		result.Resolution.Datasource == nil ||
		result.Resolution.Datasource.TargetDatabase != "org_store" {
		t.Fatalf("native Case browse Pattern/Query/Database/Data binding lost: %+v",
			result.Resolution)
	}
	if !reflect.DeepEqual(result.Result.Mapped.Rows, [][]any{{"C1", "D1"}}) ||
		len(result.Result.Mapped.Columns) != 2 ||
		result.Result.Mapped.Columns[0].Attribute != core.CanonicalID("organization.case_record.case_id") ||
		result.Result.Mapped.Columns[1].Attribute != core.CanonicalID("organization.case_record.handling_unit_id") {
		t.Fatalf("Driver result is not the native Semantic Case/handlingUnit mapping: %+v",
			result.Result.Mapped)
	}
	if probe.statement != "select case_id, handling_unit_id from public.org_case_records where case_id = $1" ||
		len(probe.args) != 1 || !reflect.DeepEqual(probe.args[0].Value, at) {
		t.Fatalf("Driver did not propagate authored SQL/Case identity to the physical executor: %q, %+v",
			probe.statement, probe.args)
	}

	// An unbound Pattern or a mismatched injected Database MUST fail. Both
	// failures are independent of the result's tempting C1/D1 visible values.
	before := probe.statement
	if _, err := boundResolver.ExecuteBrowseQuery(context.Background(), "org_view",
		"org_view.not_declared", exec, map[string]any{"case_id": at}); err == nil {
		t.Fatal("unbound Function Pattern was executed by inference")
	}
	if _, err := boundResolver.ExecuteBrowseQuery(context.Background(), "org_view",
		patternID, postgresquery.Executor{DB: db, Database: "other_store"},
		map[string]any{"case_id": at}); err == nil {
		t.Fatal("wrong physical Database was accepted by the native Pattern")
	}
	if probe.statement != before {
		t.Fatal("rejected Function candidates reached the SQL connector")
	}
}
