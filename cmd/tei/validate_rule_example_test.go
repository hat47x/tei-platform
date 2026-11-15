package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run the CLI validate operation on a Workspace that states
// example records next to its rules. The expected answers in the examples are
// written by hand from the meaning of the rules. Each negative case changes
// one cell of the document and names, without the parser, the byte offset the
// finding must point at.

const ruleExampleDocument = "semantic/organization/membership-examples.tei.md"

const ruleExampleManifest = "workspace: rule_examples\nformatVersion: 1\nsources:\n  - semantic\n" +
	"languages:\n  - tei.authoring/v1\n"

const ruleExampleSelection = "implementations:\n  - operator: organization.non_negative\n    use: tei.number.non-negative\n"

const ruleExampleData = "namespace organization\n\n" +
	"data Department {\n    attribute departmentId type std.identifier\n    attribute headcount type std.integer\n" +
	"    relation members to organization.membership many\n}\n\n" +
	"data Membership {\n    attribute personId type std.identifier\n    attribute seatNumber type std.integer\n" +
	"    attribute allocation type std.decimal\n    attribute startsOn type std.datetime\n    attribute endsOn type std.datetime\n}\n"

const ruleExampleRules = "# 所属の規則\n\n兼務の割合は正の数で書く。一つの部署に同じ人を二度登録しない。席番号は部署の中で重ならない。\n\n" +
	"```tei\nnamespace organization\n\noperator organization.non_negative: takes number\n\n" +
	"rule organization.membership.allocation_positive:\n  organization.membership.allocation is positive\n\n" +
	"rule organization.membership.allocation_not_negative:\n  organization.membership.allocation is organization.non_negative\n\n" +
	"rule organization.department.headcount_positive:\n  organization.department.headcount is positive\n\n" +
	"rule organization.department.members.person_unique:\n  organization.membership.person_id is unique per organization.department.members\n\n" +
	"rule organization.department.members.seat_unique:\n  organization.membership.seat_number is unique per organization.department.members\n```\n"

// The rows the tests edit. Each is unique in the document.
const (
	rowP001   = "| P001 | 1 | 2026-04-01 | | satisfied | satisfied |"
	rowP003   = "| P003 | 0 | 2026-10-01 | | not-satisfied | satisfied |"
	rowP004   = "| P004 | | 2026-10-01 | | undecided | undecided |"
	markKenmu = "<!-- tei: example of organization.membership expecting organization.department.members.person_unique satisfied -->"
	markTwice = "<!-- tei: example of organization.membership expecting organization.department.members.person_unique not-satisfied -->"
	markSeat  = "<!-- tei: example of organization.membership expecting organization.department.members.seat_unique not-satisfied -->"
)

const ruleExampleExamples = "# 所属の例\n\n## 兼務している人\n\n" +
	"営業部の所属である。P002は開発部と兼務し、営業部の割合は0.4である。終了日が空の行は、終了日がまだない。\n\n" +
	markKenmu + "\n" +
	"| personId | allocation | startsOn | endsOn | organization.membership.allocation_positive | organization.membership.allocation_not_negative |\n" +
	"| --- | --- | --- | --- | --- | --- |\n" +
	rowP001 + "\n" +
	"| P002 | 0.4 | 2026-04-01 | 2026-09-30 | satisfied | satisfied |\n" +
	rowP003 + "\n" +
	rowP004 + "\n\n" +
	"## 同じ人を二度登録した誤り\n\n" +
	markTwice + "\n" +
	"| personId | allocation |\n| --- | --- |\n| P900 | 0.5 |\n| P900 | 0.5 |\n\n" +
	"## 席番号は数として比べる\n\n" +
	markSeat + "\n" +
	"| seatNumber |\n| --- |\n| 12 |\n| 12.0 |\n"

