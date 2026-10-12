package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
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

func modelSourceCLIInputs(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := []byte("namespace sales\n\ndata Child {\n    attribute businessCode type std.identifier\n}\n")
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), source, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := example.CatalogFromWorld(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	attribute := catalog.Tables["sales.child"].Fields[0].ID
	input := example.Input{
		Catalog: catalog,
		Example: example.Example{
			ID: "loaded-source-case", ModelRevision: catalog.Revision,
			Base: []example.Record{{
				ID: "row-1", Table: "sales.child",
				Values: map[string]json.RawMessage{attribute: json.RawMessage(`"X"`)},
			}},
		},
		Diagram: example.Diagram{ID: "er", Placements: []example.Placement{{
			ID: "child-shape", Table: "sales.child", X: 10,
		}}},
		Presentation: example.Presentation{Frames: []example.Frame{{ID: "first", Stage: "base"}}},
	}
	proposals := example.ModelHypotheses{
		Schema: "tei.reference.example-model-hypotheses/v0",
		CandidateKeys: []example.CandidateKeyHypothesis{{
			ID: "business-identity", Table: "sales.child", Fields: []string{attribute},
		}},
	}
	inputPath := filepath.Join(t.TempDir(), "input.json")
	proposalsPath := filepath.Join(t.TempDir(), "hypotheses.json")
	for path, value := range map[string]any{inputPath: input, proposalsPath: proposals} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, inputPath, proposalsPath
}

