package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
	"tei/internal/subject"
)

func cliRun(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runCLI(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func candidateFile(t *testing.T, c Candidate) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "candidate.json")
	writeFile(t, path, string(mustJSON(t, c)))
	return path
}

func decisionsOf(t *testing.T, dir string) DecisionIndex {
	t.Helper()
	code, out, errOut := cliRun(t, "decisions", "-workspace", dir)
	if code != 0 {
		t.Fatalf("decisions code=%d err=%s", code, errOut)
	}
	var index DecisionIndex
	if err := json.Unmarshal([]byte(out), &index); err != nil {
		t.Fatalf("decisions出力を読めません: %v: %s", err, out)
	}
	return index
}

func acceptViaCLI(t *testing.T, dir, candidatePath string) Decision {
	t.Helper()
	code, out, errOut := cliRun(t, "accept", "-workspace", dir, "-candidate", candidatePath)
	if code != 0 {
		t.Fatalf("accept code=%d err=%s out=%s", code, errOut, out)
	}
	var decision Decision
	if err := json.Unmarshal([]byte(out), &decision); err != nil {
		t.Fatal(err)
	}
	return decision
}

// RQ-027は、後から入るAIが会話履歴に頼らず「accepted Decision」へ機械的に到達できることを求める。
// 受理はauthoritative sourceを書くが、なぜそのsourceがあるのかはこれまでworkspaceに残らなかった。
func TestAcceptedDecisionIsReachableFromTheWorkspaceWithoutTheConversation(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidatePath := candidateFile(t, candidate)

	// 何も受理していないworkspaceでは、記録が無いことがエラーではなく空として返る。
	if index := decisionsOf(t, dir); len(index.Records) != 0 {
		t.Fatalf("受理前に記録があります: %+v", index.Records)
	}

	// rejectはworkspaceを変えないので、記録も残さない。
	if code, _, errOut := cliRun(t, "reject", "-workspace", dir, "-candidate", candidatePath, "-reason", "見送り"); code != 0 {
		t.Fatalf("reject code=%d err=%s", code, errOut)
	}
	if index := decisionsOf(t, dir); len(index.Records) != 0 {
		t.Fatalf("rejectが記録を残しています: %+v", index.Records)
	}

	decision := acceptViaCLI(t, dir, candidatePath)
	if !strings.HasPrefix(decision.RecordPath, decisionDir+"/") {
		t.Fatalf("受理結果が記録先を示していません: %q", decision.RecordPath)
	}

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("記録が1件ではありません: %+v", index.Records)
	}
	record := index.Records[0]
	if record.Path != decision.RecordPath {
		t.Fatalf("record path=%q accept=%q", record.Path, decision.RecordPath)
	}
	if record.Status != "current" || record.SourceEdits != "intact" || record.EvidenceStatus != "current" {
		t.Fatalf("status=%q sourceEdits=%q evidenceStatus=%q", record.Status, record.SourceEdits, record.EvidenceStatus)
	}
	if record.ReviewBindingStatus != "none" ||
		record.ReviewCoverageStatus != "none" ||
		len(record.ReviewIncomplete) != 0 ||
		len(record.ReviewDiagnostics) != 0 {
		t.Fatalf(
			"review binding status=%q coverage=%q incomplete=%+v diagnostics=%+v",
			record.ReviewBindingStatus,
			record.ReviewCoverageStatus,
			record.ReviewIncomplete,
			record.ReviewDiagnostics,
		)
	}
	if index.ModelRevision != record.Decision.ModelRevisionAfter {
		t.Fatalf("index revision=%q record=%q", index.ModelRevision, record.Decision.ModelRevisionAfter)
	}

	// 記録は「何が変わったか」だけでなく「なぜか」を持つ。fingerprintへ入らないIntent / Evidenceが、
	// 会話履歴を持たない読み手にとって唯一の手掛かりになる。
	got := record.Decision
	if got.Intent != candidate.Intent {
		t.Fatalf("intent=%q want=%q", got.Intent, candidate.Intent)
	}
	if len(got.Evidence) != len(candidate.Evidence) {
		t.Fatalf("evidence=%+v want=%+v", got.Evidence, candidate.Evidence)
	}
	if got.Decision != "accepted" || !got.SourceChanged {
		t.Fatalf("decision=%q sourceChanged=%v", got.Decision, got.SourceChanged)
	}
	if got.CandidateFingerprint == "" || got.BaseModelRevision == "" || got.DerivedWorldSHA256 == "" {
		t.Fatalf("同一性を追えません: %+v", got)
	}
	if len(got.SourceEdits) == 0 || len(got.SemanticDiff) == 0 || len(got.Conformance) == 0 {
		t.Fatalf("変更内容を追えません: %+v", got)
	}
	if got.Origin == nil || got.Origin.Kind != candidate.Origin.Kind {
		t.Fatalf("origin=%+v", got.Origin)
	}
}

