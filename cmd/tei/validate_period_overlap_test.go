package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run "the periods of one person in one department do not
// overlap" through the product path: the rule and example records are written
// in TEI Markdown documents and `tei validate` reports an example the rule
// answers differently. The expected answers are written by hand.

const overlapRuleID = "organization.department.members.periods_apart"

const overlapData = "namespace organization\n\n" +
	"data Department {\n    attribute departmentId type std.identifier\n    relation members to organization.membership many\n}\n\n" +
	"data Membership {\n    attribute personId type std.identifier\n" +
	"    attribute startsOn type std.date\n    attribute endsOn type std.date\n}\n"

const overlapRules = "# 所属期間の規則\n\n一つの部署の中で、同じ人の所属期間は重ならない。終了日のない所属は続いている。\n\n" +
	"```tei\nnamespace organization\n\n" +
	"rule organization.department.members.periods_apart:\n" +
	"  organization.membership.starts_on until organization.membership.ends_on or open\n" +
	"    is not overlapping per organization.department.members\n" +
	"    for each organization.membership.person_id\n```\n"

// The marks the tests edit. Each is unique in the document.
const (
	markRehired  = "<!-- tei: example of organization.membership expecting organization.department.members.periods_apart satisfied -->"
	markDoubled  = "<!-- tei: example of organization.membership expecting organization.department.members.periods_apart not-satisfied -->"
	markUnstated = "<!-- tei: example of organization.membership expecting organization.department.members.periods_apart undecided -->"
)

const overlapExamples = "# 所属期間の例\n\n## 出て戻った人と、同じ時期にいる別の人\n\n" +
	markRehired + "\n" +
	"| personId | startsOn | endsOn |\n| --- | --- | --- |\n" +
	"| P001 | 2026-04-01 | 2026-09-30 |\n" +
	"| P001 | 2026-10-01 | |\n" +
	"| P002 | 2026-04-01 | |\n\n" +
	"## 同じ人を、重なる期間で二度登録した誤り\n\n" +
	markDoubled + "\n" +
	"| personId | startsOn | endsOn |\n| --- | --- | --- |\n" +
	"| P003 | 2026-04-01 | 2026-09-30 |\n" +
	"| P003 | 2026-09-30 | |\n\n" +
	"## 人を書いていない所属が同じ時期に二つある\n\n" +
	markUnstated + "\n" +
	"| personId | startsOn | endsOn |\n| --- | --- | --- |\n" +
	"| | 2026-04-01 | 2026-09-30 |\n" +
	"| | 2026-05-01 | 2026-05-31 |\n"

