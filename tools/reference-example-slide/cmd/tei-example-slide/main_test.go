package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	example "tei/tools/reference-example-slide"
)

func TestSyntheticExampleCreatesThreeSlides(t *testing.T) {
	out := t.TempDir()
	input := "../../testdata/partial_inspection.json"
	if err := run(input, "", false, out); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		"projection.json", "review.json", "slide-001.svg", "slide-002.svg", "slide-003.svg",
	} {
		bytes, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Fatal(err)
		}
		if len(bytes) == 0 {
			t.Fatalf("empty generated file %s", file)
		}
	}
}

func TestRealWorldDoesNotRewriteStaleCase(t *testing.T) {
	out := t.TempDir()
	input := "../../testdata/partial_inspection.json"
	world := "../../../../reference/world.json"
	err := run(input, world, false, out)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected World/example revision mismatch, got %v", err)
	}
	files, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("stale case produced output files: %d", len(files))
	}
}

func TestReviewPacketIncludesAuthoredQuestion(t *testing.T) {
	out := t.TempDir()
	input := "../../testdata/partial_inspection.json"
	if err := run(input, "", false, out); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(out, "review.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "\"authoredQuestions\"") ||
		!strings.Contains(string(content), "\"modelRevision\"") {
		t.Fatalf("review packet missing case question/revision: %s", content)
	}
}