func TestDecisionRecordReportsQualificationEvidenceFreshnessSeparately(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	snapshot, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}

	ctx := knowledgecontext.Build(
		snapshot.loaded.World,
		snapshot.index,
		subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
	)
	qualification, err := knowledgecontext.QualifyWithSources(ctx, snapshot.index, knowledgecontext.QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []knowledgecontext.DependencyRequirement{{
			Direction: "outgoing",
			Relation:  "contract-query",
			Kind:      "database-contract-query",
			ID:        "master_data.customer/find_by_id",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if qualification.Status != knowledgecontext.QualificationApplicable || len(qualification.DependencyEvidence) == 0 {
		t.Fatalf("qualification=%+v", qualification)
	}

	grounded, err := attachQualificationEvidence(
		mustJSON(t, unitPriceCandidate(snapshot.revision)),
		qualification,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision := acceptViaCLI(t, dir, candidateFile(t, grounded))

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("records=%+v", index.Records)
	}
	record := index.Records[0]
	if record.EvidenceStatus != "current" || len(record.EvidenceDiagnostics) != 0 {
		t.Fatalf("accepted Evidence is not current: %+v", record)
	}

	dependencySource := qualification.DependencyEvidence[0].Source
	path := filepath.Join(dir, filepath.FromSlash(dependencySource))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	index = decisionsOf(t, dir)
	record = index.Records[0]
	if record.Status != "superseded" {
		t.Fatalf("workspace revision change was not reported: %+v", record)
	}
	if record.SourceEdits != "intact" {
		t.Fatalf("decision-written source was incorrectly marked diverged: %+v", record)
	}
	if record.EvidenceStatus != "diverged" {
		t.Fatalf("qualification Evidence drift was not reported: %+v", record)
	}
	var codes []string
	for _, diagnostic := range record.EvidenceDiagnostics {
		codes = append(codes, diagnostic.Code)
	}
	if !hasCode(codes, "EVIDENCE_QUALIFICATION_DEPENDENCY_STALE") {
		t.Fatalf("dependency Evidence drift reason=%v", codes)
	}
	if record.Decision.RecordPath != decision.RecordPath {
		t.Fatalf("record path changed: %q != %q", record.Decision.RecordPath, decision.RecordPath)
	}
}

func TestDecisionRecordDetectsEvidenceBodyTampering(t *testing.T) {
	dir := scratchWorkspace(t)
	decision := acceptViaCLI(t, dir, candidateFile(t, unitPriceCandidate(revisionOfDir(t, dir))))
	if decision.EvidenceFingerprint == "" || len(decision.Evidence) == 0 {
		t.Fatalf("accepted decision lacks bound Evidence identity: %+v", decision)
	}

	// 元のcontent-addressed recordを外し、Evidence本文だけを書き換えた別recordを
	// recordDecisionで正しいcontent addressへ保存する。file名の改ざん検査を通過した後も、
	// Evidence本文とevidenceFingerprintの不一致を検出できる必要がある。
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(decision.RecordPath))); err != nil {
		t.Fatal(err)
	}
	tampered := decision
	tampered.RecordPath = ""
	tampered.Evidence = append([]Evidence(nil), decision.Evidence...)
	tampered.Evidence[0].Note = tampered.Evidence[0].Note + " tampered"
	if _, err := recordDecision(dir, tampered); err != nil {
		t.Fatal(err)
	}

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("records=%+v", index.Records)
	}
	record := index.Records[0]
	if record.EvidenceStatus != "diverged" {
		t.Fatalf("tampered Evidence body was not reported: %+v", record)
	}
	var codes []string
	for _, diagnostic := range record.EvidenceDiagnostics {
		codes = append(codes, diagnostic.Code)
	}
	if !hasCode(codes, "EVIDENCE_FINGERPRINT_MISMATCH") {
		t.Fatalf("fingerprint mismatch diagnostic is absent: %v", codes)
	}
}

