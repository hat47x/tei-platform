package validationgen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// The real product Workspace selects the Markdown reader and resolves the
// canonical attribute type. Only then do we lower the selected Rule to Go.
// Running the compiled Go tests its decision logic, not just the source text.
func TestSelectedMarkdownPositiveRuleGeneratesExecutableGo(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test file location is unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixture := filepath.Join(repoRoot, "internal", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatalf("actual product Workspace.Load: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
		t.Fatalf("unexpected selected product language: %+v", loaded.AuthoringSources)
	}

	const ruleID = "sales.order_line.quantity_positive"
	const attributeID = "sales.order_line.quantity"
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	var rule semantic.Validation
	var attributeType core.CanonicalID
	for _, ns := range loaded.World.Semantics {
		for _, candidate := range ns.Validations {
			if string(candidate.ID) == ruleID {
				if rule.ID != "" {
					t.Fatal("same canonical Rule is declared more than once")
				}
				rule = candidate
			}
		}
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				if string(attribute.ID) == attributeID {
					attributeType = attribute.TypeRef
				}
			}
		}
	}
	if rule.ID != ruleID || rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != attributeID || attributeType != "std.decimal" {
		t.Fatalf("selected Rule or resolved type changed: %+v, type=%s", rule, attributeType)
	}
	index, err := workspace.BuildSourceIndexFromLoad(fixture, loaded)
	if err != nil {
		t.Fatalf("same-load Source Index: %v", err)
	}
	original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
	if len(refs) != 1 || refs[0].ArtifactPath != sourcePath ||
		refs[0].Role != sourceindex.RoleDefinition {
		t.Fatalf("Rule is not bound to one selected Source: %+v", refs)
	}
	start, end := refs[0].Range.Start.Offset, refs[0].Range.End.Offset
	if start < 0 || end > len(original) || start >= end ||
		string(original[start:end]) != ruleID {
		t.Fatalf("Rule Source span does not identify selected rule: %+v", refs[0])
	}

	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
	if err := out.Err(); err != nil {
		t.Fatalf("lower selected Rule: %v", err)
	}
	code := out.File("quantity_is_positive.go")
	if !bytes.Contains(code, []byte("return quantity.Sign() > 0")) {
		t.Fatalf("Go code does not contain the Rule's actual decision:\n%s", code)
	}
	// The binding is the Trace, checked against the declared check function.
	if binding := goemittest.SingleBinding(t, out); binding != (goemittest.Binding{
		Check: "CheckQuantityIsPositive", Parameter: "quantity", Attribute: attributeID,
	}) {
		t.Fatalf("generated Go is not bound to the selected Attribute: %+v", binding)
	}
	if imports, err := goemittest.TEIImports(goemittest.Files(out)); err != nil || len(imports) != 0 {
		t.Fatalf("generated Go depends on TEI: %v %v", imports, err)
	}

	// The harness reads the attribute the Trace binds to the check, once
	// per case, and passes the value to the generated check function. A null
	// value and a value that was not fetched both reach the check as nil and
	// are undecided, as validationeval.Decide answers for them; the old
	// Source contract reported both as errors.
	const attr = "sales.order_line.quantity"
	got := goemittest.Run(t, out, []goemittest.Case{
		{Name: "positive", Row: map[string]goemittest.Cell{attr: {Text: "2.5"}}},
		{Name: "zero", Row: map[string]goemittest.Cell{attr: {Text: "0"}}},
		{Name: "negative", Row: map[string]goemittest.Cell{attr: {Text: "-1"}}},
		{Name: "null", Row: map[string]goemittest.Cell{attr: {Null: true}}},
		{Name: "missing", Row: map[string]goemittest.Cell{}},
	})
	const want = "positive:satisfied:1:" + attr + "\nzero:not-satisfied:1:" + attr +
		"\nnegative:not-satisfied:1:" + attr + "\nnull:undecided:1:" + attr + "\nmissing:undecided:1:" + attr + "\n"
	if got != want {
		t.Fatalf("wrong generated decision/read footprint:\n%s\nwant:\n%s", got, want)
	}
}

func TestPositiveGoLoweringRejectsUnsupportedOrUnresolvedMeaning(t *testing.T) {
	good := semantic.Validation{
		ID: "sales.order_line.quantity_positive", Kind: semantic.ValidationPositive,
		Attribute: "sales.order_line.quantity",
	}
	tests := []struct {
		name string
		rule semantic.Validation
		kind core.CanonicalID
		pkg  string
	}{
		{"unsupported-kind", semantic.Validation{ID: "x", Kind: semantic.ValidationRelationUnique, Attribute: "x", Relation: "y"}, "std.decimal", "main"},
		{"unknown-type", good, "", "main"},
		{"money-is-not-decimal", good, "std.money", "main"},
		{"missing-attribute", semantic.Validation{ID: "x", Kind: semantic.ValidationPositive}, "std.decimal", "main"},
		{"unexpected-relation", semantic.Validation{ID: "x", Kind: semantic.ValidationPositive, Attribute: "x", Relation: "y"}, "std.decimal", "main"},
		{"invalid-package", good, "std.decimal", "not-a-package"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if generated, err := validationgen.GeneratePositive(tc.rule, tc.kind, tc.pkg); err == nil {
				t.Fatalf("unsupported lowering was accepted: %s", generated)
			}
		})
	}
	a, _, err := tracedPositive(good, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	good.Attribute = "sales.order_line.quantity_override"
	b, trace, err := tracedPositive(good, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := observeGeneratedPositiveGo(b, trace); err != nil || bytes.Equal(a, b) ||
		trace.Attribute != good.Attribute || trace.Parameter != "quantityOverride" ||
		!bytes.Contains(b, []byte("func CheckQuantityOverrideIsPositive(quantityOverride *big.Rat) Result")) {
		t.Fatalf("semantic Attribute change did not change generated Go's read target: %+v %v", trace, err)
	}
}
