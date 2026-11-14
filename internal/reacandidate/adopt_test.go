package reacandidate

import (
	"testing"

	"tei/internal/threeelement"
)

func testModel() threeelement.ThreeElementModel {
	return threeelement.NewThreeElementModel("probe-model", "Probe model")
}

// The preview is an in-memory proposal: it adds the candidates to a copy and
// records everything an Owner needs, but it accepts nothing and writes
// nothing.
func TestPreviewAdoptionRecordsEverythingAndAcceptsNothing(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	before := testModel()
	preview, err := PreviewAdoption(set, before)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(preview.After.DataElements) != 2 || len(preview.After.CrossReferences) != 1 {
		t.Fatalf("after model = %d data %d cross-references",
			len(preview.After.DataElements), len(preview.After.CrossReferences))
	}
	if len(before.DataElements) != 0 || len(before.CrossReferences) != 0 {
		t.Fatal("the reviewed model was mutated by the preview")
	}
	if preview.Before.ModelFingerprint == preview.After.ModelFingerprint {
		t.Fatal("before and after fingerprints are identical")
	}
	if len(preview.Diff.Added) != 3 {
		t.Fatalf("diff added = %d, want 3 (2 data + 1 cross-reference)", len(preview.Diff.Added))
	}
	for _, record := range preview.Records {
		if record.Accepted {
			t.Fatalf("record %s is marked accepted", record.CandidateID)
		}
		if record.Owner != "" {
			t.Fatalf("record %s names an owner: %q", record.CandidateID, record.Owner)
		}
		if record.BeforeModelFingerprint == "" || record.AfterModelFingerprint == "" {
			t.Fatalf("record %s lost a model fingerprint", record.CandidateID)
		}
		if len(record.Basis) == 0 || len(record.Unresolved) == 0 || len(record.MeaningLoss) == 0 {
			t.Fatalf("record %s lost basis/unresolved/meaning-loss: %+v", record.CandidateID, record)
		}
		if record.SourceRevision.ArtifactPath == "" {
			t.Fatalf("record %s lost the source revision", record.CandidateID)
		}
	}
	if preview.After.DataElements[0].ID == "" {
		t.Fatal("preview element has no identity")
	}
	for _, element := range preview.After.DataElements {
		if !hasString(element.AuthorityLimits, "rea-candidate-preview-not-accepted") {
			t.Fatalf("element %s lost the not-accepted limit", element.ID)
		}
	}
}

// An unresolved endpoint is reported by the model validation as a dangling
// reference; the preview does not repair it by inventing an element.
func TestPreviewAdoptionReportsUnresolvedEndpointAsDanglingReference(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes = fixture.nodes[:1]
	fixture.edges[0].to = "node-missing"
	set := mustGenerate(t, fixture)
	preview, err := PreviewAdoption(set, testModel())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(preview.After.DataElements) != 1 {
		t.Fatalf("after data elements = %d, want 1", len(preview.After.DataElements))
	}
	reference := preview.After.CrossReferences[0]
	if reference.ToID != "unresolved:node-missing" {
		t.Fatalf("unresolved endpoint rewritten to %q", reference.ToID)
	}
	codes := map[string]bool{}
	for _, diagnostic := range preview.AfterValidation.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes["UNKNOWN_CROSS_REF_TO"] {
		t.Fatalf("after validation does not report the dangling reference: %+v", preview.AfterValidation.Diagnostics)
	}
}

// Candidates that share one external node identity are grouped structurally,
// and the grouping is recorded rather than hidden.
func TestPreviewAdoptionGroupsByExternalNodeAndRecordsIt(t *testing.T) {
	fixture := defaultEvidence()
	second := fixture.nodes[0].observations[0]
	second.id = "obs-order-again"
	fixture.nodes[0].observations = append(fixture.nodes[0].observations, second)
	set := mustGenerate(t, fixture)
	preview, err := PreviewAdoption(set, testModel())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(preview.After.DataElements) != 2 {
		t.Fatalf("after data elements = %d, want 2 (one per external node)", len(preview.After.DataElements))
	}
	merged := 0
	for _, record := range preview.Records {
		if len(record.MergedFromCandidates) > 1 {
			merged++
			if len(record.MergedFromCandidates) != 2 {
				t.Fatalf("record merged %d candidates", len(record.MergedFromCandidates))
			}
		}
	}
	if merged != 2 {
		t.Fatalf("merged records = %d, want 2 (both candidates of the shared node)", merged)
	}
}

