package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/validation"
	"tei/internal/workspace"
)

const orderFile = "semantic/sales/order.tei"

func addRelationCandidate(revision string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-order-line-order-relation",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "studio", Actor: "test"},
		Intent:            "OrderLineから、所属するOrderへ辿れるRelationを明示する（Order.linesの逆方向）",
		Evidence: []Evidence{
			{Subject: "sales.order.lines", Note: "逆方向のRelationは既にある"},
			{Source: orderFile},
		},
		Delta: []Operation{{
			Op: opAddRelation,
			AddRelation: &AddRelationSpec{
				Data: "sales.order_line", ID: "sales.order_line.order", Symbol: "order",
				Target: "sales.order", Cardinality: "one",
			},
		}},
	}
}

func TestAddRelationPreviewShowsTheDiffTheImpactAndTheUndecidedPersistence(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	ev, err := evaluate(dir, mustJSON(t, addRelationCandidate(revisionOfDir(t, dir))))
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("Previewが受理可能になりません: %+v", allDiagnostics(p))
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	if got, want := keysOf(p.SemanticDiff), []string{"added relation:sales.order_line.order"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic diff=%v want %v", got, want)
	}
	if len(p.ApplicationDiff) != 0 {
		t.Fatalf("application diff=%v", keysOf(p.ApplicationDiff))
	}
	if !reflect.DeepEqual(p.ChangedCollections, []string{"semantics"}) {
		t.Fatalf("changed collections=%v", p.ChangedCollections)
	}
	for _, unchanged := range []string{"persistenceMappings", "databases", "migrations", "applications", "queries", "environments"} {
		if !hasCode(p.UnchangedCollections, unchanged) {
			t.Fatalf("collection %s が不変として示されていません: %v", unchanged, p.UnchangedCollections)
		}
	}

	if len(p.SourceEdits) != 1 || p.SourceEdits[0].Path != orderFile || p.SourceEdits[0].Created {
		t.Fatalf("editは既存のsemantic fileへの一件である必要があります: %+v", p.SourceEdits)
	}
	if p.SourceEdits[0].InsertedText != "    relation order to sales.order one\n" {
		t.Fatalf("挿入されるsource=%q", p.SourceEdits[0].InsertedText)
	}

	edges := map[string]bool{}
	for _, e := range p.Impact {
		edges[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	for _, want := range []string{
		"relation:sales.order_line.order -> data:sales.order_line via semantic.data.relation",
		"relation:sales.order_line.order -> data:sales.order via semantic.relation.target",
		"relation:sales.order_line.order -> persistence-mapping:sales_core.order_line via persistence.mapping.realizes-endpoint",
		"relation:sales.order_line.order -> persistence-mapping:sales_core.order via persistence.mapping.realizes-endpoint",
	} {
		if !edges[want] {
			t.Fatalf("impact edge %q がありません: %v", want, p.Impact)
		}
	}
	var codes []string
	for _, u := range p.Unresolved {
		codes = append(codes, u.Code)
	}
	for _, want := range []string{"RELATION_STORAGE_NOT_AUTHORED", "DRIVER_SCENARIO_NOT_RUN", "TARGET_PROJECTION_NOT_EVALUATED"} {
		if !hasCode(codes, want) {
			t.Fatalf("unresolved %s が残っていません: %v", want, codes)
		}
	}
}

func TestAddRelationChangesOnlyTheDataBlockItTargets(t *testing.T) {
	dir := scratchWorkspace(t)
	original, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	ev, err := evaluate(dir, mustJSON(t, addRelationCandidate(revisionOfDir(t, dir))))
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview=%+v err=%v", ev.preview, err)
	}
	after := string(ev.edits[0].after)

	// 挿入したtextを取り除くと、元のsourceにbyte単位で戻る（それ以外のbyteは変わっていない）。
	if strings.Replace(after, ev.edits[0].InsertedText, "", 1) != string(original) {
		t.Fatal("挿入したtext以外のbyteが変わっています")
	}
	orderLine := after[strings.Index(after, "data OrderLine {"):]
	closing := strings.Index(orderLine, "\n}")
	body := orderLine[:closing]
	if !strings.Contains(body, "    relation order to sales.order one") || !strings.HasSuffix(body, "    relation order to sales.order one") {
		t.Fatalf("Relationが対象data blockの末尾（閉じ括弧の直前）に入っていません:\n%s", body)
	}
	orderBlock := after[strings.Index(after, "data Order {"):strings.Index(after, "data OrderLine {")]
	if strings.Contains(orderBlock, "relation order to") {
		t.Fatal("別のdata blockへ挿入されています")
	}
}

func TestMinimalAuthoringLoopAddRelation(t *testing.T) {
	dir := scratchWorkspace(t)
	baseline := scratchWorkspace(t)
	revision := revisionOfDir(t, dir)
	data := mustJSON(t, addRelationCandidate(revision))
	before := treeDigest(t, dir)

	// Reject: authoritative sourceは変わらない。
	rejected, err := reject(dir, data, "Persistence側の設計を先に決めたい")
	if err != nil || rejected.Decision != "rejected" || rejected.SourceChanged {
		t.Fatalf("rejected=%+v err=%v", rejected, err)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	// Accept: 一つのfileの、挿入した行だけが変わる。
	decision, err := accept(dir, data, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("decision=%+v err=%v", decision, err)
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

	// toolを介さず共有loaderで再読込し、Canonicalへ現れることを確認する。
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("accepted sourceをtoolなしで再読込できません: %v", err)
	}
	for _, d := range validation.Validate(loaded.World) {
		if d.Severity == validation.Error {
			t.Fatalf("再読込後のCanonicalがValidationを満たしません: %s %s", d.Code, d.Message)
		}
	}
	found := false
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			for _, r := range d.Relations {
				if r.ID == "sales.order_line.order" {
					found = r.Target == "sales.order" && r.Cardinality == "one" && d.ID == "sales.order_line" && len(r.Keys) == 0
				}
			}
		}
	}
	if !found {
		t.Fatal("accepted Relationが再読込後のCanonical Worldにありません")
	}

	// Persistence Mapping・Databaseなど、semantic層以外のCanonicalは変わっていない。
	baselineWorld, err := workspace.Load(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(baselineWorld.World.Persistence, loaded.World.Persistence) ||
		!reflect.DeepEqual(baselineWorld.World.Databases, loaded.World.Databases) ||
		!reflect.DeepEqual(baselineWorld.World.Applications, loaded.World.Applications) {
		t.Fatal("Relation追加でPersistence Mapping / Database / Applicationが変わっています")
	}
	if decision.DerivedWorldSHA256 == "" || len(decision.SemanticDiff) != 1 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestAddRelationIsRefusedForInvalidOperations(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(c *Candidate)
		wantStage string
		wantCode  string
	}{
		{"data does not exist", func(c *Candidate) {
			c.Delta[0].AddRelation.Data = "sales.nothing"
			c.Delta[0].AddRelation.ID = "sales.nothing.order"
		}, "resolve", "DATA_NOT_FOUND"},
		{"relation already exists", func(c *Candidate) {
			c.Delta[0].AddRelation.Data = "sales.order"
			c.Delta[0].AddRelation.ID = "sales.order.customer"
			c.Delta[0].AddRelation.Symbol = "customer"
			c.Delta[0].AddRelation.Target = "customer.customer"
		}, "resolve", "RELATION_ALREADY_EXISTS"},
		{"target does not exist", func(c *Candidate) { c.Delta[0].AddRelation.Target = "sales.nothing" }, "validate", "UNKNOWN_SEMANTIC_RELATION_TARGET"},
		{"key source attribute does not exist", func(c *Candidate) {
			c.Delta[0].AddRelation.Keys = []RelationKeySpec{{SourceAttribute: "sales.order_line.nope", TargetAttribute: "sales.order.order_id"}}
		}, "validate", "UNKNOWN_SEMANTIC_RELATION_KEY_SOURCE_ATTRIBUTE"},
		{"key target attribute does not exist", func(c *Candidate) {
			c.Delta[0].AddRelation.Keys = []RelationKeySpec{{SourceAttribute: "sales.order_line.quantity", TargetAttribute: "sales.order.nope"}}
		}, "validate", "UNKNOWN_SEMANTIC_RELATION_KEY_TARGET_ATTRIBUTE"},
		{"key type mismatch", func(c *Candidate) {
			c.Delta[0].AddRelation.Keys = []RelationKeySpec{{SourceAttribute: "sales.order_line.quantity", TargetAttribute: "sales.order.order_id"}}
		}, "validate", "SEMANTIC_RELATION_KEY_TYPE_MISMATCH"},
		{"explicit placement", func(c *Candidate) { c.Delta[0].Placement = orderFile }, "normalize", "CANDIDATE_NORMALIZE"},
		{"id that does not follow the symbol", func(c *Candidate) { c.Delta[0].AddRelation.ID = "sales.order_line.parent" }, "normalize", "CANDIDATE_NORMALIZE"},
		{"cardinality", func(c *Candidate) { c.Delta[0].AddRelation.Cardinality = "few" }, "normalize", "CANDIDATE_NORMALIZE"},
		{"another payload", func(c *Candidate) { c.Delta[0].Data = &DataSpec{} }, "normalize", "CANDIDATE_NORMALIZE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			before := treeDigest(t, dir)
			c := addRelationCandidate(revisionOfDir(t, dir))
			tc.mutate(&c)
			data := mustJSON(t, c)

			ev, err := evaluate(dir, data)
			if err != nil {
				t.Fatal(err)
			}
			failing := ""
			for _, s := range ev.preview.Stages {
				if s.Status == "fail" && failing == "" {
					failing = s.Name
				}
			}
			if ev.preview.Acceptable || failing != tc.wantStage || !hasCode(diagnosticCodes(ev.preview), tc.wantCode) {
				t.Fatalf("stage=%q codes=%v want stage=%q code=%s", failing, diagnosticCodes(ev.preview), tc.wantStage, tc.wantCode)
			}
			decision, err := accept(dir, data, acceptOptions{})
			if err != nil || decision.Decision != "refused" || decision.SourceChanged {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestInsertIntoDataBlockHonoursLayoutAndLineEndings(t *testing.T) {
	relation := AddRelationSpec{Data: "x.b", ID: "x.b.a", Symbol: "a", Target: "x.a", Cardinality: "many",
		Keys: []RelationKeySpec{{SourceAttribute: "x.b.k", TargetAttribute: "x.a.k"}}}

	t.Run("picks the requested data block", func(t *testing.T) {
		src := "namespace x\n\ndata A {\n    attribute k type std.text\n}\n\ndata B {\n    attribute k type std.text\n}\n"
		got, inserted, err := insertIntoDataBlock([]byte(src), "B", relation)
		if err != nil {
			t.Fatal(err)
		}
		want := "namespace x\n\ndata A {\n    attribute k type std.text\n}\n\ndata B {\n    attribute k type std.text\n    relation a to x.a many\n        key x.b.k -> x.a.k\n}\n"
		if string(got) != want || inserted != "    relation a to x.a many\n        key x.b.k -> x.a.k\n" {
			t.Fatalf("got:\n%s", got)
		}
	})
	t.Run("keeps CRLF", func(t *testing.T) {
		src := "namespace x\r\n\r\ndata B {\r\n    attribute k type std.text\r\n}\r\n"
		got, _, err := insertIntoDataBlock([]byte(src), "B", relation)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ReplaceAll(string(got), "\r\n", ""), "\n") {
			t.Fatalf("CRLFのfileにLFが混ざっています: %q", got)
		}
		if !strings.Contains(string(got), "    relation a to x.a many\r\n        key x.b.k -> x.a.k\r\n}") {
			t.Fatalf("got: %q", got)
		}
	})
	t.Run("closing brace after tabs", func(t *testing.T) {
		src := "namespace x\n\ndata B {\n    attribute k type std.text\n\t}\n"
		got, _, err := insertIntoDataBlock([]byte(src), "B", relation)
		if err != nil || !strings.Contains(string(got), "        key x.b.k -> x.a.k\n\t}") {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
	t.Run("refuses a closing brace that is not at the start of its line", func(t *testing.T) {
		src := "namespace x\n\ndata B { attribute k type std.text }\n"
		if _, _, err := insertIntoDataBlock([]byte(src), "B", relation); err == nil {
			t.Fatal("行の途中の閉じ括弧へ挿入してしまいます")
		}
	})
	t.Run("refuses an unknown data symbol", func(t *testing.T) {
		src := "namespace x\n\ndata A {\n    attribute k type std.text\n}\n"
		if _, _, err := insertIntoDataBlock([]byte(src), "B", relation); err == nil {
			t.Fatal("存在しないdataへ挿入しようとしています")
		}
	})
	t.Run("does not mistake another declaration with the same symbol for the data block", func(t *testing.T) {
		src := "namespace x\n\ncapability B {\n    input k std.text\n}\n\ndata B {\n    attribute k type std.text\n}\n"
		got, _, err := insertIntoDataBlock([]byte(src), "B", relation)
		if err != nil || strings.Index(string(got), "relation a to") < strings.Index(string(got), "data B {") {
			t.Fatalf("data以外の宣言のblockへ挿入しています: got=%q err=%v", got, err)
		}
	})
	t.Run("does not mistake an attribute named like the data for its block", func(t *testing.T) {
		src := "namespace x\n\ndata A {\n    attribute B type std.text\n}\n\ndata B {\n    attribute k type std.text\n}\n"
		got, _, err := insertIntoDataBlock([]byte(src), "B", relation)
		if err != nil || strings.Index(string(got), "relation a to") < strings.Index(string(got), "data B {") {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
}

func TestAddRelationAndAddValidationShareOneFileAndRollBackTogether(t *testing.T) {
	dir := scratchWorkspace(t)
	original, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	c := addRelationCandidate(revisionOfDir(t, dir))
	c.Delta = append(c.Delta, Operation{
		Op: opAddValidation, Namespace: "sales", ID: "sales.order.total_positive", Kind: "positive",
		Attribute: "sales.order.total", Placement: orderFile,
	})
	data := mustJSON(t, c)
	before := treeDigest(t, dir)

	ev, err := evaluate(dir, data)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("preview=%+v err=%v", ev.preview, err)
	}
	if len(ev.preview.SourceEdits) != 1 {
		t.Fatalf("同じfileへの二つのoperationが一つのeditにまとまっていません: %+v", ev.preview.SourceEdits)
	}
	wantInserted := "    relation order to sales.order one\n\n\nvalidation sales.order.total_positive positive {\n    attribute sales.order.total\n}\n"
	if got := ev.preview.SourceEdits[0].InsertedText; got != wantInserted {
		t.Fatalf("inserted=%q want %q", got, wantInserted)
	}
	afterText := string(ev.edits[0].after)
	if !strings.HasPrefix(afterText, strings.TrimRight(string(original), "\n")[:20]) ||
		strings.Index(afterText, "relation order to sales.order one") > strings.Index(afterText, "validation sales.order.total_positive") {
		t.Fatalf("挿入と追記の順序が想定と違います:\n%s", afterText)
	}

	// 受理後Conformanceが失敗すると、挿入も追記も一緒に元のbyteへ戻る。
	target := filepath.Join(dir, filepath.FromSlash(orderFile))
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
	decision, err := accept(dir, data, acceptOptions{})
	if err != nil || decision.Decision != "refused" || decision.SourceChanged {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}
