package generatorselection

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// selectedRuleReadProbe observes calls at the evaluator Source interface, not
// physical Provider reads. It refuses every attribute except the selected one.
type selectedRuleReadProbe struct {
	attribute core.CanonicalID
	calls     []core.CanonicalID
}

func (p *selectedRuleReadProbe) Number(attribute core.CanonicalID) (*big.Rat, error) {
	p.calls = append(p.calls, attribute)
	if attribute != p.attribute {
		return nil, fmt.Errorf("unexpected attribute read: %s", attribute)
	}
	return big.NewRat(1, 1), nil
}

func (p *selectedRuleReadProbe) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("unexpected relation read: %s/%s", relation, attribute)
}

func copySelectedRuleWorkspace(t *testing.T, ref string) (string, []byte) {
	t.Helper()
	root := t.TempDir()
	fixture := filepath.Join(repoRoot(t), "internal", "workspace", "testdata", "tei-md-rule")
	const rulePath = "semantic/sales/order-line-rules.tei.md"
	for _, rel := range []string{"tei.workspace.yaml", "semantic/sales/order.tei", rulePath} {
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		switch rel {
		case "semantic/sales/order.tei":
			// Both variants have the same Data/Type model. Only the Rule reference
			// changes between them, so the test really has one semantic cause.
			const before = "attribute quantity type std.decimal"
			if bytes.Count(raw, []byte(before)) != 1 {
				t.Fatalf("baseline Data has changed: %s", rel)
			}
			raw = bytes.Replace(raw, []byte(before), []byte(before+"\n    attribute threshold type std.decimal"), 1)
		case rulePath:
			if ref == "threshold" {
				const before = "sales.order_line.quantity is positive"
				if bytes.Count(raw, []byte(before)) != 1 {
					t.Fatalf("baseline Rule has changed: %s", rel)
				}
				raw = bytes.Replace(raw, []byte(before), []byte("sales.order_line.threshold is positive"), 1)
			}
		}
		target := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rule, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rulePath)))
	if err != nil {
		t.Fatal(err)
	}
	return root, rule
}

