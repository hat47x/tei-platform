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
// input as whether a value is there. This test drives that through the product
// path: a real PostgreSQL read, a selected implementation, and an answer per
// row the test writes down from the rows it inserts.
//
// The rows this test inserts, and the answer each rule must give:
//
//	order 1030, line 1: unit_price 100   -> the column holds a value
//	order 1030, line 2: unit_price NULL  -> the column holds none
//
// A value of 0 is a value, so it answers "holds a value" and never "holds none".
// The expected answers are written from these rows, not read back from the CLI.
const presenceProductDatabase = "tei_destination_order_e2e"

const presenceProductDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.has_price:\n" +
	"  takes value: presence\n" +
	"\n" +
	"operator sales.has_no_price:\n" +
	"  takes value: presence\n" +
	"\n" +
	"rule sales.order_line.price_is_stated:\n" +
	"  sales.order_line.unit_price is sales.has_price\n" +
	"\n" +
	"rule sales.order_line.price_is_empty:\n" +
	"  sales.order_line.unit_price is sales.has_no_price\n" +
	"```\n"

// presenceProductFixture creates the nullable price column and seeds the two
// rows this test asks about. Only the price column matters here, so the other
// columns hold placeholder values that satisfy the Workspace's own rules.
func presenceProductFixture(t *testing.T) (binary, dsn string) {
	t.Helper()
	dsn = os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL presence read not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != presenceProductDatabase {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != presenceProductDatabase {
		t.Fatalf("isolated PostgreSQL database check failed: actual=%q err=%v", actual, err)
	}
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	// unit_price is nullable: the point of this test is a row that holds no
	// price at all.
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
		(1, 1030, 1, 'A-100', 2, 2),
		(2, 1030, 2, 'B-200', 0, null)`); err != nil {
		t.Fatalf("cannot insert the presence rows: %v", err)
	}
	return buildCLIForLifecycle(t), dsn
}

// presenceProductWorkspace copies the reference Workspace and adds the two
// presence operators, the rules that use them, the implementations the
// composition selects, and the Application's use of the rules.
func presenceProductWorkspace(t *testing.T) string {
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
		"implementations:\n" +
		"  - operator: sales.has_price\n    use: tei.presence.holds\n" +
		"  - operator: sales.has_no_price\n    use: tei.presence.absent\n"
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-price-presence.tei.md"),
		[]byte(presenceProductDocument), 0o644); err != nil {
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
		anchor+
			"    uses validation sales.order_line.price_is_stated\n"+
			"    uses validation sales.order_line.price_is_empty\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// The shipped CLI answers a presence rule for rows that came from PostgreSQL:
// the row that holds a price satisfies "the price is stated" and not "the price
// is empty", and the row whose price is NULL answers the other way.
func TestAPresenceRuleIsAnsweredPerRowFromPostgres(t *testing.T) {
	binary, dsn := presenceProductFixture(t)
	root := presenceProductWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the presence rules did not validate: %s%s",
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

	// The answers below are written from the two rows the fixture inserted.
	for _, tc := range []struct {
		key    string
		stated string
		empty  string
		why    string
	}{
		{
			key: "1", stated: "satisfied", empty: "not-satisfied",
			why: "line 1 holds unit_price 100, which is a value",
		},
		{
			key: "2", stated: "not-satisfied", empty: "satisfied",
			why: "line 2 holds no unit_price, so the price is not stated",
		},
	} {
		t.Run("row "+tc.key, func(t *testing.T) {
			answers := read.PerRow[tc.key]
			if got := answers["sales.order_line.price_is_stated"]; got != tc.stated {
				t.Fatalf("row=%s price_is_stated=%q, want %q (%s)", tc.key, got, tc.stated, tc.why)
			}
			if got := answers["sales.order_line.price_is_empty"]; got != tc.empty {
				t.Fatalf("row=%s price_is_empty=%q, want %q (%s)", tc.key, got, tc.empty, tc.why)
			}
		})
	}
}