func TestCLIExportsOnlyUnverifiedAnswerHandoff(t *testing.T) {
	inputPath := "../../testdata/partial_inspection.json"
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	var input example.Input
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	review, err := example.Review(input)
	if err != nil {
		t.Fatal(err)
	}
	answers := example.AnswerInput{
		Schema: "tei.reference.example-review-answer/v0",
		CaseID: review.CaseID, ModelRevision: review.ModelRevision,
		ReviewFingerprint: review.Fingerprint,
		Answers: []example.AnswerEntry{{
			Question: example.QuestionRef{Kind: "authored", ID: "residual"},
			Response: "担当者に確認する", Context: "一部入荷・検収",
			Disposition: "investigate",
			ClaimedBy:   example.AnswerActor{Kind: "human", ID: "unverified-actor"},
			Evidence:    []example.EvidencePointer{{Kind: "document", Ref: "doc:minutes"}},
		}},
	}
	encoded, err := json.Marshal(answers)
	if err != nil {
		t.Fatal(err)
	}
	answerPath := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(answerPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	if err := run(inputPath, "", false, output, answerPath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(output, "answer-handoff.json"))
	if err != nil {
		t.Fatal(err)
	}
	var handoff example.AnswerHandoff
	if err := json.Unmarshal(data, &handoff); err != nil {
		t.Fatal(err)
	}
	if handoff.ReviewFingerprint != review.Fingerprint || len(handoff.Answers) != 1 ||
		handoff.Answers[0].Status != "unverified" {
		t.Fatalf("CLI incorrectly promoted answer: %+v", handoff)
	}
	triageRaw, err := os.ReadFile(filepath.Join(output, "change-triage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var triage example.ChangeTriage
	if err := json.Unmarshal(triageRaw, &triage); err != nil {
		t.Fatal(err)
	}
	if triage.AnswerFingerprint != handoff.Fingerprint || len(triage.Items) != 1 ||
		triage.Items[0].CandidateReady {
		t.Fatalf("CLI incorrectly generated accepted semantic change: %+v", triage)
	}
}

func TestCLIRejectsUnboundAnswersBeforeWriting(t *testing.T) {
	inputPath := "../../testdata/partial_inspection.json"
	answerPath := filepath.Join(t.TempDir(), "bad.json")
	// Supplying guessed fingerprints must not create even partial slides.
	if err := os.WriteFile(answerPath, []byte(`{"schema":"tei.reference.example-review-answer/v0","caseId":"partial-inspection","modelRevision":"synthetic-2026-10-10","reviewFingerprint":"sha256:guessed","answers":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	if err := run(inputPath, "", false, output, answerPath); err == nil {
		t.Fatal("invalid answer was not rejected")
	}
	files, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("created %d files before validating answers", len(files))
	}

	// Strict JSON decoding rejects unknown fields in answer input.
	if err := os.WriteFile(answerPath, []byte(`{"surprise":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, "", false, output, answerPath); err == nil {
		t.Fatal("unknown answer field was accepted")
	}
}

func TestCLIRejectsUnknownCaseInputBeforeOutput(t *testing.T) {
	source, err := os.ReadFile("../../testdata/partial_inspection.json")
	if err != nil {
		t.Fatal(err)
	}
	// A misspelled completeness property would silently invalidate
	// the claim that all target records were included in the Case.
	var root map[string]any
	if err := json.Unmarshal(source, &root); err != nil {
		t.Fatal(err)
	}
	ex := root["example"].(map[string]any)
	ex["completeTable"] = []any{"purchase_order"}
	encoded, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(t.TempDir(), "case.json")
	if err := os.WriteFile(in, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := run(in, "", false, out); err == nil {
		t.Fatal("unknown case field was silently accepted")
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("wrote outputs before validating Case schema")
	}
}

func TestCLIExportsModelHypothesesWithERReferences(t *testing.T) {
	file := filepath.Join(t.TempDir(), "proposals.json")
	proposals := example.ModelHypotheses{
		Schema: "tei.reference.example-model-hypotheses/v0",
		CandidateKeys: []example.CandidateKeyHypothesis{{
			ID: "purchase-number-candidate", Table: "purchase_order", Fields: []string{"number"},
		}},
		FunctionalDependencies: []example.FDHypothesis{{
			ID: "quantity-to-number", Table: "purchase_order",
			Determinants: []string{"quantity"}, Dependents: []string{"number"},
		}},
	}
	encoded, err := json.Marshal(proposals)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := run("../../testdata/partial_inspection.json", "", false, out, "", file); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "model-hypothesis-review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var review example.ModelHypothesisReview
	if err := json.Unmarshal(raw, &review); err != nil {
		t.Fatal(err)
	}
	if len(review.Issues) != 2 {
		t.Fatalf("missing both proposed model questions: %+v", review)
	}
	for _, issue := range review.Issues {
		if issue.Result != "unresolved" || issue.Stage != "base" ||
			len(issue.DiagramPlacementIDs) != 1 || issue.DiagramPlacementIDs[0] != "po-box" ||
			len(issue.FrameIDs) != 1 || issue.FrameIDs[0] != "first" {
			t.Fatalf("unexpected slide mapping or unverified rule promotion: %+v", issue)
		}
	}
	info, err := os.Stat(filepath.Join(out, "model-hypothesis-review.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("hypothesis review must remain private: %v", info.Mode().Perm())
	}
	// New review output does not modify the pre-existing answer fingerprint.
	without := t.TempDir()
	if err := run("../../testdata/partial_inspection.json", "", false, without); err != nil {
		t.Fatal(err)
	}
	existing, err := os.ReadFile(filepath.Join(without, "review.json"))
	if err != nil {
		t.Fatal(err)
	}
	alsoExisting, err := os.ReadFile(filepath.Join(out, "review.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(existing) != string(alsoExisting) {
		t.Fatal("adding optional hypotheses altered existing answer review")
	}
}

func TestCLIRejectsBadModelHypothesesBeforeWritingAnyOutput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "invalid-proposals.json")
	out := t.TempDir()
	if err := os.WriteFile(file, []byte(`{"schema":"tei.reference.example-model-hypotheses/v0","candidateKeys":[{"id":"k","table":"purchase_order","fields":["unknown"]}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run("../../testdata/partial_inspection.json", "", false, out, "", file); err == nil {
		t.Fatal("unknown candidate field was not rejected")
	}
	if entries, err := os.ReadDir(out); err != nil || len(entries) != 0 {
		t.Fatalf("wrote a partial output for invalid key proposal: %v / %v", entries, err)
	}
	if err := os.WriteFile(file, []byte(`{"schema":"tei.reference.example-model-hypotheses/v0","candidateKeys":[],"surprise":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run("../../testdata/partial_inspection.json", "", false, out, "", file); err == nil {
		t.Fatal("unknown proposal JSON field was not rejected")
	}
	if err := run("../../testdata/partial_inspection.json", "", false, "", "", file); err == nil {
		t.Fatal("missing output directory was accepted for hypotheses")
	}
	if err := run("", "../../../../reference/world.json", true, out, "", file); err == nil {
		t.Fatal("catalog-only mode accepted model hypotheses")
	}
}
