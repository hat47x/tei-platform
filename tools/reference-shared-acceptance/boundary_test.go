package main

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/changeanalysis"
)

var wantStages = []string{
	"decode", "normalize", "base-revision", "resolve", "apply", "reload",
	"validate", "architecture-policy", "semantic-diff", "impact", "acceptance-policy",
}

func TestScratchWorkspaceIsFaithfulToTheReferenceBaseline(t *testing.T) {
	scratch := scratchWorkspace(t)
	s, err := takeSnapshot(scratch)
	if err != nil {
		t.Fatal(err)
	}
	_, encoded, err := worldSHA256(s.loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, bytes.ReplaceAll(golden, []byte("\r\n"), []byte("\n"))) {
		t.Fatal("scratch copyから得たCanonical Worldが、Gate B / Gate Cが固定したreference/world.jsonと一致しません")
	}
}

func TestPreviewShowsTheDeltaWithoutChangingAuthoritativeSource(t *testing.T) {
	dir := scratchWorkspace(t)
	revision := revisionOfDir(t, dir)
	before := treeDigest(t, dir)

	ev, err := evaluate(dir, mustJSON(t, unitPriceCandidate(revision)))
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("Previewが受理可能になりません: %+v", allDiagnostics(p))
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	var names []string
	for _, s := range p.Stages {
		names = append(names, s.Name)
		if s.Status != "pass" {
			t.Fatalf("stage %s=%s", s.Name, s.Status)
		}
	}
	if !reflect.DeepEqual(names, wantStages) {
		t.Fatalf("stage順序=%v want %v", names, wantStages)
	}

	if len(p.SemanticDiff) != 1 || p.SemanticDiff[0].Kind != "validation" || p.SemanticDiff[0].ID != "sales.order_line.unit_price_positive" || p.SemanticDiff[0].Change != "added" {
		t.Fatalf("semantic差分が最小deltaではありません: %+v", p.SemanticDiff)
	}
	if !reflect.DeepEqual(p.ChangedCollections, []string{"semantics"}) {
		t.Fatalf("semantic層以外のcollectionが変わっています: %v", p.ChangedCollections)
	}
	for _, unchanged := range []string{"applications", "persistenceMappings", "databases", "queries", "environments"} {
		if !hasCode(p.UnchangedCollections, unchanged) {
			t.Fatalf("collection %s が不変として示されていません: %v", unchanged, p.UnchangedCollections)
		}
	}

	if len(p.SourceEdits) != 1 || p.SourceEdits[0].Path != "semantic/sales/validation.tei" {
		t.Fatalf("source editがsemantic層の既存Validation fileへ閉じていません: %+v", p.SourceEdits)
	}
	wantText := "\n\nvalidation sales.order_line.unit_price_positive positive {\n    attribute sales.order_line.unit_price\n}\n"
	if p.SourceEdits[0].InsertedText != wantText {
		t.Fatalf("追記されるsourceが最小ではありません:\n%q\nwant\n%q", p.SourceEdits[0].InsertedText, wantText)
	}
	if p.CurrentModelRevision != revision || p.ModelRevisionAfter == "" || p.ModelRevisionAfter == revision {
		t.Fatalf("revision=%s after=%s", p.CurrentModelRevision, p.ModelRevisionAfter)
	}
}

func TestPreviewImpactTracesToExactDependenciesAndKeepsUnresolvedVisible(t *testing.T) {
	dir := scratchWorkspace(t)
	ev, err := evaluate(dir, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))))
	if err != nil {
		t.Fatal(err)
	}
	want := []changeanalysis.ImpactEdge{
		{From: "attribute:sales.order_line.unit_price", To: "data:sales.order_line", Via: "semantic.data.attribute"},
		{From: "data:sales.order_line", To: "relation:sales.order.lines", Via: "semantic.relation.target"},
		{From: "relation:sales.order.lines", To: "pattern:order_entry.order_lines", Via: "application.pattern.relation"},
		{From: "validation:sales.order_line.unit_price_positive", To: "application:order_entry", Via: "application.uses-semantic-namespace"},
		{From: "validation:sales.order_line.unit_price_positive", To: "attribute:sales.order_line.unit_price", Via: "semantic.validation.attribute"},
	}
	if len(ev.preview.Impact) != len(want) {
		t.Fatalf("impact edge数=%d want %d: %+v", len(ev.preview.Impact), len(want), ev.preview.Impact)
	}
	for i, edge := range ev.preview.Impact {
		if edge.From != want[i].From || edge.To != want[i].To || edge.Via != want[i].Via || edge.Reason == "" {
			t.Fatalf("impact[%d]=%+v want %+v", i, edge, want[i])
		}
	}
	var codes []string
	for _, u := range ev.preview.Unresolved {
		codes = append(codes, u.Code)
	}
	for _, c := range []string{"APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL", "DRIVER_SCENARIO_NOT_RUN", "TARGET_PROJECTION_NOT_EVALUATED"} {
		if !hasCode(codes, c) {
			t.Fatalf("unresolved %s が成功へ読み替えられて消えています: %v", c, codes)
		}
	}
}

