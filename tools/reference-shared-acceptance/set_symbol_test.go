package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

// setSymbolCandidateは、既存subjectの表示名だけを変えるset-symbol候補を作る。
// Canonical IDはそのままなので、候補は改名であってidentity変更ではない。
func setSymbolCandidate(revision, kind, id, symbol string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-set-symbol-" + kind,
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "studio", Actor: "test"},
		Intent:            kind + " " + id + " の表示名を " + symbol + " へ改名する（Canonical IDは変えない）",
		Evidence:          []Evidence{{Source: orderFile, Note: "改名対象subjectを宣言している正本source"}},
		Delta: []Operation{{
			Op: opSetSymbol, Kind: kind, ID: id, Symbol: symbol,
		}},
	}
}

func unresolvedCodes(p Preview) []string {
	var out []string
	for _, u := range p.Unresolved {
		out = append(out, u.Code)
	}
	return out
}

func loadData(t *testing.T, dir, id string) (symbol string, found bool) {
	t.Helper()
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("accepted sourceをtoolなしで再読込できません: %v", err)
	}
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			if string(d.ID) == id {
				return d.Symbol, true
			}
		}
	}
	return "", false
}

// 受入条件1: 名前変更はbase-revision / semantic-diff / impactを含む全stageを通り、
// acceptまではauthoritative sourceを書き換えない。accept後は同じCanonical IDのまま
// symbolだけが変わり、他のfileは変わらない。
func TestSetSymbolRenamesDataWithoutChangingIdentityOrWritingBeforeAccept(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	revision := revisionOfDir(t, dir)
	raw := mustJSON(t, setSymbolCandidate(revision, "data", "sales.order_line", "Order_Line"))

	ev, err := evaluate(dir, raw)
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("改名候補が受理可能になりません: %+v", allDiagnostics(p))
	}
	var names []string
	for _, s := range p.Stages {
		names = append(names, s.Name)
		if s.Status != "pass" {
			t.Fatalf("stage %s=%s diag=%+v", s.Name, s.Status, s.Diagnostics)
		}
	}
	if !reflect.DeepEqual(names, wantStages) {
		t.Fatalf("stage順序=%v want %v", names, wantStages)
	}

	if got, want := keysOf(p.SemanticDiff), []string{"changed data:sales.order_line"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic差分=%v want %v", got, want)
	}
	if !reflect.DeepEqual(p.ChangedCollections, []string{"semantics"}) {
		t.Fatalf("変わったcollection=%v", p.ChangedCollections)
	}
	if len(p.ApplicationDiff) != 0 || len(p.EnvironmentDiff) != 0 {
		t.Fatal("semantic層以外のdiffが出ています")
	}
	if len(p.SourceEdits) != 1 || p.SourceEdits[0].Created || p.SourceEdits[0].Path != orderFile {
		t.Fatalf("editは既存のsemantic file一件である必要があります: %+v", p.SourceEdits)
	}
	if p.SourceEdits[0].InsertedText != "Order_Line" || p.SourceEdits[0].RemovedText != "OrderLine" {
		t.Fatalf("置換内容 inserted=%q removed=%q", p.SourceEdits[0].InsertedText, p.SourceEdits[0].RemovedText)
	}
	if p.CurrentModelRevision != revision || p.ModelRevisionAfter == "" || p.ModelRevisionAfter == revision {
		t.Fatalf("revision before=%s after=%s", p.CurrentModelRevision, p.ModelRevisionAfter)
	}

	// acceptまではauthoritative sourceを一切書かない。
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	decision, err := accept(dir, raw, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accept=%+v err=%v", decision, err)
	}
	after := treeDigest(t, dir)
	var changed []string
	for path, digest := range before {
		if after[path] != digest {
			changed = append(changed, path)
		}
	}
	if !reflect.DeepEqual(changed, []string{orderFile}) || len(after) != len(before) {
		t.Fatalf("変わったfile=%v file数 before=%d after=%d", changed, len(before), len(after))
	}
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "data Order_Line {") || strings.Contains(string(content), "data OrderLine {") {
		t.Fatalf("source上のdata宣言が改名されていません:\n%s", content)
	}
	if symbol, found := loadData(t, dir, "sales.order_line"); !found || symbol != "Order_Line" {
		t.Fatalf("再読込後のData symbol=%q found=%v", symbol, found)
	}
}

