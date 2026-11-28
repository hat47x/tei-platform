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
// input as whether a value is there. The evaluator, the reader and the shipped
// implementation (tei.presence.is-present) are in place; nothing showed the
// question being answered by the shipped CLI from rows that came from
// PostgreSQL.
//
// The rows this test inserts, and the answer the rule must give on each:
//
//	order 1030, line 1: unit_price 2     -> the column holds a value
//	order 1030, line 2: unit_price NULL  -> the column holds no value
//
// The expected answers are written from those rows, not read back from the CLI.
// The fixture is the one the other row-scoped product tests use, so the
// Application, the binding and the Query are the reference Workspace's own.
const presenceDatabase = "tei_destination_order_e2e"

const presenceDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.has_price:\n" +
	"  takes value: presence\n" +
	"\n" +
	"rule sales.order_line.price_is_stated:\n" +
	"  sales.order_line.unit_price is sales.has_price\n" +
	"```\n"

// openIsolatedDatabase opens the disposable database the workflow creates and
// refuses to touch any other one.
func openIsolatedDatabase(t *testing.T, dsn, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != name {
		t.Fatalf("isolated PostgreSQL database check failed: actual=%q err=%v", actual, err)
	}
	return db
}

// insertPresenceRows writes the two lines this test asks about. They replace
// the seeded lines so the answers are not mixed with the seed's own rows.
func insertPresenceRows(t *testing.T, dsn string) {
	t.Helper()
	db := openIsolatedDatabase(t, dsn, presenceDatabase)
	// The fixture declares unit_price NOT NULL. This test asks whether the
	// column HOLDS a value, so one row has to hold none: the column is made
	// nullable for the rows this test inserts.
	if _, err := db.Exec("alter table public.order_lines alter column unit_price drop not null"); err != nil {
		t.Fatalf("cannot make unit_price nullable: %v", err)
	}
	if _, err := db.Exec("delete from public.order_lines"); err != nil {
		t.Fatalf("cannot clear the seeded order lines: %v", err)
	}
	if _, err := db.Exec(`insert into public.order_lines
		(order_line_id, order_id, line_number, item_code, quantity, unit_price) values
		(1, 1030, 1, 'A-100', 2, 2),
		(2, 1030, 2, 'B-200', 0, null)`); err != nil {
		t.Fatalf("cannot insert the presence rows: %v", err)
	}
}

// presenceWorkspace copies the reference Workspace and adds the operator, the
// rule that uses it, the implementation the composition selects, and the
// Application's use of the rule.
func presenceWorkspace(t *testing.T) string {
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
		"implementations:\n  - operator: sales.has_price\n    use: tei.presence.is-present\n"
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-price-presence.tei.md"),
		[]byte(presenceDocument), 0o644); err != nil {
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
		anchor+"    uses validation sales.order_line.price_is_stated\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The shipped CLI answers a presence rule for rows that came from PostgreSQL:
// the row that holds a price satisfies it, and the row whose price is NULL does
// not. A price of 0 is a value, so it is not the second case.
func TestAPresenceRuleIsAnsweredPerRowFromPostgres(t *testing.T) {
	binary, dsn := rowScopedProductFixture(t)
	insertPresenceRows(t, dsn)
	root := presenceWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the presence rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	report, got := runRowScopedProductView(t, binary, root, "1030")
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
	if len(read.Rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(read.Rows))
	}

	for _, tc := range []struct {
		key    string
		answer string
		why    string
	}{
		{key: "1", answer: "satisfied", why: "line 1 holds unit_price 2, which is a value"},
		{key: "2", answer: "not-satisfied", why: "line 2 holds no unit_price"},
	} {
		t.Run("row "+tc.key, func(t *testing.T) {
			answer, present := read.PerRow[tc.key]["sales.order_line.price_is_stated"]
			if !present {
				t.Fatalf("row=%s: the presence rule is absent from perRow: %+v", tc.key, read.PerRow)
			}
			if answer != tc.answer {
				t.Fatalf("row=%s price_is_stated=%q, independently expected %q (%s)", tc.key, answer, tc.answer, tc.why)
			}
		})
	}
}