func TestDecisionRecordContentAddressRejectsInPlaceTampering(t *testing.T) {
	dir := scratchWorkspace(t)
	decision := acceptViaCLI(t, dir, candidateFile(t, unitPriceCandidate(revisionOfDir(t, dir))))
	path := filepath.Join(dir, filepath.FromSlash(decision.RecordPath))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var tampered Decision
	if err := json.Unmarshal(content, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered.Intent += " tampered"
	encoded, err := encodeJSON(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := loadDecisionRecords(dir); err == nil {
		t.Fatal("content-addressed decision recordのin-place改ざんを拒否していません")
	} else if !strings.Contains(err.Error(), "content hashがfile名と一致しません") {
		t.Fatalf("unexpected error: %v", err)
	}
}


// 記録はauthorityではない。消しても、偽物を置いても、accepted meaningとmodel revisionは動かない。
func TestReviewBindingClassifierRejectsInvalidFingerprint(t *testing.T) {
	binding := &ReviewBinding{
		Fingerprint: "sha256:" + strings.Repeat("g", 64),
		Coverage: ReviewCoverageSummary{
			Available: 8,
		},
	}
	bindingStatus, coverageStatus, diagnostics := classifyReviewBinding(binding)
	if bindingStatus != "invalid" || coverageStatus != "invalid" {
		t.Fatalf("binding=%q coverage=%q diagnostics=%+v", bindingStatus, coverageStatus, diagnostics)
	}
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "REVIEW_BINDING_FINGERPRINT_INVALID" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestDecisionRecordRejectsMalformedRecordedReviewCoverage(t *testing.T) {
	dir := scratchWorkspace(t)
	data := mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir)))
	decision, err := accept(dir, data, acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	decision.ReviewBinding = &ReviewBinding{
		Fingerprint: "sha256:recorded-but-not-verifiable-here",
		Coverage: ReviewCoverageSummary{
			Available:    3,
			Partial:      5,
			NotEvaluated: 0,
			Incomplete: []ReviewCoverageGap{
				{Code: "A2", Title: "影響", Coverage: reviewCoveragePartial},
				{Code: "A3", Title: "例外・診断", Coverage: reviewCoveragePartial},
				{Code: "A5", Title: "権限の変更", Coverage: reviewCoveragePartial},
				{Code: "A6", Title: "native codeへ残る箇所", Coverage: reviewCoverageNotEvaluated},
				{Code: "A7", Title: "逆追跡", Coverage: reviewCoveragePartial},
			},
		},
	}
	if _, err := recordDecision(dir, decision); err != nil {
		t.Fatal(err)
	}

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("records=%+v", index.Records)
	}
	record := index.Records[0]
	if record.ReviewBindingStatus != "invalid" || record.ReviewCoverageStatus != "invalid" {
		t.Fatalf("review status=%q coverage=%q", record.ReviewBindingStatus, record.ReviewCoverageStatus)
	}
	if len(record.ReviewDiagnostics) == 0 {
		t.Fatal("malformed review coverage produced no diagnostics")
	}
	found := false
	for _, diagnostic := range record.ReviewDiagnostics {
		if diagnostic.Code == "REVIEW_COVERAGE_INCOMPLETE_MISMATCH" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("review diagnostics=%+v", record.ReviewDiagnostics)
	}
}

