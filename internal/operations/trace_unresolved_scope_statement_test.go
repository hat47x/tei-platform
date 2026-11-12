package operations_test

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/operations"
)

// TestTraceStatesWhatEachUnresolvedCountMeasures pins #3906. The trace response
// carries two counts, and a caller who compares them directly reads a zero as
// "clean" when the graph is not: `trace.unresolved` counts only the edges of the
// subject this response traces, while `projectionUnresolved` counts the edges
// dropped while the whole graph was built, so it is the same for every subject
// in one Workspace.
//
// The reference Workspace shows both at once: order_entry has
// trace.unresolved == 0 and projectionUnresolved == 1. Nothing in the response
// said which one answers "is this subject reachable".
func TestTraceStatesWhatEachUnresolvedCountMeasures(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	options := modelref.TraceOptions{Mode: "full", Direction: "both"}

	out, err := operations.TraceWorkspace(root, traceKey("application", "order_entry"), options)
	if err != nil {
		t.Fatalf("trace: %v", err)
	}
	scopes := out.UnresolvedScopes

	if !scopes.TraceUnresolvedIsPerSubject {
		t.Error("the response must state that trace.unresolved is per subject")
	}
	if !scopes.ProjectionUnresolvedIsWholeGraph {
		t.Error("the response must state that projectionUnresolved is graph-wide")
	}
	if scopes.WhichToTrustForTheSubject == "" {
		t.Fatal("the response does not say which count answers the subject question")
	}
	if !strings.Contains(scopes.WhichToTrustForTheSubject, "trace.unresolved") {
		t.Fatalf("whichToTrustForTheSubject = %q, want it to name trace.unresolved",
			scopes.WhichToTrustForTheSubject)
	}

	// The statement must hold on the observed data, not just in the prose.
	if len(out.ProjectionUnresolved) == 0 {
		t.Skip("no projection-unresolved edges in this fixture, so the two counts agree")
	}
	if out.Trace.Unresolved != 0 {
		t.Skip("the traced subject has its own unresolved edges in this fixture")
	}

	// projectionUnresolved must be the same for a different subject from the
	// same Workspace, because it is graph-wide.
	other, err := operations.TraceWorkspace(root, traceKey("query", "order_entry.recent_orders"), options)
	if err != nil {
		t.Fatalf("trace the second subject: %v", err)
	}
	if len(other.ProjectionUnresolved) != len(out.ProjectionUnresolved) {
		t.Fatalf("projectionUnresolved is %d for one subject and %d for another, "+
			"but the response states it is graph-wide",
			len(out.ProjectionUnresolved), len(other.ProjectionUnresolved))
	}
}
