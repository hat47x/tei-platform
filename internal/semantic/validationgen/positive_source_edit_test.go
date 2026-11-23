package validationgen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is an R1 source-to-generated-decision regression. It does not involve
// a product Builder, an actual Provider, an accepted Plugin or a CLI consumer.
func TestSelectedMarkdownRuleEditRebuildsActualGoDecision(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test path unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	fixture := filepath.Join(repoRoot, "internal", "workspace", "testdata", "tei-md-rule")
	root := t.TempDir()
	const rulePath = "semantic/sales/order-line-rules.tei.md"
	const typePath = "semantic/sales/order.tei"
	for _, path := range []string{"tei.workspace.yaml", rulePath, typePath} {
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if path == typePath {
			const old = "attribute quantity type std.decimal"
			if strings.Count(string(raw), old) != 1 {
				t.Fatal("fixture no longer has the expected decimal quantity attribute")
			}
			raw = []byte(strings.Replace(string(raw), old,
				old+"\n    attribute discount type std.decimal", 1))
		}
		target := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Every stage uses the selected product Language and a fresh Load-time
	// Source Index, not a reference-only Parser or a cached canonical Rule.
	project := func() (goemit.Output, sourceindex.Index, core.CanonicalID) {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("load selected rule source: %v", err)
		}
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" {
			t.Fatalf("unexpected product authoring selection: %+v", loaded.AuthoringSources)
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatalf("index same Load-time bytes: %v", err)
		}
		const ruleID = "sales.order_line.quantity_positive"
		var rule semantic.Validation
		var attributeType core.CanonicalID
		for _, ns := range loaded.World.Semantics {
			for _, candidate := range ns.Validations {
				if candidate.ID == ruleID {
					if rule.ID != "" {
						t.Fatal("duplicate canonical Rule")
					}
					rule = candidate
				}
			}
		}
		for _, ns := range loaded.World.Semantics {
			for _, data := range ns.Data {
				for _, attribute := range data.Attributes {
					if attribute.ID == rule.Attribute {
						attributeType = attribute.TypeRef
					}
				}
			}
		}
		if rule.ID != ruleID || rule.Kind != semantic.ValidationPositive || attributeType != "std.decimal" {
			t.Fatalf("wrong typed Rule after product Load: rule=%+v type=%s", rule, attributeType)
		}
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rulePath)))
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 1 || refs[0].ArtifactPath != rulePath ||
			refs[0].Role != sourceindex.RoleDefinition ||
			refs[0].Range.Validate(raw) != nil {
			t.Fatalf("unbound or stale selected Rule source: %+v", refs)
		}
		span := refs[0].Range
		if string(raw[span.Start.Offset:span.End.Offset]) != ruleID {
			t.Fatalf("Rule ID is not at its authored byte position: %+v", span)
		}
		out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
		if err := out.Err(); err != nil {
			t.Fatalf("lower actual selected Rule: %v", err)
		}
		return out, index, rule.Attribute
	}
	// emitted joins every emitted file with its path, so equality covers the
	// whole generated package, not only the rule file.
	emitted := func(out goemit.Output) []byte {
		var all []byte
		for _, file := range out.Files {
			all = append(all, "== "+file.Path+"\n"...)
			all = append(all, file.Content...)
		}
		return all
	}
	validateStale := func(index sourceindex.Index) {
		t.Helper()
		err := index.Validate(func(path string) ([]byte, error) {
			return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		})
		if err == nil || !strings.Contains(err.Error(), "fingerprint is stale") {
			t.Fatalf("old index was accepted after source bytes changed: %v", err)
		}
	}
	// The harness reads quantity=0 and discount=2 from one fixed record. It
	// reads the attribute the generated doc comment names and reports it.
	record := []goemittest.Case{{Name: "fixed", Row: map[string]goemittest.Cell{
		"sales.order_line.quantity": {Text: "0"},
		"sales.order_line.discount": {Text: "2"},
	}}}
	execute := func(out goemit.Output) string {
		t.Helper()
		return goemittest.Run(t, out, record)
	}

	original, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rulePath)))
	if err != nil {
		t.Fatal(err)
	}
	initialOut, initialIndex, initialAttribute := project()
	initialGo := emitted(initialOut)
	if initialAttribute != "sales.order_line.quantity" {
		t.Fatalf("wrong initial read target: %s", initialAttribute)
	}
	if got := execute(initialOut); got != "fixed:not-satisfied:1:sales.order_line.quantity\n" {
		t.Fatalf("original generated decision/read: %q", got)
	}

	// Prose changes the Source revision, not the selected Rule or generated
	// decision. A stale Index must still fail rather than be silently reused.
	proseOnly := append(append([]byte(nil), original...), []byte("\n補足の文章だけを変更。\n")...)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rulePath)), proseOnly, 0o600); err != nil {
		t.Fatal(err)
	}
	validateStale(initialIndex)
	proseOut, proseIndex, proseAttribute := project()
	proseGo := emitted(proseOut)
	if proseAttribute != initialAttribute || !bytes.Equal(proseGo, initialGo) {
		t.Fatal("prose-only edit unexpectedly changed the selected Rule or generated Go")
	}

	// Only the Rule's read target changes. The Source Model/Index and generated
	// behavior must change together, despite the unchanged canonical Rule ID.
	const oldCondition = "sales.order_line.quantity is positive"
	if strings.Count(string(proseOnly), oldCondition) != 1 {
		t.Fatal("the fixture no longer has exactly one editable positive Rule")
	}
	changed := strings.Replace(string(proseOnly), oldCondition,
		"sales.order_line.discount is positive", 1)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rulePath)), []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	validateStale(proseIndex)
	changedOut, _, changedAttribute := project()
	changedGo := emitted(changedOut)
	if changedAttribute != "sales.order_line.discount" ||
		bytes.Equal(changedGo, initialGo) ||
		!bytes.Contains(changedGo, []byte("func CheckDiscountIsPositive(")) ||
		!bytes.Contains(changedGo, []byte("属性 sales.order_line.discount の値を渡します。")) {
		t.Fatal("authored Rule read target was not reflected in the regenerated Go")
	}
	if got := execute(changedOut); got != "fixed:satisfied:1:sales.order_line.discount\n" {
		t.Fatalf("edited Rule did not change actual generated decision/read: %q", got)
	}
}