func TestCLIAuditsHypothesisAgainstLoadedWorkspaceSource(t *testing.T) {
	root, inputPath, proposalsPath := modelSourceCLIInputs(t)
	out := t.TempDir()
	if err := run(inputPath, "", false, out, "", proposalsPath, root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "model-hypothesis-source-audit.json"))
	if err != nil {
		t.Fatal(err)
	}
	var audit example.ModelHypothesisSourceAudit
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatal(err)
	}
	reviewData, err := os.ReadFile(filepath.Join(out, "model-hypothesis-review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var review example.ModelHypothesisReview
	if err := json.Unmarshal(reviewData, &review); err != nil {
		t.Fatal(err)
	}
	if audit.ModelReviewFingerprint != review.Fingerprint ||
		audit.HypothesesFingerprint != review.HypothesesFingerprint ||
		len(audit.Findings) != 2 || len(review.Issues) != 1 ||
		review.Issues[0].Result != "unresolved" {
		t.Fatalf("review and source evidence were not separately bound: audit=%+v review=%+v", audit, review)
	}
	for _, finding := range audit.Findings {
		if finding.Status != "source-range-checked" || finding.SourceRange == nil ||
			finding.ArtifactPath != "semantic/model.tei" {
			t.Fatalf("missing exact declaration locator: %+v", finding)
		}
	}
	info, err := os.Stat(filepath.Join(out, "model-hypothesis-source-audit.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("source audit was not private: info=%v err=%v", info, err)
	}
}

func TestCLIRejectsForgedOrMissingWorkspaceSourceBeforeWriting(t *testing.T) {
	root, inputPath, proposalsPath := modelSourceCLIInputs(t)
	output := t.TempDir()
	if err := run(inputPath, "", false, output, "", "", root); err == nil {
		t.Fatal("source auditing without hypotheses was accepted")
	}
	if err := run(inputPath, "../../../../reference/world.json", false, output, "", proposalsPath, root); err == nil {
		t.Fatal("source-audit allowed an unrelated World override")
	}
	if err := run(inputPath, "", false, output, "", proposalsPath, filepath.Join(root, "missing")); err == nil {
		t.Fatal("nonexistent Workspace was silently accepted")
	}
	content, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	var input example.Input
	if err := json.Unmarshal(content, &input); err != nil {
		t.Fatal(err)
	}
	// Retain the copied revision: source audit must compare the whole
	// Catalog from the loaded Workspace, not trust a matching revision.
	table := input.Catalog.Tables["sales.child"]
	table.Fields[0].Required = true
	input.Catalog.Tables["sales.child"] = table
	forged, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, forged, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, "", false, output, "", proposalsPath, root); err == nil {
		t.Fatal("forged Catalog was accepted for source audit")
	}
	if entries, err := os.ReadDir(output); err != nil || len(entries) != 0 {
		t.Fatalf("audit error produced partial files: entries=%v err=%v", entries, err)
	}
}

func TestCLIExportsRuleDeclarationDraftWithoutAcceptingBusinessRule(t *testing.T) {
	root, inputPath, hypothesesPath := modelSourceCLIInputs(t)
	var input example.Input
	if err := readStrictJSON(inputPath, &input); err != nil {
		t.Fatal(err)
	}
	var hypotheses example.ModelHypotheses
	if err := readStrictJSON(hypothesesPath, &hypotheses); err != nil {
		t.Fatal(err)
	}
	review, err := example.ReviewModelHypotheses(input, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := example.AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, hypotheses)
	if err != nil {
		t.Fatal(err)
	}
	intent := example.ModelRuleDraftIntent{
		Schema:                 "tei.reference.model-rule-draft-intent/v0",
		ModelReviewFingerprint: review.Fingerprint,
		SourceAuditFingerprint: audit.Fingerprint,
		Requests: []example.ModelRuleDraftRequest{{
			IssueID:        review.Issues[0].ID,
			ProposedRuleID: "sales.child.business-key",
			Rationale:      "業務識別規則として検討するが根拠の真偽は未確認",
		}},
	}
	path := filepath.Join(t.TempDir(), "rule-intents.json")
	raw, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := run(inputPath, "", false, out, "", hypothesesPath, root, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "model-rule-declaration-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result example.ModelRuleDeclarationDraft
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.ModelReviewFingerprint != review.Fingerprint ||
		result.SourceAuditFingerprint != audit.Fingerprint ||
		len(result.Declarations) != 1 ||
		result.Declarations[0].ProposedRuleID != "sales.child.business-key" ||
		result.Declarations[0].Status != "unverified-business-rule" {
		t.Fatalf("unverified rule declaration lost evidence binding: %+v", result)
	}
	info, err := os.Stat(filepath.Join(out, "model-rule-declaration-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("draft file permissions = %v, want 0600", info.Mode().Perm())
	}
	for _, filename := range []string{"model-hypothesis-review.json", "model-hypothesis-source-audit.json", "review.json"} {
		if _, err := os.Stat(filepath.Join(out, filename)); err != nil {
			t.Fatalf("lost previous review output %s: %v", filename, err)
		}
	}
}

func TestCLIRejectsUnboundRuleDraftBeforeGeneratingFiles(t *testing.T) {
	root, inputPath, hypothesesPath := modelSourceCLIInputs(t)
	intentPath := filepath.Join(t.TempDir(), "rules.json")
	out := t.TempDir()
	stale := `{"schema":"tei.reference.model-rule-draft-intent/v0","modelReviewFingerprint":"sha256:stale","sourceAuditFingerprint":"sha256:stale","requests":[]}`
	if err := os.WriteFile(intentPath, []byte(stale), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, "", false, out, "", hypothesesPath, root, intentPath); err == nil {
		t.Fatal("stale or invalid rule review was accepted")
	}
	if entries, err := os.ReadDir(out); err != nil || len(entries) != 0 {
		t.Fatalf("invalid rule created partial files: %v %v", entries, err)
	}
	if err := run(inputPath, "", false, out, "", hypothesesPath, "", intentPath); err == nil {
		t.Fatal("rule draft without current Workspace source audit was accepted")
	}
	if err := run(inputPath, "", false, out, "", "", root, intentPath); err == nil {
		t.Fatal("rule draft without explicit hypothesis was accepted")
	}
	if err := os.WriteFile(intentPath, []byte(`{"schema":"tei.reference.model-rule-draft-intent/v0","unexpected":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(inputPath, "", false, out, "", hypothesesPath, root, intentPath); err == nil {
		t.Fatal("unknown rule intent field passed strict JSON parsing")
	}
	if entries, err := os.ReadDir(out); err != nil || len(entries) != 0 {
		t.Fatalf("invalid rule wrote outputs: %v %v", entries, err)
	}
}
