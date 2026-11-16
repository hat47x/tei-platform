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

// The public synthetic example of the field-validation slice: one number rule
// (a quantity must be positive) and one conditional requirement (a discount
// reason must be present when a discount is present). Both go through the
// product path: the rules are written in a TEI Markdown document, the example
// records sit next to it, and `tei validate` reports an example the rule
// answers differently.
//
// The expected answers are written by hand from the meaning of each rule. They
// are not read out of the implementation.

const quantityRuleID = "sales.order_line.quantity_positive"
const reasonRuleID = "sales.order_line.discount_reason_required"

const fieldValidationData = "namespace sales\n\n" +
	"data OrderLine {\n" +
	"    attribute lineNumber type std.identifier\n" +
	"    attribute quantity type std.decimal\n" +
	"    attribute discount type std.decimal\n" +
	"    attribute discountReason type std.text\n" +
	"}\n"

const fieldValidationRules = "# 譏守ｴｰ縺ｮ鬆・岼縺ｮ隕丞援\n\n" +
	"謨ｰ驥上・豁｣縺ｮ謨ｰ縺ｧ譖ｸ縺上ょ､蠑輔″繧呈嶌縺・◆陦後↓縺ｯ縲√◎縺ｮ逅・罰繧呈嶌縺上・n\n" +
	"```tei\nnamespace sales\n\n" +
	"rule sales.order_line.quantity_positive:\n" +
	"  sales.order_line.quantity is positive\n\n" +
	"rule sales.order_line.discount_reason_required:\n" +
	"  sales.order_line.discount_reason is required if sales.order_line.discount\n```\n"

// The rows the tests edit. Each is unique in the document.
//
// Zero is a value, not an absence. A discount written as 0 is a discount the
// rule speaks about, so it still needs a reason. A row with no discount at all
// leaves the cell empty, and that is the case the rule does not speak about.
const (
	// One row per case of the number rule: the ordinary value, zero and a
	// negative value. These rows have no discount, so the reason rule is
	// satisfied whatever the reason cell holds.
	rowQuantityOK   = "| L1 | 2 | | | satisfied | satisfied |"
	rowQuantityZero = "| L2 | 0 | | | not-satisfied | satisfied |"
	rowQuantityNeg  = "| L3 | -1 | | | not-satisfied | satisfied |"
	// A discount is written and the reason is not.
	rowReasonMissing = "| L4 | 2 | 10 | | satisfied | not-satisfied |"
	// A discount written as zero is still a value that needs a reason.
	rowZeroDiscount = "| L5 | 2 | 0 | | satisfied | not-satisfied |"
)

const fieldValidationExamples = "# 譏守ｴｰ縺ｮ萓欺n\n" +
	"<!-- tei: example of sales.order_line -->\n" +
	"| lineNumber | quantity | discount | discountReason | " + quantityRuleID + " | " + reasonRuleID + " |\n" +
	"| --- | --- | --- | --- | --- | --- |\n" +
	rowQuantityOK + "\n" +
	rowQuantityZero + "\n" +
	rowQuantityNeg + "\n" +
	rowReasonMissing + "\n" +
	rowZeroDiscount + "\n"