func selectedRuleContract(t *testing.T, workspaceRoot, ref string) string {
	t.Helper()
	path := validationTestPaths(t).contract
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var contract Contract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.Validation == nil || len(contract.Inputs) != 1 {
		t.Fatal("reference generator contract shape changed")
	}
	contract.Inputs[0].Attribute = "sales.order_line." + ref
	contract.Validation.Input = ref
	encoded, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(workspaceRoot, "generator-contract.json")
	if err := os.WriteFile(out, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return out
}

func selectedRuleRun(t *testing.T, binary string, input map[string]any, wantState, wantCode string) {
	t.Helper()
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary)
	cmd.Stdin = bytes.NewReader(encoded)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated binary failed on %s: %v\n%s", encoded, err, output)
	}
	var got struct {
		State string `json:"state"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("generated binary emitted invalid JSON: %s (%v)", output, err)
	}
	if got.State != wantState || got.Code != wantCode {
		t.Fatalf("input=%s: state/code=%q/%q, want %q/%q", encoded, got.State, got.Code, wantState, wantCode)
	}
}

// TestSelectedMarkdownRuleRebindThroughGenerator is the product-reader-to-
// reference-generator R1-M pressure test. It is not a production Target build,
// Plugin implementation selection, physical Provider test, or Stage2 self-build.
func TestSelectedMarkdownRuleRebindThroughGenerator(t *testing.T) {
	const ruleID = "sales.order_line.quantity_positive"
	const rulePath = "semantic/sales/order-line-rules.tei.md"
	generatorSource := filepath.Join(validationTestPaths(t).generators, generatorSources["go-positive"])
	var codeSHA [2][32]byte
	for i, ref := range []string{"quantity", "threshold"} {
		t.Run(ref, func(t *testing.T) {
			root, raw := copySelectedRuleWorkspace(t, ref)
			loaded, err := workspace.Load(root)
			if err != nil {
				t.Fatalf("selected Markdown must enter product Workspace: %v", err)
			}
			if len(loaded.AuthoringSources) != 1 ||
				loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
				loaded.AuthoringSources[0].Language.ContractVersion != "v0" ||
				loaded.AuthoringSources[0].Definitions != 2 {
				t.Fatalf("selected product reader was not used: %+v", loaded.AuthoringSources)
			}
			readByLoader := false
			for _, managed := range loaded.ManagedFiles {
				if managed.Path == rulePath {
					readByLoader = managed.ReadByLoader
				}
			}
			if !readByLoader {
				t.Fatal("selected Markdown was not read by product Loader")
			}
			wantAttr := core.CanonicalID("sales.order_line." + ref)
			rule, ok := findValidation(loaded.World, ruleID)
			if !ok || rule.ID != ruleID || rule.Kind != semantic.ValidationPositive || rule.Attribute != wantAttr {
				t.Fatalf("Rule did not follow the authored reference: %+v, exists=%t", rule, ok)
			}
			for _, diag := range validation.Validate(loaded.World) {
				if diag.Subject != nil && diag.Subject.ID == ruleID && diag.Severity == validation.Error {
					t.Fatalf("selected Rule failed normal semantic validation: %+v", diag)
				}
			}
			index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
			if err != nil {
				t.Fatal(err)
			}
			refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
			if len(refs) != 1 || refs[0].ArtifactPath != rulePath ||
				refs[0].Range.Validate(raw) != nil ||
				string(raw[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]) != ruleID {
				t.Fatalf("selected Rule has no exact source anchor: %+v", refs)
			}
			probe := &selectedRuleReadProbe{attribute: wantAttr}
			result, err := validationeval.Evaluate(rule, probe)
			if err != nil || result != validationeval.ResultSatisfied ||
				len(probe.calls) != 1 || probe.calls[0] != wantAttr {
				t.Fatalf("Rule did not request its selected attribute: result=%s error=%v reads=%v", result, err, probe.calls)
			}

			contractPath := selectedRuleContract(t, root, ref)
			candidateDir := filepath.Join(t.TempDir(), "candidate")
			if _, err := GenerateFromTransformSource(TransformGeneratorOptions{
				ID: "go-positive", WorkspaceDir: root, ContractPath: contractPath,
				GeneratorSourcePath: generatorSource, OutDir: candidateDir,
			}); err != nil {
				t.Fatalf("selected product Rule could not reach the existing generator: %v", err)
			}
			generated, err := os.ReadFile(filepath.Join(candidateDir, "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(generated, []byte("positive over "+string(wantAttr))) {
				t.Fatalf("generator did not consume the selected Attribute: %s", generated[:min(len(generated), 250)])
			}
			codeSHA[i] = sha256.Sum256(generated)
			binary := filepath.Join(t.TempDir(), "selected-rule")
			cmd := exec.Command("go", "build", "-trimpath", "-o", binary, filepath.Join(candidateDir, "main.go"))
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("generated Go could not build in a separate process: %v\n%s", err, output)
			}
			selectedRuleRun(t, binary, map[string]any{ref: "1"}, "satisfied", "")
			selectedRuleRun(t, binary, map[string]any{ref: "0"}, "not-satisfied", "")
			selectedRuleRun(t, binary, map[string]any{ref: "-1"}, "not-satisfied", "")
			selectedRuleRun(t, binary, map[string]any{ref: nil}, "unevaluable", "null")
			selectedRuleRun(t, binary, map[string]any{}, "unevaluable", "missing")
			otherRef := "threshold"
			if ref == "threshold" {
				otherRef = "quantity"
			}
			selectedRuleRun(t, binary, map[string]any{otherRef: "1"}, "unevaluable", "missing")
		})
	}
	if codeSHA[0] == codeSHA[1] {
		t.Fatal("changing the selected Rule reference did not change generated Go bytes")
	}
}

func TestSelectedMarkdownRuleRefusesTextRebind(t *testing.T) {
	root, _ := copySelectedRuleWorkspace(t, "threshold")
	const dataPath = "semantic/sales/order.tei"
	path := filepath.Join(root, filepath.FromSlash(dataPath))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const before = "attribute threshold type std.decimal"
	if bytes.Count(b, []byte(before)) != 1 {
		t.Fatal("the independent negative control lost its Data attribute")
	}
	if err := os.WriteFile(path, bytes.Replace(b, []byte(before), []byte("attribute threshold type std.text"), 1), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := selectedRuleContract(t, root, "threshold")
	_, err = GenerateFromTransformSource(TransformGeneratorOptions{
		ID: "go-positive", WorkspaceDir: root, ContractPath: contract,
		GeneratorSourcePath: filepath.Join(validationTestPaths(t).generators, generatorSources["go-positive"]),
		OutDir:              filepath.Join(t.TempDir(), "candidate"),
	})
	if err == nil || (!strings.Contains(err.Error(), "resolves to std.text") && !strings.Contains(err.Error(), "numeric")) {
		t.Fatalf("non-numeric Rule/contract mismatch reached generator: %v", err)
	}
}

// selectedRuleValueSource models only the pure evaluator's logical numeric
// input. It is intentionally not a DB Provider, query reader, or authorization
// source. A nil value represents unevaluable data at this narrow boundary.
type selectedRuleValueSource struct {
	attribute core.CanonicalID
	value     *big.Rat
	calls     []core.CanonicalID
}

func (p *selectedRuleValueSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	p.calls = append(p.calls, attribute)
	if attribute != p.attribute {
		return nil, fmt.Errorf("unexpected numeric read: %s", attribute)
	}
	return p.value, nil
}

func (p *selectedRuleValueSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("unexpected relation read: %s/%s", relation, attribute)
}

// The same selected Markdown Rule must agree across two *different* pure
// realizations: validationeval.Evaluate interprets semantic.Validation, whereas
// a deterministic Transform Generator emits Go that a different OS process
// executes. The independent table is fixed below, rather than derived from
// either evaluator's output. This does not compare physical Provider reads.
func TestSelectedRuleInterpretedAndGeneratedRealizationsAgree(t *testing.T) {
	const ruleID = "sales.order_line.quantity_positive"
	const inputName = "quantity"
	workspaceRoot, _ := copySelectedRuleWorkspace(t, inputName)
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		t.Fatalf("read selected Markdown Rule: %v", err)
	}
	rule, exists := findValidation(loaded.World, ruleID)
	if !exists || rule.Attribute != "sales.order_line.quantity" || rule.Kind != semantic.ValidationPositive {
		t.Fatalf("unexpected typed Rule: %+v exists=%t", rule, exists)
	}
	generatorSource := filepath.Join(validationTestPaths(t).generators, generatorSources["go-positive"])
	contractPath := selectedRuleContract(t, workspaceRoot, inputName)
	candidateDir := filepath.Join(t.TempDir(), "candidate")
	if _, err := GenerateFromTransformSource(TransformGeneratorOptions{
		ID:                  "go-positive",
		WorkspaceDir:        workspaceRoot,
		ContractPath:        contractPath,
		GeneratorSourcePath: generatorSource,
		OutDir:              candidateDir,
	}); err != nil {
		t.Fatalf("generate Rule implementation: %v", err)
	}
	if err := VerifyTransformCandidate(candidateDir, workspaceRoot, contractPath); err != nil {
		t.Fatalf("generated Candidate does not reproduce the recorded Source: %v", err)
	}
	binary := filepath.Join(t.TempDir(), "selected-rule")
	cmd := exec.Command("go", "build", "-trimpath", "-o", binary, filepath.Join(candidateDir, "main.go"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build generated Go as independent OS process: %v\n%s", err, output)
	}

	// "missing" and "null" have different generated-JSON adapter codes, but
	// both map to an evaluator error, never to a successful false predicate.
	// No rounding, exponent coercion, or implicit field substitution is used.
	cases := []struct {
		name      string
		input     map[string]any
		number    *big.Rat
		state     string
		code      string
		available bool
	}{
		{name: "positive", input: map[string]any{inputName: "2"}, number: big.NewRat(2, 1), state: "satisfied", available: true},
		{name: "positive-fraction", input: map[string]any{inputName: "0.01"}, number: big.NewRat(1, 100), state: "satisfied", available: true},
		{name: "zero", input: map[string]any{inputName: "0"}, number: big.NewRat(0, 1), state: "not-satisfied", available: true},
		{name: "negative", input: map[string]any{inputName: "-1"}, number: big.NewRat(-1, 1), state: "not-satisfied", available: true},
		{name: "negative-zero", input: map[string]any{inputName: "-0"}, number: big.NewRat(0, 1), state: "not-satisfied", available: true},
		{name: "explicit-null", input: map[string]any{inputName: nil}, state: "unevaluable", code: "null"},
		{name: "absent", input: map[string]any{}, state: "unevaluable", code: "missing"},
		{name: "wrong-type", input: map[string]any{inputName: true}, state: "unevaluable", code: "type-mismatch"},
	}
	attribute := core.CanonicalID("sales.order_line.quantity")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &selectedRuleValueSource{attribute: attribute, value: tc.number}
			interpreted, evalErr := validationeval.Evaluate(rule, source)
			if len(source.calls) != 1 || source.calls[0] != attribute {
				t.Fatalf("interpreter changed the logical Read requirement: %+v", source.calls)
			}
			if tc.available {
				if evalErr != nil || string(interpreted) != tc.state {
					t.Fatalf("interpreted state=%q err=%v; want %q", interpreted, evalErr, tc.state)
				}
			} else if evalErr == nil || interpreted != "" {
				t.Fatalf("unevaluable input became a valid predicate: state=%q err=%v", interpreted, evalErr)
			}
			selectedRuleRun(t, binary, tc.input, tc.state, tc.code)
		})
	}
	// A wrong key may carry a valid number but cannot satisfy the selected
	// contract; the generated Program must not silently read it.
	selectedRuleRun(t, binary, map[string]any{"threshold": "2"}, "unevaluable", "missing")
}
