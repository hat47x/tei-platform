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

func authoredRuleRetractionFixture(t *testing.T) (string, []byte, []byte) {
	t.Helper()
	root := scratchWorkspace(t)
	before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	adding := logicalRuleCandidate(revisionOfDir(t, root))
	decision, err := accept(root, mustJSON(t, adding), acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("could not establish existing authored keys / FD: %+v %v", decision, err)
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("fixture did not create a canonical source rule")
	}
	return root, before, after
}

func retractionCandidate(root string, source []byte) Candidate {
	fp := sourceindex.Fingerprint(source)
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-retract-two-business-rules",
		BaseModelRevision: revisionOfDirMust(root),
		Origin:            Origin{Kind: "studio", Actor: "reviewer-test"},
		Intent:            "既存の論理規則を明示的に撤回し、旧参照への影響を確認する",
		Evidence:          []Evidence{{Source: orderFile, SourceFingerprint: fp}},
		Delta: []Operation{
			{Op: opRemoveCandidateKey, RemoveBusinessRule: &RemoveBusinessRuleSpec{
				Data: "sales.order_line", ID: "sales.order_line.business_line",
				Symbol: "businessLine", SourceFingerprint: fp,
			}},
			{Op: opRemoveFunctionalDependency, RemoveBusinessRule: &RemoveBusinessRuleSpec{
				Data: "sales.order_line", ID: "sales.order_line.item_by_line",
				Symbol: "itemByLine", SourceFingerprint: fp,
			}},
		},
	}
}

func revisionOfDirMust(root string) string {
	s, err := loadSnapshot(root)
	if err != nil {
		panic(err)
	}
	return s.revision
}

func TestTypedRuleRetractionReportsHistoricalImpactAndDeletesOnlyOwnBlocks(t *testing.T) {
	root, original, authored := authoredRuleRetractionFixture(t)
	before := treeDigest(t, root)
	c := retractionCandidate(root, authored)
	raw := mustJSON(t, c)
	ev, err := evaluate(root, raw)
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("retraction preview refused: %+v", allDiagnostics(p))
	}
	if got, want := keysOf(p.SemanticDiff), []string{
		"removed candidate-key:sales.order_line.business_line",
		"removed functional-dependency:sales.order_line.item_by_line",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong exact removal delta: got %v want %v", got, want)
	}
	if len(p.SourceEdits) != 1 || p.SourceEdits[0].Path != orderFile || p.SourceEdits[0].InsertedText != "" ||
		!strings.Contains(p.SourceEdits[0].RemovedText, "candidate-key businessLine {") ||
		!strings.Contains(p.SourceEdits[0].RemovedText, "functional-dependency itemByLine {") {
		t.Fatalf("source retraction was not an exact deletion: %+v", p.SourceEdits)
	}
	var prior, warning bool
	for _, edge := range p.Impact {
		if edge.From == "candidate-key:sales.order_line.business_line" && edge.To == "attribute:sales.order_line.line_number" &&
			edge.Via == "previous.semantic.candidate-key.field" {
			prior = true
		}
	}
	for _, u := range p.Unresolved {
		if u.Code == "LOGICAL_RULE_RETRACTION_REQUIRES_REVIEW" {
			warning = true
		}
	}
	if !prior || !warning {
		t.Fatalf("retracted business assertion was treated as no impact: edges=%+v unresolved=%+v", p.Impact, p.Unresolved)
	}
	assertTreeUnchanged(t, before, treeDigest(t, root))
	d, err := accept(root, raw, acceptOptions{})
	if err != nil || d.Decision != "accepted" || !d.SourceChanged {
		t.Fatalf("reviewed source retraction failed: %+v %v", d, err)
	}
	final, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(final, original) {
		t.Fatalf("retraction modified unrelated source bytes\n--- before ---\n%s\n--- after ---\n%s", original, final)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if data.ID == "sales.order_line" && (len(data.CandidateKeys) != 0 || len(data.FunctionalDependencies) != 0) {
				t.Fatalf("removed rules remained in canonical Data: %+v", data)
			}
		}
	}
}

