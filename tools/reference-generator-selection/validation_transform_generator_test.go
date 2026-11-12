package generatorselection

// Generator Source (transform JSON) -> executing generator (transform
// evaluator) -> Target artifact (generated Go) -> qualified by Select against
// the independent Expected (hand-written testdata/validation-positive/fixtures.json).
// Nothing here makes the fixture from a generator, an evaluator, or a candidate.

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

const validationPositiveID = "sales.order_line.quantity_positive"

type validationPaths struct {
	workspace, contract, fixtures, generators string
}

func validationTestPaths(t *testing.T) validationPaths {
	t.Helper()
	root := repoRoot(t)
	base := filepath.Join(root, "tools", "reference-generator-selection", "testdata", "validation-positive")
	return validationPaths{
		workspace:  filepath.Join(root, "reference", "workspace"),
		contract:   filepath.Join(base, "contract.json"),
		fixtures:   filepath.Join(base, "fixtures.json"),
		generators: filepath.Join(base, "generator"),
	}
}

func validationWorld(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load(validationTestPaths(t).workspace)
	if err != nil {
		t.Fatal(err)
	}
	return loaded.World
}

func acceptedValidation(t *testing.T, id string) semantic.Validation {
	t.Helper()
	validation, ok := findValidation(validationWorld(t), id)
	if !ok {
		t.Fatalf("validation %s is not in the Workspace", id)
	}
	return validation
}

// generatorSources maps a candidate id to its Generator Source file.
var generatorSources = map[string]string{
	"go-positive":                      "go-positive.transform.json",
	"mutant-zero-accepted":             "mutant-zero-accepted.transform.json",
	"mutant-null-as-satisfied":         "mutant-null-as-satisfied.transform.json",
	"mutant-float64":                   "mutant-float64.transform.json",
	"mutant-mismatch-as-not-satisfied": "mutant-mismatch-as-not-satisfied.transform.json",
}

// wantFailingCases is what each wrong Generator Source must fail on.
var wantFailingCases = map[string][]string{
	"go-positive":                      nil,
	"mutant-zero-accepted":             {"negative-zero", "zero", "zero-scale-2"},
	"mutant-null-as-satisfied":         {"missing", "null"},
	"mutant-float64":                   {"positive-400-zeros-then-1"},
	"mutant-mismatch-as-not-satisfied": {"type-mismatch-array", "type-mismatch-bool", "type-mismatch-empty-string", "type-mismatch-text"},
}

func generateValidationCandidate(t *testing.T, p validationPaths, id, outDir string) CandidateManifest {
	t.Helper()
	manifest, err := GenerateFromTransformSource(TransformGeneratorOptions{
		ID: id, WorkspaceDir: p.workspace, ContractPath: p.contract,
		GeneratorSourcePath: filepath.Join(p.generators, generatorSources[id]), OutDir: outDir,
	})
	if err != nil {
		t.Fatalf("generate %s: %v", id, err)
	}
	return manifest
}

var (
	validationSelectOnce    sync.Once
	validationSelectReceipt Receipt
	validationSelectErr     error
	validationSelectDir     string
)

// validationReceipt generates every Generator Source's Target artifact into one
// directory and runs Select once; the receipt is shared by the tests below.
func validationReceipt(t *testing.T) Receipt {
	t.Helper()
	p := validationTestPaths(t)
	validationSelectOnce.Do(func() {
		dir, err := os.MkdirTemp("", "tei-validation-transform-*")
		if err != nil {
			validationSelectErr = err
			return
		}
		validationSelectDir = dir
		for id := range generatorSources {
			if _, err := GenerateFromTransformSource(TransformGeneratorOptions{
				ID: id, WorkspaceDir: p.workspace, ContractPath: p.contract,
				GeneratorSourcePath: filepath.Join(p.generators, generatorSources[id]), OutDir: filepath.Join(dir, id),
			}); err != nil {
				validationSelectErr = fmt.Errorf("generate %s: %w", id, err)
				return
			}
		}
		validationSelectReceipt, validationSelectErr = Select(context.Background(), p.workspace, p.contract, p.fixtures, dir)
	})
	if validationSelectErr != nil {
		t.Fatal(validationSelectErr)
	}
	return validationSelectReceipt
}