// 受入条件2: identifierを変える候補は拒否する。symbolから導かれるsegmentがIDの
// 最後のsegmentと異なる場合はnormalizeで止め、存在しないsubjectはresolveで止める。
func TestSetSymbolRefusesIdentifierChangesAndUnknownOrUnchangedSubjects(t *testing.T) {
	cases := []struct {
		name            string
		stage, code     string
		messageContains string
		mutate          func(*Candidate)
	}{
		{"identifier change", "normalize", "CANDIDATE_NORMALIZE", "identifier", func(c *Candidate) {
			c.Delta[0].Symbol = "Item"
		}},
		{"unknown kind", "normalize", "CANDIDATE_NORMALIZE", "", func(c *Candidate) {
			c.Delta[0].Kind = "type"
		}},
		{"explicit placement", "normalize", "CANDIDATE_NORMALIZE", "", func(c *Candidate) {
			c.Delta[0].Placement = orderFile
		}},
		{"mixed payload", "normalize", "CANDIDATE_NORMALIZE", "", func(c *Candidate) {
			c.Delta[0].Data = &DataSpec{}
		}},
		{"unknown subject", "resolve", "SYMBOL_SUBJECT_NOT_FOUND", "", func(c *Candidate) {
			c.Delta[0].ID = "sales.missing"
			c.Delta[0].Symbol = "Missing"
		}},
		{"unchanged symbol", "resolve", "SYMBOL_UNCHANGED", "", func(c *Candidate) {
			c.Delta[0].Symbol = "OrderLine"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			baseline := treeDigest(t, dir)
			c := setSymbolCandidate(revisionOfDir(t, dir), "data", "sales.order_line", "Order_Line")
			tc.mutate(&c)
			raw := mustJSON(t, c)

			ev, err := evaluate(dir, raw)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable {
				t.Fatalf("%s が受理されました: %+v", tc.name, ev.preview)
			}
			failing := ""
			for _, s := range ev.preview.Stages {
				if s.Status == "fail" && failing == "" {
					failing = s.Name
				}
			}
			if failing != tc.stage || !hasCode(diagnosticCodes(ev.preview), tc.code) {
				t.Fatalf("stage=%q codes=%v want stage=%q code=%s", failing, diagnosticCodes(ev.preview), tc.stage, tc.code)
			}
			if tc.messageContains != "" {
				found := false
				for _, d := range allDiagnostics(ev.preview) {
					if strings.Contains(d.Message, tc.messageContains) {
						found = true
					}
				}
				if !found {
					t.Fatalf("診断messageに %q がありません: %+v", tc.messageContains, allDiagnostics(ev.preview))
				}
			}
			decision, err := accept(dir, raw, acceptOptions{})
			if err != nil || decision.Decision != "refused" || decision.SourceChanged {
				t.Fatalf("拒否されるべき候補がsourceを変えました: %+v err=%v", decision, err)
			}
			assertTreeUnchanged(t, baseline, treeDigest(t, dir))
		})
	}
}