func overlapWorkspace(t *testing.T, rules, examples string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml":                            "workspace: period_overlap\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n",
		"semantic/organization/organization.tei":        overlapData,
		"semantic/organization/membership-rules.tei.md": rules,
		ruleExampleDocument:                             examples,
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

func TestValidateAcceptsPeriodExamplesTheRuleAgreesWith(t *testing.T) {
	code, report := runRuleExampleValidate(t, overlapWorkspace(t, overlapRules, overlapExamples))
	if code != 0 || !report.OK || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

func TestValidateReportsAPeriodRuleThatAnswersDifferentlyFromItsExample(t *testing.T) {
	const expecting = "expecting organization.department.members.periods_apart "
	cases := []struct {
		name             string
		old, replacement string
		// heading occurs once; the finding is the answer word of the mark
		// that follows it.
		heading string
		message string
	}{
		{"moving to the next day is expected to overlap",
			markRehired, strings.Replace(markRehired, "periods_apart satisfied", "periods_apart not-satisfied", 1),
			"## 出て戻った人と、同じ時期にいる別の人\n\n", "expects not-satisfied, but rule " + overlapRuleID + " answers satisfied"},
		{"sharing the last day is expected not to overlap",
			markDoubled, strings.Replace(markDoubled, "periods_apart not-satisfied", "periods_apart satisfied", 1),
			"## 同じ人を、重なる期間で二度登録した誤り\n\n", "expects satisfied, but rule " + overlapRuleID + " answers not-satisfied"},
		{"the second period is moved one day later and the expectation is not",
			"| P003 | 2026-09-30 | |", "| P003 | 2026-10-01 | |",
			"## 同じ人を、重なる期間で二度登録した誤り\n\n", "expects not-satisfied, but rule " + overlapRuleID + " answers satisfied"},
		{"the second period belongs to another person and the expectation is not changed",
			"| P003 | 2026-09-30 | |", "| P004 | 2026-09-30 | |",
			"## 同じ人を、重なる期間で二度登録した誤り\n\n", "expects not-satisfied, but rule " + overlapRuleID + " answers satisfied"},
		{"a person comes back while still there",
			"| P001 | 2026-10-01 | |", "| P001 | 2026-09-01 | |",
			"## 出て戻った人と、同じ時期にいる別の人\n\n", "expects satisfied, but rule " + overlapRuleID + " answers not-satisfied"},
		{"two rows without a person are expected not to overlap",
			markUnstated, strings.Replace(markUnstated, "periods_apart undecided", "periods_apart satisfied", 1),
			"## 人を書いていない所属が同じ時期に二つある\n\n",
			"answers undecided (relation-key-null (relation organization.department.members / attribute organization.membership.person_id))"},
		{"a row without a start",
			"| P002 | 2026-04-01 | |", "| P002 | | |",
			"## 出て戻った人と、同じ時期にいる別の人\n\n",
			"answers undecided (value-null (relation organization.department.members / attribute organization.membership.starts_on))"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := replaceOnce(t, overlapExamples, tc.old, tc.replacement)
			code, report := runRuleExampleValidate(t, overlapWorkspace(t, overlapRules, content))
			if code != 1 || len(report.Diagnostics) != 1 {
				t.Fatalf("code=%d report=%+v", code, report)
			}
			got := report.Diagnostics[0]
			if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || !strings.Contains(got.Diagnostic.Message, tc.message) {
				t.Fatalf("code=%s message=%q, want %q", got.Diagnostic.Code, got.Diagnostic.Message, tc.message)
			}
			at := tc.heading + "<!-- tei: example of organization.membership " + expecting
			if strings.Count(content, at) != 1 {
				t.Fatalf("%q must occur exactly once", at)
			}
			if want := strings.Index(content, at) + len(at); got.Sources[0].Range.Start.Offset != want {
				t.Fatalf("finding at offset %d, want %d", got.Sources[0].Range.Start.Offset, want)
			}
		})
	}
}

// What the rule says about an open period and about people comes from the
// words of the rule. Take a word away and the same examples answer
// differently.
func TestThePeriodRuleMeansWhatItsWordsSay(t *testing.T) {
	t.Run("without or open, a period with no end is not decided", func(t *testing.T) {
		rules := replaceOnce(t, overlapRules, "organization.membership.ends_on or open", "organization.membership.ends_on")
		code, report := runRuleExampleValidate(t, overlapWorkspace(t, rules, overlapExamples))
		// The first two examples have a row without an end.
		if code != 1 || len(report.Diagnostics) != 2 {
			t.Fatalf("code=%d report=%+v", code, report)
		}
		for _, got := range report.Diagnostics {
			if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" ||
				!strings.Contains(got.Diagnostic.Message, "answers undecided (value-null (relation organization.department.members / attribute organization.membership.ends_on))") {
				t.Fatalf("got=%+v", got)
			}
		}
	})
	t.Run("without for each, two people at the same time overlap", func(t *testing.T) {
		rules := replaceOnce(t, overlapRules, "\n    for each organization.membership.person_id", "")
		code, report := runRuleExampleValidate(t, overlapWorkspace(t, rules, overlapExamples))
		// P001 and P002 share days; so do the two rows without a person.
		if code != 1 || len(report.Diagnostics) != 2 {
			t.Fatalf("code=%d report=%+v", code, report)
		}
		for _, got := range report.Diagnostics {
			if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || !strings.Contains(got.Diagnostic.Message, "answers not-satisfied") {
				t.Fatalf("got=%+v", got)
			}
		}
	})
}

// A column the example does not have was not written, so the rule cannot be
// decided from it.
func TestAPeriodExampleWithoutAColumnIsNotDecided(t *testing.T) {
	table := func(expect string) string {
		return "# 人を書いていない例\n\n<!-- tei: example of organization.membership expecting " + overlapRuleID + " " + expect + " -->\n" +
			"| startsOn | endsOn |\n| --- | --- |\n| 2026-04-01 | 2026-09-30 |\n| 2026-05-01 | |\n"
	}
	code, report := runRuleExampleValidate(t, overlapWorkspace(t, overlapRules, table("undecided")))
	if code != 0 || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	code, report = runRuleExampleValidate(t, overlapWorkspace(t, overlapRules, table("not-satisfied")))
	if code != 1 || len(report.Diagnostics) != 1 || !strings.Contains(report.Diagnostics[0].Diagnostic.Message, "relation-key-not-fetched") {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

func TestValidateRefusesAPeriodRuleOrExampleItCannotUse(t *testing.T) {
	t.Run("the rule written as a column", func(t *testing.T) {
		examples := "# 列に書いた例\n\n<!-- tei: example of organization.membership -->\n" +
			"| personId | startsOn | endsOn | " + overlapRuleID + " |\n| --- | --- | --- | --- |\n| P001 | 2026-04-01 | | satisfied |\n"
		code, report := runRuleExampleValidate(t, overlapWorkspace(t, overlapRules, examples))
		if code != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Diagnostic.Code != "PARSE_ERROR" ||
			!strings.Contains(report.Diagnostics[0].Diagnostic.Message, "is about all the rows together") {
			t.Fatalf("code=%d report=%+v", code, report)
		}
	})
	t.Run("a relation to one row", func(t *testing.T) {
		data := replaceOnce(t, overlapData, "relation members to organization.membership many", "relation members to organization.membership one")
		root := overlapWorkspace(t, overlapRules, overlapExamples)
		if err := os.WriteFile(filepath.Join(root, "semantic", "organization", "organization.tei"), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		code, report := runRuleExampleValidate(t, root)
		if code != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Diagnostic.Code != "SEMANTIC_VALIDATION_RELATION_NOT_MANY" ||
			report.Diagnostics[0].Sources[0].ArtifactPath != "semantic/organization/membership-rules.tei.md" ||
			report.Diagnostics[0].Sources[0].Range.Start.Offset != strings.Index(overlapRules, overlapRuleID) {
			t.Fatalf("code=%d report=%+v", code, report)
		}
	})
}
