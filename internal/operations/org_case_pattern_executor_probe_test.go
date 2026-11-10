package operations

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"reflect"
	"testing"

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
	result, err := resolver.ExecuteBrowseQuery(context.Background(), "org_view",
		patternID, exec, map[string]any{"case_id": "C1"})
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
		len(probe.args) != 1 || probe.args[0].Value != "C1" {
		t.Fatalf("Driver did not propagate authored SQL/Case identity to the physical executor: %q, %+v",
			probe.statement, probe.args)
	}

	// An unbound Pattern or a mismatched injected Database MUST fail. Both
	// failures are independent of the result's tempting C1/D1 visible values.
	before := probe.statement
	if _, err := resolver.ExecuteBrowseQuery(context.Background(), "org_view",
		"org_view.not_declared", exec, map[string]any{"case_id": "C1"}); err == nil {
		t.Fatal("unbound Function Pattern was executed by inference")
	}
	if _, err := resolver.ExecuteBrowseQuery(context.Background(), "org_view",
		patternID, postgresquery.Executor{DB: db, Database: "other_store"},
		map[string]any{"case_id": "C1"}); err == nil {
		t.Fatal("wrong physical Database was accepted by the native Pattern")
	}
	if probe.statement != before {
		t.Fatal("rejected Function candidates reached the SQL connector")
	}
}
