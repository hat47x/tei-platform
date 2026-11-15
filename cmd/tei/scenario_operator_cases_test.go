package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// A Workspace states three examples of sales.non_negative next to its
// declaration. The scenario value is 5, which "not negative" and "not zero"
// both satisfy, so the result alone cannot tell which implementation ran.

func scenarioOperatorCasesWorkspace(t *testing.T, implementation string) string {
	t.Helper()
	root := scenarioOperatorWorkspace(t, "implementations:\n  - operator: sales.non_negative\n    use: "+implementation+"\n")
	manifest := filepath.Join(root, "tei.workspace.yaml")
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "tei.rule-authoring/v0") != 1 {
		t.Fatal("fixture manifest must select the fence-only language once")
	}
	if err := os.WriteFile(manifest, []byte(strings.Replace(string(raw), "tei.rule-authoring/v0", "tei.authoring/v1", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := "# 値引きの検査の例\n\n<!-- tei: cases of sales.non_negative -->\n" +
		"| value | expect |\n| --- | --- |\n| 300 | satisfied |\n| 0 | satisfied |\n| -0.01 | not-satisfied |\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "discount-cases.tei.md"), []byte(cases), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestScenarioUsesAnImplementationOnlyIfItAgreesWithTheExamples(t *testing.T) {
	t.Run("the implementation that agrees decides the rule", func(t *testing.T) {
		loaded, err := workspace.Load(scenarioOperatorCasesWorkspace(t, "tei.number.non-negative"))
		if err != nil {
			t.Fatal(err)
		}
		results, called, err := runScenarioOperatorStep(t, loaded, "5", "satisfied", validationeval.BuiltinImplementations())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(results, map[string]string{scenarioOperatorRule: "satisfied"}) ||
			!reflect.DeepEqual(called, map[string]string{scenarioOperatorRule: "tei.number.non-negative"}) {
			t.Fatalf("results=%v called=%v", results, called)
		}
	})

	t.Run("the implementation that disagrees on an example is refused", func(t *testing.T) {
		loaded, err := workspace.Load(scenarioOperatorCasesWorkspace(t, "tei.number.non-zero"))
		if err != nil {
			t.Fatal(err)
		}
		results, called, err := runScenarioOperatorStep(t, loaded, "5", "satisfied", validationeval.BuiltinImplementations())
		if err == nil || !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") ||
			!strings.Contains(err.Error(), "states that 0 is satisfied") {
			t.Fatalf("err=%v", err)
		}
		if results != nil || called != nil {
			t.Fatalf("a refused implementation produced results=%v called=%v", results, called)
		}
	})

	t.Run("without the examples the same wrong implementation passes", func(t *testing.T) {
		// Same Workspace minus the case table: this is the gap the examples close.
		root := scenarioOperatorCasesWorkspace(t, "tei.number.non-zero")
		if err := os.Remove(filepath.Join(root, "semantic", "sales", "discount-cases.tei.md")); err != nil {
			t.Fatal(err)
		}
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		results, _, err := runScenarioOperatorStep(t, loaded, "5", "satisfied", validationeval.BuiltinImplementations())
		if err != nil || results[scenarioOperatorRule] != "satisfied" {
			t.Fatalf("results=%v err=%v", results, err)
		}
	})
}
