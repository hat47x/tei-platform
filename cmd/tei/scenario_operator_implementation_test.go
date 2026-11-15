package main

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/scenariotest"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// These tests drive the function `tei scenario run` calls to evaluate rules,
// on a Workspace read by the product Load. The operator is declared in a
// document, the implementation is selected in the manifest, and the expected
// results are the definitions of "not negative" and "not zero" on the value 0,
// where the two disagree.

const scenarioOperatorRule = "sales.order_line.discount_non_negative"

func scenarioOperatorWorkspace(t *testing.T, implementations string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml": "workspace: scenario_operator\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n" +
			"languages:\n  - tei.rule-authoring/v0\n" + implementations,
		"semantic/sales/order.tei": "namespace sales\n\ndata OrderLine {\n    attribute discount type std.decimal\n}\n",
		"semantic/sales/discount-rules.tei.md": "# 値引きの規則\n\n値引きは負の数にしない。0は正しい入力である。\n\n" +
			"```tei\nnamespace sales\n\noperator sales.non_negative: takes number\n\n" +
			"rule sales.order_line.discount_non_negative:\n  sales.order_line.discount is sales.non_negative\n```\n",
		"applications/discount-check/application.tei": "application discount_check {\n    uses semantic sales\n" +
			"    uses validation sales.order_line.discount_non_negative\n}\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runScenarioOperatorStep(t *testing.T, loaded workspace.Result, discount, expected string, available []validationeval.NamedImplementation) (map[string]string, map[string]string, error) {
	t.Helper()
	step := scenariotest.Step{
		ID: "discount", Operation: "evaluate-validations",
		ExpectValidations: map[string]string{scenarioOperatorRule: expected},
	}
	fixture := scenariotest.ValidationFixture{Numbers: map[string]string{"sales.order_line.discount": discount}}
	return executeScenarioValidations(driver.NewResolver(loaded.World), "discount_check", step, fixture,
		scenarioImplementationSelections(loaded.OperatorImplementations), available)
}

func TestScenarioDecidesAnOperatorRuleWithTheSelectedImplementation(t *testing.T) {
	const useNonNegative = "implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n"
	const useNonZero = "implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-zero\n"

	loadedNonNegative, err := workspace.Load(scenarioOperatorWorkspace(t, useNonNegative))
	if err != nil {
		t.Fatal(err)
	}
	loadedNonZero, err := workspace.Load(scenarioOperatorWorkspace(t, useNonZero))
	if err != nil {
		t.Fatal(err)
	}

	// The selection is not meaning: the same documents load to the same World.
	left, _ := json.Marshal(loadedNonNegative.World)
	right, _ := json.Marshal(loadedNonZero.World)
	if string(left) != string(right) {
		t.Fatal("selecting another implementation changed the loaded World")
	}

	cases := []struct {
		name     string
		loaded   workspace.Result
		discount string
		want     string
		called   string
	}{
		{"non-negative on 0", loadedNonNegative, "0", "satisfied", "tei.number.non-negative"},
		{"non-negative on -0.01", loadedNonNegative, "-0.01", "not-satisfied", "tei.number.non-negative"},
		{"non-negative on 300", loadedNonNegative, "300", "satisfied", "tei.number.non-negative"},
		{"non-zero on 0", loadedNonZero, "0", "not-satisfied", "tei.number.non-zero"},
		{"non-zero on -0.01", loadedNonZero, "-0.01", "satisfied", "tei.number.non-zero"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, called, err := runScenarioOperatorStep(t, tc.loaded, tc.discount, tc.want, validationeval.BuiltinImplementations())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(results, map[string]string{scenarioOperatorRule: tc.want}) {
				t.Fatalf("results=%v", results)
			}
			if !reflect.DeepEqual(called, map[string]string{scenarioOperatorRule: tc.called}) {
				t.Fatalf("called=%v, want the selected implementation %s", called, tc.called)
			}
		})
	}

	t.Run("a wrong expectation is reported, not adopted", func(t *testing.T) {
		if _, _, err := runScenarioOperatorStep(t, loadedNonZero, "0", "satisfied", validationeval.BuiltinImplementations()); err == nil ||
			!strings.Contains(err.Error(), "result=not-satisfied, expected=satisfied") {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestScenarioRefusesAnOperatorRuleWithoutExactlyOneAvailableImplementation(t *testing.T) {
	selected, err := workspace.Load(scenarioOperatorWorkspace(t,
		"implementations:\n  - operator: sales.non_negative\n    use: tei.number.non-negative\n"))
	if err != nil {
		t.Fatal(err)
	}
	unselected, err := workspace.Load(scenarioOperatorWorkspace(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := workspace.Load(scenarioOperatorWorkspace(t,
		"implementations:\n  - operator: sales.non_negative\n    use: example.not-carried\n"))
	if err != nil {
		t.Fatal(err)
	}

	builtin := validationeval.BuiltinImplementations()
	calls := 0
	shadow := validationeval.NamedImplementation{
		ID: "tei.number.non-negative", Input: semantic.OperatorInputNumber,
		Number: func(*big.Rat) (bool, error) { calls++; return true, nil },
	}

	cases := []struct {
		name      string
		loaded    workspace.Result
		available []validationeval.NamedImplementation
		want      string
	}{
		{"nothing selected", unselected, builtin, "no selected implementation"},
		{"selected name this tei does not carry", unknown, builtin, "TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE"},
		{"nothing available", selected, nil, "TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE"},
		{"two implementations under the selected name", selected, append([]validationeval.NamedImplementation{shadow}, builtin...), "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS"},
		{"two implementations under the selected name, other order", selected, append(append([]validationeval.NamedImplementation{}, builtin...), shadow), "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, called, err := runScenarioOperatorStep(t, tc.loaded, "0", "satisfied", tc.available)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want %q", err, tc.want)
			}
			if results != nil || called != nil {
				t.Fatalf("a refused step still produced results=%v called=%v", results, called)
			}
		})
	}
	if calls != 0 {
		t.Fatalf("an ambiguous implementation was called %d times", calls)
	}
}
