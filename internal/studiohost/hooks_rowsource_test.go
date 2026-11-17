package studiohost

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/query"
	"tei/internal/validation"
)

// rowSourceWorkspace writes a workspace whose record pattern declares a row
// scoped hook whose source is a query the application declares.
//
// hookWorkspace supplies the semantic data and the command the hook uses; this
// fixture adds only the query and the declaration under test, so the rest of the
// world stays the one the other Studio hook tests load and validate.
func rowSourceWorkspace(t *testing.T) string {
	t.Helper()
	root := hookWorkspaceWithoutCall(t)
	// The workspace manifest of the base fixture lists semantic and applications;
	// the datasource target is resolved from the databases source, so it is added
	// here rather than by editing the shared fixture.
	manifest := "workspace: studio_hook_row_source_fixture\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	// The application must declare the datasource the query names, and that
	// datasource must resolve to a declared database. Without either, the query is
	// refused before the hook is reached and the fixture would test validation
	// rather than the row-source wiring.
	application := `application order_entry {
    uses semantic sales

    database sales {
        target sales_core
        access read-write
    }
}
`
	if err := os.WriteFile(filepath.Join(root, "applications", "order_entry", "application.tei"), []byte(application), 0o600); err != nil {
		t.Fatal(err)
	}
	databaseDir := filepath.Join(root, "databases", "sales-core")
	if err := os.MkdirAll(databaseDir, 0o700); err != nil {
		t.Fatal(err)
	}
	database := `database sales_core {
    kind relational
    ownership managed-here
    uses semantic sales
}
`
	if err := os.WriteFile(filepath.Join(databaseDir, "database.tei"), []byte(database), 0o600); err != nil {
		t.Fatal(err)
	}
	queryDir := filepath.Join(root, "applications", "order_entry", "queries")
	if err := os.MkdirAll(queryDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// The query declares its own result mapping. That mapping is what makes the
	// hook's declared read deliverable, so the fixture writes it rather than
	// letting the conversion guess.
	querySource := `query order_entry.order_list {
    datasource sales
    sql sqlite {
        SELECT quantity FROM order_lines
    }
    result {
        column quantity -> sales.order.quantity
    }
}
`
	if err := os.WriteFile(filepath.Join(queryDir, "order_list.tei"), []byte(querySource), 0o600); err != nil {
		t.Fatal(err)
	}
	pattern := `pattern order_entry.order_record record {
    data sales.order

    interaction {
        item sales.order.status

        command sales.submit {
            input order sales.order.status
        }

        hook before-update {
            scope row
            rows from query order_entry.order_list
            reads sales.order.quantity
            on-failure reject
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(root, "applications", "order_entry", "patterns", "order-record.tei"), []byte(pattern), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// rowSourceColumns is the source query's result mapping: one column, carried as
// the attribute the hook declares to read.
func rowSourceColumns() []query.ResultColumnMapping {
	return []query.ResultColumnMapping{{Column: "quantity", Attribute: "sales.order.quantity"}}
}

// TestRunHooksForRowSourceAppliesTheDeclaredSource checks the Studio wiring: the
// rows a caller holds are converted only when they came from the query the hook
// named, and the row-scoped hook then runs once per row.
func TestRunHooksForRowSourceAppliesTheDeclaredSource(t *testing.T) {
	host, err := NewReadOnly([]string{rowSourceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, failure := host.loadWorkspace("workspace-1")
	if failure != nil {
		t.Fatalf("load: %+v", failure)
	}
	// The declaration is one validation accepts, so the run below executes a
	// declarable shape rather than a fixture only the runner would take.
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("validation refuses the row-scoped declaration: %s %s", diagnostic.Code, diagnostic.Message)
		}
		if diagnostic.Code == "APPLICATION_PATTERN_INTERACTION_HOOK_ROW_SCOPE_HAS_NO_SUPPLIER" {
			t.Fatalf("a declared row source still warns that no host supplies rows: %s", diagnostic.Message)
		}
	}
	rule := &countingPageRule{}
	result := NewRowSourceResult("order_entry.order_list", rowSourceColumns(), [][]any{{"1"}, {"2"}, {"3"}})
	report, err := host.RunHooksForRowSource(loaded.World, "order_entry", "order_entry.order_record", result, rule)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rule.values) != 3 || rule.values[0] != "1" || rule.values[1] != "2" || rule.values[2] != "3" {
		t.Fatalf("the hook was not applied once per row: %v", rule.values)
	}
	if report == nil || len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("report=%+v", report)
	}
}

// TestRunHooksForRowSourceRefusesAnotherQuerysResult makes the declaration the
// guard at the Studio boundary too: a page from a query the hook did not name is
// refused, because its rows belong to another subject.
func TestRunHooksForRowSourceRefusesAnotherQuerysResult(t *testing.T) {
	host, err := NewReadOnly([]string{rowSourceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, failure := host.loadWorkspace("workspace-1")
	if failure != nil {
		t.Fatalf("load: %+v", failure)
	}
	result := NewRowSourceResult("order_entry.other_list", rowSourceColumns(), [][]any{{"1"}})
	if _, err := host.RunHooksForRowSource(loaded.World, "order_entry", "order_entry.order_record", result, &countingPageRule{}); err == nil {
		t.Fatal("a result from another query was applied to the declaration")
	}
}

// TestRunHooksForRowSourceReportsAnEmptyPageAsSkipped pins the limit this wiring
// shares with the CLI: hook.Runner.Rows carries rows, not "was supplied", so an
// empty page is reported as a skipped phase with a reason naming the source. It
// is not reported as executed, so an empty page never reads as a passed phase.
func TestRunHooksForRowSourceReportsAnEmptyPageAsSkipped(t *testing.T) {
	host, err := NewReadOnly([]string{rowSourceWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, failure := host.loadWorkspace("workspace-1")
	if failure != nil {
		t.Fatalf("load: %+v", failure)
	}
	rule := &countingPageRule{}
	result := NewRowSourceResult("order_entry.order_list", rowSourceColumns(), nil)
	report, err := host.RunHooksForRowSource(loaded.World, "order_entry", "order_entry.order_record", result, rule)
	if err != nil {
		t.Fatalf("an empty page was refused: %v", err)
	}
	if len(rule.values) != 0 {
		t.Fatalf("a rule ran for an empty page: %v", rule.values)
	}
	if report == nil {
		t.Fatal("no hook report")
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "before-update" {
		t.Fatalf("skipped=%v executed=%v", report.Skipped, report.Executed)
	}
}
