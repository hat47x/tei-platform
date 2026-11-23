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

// A selected Markdown Rule must determine generated Go. Updating its actual
// predicate (not a hand-built semantic.Validation) must change the generated
// read target and the decision of an independent Go process.
func TestPositiveGoLoweringTracksSelectedMarkdownSourceEdit(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixture := filepath.Join(repoRoot, "internal", "workspace", "testdata", "tei-md-rule")
	root := t.TempDir()

	const nativePath = "semantic/sales/order.tei"
	const rulePath = "semantic/sales/order-line-rules.tei.md"
	for _, name := range []string{"tei.workspace.yaml", nativePath, rulePath} {
		original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, original, 0600); err != nil {
			t.Fatal(err)
		}
	}

	// The existing fixture's line_number has type std.integer, which this
	// deliberately narrow lowering must reject. Add a second std.decimal
	// attribute only to the isolated test workspace; never alter the fixture.
	nativeFile := filepath.Join(root, filepath.FromSlash(nativePath))
	native, err := os.ReadFile(nativeFile)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    attribute quantity type std.decimal"
	if strings.Count(string(native), anchor) != 1 {
		t.Fatal("expected one quantity declaration in product fixture")
	}
	native = []byte(strings.Replace(string(native), anchor,
		anchor+"\n    attribute bonusQuantity type std.decimal", 1))
	if err := os.WriteFile(nativeFile, native, 0600); err != nil {
		t.Fatal(err)
	}

	const ruleID = "sales.order_line.quantity_positive"
	const firstAttribute = "sales.order_line.quantity"
	const secondAttribute = "sales.order_line.bonus_quantity"
	subject := sourceindex.Subject{Kind: "semantic-validation", ID: ruleID}

	loadAndLower := func() (goemit.Output, sourceindex.Index) {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("selected product Workspace.Load: %v", err)
		}
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
			loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
			t.Fatalf("wrong selected language: %+v", loaded.AuthoringSources)
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatalf("build same-load index: %v", err)
		}
		refs := index.ReferencesFor(subject)
		if len(refs) != 1 || refs[0].ArtifactPath != rulePath ||
			refs[0].Role != sourceindex.RoleDefinition {
			t.Fatalf("selected rule source not uniquely indexed: %+v", refs)
		}
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rulePath)))
		if err != nil {
			t.Fatal(err)
		}
		start, end := refs[0].Range.Start.Offset, refs[0].Range.End.Offset
		if start < 0 || end > len(source) || start >= end ||
			string(source[start:end]) != ruleID {
			t.Fatalf("rule ID range not backed by the selected original bytes: %+v", refs[0])
		}

		var selected semantic.Validation
		var attributeType core.CanonicalID
		for _, ns := range loaded.World.Semantics {
			for _, rule := range ns.Validations {
				if string(rule.ID) == ruleID {
					if selected.ID != "" {
						t.Fatal("ambiguous Rule ID")
					}
					selected = rule
				}
			}
		}
		if selected.ID != ruleID || selected.Kind != semantic.ValidationPositive {
			t.Fatalf("unexpected Rule after actual Load: %+v", selected)
		}
		for _, ns := range loaded.World.Semantics {
			for _, data := range ns.Data {
				for _, attr := range data.Attributes {
					if attr.ID == selected.Attribute {
						if attributeType != "" {
							t.Fatalf("ambiguous Attribute %s", selected.Attribute)
						}
						attributeType = attr.TypeRef
					}
				}
			}
		}
		if attributeType != "std.decimal" {
			t.Fatalf("unresolved or changed type for %s: %s", selected.Attribute, attributeType)
		}
		out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: selected, AttributeType: attributeType}}})
		if err := out.Err(); err != nil {
			t.Fatalf("lower actual selected Rule: %v", err)
		}
		return out, index
	}
	ruleGo := func(out goemit.Output) []byte { return out.File(out.Trace[0].File) }
	binding := func(out goemit.Output) goemittest.Binding { return goemittest.SingleBinding(t, out) }

	// Inputs remain identical across both processes: quantity=0 and bonus=2.
	// Only the Markdown predicate selects which field the generated Go is
	// bound to; the harness reads the field the Trace binds to the check
	// function and reports what it read.
	inputs := []goemittest.Case{{Name: "fixed", Row: map[string]goemittest.Cell{
		firstAttribute:  {Text: "0"},
		secondAttribute: {Text: "2"},
	}}}
	runGenerated := func(out goemit.Output, want string) {
		t.Helper()
		if got := goemittest.Run(t, out, inputs); got != want {
			t.Fatalf("generated Go run output=%q, want=%q", got, want)
		}
	}

	before, oldIndex := loadAndLower()
	if binding(before) != (goemittest.Binding{Check: "CheckQuantityIsPositive", Parameter: "quantity", Attribute: firstAttribute}) {
		t.Fatal("initial generated Go is not bound to the selected quantity Attribute")
	}
	runGenerated(before, "fixed:not-satisfied:1:"+firstAttribute+"\n")

	ruleFile := filepath.Join(root, filepath.FromSlash(rulePath))
	original, err := os.ReadFile(ruleFile)
	if err != nil {
		t.Fatal(err)
	}
	const oldPredicate = "sales.order_line.quantity is positive"
	const newPredicate = "sales.order_line.bonus_quantity is positive"
	if strings.Count(string(original), oldPredicate) != 1 {
		t.Fatal("fixture has no unique original quantity predicate")
	}
	edited := []byte(strings.Replace(string(original), oldPredicate, newPredicate, 1))
	if err := os.WriteFile(ruleFile, edited, 0600); err != nil {
		t.Fatal(err)
	}
	if err := oldIndex.Validate(func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	}); err == nil {
		t.Fatal("old Source Index remained valid after actual Markdown edit")
	}

	after, newIndex := loadAndLower()
	if bytes.Equal(ruleGo(before), ruleGo(after)) ||
		binding(after) != (goemittest.Binding{Check: "CheckBonusQuantityIsPositive", Parameter: "bonusQuantity", Attribute: secondAttribute}) ||
		bytes.Contains(ruleGo(after), []byte("func CheckQuantityIsPositive(")) {
		t.Fatal("edited selected Rule did not change the generated Go binding")
	}
	if err := newIndex.Validate(func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	}); err != nil {
		t.Fatalf("fresh Load Index is invalid: %v", err)
	}
	runGenerated(after, "fixed:satisfied:1:"+secondAttribute+"\n")
}
