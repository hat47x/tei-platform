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

// These tests run a rule that compares two dates of one record through the
// product path: the rule is written in a TEI Markdown document, example
// records are written next to it, and `tei validate` reports an example the
// rule answers differently. The expected answers are written by hand.

const dateRuleID = "organization.membership.period_in_order"

const dateRuleData = "namespace organization\n\n" +
	"data Membership {\n    attribute personId type std.identifier\n" +
	"    attribute startsOn type std.date\n    attribute endsOn type std.date\n    attribute registeredAt type std.datetime\n}\n"

const dateRuleRules = "# 所属期間の規則\n\n開始日は、終了日があれば、それより後ではない。終了日のない所属は続いている。\n\n" +
	"```tei\nnamespace organization\n\n" +
	"rule organization.membership.period_in_order:\n" +
	"  organization.membership.starts_on is not after organization.membership.ends_on if present\n```\n"

// The rows the tests edit. Each is unique in the document.
const (
	dateRowOpen     = "| P001 | 2026-04-01 | | satisfied |"
	dateRowOneDay   = "| P002 | 2026-04-01 | 2026-04-01 | satisfied |"
	dateRowReversed = "| P004 | 2026-10-01 | 2026-09-30 | not-satisfied |"
	dateRowNoStart  = "| P005 | | 2026-09-30 | undecided |"
)

const dateRuleExamples = "# 所属期間の例\n\n" +
	"<!-- tei: example of organization.membership -->\n" +
	"| personId | startsOn | endsOn | organization.membership.period_in_order |\n| --- | --- | --- | --- |\n" +
	dateRowOpen + "\n" +
	dateRowOneDay + "\n" +
	"| P003 | 2026-04-01 | 2026-09-30 | satisfied |\n" +
	dateRowReversed + "\n" +
	dateRowNoStart + "\n"