func TestTypedRuleRetractionRejectsAbsentStaleOrForgedTargets(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Candidate)
	}{
		{"stale source", func(c *Candidate) { c.Delta[0].RemoveBusinessRule.SourceFingerprint = "sha256:" + strings.Repeat("0", 64) }},
		{"missing rule", func(c *Candidate) {
			c.Delta[0].RemoveBusinessRule.ID = "sales.order_line.unseen"
			c.Delta[0].RemoveBusinessRule.Symbol = "unseen"
		}},
		{"wrong kind", func(c *Candidate) {
			c.Delta[0].Op = opRemoveFunctionalDependency
			c.Delta[1].Op = opRemoveCandidateKey
		}},
		{"wrong symbol", func(c *Candidate) { c.Delta[0].RemoveBusinessRule.Symbol = "wrong" }},
		{"foreign Data", func(c *Candidate) {
			c.Delta[0].RemoveBusinessRule.Data = "sales.other"
			c.Delta[0].RemoveBusinessRule.ID = "sales.other.business_line"
		}},
		{"bad fingerprint", func(c *Candidate) { c.Delta[0].RemoveBusinessRule.SourceFingerprint = "unguarded" }},
		{"forged placement", func(c *Candidate) { c.Delta[0].Placement = "../danger" }},
		{"extra payload", func(c *Candidate) { c.Delta[0].AddRelation = &AddRelationSpec{} }},
		{"duplicate retraction", func(c *Candidate) { c.Delta[1] = c.Delta[0] }},
		{"stale revision", func(c *Candidate) { c.BaseModelRevision = "sha256:" + strings.Repeat("f", 64) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, _, authored := authoredRuleRetractionFixture(t)
			baseline := treeDigest(t, root)
			c := retractionCandidate(root, authored)
			tc.edit(&c)
			raw := mustJSON(t, c)
			ev, err := evaluate(root, raw)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable {
				t.Fatalf("%s unexpectedly accepted: %+v", tc.name, ev.preview)
			}
			d, err := accept(root, raw, acceptOptions{})
			if err != nil || d.Decision != "refused" || d.SourceChanged {
				t.Fatalf("%s mutated a protected source: %+v %v", tc.name, d, err)
			}
			assertTreeUnchanged(t, baseline, treeDigest(t, root))
		})
	}
}

func TestRemoveRuleSourceRejectsUnsafeOrAmbiguousPresentation(t *testing.T) {
	op := Operation{Op: opRemoveCandidateKey, RemoveBusinessRule: &RemoveBusinessRuleSpec{
		Data: "sales.order_line", ID: "sales.order_line.business_line", Symbol: "businessLine",
	}}
	valid := []byte("namespace sales\r\ndata OrderLine {\r\n" +
		"    attribute itemCode type std.code\r\n" +
		"    candidate-key businessLine {\r\n" +
		"        field sales.order_line.item_code\r\n" +
		"    }\r\n}\r\n")
	after, removed, err := removeExactBusinessRuleBlock(valid, op)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(removed, "candidate-key businessLine") || strings.Contains(string(after), "candidate-key") ||
		!strings.Contains(string(after), "    attribute itemCode type std.code\r\n}\r\n") {
		t.Fatalf("CRLF or unrelated Data source lost: %q", after)
	}
	for _, input := range [][]byte{
		[]byte(strings.Replace(string(valid), "    candidate-key businessLine {", "  attribute itemCode type std.code candidate-key businessLine {", 1)),
		[]byte(strings.Replace(string(valid), "    }\r\n}\r\n", "    } // keep-this\r\n}\r\n", 1)),
		[]byte(strings.Replace(string(valid), "businessLine", "different", 1)),
	} {
		if _, _, err := removeExactBusinessRuleBlock(input, op); err == nil {
			t.Fatalf("ambiguous or mismatched layout was removed: %q", input)
		}
	}
}