func TestMain(m *testing.M) {
	code := m.Run()
	if validationSelectDir != "" {
		_ = os.RemoveAll(validationSelectDir)
	}
	os.Exit(code)
}

func failingCases(candidate CandidateReceipt) []string {
	var failed []string
	for _, result := range candidate.Results {
		if !result.Passed {
			failed = append(failed, result.CaseID)
		}
	}
	sort.Strings(failed)
	return failed
}

func TestTransformGeneratedValidationPassesSelection(t *testing.T) {
	receipt := validationReceipt(t)
	for _, candidate := range receipt.Candidates {
		if candidate.ID != "go-positive" {
			continue
		}
		if !candidate.Passed {
			t.Fatalf("correct Generator Source was rejected: %s; failing cases %v", candidate.RejectReason, failingCases(candidate))
		}
		if len(candidate.Results) != 15 {
			t.Fatalf("expected 15 cases, got %d", len(candidate.Results))
		}
		if receipt.GrantsRuntimeAuthority || receipt.ProvesRealProvider {
			t.Fatal("selection claimed authority or a real provider")
		}
		return
	}
	t.Fatal("go-positive candidate is missing from the receipt")
}

func TestTransformGeneratedMutantsFailSelection(t *testing.T) {
	receipt := validationReceipt(t)
	seen := map[string]bool{}
	for _, candidate := range receipt.Candidates {
		want, known := wantFailingCases[candidate.ID]
		if !known || candidate.ID == "go-positive" {
			continue
		}
		seen[candidate.ID] = true
		if candidate.Passed {
			t.Errorf("%s was qualified", candidate.ID)
			continue
		}
		if got := failingCases(candidate); !reflect.DeepEqual(got, want) {
			t.Errorf("%s failed cases %v, want %v (reject: %s)", candidate.ID, got, want, candidate.RejectReason)
		}
	}
	for id := range wantFailingCases {
		if id != "go-positive" && !seen[id] {
			t.Errorf("mutant %s is missing from the receipt", id)
		}
	}
	if len(receipt.Qualified) != 1 || receipt.Qualified[0] != "go-positive" {
		t.Fatalf("qualified = %v, want only go-positive", receipt.Qualified)
	}
}

func TestTransformGeneratorIsDeterministicAndUsesItsInput(t *testing.T) {
	p := validationTestPaths(t)
	sourceRaw, err := os.ReadFile(filepath.Join(p.generators, generatorSources["go-positive"]))
	if err != nil {
		t.Fatal(err)
	}
	validation := acceptedValidation(t, validationPositiveID)
	first, firstInput, err := evaluateValidationGenerator(sourceRaw, validation, "quantity")
	if err != nil {
		t.Fatal(err)
	}
	second, secondInput, err := evaluateValidationGenerator(sourceRaw, validation, "quantity")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || string(firstInput) != string(secondInput) {
		t.Fatal("the same Generator Source and input produced different bytes")
	}
	if !strings.Contains(string(first), validationPositiveID) {
		t.Fatal("Target artifact does not carry the Validation ID")
	}
	other := validation
	other.ID = core.CanonicalID(validationPositiveID + "_other")
	changed, _, err := evaluateValidationGenerator(sourceRaw, other, "quantity")
	if err != nil {
		t.Fatal(err)
	}
	if digest(changed) == digest(first) {
		t.Fatal("changing the Validation ID did not change the Target artifact")
	}
	renamed, _, err := evaluateValidationGenerator(sourceRaw, validation, "qty")
	if err != nil {
		t.Fatal(err)
	}
	if digest(renamed) == digest(first) {
		t.Fatal("changing the input key did not change the Target artifact")
	}
}