func dateRuleWorkspace(t *testing.T, rules, examples string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml":                            "workspace: date_rule\nformatVersion: 1\nsources:\n  - semantic\n  - applications\nlanguages:\n  - tei.authoring/v1\n",
		"semantic/organization/organization.tei":        dateRuleData,
		"semantic/organization/membership-rules.tei.md": rules,
		"applications/membership-check/application.tei": "application membership_check {\n    uses semantic organization\n" +
			"    uses validation organization.membership.period_in_order\n}\n",
	}
	if examples != "" {
		files[ruleExampleDocument] = examples
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

func TestValidateAcceptsDateExamplesTheRuleAgreesWith(t *testing.T) {
	code, report := runRuleExampleValidate(t, dateRuleWorkspace(t, dateRuleRules, dateRuleExamples))
	if code != 0 || !report.OK || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

func TestValidateReportsADateRuleThatAnswersDifferentlyFromItsExample(t *testing.T) {
	cases := []struct {
		name             string
		old, replacement string
		at               string // the edited text up to the finding
		message          string
	}{
		{"an open period is expected to be out of order",
			dateRowOpen, "| P001 | 2026-04-01 | | not-satisfied |",
			"| P001 | 2026-04-01 | | ", "expects not-satisfied, but rule " + dateRuleID + " answers satisfied"},
		{"an open period is expected to be undecided",
			dateRowOpen, "| P001 | 2026-04-01 | | undecided |",
			"| P001 | 2026-04-01 | | ", "expects undecided, but rule " + dateRuleID + " answers satisfied"},
		{"a period of one day is expected to be out of order",
			dateRowOneDay, "| P002 | 2026-04-01 | 2026-04-01 | not-satisfied |",
			"| P002 | 2026-04-01 | 2026-04-01 | ", "answers satisfied"},
		{"a reversed period is expected to be in order",
			dateRowReversed, "| P004 | 2026-10-01 | 2026-09-30 | satisfied |",
			"| P004 | 2026-10-01 | 2026-09-30 | ", "expects satisfied, but rule " + dateRuleID + " answers not-satisfied"},
		{"the end is moved one day and the expectation is not",
			dateRowReversed, "| P004 | 2026-10-01 | 2026-10-01 | not-satisfied |",
			"| P004 | 2026-10-01 | 2026-10-01 | ", "expects not-satisfied, but rule " + dateRuleID + " answers satisfied"},
		// Compared as texts, 2026-9-30 would not be readable and 2026-10-01
		// sorts before 2026-9-30; the dates are compared as dates.
		{"a period across a month with two digits",
			dateRowReversed, "| P004 | 2026-09-30 | 2026-10-01 | not-satisfied |",
			"| P004 | 2026-09-30 | 2026-10-01 | ", "answers satisfied"},
		{"a record without a start is expected to be in order",
			dateRowNoStart, "| P005 | | 2026-09-30 | satisfied |",
			"| P005 | | 2026-09-30 | ", "answers undecided (value-null (attribute organization.membership.starts_on))"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := replaceOnce(t, dateRuleExamples, tc.old, tc.replacement)
			code, report := runRuleExampleValidate(t, dateRuleWorkspace(t, dateRuleRules, content))
			if code != 1 || len(report.Diagnostics) != 1 {
				t.Fatalf("code=%d report=%+v", code, report)
			}
			got := report.Diagnostics[0]
			if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || !strings.Contains(got.Diagnostic.Message, tc.message) {
				t.Fatalf("code=%s message=%q, want %q", got.Diagnostic.Code, got.Diagnostic.Message, tc.message)
			}
			if strings.Count(content, tc.at) != 1 {
				t.Fatalf("%q must occur exactly once", tc.at)
			}
			if want := strings.Index(content, tc.at) + len(tc.at); got.Sources[0].Range.Start.Offset != want {
				t.Fatalf("finding at offset %d, want %d", got.Sources[0].Range.Start.Offset, want)
			}
		})
	}
}

// `if present` is what makes an open period satisfy the rule. Without the
// words, the same record is not decided: nothing is assumed about a period
// that has no end.
func TestWithoutIfPresentAnOpenPeriodIsNotDecided(t *testing.T) {
	rules := replaceOnce(t, dateRuleRules, "organization.membership.ends_on if present", "organization.membership.ends_on")
	code, report := runRuleExampleValidate(t, dateRuleWorkspace(t, rules, dateRuleExamples))
	if code != 1 || len(report.Diagnostics) != 1 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	got := report.Diagnostics[0]
	want := strings.Index(dateRuleExamples, dateRowOpen) + len("| P001 | 2026-04-01 | | ")
	if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || got.Sources[0].Range.Start.Offset != want ||
		!strings.Contains(got.Diagnostic.Message, "answers undecided (value-null (attribute organization.membership.ends_on))") {
		t.Fatalf("got=%+v, want a mismatch at offset %d", got, want)
	}
}

func TestValidateRefusesADateItCannotRead(t *testing.T) {
	cases := []struct {
		name, old, replacement, at, message string
	}{
		{"a date with slashes", dateRowOneDay, "| P002 | 2026/04/01 | 2026-04-01 | satisfied |", "2026/04/01", "item startsOn is a date"},
		{"a day the calendar does not have", dateRowOneDay, "| P002 | 2026-02-30 | 2026-04-01 | satisfied |", "2026-02-30", "item startsOn is a date"},
		{"a date with a time", dateRowReversed, "| P004 | 2026-10-01 | 2026-09-30T00:00 | not-satisfied |", "2026-09-30T00:00", "item endsOn is a date"},
		{"a word for no end", dateRowOpen, "| P001 | 2026-04-01 | 未定 | satisfied |", "未定", "item endsOn is a date"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := replaceOnce(t, dateRuleExamples, tc.old, tc.replacement)
			code, report := runRuleExampleValidate(t, dateRuleWorkspace(t, dateRuleRules, content))
			if code != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Diagnostic.Code != "PARSE_ERROR" ||
				!strings.Contains(report.Diagnostics[0].Diagnostic.Message, tc.message) ||
				report.Diagnostics[0].Sources[0].Range.Start.Offset != strings.Index(content, tc.at) {
				t.Fatalf("code=%d report=%+v", code, report)
			}
		})
	}
}

