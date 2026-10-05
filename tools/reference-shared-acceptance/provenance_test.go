package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOriginDoesNotChangeTheDecisionOrTheAcceptedSource(t *testing.T) {
	origins := []Origin{
		{Kind: "file"},
		{Kind: "cli", Actor: "developer"},
		{Kind: "studio", Actor: "studio-session-7"},
		{Kind: "ai", Actor: "assistant-x", Notes: map[string]string{
			"model": "model-x-2026", "promptDigest": "sha256:feedface", "tokens": "1234",
		}},
	}
	type outcome struct {
		fingerprint string
		after       string
		derivedSHA  string
		inserted    string
		tree        map[string]string
	}
	var outcomes []outcome
	for _, origin := range origins {
		dir := scratchWorkspace(t)
		c := unitPriceCandidate(revisionOfDir(t, dir))
		c.ID = "cand-" + origin.Kind
		c.Origin = origin
		derivedPath := filepath.Join(t.TempDir(), "world.json")

		decision, err := accept(dir, mustJSON(t, c), acceptOptions{DerivedOut: derivedPath})
		if err != nil || decision.Decision != "accepted" {
			t.Fatalf("origin=%s decision=%+v err=%v", origin.Kind, decision, err)
		}
		derived, err := os.ReadFile(derivedPath)
		if err != nil {
			t.Fatal(err)
		}
		for _, private := range []string{"model-x-2026", "sha256:feedface", "studio-session-7", "assistant-x", "developer"} {
			if bytes.Contains(derived, []byte(private)) {
				t.Fatalf("origin固有のEvidence %q がCanonical Worldへ入っています", private)
			}
			for path, content := range readTree(t, dir) {
				if strings.Contains(content, private) {
					t.Fatalf("origin固有のEvidence %q がauthoritative source %s へ入っています", private, path)
				}
			}
		}
		outcomes = append(outcomes, outcome{
			fingerprint: decision.CandidateFingerprint,
			after:       decision.ModelRevisionAfter,
			derivedSHA:  decision.DerivedWorldSHA256,
			inserted:    decision.SourceEdits[0].InsertedText,
			tree:        treeDigest(t, dir),
		})
	}
	for i := 1; i < len(outcomes); i++ {
		if !reflect.DeepEqual(outcomes[0], outcomes[i]) {
			t.Fatalf("入口が%sと%sで、受入結果が異なります:\n%+v\n%+v", origins[0].Kind, origins[i].Kind, outcomes[0], outcomes[i])
		}
	}
}

func TestFingerprintIgnoresProvenanceAndOrderButNotMeaning(t *testing.T) {
	revision := "sha256:" + strings.Repeat("a", 64)
	fingerprint := func(mutate func(c *Candidate)) string {
		c := unitPriceCandidate(revision)
		c.Delta = append(c.Delta, Operation{Op: opAddValidation, Namespace: "sales", ID: "sales.order.total_positive", Kind: "positive", Attribute: "sales.order.total"})
		mutate(&c)
		_, fp, err := normalizeCandidate(c)
		if err != nil {
			t.Fatal(err)
		}
		return fp
	}
	base := fingerprint(func(c *Candidate) {})
	same := map[string]func(c *Candidate){
		"delta order": func(c *Candidate) { c.Delta[0], c.Delta[1] = c.Delta[1], c.Delta[0] },
		"origin":      func(c *Candidate) { c.Origin = Origin{Kind: "ai", Actor: "x", Notes: map[string]string{"k": "v"}} },
		"intent":      func(c *Candidate) { c.Intent = "別の説明文" },
		"evidence":    func(c *Candidate) { c.Evidence = nil },
		"candidate id": func(c *Candidate) {
			c.ID = "another-id"
		},
	}
	for name, mutate := range same {
		if got := fingerprint(mutate); got != base {
			t.Fatalf("%sの違いでfingerprintが変わりました", name)
		}
	}
	differs := map[string]func(c *Candidate){
		"attribute":  func(c *Candidate) { c.Delta[0].Attribute = "sales.order_line.quantity" },
		"kind":       func(c *Candidate) { c.Delta[1].Kind = "relation-unique"; c.Delta[1].Relation = "sales.order.lines" },
		"base":       func(c *Candidate) { c.BaseModelRevision = "sha256:" + strings.Repeat("b", 64) },
		"validation": func(c *Candidate) { c.Delta[0].ID = "sales.order_line.unit_price_other" },
	}
	for name, mutate := range differs {
		if got := fingerprint(mutate); got == base {
			t.Fatalf("意味の違い(%s)がfingerprintへ現れていません", name)
		}
	}
}

