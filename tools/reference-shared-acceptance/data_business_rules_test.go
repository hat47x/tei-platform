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

const packageLinePath = "semantic/fulfillment/package_line.tei"

func composedDataBusinessRuleCandidate(revision string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "candidate-fulfillment-package-line-business-identity",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "studio", Actor: "test"},
		Intent:            "新しいDataと属性、複合候補キー、関数従属性を一回の受理で宣言する",
		Evidence:          []Evidence{{Source: orderFile}},
		Delta: []Operation{{
			Op: opAddData, Placement: packageLinePath,
			Data: &DataSpec{
				Namespace: "fulfillment", ID: "fulfillment.package_line", Symbol: "PackageLine",
				Attributes: []AttributeSpec{
					{ID: "fulfillment.package_line.order_ref", Symbol: "orderRef", Type: "std.identifier"},
					{ID: "fulfillment.package_line.line_no", Symbol: "lineNo", Type: "std.integer"},
					{ID: "fulfillment.package_line.label", Symbol: "label", Type: "std.text"},
				},
				CandidateKeys: []AddCandidateKeySpec{{
					Data: "fulfillment.package_line", ID: "fulfillment.package_line.business_line", Symbol: "businessLine",
					Attributes: []string{"fulfillment.package_line.order_ref", "fulfillment.package_line.line_no"},
				}},
				FunctionalDependencies: []AddFunctionalDependencySpec{{
					Data: "fulfillment.package_line", ID: "fulfillment.package_line.label_by_line", Symbol: "labelByLine",
					Determinants: []string{"fulfillment.package_line.order_ref", "fulfillment.package_line.line_no"},
					Dependents:   []string{"fulfillment.package_line.label"},
				}},
			},
		}},
	}
}

func TestAddDataWithCompositeKeyAndFDUsesOneSharedAcceptance(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	c := composedDataBusinessRuleCandidate(revisionOfDir(t, dir))
	raw := mustJSON(t, c)
	ev, err := evaluate(dir, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("typed Data composition was rejected: %+v", allDiagnostics(ev.preview))
	}
	want := []string{
		"added attribute:fulfillment.package_line.label",
		"added attribute:fulfillment.package_line.line_no",
		"added attribute:fulfillment.package_line.order_ref",
		"added candidate-key:fulfillment.package_line.business_line",
		"added data:fulfillment.package_line",
		"added functional-dependency:fulfillment.package_line.label_by_line",
		"added namespace:fulfillment",
	}
	if got := keysOf(ev.preview.SemanticDiff); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected canonical Data+rule diff: have=%v want=%v", got, want)
	}
	if len(ev.preview.SourceEdits) != 1 || !ev.preview.SourceEdits[0].Created ||
		ev.preview.SourceEdits[0].Path != packageLinePath {
		t.Fatalf("expected one new semantic file: %+v", ev.preview.SourceEdits)
	}
	text := ev.preview.SourceEdits[0].InsertedText
	for _, declaration := range []string{
		"attribute orderRef type std.identifier",
		"candidate-key businessLine {",
		"field fulfillment.package_line.order_ref",
		"functional-dependency labelByLine {",
		"determinant fulfillment.package_line.line_no",
		"dependent fulfillment.package_line.label",
	} {
		if !strings.Contains(text, declaration) {
			t.Fatalf("missing %q in generated Data source %q", declaration, text)
		}
	}
	for _, name := range []string{"persistenceMappings", "databases", "migrations", "applications", "environments"} {
		if !hasCode(ev.preview.UnchangedCollections, name) {
			t.Fatalf("non-semantic World collection %s changed", name)
		}
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
	decision, err := accept(dir, raw, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accept=%+v err=%v", decision, err)
	}
	after := treeDigest(t, dir)
	for path, sha := range before {
		if after[path] != sha {
			t.Fatalf("unrelated source file %s changed", path)
		}
	}
	if len(after) != len(before)+1 || after[packageLinePath] == "" {
		t.Fatalf("unexpected accepted files: %d -> %d", len(before), len(after))
	}
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ns := range loaded.World.Semantics {
		for _, data := range ns.Data {
			if data.ID != "fulfillment.package_line" {
				continue
			}
			found = len(data.Attributes) == 3 && len(data.CandidateKeys) == 1 &&
				len(data.FunctionalDependencies) == 1 &&
				data.CandidateKeys[0].ID == "fulfillment.package_line.business_line" &&
				data.FunctionalDependencies[0].ID == "fulfillment.package_line.label_by_line"
		}
	}
	if !found {
		t.Fatalf("the new Data and business assertions were not loaded: %+v", loaded.World.Semantics)
	}
	idx, err := workspace.BuildSourceIndex(dir, loaded)
	if err != nil {
		t.Fatal(err)
	}
	sourceBytes, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(packageLinePath)))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ kind, id, symbol string }{
		{"semantic-candidate-key", "fulfillment.package_line.business_line", "businessLine"},
		{"semantic-functional-dependency", "fulfillment.package_line.label_by_line", "labelByLine"},
	} {
		refs := idx.ReferencesFor(sourceindex.Subject{Kind: tc.kind, ID: tc.id})
		if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition || refs[0].ArtifactPath != packageLinePath {
			t.Fatalf("accepted canonical source anchor missing: %+v", refs)
		}
		rng := refs[0].Range
		if string(sourceBytes[rng.Start.Offset:rng.End.Offset]) != tc.symbol {
			t.Fatalf("wrong canonical token range for %s: %+v", tc.id, rng)
		}
	}
}

