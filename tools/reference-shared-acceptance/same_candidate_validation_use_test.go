package main

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/workspace"
)

// ADR-0029の確かめ方1は、正本へValidationを一件足し、Applicationへ宣言を一行足すだけで、
// その規則が評価されることを求める。両方を同じcandidateに入れたとき、参照先はまだ基準のWorldに無いため、
// resolveが誤って拒否しないことを確かめる。参照先のIDの完全一致はresolveが確かめ、overlay適用後のvalidateは
// 別の確認として残る。このtestはvalidateが参照を拒否する経路は通さない（その分岐はinternal/validationのtestが単独で確かめる）。
// 参照先がどのcandidateにも無い場合は従来どおりresolveが拒否することも同時に確かめる。
func TestApplicationValidationUseMayReferenceAValidationAddedByTheSameCandidate(t *testing.T) {
	dir := scratchWorkspaceBeforeSelection(t)
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
	dir = scratchWorkspaceBeforeSelection(t)
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
