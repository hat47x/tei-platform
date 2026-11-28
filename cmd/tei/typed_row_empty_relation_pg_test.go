package main

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// A rule that reads a column of days is answered perRelation for the rows that
// came from PostgreSQL. Two cases are not covered by the test beside this one,
// and both are the answers a typed row read is most likely to get wrong:
//
//	order 1003  the relation holds no line at all
//	            no row refutes the rule, and no row holds a day it could not
//	            read, so the rule is satisfied rather than undecided
//	order 1004  one of the two lines holds no day
//	            the day nobody read is not the zero day: the rule is undecided
//	            with the row's own reason
//
// The expectations are written from the rows this test inserts, not read back
// from the CLI.

const (
	emptyOrder  = "1003"
	absentOrder = "1004"
)

// insertDaysForTwoMoreOrders adds the two orders this test asks about to the
// rows the shared fixture seeded. 1003 is left with no line at all; 1004 gets
// two lines, one of which holds no promised day.
func insertDaysForTwoMoreOrders(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	// Order 1003 holds no line: the reference seed inserts lines for 1001 and
	// 1002 only, and this test adds nothing for it.
	if _, err := db.Exec(`insert into public.order_lines
		(order_line_id, order_id, line_number, item_code, quantity, unit_price, promised_on, deadline_on) values
		(401, 1004, 1, 'A-100', 2, 100, '2026-04-02', '2026-04-01'),
		(402, 1004, 2, 'B-200', 0, 250, null, '2026-04-01')`); err != nil {
		t.Fatalf("cannot insert the absent-day rows: %v", err)
	}
}

// An empty relation satisfies a rule whose rows all satisfy it, and a relation
// with a row that holds no day is undecided with that row's own reason.
func TestATypedRowRuleOverAnEmptyOrAbsentRelation(t *testing.T) {
	binary, dsn := typedRowFixture(t)
	insertDaysForTwoMoreOrders(t, dsn)
	root := typedRowWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	for _, tc := range []struct {
		order    string
		want     string
		reason   string
		rowsHeld int
		why      string
	}{
		{
			order:    emptyOrder,
			want:     "satisfied",
			rowsHeld: 0,
			why:      "the relation holds no line, so no row refutes the rule",
		},
		{
			order:    absentOrder,
			want:     "undecided",
			reason:   "value-null",
			rowsHeld: 2,
			why:      "one line holds no promised day, and an absent day is not the zero day",
		},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			report, got := readTypedRowRule(t, binary, root, tc.order)
			if got.code != 0 || !report.OK || report.Error != nil {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			read := report.Result
			if read == nil {
				t.Fatalf("the CLI reported no result: %s", got.stdout)
			}
			answer, present := read.PerRelation["sales.order.lines.promised_not_before_deadline"]
			if !present {
				t.Fatalf("the typed row rule is absent from perRelation: %+v", read.PerRelation)
			}
			if answer != tc.want {
				t.Fatalf("order=%s perRelation=%q (undecided=%v), independently expected %q (%s)",
					tc.order, answer, read.PerRelationUndecided, tc.want, tc.why)
			}
			if tc.want == "undecided" {
				reason := read.PerRelationUndecided["sales.order.lines.promised_not_before_deadline"]
				if !strings.Contains(reason, tc.reason) {
					t.Fatalf("order=%s undecided reason=%q, want %q", tc.order, reason, tc.reason)
				}
				return
			}
			if len(read.PerRelationUndecided) != 0 {
				t.Fatalf("order=%s: the rule is answered and also undecided: %v", tc.order, read.PerRelationUndecided)
			}
		})
	}
}
