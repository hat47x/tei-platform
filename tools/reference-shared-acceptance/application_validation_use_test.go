package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/workspace"
)

// ADR-0029: an Application evaluates the Validations it declares with `uses validation`, in
// declaration order. `add-application-validation-use` adds one such declaration to an existing
// Application source through the same acceptance boundary as every other change.
const orderEntryApplicationSource = "applications/order-entry/application.tei"

// scratchWorkspaceBeforeSelectionは、order_entryが`uses validation`を宣言していない状態のcopyを返す。
// Referenceがその宣言を受理した後でも、「宣言を足す変更」を同じ前提で試せるようにする。
func scratchWorkspaceBeforeSelection(t *testing.T) string {
	t.Helper()
	dir := scratchWorkspace(t)
	path := filepath.Join(dir, filepath.FromSlash(orderEntryApplicationSource))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "uses validation ") {
			continue
		}
		kept = append(kept, line)
	}
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
func validationUseOp(application, validation string) Operation {
	return Operation{
		Op:        opAddApplicationValidationUse,
		Placement: orderEntryApplicationSource,
		ApplicationValidationUse: &ApplicationValidationUseSpec{
			Application: application,
			Validation:  validation,
		},
	}
}

func validationUseCandidate(revision string, ops ...Operation) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-order-entry-validation-selection",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "cli", Actor: "test"},
		Intent:            "order_entryが受注確定の前に評価するValidationを、Applicationの宣言として正本へ置く",
		Evidence: []Evidence{
			{Source: orderEntryApplicationSource, Note: "宣言を足す対象のApplication source"},
		},
		Delta: ops,
	}
}

func orderEntrySelection() []Operation {
	return []Operation{
		validationUseOp("order_entry", "sales.order_line.quantity_positive"),
		validationUseOp("order_entry", "sales.order.lines.line_number_unique"),
	}
}

func TestApplicationValidationUsePreviewReportsAChangedApplicationAndNothingElse(t *testing.T) {
	dir := scratchWorkspaceBeforeSelection(t)
	ev, err := evaluate(dir, mustJSON(t, validationUseCandidate(revisionOfDir(t, dir), orderEntrySelection()...)))
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("Previewが受理可能になりません: %v", allDiagnostics(p))
	}
	// 宣言の追加は、新しいsubjectではなく、既存Applicationの変更である。意味（semantic）は動かない。
	if got := keysOf(p.ApplicationDiff); len(got) != 1 || got[0] != "changed application:order_entry" {
		t.Fatalf("application差分=%v", got)
	}
	if len(p.SemanticDiff) != 0 || len(p.EnvironmentDiff) != 0 {
		t.Fatalf("意味・Environmentの差分が出ています: semantic=%v environment=%v", p.SemanticDiff, p.EnvironmentDiff)
	}
	if got := p.ChangedCollections; len(got) != 1 || got[0] != "applications" {
		t.Fatalf("changed collections=%v want [applications]", got)
	}

	// 承認者が「何が評価対象になったか」を影響から読める。宣言したValidationごとに辺がある。
	edges := map[string]bool{}
	for _, e := range p.Impact {
		if e.Reason == "" {
			t.Fatalf("辺に導出理由がありません: %+v", e)
		}
		edges[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	for _, want := range []string{
		"application:order_entry -> validation:sales.order_line.quantity_positive via application.uses-validation",
		"application:order_entry -> validation:sales.order.lines.line_number_unique via application.uses-validation",
	} {
		if !edges[want] {
			t.Fatalf("辺 %q がありません: %v", want, p.Impact)
		}
	}
}

func TestApplicationValidationUseAcceptWritesOnlyTwoLinesAndTheDeclarationOrderIsTheEvaluationOrder(t *testing.T) {
	dir := scratchWorkspaceBeforeSelection(t)
	before := treeDigest(t, dir)

	decision, err := accept(dir, mustJSON(t, validationUseCandidate(revisionOfDir(t, dir), orderEntrySelection()...)), acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}

	var changed []string
	for path, digest := range treeDigest(t, dir) {
		if before[path] != digest {
			changed = append(changed, path)
		}
	}
	if len(changed) != 1 || changed[0] != orderEntryApplicationSource {
		t.Fatalf("変更されたfile=%v", changed)
	}
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderEntryApplicationSource)))
	if err != nil {
		t.Fatal(err)
	}
	want := "    uses semantic security\n" +
		"    uses validation sales.order_line.quantity_positive\n" +
		"    uses validation sales.order.lines.line_number_unique\n"
	if !strings.Contains(string(content), want) {
		t.Fatalf("宣言が最後の`uses`の後ろへ、宣言順に入っていません:\n%s", content)
	}
	for _, keep := range []string{"database sales {", "database master {", "contract sales_core.order ^4"} {
		if !strings.Contains(string(content), keep) {
			t.Fatalf("既存宣言 %q が失われています", keep)
		}
	}
	for _, c := range decision.Conformance {
		if !c.Passed {
			t.Fatalf("受理後のConformance %s が通っていません", c.Name)
		}
	}

	// Toolを介さず、通常のLoaderで読み直し、Driverが宣言順のまま解決する。
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := driver.NewResolver(loaded.World).ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].ID != "sales.order_line.quantity_positive" || rules[1].ID != "sales.order.lines.line_number_unique" {
		t.Fatalf("Driverが解決した順序=%+v", rules)
	}
}

