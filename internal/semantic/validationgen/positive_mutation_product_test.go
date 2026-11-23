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
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// The model stays fixed while the selected Markdown Rule changes one read
// target. Both versions are reloaded, regenerated and run as separate Go
// processes against identical input values. This is R1, not a Builder or
// runtime Provider qualification.
func TestSelectedMarkdownRuleMutationChangesGeneratedGoAndRealResult(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate product test")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixture := filepath.Join(repoRoot, "internal", "workspace", "testdata", "tei-md-rule")
	root := t.TempDir()
	const rulePath = "semantic/sales/order-line-rules.tei.md"
	const modelPath = "semantic/sales/order.tei"
	for _, path := range []string{"tei.workspace.yaml", modelPath, rulePath} {
		data, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if path == modelPath {
			// Both attempts use this same two-decimal-attribute model.
			const marker = "    attribute quantity type std.decimal\n"
			if bytes.Count(data, []byte(marker)) != 1 {
				t.Fatal("model fixture has no unique quantity declaration")
			}
			data = bytes.Replace(data, []byte(marker),
				[]byte(marker+"    attribute secondQuantity type std.decimal\n"), 1)
		}
		destination := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	const ruleID = "sales.order_line.quantity_positive"
	type proof struct {
		out                 goemit.Output
		code                []byte
		modelRevision       string
		artifactFingerprint string
		rangeStart          int
		rangeEnd            int
	}
	generate := func(wantAttribute string) proof {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("load selected Markdown Rule: %v", err)
		}
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
			loaded.AuthoringSources[0].Language.ContractVersion != "v0" ||
			loaded.AuthoringSources[0].Definitions != 2 {
			t.Fatalf("unexpected real Language selection: %+v", loaded.AuthoringSources)
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatalf("Source Index from same Workspace load: %v", err)
		}
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
		if len(refs) != 1 || refs[0].ArtifactPath != rulePath ||
			refs[0].Role != sourceindex.RoleDefinition {
			t.Fatalf("selected Rule has wrong original Source: %+v", refs)
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rulePath)))
		if err != nil {
			t.Fatal(err)
		}
		start, end := refs[0].Range.Start.Offset, refs[0].Range.End.Offset
		if start < 0 || end > len(raw) || start >= end || string(raw[start:end]) != ruleID {
			t.Fatalf("Rule byte anchor is invalid: %+v", refs[0])
		}
		var rule semantic.Validation
		var attributeType core.CanonicalID
		for _, ns := range loaded.World.Semantics {
			for _, candidate := range ns.Validations {
				if string(candidate.ID) == ruleID {
					if rule.ID != "" {
						t.Fatal("ambiguous Rule in typed World")
					}
					rule = candidate
				}
			}
			for _, data := range ns.Data {
				for _, attribute := range data.Attributes {
					if string(attribute.ID) == wantAttribute {
						attributeType = attribute.TypeRef
					}
				}
			}
		}
		if rule.ID != ruleID || string(rule.Attribute) != wantAttribute ||
			rule.Kind != semantic.ValidationPositive || attributeType != "std.decimal" {
			t.Fatalf("typed Rule or resolved Attribute changed: %+v, type=%s", rule, attributeType)
		}
		out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
		if err := out.Err(); err != nil {
			t.Fatalf("lower actual selected Rule: %v", err)
		}
		code := out.File(out.Trace[0].File)
		bindings, err := goemittest.Bindings(out)
		if err != nil || len(bindings) != 1 || bindings[0].Attribute != wantAttribute ||
			!bytes.Contains(code, []byte(bindings[0].Parameter+".Sign() > 0")) ||
			bytes.Contains(code, []byte("validationeval")) {
			t.Fatalf("generated Go does not contain the selected Rule's real binding and decision: %+v %v", bindings, err)
		}
		var artifactFP string
		for _, artifact := range index.Artifacts {
			if artifact.Path == rulePath {
				artifactFP = artifact.Fingerprint
			}
		}
		if artifactFP != sourceindex.Fingerprint(raw) {
			t.Fatal("selected Source artifact does not match original bytes")
		}
		return proof{
			out: out, code: code, modelRevision: sourceindex.ModelRevision(index),
			artifactFingerprint: artifactFP, rangeStart: start, rangeEnd: end,
		}
	}

	before := generate("sales.order_line.quantity")
	path := filepath.Join(root, filepath.FromSlash(rulePath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const oldPredicate = "sales.order_line.quantity is positive"
	const newPredicate = "sales.order_line.second_quantity is positive"
	if bytes.Count(raw, []byte(oldPredicate)) != 1 {
		t.Fatal("selected predicate occurrence is not unique")
	}
	changed := bytes.Replace(raw, []byte(oldPredicate), []byte(newPredicate), 1)
	if err := os.WriteFile(path, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	after := generate("sales.order_line.second_quantity")
	if bytes.Equal(before.code, after.code) ||
		before.modelRevision == after.modelRevision ||
		before.artifactFingerprint == after.artifactFingerprint ||
		before.rangeStart != after.rangeStart || before.rangeEnd != after.rangeEnd {
		t.Fatal("one Rule edit was not reflected in generated Go and Source revision")
	}

	// The entry code and fixed test input are identical for both Go attempts.
	// Only the generated Go, and the binding the harness takes from its doc
	// comment, change between actual process runs.
	fixed := []goemittest.Case{{Name: "fixed", Row: map[string]goemittest.Cell{
		"sales.order_line.quantity":        {Text: "0"},
		"sales.order_line.second_quantity": {Text: "2"},
	}}}
	if got := goemittest.Run(t, before.out, fixed); got != "fixed:not-satisfied:1:sales.order_line.quantity\n" {
		t.Fatalf("old generated Rule ignored fixed input: %q", got)
	}
	if got := goemittest.Run(t, after.out, fixed); got != "fixed:satisfied:1:sales.order_line.second_quantity\n" {
		t.Fatalf("re-generated Rule ignored changed read target: %q", got)
	}
}
