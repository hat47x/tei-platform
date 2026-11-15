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
		code, err := validationgen.GeneratePositive(rule, attributeType, "main")
		if err != nil {
			t.Fatalf("lower actual selected Rule: %v", err)
		}
		if !bytes.Contains(code, []byte("value.Sign() > 0")) ||
			!bytes.Contains(code, []byte("source.Number(\""+wantAttribute+"\")")) ||
			bytes.Contains(code, []byte("validationeval.Evaluate(")) {
			t.Fatal("generated Go does not contain the selected Rule's real read and decision")
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
			code: code, modelRevision: sourceindex.ModelRevision(index),
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
	// Only the generated Go source is replaced between actual process runs.
	dir, err := os.MkdirTemp(repoRoot, "validationgen_mutation_")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	const runner = `package main
import (
    "errors"
    "fmt"
    "math/big"
    "tei/internal/core"
)
type fixedSource struct { reads []core.CanonicalID }
func (s *fixedSource) Number(id core.CanonicalID) (*big.Rat, error) {
    s.reads = append(s.reads, id)
    switch id {
    case "sales.order_line.quantity": return big.NewRat(0, 1), nil
    case "sales.order_line.second_quantity": return big.NewRat(2, 1), nil
    default: return nil, fmt.Errorf("unexpected numeric read %s", id)
    }
}
func (s *fixedSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
    return nil, errors.New("unexpected relation read")
}
func main() {
    s := &fixedSource{}
    outcome, err := EvaluateSelected(s)
    if err != nil || len(s.reads) != 1 { panic(fmt.Sprintf("outcome=%s error=%v reads=%v", outcome, err, s.reads)) }
    fmt.Printf("%s:%s:%d\n", outcome, s.reads[0], len(s.reads))
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(runner), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(code []byte) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "generated.go"), code, 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "run", ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("generated Go failed in another process: %v\n%s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if got := run(before.code); got != "not-satisfied:sales.order_line.quantity:1" {
		t.Fatalf("old generated Rule ignored fixed input: %q", got)
	}
	if got := run(after.code); got != "satisfied:sales.order_line.second_quantity:1" {
		t.Fatalf("re-generated Rule ignored changed read target: %q", got)
	}
}
