package validationgen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

	code, err := validationgen.GeneratePositive(rule, attributeType, "main")
	if err != nil {
		t.Fatalf("lower selected Rule: %v", err)
	}
	if !bytes.Contains(code, []byte("value.Sign() > 0")) ||
		!bytes.Contains(code, []byte("source.Number(")) ||
		bytes.Contains(code, []byte("validationeval.Evaluate(")) {
		t.Fatal("Go code does not contain the Rule's actual decision and read logic")
	}

	dir, err := os.MkdirTemp(repoRoot, "validationgen_")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	write := func(name, source string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("generated.go", string(code))
	write("main.go", strings.Join([]string{
		"package main",
		"import (\"errors\"; \"fmt\"; \"math/big\"; \"tei/internal/core\")",
		"type numberSource struct { n *big.Rat; missing bool; reads int }",
		"func (s *numberSource) Number(id core.CanonicalID) (*big.Rat, error) {",
		" if id != \"sales.order_line.quantity\" { return nil, errors.New(\"wrong attribute\") }",
		" s.reads++",
		" if s.missing { return nil, errors.New(\"missing input\") }",
		" return s.n, nil",
		"}",
		"func (s *numberSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {",
		" return nil, errors.New(\"unexpected relation read\")",
		"}",
		"func main() {",
		" for _, tc := range []struct { name, number string; missing bool }{",
		"  {\"positive\", \"2.5\", false}, {\"zero\", \"0\", false},",
		"  {\"negative\", \"-1\", false}, {\"null\", \"\", false},",
		"  {\"missing\", \"\", true},",
		" } {",
		"  var n *big.Rat",
		"  if tc.number != \"\" { n = new(big.Rat); if _, ok := n.SetString(tc.number); !ok { panic(tc.number) } }",
		"  source := &numberSource{n: n, missing: tc.missing}",
		"  result, err := EvaluateSelected(source)",
		"  if err != nil { fmt.Printf(\"%s:error:%d\\n\", tc.name, source.reads) } else {",
		"   fmt.Printf(\"%s:%s:%d\\n\", tc.name, result, source.reads)",
		"  }",
		" }",
		"}",
	}, "\n")+"\n")
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated Go did not compile/run: %v\n%s", err, out)
	}
	const want = "positive:satisfied:1\nzero:not-satisfied:1\nnegative:not-satisfied:1\nnull:error:1\nmissing:error:1\n"
	if string(out) != want {
		t.Fatalf("wrong generated decision/read footprint:\n%s\nwant:\n%s", out, want)
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
	a, err := validationgen.GeneratePositive(good, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	good.Attribute = "sales.order_line.quantity_override"
	b, err := validationgen.GeneratePositive(good, "std.decimal", "main")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) || !bytes.Contains(b, []byte("quantity_override")) {
		t.Fatal("semantic Attribute change did not change generated Go's read target")
	}
}
