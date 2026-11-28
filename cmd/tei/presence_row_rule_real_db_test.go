package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next product change is to read a `presence`
// input over every row of a relation: "does every row of this relation hold a
// value?" The one-record question is answered from PostgreSQL rows
// (TestAPresenceRuleIsAnsweredPerRowFromPostgres). This test asks the row
// question from the same kind of rows.
//
// The rows this test inserts, and the answer the rule must give for each
// order:
//
//	order 1031: unit_price 100, NULL, 0   -> one row holds no value
//	order 1032: unit_price 100, 0         -> every row holds a value
//	order 1033: no lines                  -> the empty relation
//
// 1031 is not-satisfied because the second line's price is NULL. A price of 0
// is a value, so the third line of 1031 is not the counterexample; the NULL is.
// 1032 holds 0 as well and is satisfied, which is what keeps "zero" and "no
// value" apart. 1033 has no rows at all, and "every row holds a value" is true
// of no rows (section 8.1: the empty relation is one example).
//
// The expected answers are written from those rows, not read back from the CLI.
func TestARowScopedPresenceRuleDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL presence read not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatalf("refusing to modify the database without the disposable-database confirmation %q", databaseName)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize the isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != databaseName {
		t.Fatalf("expected %q, got %q: %v", databaseName, actual, err)
	}
	// unit_price is nullable here: the point of the test is a row that holds
	// none, which the reference DDL does not allow.
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2)
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	if _, err := db.Exec(`insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price)
		values
			(1, 1031, 1, 'A-100', 2, 100),
			(2, 1031, 2, 'B-200', 0, null),
			(3, 1031, 3, 'C-300', 5, 0),
			(4, 1032, 1, 'A-100', 3, 100),
			(5, 1032, 2, 'B-200', 12, 0)
		`); err != nil {
		t.Fatalf("cannot seed the detail rows: %v", err)
	}
	// The states the expectations are read from are counted in SQL first, so a
	// mistake in the seed fails here rather than as a wrong expectation.
	var nullPrices, zeroPrices, order1031Rows, order1032Rows int
	if err := db.QueryRow(`select
			count(*) filter (where unit_price is null),
			count(*) filter (where unit_price = 0),
			count(*) filter (where order_id = 1031),
			count(*) filter (where order_id = 1032)
		from public.order_lines`).Scan(&nullPrices, &zeroPrices, &order1031Rows, &order1032Rows); err != nil {
		t.Fatal(err)
	}
	if nullPrices != 1 || zeroPrices != 2 || order1031Rows != 3 || order1032Rows != 2 {
		t.Fatalf("the seeded rows state null prices=%d zero prices=%d order 1031 rows=%d order 1032 rows=%d, not 1/2/3/2",
			nullPrices, zeroPrices, order1031Rows, order1032Rows)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installRowPresenceRule(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the row-presence rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order.lines.every_price_stated"
	for _, tc := range []struct {
		order  string
		rows   int
		answer string
		why    string
	}{
		{order: "1031", rows: 3, answer: "not-satisfied",
			why: "line 2 holds no unit_price, and the operator states that a counterexample among the rows read is enough"},
		{order: "1032", rows: 2, answer: "satisfied",
			why: "both lines hold a price; 0 is a value and not an absent one"},
		{order: "1033", rows: 0, answer: "satisfied",
			why: "the relation is empty, and every row holding a value is true of no rows"},
	} {
		t.Run("order_"+tc.order, func(t *testing.T) {
			report, got := runRowScopedProductView(t, binary, root, tc.order)
			if got.code != 0 || !report.OK || report.Error != nil {
				t.Fatalf("order=%s: CLI failed: code=%d stdout=%s stderr=%s", tc.order, got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			read := report.Result
			if read == nil {
				t.Fatalf("order=%s: the report carried no result", tc.order)
			}
			if read.Coverage != "complete" {
				t.Fatalf("order=%s: coverage=%q, want complete", tc.order, read.Coverage)
			}
			if len(read.Rows) != tc.rows {
				t.Fatalf("order=%s: rows=%d, want %d", tc.order, len(read.Rows), tc.rows)
			}
			answer, present := read.PerRelation[ruleID]
			if !present {
				t.Fatalf("order=%s: %s is absent from perRelation: %+v", tc.order, ruleID, read.PerRelation)
			}
			if answer != tc.answer {
				t.Fatalf("order=%s: %s=%q, independently expected %q (%s)", tc.order, ruleID, answer, tc.answer, tc.why)
			}
			if _, undecided := read.PerRelationUndecided[ruleID]; undecided {
				t.Fatalf("order=%s: %s is also reported undecided: %+v", tc.order, ruleID, read.PerRelationUndecided)
			}
		})
	}
}

// installRowPresenceRule adds the row-scoped presence operator, the rule that
// asks it of the order's lines, the shipped implementation the composition
// selects, and the Application's use of the rule.
func installRowPresenceRule(t *testing.T, root string) {
	t.Helper()
	document := "# 明細の単価\n\n" +
		"受注の明細すべてに、単価の値があることを述べる。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.every_price_stated:\n" +
		"  takes value: presence of each row\n" +
		"  refutes on some rows\n\n" +
		"rule sales.order.lines.every_price_stated:\n" +
		"  sales.order_line.unit_price is sales.every_price_stated per sales.order.lines\n\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-row-presence.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatal("the reference Workspace already selects a language; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n  - operator: sales.every_price_stated\n    use: tei.presence.every-present\n"+
			"")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	appContent, err := os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	if strings.Count(string(appContent), anchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(appContent), anchor,
		anchor+"    uses validation sales.order.lines.every_price_stated\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