func TestAcceptUpdatesSourceAndTheAcceptedMeaningReloadsWithoutTheTool(t *testing.T) {
	dir := scratchWorkspace(t)
	baseline := scratchWorkspace(t)
	revision := revisionOfDir(t, dir)

	decision, err := accept(dir, mustJSON(t, unitPriceCandidate(revision)), acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("decision=%+v", decision)
	}
	for _, c := range decision.Conformance {
		if !c.Passed {
			t.Fatalf("受理後のConformance %s が通っていません", c.Name)
		}
	}
	if decision.ModelRevisionBefore != revision || decision.ModelRevisionAfter == revision {
		t.Fatalf("revision before=%s after=%s", decision.ModelRevisionBefore, decision.ModelRevisionAfter)
	}

	// 以降はtoolのコードを一切使わず、共有loader / Validationだけでaccepted sourceを読み直す。
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("accepted sourceをtoolなしで再読込できません: %v", err)
	}
	for _, d := range validation.Validate(loaded.World) {
		if d.Severity == validation.Error {
			t.Fatalf("再読込後のCanonicalがValidationを満たしません: %s %s", d.Code, d.Message)
		}
	}
	rule, ok := findAcceptedRule(loaded, "sales.order_line.unit_price_positive")
	if !ok {
		t.Fatal("accepted Validationが再読込後のCanonical Worldにありません")
	}
	original, err := workspace.Load(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if _, existed := findAcceptedRule(original, "sales.order_line.unit_price_positive"); existed {
		t.Fatal("変更前のReference baselineに既にruleがあります")
	}

	// before / after behavior: 同じ入力に対する判定が、accepted ruleによって変わる。
	for _, tc := range []struct {
		price string
		want  validationeval.Result
	}{
		{"12.50", validationeval.ResultSatisfied},
		{"0", validationeval.ResultNotSatisfied},
		{"-3", validationeval.ResultNotSatisfied},
	} {
		value, ok := new(big.Rat).SetString(tc.price)
		if !ok {
			t.Fatalf("数値%qを読めません", tc.price)
		}
		got, err := validationeval.Evaluate(rule, numberSource{core.CanonicalID("sales.order_line.unit_price"): value})
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("単価%sの判定=%s want %s", tc.price, got, tc.want)
		}
	}
}