// The comparison is revision- and identity-aware. Identical current bytes in
// a different attempt are NOT evidence of continuity and do not revive
// anything.
func TestCompareCandidateSetsSameBytesDifferentAttempt(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	second := defaultEvidence()
	second.evidenceID = "ev-minimal-2"
	afterSet := mustGenerate(t, second)
	comparison := CompareCandidateSets(first, afterSet)
	if len(comparison.Entries) != len(first.Candidates) {
		t.Fatalf("entries = %d, candidates = %d", len(comparison.Entries), len(first.Candidates))
	}
	for _, entry := range comparison.Entries {
		if entry.Status != ComparisonSameBytesNewAttempt {
			t.Fatalf("entry %s status = %q", entry.CandidateID, entry.Status)
		}
		if !hasString(entry.Notes, "no-prior-review-grant-or-acceptance-is-revived") {
			t.Fatalf("entry %s does not refuse revival: %v", entry.CandidateID, entry.Notes)
		}
	}
}

// A changed source revision keeps the candidate identity and is reported as a
// revision change that requires fresh review.
func TestCompareCandidateSetsRevisionChange(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	changed := defaultEvidence()
	changed.nodes[0].observations[0].artifact = digestB
	afterSet := mustGenerate(t, changed)
	comparison := CompareCandidateSets(first, afterSet)
	statuses := map[string]int{}
	for _, entry := range comparison.Entries {
		statuses[entry.Status]++
	}
	if statuses[ComparisonRevisionChanged] != 1 {
		t.Fatalf("revision-changed entries = %d, want 1 (%v)", statuses[ComparisonRevisionChanged], statuses)
	}
	if statuses[ComparisonUnchanged] != 2 {
		t.Fatalf("unchanged entries = %d, want 2 (%v)", statuses[ComparisonUnchanged], statuses)
	}
	for _, entry := range comparison.Entries {
		if entry.Status == ComparisonRevisionChanged &&
			!hasString(entry.Notes, "fresh-owner-review-is-required") {
			t.Fatalf("entry %s does not require fresh review", entry.CandidateID)
		}
	}
}

// A→B→A: the bytes return to the original content. The comparison reports the
// current equality and explicitly refuses to claim that B never existed.
func TestCompareCandidateSetsReturnToOriginalBytes(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	changed := defaultEvidence()
	changed.nodes[0].observations[0].artifact = digestB
	intermediate := mustGenerate(t, changed)
	if len(intermediate.Candidates) != len(first.Candidates) {
		t.Fatalf("intermediate candidates = %d", len(intermediate.Candidates))
	}
	back := defaultEvidence()
	back.evidenceID = "ev-minimal-3"
	returned := mustGenerate(t, back)
	comparison := CompareCandidateSets(first, returned)
	for _, entry := range comparison.Entries {
		if entry.Status != ComparisonSameBytesNewAttempt {
			t.Fatalf("entry %s status = %q", entry.CandidateID, entry.Status)
		}
		if !hasString(entry.Notes, "identical-current-bytes-are-not-evidence-that-nothing-changed-in-between") {
			t.Fatalf("entry %s does not refuse the no-intermediate-change claim: %v", entry.CandidateID, entry.Notes)
		}
	}
	// The intermediate revision is a genuinely different current state.
	intermediateComparison := CompareCandidateSets(first, intermediate)
	changedCount := 0
	for _, entry := range intermediateComparison.Entries {
		if entry.Status == ComparisonRevisionChanged {
			changedCount++
		}
	}
	if changedCount != 1 {
		t.Fatalf("intermediate revision-changed entries = %d, want 1", changedCount)
	}
}

// A renumbered external identity over the same current bytes is reported as a
// renumbering, not as a removal plus an addition.
func TestCompareCandidateSetsExternalIdentityRenumbering(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	renumbered := defaultEvidence()
	renumbered.evidenceID = "ev-minimal-4"
	for i := range renumbered.nodes {
		renumbered.nodes[i].id = renumbered.nodes[i].id + "-r2"
		for j := range renumbered.nodes[i].observations {
			renumbered.nodes[i].observations[j].id = renumbered.nodes[i].observations[j].id + "-r2"
		}
	}
	renumbered.edges[0].from = renumbered.edges[0].from + "-r2"
	renumbered.edges[0].to = renumbered.edges[0].to + "-r2"
	renumbered.edges[0].id = renumbered.edges[0].id + "-r2"
	afterSet := mustGenerate(t, renumbered)
	comparison := CompareCandidateSets(first, afterSet)
	if len(comparison.Entries) != len(first.Candidates) {
		t.Fatalf("entries = %d, want %d", len(comparison.Entries), len(first.Candidates))
	}
	for _, entry := range comparison.Entries {
		if entry.Status != ComparisonRenumbered {
			t.Fatalf("entry %s status = %q, want renumbered", entry.CandidateID, entry.Status)
		}
		if !hasString(entry.Notes, "a-renumbered-candidate-requires-fresh-owner-review") {
			t.Fatalf("entry %s does not require fresh review: %v", entry.CandidateID, entry.Notes)
		}
	}
}

