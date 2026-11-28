package main

import (
	"database/sql"
	// json is used by the shared reader in typed_row_same_as_real_db_test.go
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// A rule whose operator reads every row of a relation as a truth has to be
// decided from rows that came from a real boolean column, and the answer must
// appear in perRelation.
//
// The shipped row forms are tei.text.every-same-as, tei.truth.every-is-true and
// tei.any.every-same-as. #6096 drove the text and `any` forms through the product
// path; the truth form is named by no test under cmd/tei at all, so nothing had
// read a boolean column through the product path.
//
// The expected answers are written from the seed's lines and from "every line of
// this order is urgent", not read back from the CLI:
//
//	order 1001: flag f, t, f  -> one row holds false, so the rule does not hold
//	order 1004: flag t, t     -> every row holds true, so the rule holds
//	order 1003: no lines      -> there is no row to refute it, so it holds
//
// A truth is read as a truth and never as the number 1: a boolean column arrives
// as a bool, and the reader keeps that.
func TestARowScopedTruthRuleDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL truth row read not run")
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
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	// The column is boolean, so the driver hands the reader a bool and the value
	// carries the type it arrived as (decision section 8.4).
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null,
		urgent boolean not null
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	// The seed does not state this column, so the lines are written here: the
	// five seed lines with the flags each one carries. Order 1001 holds false on
	// its first and third lines, and every line of 1002 holds true; 1003 holds
	// no line, which is why it is absent.
	if _, err := db.Exec(`
		insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price, urgent)
		values
			(1, 1001, 1, 'A-100', 2, 100, false),
			(2, 1001, 2, 'B-200', 0, 250, true),
			(3, 1001, 2, 'C-300', 5, 0, false),
			(4, 1002, 1, 'A-100', 3, 100, true),
			(5, 1002, 2, 'B-200', 12, 250, true)
	`); err != nil {
		t.Fatalf("cannot seed the truth column: %v", err)
	}
	// The expectation below is stated from these flags: 1001 holds one false,
	// and the two lines of 1002 are both true.
	var flags []string
	rows, err := db.Query("select urgent::text from public.order_lines where order_id = 1001 order by order_line_id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var flag string
		if err := rows.Scan(&flag); err != nil {
			t.Fatal(err)
		}
		flags = append(flags, flag)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	if len(flags) != 3 || flags[0] != "false" || flags[1] != "true" || flags[2] != "false" {
		t.Fatalf("order 1001 reads %v, not the three flags this test expects", flags)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installRowTypedTruthRule(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the row truth rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order.lines.all_urgent"
	for _, tc := range []struct {
		order       string
		want        string
		explanation string
	}{
		// The seed holds A-100, B-200 and C-300 for 1001 and the flags are
		// false, true, false: the first row refutes the statement.
		{order: "1001", want: "not-satisfied", explanation: "one line holds false"},
		// Every line of 1002 holds true, so no row refutes it.
		{order: "1002", want: "satisfied", explanation: "every line holds true"},
		// The seed states that 1003 holds no lines, so there is no row a
		// counterexample could be in.
		{order: "1003", want: "satisfied", explanation: "an empty relation has no row to refute it"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			read := readTypedRowRelation(t, binary, root, tc.order)
			answer := read.PerRelation[ruleID]
			if answer != tc.want {
				t.Fatalf("order=%s %s=%q, want %q: %s", tc.order, ruleID, answer, tc.want, tc.explanation)
			}
			if reason := read.PerRelationUndecided[ruleID]; reason != "" {
				t.Fatalf("order=%s %s is undecided (%q) although the rows were read", tc.order, ruleID, reason)
			}
			if read.Coverage != "complete" {
				t.Fatalf("order=%s coverage=%q, want complete", tc.order, read.Coverage)
			}
		})
	}
}

// installRowTypedTruthRule declares the boolean attribute, makes the detail
// Query return its column, maps it to the attribute, adds a row-scoped operator
// that reads every line's truth, and selects the shipped implementation.
func installRowTypedTruthRule(t *testing.T, root string) {
	t.Helper()
	// 1. The attribute, declared as a truth.
	dataPath := filepath.Join(root, "semantic", "sales", "order.tei")
	data, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	const attributes = "    attribute unitPrice type std.money\n"
	edited := replaceOnce(t, string(data), attributes,
		attributes+"    attribute urgent type std.boolean\n")
	if err := os.WriteFile(dataPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. The Query returns the column. Without this the reader would report the
	// attribute as not read, which is a different state from a truth.
	sqlPath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
	sqlText, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	const column = "    unit_price\n"
	editedSQL := replaceOnce(t, string(sqlText), column, "    unit_price,\n    urgent\n")
	if err := os.WriteFile(sqlPath, []byte(editedSQL), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. The mapping, so the column reaches the attribute.
	realizationPath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.tei")
	realization, err := os.ReadFile(realizationPath)
	if err != nil {
		t.Fatal(err)
	}
	const mapping = "        column unit_price -> sales.order_line.unit_price\n"
	editedRealization := replaceOnce(t, string(realization), mapping,
		mapping+"        column urgent -> sales.order_line.urgent\n")
	if err := os.WriteFile(realizationPath, []byte(editedRealization), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. The operator and the rule. The operator reads one column of truths, so
	// its single input is the rule's subject.
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-truth-rows.tei.md"),
		[]byte(rowTruthDocument), 0o644); err != nil {
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
			"implementations:\n  - operator: sales.every_line_true\n    use: tei.truth.every-is-true\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	appContent, err := os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	final := replaceOnce(t, string(appContent), anchor,
		anchor+"    uses validation sales.order.lines.all_urgent\n")
	if err := os.WriteFile(application, []byte(final), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rowTruthDocument states the operator and the rule: every line of the relation
// holds true.
const rowTruthDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.every_line_true:\n" +
	"  takes value: truth of each row\n" +
	"  refutes on some rows\n" +
	"\n" +
	"rule sales.order.lines.all_urgent:\n" +
	"  sales.order_line.urgent is sales.every_line_true per sales.order.lines\n" +
	"```\n"
