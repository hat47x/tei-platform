package validationgen_test

import (
	"bytes"
	"encoding/json"
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

// One product-selected typed Rule has two executable realizations: the
// existing evaluator and genuinely generated Go. They must agree on values,
// failures and read count for the same synthetic Source contract. This test
// does not claim an actual Provider, a selected deployment mode, an external
// Host, or business acceptance.
func TestSelectedPositiveRuleInterpretedAndGeneratedRealizations(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	fixture := filepath.Join("..", "..", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
		t.Fatalf("the selected product reader changed: %+v", loaded.AuthoringSources)
	}
	const ruleID = "sales.order_line.quantity_positive"
	var rule semantic.Validation
	var attributeType core.CanonicalID
	for _, ns := range loaded.World.Semantics {
		for _, r := range ns.Validations {
			if string(r.ID) == ruleID {
				if rule.ID != "" {
					t.Fatal("the selected Rule has more than one definition")
				}
				rule = r
			}
		}
		for _, data := range ns.Data {
			for _, a := range data.Attributes {
				if a.ID == rule.Attribute {
					attributeType = a.TypeRef
				}
			}
		}
	}
	if rule.Kind != semantic.ValidationPositive ||
		string(rule.Attribute) != "sales.order_line.quantity" ||
		attributeType != "std.decimal" {
		t.Fatalf("the same selected Rule cannot be realized twice: %+v type=%s", rule, attributeType)
	}
	source, err := validationgen.GeneratePositive(rule, attributeType, "main")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(source, []byte("value.Sign() > 0")) ||
		bytes.Contains(source, []byte("validationeval.Evaluate(")) {
		t.Fatal("generated Go must decide itself, not delegate to the interpreted implementation")
	}
	// The existing evaluator can consume a number, but the generated
	// realization requires its declared type. Never silently fall back.
	if _, err := validationgen.GeneratePositive(rule, "std.text", "main"); err == nil {
		t.Fatal("unsupported generated realization silently accepted a different type")
	}
	ruleBytes, err := json.Marshal(rule)
	if err != nil {
		t.Fatal(err)
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine the package source directory")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	dir, err := os.MkdirTemp(repoRoot, "tei-realization-parity-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	for name, data := range map[string][]byte{
		"generated.go": source,
		"rule.json": ruleBytes,
		"main.go": []byte(realizationParityConsumer),
	} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("two actual Go realizations could not be compared: %v\n%s", err, got)
	}
	const want = "positive:satisfied:1\nzero:not-satisfied:1\nnegative:not-satisfied:1\nnull:error:1\nmissing:error:1\n"
	if string(got) != want {
		t.Fatalf("interpreted/generated decisions or source reads differ:\n%s\nwant:\n%s", got, want)
	}
}

// The runner is an independent Go consumer, compiled together with the actual
// generated code. The product Workspace's typed Rule crosses as JSON bytes;
// neither realization is inferred from a prewritten expected string.
const realizationParityConsumer = `package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

type paritySource struct {
	value *big.Rat
	missing bool
	reads int
}

func (s *paritySource) Number(id core.CanonicalID) (*big.Rat, error) {
	if id != "sales.order_line.quantity" {
		return nil, fmt.Errorf("unexpected attribute %s", id)
	}
	s.reads++
	if s.missing { return nil, errors.New("value not obtained") }
	return s.value, nil
}
func (s *paritySource) RelationKeys(_, _ core.CanonicalID) ([]string, error) {
	return nil, errors.New("unexpected relation read")
}

func resultName(result validationeval.Result, err error) string {
	if err != nil {
		if result != "" { panic("failed evaluation also returned an outcome") }
		return "error"
	}
	return string(result)
}

func main() {
	raw, err := os.ReadFile("rule.json")
	if err != nil { panic(err) }
	var rule semantic.Validation
	if err := json.Unmarshal(raw, &rule); err != nil { panic(err) }
	if rule.ID != "sales.order_line.quantity_positive" ||
		rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != "sales.order_line.quantity" {
		panic("the input is not the chosen product Rule")
	}
	for _, tc := range []struct {name string; value string; missing bool} {
		{"positive", "2.5", false},
		{"zero", "0", false},
		{"negative", "-1", false},
		{"null", "", false},
		{"missing", "", true},
	} {
		var n *big.Rat
		if tc.value != "" {
			n = new(big.Rat)
			if _, ok := n.SetString(tc.value); !ok { panic(tc.value) }
		}
		a, b := &paritySource{value:n, missing:tc.missing}, &paritySource{value:n, missing:tc.missing}
		interpreted, interpretedError := validationeval.Evaluate(rule, a)
		generated, generatedError := EvaluateSelected(b)
		left, right := resultName(interpreted, interpretedError), resultName(generated, generatedError)
		if left != right || a.reads != 1 || b.reads != 1 {
			panic(fmt.Sprintf("%s: interpreted=%s/%d generated=%s/%d", tc.name, left, a.reads, right, b.reads))
		}
		fmt.Printf("%s:%s:%d\n", tc.name, left, a.reads)
	}
}
`
