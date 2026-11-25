package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// plan/NOW.md road item 5's next product change, and decision section 15.5's
// last part: a rule whose operator declares an input read over all rows must be
// decided through the CLI, from rows a real PostgreSQL returned, with answers
// written by hand from the seed.
//
// The seed is testdata/destination-order-change/seed.sql. The rule says every
// line's quantity is at most that same line's unit_price:
//
//	order 1001: key 1 quantity 2 / price 100 -> holds
//	            key 2 quantity 0 / price 250 -> holds
//	            key 3 quantity 5 / price 0   -> does NOT hold
//	order 1002: key 4 quantity 3 / price 100 -> holds
//	            key 5 quantity 12 / price 250 -> holds
//
// So 1001 is not-satisfied (one counterexample was read) and 1002 is satisfied
// (every row holds and the rows are all there). A read that compared a row's
// quantity against another row's price would answer 1001 satisfied, because
// every key's quantity is under 100 or 250; the not-satisfied answer is what
// separates a same-row comparison from a cross-row one.

const eachRowOperatorDocument = "```tei\nnamespace sales\n\n" +
	"operator sales.every_line_at_most:\n" +
	"  takes value: number of each row\n" +
	"  takes limit: number of each row\n" +
	"  refutes on some rows\n\n" +
	"rule sales.order_line.every_quantity_at_most_price:\n" +
	"  sales.order_line.quantity is sales.every_line_at_most limit: sales.order_line.unit_price per sales.order.lines\n" +
	"```\n"

// eachRowWorkspace prepares the reference Workspace with the row-scoped
// operator, its shipped implementation, and its rule.
func eachRowWorkspace(t *testing.T) string {
	t.Helper()
	return operatorMultiInputWorkspaceWithManifest(t, eachRowOperatorDocument,
		"implementations:\n  - operator: sales.every_line_at_most\n    use: tei.number.every-at-most\n")
}

// eachRowView runs the CLI and returns the answers of the rules asked of the
// relation, keyed by rule.
func eachRowView(t *testing.T, binary, root, orderID string) (map[string]string, cliProcessStreams) {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+orderID,
		"-format", "json",
	)
	relation := map[string]string{}
	if got.code != 0 {
		return relation, got
	}
	var report operatorMultiInputViewReport
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
	}
	if report.Result == nil {
		return relation, got
	}
	for rule, answer := range report.Result.PerRelation {
		relation[rule] = answer
	}
	return relation, got
}

// TestARowScopedRuleIsDecidedThroughTheCLIFromRealPostgresRows checks the path
// a user takes: tei validate, then tei view -environment over rows that came
// from PostgreSQL, with the answers decided from the seed.
func TestARowScopedRuleIsDecidedThroughTheCLIFromRealPostgresRows(t *testing.T) {
	binary, _ := operatorMultiInputFixture(t)
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	root := eachRowWorkspace(t)

	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	const rule = "sales.order_line.every_quantity_at_most_price"
	for _, tc := range []struct {
		order string
		want  string
	}{
		// Key 3 (quantity 5, unit_price 0) is the counterexample.
		{order: "1001", want: "not-satisfied"},
		// Every line of 1002 is under its own price, and the rows are all
		// there, so the rule holds for the relation.
		{order: "1002", want: "satisfied"},
	} {
		t.Run(tc.order, func(t *testing.T) {
			relation, got := eachRowView(t, binary, root, tc.order)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			answer, present := relation[rule]
			if !present {
				t.Fatalf("order=%s has no relation answer for %s: %v", tc.order, rule, relation)
			}
			if answer != tc.want {
				t.Fatalf("order=%s %s=%q, independently expected %q", tc.order, rule, answer, tc.want)
			}
			// The rule is about the relation, so it has no per-key answers.
			// A row-by-row answer would appear here and would look decided.
			if perRow := relationOfOne(t, got.stdout, rule); perRow != "" {
				t.Fatalf("order=%s compiled a row answer for a relation rule: %q", tc.order, perRow)
			}
		})
	}
}

// relationOfOne reports the per-key answer of a rule when the report carries
// one, so a test can show that a relation rule did not become a row rule.
func relationOfOne(t *testing.T, stdout, rule string) string {
	t.Helper()
	var report operatorMultiInputViewReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil || report.Result == nil {
		return ""
	}
	for _, answers := range report.Result.PerRow {
		if answer, present := answers[rule]; present {
			return answer
		}
	}
	return ""
}
