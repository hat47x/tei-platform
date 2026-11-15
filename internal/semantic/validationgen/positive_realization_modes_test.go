package validationgen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
	"tei/internal/workspace"
)

// Both realizations use one selected typed positive Rule and an independent
// expected result table. This is local evaluation conformance, NOT evidence
// of a selected production Provider, runtime authority or a product Artifact.
func TestSelectedPositiveRuleDirectAndGeneratedEvaluation(t *testing.T) {
	if testing.Short() {
		t.Skip("generated Go must be compiled and run")
	}
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	fixture := filepath.Join(root, "internal", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" {
		t.Fatalf("unexpected selected Language Reader: %+v", loaded.AuthoringSources)
	}

	var rule semantic.Validation
	var attributeType core.CanonicalID
	for _, ns := range loaded.World.Semantics {
		for _, candidate := range ns.Validations {
			if candidate.ID == "sales.order_line.quantity_positive" {
				if rule.ID != "" {
					t.Fatal("duplicate selected Rule")
				}
				rule = candidate
			}
		}
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				if attribute.ID == rule.Attribute {
					attributeType = attribute.TypeRef
				}
			}
		}
	}
	if rule.ID != "sales.order_line.quantity_positive" ||
		rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != "sales.order_line.quantity" ||
		rule.Relation != "" || attributeType != "std.decimal" {
		t.Fatalf("unsupported or unresolved selected Rule: %+v, type=%s", rule, attributeType)
	}

	generated, err := validationgen.GeneratePositive(rule, attributeType, "main")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(generated, []byte("value.Sign() > 0")) ||
		bytes.Contains(generated, []byte("validationeval.Evaluate(")) {
		t.Fatal("generated Go does not own the positive decision")
	}

	// Each realization is checked against fixed expectations, not against
	// the other realization's output. Runtime evaluation is limited to the
	// Shape that GeneratePositive accepts.
	const consumer = `package main
import (
	"fmt"
	"math/big"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

type numberSource struct {
	number *big.Rat
	unavailable bool
	reads int
	requested core.CanonicalID
}
func (s *numberSource) Number(id core.CanonicalID) (*big.Rat, error) {
	s.reads++
	s.requested = id
	if s.unavailable { return nil, fmt.Errorf("unfetched") }
	return s.number, nil
}
func (s *numberSource) RelationKeys(_, _ core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("unexpected relation read")
}
func main() {
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_positive",
		Kind: semantic.ValidationPositive,
		Attribute: "sales.order_line.quantity",
	}
	evaluators := []struct {
		name string
		evaluate func(validationeval.Source) (validationeval.Result, error)
	}{
		{"direct", func(s validationeval.Source) (validationeval.Result, error) {
			return validationeval.Evaluate(rule, s)
		}},
		{"generated", EvaluateSelected},
	}
	cases := []struct {
		name, value, want string
		unavailable bool
	}{
		{"positive", "2.5", "satisfied", false},
		{"zero", "0", "not-satisfied", false},
		{"negative", "-1", "not-satisfied", false},
		{"null", "", "error", false},
		{"unavailable", "", "error", true},
	}
	for _, c := range cases {
		var value *big.Rat
		if c.value != "" {
			value = new(big.Rat)
			if _, ok := value.SetString(c.value); !ok { panic(c.value) }
		}
		for _, evaluator := range evaluators {
			s := &numberSource{number: value, unavailable: c.unavailable}
			result, err := evaluator.evaluate(s)
			if c.want == "error" {
				if err == nil || result != "" {
					panic(fmt.Sprintf("%s/%s: expected a failed evaluation, got %s / %v", evaluator.name, c.name, result, err))
				}
			} else if err != nil || string(result) != c.want {
				panic(fmt.Sprintf("%s/%s: got %s / %v, want %s", evaluator.name, c.name, result, err, c.want))
			}
			if s.reads != 1 || s.requested != rule.Attribute {
				panic(fmt.Sprintf("%s/%s: wrong read count or attribute: %d %s", evaluator.name, c.name, s.reads, s.requested))
			}
		}
	}
	for _, evaluator := range evaluators {
		result, err := evaluator.evaluate(nil)
		if err == nil || result != "" {
			panic(fmt.Sprintf("%s: nil source accepted", evaluator.name))
		}
	}
	fmt.Println("direct-and-generated:passed")
}
`
	directory, err := os.MkdirTemp(root, "tei-realization-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	if err := os.WriteFile(filepath.Join(directory, "generated.go"), generated, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(consumer), 0600); err != nil {
		t.Fatal(err)
	}

	command := exec.Command(goExe, "run", ".")
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("direct/generated Go comparison failed: %v\n%s", err, output)
	}
	if string(output) != "direct-and-generated:passed\n" {
		t.Fatalf("unexpected direct/generated comparison output: %s", output)
	}
}