// Structural fallback may pair a reversed edge as a possible external-ID
// renumbering. That comparison is only a review hint: it must explicitly
// identify the unverified endpoints, not quietly qualify Source editing.
func TestCompareCandidateSetsReversedEdgeNeedsEndpointMapping(t *testing.T) {
	before := mustGenerate(t, defaultEvidence())
	reversed := defaultEvidence()
	reversed.evidenceID = "ev-reversed-edge"
	reversed.edges[0].from, reversed.edges[0].to = reversed.edges[0].to, reversed.edges[0].from
	after := mustGenerate(t, reversed)
	comparison := CompareCandidateSets(before, after)
	var paired int
	for _, entry := range comparison.Entries {
		if entry.Status != ComparisonRenumbered {
			continue
		}
		paired++
		if !hasString(entry.Notes, "relation-endpoint-mapping-is-unverified") ||
			!hasString(entry.Notes, "source-backed-endpoint-mapping-is-required-before-edit") ||
			!hasString(entry.Notes, "a-renumbered-candidate-requires-fresh-owner-review") {
			t.Fatalf("reversed edge received an unqualified renumber claim: %+v", entry)
		}
	}
	if paired != 1 {
		t.Fatalf("reversed edge pairing = %d, want one qualified review hint: %+v", paired, comparison.Entries)
	}
}

// Added and removed candidates are reported without a structural match.
func TestCompareCandidateSetsAddedAndRemoved(t *testing.T) {
	first := mustGenerate(t, defaultEvidence())
	grown := defaultEvidence()
	grown.evidenceID = "ev-minimal-5"
	grown.nodes = append(grown.nodes, nodeFixture{
		id:   "node-extra",
		kind: "module",
		observations: []observationFixture{{
			id: "obs-extra", state: "observed", authority: "derived",
			artifact: digestA, locKind: "source-range", locSource: "src/extra.js",
			startLine: 1, startCol: 1, endLine: 2, endCol: 1, coverage: "complete",
		}},
	})
	afterSet := mustGenerate(t, grown)
	comparison := CompareCandidateSets(first, afterSet)
	statuses := map[string]int{}
	for _, entry := range comparison.Entries {
		statuses[entry.Status]++
	}
	if statuses[ComparisonAdded] != 1 {
		t.Fatalf("added = %d, want 1 (%v)", statuses[ComparisonAdded], statuses)
	}
	if statuses[ComparisonSameBytesNewAttempt] != 3 {
		t.Fatalf("same-bytes-different-attempt = %d, want 3 (%v)", statuses[ComparisonSameBytesNewAttempt], statuses)
	}
	reverse := CompareCandidateSets(afterSet, first)
	statuses = map[string]int{}
	for _, entry := range reverse.Entries {
		statuses[entry.Status]++
	}
	if statuses[ComparisonRemoved] != 1 {
		t.Fatalf("removed = %d, want 1 (%v)", statuses[ComparisonRemoved], statuses)
	}
}

// The preview diff is the existing three-element diff, so a reviewer reads one
// diff vocabulary for adoption comparison.
func TestPreviewAdoptionUsesExistingModelDiff(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	preview, err := PreviewAdoption(set, testModel())
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Diff.SchemaVersion != "tei.three-element.diff/v0" {
		t.Fatalf("diff schema = %q", preview.Diff.SchemaVersion)
	}
	details := map[string]bool{}
	for _, entry := range preview.Diff.Added {
		details[entry.Detail] = true
	}
	if !details["data element"] {
		t.Fatalf("diff details = %v", details)
	}
	crossReference := false
	for _, entry := range preview.Diff.Added {
		if entry.Category == "cross-reference" {
			crossReference = true
		}
	}
	if !crossReference {
		t.Fatalf("diff has no cross-reference entry: %+v", preview.Diff.Added)
	}
}