// 受入条件3: 依存定義を変える改名は、relation keyやcandidate keyのfieldをimpactに出す。
// symbolだけを変えてもIDは不変なので、既存のchangeanalysisがchanged subjectの依存辺を辿れる。
func TestSetSymbolSurfacesDependentRelationKeysAndCandidateKeyFieldsInImpact(t *testing.T) {
	t.Run("relation key attributes", func(t *testing.T) {
		dir := scratchWorkspace(t)
		baseline := treeDigest(t, dir)
		raw := mustJSON(t, setSymbolCandidate(revisionOfDir(t, dir), "relation", "sales.order.customer", "Customer"))
		ev, err := evaluate(dir, raw)
		if err != nil {
			t.Fatal(err)
		}
		p := ev.preview
		if !p.Acceptable {
			t.Fatalf("relation改名が受理可能になりません: %+v", allDiagnostics(p))
		}
		if got, want := keysOf(p.SemanticDiff), []string{"changed relation:sales.order.customer"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("semantic差分=%v want %v", got, want)
		}
		edges := map[string]bool{}
		for _, e := range p.Impact {
			edges[e.From+" -> "+e.To+" via "+e.Via] = true
		}
		for _, want := range []string{
			"relation:sales.order.customer -> attribute:sales.order.customer_id via semantic.relation.key.source",
			"relation:sales.order.customer -> attribute:customer.customer.customer_id via semantic.relation.key.target",
		} {
			if !edges[want] {
				t.Fatalf("relation keyのimpact edge %q がありません: %+v", want, p.Impact)
			}
		}
		if !hasCode(unresolvedCodes(p), "RELATION_RETRACTION_REQUIRES_REVIEW") {
			t.Fatalf("改名の再確認要求が残っていません: %v", unresolvedCodes(p))
		}
		assertTreeUnchanged(t, baseline, treeDigest(t, dir))

		decision, err := accept(dir, raw, acceptOptions{})
		if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
			t.Fatalf("relation改名のaccept=%+v err=%v", decision, err)
		}
		loaded, err := workspace.Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, ns := range loaded.World.Semantics {
			for _, d := range ns.Data {
				for _, r := range d.Relations {
					if string(r.ID) == "sales.order.customer" {
						found = r.Symbol == "Customer" && len(r.Keys) == 1 &&
							string(r.Keys[0].SourceAttribute) == "sales.order.customer_id"
					}
				}
			}
		}
		if !found {
			t.Fatal("改名後のrelationが同じID・keyのまま再読込できません")
		}
	})

	t.Run("candidate key fields", func(t *testing.T) {
		dir := scratchWorkspace(t)
		// 参照workspaceにはcandidate keyが無いため、add-candidate-keyで先に宣言を確立する。
		if d, err := accept(dir, mustJSON(t, logicalRuleCandidate(revisionOfDir(t, dir))), acceptOptions{}); err != nil ||
			d.Decision != "accepted" {
			t.Fatalf("candidate keyの前提を作れません: %+v err=%v", d, err)
		}
		baseline := treeDigest(t, dir)
		raw := mustJSON(t, setSymbolCandidate(revisionOfDir(t, dir), "candidate-key", "sales.order_line.business_line", "BusinessLine"))
		ev, err := evaluate(dir, raw)
		if err != nil {
			t.Fatal(err)
		}
		p := ev.preview
		if !p.Acceptable {
			t.Fatalf("candidate-key改名が受理可能になりません: %+v", allDiagnostics(p))
		}
		if got, want := keysOf(p.SemanticDiff), []string{"changed candidate-key:sales.order_line.business_line"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("semantic差分=%v want %v", got, want)
		}
		edges := map[string]bool{}
		for _, e := range p.Impact {
			edges[e.From+" -> "+e.To+" via "+e.Via] = true
		}
		for _, want := range []string{
			"candidate-key:sales.order_line.business_line -> data:sales.order_line via semantic.data.candidate-key",
			"candidate-key:sales.order_line.business_line -> attribute:sales.order_line.line_number via semantic.candidate-key.field",
		} {
			if !edges[want] {
				t.Fatalf("candidate keyのimpact edge %q がありません: %+v", want, p.Impact)
			}
		}
		if !hasCode(unresolvedCodes(p), "LOGICAL_RULE_RETRACTION_REQUIRES_REVIEW") {
			t.Fatalf("改名の再確認要求が残っていません: %v", unresolvedCodes(p))
		}

		decision, err := accept(dir, raw, acceptOptions{})
		if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
			t.Fatalf("candidate-key改名のaccept=%+v err=%v", decision, err)
		}
		after := treeDigest(t, dir)
		var changed []string
		for path, digest := range baseline {
			if after[path] != digest {
				changed = append(changed, path)
			}
		}
		if !reflect.DeepEqual(changed, []string{orderFile}) {
			t.Fatalf("変わったfile=%v want %s", changed, orderFile)
		}
		loaded, err := workspace.Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, ns := range loaded.World.Semantics {
			for _, d := range ns.Data {
				for _, k := range d.CandidateKeys {
					if string(k.ID) == "sales.order_line.business_line" {
						found = k.Symbol == "BusinessLine" && len(k.Attributes) == 2
					}
				}
			}
		}
		if !found {
			t.Fatal("改名後のcandidate keyが同じID・fieldのまま再読込できません")
		}
	})
}

// 受入条件4: #1804の名前置換Proposalを、このset-symbol候補へ変換できることをREADMEで示す。
// 生成器は作らず、変換手順と例を文書として固定する。
func TestSetSymbolReadmeDocumentsTheNameReplacementConversion(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "tools", "reference-shared-acceptance", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{`"op": "set-symbol"`, "#1804", "Order_Line"} {
		if !strings.Contains(text, want) {
			t.Fatalf("READMEに %q がありません", want)
		}
	}
}