func TestApplicationValidationUseKeepsTheOrderTheCandidateDeclares(t *testing.T) {
	dir := scratchWorkspaceBeforeSelection(t)
	reversed := []Operation{
		validationUseOp("order_entry", "sales.order.lines.line_number_unique"),
		validationUseOp("order_entry", "sales.order_line.quantity_positive"),
	}
	if _, err := accept(dir, mustJSON(t, validationUseCandidate(revisionOfDir(t, dir), reversed...)), acceptOptions{}); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := driver.NewResolver(loaded.World).ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].ID != "sales.order.lines.line_number_unique" {
		t.Fatalf("candidateの順序が評価順へ伝わっていません: %+v", rules)
	}
}

func TestApplicationValidationUseRejectsWhatItCannotPlaceOrJustify(t *testing.T) {
	cases := []struct {
		name string
		op   Operation
		code string
	}{
		{"unknown application", validationUseOp("nope", "sales.order_line.quantity_positive"), "APPLICATION_NOT_FOUND"},
		{"unknown validation", validationUseOp("order_entry", "sales.order_line.missing_rule"), "VALIDATION_NOT_FOUND"},
		{
			"placement belongs to another application",
			Operation{
				Op: opAddApplicationValidationUse, Placement: "applications/repair-intake/application.tei",
				ApplicationValidationUse: &ApplicationValidationUseSpec{
					Application: "order_entry", Validation: "sales.order_line.quantity_positive",
				},
			},
			"PLACEMENT_NOT_OWNED",
		},
	}
	for _, tc := range cases {
		dir := scratchWorkspaceBeforeSelection(t)
		ev, err := evaluate(dir, mustJSON(t, validationUseCandidate(revisionOfDir(t, dir), tc.op)))
		if err != nil {
			t.Fatal(err)
		}
		if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), tc.code) {
			t.Fatalf("%s: acceptable=%v codes=%v want %s", tc.name, ev.preview.Acceptable, diagnosticCodes(ev.preview), tc.code)
		}
	}

	// 既に宣言済みなら、二度目は拒否する（同じ規則を二重に評価させない）。
	dir := scratchWorkspaceBeforeSelection(t)
	first := validationUseCandidate(revisionOfDir(t, dir), validationUseOp("order_entry", "sales.order_line.quantity_positive"))
	if _, err := accept(dir, mustJSON(t, first), acceptOptions{}); err != nil {
		t.Fatal(err)
	}
	again := validationUseCandidate(revisionOfDir(t, dir), validationUseOp("order_entry", "sales.order_line.quantity_positive"))
	ev, err := evaluate(dir, mustJSON(t, again))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "APPLICATION_VALIDATION_USE_ALREADY_EXISTS") {
		t.Fatalf("重複宣言を拒否していません: %v", diagnosticCodes(ev.preview))
	}
}

// 層をまたぐ正しさ（そのnamespaceをApplicationが使っているか）は、境界が再実装せず、
// 共有Validationのdiagnosticとして現れる。
func TestApplicationValidationUseLeavesCrossLayerCorrectnessToTheSharedValidation(t *testing.T) {
	dir := scratchWorkspaceBeforeSelection(t)
	op := Operation{
		Op: opAddApplicationValidationUse, Placement: "applications/repair-intake/application.tei",
		ApplicationValidationUse: &ApplicationValidationUseSpec{
			Application: "repair_intake", Validation: "sales.order_line.quantity_positive",
		},
	}
	c := validationUseCandidate(revisionOfDir(t, dir), op)
	c.Evidence = []Evidence{{Source: "applications/repair-intake/application.tei"}}
	ev, err := evaluate(dir, mustJSON(t, c))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "APPLICATION_VALIDATION_SEMANTIC_NOT_USED") {
		t.Fatalf("namespaceを使わないApplicationへの宣言を拒否していません: acceptable=%v codes=%v", ev.preview.Acceptable, diagnosticCodes(ev.preview))
	}
}

// 宣言順が評価順なので、順序の違うcandidateは別のcandidateである（fingerprintが違う）。
func TestApplicationValidationUseOrderIsPartOfTheCandidateIdentity(t *testing.T) {
	dir := scratchWorkspaceBeforeSelection(t)
	revision := revisionOfDir(t, dir)
	forward := validationUseCandidate(revision, orderEntrySelection()...)
	backward := validationUseCandidate(revision, orderEntrySelection()[1], orderEntrySelection()[0])

	_, forwardFingerprint, err := normalizeCandidate(forward)
	if err != nil {
		t.Fatal(err)
	}
	_, backwardFingerprint, err := normalizeCandidate(backward)
	if err != nil {
		t.Fatal(err)
	}
	if forwardFingerprint == backwardFingerprint {
		t.Fatal("評価順の違う二つのcandidateが同じfingerprintになっています")
	}
}
