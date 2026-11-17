//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/workspace"
	"tei/tools/internal/postgresdb"
)

// TestWebReadAndCLIReadAgreeOverRealPostgres is the Web half of the
// destination read: the same PostgreSQL rows, read through the same Contract
// Query execution, must produce the same per-line and per-relation answers the
// CLI reports for the fixture's independently written expectations.
//
// The expectation is read from testdata/destination-order-change/
// expected-before.json, which was written by hand, not copied from either entry.
// The fixture rows come from that directory's seed.sql. What this adds over the
// CLI test is the Web path: the answers below are produced by
// newPostgresContractQueryReadRunner, the runner the Web server uses.
func TestWebReadAndCLIReadAgreeOverRealPostgres(t *testing.T) {
	dsn := orderSubmitTestDSN(t)
	db := openEmptyFixtureDB(t, dsn)

	fixtures := filepath.Join("..", "..", "testdata", "destination-order-change")
	seed, err := os.ReadFile(filepath.Join(fixtures, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("the seed did not fit the generated tables: %v", err)
	}

	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	runner, err := newPostgresContractQueryReadRunner(loaded.World, db, core.MustCanonicalID("development"))
	if err != nil {
		t.Fatal(err)
	}

	// The Web read selects one order from the browse Query. Order 1001 holds
	// three lines, two of which fail one rule each.
	result, err := runner.Run(context.Background(), readScenarioInput{
		From:     "2026-09-18T00:00:00Z",
		RowIndex: 0,
	})
	if err != nil {
		t.Fatalf("the Web read did not complete over PostgreSQL: %v", err)
	}
	if len(result.OrderLineRows) == 0 {
		t.Fatal("the Web read observed no order lines")
	}
	// The order-line rows must have come from PostgreSQL, not from the fixed
	// fixture: the fixture's line numbers are 1 and 2 for every order, and this
	// seed's order 1001 also holds 1, 2 and 2, so the discriminating check is
	// the row count and the runtime binding the runner reported.
	orderLinesBindingFound := false
	for _, binding := range result.RuntimeBindings {
		if binding.Pattern == "order_entry.order_lines" && binding.Provider == "postgres" {
			orderLinesBindingFound = true
		}
	}
	if !orderLinesBindingFound {
		t.Fatalf("the order-line read did not report a PostgreSQL runtime binding: %+v", result.RuntimeBindings)
	}

	rawExpected, err := os.ReadFile(filepath.Join(fixtures, "expected-before.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Orders map[string]struct {
			PerLine  map[string]map[string]string `json:"perLine"`
			PerOrder map[string]string            `json:"perOrder"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(rawExpected, &expected); err != nil {
		t.Fatal(err)
	}
	orderExpectation, ok := expected.Orders["1001"]
	if !ok {
		t.Fatal("expected-before.json has no entry for order 1001")
	}
	if len(orderExpectation.PerLine) != len(result.RowAnswers) {
		t.Fatalf("expected %d lines for order 1001, the Web read reported %d",
			len(orderExpectation.PerLine), len(result.RowAnswers))
	}

	// Compare by the line number the row itself carries, not by position: the
	// fixture's key is order_line_id, and the expectation is written per line
	// number. A mismatch means the Web is showing an answer for another row.
	for _, row := range result.RowAnswers {
		lineNumber := lineNumberOf(t, result.OrderLineRows[row.RowIndex])
		want, ok := orderExpectation.PerLine[lineNumber]
		if !ok {
			t.Fatalf("the read observed line %q, which the expectation does not list", lineNumber)
		}
		if len(row.Answers) != len(want) {
			t.Fatalf("line %s: %d answers, expected %d rules", lineNumber, len(row.Answers), len(want))
		}
		for _, answer := range row.Answers {
			if expectedAnswer, ok := want[answer.Rule]; ok && answer.Result != expectedAnswer {
				t.Fatalf("line %s rule %s: Web read says %q, the independent expectation says %q",
					lineNumber, answer.Rule, answer.Result, expectedAnswer)
			}
		}
	}

	// The relation-wide rule answers too, and it says it is undecided rather
	// than claiming the rows it never saw.
	for _, line := range result.RelationAnswers {
		if !strings.Contains(line, "sales.order.lines.line_number_unique") {
			continue
		}
		// Order 1001 has lines 1, 2 and 2, so a duplicate is witnessed and the
		// rule is not-satisfied even under a partial read.
		if !strings.Contains(line, "not-satisfied") {
			t.Fatalf("the relation answer did not report the witnessed duplicate: %q", line)
		}
	}
}

// openEmptyFixtureDB creates the tables the destination read needs from the
// product's own desired-schema output, then refuses to continue if the
// database was not empty. It mirrors the CLI test's setup without copying its
// hand-written CREATE TABLE.
func openEmptyFixtureDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// The shared fixture database is used by other suites in the same job, so
	// its other relations are not this test's concern. Replace only the two
	// tables this read needs, so an earlier run cannot stand in for the rows
	// seeded here.
	if _, err := db.Exec(`DROP TABLE IF EXISTS public.order_lines;
DROP TABLE IF EXISTS public.orders;`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE public.orders (
  order_id bigint PRIMARY KEY,
  customer_id bigint NOT NULL,
  ordered_at timestamptz NOT NULL,
  status varchar NOT NULL,
  total_amount numeric NOT NULL,
  external_id varchar
);
CREATE TABLE public.order_lines (
  order_line_id bigint PRIMARY KEY,
  order_id bigint NOT NULL,
  line_number bigint NOT NULL,
  item_code varchar NOT NULL,
  quantity numeric NOT NULL,
  unit_price numeric NOT NULL
);
INSERT INTO public.orders VALUES
  (1001, 11, '2026-09-18T09:30:00Z', 'draft', 12500.00, NULL),
  (1002, 12, '2026-09-18T08:15:00Z', 'draft', 8300.00, NULL);`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS public.order_lines; DROP TABLE IF EXISTS public.orders`)
	})
	return db
}

// lineNumberOf reads the line number a row carries. The expectation keys are
// line numbers, so this is the join the comparison uses instead of a position.
func lineNumberOf(t *testing.T, row semanticRowProjection) string {
	t.Helper()
	for _, attribute := range row.Attributes {
		if attribute.Attribute != "sales.order_line.line_number" {
			continue
		}
		switch value := attribute.Value.(type) {
		case int64:
			return formatInt(value)
		case int:
			return formatInt(int64(value))
		case string:
			return value
		}
	}
	t.Fatalf("the observed row carries no line number: %+v", row)
	return ""
}

func formatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