func TestAddDataBusinessRuleRejectsUnrelatedOrUndeclaredAttributes(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*DataSpec)
	}{
		{"field not in new Data", func(d *DataSpec) { d.CandidateKeys[0].Attributes[0] = "fulfillment.package_line.unknown" }},
		{"FD dependent not in new Data", func(d *DataSpec) {
			d.FunctionalDependencies[0].Dependents = []string{"fulfillment.package_line.unknown"}
		}},
		{"another Data owns rule", func(d *DataSpec) {
			d.CandidateKeys[0].Data = "fulfillment.other"
			d.CandidateKeys[0].ID = "fulfillment.other.business_line"
			d.CandidateKeys[0].Attributes = []string{"fulfillment.other.order_ref"}
		}},
		{"key duplicates FD identity", func(d *DataSpec) {
			d.FunctionalDependencies[0].ID = d.CandidateKeys[0].ID
			d.FunctionalDependencies[0].Symbol = d.CandidateKeys[0].Symbol
		}},
		{"key duplicates attribute identity", func(d *DataSpec) {
			d.CandidateKeys[0].ID = "fulfillment.package_line.order_ref"
			d.CandidateKeys[0].Symbol = "orderRef"
		}},
		{"repeated composite key field", func(d *DataSpec) {
			d.CandidateKeys[0].Attributes = []string{"fulfillment.package_line.line_no", "fulfillment.package_line.line_no"}
		}},
		{"empty dependent group", func(d *DataSpec) { d.FunctionalDependencies[0].Dependents = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			before := treeDigest(t, dir)
			c := composedDataBusinessRuleCandidate(revisionOfDir(t, dir))
			tc.mutate(c.Delta[0].Data)
			raw := mustJSON(t, c)
			ev, err := evaluate(dir, raw)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable || !hasCode(diagnosticCodes(ev.preview), "CANDIDATE_NORMALIZE") {
				t.Fatalf("%s was not refused in normalize: %+v", tc.name, ev.preview)
			}
			decision, err := accept(dir, raw, acceptOptions{})
			if err != nil || decision.Decision != "refused" || decision.SourceChanged {
				t.Fatalf("invalid Data+rule candidate changed source: %+v err=%v", decision, err)
			}
			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestAddDataWithoutBusinessRulesRetainsOldSourceAndMeaning(t *testing.T) {
	data := DataSpec{
		Namespace: "simple", ID: "simple.entity", Symbol: "Entity",
		Attributes: []AttributeSpec{{ID: "simple.entity.name", Symbol: "name", Type: "std.text"}},
	}
	got := renderData(data)
	if got != "namespace simple\n\ndata Entity {\n    attribute name type std.text\n}" {
		t.Fatalf("legacy add-data source changed: %q", got)
	}
	if len(data.CandidateKeys) != 0 || len(data.FunctionalDependencies) != 0 {
		t.Fatal("schema-only Data received invented business keys or FDs")
	}
}
