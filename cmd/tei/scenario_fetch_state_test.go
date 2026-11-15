package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/scenariotest"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// These tests drive the function `tei scenario run` calls to evaluate rules,
// on a Workspace read by the product Load, with a Fixture parsed from JSON as
// the command parses it. The Workspace has one rule over a number and one
// over a relation. The Fixture says what was read and what was not.

const (
	fetchStateDiscountRule = "sales.order_line.discount_non_negative"
	fetchStateUniqueRule   = "sales.order.lines.line_number_unique"
)

func fetchStateWorkspace(t *testing.T, implementation string) workspace.Result {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml": "workspace: scenario_fetch_state\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n" +
			"languages:\n  - tei.rule-authoring/v0\nimplementations:\n  - operator: sales.non_negative\n    use: " + implementation + "\n",
		"semantic/sales/order.tei": "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n    relation lines to sales.order_line many\n}\n\n" +
			"data OrderLine {\n    attribute lineNumber type std.integer\n    attribute discount type std.decimal\n}\n",
		"semantic/sales/order-rules.tei.md": "# 受注明細の規則\n\n値引きは負の数にしない。明細番号は一つの受注の中で重ならない。\n\n" +
			"```tei\nnamespace sales\n\noperator sales.non_negative: takes number\n\n" +
			"rule sales.order_line.discount_non_negative:\n  sales.order_line.discount is sales.non_negative\n\n" +
			"rule sales.order.lines.line_number_unique:\n  sales.order_line.line_number is unique per sales.order.lines\n```\n",
		"applications/order-check/application.tei": "application order_check {\n    uses semantic sales\n" +
			"    uses validation sales.order_line.discount_non_negative\n    uses validation sales.order.lines.line_number_unique\n}\n",
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
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

// runFetchStateStep evaluates both rules. discount and lines are the JSON
// members a Fixture author writes.
func runFetchStateStep(t *testing.T, loaded workspace.Result, discount, lines, expectDiscount, expectUnique string) (map[string]string, error) {
	t.Helper()
	fixture, err := scenariotest.ParseValidationFixture([]byte(`{` + discount + `, "relationKeys": [` +
		`{"relation": "sales.order.lines", "attribute": "sales.order_line.line_number", ` + lines + `}]}`))
	if err != nil {
		t.Fatalf("the Fixture of this test is not readable: %v", err)
	}
	step := scenariotest.Step{
		ID: "check", Operation: "evaluate-validations",
		ExpectValidations: map[string]string{fetchStateDiscountRule: expectDiscount, fetchStateUniqueRule: expectUnique},
	}
	results, _, err := executeScenarioValidations(driver.NewResolver(loaded.World), "order_check", step, fixture,
		scenarioImplementationSelections(loaded.OperatorImplementations), validationeval.BuiltinImplementations())
	return results, err
}

const (
	discountPresent = `"numbers": {"sales.order_line.discount": "5"}`
	linesComplete   = `"keys": ["1", "2", "3"]`
)

func TestScenarioDecidesRulesWhenEverythingWasRead(t *testing.T) {
	loaded := fetchStateWorkspace(t, "tei.number.non-negative")
	results, err := runFetchStateStep(t, loaded, discountPresent, linesComplete, "satisfied", "satisfied")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{fetchStateDiscountRule: "satisfied", fetchStateUniqueRule: "satisfied"}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("results=%v", results)
	}

	// Stating "complete" is the same as stating nothing.
	results, err = runFetchStateStep(t, loaded, discountPresent, linesComplete+`, "coverage": "complete"`, "satisfied", "satisfied")
	if err != nil || !reflect.DeepEqual(results, want) {
		t.Fatalf("results=%v err=%v", results, err)
	}
}

