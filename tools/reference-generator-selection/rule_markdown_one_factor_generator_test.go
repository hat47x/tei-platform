package generatorselection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// Source-interface observations do not prove physical Provider I/O.
type ruleReadProbe struct{ reads []core.CanonicalID }

func (p *ruleReadProbe) Number(id core.CanonicalID) (*big.Rat, error) {
	p.reads = append(p.reads, id)
	switch id {
	case "sales.order_line.quantity":
		return big.NewRat(2, 1), nil
	case "sales.order_line.approved_quantity":
		return big.NewRat(-2, 1), nil
	}
	return nil, fmt.Errorf("unexpected read %s", id)
}
func (*ruleReadProbe) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	return nil, fmt.Errorf("unexpected relation read %s/%s", relation, attribute)
}

// Every version has the same two decimal attributes; only one Rule reference changes.
func ruleWorkspace(t *testing.T, attr string) (string, []byte) {
	t.Helper()
	root := t.TempDir()
	fixture := filepath.Join(repoRoot(t), "internal", "workspace", "testdata", "tei-md-rule")
	var markdown []byte
	for _, name := range []string{"tei.workspace.yaml", "semantic/sales/order.tei", "semantic/sales/order-line-rules.tei.md"} {
		data, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, "order.tei") {
			old := "attribute quantity type std.decimal"
			if strings.Count(string(data), old) != 1 {
				t.Fatal("model fixture drifted")
			}
			data = []byte(strings.Replace(string(data), old, old+"\n    attribute approvedQuantity type std.decimal", 1))
		}
		if strings.HasSuffix(name, ".tei.md") {
			old := "sales.order_line.quantity is positive"
			if strings.Count(string(data), old) != 1 {
				t.Fatal("rule fixture drifted")
			}
			data = []byte(strings.Replace(string(data), old, attr+" is positive", 1))
			markdown = data
		}
		destination := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, markdown
}

func ruleContract(t *testing.T, attribute, input string) string {
	t.Helper()
	data, err := os.ReadFile(validationTestPaths(t).contract)
	if err != nil {
		t.Fatal(err)
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Inputs) != 1 || contract.Validation == nil {
		t.Fatal("contract fixture drifted")
	}
	contract.Inputs[0].Attribute, contract.Validation.Input = attribute, input
	data, err = json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func runRuleGenerator(t *testing.T, root, contract string) ([]byte, error) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "candidate")
	paths := validationTestPaths(t)
	_, err := GenerateFromTransformSource(TransformGeneratorOptions{
		ID: "go-positive", WorkspaceDir: root, ContractPath: contract,
		GeneratorSourcePath: filepath.Join(paths.generators, generatorSources["go-positive"]), OutDir: out,
	})
	if err != nil {
		return nil, err
	}
	if err := VerifyTransformCandidate(out, root, contract); err != nil {
		return nil, err
	}
	generated, err := os.ReadFile(filepath.Join(out, "main.go"))
	if err != nil {
		return nil, err
	}
	binary := filepath.Join(t.TempDir(), "generated")
	if output, err := exec.Command("go", "build", "-trimpath", "-o", binary, filepath.Join(out, "main.go")).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("separate Go build failed: %v: %s", err, output)
	}
	command := exec.Command(binary)
	command.Stdin = strings.NewReader("{\"quantity\":\"2\",\"approvedQuantity\":\"-2\"}")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("separate Go run failed: %v: %s", err, output)
	}
	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}
	if result["state"] != "satisfied" && result["state"] != "not-satisfied" {
		return nil, fmt.Errorf("unexpected generated result: %s", output)
	}
	return append(append([]byte(nil), generated...), []byte("\nRESULT="+result["state"].(string))...), nil
}

func TestSelectedMarkdownRuleOneFactorChangeReachesGeneratedGo(t *testing.T) {
	const id = "sales.order_line.quantity_positive"
	cases := []struct{ attr, input, expected string }{
		{"sales.order_line.quantity", "quantity", "satisfied"},
		{"sales.order_line.approved_quantity", "approvedQuantity", "not-satisfied"},
	}
	var previousMarkdown, previousGo []byte
	for i, tc := range cases {
		root, markdown := ruleWorkspace(t, tc.attr)
		if i == 0 {
			previousMarkdown = markdown
		} else {
			want := strings.Replace(string(previousMarkdown), cases[0].attr+" is positive", tc.attr+" is positive", 1)
			if want != string(markdown) {
				t.Fatal("Rule edit was not one-factor")
			}
		}
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(loaded.AuthoringSources) != 1 || loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" {
			t.Fatalf("selected Reader not observed: %+v", loaded.AuthoringSources)
		}
		rule, ok := findValidation(loaded.World, id)
		if !ok || string(rule.Attribute) != tc.attr {
			t.Fatalf("typed Rule not updated: %+v", rule)
		}
		for _, d := range validation.Validate(loaded.World) {
			if d.Code == "UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE" || d.Code == "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" {
				t.Fatalf("Rule type invalid: %+v", d)
			}
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
		if len(refs) != 1 || string(markdown[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]) != id {
			t.Fatalf("Source anchor lost: %+v", refs)
		}
		probe := &ruleReadProbe{}
		result, err := validationeval.Evaluate(rule, probe)
		if err != nil || string(result) != tc.expected || len(probe.reads) != 1 || string(probe.reads[0]) != tc.attr {
			t.Fatalf("Rule evaluation/read changed incorrectly: %s %v %v", result, err, probe.reads)
		}
		generated, err := runRuleGenerator(t, root, ruleContract(t, tc.attr, tc.input))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(generated, []byte(tc.attr)) || !bytes.HasSuffix(generated, []byte("RESULT="+tc.expected)) {
			t.Fatal("generated Go did not follow edited Rule and independent Expected")
		}
		if i == 0 {
			previousGo = generated
		} else if bytes.Equal(generated, previousGo) {
			t.Fatal("one-factor change did not alter generated Go")
		}
	}
	changed, _ := ruleWorkspace(t, cases[1].attr)
	_, err := runRuleGenerator(t, changed, ruleContract(t, cases[0].attr, cases[0].input))
	if err == nil || !strings.Contains(err.Error(), "contract input is") {
		t.Fatalf("stale contract not rejected: %v", err)
	}

	textRoot, _ := ruleWorkspace(t, "sales.order_line.item_name")
	textWorld, err := workspace.Load(textRoot)
	if err != nil {
		t.Fatal(err)
	}
	typeError := false
	for _, d := range validation.Validate(textWorld.World) {
		if d.Code == "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" {
			typeError = true
		}
	}
	if !typeError {
		t.Fatal("text-based positive Rule escaped type validation")
	}
	_, err = runRuleGenerator(t, textRoot, ruleContract(t, "sales.order_line.item_name", "itemName"))
	if err == nil || !strings.Contains(err.Error(), "resolves to std.text") {
		t.Fatalf("non-numeric Rule was not rejected: %v", err)
	}
}
