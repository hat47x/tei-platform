package exampleslide

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestReviewTurnsPartialDataIntoEvidenceQuestion(t *testing.T) {
	input := keyedInput()
	input.Example.CompleteTables = nil
	packet, err := Review(input)
	if err != nil {
		t.Fatal(err)
	}
	if packet.CaseID != "multi-key" || packet.ModelRevision != input.Catalog.Revision {
		t.Fatalf("missing revision binding: %+v", packet)
	}
	if len(packet.Issues) != 1 {
		t.Fatalf("expected one review issue, got %d", len(packet.Issues))
	}
	issue := packet.Issues[0]
	if issue.Result != "unresolved" || issue.Reason != "target-table-is-partial" ||
		issue.ReviewTarget != "example-completeness" {
		t.Fatalf("incorrect review routing: %+v", issue)
	}
	if issue.RelationID != "child.parent" || issue.SourceRecordID != "b" ||
		issue.Stage != "base" || len(issue.FrameIDs) != 1 ||
		issue.FrameIDs[0] != "slide-1" {
		t.Fatalf("lost exact source linkage: %+v", issue)
	}
	if issue.Question == "" || len(issue.EvidenceNeeded) == 0 {
		t.Fatal("review has no actionable question/evidence")
	}
	// The derived question MUST NOT repair the partial completeness declaration.
	if len(input.Example.CompleteTables) != 0 {
		t.Fatal("review modified the authored Case")
	}
}

func TestReviewContradictionNeedsDecisionNotAutomaticPatch(t *testing.T) {
	input := keyedInput()
	extra := input.Example.Base[0]
	extra.ID = "also-a"
	input.Example.Base = append(input.Example.Base, extra)
	packet, err := Review(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Issues) != 1 {
		t.Fatalf("expected contradiction, got %+v", packet.Issues)
	}
	issue := packet.Issues[0]
	if issue.Result != "contradiction" ||
		issue.ReviewTarget != "cross-view-contradiction" ||
		issue.Reason != "multiple-targets-for-one" {
		t.Fatalf("contradiction was not escalated for review: %+v", issue)
	}
	serialized, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"\"accepted\"", "\"patch\"", "\"resolved\""} {
		if strings.Contains(string(serialized), forbidden) {
			t.Fatalf("inferred an accepted decision or patch: %s", serialized)
		}
	}
}

func TestReviewDeduplicatesRepeatedStageButRetainsFrames(t *testing.T) {
	input := keyedInput()
	input.Example.CompleteTables = nil
	input.Presentation.Frames = []Frame{
		{ID: "z", Stage: "base"},
		{ID: "a", Stage: "base"},
	}
	packet, err := Review(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Issues) != 1 || !reflect.DeepEqual(packet.Issues[0].FrameIDs, []string{"a", "z"}) {
		t.Fatalf("lost slide provenance or duplicated question: %+v", packet.Issues)
	}
	firstID := packet.Issues[0].ID
	input.Presentation.Frames[0], input.Presentation.Frames[1] =
		input.Presentation.Frames[1], input.Presentation.Frames[0]
	again, err := Review(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(packet, again) || again.Issues[0].ID != firstID {
		t.Fatalf("review changed after slide reorder: before=%+v, after=%+v", packet, again)
	}
}

func TestReviewIssueIdentityBindsRevisionAndStage(t *testing.T) {
	id := reviewIssueID("case", "rev1", "base", "rel", "record", "target-table-is-partial")
	if id == reviewIssueID("case", "rev2", "base", "rel", "record", "target-table-is-partial") {
		t.Fatal("model revision was omitted from review ID")
	}
	if id == reviewIssueID("case", "rev1", "later", "rel", "record", "target-table-is-partial") {
		t.Fatal("stage was omitted from review ID")
	}
	if id == reviewIssueID("ca:se", "rev1", "base", "rel", "record", "target-table-is-partial") {
		t.Fatal("case identity was omitted from review ID")
	}
}

func TestReviewPreservesAuthoredQuestionsSeparately(t *testing.T) {
	input := fixture(t)
	packet, err := Review(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Authored) != 1 || packet.Authored[0].ID != "residual" {
		t.Fatalf("authored question disappeared: %+v", packet.Authored)
	}
	if len(packet.Issues) != 0 {
		t.Fatalf("fabricated semantic-relation findings: %+v", packet.Issues)
	}
}

func TestReviewDoesNotMutateCase(t *testing.T) {
	input := keyedInput()
	before, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Review(input); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("review changed original evidence or model input")
	}
}

func TestUnknownReasonStillProducesReviewableQuestion(t *testing.T) {
	finding := RelationFinding{Reason: "some-new-reason"}
	q, evidence, target := proposedQuestion(finding)
	if q == "" || len(evidence) == 0 || target != "unresolved" {
		t.Fatalf("unrecognized reason silently discarded: %q, %v, %q", q, evidence, target)
	}
}
