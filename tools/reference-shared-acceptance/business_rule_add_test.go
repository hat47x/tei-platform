package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func logicalRuleCandidate(revision string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-orderline-explicit-key-and-fd",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "studio", Actor: "reviewer-test"},
		Intent:            "明示されたData内に論理候補キーと関数従属性の宣言を追加する",
		Evidence: []Evidence{
			{Source: orderFile, Note: "対象Dataと属性の現在のSource"},
		},
		Delta: []Operation{
			{Op: opAddCandidateKey, AddCandidateKey: &AddCandidateKeySpec{
				Data: "sales.order_line", ID: "sales.order_line.business_line", Symbol: "businessLine",
				Attributes: []string{"sales.order_line.order_line_id", "sales.order_line.line_number"},
			}},
			{Op: opAddFunctionalDependency, AddFunctionalDependency: &AddFunctionalDependencySpec{
				Data: "sales.order_line", ID: "sales.order_line.item_by_line", Symbol: "itemByLine",
				Determinants: []string{"sales.order_line.order_line_id", "sales.order_line.line_number"},
				Dependents: []string{"sales.order_line.item_code"},
			}},
		},
	}
}

func TestExplicitLogicalRulePreviewAndAcceptUseSharedBoundary(t *testing.T) {
	dir := scratchWorkspace(t)
	baseline := treeDigest(t, dir)
	candidate := logicalRuleCandidate(revisionOfDir(t, dir))
	raw := mustJSON(t, candidate)

	ev, err := evaluate(dir, raw)
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("typed rule preview not acceptable: %+v", allDiagnostics(p))
	}
	if got, want := keysOf(p.SemanticDiff), []string{
		"added candidate-key:sales.order_line.business_line",
		"added functional-dependency:sales.order_line.item_by_line",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic diff %v want %v", got, want)
	}
	if got, want := p.ChangedCollections, []string{"semantics"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected collections %v", got)
	}
	if len(p.SourceEdits) != 1 || p.SourceEdits[0].Created || p.SourceEdits[0].Path != orderFile ||
		!strings.Contains(p.SourceEdits[0].InsertedText, "candidate-key businessLine {") ||
		!strings.Contains(p.SourceEdits[0].InsertedText, "functional-dependency itemByLine {") {
		t.Fatalf("expected only one Data source edit, got %+v", p.SourceEdits)
	}
	for _, name := range []string{"persistenceMappings", "databases", "migrations", "applications", "environments"} {
		if !hasCode(p.UnchangedCollections, name) {
			t.Fatalf("physical/application collection %s unexpectedly changed", name)
		}
	}
	edges := make(map[string]bool)
	for _, edge := range p.Impact {
		edges[edge.From+" -> "+edge.To+" via "+edge.Via] = true
	}
	for _, want := range []string{
		"candidate-key:sales.order_line.business_line -> data:sales.order_line via semantic.data.candidate-key",
		"candidate-key:sales.order_line.business_line -> attribute:sales.order_line.line_number via semantic.candidate-key.field",
		"functional-dependency:sales.order_line.item_by_line -> attribute:sales.order_line.item_code via semantic.functional-dependency.dependent",
	} {
		if !edges[want] {
			t.Fatalf("missing exact logical-rule impact edge: %s", want)
		}
	}
	foundUnverified := false
	for _, unresolved := range p.Unresolved {
		if unresolved.Code == "LOGICAL_BUSINESS_RULE_UNVERIFIED" {
			foundUnverified = true
		}
	}
	if !foundUnverified {
		t.Fatalf("a logical rule was mistaken for verified business truth: %+v", p.Unresolved)
	}
	assertTreeUnchanged(t, baseline, treeDigest(t, dir))

	decision, err := accept(dir, raw, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accept=%+v err=%v", decision, err)
	}
	after := treeDigest(t, dir)
	var changed []string
	for name, digest := range baseline {
		if after[name] != digest {
			changed = append(changed, name)
		}
	}
	if !reflect.DeepEqual(changed, []string{orderFile}) {
		t.Fatalf("source changes %v want only %s", changed, orderFile)
	}
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if data.ID != "sales.order_line" {
				continue
			}
			if len(data.CandidateKeys) != 1 || len(data.FunctionalDependencies) != 1 ||
				data.CandidateKeys[0].ID != "sales.order_line.business_line" ||
				data.FunctionalDependencies[0].ID != "sales.order_line.item_by_line" {
				t.Fatalf("accepted rules not loaded: %+v", data)
			}
		}
	}
	idx, err := workspace.BuildSourceIndex(dir, loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ kind, id, symbol string }{
		{"semantic-candidate-key", "sales.order_line.business_line", "businessLine"},
		{"semantic-functional-dependency", "sales.order_line.item_by_line", "itemByLine"},
	} {
		refs := idx.ReferencesFor(sourceindex.Subject{Kind: tc.kind, ID: tc.id})
		if len(refs) != 1 || refs[0].ArtifactPath != orderFile ||
			refs[0].Role != sourceindex.RoleDefinition {
			t.Fatalf("accepted source identity not exactly indexed: %+v", refs)
		}
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
		if err != nil {
			t.Fatal(err)
		}
		rng := refs[0].Range
		if string(content[rng.Start.Offset:rng.End.Offset]) != tc.symbol {
			t.Fatalf("source anchor did not name business rule %s: %+v", tc.id, rng)
		}
	}
}