// fieldValidationWorkspace writes one Workspace. The applications root is
// declared only when an Application is written, because a declared source root
// with nothing in it is reported as a finding.
func fieldValidationWorkspace(t *testing.T, rules, examples string, withApplication bool) string {
	t.Helper()
	root := t.TempDir()
	manifest := "workspace: field_validation\nformatVersion: 1\nsources:\n  - semantic\n"
	if withApplication {
		manifest += "  - applications\n"
	}
	manifest += "languages:\n  - tei.authoring/v1\n"
	files := map[string]string{
		"tei.workspace.yaml":                     manifest,
		"semantic/sales/sales.tei":               fieldValidationData,
		"semantic/sales/order-line-rules.tei.md": rules,
	}
	if examples != "" {
		files["semantic/sales/order-line-examples.tei.md"] = examples
	}
	if withApplication {
		files["applications/line-check/application.tei"] = "application line_check {\n    uses semantic sales\n" +
			"    uses validation " + quantityRuleID + "\n    uses validation " + reasonRuleID + "\n}\n"
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

// The positive control: the example and the rules agree, so nothing is
// reported. This is what the whole change is for: the two conditions live in
// the data model as rules, not in application code.
func TestValidateAcceptsFieldValidationExamples(t *testing.T) {
	code, report := runRuleExampleValidate(t, fieldValidationWorkspace(t, fieldValidationRules, fieldValidationExamples, false))
	if code != 0 || !report.OK || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

// Each negative case changes one expectation in the document and names,
// without the parser, the byte offset the finding must point at. The message
// names which rule answered and how.
func TestValidateReportsFieldValidationMismatches(t *testing.T) {
	cases := []struct {
		name       string
		old, new   string
		at         string
		wantRule   string
		wantAnswer string
	}{
		{
			name: "a zero quantity is expected to be positive",
			old:  rowQuantityZero, new: "| L2 | 0 | | | satisfied | satisfied |",
			at: "| L2 | 0 | | | ", wantRule: quantityRuleID, wantAnswer: "expects satisfied, but rule " + quantityRuleID + " answers not-satisfied",
		},
		{
			name: "a negative quantity is expected to be positive",
			old:  rowQuantityNeg, new: "| L3 | -1 | | | satisfied | satisfied |",
			at: "| L3 | -1 | | | ", wantRule: quantityRuleID, wantAnswer: "expects satisfied, but rule " + quantityRuleID + " answers not-satisfied",
		},
		{
			name: "a missing reason is expected to be present",
			old:  rowReasonMissing, new: "| L4 | 2 | 10 | | satisfied | satisfied |",
			at: "| L4 | 2 | 10 | | satisfied | ", wantRule: reasonRuleID, wantAnswer: "expects satisfied, but rule " + reasonRuleID + " answers not-satisfied",
		},
		{
			// A discount written as zero is a value, so the rule speaks about
			// the record. Reading zero as "no discount" is the mistake this
			// row catches.
			name: "a zero discount is expected to need no reason",
			old:  rowZeroDiscount, new: "| L5 | 2 | 0 | | satisfied | satisfied |",
			at: "| L5 | 2 | 0 | | satisfied | ", wantRule: reasonRuleID, wantAnswer: "expects satisfied, but rule " + reasonRuleID + " answers not-satisfied",
		},
		{
			// A record without a discount is not spoken about, so the rule is
			// satisfied even though the reason is empty. Expecting a rejection
			// here is the mistake the rule prevents.
			name: "a missing reason without a discount is expected to be rejected",
			old:  rowQuantityOK, new: "| L1 | 2 | | | satisfied | not-satisfied |",
			at: "| L1 | 2 | | | satisfied | ", wantRule: reasonRuleID, wantAnswer: "expects not-satisfied, but rule " + reasonRuleID + " answers satisfied",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := replaceOnce(t, fieldValidationExamples, tc.old, tc.new)
			code, report := runRuleExampleValidate(t, fieldValidationWorkspace(t, fieldValidationRules, content, false))
			if code != 1 || len(report.Diagnostics) != 1 {
				t.Fatalf("code=%d report=%+v", code, report)
			}
			got := report.Diagnostics[0]
			if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || !strings.Contains(got.Diagnostic.Message, tc.wantAnswer) {
				t.Fatalf("code=%s message=%q, want %q", got.Diagnostic.Code, got.Diagnostic.Message, tc.wantAnswer)
			}
			if got.Diagnostic.Subject == nil || got.Diagnostic.Subject.ID != tc.wantRule {
				t.Fatalf("subject=%+v, want rule %s", got.Diagnostic.Subject, tc.wantRule)
			}
			if want := strings.Index(content, tc.at) + len(tc.at); got.Sources[0].Range.Start.Offset != want {
				t.Fatalf("finding at offset %d, want %d", got.Sources[0].Range.Start.Offset, want)
			}
		})
	}
}

// The difference between "the record holds no value" and "the attribute was
// not read" is kept. Dropping the discount column makes the trigger unknown,
// so a rule that is about that trigger cannot be decided: the answer is
// undecided, not satisfied and not not-satisfied.
func TestAFieldNotWrittenIsNotAnEmptyValue(t *testing.T) {
	table := func(expect string) string {
		return "# 蛟､蠑輔″縺ｮ蛻励ｒ譖ｸ縺・※縺・↑縺・ｾ欺n\n<!-- tei: example of sales.order_line -->\n" +
			"| lineNumber | quantity | discountReason | " + quantityRuleID + " | " + reasonRuleID + " |\n" +
			"| --- | --- | --- | --- | --- |\n" +
			"| L1 | 2 | | satisfied | " + expect + " |\n"
	}
	code, report := runRuleExampleValidate(t, fieldValidationWorkspace(t, fieldValidationRules, table("undecided"), false))
	if code != 0 || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	code, report = runRuleExampleValidate(t, fieldValidationWorkspace(t, fieldValidationRules, table("satisfied"), false))
	if code != 1 || len(report.Diagnostics) != 1 ||
		!strings.Contains(report.Diagnostics[0].Diagnostic.Message, "answers undecided (value-not-fetched (attribute sales.order_line.discount))") {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

// A validation that cannot name its parts is refused at the rule, and the
// examples are not asked. This keeps a broken definition from being checked
// against examples, which would report the same cause twice.
func TestValidateRefusesARequiredIfOnAnotherData(t *testing.T) {
	rules := replaceOnce(t, fieldValidationRules,
		"sales.order_line.discount_reason is required if sales.order_line.discount",
		"sales.order_line.discount_reason is required if sales.order_line.line_number")
	// The same Data, so this one is accepted: the control for the refusal
	// below.
	if code, report := runRuleExampleValidate(t, fieldValidationWorkspace(t, rules, "", false)); code != 0 {
		t.Fatalf("control: code=%d report=%+v", code, report)
	}

	// Add a second Data and point the trigger at it. Everything the number
	// rule needs stays declared, so the only finding is the cross-Data one.
	other := fieldValidationData + "\n" +
		"data OrderHeader {\n    attribute note type std.text\n}\n"
	rules = replaceOnce(t, fieldValidationRules,
		"sales.order_line.discount_reason is required if sales.order_line.discount",
		"sales.order_line.discount_reason is required if sales.order_header.note")
	root := fieldValidationWorkspace(t, rules, "", false)
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "sales.tei"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	code, report := runRuleExampleValidate(t, root)
	if code != 1 || len(report.Diagnostics) != 1 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	got := report.Diagnostics[0]
	if got.Diagnostic.Code != "SEMANTIC_VALIDATION_REQUIRED_IF_REQUIRES_SAME_DATA" {
		t.Fatalf("code=%s message=%q", got.Diagnostic.Code, got.Diagnostic.Message)
	}
}

// The same two rules in a scenario, with presence and numbers stated by a
// Fixture parsed from JSON as the command parses it. This is the headless
// path: no example document, only a Fixture and an expected answer.
func TestScenarioDecidesFieldValidationRules(t *testing.T) {
	// A run needs an Application that selects the rules, so the selection goes
	// through the existing `uses validation` path rather than a rule list
	// invented by the Scenario.
	root := fieldValidationWorkspace(t, fieldValidationRules, "", true)
	application := "application line_check {\n    uses semantic sales\n" +
		"    uses validation " + quantityRuleID + "\n    uses validation " + reasonRuleID + "\n}\n"
	if err := os.MkdirAll(filepath.Join(root, "applications", "line-check"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "applications", "line-check", "application.tei"), []byte(application), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load with an Application: %v", err)
	}

	run := func(fixtureJSON string, expected map[string]string) (map[string]string, error) {
		fixture, err := scenariotest.ParseValidationFixture([]byte(fixtureJSON))
		if err != nil {
			t.Fatalf("the Fixture of this test is not readable: %v", err)
		}
		step := scenariotest.Step{ID: "check", Operation: "evaluate-validations", ExpectValidations: expected}
		results, _, err := executeScenarioValidations(driver.NewResolver(loaded.World), "line_check", step, fixture,
			scenarioImplementationSelections(loaded.OperatorImplementations), validationeval.BuiltinImplementations())
		return results, err
	}
	const head = `{"numbers": {"sales.order_line.quantity": "2"}, "relationKeys": [], `

	// The satisfied pair: a positive quantity, and a discount present with its
	// reason present.
	t.Run("both rules hold", func(t *testing.T) {
		fixture := head + `"presenceStates": {"sales.order_line.discount": "present", "sales.order_line.discount_reason": "present"}}`
		results, err := run(fixture, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "satisfied"})
		if err != nil || !reflect.DeepEqual(results, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "satisfied"}) {
			t.Fatalf("results=%v err=%v", results, err)
		}
	})
	// A zero quantity is a business rejection; the reason rule is untouched.
	t.Run("a zero quantity is rejected", func(t *testing.T) {
		fixture := `{"numbers": {"sales.order_line.quantity": "0"}, "relationKeys": [], ` +
			`"presenceStates": {"sales.order_line.discount": "present", "sales.order_line.discount_reason": "present"}}`
		results, err := run(fixture, map[string]string{quantityRuleID: "not-satisfied", reasonRuleID: "satisfied"})
		if err != nil || !reflect.DeepEqual(results, map[string]string{quantityRuleID: "not-satisfied", reasonRuleID: "satisfied"}) {
			t.Fatalf("results=%v err=%v", results, err)
		}
	})
	// A discount without a reason is the conditional rejection.
	t.Run("a discount without a reason is rejected", func(t *testing.T) {
		fixture := head + `"presenceStates": {"sales.order_line.discount": "present", "sales.order_line.discount_reason": "null"}}`
		results, err := run(fixture, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "not-satisfied"})
		if err != nil || !reflect.DeepEqual(results, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "not-satisfied"}) {
			t.Fatalf("results=%v err=%v", results, err)
		}
	})
	// No discount: the reason rule does not speak, so an empty reason passes.
	t.Run("no discount means the reason rule does not speak", func(t *testing.T) {
		fixture := head + `"presenceStates": {"sales.order_line.discount": "null", "sales.order_line.discount_reason": "null"}}`
		results, err := run(fixture, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "satisfied"})
		if err != nil || !reflect.DeepEqual(results, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "satisfied"}) {
			t.Fatalf("results=%v err=%v", results, err)
		}
	})

	// An unread presence value is not decided, and cannot be expected either
	// way.
	t.Run("an unread presence is not decided", func(t *testing.T) {
		fixture := head + `"presenceStates": {"sales.order_line.discount": "present", "sales.order_line.discount_reason": "not-fetched"}}`
		for _, expected := range []string{"satisfied", "not-satisfied"} {
			results, err := run(fixture, map[string]string{quantityRuleID: "satisfied", reasonRuleID: expected})
			if err == nil || !strings.Contains(err.Error(), "TEI_VALIDATION_UNDECIDED") || results != nil {
				t.Fatalf("expected=%s results=%v err=%v", expected, results, err)
			}
		}
	})
	// The two kinds of "no value" stay apart: a missing trigger is decided,
	// and a missing presence statement is a Fixture error.
	t.Run("a presence value nobody stated is refused", func(t *testing.T) {
		fixture := head + `"presenceStates": {"sales.order_line.discount_reason": "present"}}`
		results, err := run(fixture, map[string]string{quantityRuleID: "satisfied", reasonRuleID: "satisfied"})
		if err == nil || !strings.Contains(err.Error(), "missing Validation presence state for Attribute sales.order_line.discount") || results != nil {
			t.Fatalf("results=%v err=%v", results, err)
		}
	})
}
