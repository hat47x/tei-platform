package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next product change is the row form of a presence
// input: `of each row` asks, for every row of a relation, whether that row holds
// a value (decision section 8.6). This test drives it through the product path:
// a real PostgreSQL read, a selected implementation, and an answer in
// perRelation the test writes down from the rows it inserts.
//
// The rows this test inserts, and the answer the rule must give:
//
//	order 1040, line 1: unit_price 2     (holds a value)
//	order 1040, line 2: unit_price 2     (holds a value)  -> satisfied
//	order 1041, line 1: unit_price 2     (holds a value)
//	order 1041, line 2: unit_price NULL  (holds none)     -> not-satisfied
//	order 1042                            no line at all  -> satisfied
//
// The expected answers are written from those rows, not read back from the CLI.
const presenceRowDatabase = "tei_destination_order_e2e"

const presenceRowDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.every_price_stated:\n" +
	"  takes value: presence of each row\n" +
	"  refutes on some rows\n" +
	"\n" +
	"rule sales.order.lines.every_price_stated:\n" +
	"  sales.order_line.unit_price is sales.every_price_stated per sales.order.lines\n" +
	"```\n"

// openPresenceRowDatabase opens the disposable database the workflow creates
// and refuses to touch any other one.
func openPresenceRowDatabase(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != presenceRowDatabase {
		t.Fatalf("isolated PostgreSQL database check failed: actual=%q err=%v", actual, err)
	}
	return db
}

// presenceRowFixture creates the nullable price column and inserts the three
// orders this test asks about.
func presenceRowFixture(t *testing.T) (binary, dsn string) {
	t.Helper()
	dsn = os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL presence-column read not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != presenceRowDatabase {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db := openPresenceRowDatabase(t, dsn)
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	// unit_price is nullable: the second order holds a line whose price is
	// NULL, which is the counterexample this test asks about.
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
		(order_line_id, order_id, line_number, item_code, quantity, unit_price) values
		(1, 1040, 1, 'A-100', 2, 2),
		(2, 1040, 2, 'B-200', 0, 2),
		(3, 1041, 1, 'A-100', 2, 2),
		(4, 1041, 2, 'B-200', 0, null)`); err != nil {
		t.Fatalf("cannot insert the presence-column rows: %v", err)
	}
	return buildCLIForLifecycle(t), dsn
}

// presenceRowWorkspace copies the reference Workspace and adds the operator, the
// rule that uses it, and the implementation the composition selects.
func presenceRowWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)

	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatal("the reference Workspace already selects a language")
	}
	additions := "languages:\n  - tei.authoring/v1\n" +
		"implementations:\n  - operator: sales.every_price_stated\n    use: tei.presence.every-is-present\n"
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-every-price-stated.tei.md"),
		[]byte(presenceRowDocument), 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	content, err = os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	if strings.Count(string(content), anchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(content), anchor,
		anchor+"    uses validation sales.order.lines.every_price_stated\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The shipped CLI answers a presence-column rule for rows that came from
// PostgreSQL, in perRelation and never per row.
func TestAPresenceColumnRuleIsAnsweredPerRelationFromPostgres(t *testing.T) {
	binary, dsn := presenceRowFixture(t)
	root := presenceRowWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the presence-column rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order.lines.every_price_stated"
	for _, tc := range []struct {
		order  string
		lines  int
		answer string
		why    string
	}{
		{order: "1040", lines: 2, answer: "satisfied", why: "both lines hold a price"},
		{order: "1041", lines: 2, answer: "not-satisfied", why: "line 2 holds no price, which refutes the rule"},
		{order: "1042", lines: 0, answer: "satisfied", why: "no row refutes a rule over an empty relation"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			report, got := runRowScopedProductView(t, binary, root, tc.order)
			if got.code != 0 || !report.OK || report.Error != nil {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			read := report.Result
			if read == nil || read.Coverage != "complete" {
				t.Fatalf("the read did not cover the relation: %+v", read)
			}
			if len(read.Rows) != tc.lines {
				t.Fatalf("rows=%d, want %d", len(read.Rows), tc.lines)
			}
			answer, present := read.PerRelation[ruleID]
			if !present {
				t.Fatalf("the presence-column rule is absent from perRelation: %+v", read.PerRelation)
			}
			if answer != tc.answer {
				t.Fatalf("order=%s perRelation=%q, independently expected %q (%s)", tc.order, answer, tc.answer, tc.why)
			}
			if len(read.PerRelationUndecided) != 0 {
				t.Fatalf("undecided answers where every row was read: %v", read.PerRelationUndecided)
			}
			for key, answers := range read.PerRow {
				if _, present := answers[ruleID]; present {
					t.Fatalf("the row-scoped rule was also answered for row %s", key)
				}
			}
		})
	}
}
