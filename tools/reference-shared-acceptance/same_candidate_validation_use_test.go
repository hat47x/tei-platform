package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/workspace"
)

// 受理済みのorder_entry規則（sales.order_line.line_number_positive）を取り除いたcopyを返す。
// 本testは、その規則を同じcandidateで足す場面を確かめるため、規則が正本に入る前の状態から始める。
// 取り除くのは受理時に追加された宣言だけで、戻した結果のsha256が受理前のfingerprintと一致することを確かめる。
const (
	practicalBValidationSource = "semantic/sales/validation.tei"
	practicalBValidationBefore = "sha256:bd641910f993ef84cbab1ccf3616de8d98c715e7679268115a71a0327e126633"
	practicalBValidationInsert = "\nvalidation sales.order_line.line_number_positive positive {\n    attribute sales.order_line.line_number\n}\n"
)

func scratchWorkspaceBeforePracticalB(t *testing.T) string {
	t.Helper()
	dir := scratchWorkspaceBeforeSelection(t)
	path := filepath.Join(dir, filepath.FromSlash(practicalBValidationSource))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, practicalBValidationInsert) != 1 {
		t.Fatalf("%s に受理済みの規則の宣言が一つだけ見つかりません", practicalBValidationSource)
	}
	restored := strings.Replace(content, practicalBValidationInsert, "", 1)
	sum := sha256.Sum256([]byte(restored))
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != practicalBValidationBefore {
		t.Fatalf("%s を戻した結果が受理前のfingerprintと一致しません: got=%s", practicalBValidationSource, got)
	}
	if err := os.WriteFile(path, []byte(restored), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// ADR-0029の確かめ方1は、正本へValidationを一件足し、Applicationへ宣言を一行足すだけで、
// その規則が評価されることを求める。両方を同じcandidateに入れたとき、参照先はまだ基準のWorldに無いため、
// resolveが誤って拒否しないことを確かめる。参照先のIDの完全一致はresolveが確かめ、overlay適用後のvalidateは
// 別の確認として残る。このtestはvalidateが参照を拒否する経路は通さない（その分岐はinternal/validationのtestが単独で確かめる）。
// 参照先がどのcandidateにも無い場合は従来どおりresolveが拒否することも同時に確かめる。
func TestApplicationValidationUseMayReferenceAValidationAddedByTheSameCandidate(t *testing.T) {
	dir := scratchWorkspaceBeforePracticalB(t)
	addRule := Operation{
		Op:        opAddValidation,
		Namespace: "sales",
		ID:        "sales.order_line.line_number_positive",
		Kind:      "positive",
		Attribute: "sales.order_line.line_number",
		Placement: "semantic/sales/validation.tei",
	}
	c := validationUseCandidate(
		revisionOfDir(t, dir),
		addRule,
		validationUseOp("order_entry", "sales.order_line.line_number_positive"),
	)

	ev, err := evaluate(dir, mustJSON(t, c))
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("同じcandidateのValidation追加と宣言を拒否しました: %v", allDiagnostics(p))
	}
	if got := keysOf(p.SemanticDiff); len(got) != 1 || got[0] != "added validation:sales.order_line.line_number_positive" {
		t.Fatalf("semantic差分=%v", got)
	}
	if got := keysOf(p.ApplicationDiff); len(got) != 1 || got[0] != "changed application:order_entry" {
		t.Fatalf("application差分=%v", got)
	}

	decision, err := accept(dir, mustJSON(t, c), acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := driver.NewResolver(loaded.World).ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].ID != "sales.order_line.line_number_positive" {
		t.Fatalf("受理後にDriverが解決した規則=%+v", rules)
	}

	// 参照先が、基準のWorldにもこのcandidateにも無いなら、従来どおり拒否する。
	dir = scratchWorkspaceBeforePracticalB(t)
	missing := validationUseCandidate(
		revisionOfDir(t, dir),
		addRule,
		validationUseOp("order_entry", "sales.order_line.missing_rule"),
	)
	ev, err = evaluate(dir, mustJSON(t, missing))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "VALIDATION_NOT_FOUND") {
		t.Fatalf("どこにも無い規則の宣言を拒否していません: acceptable=%v codes=%v", ev.preview.Acceptable, diagnosticCodes(ev.preview))
	}
}
