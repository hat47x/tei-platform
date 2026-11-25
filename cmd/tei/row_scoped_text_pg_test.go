package main

import (
	"strings"
	"testing"
)

// `tei view -environment` prints the same destinationRead in three forms. The
// JSON form puts a row-scoped answer in `perRelation`, and the HTML form prints
// it in the relation section. The TEXT form prints it too, on a line of its own
// after the row blocks — with no leading indentation, which is what distinguishes
// it from a row's answers.
//
// Nothing checked that on a real read: the text-form tests build a synthetic
// destinationRead. Measured here, a row-scoped rule over real PostgreSQL rows
// prints `sales.order.lines.quantity_at_most_5 satisfied` at the left margin, and
// never under a `row` block.
func TestRowScopedAnswerIsPrintedOutsideTheRowBlocksInTheTextForm(t *testing.T) {
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
		// 1001 holds 2, 0 and 5, so every quantity is at most the constant 5.
		{order: "1001", keys: []string{"1", "2", "3"}, answer: "satisfied"},
		// 1002 holds 3 and 12; 12 is a counterexample and the contract states
		// that a witnessed counterexample among the rows is enough.
		{order: "1002", keys: []string{"4", "5"}, answer: "not-satisfied"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+tc.order,
			)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the text output")
			}
			text := got.stdout
			// The relation answer is printed at the left margin, on its own
			// line, with no row indentation.
			line := ruleID + " " + tc.answer
			found := false
			for _, candidate := range strings.Split(text, "\n") {
				if candidate == line {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("order=%s: the text form does not print %q on its own line:\n%s",
					tc.order, line, text)
			}
			// It is not printed under a row: a row-scoped answer beside a row
			// would read as that row's answer.
			for _, key := range tc.keys {
				block := textRowBlock(t, text, key)
				if strings.Contains(block, ruleID) {
					t.Fatalf("order=%s row=%s: the row-scoped rule is printed inside the row block:\n%s",
						tc.order, key, block)
				}
			}
			// The same read's JSON form names the rule in perRelation, so the
			// two forms describe one answer rather than different ones.
			jsonGot := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+tc.order,
				"-format", "json",
			)
			if jsonGot.code != 0 {
				t.Fatalf("JSON form failed: code=%d stderr=%s", jsonGot.code, jsonGot.stderr)
			}
			if !strings.Contains(jsonGot.stdout, `"`+ruleID+`":"`+tc.answer+`"`) {
				t.Fatalf("order=%s: the JSON form does not carry %s=%s:\n%s",
					tc.order, ruleID, tc.answer, jsonGot.stdout)
			}
		})
	}
}

// textRowBlock returns the lines of a `row <key>` block: from the row line to
// the next row line or the first unindented line after it, whichever comes first.
func textRowBlock(t *testing.T, text, key string) string {
	t.Helper()
	const rowPrefix = "row "
	lines := strings.Split(text, "\n")
	start := -1
	for index, line := range lines {
		if line == rowPrefix+key {
			start = index
			break
		}
	}
	if start < 0 {
		t.Fatalf("the text form has no block for row %s", key)
	}
	end := start + 1
	for end < len(lines) {
		line := lines[end]
		if strings.HasPrefix(line, rowPrefix) || (line != "" && !strings.HasPrefix(line, "  ")) {
			break
		}
		end++
	}
	return strings.Join(lines[start:end], "\n")
}