func TestDecisionRecordIsNotAuthorityOverMeaning(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	acceptViaCLI(t, dir, candidateFile(t, unitPriceCandidate(revisionOfDir(t, dir))))

	accepted, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	acceptedSHA, _, err := worldSHA256(accepted.loaded.World)
	if err != nil {
		t.Fatal(err)
	}

	assertMeaningUnchanged := func(what string) {
		t.Helper()
		s, err := takeSnapshot(dir)
		if err != nil {
			t.Fatal(err)
		}
		sha, _, err := worldSHA256(s.loaded.World)
		if err != nil {
			t.Fatal(err)
		}
		if s.revision != accepted.revision || sha != acceptedSHA {
			t.Fatalf("%sでmeaningまたはmodel revisionが変わりました: revision=%s world=%s", what, s.revision, sha)
		}
	}

	if err := os.RemoveAll(filepath.Join(dir, filepath.FromSlash(decisionDir))); err != nil {
		t.Fatal(err)
	}
	assertMeaningUnchanged("記録を全消去した後")
	if index := decisionsOf(t, dir); len(index.Records) != 0 {
		t.Fatalf("消去後も記録が見えます: %+v", index.Records)
	}

	// 実際には起きていない受理を主張する記録を置く。
	zero := "sha256:" + strings.Repeat("0", 64)
	forged := Decision{
		Schema:               decisionSchema,
		Decision:             "accepted",
		Intent:               "実在しない受理",
		CandidateFingerprint: zero,
		BaseModelRevision:    zero,
		ModelRevisionAfter:   zero,
		SourceChanged:        true,
		SourceEdits: []SourceEdit{{
			Path:             "semantic/sales/validation.tei",
			AfterFingerprint: zero,
		}},
	}
	if _, err := recordDecision(dir, forged); err != nil {
		t.Fatal(err)
	}
	assertMeaningUnchanged("偽の記録を置いた後")

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("記録が1件ではありません: %+v", index.Records)
	}
	// 偽物は読めるが、現在のworkspaceと照合した結果として食い違いが出る。黙って信用しない。
	if index.Records[0].Status != "superseded" || index.Records[0].SourceEdits != "diverged" {
		t.Fatalf("偽の記録が現行として扱われています: %+v", index.Records[0])
	}
	if len(index.Records[0].Diverged) != 1 || index.Records[0].Diverged[0] != "semantic/sales/validation.tei" {
		t.Fatalf("食い違ったfileを示していません: %+v", index.Records[0].Diverged)
	}
}

func TestDecisionRecordBecomesSupersededWhenTheWorkspaceMovesOn(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	first := acceptViaCLI(t, dir, candidateFile(t, unitPriceCandidate(revisionOfDir(t, dir))))
	second := acceptViaCLI(t, dir, candidateFile(t, secondApplicationCandidate(revisionOfDir(t, dir))))

	index := decisionsOf(t, dir)
	if len(index.Records) != 2 {
		t.Fatalf("記録が2件ではありません: %+v", index.Records)
	}
	status := map[string]DecisionRecord{}
	for _, r := range index.Records {
		status[r.Path] = r
	}
	if got := status[first.RecordPath]; got.Status != "superseded" {
		t.Fatalf("先の受理がsupersededになっていません: %+v", got)
	}
	if got := status[second.RecordPath]; got.Status != "current" {
		t.Fatalf("後の受理がcurrentになっていません: %+v", got)
	}
	// revisionが進んでも、先の受理が書いたfile自体は変わっていない。両者を混同しない。
	if got := status[first.RecordPath]; got.SourceEdits != "intact" {
		t.Fatalf("先の受理のsourceEdits=%q", got.SourceEdits)
	}
}