func ruleExampleWorkspace(t *testing.T, manifestTail, examples string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml":                            ruleExampleManifest + manifestTail,
		"semantic/organization/organization.tei":        ruleExampleData,
		"semantic/organization/membership-rules.tei.md": ruleExampleRules,
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

type ruleExampleReport struct {
	OK               bool   `json:"ok"`
	WorldFingerprint string `json:"worldFingerprint"`
	Diagnostics      []struct {
		Diagnostic struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Subject *struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"subject"`
		} `json:"diagnostic"`
		Sources []struct {
			ArtifactPath string `json:"artifactPath"`
			Range        struct {
				Start struct {
					Offset int `json:"offset"`
					Line   int `json:"line"`
					Column int `json:"column"`
				} `json:"start"`
				End struct {
					Offset int `json:"offset"`
				} `json:"end"`
			} `json:"range"`
		} `json:"sources"`
	} `json:"diagnostics"`
}

func runRuleExampleValidate(t *testing.T, root string) (int, ruleExampleReport) {
	t.Helper()
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil {
		t.Fatalf("validate: %v output=%s", err, &out)
	}
	var report ruleExampleReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("%v: %s", err, &out)
	}
	return code, report
}

// replaceOnce edits the examples and fails the test if the edit is ambiguous.
func replaceOnce(t *testing.T, content, old, replacement string) string {
	t.Helper()
	if strings.Count(content, old) != 1 {
		t.Fatalf("%q must occur exactly once", old)
	}
	return strings.Replace(content, old, replacement, 1)
}

func TestValidateAcceptsExamplesTheRulesAgreeWith(t *testing.T) {
	code, report := runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, ruleExampleExamples))
	if code != 0 || !report.OK || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

// Examples define nothing: the same definitions with and without the example
// document are the same World.
func TestExamplesDoNotChangeTheWorld(t *testing.T) {
	_, with := runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, ruleExampleExamples))
	_, without := runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, ""))
	if with.WorldFingerprint == "" || with.WorldFingerprint != without.WorldFingerprint {
		t.Fatalf("world fingerprint with examples=%q, without=%q", with.WorldFingerprint, without.WorldFingerprint)
	}
}

func TestValidateReportsARuleThatAnswersDifferentlyFromItsExample(t *testing.T) {
	cases := []struct {
		name             string
		old, replacement string
		// at is the edited text up to the place the finding must start. It
		// occurs once in the edited document.
		at      string
		rule    string
		message string
	}{
		{"zero is expected to be positive",
			rowP003, "| P003 | 0 | 2026-10-01 | | satisfied | satisfied |",
			"| P003 | 0 | 2026-10-01 | | ", "organization.membership.allocation_positive",
			"the example expects satisfied, but rule organization.membership.allocation_positive answers not-satisfied"},
		{"one is expected not to be positive",
			rowP001, "| P001 | 1 | 2026-04-01 | | not-satisfied | satisfied |",
			"| P001 | 1 | 2026-04-01 | | ", "organization.membership.allocation_positive",
			"expects not-satisfied, but rule organization.membership.allocation_positive answers satisfied"},
		{"an empty allocation is expected not to be positive",
			rowP004, "| P004 | | 2026-10-01 | | not-satisfied | undecided |",
			"| P004 | | 2026-10-01 | | ", "organization.membership.allocation_positive",
			"answers undecided (value-null (attribute organization.membership.allocation))"},
		{"an empty allocation is expected not to be negative, as zero would be",
			rowP004, "| P004 | | 2026-10-01 | | undecided | satisfied |",
			"| P004 | | 2026-10-01 | | undecided | ", "organization.membership.allocation_not_negative",
			"expects satisfied, but rule organization.membership.allocation_not_negative answers undecided"},
		{"the value is changed and the expectation is not",
			rowP003, "| P003 | 0.1 | 2026-10-01 | | not-satisfied | satisfied |",
			"| P003 | 0.1 | 2026-10-01 | | ", "organization.membership.allocation_positive",
			"expects not-satisfied, but rule organization.membership.allocation_positive answers satisfied"},
		{"different people are expected to be a duplicate",
			markKenmu, strings.Replace(markKenmu, "person_unique satisfied", "person_unique not-satisfied", 1) + " ",
			"営業部の割合は0.4である。終了日が空の行は、終了日がまだない。\n\n<!-- tei: example of organization.membership expecting organization.department.members.person_unique ",
			"organization.department.members.person_unique",
			"expects not-satisfied, but rule organization.department.members.person_unique answers satisfied"},
		{"the same person twice is expected to be unique",
			markTwice, strings.Replace(markTwice, "person_unique not-satisfied", "person_unique satisfied", 1),
			"## 同じ人を二度登録した誤り\n\n<!-- tei: example of organization.membership expecting organization.department.members.person_unique ",
			"organization.department.members.person_unique",
			"expects satisfied, but rule organization.department.members.person_unique answers not-satisfied"},
		{"people whose ids differ only in case are expected to be a duplicate",
			"| P900 | 0.5 |\n| P900 | 0.5 |\n", "| P900 | 0.5 |\n| p900 | 0.5 |\n",
			"## 同じ人を二度登録した誤り\n\n<!-- tei: example of organization.membership expecting organization.department.members.person_unique ",
			"organization.department.members.person_unique",
			"expects not-satisfied, but rule organization.department.members.person_unique answers satisfied"},
		{"two people without an id are expected to be unique",
			"| P900 | 0.5 |\n| P900 | 0.5 |\n", "| | 0.5 |\n| | 0.5 |\n",
			"## 同じ人を二度登録した誤り\n\n<!-- tei: example of organization.membership expecting organization.department.members.person_unique ",
			"organization.department.members.person_unique",
			"answers undecided (relation-key-null (relation organization.department.members / attribute organization.membership.person_id))"},
		{"seat numbers that differ as numbers are expected to be a duplicate",
			"| 12 |\n| 12.0 |\n", "| 12 |\n| 12.5 |\n",
			"## 席番号は数として比べる\n\n<!-- tei: example of organization.membership expecting organization.department.members.seat_unique ",
			"organization.department.members.seat_unique",
			"expects not-satisfied, but rule organization.department.members.seat_unique answers satisfied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := replaceOnce(t, ruleExampleExamples, tc.old, tc.replacement)
			code, report := runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, content))
			if code != 1 || report.OK || len(report.Diagnostics) != 1 {
				t.Fatalf("code=%d report=%+v", code, report)
			}
			got := report.Diagnostics[0]
			if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || !strings.Contains(got.Diagnostic.Message, tc.message) {
				t.Fatalf("code=%s message=%q, want RULE_EXAMPLE_MISMATCH with %q", got.Diagnostic.Code, got.Diagnostic.Message, tc.message)
			}
			if got.Diagnostic.Subject == nil || got.Diagnostic.Subject.Kind != "semantic-validation" || got.Diagnostic.Subject.ID != tc.rule {
				t.Fatalf("subject=%+v, want rule %s", got.Diagnostic.Subject, tc.rule)
			}
			if len(got.Sources) != 1 || got.Sources[0].ArtifactPath != ruleExampleDocument {
				t.Fatalf("sources=%+v, want the example document", got.Sources)
			}
			if strings.Count(content, tc.at) != 1 {
				t.Fatalf("%q must occur exactly once in the edited document", tc.at)
			}
			wantOffset := strings.Index(content, tc.at) + len(tc.at)
			wantLine := 1 + strings.Count(content[:wantOffset], "\n")
			start := got.Sources[0].Range.Start
			if start.Offset != wantOffset || start.Line != wantLine {
				t.Fatalf("finding at offset %d line %d, want offset %d line %d", start.Offset, start.Line, wantOffset, wantLine)
			}
			// The finding covers the expected answer as written.
			written := content[start.Offset:got.Sources[0].Range.End.Offset]
			if written != "satisfied" && written != "not-satisfied" && written != "undecided" {
				t.Fatalf("the finding covers %q, not an expected answer", written)
			}
		})
	}
}

// An item the table has no column for was not written. It is not read as an
// empty value, and the rule that needs it stays undecided.
func TestAnItemWithoutAColumnIsNotAnEmptyValue(t *testing.T) {
	table := func(expect string) string {
		return "# 割合を書いていない例\n\n<!-- tei: example of organization.membership -->\n" +
			"| personId | organization.membership.allocation_positive |\n| --- | --- |\n| P001 | " + expect + " |\n"
	}
	code, report := runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, table("undecided")))
	if code != 0 || len(report.Diagnostics) != 0 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	for _, expect := range []string{"satisfied", "not-satisfied"} {
		code, report = runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, table(expect)))
		if code != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" ||
			!strings.Contains(report.Diagnostics[0].Diagnostic.Message, "value-not-fetched") {
			t.Fatalf("expect=%s code=%d report=%+v", expect, code, report)
		}
	}
}

// The same example checks the implementation the Workspace selected. With
// "not zero" selected for the operator declared as "not negative", the record
// with allocation 0 answers differently from the example.
func TestExamplesFindAWronglySelectedImplementation(t *testing.T) {
	selection := strings.Replace(ruleExampleSelection, "tei.number.non-negative", "tei.number.non-zero", 1)
	code, report := runRuleExampleValidate(t, ruleExampleWorkspace(t, selection, ruleExampleExamples))
	if code != 1 || len(report.Diagnostics) != 1 {
		t.Fatalf("code=%d report=%+v", code, report)
	}
	got := report.Diagnostics[0]
	wantOffset := strings.Index(ruleExampleExamples, rowP003) + len("| P003 | 0 | 2026-10-01 | | not-satisfied | ")
	if got.Diagnostic.Code != "RULE_EXAMPLE_MISMATCH" || got.Sources[0].Range.Start.Offset != wantOffset ||
		!strings.Contains(got.Diagnostic.Message, "expects satisfied, but rule organization.membership.allocation_not_negative answers not-satisfied") {
		t.Fatalf("got=%+v, want a mismatch at offset %d", got, wantOffset)
	}
}

// Without a selected implementation the operator rule cannot be asked. Its
// examples are reported as not checked; they do not pass silently. The rules
// that need no implementation are still checked and agree.
func TestExamplesOfARuleThatCannotBeAskedAreNotReportedAsAgreeing(t *testing.T) {
	code, report := runRuleExampleValidate(t, ruleExampleWorkspace(t, "", ruleExampleExamples))
	if code != 1 || len(report.Diagnostics) != 4 {
		t.Fatalf("code=%d diagnostics=%d report=%+v", code, len(report.Diagnostics), report)
	}
	for _, got := range report.Diagnostics {
		if got.Diagnostic.Code != "RULE_EXAMPLE_NOT_CHECKED" || got.Diagnostic.Subject == nil ||
			got.Diagnostic.Subject.ID != "organization.membership.allocation_not_negative" {
			t.Fatalf("got=%+v", got)
		}
	}
}

// A definition the validator refuses is reported once, as a definition
// finding. Its examples are not asked about records the rule cannot read.
func TestExamplesAreNotCheckedAgainstARefusedDefinition(t *testing.T) {
	root := ruleExampleWorkspace(t, ruleExampleSelection, ruleExampleExamples)
	path := filepath.Join(root, "semantic", "organization", "membership-rules.tei.md")
	rules := replaceOnce(t, ruleExampleRules, "organization.membership.allocation is positive", "organization.membership.starts_on is positive")
	if err := os.WriteFile(path, []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	code, report := runRuleExampleValidate(t, root)
	if code != 1 || len(report.Diagnostics) != 1 ||
		report.Diagnostics[0].Diagnostic.Code != "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}

func TestValidateRefusesAnExampleItCannotRead(t *testing.T) {
	header := "| personId | allocation | startsOn | endsOn | organization.membership.allocation_positive | organization.membership.allocation_not_negative |"
	cases := []struct {
		name             string
		old, replacement string
		// at occurs once in the edited document; the finding starts there.
		at      string
		message string
	}{
		{"data that is not declared",
			markSeat, strings.Replace(markSeat, "organization.membership ", "organization.assignment ", 1),
			"organization.assignment", "no source declares that data"},
		{"data without a namespace",
			markSeat, strings.Replace(markSeat, "organization.membership ", "Membership ", 1),
			"Membership expecting", "is not a qualified canonical id"},
		{"an item the data does not have",
			"| seatNumber |\n", "| seatNo |\n",
			"seatNo", `has no item "seatNo"`},
		{"a rule that is not declared",
			header, strings.Replace(header, "allocation_positive", "allocation_positiv", 1),
			"organization.membership.allocation_positiv ", "no source declares that rule"},
		{"a rule about all the rows written as a column",
			header, strings.Replace(header, "organization.membership.allocation_positive", "organization.department.members.seat_unique", 1),
			"organization.department.members.seat_unique |", "is about all the rows together"},
		{"a rule about one record written in the mark",
			markSeat, strings.Replace(markSeat, "organization.department.members.seat_unique", "organization.membership.allocation_positive", 1),
			"organization.membership.allocation_positive not-satisfied -->\n| seatNumber", "is about one record"},
		{"a rule that reads another data",
			header, strings.Replace(header, "organization.membership.allocation_positive", "organization.department.headcount_positive", 1),
			"organization.department.headcount_positive", "which is not an item of data organization.membership"},
		{"a number written with an exponent",
			"| P002 | 0.4 |", "| P002 | 4e-1 |",
			"4e-1", "item allocation is a number"},
		{"a number with a unit",
			"| 12.0 |\n", "| 12番 |\n",
			"12番", "item seatNumber is a number"},
		{"an expected answer that is not a word of the notation",
			rowP001, "| P001 | 1 | 2026-04-01 | | ok | satisfied |",
			"ok |", "is written satisfied, not-satisfied or undecided"},
		{"an expected answer in another case",
			markTwice, strings.Replace(markTwice, "not-satisfied", "Not-Satisfied", 1),
			"Not-Satisfied", "is written satisfied, not-satisfied or undecided"},
		{"an example that expects nothing",
			markSeat, "<!-- tei: example of organization.membership -->",
			"<!-- tei: example of organization.membership -->", "states no expected answer"},
		{"expecting without an answer",
			markSeat, "<!-- tei: example of organization.membership expecting organization.department.members.seat_unique -->",
			"<!-- tei: example of organization.membership expecting organization.department.members.seat_unique -->", "write pairs of a rule and its answer"},
		{"a word that is not expecting",
			markSeat, strings.Replace(markSeat, " expecting ", " expects ", 1),
			"expects ", "write expecting <rule> <answer>"},
		{"one rule expected twice in the mark",
			markSeat, strings.Replace(markSeat, " -->", " organization.department.members.seat_unique satisfied -->", 1),
			"organization.department.members.seat_unique satisfied", "is expected more than once"},
		{"one item in two columns",
			"| seatNumber |\n| --- |\n| 12 |\n| 12.0 |\n", "| seatNumber | seatNumber |\n| --- | --- |\n| 12 | 12 |\n| 12.0 | 13 |\n",
			"seatNumber |\n| --- | --- |", "appears more than once"},
		{"a rule column that expects nothing in any row",
			"| P900 | 0.5 |\n| P900 | 0.5 |\n", "| P900 | 0.5 |\n| P900 | 0.5 |\n\n<!-- tei: example of organization.membership -->\n| personId | organization.membership.allocation_positive |\n| --- | --- |\n| P777 | |\n",
			"organization.membership.allocation_positive |\n| --- | --- |\n| P777", "states no expected answer in any row"},
		{"an example without an item column",
			"| seatNumber |\n| --- |\n| 12 |\n| 12.0 |\n", "| organization.membership.allocation_positive |\n| --- |\n| satisfied |\n",
			markSeat, "needs at least one item column"},
		{"a mark the language does not know",
			markSeat, "<!-- tei: examples of organization.membership -->",
			"<!-- tei: examples of organization.membership -->", "unknown tei mark"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := replaceOnce(t, ruleExampleExamples, tc.old, tc.replacement)
			code, report := runRuleExampleValidate(t, ruleExampleWorkspace(t, ruleExampleSelection, content))
			if code != 1 || report.OK || len(report.Diagnostics) != 1 {
				t.Fatalf("code=%d report=%+v", code, report)
			}
			got := report.Diagnostics[0]
			if got.Diagnostic.Code != "PARSE_ERROR" || !strings.Contains(got.Diagnostic.Message, tc.message) {
				t.Fatalf("code=%s message=%q, want PARSE_ERROR with %q", got.Diagnostic.Code, got.Diagnostic.Message, tc.message)
			}
			if len(got.Sources) != 1 || got.Sources[0].ArtifactPath != ruleExampleDocument {
				t.Fatalf("sources=%+v", got.Sources)
			}
			if strings.Count(content, tc.at) != 1 {
				t.Fatalf("%q must occur exactly once in the edited document", tc.at)
			}
			wantOffset := strings.Index(content, tc.at)
			if start := got.Sources[0].Range.Start; start.Offset != wantOffset || start.Line != 1+strings.Count(content[:wantOffset], "\n") {
				t.Fatalf("finding at offset %d line %d, want offset %d", start.Offset, start.Line, wantOffset)
			}
		})
	}
}

// The older language reads fences only. An example table under it is refused
// instead of being skipped.
func TestTheFenceOnlyLanguageDoesNotSkipAnExampleTable(t *testing.T) {
	root := ruleExampleWorkspace(t, ruleExampleSelection, ruleExampleExamples)
	manifest := strings.Replace(ruleExampleManifest, "tei.authoring/v1", "tei.rule-authoring/v0", 1) + ruleExampleSelection
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	code, report := runRuleExampleValidate(t, root)
	if code != 1 || len(report.Diagnostics) == 0 || report.Diagnostics[0].Diagnostic.Code != "PARSE_ERROR" ||
		report.Diagnostics[0].Sources[0].ArtifactPath != ruleExampleDocument {
		t.Fatalf("code=%d report=%+v", code, report)
	}
}