func TestTransformGeneratorFailsClosed(t *testing.T) {
	p := validationTestPaths(t)
	sourceRaw, err := os.ReadFile(filepath.Join(p.generators, generatorSources["go-positive"]))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Source rejects a relation-unique Validation", func(t *testing.T) {
		relation := acceptedValidation(t, "sales.order.lines.line_number_unique")
		if _, _, err := evaluateValidationGenerator(sourceRaw, relation, "lineNumber"); err == nil || !strings.Contains(err.Error(), "assert-enum") {
			t.Fatalf("relation-unique Validation was not stopped by the Source: %v", err)
		}
	})
	t.Run("Source rejects an input key that is blank", func(t *testing.T) {
		if _, _, err := evaluateValidationGenerator(sourceRaw, acceptedValidation(t, validationPositiveID), " "); err == nil {
			t.Fatal("blank input key was accepted")
		}
	})
	variants := map[string]func(map[string]any){
		"unknown Validation ID": func(c map[string]any) {
			c["validation"].(map[string]any)["id"] = "sales.order_line.quantity_proposed_positive"
		},
		"relation-unique Validation": func(c map[string]any) {
			v := c["validation"].(map[string]any)
			v["id"], v["kind"] = "sales.order.lines.line_number_unique", "relation-unique"
		},
		"input attribute differs from the Validation attribute": func(c map[string]any) {
			c["inputs"] = []any{map[string]any{"attribute": "sales.order_line.unit_price", "primitive": "std.money"}}
		},
		"kind differs from the Workspace": func(c map[string]any) {
			c["validation"].(map[string]any)["id"] = "sales.order.lines.line_number_unique"
		},
		"rounding set": func(c map[string]any) {
			c["rounding"] = map[string]any{"precision": 8, "scale": 2, "mode": "half-up"}
		},
		"projection set": func(c map[string]any) {
			c["projection"] = map[string]any{"kind": "identity"}
		},
		"two inputs": func(c map[string]any) {
			c["inputs"] = append(c["inputs"].([]any), map[string]any{"attribute": "sales.order_line.unit_price", "primitive": "std.money"})
		},
		"input key with a quote": func(c map[string]any) {
			c["validation"].(map[string]any)["input"] = `quantity"`
		},
	}
	names := make([]string, 0, len(variants))
	for name := range variants {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			var contract map[string]any
			raw, err := os.ReadFile(p.contract)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &contract); err != nil {
				t.Fatal(err)
			}
			variants[name](contract)
			changed, err := json.Marshal(contract)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			contractPath := filepath.Join(dir, "contract.json")
			if err := os.WriteFile(contractPath, changed, 0o644); err != nil {
				t.Fatal(err)
			}
			outDir := filepath.Join(dir, "candidate")
			_, err = GenerateFromTransformSource(TransformGeneratorOptions{
				ID: "x", WorkspaceDir: p.workspace, ContractPath: contractPath,
				GeneratorSourcePath: filepath.Join(p.generators, generatorSources["go-positive"]), OutDir: outDir,
			})
			if err == nil {
				t.Fatal("generation succeeded")
			}
			if _, statErr := os.Stat(outDir); !os.IsNotExist(statErr) {
				t.Fatalf("a candidate directory was written: %v", statErr)
			}
			// Select must refuse the same contract before it looks at any candidate.
			if _, err := Select(context.Background(), p.workspace, contractPath, p.fixtures, t.TempDir()); err == nil {
				t.Fatal("Select accepted the contract")
			}
		})
	}
}