func TestAcceptRefusesAStaleCandidateAndNeverAppliesItSilently(t *testing.T) {
	dir := scratchWorkspace(t)
	staleRevision := revisionOfDir(t, dir)

	first := unitPriceCandidate(staleRevision)
	if d, err := accept(dir, mustJSON(t, first), acceptOptions{}); err != nil || d.Decision != "accepted" {
		t.Fatalf("先行するcandidateを受理できません: %+v %v", d, err)
	}
	afterFirst := treeDigest(t, dir)

	second := unitPriceCandidate(staleRevision)
	second.ID = "cand-total-positive"
	second.Delta = []Operation{{
		Op: opAddValidation, Namespace: "sales", ID: "sales.order.total_positive", Kind: "positive", Attribute: "sales.order.total",
	}}

	ev, err := evaluate(dir, mustJSON(t, second))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || ev.preview.Stale == nil {
		t.Fatalf("staleなcandidateがPreviewで検出されていません: %+v", ev.preview)
	}
	if got := stageStatus(t, ev.preview, "base-revision").Status; got != "fail" {
		t.Fatalf("base-revision=%s", got)
	}
	for _, name := range wantStages[4:] {
		if got := stageStatus(t, ev.preview, name).Status; got != "skipped" {
			t.Fatalf("staleなのにstage %s=%s（後続は実行しない）", name, got)
		}
	}

	decision, err := accept(dir, mustJSON(t, second), acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged || decision.Stale == nil {
		t.Fatalf("decision=%+v", decision)
	}
	wantOptions := []string{"reject", "re-read-current-revision-and-regenerate-candidate", "explicit-rebase-candidate", "manual-conflict-resolution"}
	if !reflect.DeepEqual(decision.Stale.Options, wantOptions) {
		t.Fatalf("staleの分岐先=%v", decision.Stale.Options)
	}
	assertTreeUnchanged(t, afterFirst, treeDigest(t, dir))

	// 現在のrevisionから作り直せば受理できる。
	regenerated := second
	regenerated.BaseModelRevision = decision.Stale.CurrentModelRevision
	if d, err := accept(dir, mustJSON(t, regenerated), acceptOptions{}); err != nil || d.Decision != "accepted" {
		t.Fatalf("現在のrevisionから再生成したcandidateを受理できません: %+v %v", d, err)
	}
}

func TestInvalidCandidatesAreRefusedBeforeAnySourceChange(t *testing.T) {
	type mutation func(c *Candidate)
	cases := []struct {
		name      string
		mutate    mutation
		raw       string
		wantStage string
		wantCode  string
	}{
		{"unknown JSON field", nil, `{"schema":"tei.reference.change-candidate/v0","surprise":true}`, "decode", "CANDIDATE_DECODE"},
		{"unsupported operation", func(c *Candidate) { c.Delta[0].Op = "delete-everything" }, "", "normalize", "CANDIDATE_NORMALIZE"},
		{"relation-unique without relation", func(c *Candidate) { c.Delta[0].Kind = "relation-unique" }, "", "normalize", "CANDIDATE_NORMALIZE"},
		{"id outside the namespace", func(c *Candidate) { c.Delta[0].ID = "customer.thing" }, "", "normalize", "CANDIDATE_NORMALIZE"},
		{"unknown attribute", func(c *Candidate) { c.Delta[0].Attribute = "sales.order_line.no_such_attribute" }, "", "resolve", "ATTRIBUTE_NOT_FOUND"},
		{"validation already exists", func(c *Candidate) { c.Delta[0].ID = "sales.order_line.quantity_positive" }, "", "resolve", "VALIDATION_ALREADY_EXISTS"},
		{"unknown namespace", func(c *Candidate) { c.Delta[0].Namespace = "nosuch"; c.Delta[0].ID = "nosuch.thing" }, "", "resolve", "NAMESPACE_NOT_FOUND"},
		{"evidence subject does not exist", func(c *Candidate) { c.Evidence = []Evidence{{Subject: "sales.order.invented_rule"}} }, "", "resolve", "EVIDENCE_SUBJECT_NOT_FOUND"},
		{"evidence source does not exist", func(c *Candidate) { c.Evidence = []Evidence{{Source: "semantic/sales/invented.tei"}} }, "", "resolve", "EVIDENCE_SOURCE_NOT_FOUND"},
		{"placement in another layer", func(c *Candidate) { c.Delta[0].Placement = "applications/order-entry/application.tei" }, "", "resolve", "LAYER_VIOLATION"},
		{"placement in another namespace", func(c *Candidate) { c.Delta[0].Placement = "semantic/customer/customer.tei" }, "", "resolve", "PLACEMENT_NAMESPACE_MISMATCH"},
		{"placement is not a managed file", func(c *Candidate) { c.Delta[0].Placement = "semantic/sales/nowhere.tei" }, "", "resolve", "PLACEMENT_NOT_MANAGED_SOURCE"},
		{"positive on a text attribute", func(c *Candidate) {
			c.Delta[0].ID = "sales.order.external_id_positive"
			c.Delta[0].Attribute = "sales.order.external_id"
		}, "", "validate", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			before := treeDigest(t, dir)
			data := []byte(tc.raw)
			if tc.mutate != nil {
				c := unitPriceCandidate(revisionOfDir(t, dir))
				tc.mutate(&c)
				data = mustJSON(t, c)
			}

			ev, err := evaluate(dir, data)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable {
				t.Fatalf("不正なcandidateが受理可能になりました: %+v", ev.preview.Stages)
			}
			failing := ""
			for _, s := range ev.preview.Stages {
				if s.Status == "fail" && failing == "" {
					failing = s.Name
				}
			}
			if failing != tc.wantStage {
				t.Fatalf("最初に失敗したstage=%s want %s: %+v", failing, tc.wantStage, allDiagnostics(ev.preview))
			}
			if tc.wantCode != "" && !hasCode(diagnosticCodes(ev.preview), tc.wantCode) {
				t.Fatalf("診断=%v want %s", diagnosticCodes(ev.preview), tc.wantCode)
			}
			if tc.wantStage == "validate" && len(stageStatus(t, ev.preview, "validate").Diagnostics) == 0 {
				t.Fatal("validate stageが診断を示していません")
			}

			decision, err := accept(dir, data, acceptOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if decision.Decision != "refused" || decision.SourceChanged || len(decision.Refusals) == 0 {
				t.Fatalf("decision=%+v", decision)
			}
			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestWholeModelSizedDeltasAreRefusedByThePolicy(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	c := unitPriceCandidate(revisionOfDir(t, dir))
	c.Delta = nil
	for i := 1; i <= maxSubjectChanges+1; i++ {
		id := "sales.order_line.quantity_check_" + string(rune('a'+i-1))
		c.Delta = append(c.Delta, Operation{Op: opAddValidation, Namespace: "sales", ID: id, Kind: "positive", Attribute: "sales.order_line.quantity"})
	}
	ev, err := evaluate(dir, mustJSON(t, c))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "DELTA_TOO_LARGE") {
		t.Fatalf("大きすぎるdeltaが受理可能です: %v", diagnosticCodes(ev.preview))
	}
	if got := stageStatus(t, ev.preview, "semantic-diff").Status; got != "pass" {
		t.Fatalf("semantic-diff=%s（deltaは宣言どおりで、policyだけが拒否する）", got)
	}
	decision, err := accept(dir, mustJSON(t, c), acceptOptions{})
	if err != nil || decision.Decision != "refused" {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestRejectLeavesAuthoritativeSourceByteIdentical(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	revision := revisionOfDir(t, dir)

	decision, err := reject(dir, mustJSON(t, unitPriceCandidate(revision)), "今回は見送る")
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "rejected" || decision.SourceChanged || decision.ModelRevisionBefore != decision.ModelRevisionAfter || decision.Reason == "" {
		t.Fatalf("decision=%+v", decision)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	stale := unitPriceCandidate("sha256:" + strings.Repeat("0", 64))
	staleDecision, err := reject(dir, mustJSON(t, stale), "古いrevision")
	if err != nil || staleDecision.Decision != "rejected" {
		t.Fatalf("staleなcandidateもrejectできる必要があります: %+v %v", staleDecision, err)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestAcceptDetectsSourceChangedAfterPreviewAndRestoresPartialWrites(t *testing.T) {
	dir := scratchWorkspace(t)
	c := unitPriceCandidate(revisionOfDir(t, dir))
	c.Delta = append(c.Delta, Operation{
		Op: opAddValidation, Namespace: "sales", ID: "sales.order.total_positive", Kind: "positive",
		Attribute: "sales.order.total", Placement: "semantic/sales/order.tei",
	})
	data := mustJSON(t, c)
	before := treeDigest(t, dir)

	// edit順は path 順: [0] semantic/sales/order.tei, [1] semantic/sales/validation.tei
	concurrent := filepath.Join(dir, "semantic", "sales", "validation.tei")
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index == 1 {
			content, err := os.ReadFile(concurrent)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(concurrent, append(content, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	defer func() { beforeEachWriteHook = nil }()

	decision, err := accept(dir, data, acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged || !hasCode(refusalCodes(decision), "SOURCE_CHANGED_DURING_ACCEPT") {
		t.Fatalf("decision=%+v", decision)
	}
	after := treeDigest(t, dir)
	if after["semantic/sales/order.tei"] != before["semantic/sales/order.tei"] {
		t.Fatal("先に書いたfileが元へ戻されていません（partial writeが残っています）")
	}
	if after["semantic/sales/validation.tei"] == before["semantic/sales/validation.tei"] {
		t.Fatal("並行して入った変更を上書きしてしまいました")
	}
}

func refusalCodes(d Decision) []string {
	var out []string
	for _, r := range d.Refusals {
		out = append(out, r.Code)
	}
	return out
}

func findAcceptedRule(loaded workspace.Result, id core.CanonicalID) (semantic.Validation, bool) {
	for _, ns := range loaded.World.Semantics {
		if r, found := findRule(ns.Validations, id); found {
			return r, true
		}
	}
	return semantic.Validation{}, false
}

func TestAcceptRestoresTheSourceWhenPostAcceptConformanceFails(t *testing.T) {
	dir := scratchWorkspace(t)
	c := unitPriceCandidate(revisionOfDir(t, dir))
	before := treeDigest(t, dir)

	target := filepath.Join(dir, "semantic", "sales", "validation.tei")
	afterWritesHook = func(root string) {
		content, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, append(content, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { afterWritesHook = nil }()

	decision, err := accept(dir, mustJSON(t, c), acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged || !hasCode(refusalCodes(decision), "POST_ACCEPT_CONFORMANCE_FAILED") {
		t.Fatalf("decision=%+v", decision)
	}
	failed := false
	for _, check := range decision.Conformance {
		if check.Name == "model-revision-matches-preview" && !check.Passed {
			failed = true
		}
	}
	if !failed {
		t.Fatalf("失敗したConformance checkが記録されていません: %+v", decision.Conformance)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}
