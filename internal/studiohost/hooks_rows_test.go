package studiohost

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/validation"
)

// rowHookWorkspace writes a workspace whose pattern declares a row-scoped hook.
// The fixture reuses hookWorkspaceWithoutCall's shape so the rest of the world —
// the semantic namespace, the data and the capability — is the one the other
// Studio hook tests already load and validate.
func rowHookWorkspace(t *testing.T) string {
	t.Helper()
	root := hookWorkspaceWithoutCall(t)
	pattern := `pattern order_entry.order_record record {
    data sales.order

    interaction {
        item sales.order.quantity

        command sales.submit {
            input order sales.order.quantity
        }

        hook before-update {
            scope row
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

// rowQuantityRows is the page the caller hands to the Studio hook entry, one map
// per row, plus an attribute the declaration does not read.
func rowQuantityRows(values ...string) []map[core.CanonicalID]string {
	rows := make([]map[core.CanonicalID]string, 0, len(values))
	for _, value := range values {
		rows = append(rows, map[core.CanonicalID]string{
			"sales.order.quantity": value,
			"sales.order.status":   "not-declared",
		})
	}
	return rows
}

// countingPageRule records the quantity each invocation was decided with.
type countingPageRule struct {
	inner     hook.Implementation
	values    []string
	sawUnread bool
}

func (r *countingPageRule) Run(ctx context.Context, invocation hook.Invocation) error {
	r.values = append(r.values, invocation.Reads["sales.order.quantity"])
	if _, ok := invocation.Reads["sales.order.status"]; ok {
		r.sawUnread = true
	}
	if r.inner != nil {
		return r.inner.Run(ctx, invocation)
	}
	return nil
}

// TestRunHooksWithRowsAppliesTheRowScopeToEveryRow is the Studio wiring check:
// the host's hook entry hands the row set to hook.Runner.Rows, so the hook is
// applied once per row with that row's declared values.
func TestRunHooksWithRowsAppliesTheRowScopeToEveryRow(t *testing.T) {
	host, err := NewReadOnly([]string{rowHookWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, failure := host.loadWorkspace("workspace-1")
	if failure != nil {
		t.Fatalf("load: %+v", failure)
	}
	// The declaration this test runs is one validation accepts, so the run below
	// is the execution of a declarable shape and not of a fixture only the runner
	// would take.
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("validation refuses the row-scoped declaration: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	rule := &countingPageRule{}
	report, err := host.runHooksWithRows(
		loaded.World,
		"order_entry",
		"order_entry.order_record",
		map[string]string{"sales.order.quantity": "999"},
		rowQuantityRows("1", "2", "3"),
		rule,
	)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no hook report for a pattern that declares a hook")
	}
	if len(rule.values) != 3 || rule.values[0] != "1" || rule.values[1] != "2" || rule.values[2] != "3" {
		t.Fatalf("the hook was not applied once per row: %v", rule.values)
	}
	if rule.sawUnread {
		t.Fatal("a row reached the rule with an attribute the declaration does not read")
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
	if report.Note == "" {
		t.Fatal("the report does not state its boundary")
	}
}

// TestRunHooksWithRowsNamesTheRefusedRow checks that a refusal at one row stops
// there and that the row number reaches the Studio caller.
func TestRunHooksWithRowsNamesTheRefusedRow(t *testing.T) {
	host, err := NewReadOnly([]string{rowHookWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, failure := host.loadWorkspace("workspace-1")
	if failure != nil {
		t.Fatalf("load: %+v", failure)
	}
	rule := &countingPageRule{inner: hook.ReferenceRule{Positive: []core.CanonicalID{"sales.order.quantity"}}}
	report, err := host.runHooksWithRows(
		loaded.World,
		"order_entry",
		"order_entry.order_record",
		nil,
		rowQuantityRows("1", "0", "3"),
		rule,
	)
	if err == nil {
		t.Fatal("a non-positive row did not stop the run")
	}
	var refusal *hook.RowRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("the refusal does not carry the row: %v", err)
	}
	if refusal.Row != 1 {
		t.Fatalf("row=%d, want the second row (index 1)", refusal.Row)
	}
	if len(rule.values) != 2 {
		t.Fatalf("the rule ran for %v, want the two rows up to the refusal", rule.values)
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if report.Outcome != hook.OutcomeValueRejected {
		t.Fatalf("outcome=%q", report.Outcome)
	}
	if reason := report.Reasons["before-update"]; !strings.Contains(reason, "row 1") {
		t.Fatalf("the reason does not name the refused row: %q", reason)
	}
	if report.CommandOutcome != "value-rejected" {
		t.Fatalf("commandOutcome=%q, want the Command name for the hook's outcome", report.CommandOutcome)
	}
}

// TestRunHooksWithoutRowsSkipsTheRowScope pins the other side at the Studio
// boundary: the reference command execution supplies no row set, so the phase is
// skipped with a reason and never run once for the record.
func TestRunHooksWithoutRowsSkipsTheRowScope(t *testing.T) {
	host, err := NewReadOnly([]string{rowHookWorkspace(t)})
	if err != nil {
		t.Fatal(err)
	}
	loaded, failure := host.loadWorkspace("workspace-1")
	if failure != nil {
		t.Fatalf("load: %+v", failure)
	}
	rule := &countingPageRule{}
	report, err := host.runHooks(loaded.World, "order_entry", "order_entry.order_record", map[string]string{"sales.order.quantity": "1"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rule.values) != 0 {
		t.Fatalf("a row-scoped hook ran without a row set: %v", rule.values)
	}
	if report == nil {
		t.Fatal("no hook report")
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "before-update" {
		t.Fatalf("skipped=%v", report.Skipped)
	}
	if reason := report.Reasons["before-update"]; !strings.Contains(reason, "row set") {
		t.Fatalf("the reason does not name the missing row set: %q", reason)
	}
}
