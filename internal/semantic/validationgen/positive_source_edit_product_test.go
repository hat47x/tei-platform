package validationgen_test

import (
	"bytes"
	"fmt"
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

	loadAndLower := func() ([]byte, sourceindex.Index) {
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
		goSource, err := validationgen.GeneratePositive(selected, attributeType, "main")
		if err != nil {
			t.Fatalf("lower actual selected Rule: %v", err)
		}
		return goSource, index
	}

	// Inputs remain identical across both processes: quantity=0 and bonus=2.
	// Only the Markdown predicate selects which field the generated Go reads.
	runGenerated := func(generated []byte, want string) {
		t.Helper()
		dir, err := os.MkdirTemp(repoRoot, "validationgen_edit_")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		harness := strings.Join([]string{
			"package main",
			"import (\"fmt\"; \"math/big\"; \"tei/internal/core\")",
			"type input struct { read core.CanonicalID; count int }",
			"func (s *input) Number(id core.CanonicalID) (*big.Rat, error) {",
			" s.count++; s.read = id",
			" switch id {",
			" case \"sales.order_line.quantity\": return big.NewRat(0, 1), nil",
			" case \"sales.order_line.bonus_quantity\": return big.NewRat(2, 1), nil",
			" default: return nil, fmt.Errorf(\"unexpected read: %s\", id)",
			" }",
			"}",
			"func (s *input) RelationKeys(_, _ core.CanonicalID) ([]string, error) {",
			" return nil, fmt.Errorf(\"unexpected relation read\")",
			"}",
			"func main() {",
			" s := &input{}",
			" result, err := EvaluateSelected(s)",
			" fmt.Printf(\"%s|%s|%d|%v\\n\", result, s.read, s.count, err)",
			"}",
		}, "\n") + "\n"
		for name, data := range map[string][]byte{
			"generated.go": generated, "main.go": []byte(harness),
		} {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		output, err := cmd.CombinedOutput()
		if err != nil || string(output) != want {
			t.Fatalf("generated Go run err=%v, output=%q, want=%q", err, output, want)
		}
	}

	before, oldIndex := loadAndLower()
	if !bytes.Contains(before, []byte("source.Number(\""+firstAttribute+"\")")) {
		t.Fatal("initial generated Go did not read the selected quantity Attribute")
	}
	runGenerated(before, "not-satisfied|"+firstAttribute+"|1|<nil>\n")

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
	if bytes.Equal(before, after) ||
		!bytes.Contains(after, []byte("source.Number(\""+secondAttribute+"\")")) ||
		bytes.Contains(after, []byte("source.Number(\""+firstAttribute+"\")")) {
		t.Fatal("edited selected Rule did not change the generated Go read target")
	}
	if err := newIndex.Validate(func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	}); err != nil {
		t.Fatalf("fresh Load Index is invalid: %v", err)
	}
	runGenerated(after, "satisfied|"+secondAttribute+"|1|<nil>\n")

	if len(before) == 0 || len(after) == 0 {
		t.Fatal(fmt.Errorf("empty generated Go after Rule mutation"))
	}
}
