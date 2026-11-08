package operations

import (
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

// traceContinuationFixture is the query used by the tests below. sales.order
// has more than two outgoing edges in the reference Workspace, so a limit of
// two leaves a next page.
func traceContinuationFixture() (string, modelref.Key) {
	return referenceWorkspaceRoot(), modelref.Key{
		Scope:   "continuation",
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order"},
	}
}

// A caller that continues a trace must be able to find the value to send back
// next to nextOffset. The response previously carried two fields named
// snapshotFingerprint: modelref's graph-only one inside trace, and the
// Workspace-wide one at the top level. Only the second is accepted as
// expectedFingerprint, so picking the one beside nextOffset failed with
// STALE_SNAPSHOT and reported a version change that had not happened (#3625).
func TestTraceCarriesTheContinuationFingerprintBesideNextOffset(t *testing.T) {
	root, key := traceContinuationFixture()
	first, err := TraceWorkspace(root, key, modelref.TraceOptions{Mode: "full", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if first.Trace.NextOffset == nil {
		t.Fatalf("the fixture did not paginate: total=%d rows=%d", first.Trace.Total, len(first.Trace.Rows))
	}
	if first.Trace.ContinuationFingerprint == "" {
		t.Fatal("continuationFingerprint is empty")
	}
	if first.Trace.ContinuationFingerprint != first.SnapshotFingerprint {
		t.Fatalf("continuationFingerprint=%q, top-level snapshotFingerprint=%q",
			first.Trace.ContinuationFingerprint, first.SnapshotFingerprint)
	}
	if first.Trace.ContinuationFingerprint == first.Trace.SnapshotFingerprint {
		t.Fatalf("continuationFingerprint must not be the graph-only value: %q",
			first.Trace.ContinuationFingerprint)
	}

	// The value beside nextOffset must actually work as the continuation token.
	second, err := TraceWorkspace(root, key, modelref.TraceOptions{
		Mode: "full", Offset: *first.Trace.NextOffset, Limit: 2,
		ExpectedFingerprint: first.Trace.ContinuationFingerprint,
	})
	if err != nil {
		t.Fatalf("continuing with the advertised value failed: %v", err)
	}
	if len(second.Trace.Rows) == 0 {
		t.Fatal("the second page is empty")
	}

	// The graph-only value is not a continuation token and must stay refused.
	if _, err := TraceWorkspace(root, key, modelref.TraceOptions{
		Mode: "full", Offset: *first.Trace.NextOffset, Limit: 2,
		ExpectedFingerprint: first.Trace.SnapshotFingerprint,
	}); err == nil {
		t.Fatal("the graph-only fingerprint was accepted as a continuation token")
	}
}

// summary mode carries the same value, so a caller that starts with summary and
// then asks for full pages does not have to re-learn which field to use.
func TestTraceSummaryCarriesTheSameContinuationFingerprint(t *testing.T) {
	root, key := traceContinuationFixture()
	summary, err := TraceWorkspace(root, key, modelref.TraceOptions{Mode: "summary"})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Trace.ContinuationFingerprint != summary.SnapshotFingerprint {
		t.Fatalf("summary continuationFingerprint=%q, top-level=%q",
			summary.Trace.ContinuationFingerprint, summary.SnapshotFingerprint)
	}
	if summary.Trace.ContinuationFingerprint == "" {
		t.Fatal("summary continuationFingerprint is empty")
	}
}
