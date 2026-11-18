package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

const authoringRequiredFile = "semantic/studio/inspection-response.tei.md"
const authoringRequiredID = "studio.inspection_response.snapshot_fingerprint"

func requiredAuthoringWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "studio-scratch")
	writeFile(t, filepath.Join(root, "tei.workspace.yaml"),
		"workspace: studio_reader\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n")
	source := "# Authoring Studio model\n\n" +
		"The word no in this documentation must survive exactly.\n\n" +
		"<!-- tei: data InspectionResponse in studio -->\n" +
		"| attribute | type | required | display |\n" +
		"| --- | --- | --- | --- |\n" +
		"| schemaVersion | std.text | yes | Schema |\n" +
		"| snapshotFingerprint | std.text | no | Snapshot |\n" +
		"\nUnmarked table (not a declaration):\n\n| required | other |\n| --- | --- |\n| no | preserved |\n"
	writeFile(t, filepath.Join(root, filepath.FromSlash(authoringRequiredFile)), source)
	return root, source
}

func requiredCandidate(revision string, value *bool) Candidate {
	return Candidate{
		Schema: candidateSchema, ID: "studio-required-change",
		BaseModelRevision: revision,
		Origin: Origin{Kind: "studio", Actor: "test"},
		Intent: "Change the required property of the existing Data attribute without touching other Source",
		Delta: []Operation{{Op: opSetAttributeRequired, ID: authoringRequiredID, Required: value}},
	}
}

func inspectedRequired(t *testing.T, root string) bool {
	t.Helper()
	v, err := operations.InspectSubject(root, sourceindex.Subject{Kind: "semantic-data", ID: "studio.inspection_response"})
	if err != nil {
		t.Fatal(err)
	}
	// The Canonical bytes are what the existing Inspector Reader makes visible.
	// Test the expected boolean by inspecting a focused real Operation result.
	text := string(v.Canonical)
	if !strings.Contains(text, "\"id\":\"studio.inspection_response\"") {
		t.Fatalf("wrong Canonical Data: %s", text)
	}
	return strings.Contains(text, "\"id\":\"studio.inspection_response.snapshot_fingerprint\",\"symbol\":\"snapshotFingerprint\"") &&
		strings.Contains(text, "\"required\":true")
}

func TestStudioRequiredCandidatePreviewAcceptAndRealReload(t *testing.T) {
	root, source := requiredAuthoringWorkspace(t)
	beforeTree := treeDigest(t, root)
	if inspectedRequired(t, root) {
		t.Fatal("baseline required must be false")
	}
	yes := true
	raw := mustJSON(t, requiredCandidate(revisionOfDir(t, root), &yes))
	ev, err := evaluate(root, raw)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview must accept a single exact Data cell: err=%v diag=%+v", err, allDiagnostics(ev.preview))
	}
	if len(ev.preview.SourceEdits) != 1 {
		t.Fatalf("preview returned %d Source edits", len(ev.preview.SourceEdits))
	}
	edit := ev.preview.SourceEdits[0]
	if edit.Path != authoringRequiredFile || edit.Created || edit.InsertedText != "yes" || edit.RemovedText != "no" {
		t.Fatalf("preview wrote the wrong Source location: %+v", edit)
	}
	if !reflect.DeepEqual(ev.preview.ChangedCollections, []string{"semantics"}) {
		t.Fatalf("unexpected Canonical collections changed: %+v", ev.preview.ChangedCollections)
	}
	assertTreeUnchanged(t, beforeTree, treeDigest(t, root))

	decision, err := accept(root, raw, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accepted Source not written: %+v %v", decision, err)
	}
	expected := strings.Replace(source, "| snapshotFingerprint | std.text | no | Snapshot |",
		"| snapshotFingerprint | std.text | yes | Snapshot |", 1)
	updated, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(authoringRequiredFile)))
	if err != nil || string(updated) != expected {
		t.Fatalf("Source sugar/prose was changed or the target row was not saved: %v\n%s", err, updated)
	}
	if !inspectedRequired(t, root) {
		t.Fatal("real InspectSubject did not reload the changed Data property")
	}
	changedTree := treeDigest(t, root)
	changed := []string{}
	for name, digest := range beforeTree {
		if changedTree[name] != digest {
			changed = append(changed, name)
		}
	}
	// The Acceptance Decision ledger may add its own records. Source file
	// edits, however, must be restricted to the intended Data document.
	if !reflect.DeepEqual(changed, []string{authoringRequiredFile}) {
		t.Fatalf("unexpected existing Source files changed: %v", changed)
	}
	stale, err := evaluate(root, raw)
	if err != nil || stale.preview.Acceptable || !hasCode(diagnosticCodes(stale.preview), "STALE_BASE_MODEL_REVISION") {
		t.Fatalf("previous Studio draft was accepted after Source moved: err=%v diag=%v", err, diagnosticCodes(stale.preview))
	}
}

func TestStudioRequiredCandidateFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name string
		change func(*Candidate)
		code string
	}{
		{"missing explicit boolean", func(c *Candidate) { c.Delta[0].Required = nil }, "CANDIDATE_NORMALIZE"},
		{"unknown attribute", func(c *Candidate) { c.Delta[0].ID = "studio.unknown" }, "ATTRIBUTE_NOT_FOUND"},
		{"unrequested placement", func(c *Candidate) { c.Delta[0].Placement = authoringRequiredFile }, "CANDIDATE_NORMALIZE"},
		{"wrong operation payload", func(c *Candidate) { c.Delta[0].Symbol = "Other" }, "CANDIDATE_NORMALIZE"},
		{"no change", func(c *Candidate) { v := false; c.Delta[0].Required = &v }, "ATTRIBUTE_REQUIRED_UNCHANGED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := requiredAuthoringWorkspace(t)
			before := treeDigest(t, root)
			yes := true
			candidate := requiredCandidate(revisionOfDir(t, root), &yes)
			tc.change(&candidate)
			ev, err := evaluate(root, mustJSON(t, candidate))
			if err != nil || ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), tc.code) {
				t.Fatalf("unexpected evaluation: err=%v acceptable=%t diag=%+v", err, ev.preview.Acceptable, allDiagnostics(ev.preview))
			}
			assertTreeUnchanged(t, before, treeDigest(t, root))
		})
	}
}

func TestStudioRequiredCandidateDoesNotRewriteNativeTEISource(t *testing.T) {
	root := scratchWorkspace(t)
	before := treeDigest(t, root)
	yes := true
	c := requiredCandidate(revisionOfDir(t, root), &yes)
	c.Delta[0].ID = "sales.order_line.order_line_id"
	ev, err := evaluate(root, mustJSON(t, c))
	if err != nil || ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "ATTRIBUTE_SOURCE_FORMAT_UNSUPPORTED") {
		t.Fatalf("native .tei Source was not refused: err=%v diag=%+v", err, allDiagnostics(ev.preview))
	}
	assertTreeUnchanged(t, before, treeDigest(t, root))
}
