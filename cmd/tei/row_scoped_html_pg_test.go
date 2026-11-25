package main

import (
	"html"
	"strings"
	"testing"
)

// `tei view -environment -format html` prints the same destinationRead the JSON
// form prints. A row-scoped rule is answered for the relation, never for a row,
// so its answer belongs in the page's relation section — and it must not appear
// beside a row, where a reader would take it as that row's answer.
//
// The page had a relation section (`RelationAnswers`) before any row-scoped rule
// could reach it, so this pins that the section is actually filled from a real
// PostgreSQL read and that the row blocks do not claim the rule.
func TestRowScopedAnswerAppearsInTheRelationSectionOfTheHTMLPage(t *testing.T) {
	binary, dsn := rowScopedProductFixture(t)
	root := rowScopedProductWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order.lines.quantity_at_most_5"
	for _, tc := range []struct {
		order  string
		keys   []string
		answer string
	}{
		// 1001 holds 2, 0 and 5: every quantity is at most the constant 5.
		{order: "1001", keys: []string{"1", "2", "3"}, answer: "satisfied"},
		// 1002 holds 3 and 12: 12 is a counterexample and the contract states
		// that a witnessed counterexample among the rows is enough.
		{order: "1002", keys: []string{"4", "5"}, answer: "not-satisfied"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+tc.order,
				"-format", "html",
			)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			page := got.stdout
			if strings.Contains(page, dsn) {
				t.Fatal("the database connection string appeared in the HTML page")
			}
			if !strings.Contains(page, "<!doctype html>") {
				t.Fatalf("the HTML form did not return a page: %s", page)
			}
			if !strings.Contains(page, "<h2>関係全体の答え</h2>") {
				t.Fatalf("the page has no relation-answer section: %s", page)
			}

			// The answer is a row of the relation section, in the same three
			// columns the JSON form states: rule, answer, reason.
			want := "<tr class=\"answer-" + classOfAnswer(tc.answer) + "\"><td>" +
				html.EscapeString(ruleID) + "</td><td>" + html.EscapeString(tc.answer) + "</td><td></td></tr>"
			if !strings.Contains(page, want) {
				t.Fatalf("order=%s: the relation section does not hold %q\npage=%s", tc.order, want, page)
			}

			// No row block claims this rule: a row-scoped answer printed beside
			// a row would read as that row's answer.
			for _, key := range tc.keys {
				marker := "<h3>" + html.EscapeString(key) + "</h3>"
				if !strings.Contains(page, marker) {
					t.Fatalf("order=%s: the page has no block for row %s", tc.order, key)
				}
				block := rowBlockOf(t, page, marker)
				if strings.Contains(block, html.EscapeString(ruleID)) {
					t.Fatalf("order=%s row=%s: the row-scoped rule is printed beside a row:\n%s",
						tc.order, key, block)
				}
			}
		})
	}

	// The page names the source of its answers, so a reader can tell a real
	// read from a rendering of something else.
	if !strings.Contains(strings.ToLower(readHTMLForOrder(t, binary, root, "1001")), "order_entry.order_lines") {
		t.Fatal("the page does not name the pattern it rendered")
	}

}

// rowBlockOf returns the part of the page that belongs to one row: from the
// row's heading to the next row heading or the relation section, whichever
// comes first.
func rowBlockOf(t *testing.T, page, marker string) string {
	t.Helper()
	start := strings.Index(page, marker)
	if start < 0 {
		return ""
	}
	rest := page[start+len(marker):]
	end := len(rest)
	for _, stop := range []string{"<h3>", "<h2>"} {
		if index := strings.Index(rest, stop); index >= 0 && index < end {
			end = index
		}
	}
	return rest[:end]
}

// classOfAnswer maps an answer to the CSS class the page uses for it. A decided
// answer has one of the two result classes; anything else would be a reason and
// is not used here.
func classOfAnswer(answer string) string {
	switch answer {
	case "satisfied":
		return "satisfied"
	case "not-satisfied":
		return "not-satisfied"
	}
	return "undecided"
}

func readHTMLForOrder(t *testing.T, binary, root, order string) string {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+order,
		"-format", "html",
	)
	if got.code != 0 {
		t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
	}
	return got.stdout
}