func TestBusinessRulesDoNotModifySourceOnInvalidOrStaleCandidate(t *testing.T) {
	cases := []struct {
		name, stage, code string
		edit func(*Candidate)
	}{
		{"unknown Data", "resolve", "DATA_NOT_FOUND", func(c *Candidate) {
			c.Delta[0].AddCandidateKey.Data = "sales.missing"
			c.Delta[0].AddCandidateKey.ID = "sales.missing.business_line"
			c.Delta[0].AddCandidateKey.Attributes = []string{"sales.missing.field"}
		}},
		{"unknown attribute", "resolve", "ATTRIBUTE_NOT_FOUND", func(c *Candidate) {
			c.Delta[0].AddCandidateKey.Attributes = []string{"sales.order_line.missing"}
		}},
		{"cross-rule duplicated ID", "resolve", "BUSINESS_RULE_ID_ALREADY_EXISTS", func(c *Candidate) {
			c.Delta[1].AddFunctionalDependency.ID = "sales.order_line.business_line"
			c.Delta[1].AddFunctionalDependency.Symbol = "businessLine"
		}},
		{"repeated attribute", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].AddCandidateKey.Attributes = []string{"sales.order_line.line_number", "sales.order_line.line_number"}
		}},
		{"cross Data attribute", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].AddCandidateKey.Attributes = []string{"sales.order.order_id"}
		}},
		{"missing determinants", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[1].AddFunctionalDependency.Determinants = nil
		}},
		{"wrong symbol ID", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].AddCandidateKey.ID = "sales.order_line.some_other_name"
		}},
		{"forbidden placement", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].Placement = orderFile
		}},
		{"unrelated op payload", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].AddRelation = &AddRelationSpec{}
		}},
		{"stale model revision", "base-revision", "STALE_BASE_MODEL_REVISION", func(c *Candidate) {
			c.BaseModelRevision = "sha256:" + strings.Repeat("0", 64)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			before := treeDigest(t, dir)
			c := logicalRuleCandidate(revisionOfDir(t, dir))
			tc.edit(&c)
			data := mustJSON(t, c)
			ev, err := evaluate(dir, data)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), tc.code) {
				t.Fatalf("wrong failure for %s: %+v", tc.name, ev.preview)
			}
			decision, err := accept(dir, data, acceptOptions{})
			if err != nil || decision.Decision != "refused" || decision.SourceChanged {
				t.Fatalf("invalid rule changed authoritative source: %+v err=%v", decision, err)
			}
			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestBusinessRuleInsertionHonorsNestedBracesAndPreservesSourceBytes(t *testing.T) {
	original := []byte("namespace sales\r\n\r\ndata OrderLine {\r\n    attribute itemCode type std.code\r\n    candidate-key existingKey {\r\n        field sales.order_line.item_code\r\n    }\r\n}\r\n")
	op := Operation{Op: opAddFunctionalDependency, AddFunctionalDependency: &AddFunctionalDependencySpec{
		Data: "sales.order_line", ID: "sales.order_line.existing_fd", Symbol: "existingFd",
		Determinants: []string{"sales.order_line.item_code"}, Dependents: []string{"sales.order_line.item_code"},
	}}
	after, inserted, err := insertBusinessRuleIntoDataBlock(original, "OrderLine", op)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(inserted, "    functional-dependency existingFd {\r\n") ||
		!strings.Contains(string(after), "    }\r\n"+inserted+"}\r\n") ||
		strings.Replace(string(after), inserted, "", 1) != string(original) {
		t.Fatalf("nested source boundary or CRLF lost: %q", after)
	}
	if _, _, err := insertBusinessRuleIntoDataBlock([]byte("data OrderLine { attribute x type std.text }"), "OrderLine", op); err == nil {
		t.Fatal("inline outer brace should fail closed")
	}
}

func TestBusinessRuleSourceIsNotGeneratedOnRejectedCandidate(t *testing.T) {
	dir := scratchWorkspace(t)
	base := treeDigest(t, dir)
	c := logicalRuleCandidate(revisionOfDir(t, dir))
	data := mustJSON(t, c)
	d, err := reject(dir, data, "事実検証待ち")
	if err != nil || d.Decision != "rejected" || d.SourceChanged {
		t.Fatalf("rejection: %+v err=%v", d, err)
	}
	assertTreeUnchanged(t, base, treeDigest(t, dir))
	if _, err := os.Stat(filepath.Join(dir, ".tei", "decisions")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
