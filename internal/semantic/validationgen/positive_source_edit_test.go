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
	project := func() ([]byte, sourceindex.Index, core.CanonicalID) {
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
		code, err := validationgen.GeneratePositive(rule, attributeType, "main")
		if err != nil {
			t.Fatalf("lower actual selected Rule: %v", err)
		}
		return code, index, rule.Attribute
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
	execute := func(code []byte) string {
		t.Helper()
		dir, err := os.MkdirTemp(repoRoot, "validationgen_source_change_")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(dir) })
		for name, raw := range map[string][]byte{
			"generated.go": code,
			"main.go": []byte(strings.Join([]string{
				"package main",
				"import (\"fmt\"; \"math/big\"; \"tei/internal/core\")",
				"type testSource struct { reads []string }",
				"func (s *testSource) Number(id core.CanonicalID) (*big.Rat, error) {",
				" s.reads = append(s.reads, string(id))",
				" switch id {",
				" case \"sales.order_line.quantity\": return big.NewRat(0, 1), nil",
				" case \"sales.order_line.discount\": return big.NewRat(2, 1), nil",
				" default: return nil, fmt.Errorf(\"unknown attribute %s\", id)",
				" }",
				"}",
				"func (s *testSource) RelationKeys(_, _ core.CanonicalID) ([]string, error) {",
				" return nil, fmt.Errorf(\"unexpected relation read\")",
				"}",
				"func main() {",
				" source := &testSource{}",
				" result, err := EvaluateSelected(source)",
				" if err != nil { panic(err) }",
				" fmt.Printf(\"%s:%v\\n\", result, source.reads)",
				"}",
			}, "\n") + "\n"),
		} {
			if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("generated Go did not execute: %v\n%s", err, out)
		}
		return string(out)
	}

	original, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rulePath)))
	if err != nil {
		t.Fatal(err)
	}
	initialGo, initialIndex, initialAttribute := project()
	if initialAttribute != "sales.order_line.quantity" {
		t.Fatalf("wrong initial read target: %s", initialAttribute)
	}
	if got := execute(initialGo); got != "not-satisfied:[sales.order_line.quantity]\n" {
		t.Fatalf("original generated decision/read: %q", got)
	}

	// Prose changes the Source revision, not the selected Rule or generated
	// decision. A stale Index must still fail rather than be silently reused.
	proseOnly := append(append([]byte(nil), original...), []byte("\n補足の文章だけを変更。\n")...)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rulePath)), proseOnly, 0o600); err != nil {
		t.Fatal(err)
	}
	validateStale(initialIndex)
	proseGo, proseIndex, proseAttribute := project()
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
	changedGo, _, changedAttribute := project()
	if changedAttribute != "sales.order_line.discount" ||
		bytes.Equal(changedGo, initialGo) ||
		!bytes.Contains(changedGo, []byte("source.Number(\"sales.order_line.discount\")")) {
		t.Fatal("authored Rule read target was not reflected in the regenerated Go")
	}
	if got := execute(changedGo); got != "satisfied:[sales.order_line.discount]\n" {
		t.Fatalf("edited Rule did not change actual generated decision/read: %q", got)
	}
}