// A moment in time is not a calendar date. The rule is refused where it is
// written, with the validator's code, and the examples are not asked.
func TestValidateRefusesNotAfterOnAMomentInTime(t *testing.T) {
	rules := replaceOnce(t, dateRuleRules, "organization.membership.ends_on if present", "organization.membership.registered_at if present")
	code, report := runRuleExampleValidate(t, dateRuleWorkspace(t, rules, ""))
	if code != 1 || len(report.Diagnostics) != 1 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	got := report.Diagnostics[0]
	if got.Diagnostic.Code != "SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_DATE_ATTRIBUTES" || len(got.Sources) != 1 ||
		got.Sources[0].ArtifactPath != "semantic/organization/membership-rules.tei.md" ||
		got.Sources[0].Range.Start.Offset != strings.Index(rules, dateRuleID) {
		t.Fatalf("got=%+v", got)
	}
}

// The same rule in a scenario, with dates stated by a Fixture parsed from
// JSON as the command parses it.
func TestScenarioDecidesADateRule(t *testing.T) {
	loaded, err := workspace.Load(dateRuleWorkspace(t, dateRuleRules, ""))
	if err != nil {
		t.Fatal(err)
	}
	run := func(fixtureJSON, expected string) (map[string]string, error) {
		fixture, err := scenariotest.ParseValidationFixture([]byte(fixtureJSON))
		if err != nil {
			t.Fatalf("the Fixture of this test is not readable: %v", err)
		}
		step := scenariotest.Step{ID: "check", Operation: "evaluate-validations", ExpectValidations: map[string]string{dateRuleID: expected}}
		results, _, err := executeScenarioValidations(driver.NewResolver(loaded.World), "membership_check", step, fixture,
			scenarioImplementationSelections(loaded.OperatorImplementations), validationeval.BuiltinImplementations())
		return results, err
	}
	const head = `{"numbers": {}, "relationKeys": [], `
	decided := []struct{ name, fixture, expected string }{
		{"in order", head + `"dates": {"organization.membership.starts_on": "2026-04-01", "organization.membership.ends_on": "2026-09-30"}}`, "satisfied"},
		{"reversed", head + `"dates": {"organization.membership.starts_on": "2026-10-01", "organization.membership.ends_on": "2026-09-30"}}`, "not-satisfied"},
		{"no end", head + `"dates": {"organization.membership.starts_on": "2026-04-01"}, "dateStates": {"organization.membership.ends_on": "null"}}`, "satisfied"},
	}
	for _, tc := range decided {
		results, err := run(tc.fixture, tc.expected)
		if err != nil || !reflect.DeepEqual(results, map[string]string{dateRuleID: tc.expected}) {
			t.Fatalf("%s: results=%v err=%v", tc.name, results, err)
		}
	}
	stopped := []struct{ name, fixture, mention string }{
		{"the end was not read", head + `"dates": {"organization.membership.starts_on": "2026-04-01"}, "dateStates": {"organization.membership.ends_on": "not-fetched"}}`,
			"TEI_VALIDATION_UNDECIDED: Canonical Validation " + dateRuleID + " is neither satisfied nor not-satisfied: value-not-fetched (attribute organization.membership.ends_on)"},
		{"no start", head + `"dates": {"organization.membership.ends_on": "2026-09-30"}, "dateStates": {"organization.membership.starts_on": "null"}}`,
			"value-null (attribute organization.membership.starts_on)"},
		{"a date nobody stated", head + `"dates": {"organization.membership.starts_on": "2026-04-01"}}`, "missing Validation date Attribute organization.membership.ends_on"},
	}
	for _, tc := range stopped {
		// A rule that was not decided cannot be expected either way.
		for _, expected := range []string{"satisfied", "not-satisfied"} {
			results, err := run(tc.fixture, expected)
			if err == nil || !strings.Contains(err.Error(), tc.mention) || results != nil {
				t.Fatalf("%s expected=%s: results=%v err=%v", tc.name, expected, results, err)
			}
		}
	}

	// A date no rule read is refused even though the rule itself is decided
	// as expected.
	results, err := run(head+`"dates": {"organization.membership.starts_on": "2026-04-01", "organization.membership.ends_on": "2026-09-30", "organization.membership.left_on": "2026-09-30"}}`, "satisfied")
	if err == nil || !strings.Contains(err.Error(), "unused date inputs") || results != nil {
		t.Fatalf("results=%v err=%v", results, err)
	}
}