func TestSemanticAndEvidenceFingerprintsRemainIndependent(t *testing.T) {
	revision := "sha256:" + strings.Repeat("a", 64)
	base := unitPriceCandidate(revision)
	normalizedBase, semanticBase, err := normalizeCandidate(base)
	if err != nil {
		t.Fatal(err)
	}
	evidenceBase := evidenceFingerprintOf(normalizedBase.Evidence)
	if evidenceBase == "" {
		t.Fatal("candidate with Evidence has no evidenceFingerprint")
	}

	changedEvidence := unitPriceCandidate(revision)
	changedEvidence.Evidence = append([]Evidence(nil), changedEvidence.Evidence...)
	changedEvidence.Evidence[0].Note = changedEvidence.Evidence[0].Note + "（別の説明）"
	normalizedChanged, semanticChanged, err := normalizeCandidate(changedEvidence)
	if err != nil {
		t.Fatal(err)
	}
	evidenceChanged := evidenceFingerprintOf(normalizedChanged.Evidence)

	if semanticChanged != semanticBase {
		t.Fatalf("Evidence change altered semantic fingerprint: before=%s after=%s", semanticBase, semanticChanged)
	}
	if evidenceChanged == evidenceBase {
		t.Fatalf("Evidence change was not reflected in evidenceFingerprint: %s", evidenceBase)
	}
}

func TestPreviewAndAcceptedDecisionBindSameEvidenceFingerprint(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))

	ev, err := evaluate(dir, mustJSON(t, candidate))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("preview is not acceptable: %+v", ev.preview.Stages)
	}
	if ev.preview.EvidenceFingerprint == "" {
		t.Fatal("preview has no evidenceFingerprint")
	}

	decision, err := accept(dir, mustJSON(t, candidate), acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "accepted" {
		t.Fatalf("decision=%+v", decision)
	}
	if decision.CandidateFingerprint != ev.preview.CandidateFingerprint {
		t.Fatalf("semantic fingerprint drifted: preview=%s decision=%s", ev.preview.CandidateFingerprint, decision.CandidateFingerprint)
	}
	if decision.EvidenceFingerprint != ev.preview.EvidenceFingerprint {
		t.Fatalf("Evidence fingerprint drifted: preview=%s decision=%s", ev.preview.EvidenceFingerprint, decision.EvidenceFingerprint)
	}
}

func TestRelationUniqueGoesThroughTheSameBoundary(t *testing.T) {
	dir := scratchWorkspace(t)
	c := unitPriceCandidate(revisionOfDir(t, dir))
	c.Delta = []Operation{{
		Op: opAddValidation, Namespace: "sales", ID: "sales.order.lines.item_code_unique",
		Kind: "relation-unique", Relation: "sales.order.lines", Attribute: "sales.order_line.item_code",
	}}
	c.Evidence = []Evidence{{Subject: "sales.order.lines.line_number_unique", Note: "同じrelation-unique形式の既存rule"}}

	decision, err := accept(dir, mustJSON(t, c), acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	wantText := "\n\nvalidation sales.order.lines.item_code_unique relation-unique {\n    relation sales.order.lines\n    attribute sales.order_line.item_code\n}\n"
	if decision.SourceEdits[0].InsertedText != wantText {
		t.Fatalf("追記されたsource=%q", decision.SourceEdits[0].InsertedText)
	}
}

func TestDerivedArtifactsAreDisposableAndRegenerableFromTheAcceptedSource(t *testing.T) {
	dir := scratchWorkspace(t)
	derivedPath := filepath.Join(t.TempDir(), "derived", "world.json")
	decision, err := accept(dir, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))), acceptOptions{DerivedOut: derivedPath})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	first, err := os.ReadFile(derivedPath)
	if err != nil {
		t.Fatal(err)
	}

	// Decision recordも派生物も消しても、accepted sourceから同じ派生物を再生成できる。
	if err := os.RemoveAll(filepath.Dir(derivedPath)); err != nil {
		t.Fatal(err)
	}
	treeBefore := treeDigest(t, dir)
	s, err := takeSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	sha, regenerated, err := worldSHA256(s.loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, regenerated) || sha != decision.DerivedWorldSHA256 {
		t.Fatal("派生物を消去した後、accepted sourceから同じCanonical Worldを再生成できません")
	}
	assertTreeUnchanged(t, treeBefore, treeDigest(t, dir))

	// 別の場所へ複製したaccepted sourceからも同じ結果になる（pathやtoolの状態に依存しない）。
	copyDir := filepath.Join(t.TempDir(), "elsewhere")
	if err := copyTree(dir, copyDir); err != nil {
		t.Fatal(err)
	}
	s2, err := takeSnapshot(copyDir)
	if err != nil {
		t.Fatal(err)
	}
	sha2, _, err := worldSHA256(s2.loaded.World)
	if err != nil || sha2 != sha || s2.revision != s.revision {
		t.Fatalf("複製したsourceの再生成結果が違います: %s vs %s", sha2, sha)
	}
}