func TestDecisionRecordReportsDivergedSourceWhenTheFileItWroteChanges(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	decision := acceptViaCLI(t, dir, candidateFile(t, unitPriceCandidate(revisionOfDir(t, dir))))
	if len(decision.SourceEdits) == 0 {
		t.Fatal("受理がsourceを書いていません")
	}
	target := decision.SourceEdits[0].Path

	if index := decisionsOf(t, dir); index.Records[0].SourceEdits != "intact" {
		t.Fatalf("変更前からdivergedです: %+v", index.Records[0])
	}

	path := filepath.Join(dir, filepath.FromSlash(target))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, []byte("\n// 後から人が編集した\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	index := decisionsOf(t, dir)
	record := index.Records[0]
	if record.SourceEdits != "diverged" || len(record.Diverged) != 1 || record.Diverged[0] != target {
		t.Fatalf("書いたfileの変更を検出していません: %+v", record)
	}
}

func TestRecordingTheSameDecisionTwiceDoesNotAccumulate(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	decision, err := accept(dir, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))), acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	first, err := recordDecision(dir, decision)
	if err != nil {
		t.Fatal(err)
	}
	second, err := recordDecision(dir, decision)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("同じ決定が別のpathへ保存されました: %q %q", first, second)
	}
	if index := decisionsOf(t, dir); len(index.Records) != 1 {
		t.Fatalf("記録が増えています: %+v", index.Records)
	}
}

func TestUnreadableDecisionRecordFailsClosed(t *testing.T) {
	for name, content := range map[string]string{
		"未知のfield": `{"schema":"tei.reference.change-decision/v0","somethingElse":1}`,
		"壊れたJSON":  `{"schema":`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := scratchWorkspaceBeforeSecondApplication(t)
			acceptViaCLI(t, dir, candidateFile(t, unitPriceCandidate(revisionOfDir(t, dir))))
			writeFile(t, filepath.Join(dir, filepath.FromSlash(decisionDir), "broken.json"), content)

			if code, _, _ := cliRun(t, "decisions", "-workspace", dir); code == 0 {
				t.Fatal("読めない記録を黙って飛ばしています")
			}
		})
	}
}

// 独立レビュー（GPT-5.6 Sol、2026-09-21）のR6が指摘したとおり、派生物の書き出しだけが
// 原子的ではなかった。`reference/world.json`のようなgoldenを指された状態で書き込みが失敗すると、
// 既にあるfileを壊したままsourceだけ巻き戻る可能性があった。
func TestAcceptDoesNotDamageAnExistingDerivedArtifactWhenItCannotBeWritten(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	before := revisionOfDir(t, dir)

	outDir := t.TempDir()
	derived := filepath.Join(outDir, "world.json")
	const original = "{\"golden\": \"元の内容\"}\n"
	writeFile(t, derived, original)

	// directoryを書けなくして、派生物の書き出しだけを失敗させる。
	if err := os.Chmod(outDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outDir, 0o755) })

	decision, err := accept(dir, mustJSON(t, unitPriceCandidate(before)), acceptOptions{DerivedOut: derived})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" {
		t.Fatalf("派生物を書けないのに受理しています: %+v", decision)
	}
	var codes []string
	for _, r := range decision.Refusals {
		codes = append(codes, r.Code)
	}
	if !hasCode(codes, "DERIVED_WRITE_FAILED") {
		t.Fatalf("拒否理由=%v", codes)
	}

	content, err := os.ReadFile(derived)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != original {
		t.Fatalf("既にあった派生物が壊れています: %q", string(content))
	}
	if after := revisionOfDir(t, dir); after != before {
		t.Fatalf("sourceが巻き戻っていません: before=%s after=%s", before, after)
	}
	if index := decisionsOf(t, dir); len(index.Records) != 0 {
		t.Fatalf("受理していないのに記録があります: %+v", index.Records)
	}
}
