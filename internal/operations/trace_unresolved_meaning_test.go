package operations_test

import (
	"path/filepath"
	"testing"

	"tei/internal/modelref"
	"tei/internal/operations"
	"tei/internal/subject"
)

// traceKey builds the exact key TraceWorkspace requires for one subject.
func traceKey(kind, id string) modelref.Key {
	return modelref.Key{Scope: "s", Subject: subject.Subject{Kind: kind, ID: id}}
}

// TestProjectionUnresolvedIsGraphWideNotSubjectScoped pins the contract sentence
// added for #3906. A reader that sees trace.unresolved = 0 must not conclude the
// Workspace has no unresolved references: projectionUnresolved reports the whole
// graph's dropped references, and it is identical for every subject.
func TestProjectionUnresolvedIsGraphWideNotSubjectScoped(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	keys := []modelref.Key{
		traceKey("semantic-data", "sales.order"),
		traceKey("semantic-relation", "sales.order.customer"),
		traceKey("semantic-type", "std.text"),
	}
	options := modelref.TraceOptions{Mode: "full", Direction: "both"}

	var reference []string
	for i, key := range keys {
		result, err := operations.TraceWorkspace(root, key, options)
		if err != nil {
			t.Fatalf("%s: trace: %v", key.Subject.ID, err)
		}
		if i == 0 {
			reference = result.ProjectionUnresolved
			if len(reference) == 0 {
				t.Skip("this Workspace has no projection-unresolved references")
			}
			continue
		}
		// The same graph-wide list, whatever subject was asked for.
		if len(result.ProjectionUnresolved) != len(reference) {
			t.Fatalf("%s: projectionUnresolved=%d, want the graph-wide %d",
				key.Subject.ID, len(result.ProjectionUnresolved), len(reference))
		}
		for j := range reference {
			if result.ProjectionUnresolved[j] != reference[j] {
				t.Fatalf("%s: projectionUnresolved[%d]=%q, want %q",
					key.Subject.ID, j, result.ProjectionUnresolved[j], reference[j])
			}
		}
	}
}

// The inner count is a count of rows, not a claim about the graph. When no row
// carries an issue the count is zero even though the graph-wide list is not.
func TestTraceUnresolvedCountsRowsNotGraphWideFindings(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	result, err := operations.TraceWorkspace(root, traceKey("semantic-data", "sales.order"),
		modelref.TraceOptions{Mode: "full", Direction: "both"})
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	rowsWithIssue := 0
	for _, row := range result.Trace.Rows {
		if row.Issue != "" {
			rowsWithIssue++
		}
	}
	if result.Trace.Unresolved != rowsWithIssue {
		t.Fatalf("trace.unresolved=%d but %d rows carry an issue",
			result.Trace.Unresolved, rowsWithIssue)
	}
	// The distinguishing case the contract now describes: readable from one
	// answer, with both a zero row count and a non-empty graph-wide list.
	if len(result.Trace.Rows) > 0 && len(result.ProjectionUnresolved) > 0 &&
		result.Trace.Unresolved == 0 {
		t.Logf("observed: %d rows, trace.unresolved=0, projectionUnresolved=%d",
			len(result.Trace.Rows), len(result.ProjectionUnresolved))
	}
}

// The six issue values are declared, but the projection builds its rules from
// the edges it just emitted, so the three rule-shaped values cannot arise on
// this path. Pinning that is what keeps the contract sentence from becoming
// false if the construction changes.
func TestProjectionRulesComeFromTheEdgesItEmits(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	result, err := operations.TraceWorkspace(root, traceKey("semantic-data", "sales.order"),
		modelref.TraceOptions{Mode: "full", Direction: "both"})
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	for _, row := range result.Trace.Rows {
		switch row.Issue {
		case "", "invalid-source", "ambiguous-source", "invalid-target":
			// Reachable on this path.
		case "invalid-edge-type", "invalid-rule", "conflicting-rule":
			t.Fatalf("a row carries %q, which the projection cannot produce: a rule "+
				"is created from each edge it emits, so no edge exists without a rule",
				row.Issue)
		default:
			t.Fatalf("a row carries the undeclared issue %q", row.Issue)
		}
	}
}