func TestCLIRunsPreviewRejectAndAcceptAndGuardsTheReferenceBaseline(t *testing.T) {
	run := func(args ...string) (int, string, string) {
		var stdout, stderr bytes.Buffer
		code := runCLI(args, &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}

	dir := scratchWorkspace(t)
	revision := revisionOfDir(t, dir)
	candidatePath := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(revision)), 0o644); err != nil {
		t.Fatal(err)
	}
	before := treeDigest(t, dir)

	code, out, _ := run("revision", "-workspace", dir)
	var rev map[string]string
	if code != 0 || json.Unmarshal([]byte(out), &rev) != nil || rev["modelRevision"] != revision {
		t.Fatalf("revision code=%d out=%s", code, out)
	}

	code, out, _ = run("preview", "-workspace", dir, "-candidate", candidatePath)
	var preview Preview
	if code != 0 || json.Unmarshal([]byte(out), &preview) != nil || !preview.Acceptable {
		t.Fatalf("preview code=%d out=%s", code, out)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	stalePath := filepath.Join(t.TempDir(), "stale.json")
	if err := os.WriteFile(stalePath, mustJSON(t, unitPriceCandidate("sha256:"+strings.Repeat("0", 64))), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = run("preview", "-workspace", dir, "-candidate", stalePath)
	if code != 1 || !strings.Contains(out, "STALE_BASE_MODEL_REVISION") {
		t.Fatalf("staleなpreviewは失敗終了し理由を示す必要があります: code=%d out=%s", code, out)
	}

	code, out, _ = run("reject", "-workspace", dir, "-candidate", candidatePath, "-reason", "見送り")
	if code != 0 || !strings.Contains(out, `"decision": "rejected"`) || !strings.Contains(out, `"recordPath": ".tei/decisions/`) {
		t.Fatalf("reject code=%d out=%s", code, out)
	}
	if got := revisionOfDir(t, dir); got != revision {
		t.Fatalf("reject Decision recordがmodel revisionを変えました: got=%s want=%s", got, revision)
	}

	// Gate B / Gate Cが固定したbaselineの名前を持つ場所へは、明示しない限りacceptしない。
	baselineLike := filepath.Join(t.TempDir(), "reference", "workspace")
	if err := copyTree(dir, baselineLike); err != nil {
		t.Fatal(err)
	}
	baselineBefore := treeDigest(t, baselineLike)
	baselineDecisionCount := len(decisionsOf(t, baselineLike).Records)
	code, out, errOut = run("reject", "-workspace", baselineLike, "-candidate", candidatePath, "-reason", "baselineでは記録しない")
	if code != 0 || !strings.Contains(out, `"decision": "rejected"`) || !strings.Contains(errOut, "決定記録を書きませんでした") {
		t.Fatalf("baselineのreject record guardが働いていません: code=%d out=%s err=%s", code, out, errOut)
	}
	if got := len(decisionsOf(t, baselineLike).Records); got != baselineDecisionCount {
		t.Fatalf("baselineのDecision recordが明示許可なしで増えました: got=%d want=%d", got, baselineDecisionCount)
	}
	assertTreeUnchanged(t, baselineBefore, treeDigest(t, baselineLike))

	code, _, errOut := run("accept", "-workspace", baselineLike, "-candidate", candidatePath)
	if code != 2 || !strings.Contains(errOut, "baseline") {
		t.Fatalf("baselineへのacceptが止まっていません: code=%d err=%s", code, errOut)
	}
	assertTreeUnchanged(t, baselineBefore, treeDigest(t, baselineLike))

	// 外部CLIからのsource mutationは、Review Projectionを一度も固定せずに実行できない。
	normalBefore := treeDigest(t, dir)
	code, _, errOut = run("accept", "-workspace", dir, "-candidate", candidatePath)
	if code != 2 || !strings.Contains(errOut, "expected-review-fingerprint") {
		t.Fatalf("review bindingなしのacceptが止まっていません: code=%d err=%s", code, errOut)
	}
	assertTreeUnchanged(t, normalBefore, treeDigest(t, dir))

	derived := filepath.Join(t.TempDir(), "world.json")
	reviewFingerprint := reviewFingerprintViaCLI(t, dir, candidatePath)
	code, out, _ = run(
		"accept",
		"-workspace", dir,
		"-candidate", candidatePath,
		"-expected-review-fingerprint", reviewFingerprint,
		"-derived-out", derived,
	)
	if code != 0 || !strings.Contains(out, `"decision": "accepted"`) {
		t.Fatalf("accept code=%d out=%s", code, out)
	}
	if _, err := os.Stat(derived); err != nil {
		t.Fatalf("派生物が出力されていません: %v", err)
	}
	baselineReviewFingerprint := reviewFingerprintViaCLI(t, baselineLike, candidatePath)
	code, _, _ = run(
		"accept",
		"-workspace", baselineLike,
		"-candidate", candidatePath,
		"-expected-review-fingerprint", baselineReviewFingerprint,
		"-allow-reference-baseline",
	)
	if code != 0 {
		t.Fatalf("明示的なbaseline許可とreview bindingがあればscratchへacceptできる必要があります: code=%d", code)
	}

	for _, args := range [][]string{{}, {"bogus"}, {"preview"}, {"preview", "-workspace", dir}} {
		if code, _, _ := run(args...); code != 2 {
			t.Fatalf("usage error %v の終了コード=%d", args, code)
		}
	}
}

func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for path := range treeDigest(t, dir) {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		out[path] = string(content)
	}
	return out
}