func TestTransformCandidateBindsGeneratorSourceDigest(t *testing.T) {
	p := validationTestPaths(t)
	sourceRaw, err := os.ReadFile(filepath.Join(p.generators, generatorSources["go-positive"]))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	candidateDir := filepath.Join(dir, "go-positive")
	manifest := generateValidationCandidate(t, p, "go-positive", candidateDir)
	if manifest.Generation == nil || manifest.Generation.PromptDigest != digest(sourceRaw) {
		t.Fatalf("manifest does not record the Generator Source digest: %+v", manifest.Generation)
	}
	if manifest.Lineage != nil {
		t.Fatal("Lineage re-runs a Go generator and must not be set for a transform Source")
	}
	if err := VerifyTransformCandidate(candidateDir, p.workspace, p.contract); err != nil {
		t.Fatalf("untouched candidate failed verification: %v", err)
	}

	sourcePath := filepath.Join(candidateDir, transformSourceFile)
	t.Run("changed Source is rejected by Select", func(t *testing.T) {
		if err := os.WriteFile(sourcePath, append(sourceRaw, ' '), 0o644); err != nil {
			t.Fatal(err)
		}
		defer os.WriteFile(sourcePath, sourceRaw, 0o644)
		receipt, err := Select(context.Background(), p.workspace, p.contract, p.fixtures, dir)
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Passed || len(receipt.Candidates) != 1 || !strings.Contains(receipt.Candidates[0].RejectReason, "generation file verification failed") {
			t.Fatalf("tampered Generator Source was not rejected: %+v", receipt)
		}
		if err := VerifyTransformCandidate(candidateDir, p.workspace, p.contract); err == nil {
			t.Fatal("VerifyTransformCandidate accepted a changed Source")
		}
	})

	t.Run("Target artifact edited with matching digests is caught only by re-evaluation", func(t *testing.T) {
		mainPath := filepath.Join(candidateDir, "main.go")
		original, err := os.ReadFile(mainPath)
		if err != nil {
			t.Fatal(err)
		}
		defer os.WriteFile(mainPath, original, 0o644)
		edited := []byte(strings.Replace(string(original), "value.Sign() > 0", "value.Sign() >= 0", 1))
		if string(edited) == string(original) {
			t.Fatal("edit did not apply")
		}
		if err := os.WriteFile(mainPath, edited, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := VerifyTransformCandidate(candidateDir, p.workspace, p.contract); err == nil {
			t.Fatal("VerifyTransformCandidate accepted an edited Target artifact")
		}
	})
}

// numberSource feeds validationeval.Evaluate from one fixture input. It is a
// differential oracle only: the fixture stays the authority.
type numberSource struct{ input map[string]json.RawMessage }

func (s numberSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	raw, ok := s.input["quantity"]
	if !ok {
		return nil, fmt.Errorf("no value for %s", attribute)
	}
	var text *string
	if err := json.Unmarshal(raw, &text); err != nil || text == nil {
		return nil, fmt.Errorf("value for %s is not a decimal string", attribute)
	}
	value, ok := new(big.Rat).SetString(*text)
	if !ok {
		return nil, fmt.Errorf("value for %s is not a decimal", attribute)
	}
	return value, nil
}

func (numberSource) RelationKeys(core.CanonicalID, core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("not used")
}

func TestFixtureAgreesWithValidationEvalOnExpressibleCases(t *testing.T) {
	p := validationTestPaths(t)
	var fixture Fixture
	if _, err := readJSON(p.fixtures, &fixture); err != nil {
		t.Fatal(err)
	}
	rule := acceptedValidation(t, validationPositiveID)
	for _, tc := range fixture.Cases {
		var input map[string]json.RawMessage
		if err := json.Unmarshal(tc.Input, &input); err != nil {
			t.Fatal(err)
		}
		got, err := validationeval.Evaluate(rule, numberSource{input})
		switch tc.Want.State {
		case "satisfied", "not-satisfied":
			if err != nil || string(got) != tc.Want.State {
				t.Errorf("%s: oracle = %q, %v; fixture says %s", tc.ID, got, err, tc.Want.State)
			}
		case "unevaluable":
			// The oracle only reports that evaluation failed; it has no codes.
			if err == nil {
				t.Errorf("%s: oracle evaluated to %q; fixture says unevaluable/%s", tc.ID, got, tc.Want.Code)
			}
		default:
			t.Errorf("%s: unexpected fixture state %q", tc.ID, tc.Want.State)
		}
	}
}
