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
	if err != nil { t.Fatal(err) }
	var input example.Input
	if err := json.Unmarshal(raw, &input); err != nil { t.Fatal(err) }
	review, err := example.Review(input)
	if err != nil { t.Fatal(err) }
	answers := example.AnswerInput{
		Schema: "tei.reference.example-review-answer/v0",
		CaseID: review.CaseID, ModelRevision: review.ModelRevision,
		ReviewFingerprint: review.Fingerprint,
		Answers: []example.AnswerEntry{{
			Question: example.QuestionRef{Kind:"authored",ID:"residual"},
			Response: "担当者に確認する", Context: "一部入荷・検収",
			Disposition: "investigate",
			ClaimedBy: example.AnswerActor{Kind:"human",ID:"unverified-actor"},
			Evidence: []example.EvidencePointer{{Kind:"document",Ref:"doc:minutes"}},
		}},
	}
	encoded, err := json.Marshal(answers)
	if err != nil { t.Fatal(err) }
	answerPath := filepath.Join(t.TempDir(), "answers.json")
	if err := os.WriteFile(answerPath, encoded, 0600); err != nil { t.Fatal(err) }
	output := t.TempDir()
	if err := run(inputPath, "", false, output, answerPath); err != nil { t.Fatal(err) }
	data, err := os.ReadFile(filepath.Join(output, "answer-handoff.json"))
	if err != nil { t.Fatal(err) }
	var handoff example.AnswerHandoff
	if err := json.Unmarshal(data, &handoff); err != nil { t.Fatal(err) }
	if handoff.ReviewFingerprint != review.Fingerprint || len(handoff.Answers)!=1 ||
		handoff.Answers[0].Status!="unverified" {
		t.Fatalf("CLI incorrectly promoted answer: %+v", handoff)
	}
	triageRaw, err := os.ReadFile(filepath.Join(output, "change-triage.json"))
	if err != nil { t.Fatal(err) }
	var triage example.ChangeTriage
	if err := json.Unmarshal(triageRaw, &triage); err != nil { t.Fatal(err) }
	if triage.AnswerFingerprint != handoff.Fingerprint || len(triage.Items)!=1 ||
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
	files,err:=os.ReadDir(output)
	if err!=nil {t.Fatal(err)}
	if len(files)!=0 { t.Fatalf("created %d files before validating answers",len(files)) }

	// Strict JSON decoding rejects unknown fields in answer input.
	if err := os.WriteFile(answerPath, []byte(`{"surprise":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, "", false, output, answerPath); err == nil {
		t.Fatal("unknown answer field was accepted")
	}
}