func TestScenarioStopsWithTheReasonAValueIsMissing(t *testing.T) {
	loaded := fetchStateWorkspace(t, "tei.number.non-negative")
	cases := []struct {
		name            string
		discount, lines string
		rule, reason    string
	}{
		{"the discount is empty", `"numbers": {}, "numberStates": {"sales.order_line.discount": "null"}`, linesComplete,
			fetchStateDiscountRule, "value-null (attribute sales.order_line.discount)"},
		{"the discount was not read", `"numbers": {}, "numberStates": {"sales.order_line.discount": "not-fetched"}`, linesComplete,
			fetchStateDiscountRule, "value-not-fetched (attribute sales.order_line.discount)"},
		{"only a part of the lines was read", discountPresent, linesComplete + `, "coverage": "partial"`,
			fetchStateUniqueRule, "relation-partially-read (relation sales.order.lines / attribute sales.order_line.line_number)"},
		{"a line number was not read", discountPresent, linesComplete + `, "notFetchedKeys": 1`,
			fetchStateUniqueRule, "relation-key-not-fetched (relation sales.order.lines / attribute sales.order_line.line_number)"},
		{"two lines have no number", discountPresent, linesComplete + `, "nullKeys": 2`,
			fetchStateUniqueRule, "relation-key-null (relation sales.order.lines / attribute sales.order_line.line_number)"},
	}
	for _, tc := range cases {
		// An undecided rule cannot be expected as satisfied or as not
		// satisfied: neither expectation lets the step pass.
		for _, expected := range []string{"satisfied", "not-satisfied"} {
			t.Run(tc.name+", expected "+expected, func(t *testing.T) {
				expectDiscount, expectUnique := "satisfied", "satisfied"
				if tc.rule == fetchStateDiscountRule {
					expectDiscount = expected
				} else {
					expectUnique = expected
				}
				results, err := runFetchStateStep(t, loaded, tc.discount, tc.lines, expectDiscount, expectUnique)
				if err == nil {
					t.Fatalf("the step passed with results=%v", results)
				}
				for _, part := range []string{"TEI_VALIDATION_UNDECIDED", tc.rule, tc.reason} {
					if !strings.Contains(err.Error(), part) {
						t.Fatalf("err=%v, want it to contain %q", err, part)
					}
				}
				if results != nil {
					t.Fatalf("a stopped step returned results=%v", results)
				}
			})
		}
	}
}

// What was seen still decides. A duplicate in the part that was read is a
// duplicate, and one line without a number shares it with no other line.
func TestScenarioDecidesFromWhatWasSeen(t *testing.T) {
	loaded := fetchStateWorkspace(t, "tei.number.non-negative")
	cases := []struct {
		name, lines, unique string
	}{
		{"a duplicate in a part", `"keys": ["1", "2", "1"], "coverage": "partial"`, "not-satisfied"},
		{"a duplicate beside unread and empty lines", `"keys": ["7", "7"], "nullKeys": 2, "notFetchedKeys": 1`, "not-satisfied"},
		{"one line without a number", `"keys": ["1", "2"], "nullKeys": 1`, "satisfied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := runFetchStateStep(t, loaded, discountPresent, tc.lines, "satisfied", tc.unique)
			if err != nil {
				t.Fatal(err)
			}
			if results[fetchStateUniqueRule] != tc.unique {
				t.Fatalf("results=%v", results)
			}
		})
	}
}

// With the discount missing, "not negative" and "not zero" would both have
// nothing to answer. The wrong one is still refused: zero decides between
// them only if a missing value is read as zero, and it is not.
func TestScenarioDoesNotReadAMissingDiscountAsZero(t *testing.T) {
	for _, implementation := range []string{"tei.number.non-negative", "tei.number.non-zero"} {
		t.Run(implementation, func(t *testing.T) {
			loaded := fetchStateWorkspace(t, implementation)
			// As zero, the first would answer satisfied and the second
			// not-satisfied. Neither answer may come back.
			for _, expected := range []string{"satisfied", "not-satisfied"} {
				results, err := runFetchStateStep(t, loaded, `"numbers": {}, "numberStates": {"sales.order_line.discount": "null"}`,
					linesComplete, expected, "satisfied")
				if err == nil || !strings.Contains(err.Error(), "TEI_VALIDATION_UNDECIDED") || !strings.Contains(err.Error(), "value-null") {
					t.Fatalf("expected=%s results=%v err=%v", expected, results, err)
				}
			}
		})
	}
}

func TestScenarioRefusesAStateNoRuleRead(t *testing.T) {
	loaded := fetchStateWorkspace(t, "tei.number.non-negative")
	results, err := runFetchStateStep(t, loaded,
		discountPresent+`, "numberStates": {"sales.order_line.unit_price": "not-fetched"}`, linesComplete, "satisfied", "satisfied")
	if err == nil || !strings.Contains(err.Error(), "unused") {
		t.Fatalf("results=%v err=%v", results, err)
	}
}
