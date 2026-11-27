package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A rule whose operator reads a column of days is written, selected and checked
// through the product path: the declaration and the rule are in a TEI Markdown
// document, the operator's example is a table of days next to it, and
// `tei validate` reports an example the selected implementation answers
// differently. The expected answers are written by hand from what the operator
// means — every row's day is not before its deadline's day.

const dayRowData = "namespace organization\n\n" +
	"data Department {\n    attribute departmentId type std.identifier\n" +
	"    relation members to organization.membership many\n}\n\n" +
	"data Membership {\n    attribute personId type std.identifier\n" +
	"    attribute startsOn type std.date\n    attribute endsOn type std.date\n}\n"

const dayRowRules = "# 所属の開始日\n\nどの所属も、終了日があるならそれより前には始まらない。\n\n" +
	"```tei\nnamespace organization\n\n" +
	"operator organization.every_not_before:\n" +
	"  takes value: date of each row\n" +
	"  takes deadline: date\n\n" +
	"rule organization.department.members.starts_not_before_end:\n" +
	"  organization.membership.starts_on is organization.every_not_before deadline: organization.membership.ends_on per organization.department.members\n```\n"

// The rows the tests edit. Each is unique in the document.
const (
	dayRowSatisfied    = "| 2026-04-01 | 2026-04-01 |"
	dayRowNotSatisfied = "| 2026-03-31 | 2026-04-05 |"
)

func dayRowExamples(satisfied, notSatisfied string) string {
	return "# 開始日の例\n\n" +
		"<!-- tei: cases of organization.every_not_before expecting satisfied -->\n" +
		"| value | deadline |\n| --- | --- |\n" +
		satisfied + "\n" +
		"| 2026-04-10 | 2026-04-05 |\n\n" +
		"<!-- tei: cases of organization.every_not_before expecting not-satisfied -->\n" +
		"| value | deadline |\n| --- | --- |\n" +
		notSatisfied + "\n"
}

const dayRowSelection = "implementations:\n  - operator: organization.every_not_before\n    use: tei.day.every-not-before\n"

func dayRowWorkspace(t *testing.T, manifestTail, examples string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml":                            ruleExampleManifest + manifestTail,
		"semantic/organization/organization.tei":        dayRowData,
		"semantic/organization/membership-rules.tei.md": dayRowRules,
		"applications/membership-check/application.tei": "application membership_check {\n    uses semantic organization\n" +
			"    uses validation organization.department.members.starts_not_before_end\n}\n",
	}
	if examples != "" {
		files[dayRowExampleDocument] = examples
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

const dayRowExampleDocument = "semantic/organization/day-rows.tei.md"

// The two examples the operator states are asked of the implementation this tei
// carries, and it answers both as written.
func TestValidateAcceptsTheRowExamplesOfADayOperator(t *testing.T) {
	code, report := runRuleExampleValidate(t, dayRowWorkspace(t, dayRowSelection,
		dayRowExamples(dayRowSatisfied, dayRowNotSatisfied)))
	if code != 0 || !report.OK || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

// Changing only the answer an example states stops validate: the example now
// says something the selected implementation contradicts, and no row of a real
// database is read for it.
func TestValidateRefusesADayRowExampleWhoseAnswerChanged(t *testing.T) {
	code, report := runRuleExampleValidate(t, dayRowWorkspace(t, dayRowSelection,
		dayRowExamples(dayRowNotSatisfied, dayRowNotSatisfied)))
	if code != 1 || len(report.Diagnostics) != 1 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	got := report.Diagnostics[0]
	if got.Diagnostic.Code != "OPERATOR_SELECTION_UNRESOLVED" ||
		!strings.Contains(got.Diagnostic.Message, "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") {
		t.Fatalf("code=%q message=%q", got.Diagnostic.Code, got.Diagnostic.Message)
	}
	if !strings.Contains(got.Diagnostic.Message, "not-satisfied") {
		t.Fatalf("the refusal does not say what the example states: %q", got.Diagnostic.Message)
	}
}

// A column of days is not decided by the implementation that decides a column
// of numbers: the selection is refused by name and by type, before any row is
// read.
func TestValidateRefusesANumberImplementationForAColumnOfDays(t *testing.T) {
	const wrongType = "implementations:\n  - operator: organization.every_not_before\n    use: tei.number.every-at-most\n"
	code, report := runRuleExampleValidate(t, dayRowWorkspace(t, wrongType, ""))
	if code != 1 || len(report.Diagnostics) == 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	message := report.Diagnostics[0].Diagnostic.Message
	if !strings.Contains(message, "date") || !strings.Contains(message, "number") {
		t.Fatalf("the refusal does not name the two types: %q", message)
	}
}

// The declaration of the operator is read with its type and its range, so a
// rule about a column of days reaches the row path instead of being refused as
// a column with no reader.
func TestValidateAcceptsTheDeclarationOfAColumnOfDays(t *testing.T) {
	code, report := runRuleExampleValidate(t, dayRowWorkspace(t, dayRowSelection, ""))
	if code != 0 || !report.OK {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}
