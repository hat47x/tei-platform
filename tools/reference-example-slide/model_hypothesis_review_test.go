package exampleslide

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func reviewModelFixture() (Input, ModelHypotheses) {
	catalog, example, fd := fdProbeFixture()
	in := Input{
		Catalog: catalog, Example: example,
		Diagram: Diagram{ID: "er", Placements: []Placement{
			{ID: "er-right", Table: "order_line", X: 200},
			{ID: "er-left", Table: "order_line", X: 10, Hide: []string{"order_id"}},
		}},
		Presentation: Presentation{Frames: []Frame{
			{ID: "later", Stage: "later"},
			{ID: "base-2", Stage: "base"},
			{ID: "base-1", Stage: "base"},
		}},
	}
	in.Example.Stages = []Stage{{ID: "later"}}
	return in, ModelHypotheses{
		Schema: modelHypothesesSchema,
		CandidateKeys: []CandidateKeyHypothesis{{
			ID: "business-line-key", Table: "order_line",
			Fields: []string{"order_id", "line_no"},
		}},
		FunctionalDependencies: []FDHypothesis{fd},
	}
}

func TestModelHypothesisReviewLinksProvenCollisionsToERPlacements(t *testing.T) {
	in, hypotheses := reviewModelFixture()
	packet, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Schema != "tei.reference.example-model-hypothesis-review/v0" ||
		packet.CaseID != in.Example.ID || packet.ModelRevision != in.Catalog.Revision ||
		!strings.HasPrefix(packet.Fingerprint, "sha256:") || len(packet.Issues) != 2 {
		t.Fatalf("invalid review envelope: %+v", packet)
	}
	for _, issue := range packet.Issues {
		if issue.Result != "contradiction" || issue.Stage != "base" ||
			!reflect.DeepEqual(issue.DiagramPlacementIDs, []string{"er-left", "er-right"}) ||
			!reflect.DeepEqual(issue.FrameIDs, []string{"base-1", "base-2"}) ||
			!reflect.DeepEqual(issue.RecordPairs, []FDRecordPair{{LeftID: "record-a", RightID: "record-b"}}) ||
			issue.ID == "" || issue.Question == "" || len(issue.EvidenceNeeded) == 0 {
			t.Fatalf("lost ER identity or evidence: %+v", issue)
		}
	}
	if packet.Issues[0].Kind != "candidate-key" ||
		packet.Issues[1].Kind != "functional-dependency" ||
		packet.Issues[0].ReviewTarget != "business-identity-proposal" {
		t.Fatalf("model hypotheses were conflated or accepted: %+v", packet.Issues)
	}
}

func TestModelHypothesisReviewStableAcrossPresentationAndProposalOrdering(t *testing.T) {
	in, hypotheses := reviewModelFixture()
	original, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	in.Presentation.Frames[0], in.Presentation.Frames[2] =
		in.Presentation.Frames[2], in.Presentation.Frames[0]
	in.Diagram.Placements[0], in.Diagram.Placements[1] =
		in.Diagram.Placements[1], in.Diagram.Placements[0]
	hypotheses.CandidateKeys[0].Fields[0], hypotheses.CandidateKeys[0].Fields[1] =
		hypotheses.CandidateKeys[0].Fields[1], hypotheses.CandidateKeys[0].Fields[0]
	hypotheses.FunctionalDependencies[0].Determinants[0], hypotheses.FunctionalDependencies[0].Determinants[1] =
		hypotheses.FunctionalDependencies[0].Determinants[1], hypotheses.FunctionalDependencies[0].Determinants[0]
	reordered, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, reordered) {
		t.Fatalf("layout or unordered hypothesis components changed review: before=%+v after=%+v", original, reordered)
	}
}

func TestModelHypothesisReviewBindsToActualExampleAndHypothesis(t *testing.T) {
	in, hypotheses := reviewModelFixture()
	before, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	in.Example.Base[0].Values["item"] = json.RawMessage(`"item-A"`)
	changed, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	if before.Fingerprint == changed.Fingerprint ||
		before.Issues[0].ID == changed.Issues[0].ID ||
		changed.Issues[1].Result != "unresolved" {
		t.Fatalf("stale example witness was reused: before=%+v after=%+v", before, changed)
	}
	in, hypotheses = reviewModelFixture()
	hypotheses.CandidateKeys[0].Fields = []string{"row_id"}
	differentQuestion, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	if differentQuestion.Issues[0].Result != "unresolved" ||
		before.HypothesesFingerprint == differentQuestion.HypothesesFingerprint ||
		before.Issues[0].ID == differentQuestion.Issues[0].ID {
		t.Fatalf("new key proposal was not kept distinct: %+v", differentQuestion)
	}
}

func TestModelHypothesisReviewSupportsOffDiagramTablesWithoutFabricatingPlacement(t *testing.T) {
	in, hypotheses := reviewModelFixture()
	in.Diagram.Placements = nil
	in.Presentation.Frames = []Frame{{ID: "late-only", Stage: "later"}}
	got, err := ReviewModelHypotheses(in, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range got.Issues {
		if len(issue.DiagramPlacementIDs) != 0 || len(issue.FrameIDs) != 0 ||
			issue.Stage != "base" {
			t.Fatalf("invented placement or later-stage evidence: %+v", issue)
		}
	}
}

func TestModelHypothesisReviewRejectsStaleOrUnvalidatedProposals(t *testing.T) {
	type modify func(*Input, *ModelHypotheses)
	tests := []struct {
		name string
		edit modify
	}{
		{"schema", func(_ *Input, h *ModelHypotheses) { h.Schema = "wrong" }},
		{"empty", func(_ *Input, h *ModelHypotheses) { h.CandidateKeys = nil; h.FunctionalDependencies = nil }},
		{"unknown-attribute", func(_ *Input, h *ModelHypotheses) {
			h.CandidateKeys[0].Fields = []string{"not-an-attribute"}
		}},
		{"duplicate-key", func(_ *Input, h *ModelHypotheses) {
			h.CandidateKeys = append(h.CandidateKeys, h.CandidateKeys[0])
		}},
		{"invalid-FD", func(_ *Input, h *ModelHypotheses) { h.FunctionalDependencies[0].Dependents = nil }},
		{"revision", func(in *Input, _ *ModelHypotheses) { in.Example.ModelRevision = "stale" }},
		{"diagram", func(in *Input, _ *ModelHypotheses) { in.Diagram.Placements[0].Table = "unknown" }},
		{"stage", func(in *Input, _ *ModelHypotheses) { in.Presentation.Frames[0].Stage = "unknown" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, h := reviewModelFixture()
			tt.edit(&in, &h)
			if _, err := ReviewModelHypotheses(in, h); err == nil {
				t.Fatal("invalid model proposal or illustrative source accepted")
			}
		})
	}
}

func TestModelHypothesisReviewDoesNotMutateInputsOrExistingAnswerFingerprint(t *testing.T) {
	in, hypotheses := reviewModelFixture()
	before, err := json.Marshal(struct {
		Input Input
		Model ModelHypotheses
	}{in, hypotheses})
	if err != nil {
		t.Fatal(err)
	}
	reviewBefore, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReviewModelHypotheses(in, hypotheses); err != nil {
		t.Fatal(err)
	}
	reviewAfter, err := Review(in)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(struct {
		Input Input
		Model ModelHypotheses
	}{in, hypotheses})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) ||
		reviewBefore.Fingerprint != reviewAfter.Fingerprint {
		t.Fatal("model hypothesis review changed its inputs or the existing answer binding")
	}
}
