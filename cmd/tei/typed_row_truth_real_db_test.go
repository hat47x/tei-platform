package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next named change asks for the shipped
// implementations of the types other than a calendar day to be carried through
// to real PostgreSQL rows. Six of them now are, and one remains: the row form of
// truth (`tei.truth.every-is-true`), which no test under `cmd/tei` named at all.
// Measured on main, every row-scoped real-PostgreSQL check uses a number, a day
// or a text rule.
//
// What this test fixes, on rows whose flag comes from a real `boolean` column:
//
//   - a truth is read as a truth: `tei.truth.every-is-true` answers from the
//     boolean the database returned, not from the number of some column;
//   - every row true is satisfied; a row holding false is the counterexample a
//     `refutes on some rows` declaration is about, so that order is
//     not-satisfied; an order with no rows is satisfied because there is no row
//     to refute it.
//
// The expected answers are written from the values this test inserts and from
// the decision table, not read back from the CLI: the flag column is queried
// first and the expectation is stated from what it holds.
func TestARowScopedTruthRuleDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL row truth read not run")
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
	// `flag_note` is a `boolean` column, so the driver hands back a bool and not
	// a number. NULL is left out of the seed on purpose: a row that holds no
	// value is a different case, pinned below in its own order.
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null,
		flag_note boolean not null
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	if _, err := db.Exec(`
		insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price, flag_note)
		values
			(1, 1001, 1, 'A-100', 2, 100, true),
			(2, 1001, 2, 'B-200', 0, 250, true),
			(3, 1002, 1, 'C-300', 5, 0, true),
			(4, 1002, 2, 'D-400', 1, 10, false)
	`); err != nil {
		t.Fatalf("cannot seed the flag column: %v", err)
	}
	// The expectation is stated from what the column holds, read back first.
	for _, tc := range []struct {
		order string
		want  []bool
	}{
		{order: "1001", want: []bool{true, true}},
		{order: "1002", want: []bool{true, false}},
		{order: "1003", want: nil},
	} {
		rows, err := db.Query("select flag_note from public.order_lines where order_id = $1 order by order_line_id", tc.order)
		if err != nil {
			t.Fatal(err)
		}
		var got []bool
		for rows.Next() {
			var flag bool
			if err := rows.Scan(&flag); err != nil {
				t.Fatal(err)
			}
			got = append(got, flag)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		_ = rows.Close()
		if len(got) != len(tc.want) {
			t.Fatalf("order %s holds %d flags, not the %d this test expects", tc.order, len(got), len(tc.want))
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("order %s flag %d reads %v, not %v", tc.order, i, got[i], tc.want[i])
			}
		}
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installRowTruthRule(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the row truth rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order.lines.every_flag_is_true"
	for _, tc := range []struct {
		order string
		want  string
		why   string
	}{
		{order: "1001", want: "satisfied", why: "both rows hold true, so no row refutes it"},
		{order: "1002", want: "not-satisfied", why: "one row holds false and it was read"},
		{order: "1003", want: "satisfied", why: "an empty relation has no row to refute it"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			read := readTypedRowRelation(t, binary, root, tc.order)
			if read.Coverage != "complete" {
				t.Fatalf("order=%s coverage=%q, want complete", tc.order, read.Coverage)
			}
			if answer := read.PerRelation[ruleID]; answer != tc.want {
				t.Fatalf("order=%s %s=%q, want %q: %s", tc.order, ruleID, answer, tc.want, tc.why)
			}
			if reason := read.PerRelationUndecided[ruleID]; reason != "" {
				t.Fatalf("order=%s %s is undecided (%q) although the rows were read", tc.order, ruleID, reason)
			}
		})
	}
}

// installRowTruthRule adds a row-scoped truth attribute to the order line, makes
// the detail Query return the column, maps it, and puts a rule over it with the
// shipped `tei.truth.every-is-true`.
func installRowTruthRule(t *testing.T, root string) {
	t.Helper()
	// 1. The attribute. `std.boolean` is the type the standard names for a truth.
	dataPath := filepath.Join(root, "semantic", "sales", "order.tei")
	data, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	const attributes = "    attribute unitPrice type std.money\n"
	if strings.Count(string(data), attributes) != 1 {
		t.Fatalf("order.tei does not hold %q exactly once", attributes)
	}
	edited := replaceOnce(t, string(data), attributes,
		attributes+"    attribute flagNote type std.boolean\n")
	if err := os.WriteFile(dataPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. The Query returns the column. Without this the reader reports the
	// attribute as not read, which is a different state from a truth.
	sqlPath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
	sqlText, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	const column = "    unit_price\n"
	editedSQL := replaceOnce(t, string(sqlText), column, "    unit_price,\n    flag_note\n")
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
		mapping+"        column flag_note -> sales.order_line.flag_note\n")
	if err := os.WriteFile(realizationPath, []byte(editedRealization), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. The operator and the rule.
	document := "# 明細すべての印\n\n" +
		"明細のどの行も、印が真であることを述べる。行が無ければ反例も無い。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.every_flag_holds:\n" +
		"  takes value: truth of each row\n" +
		"  refutes on some rows\n\n" +
		"rule sales.order.lines.every_flag_is_true:\n" +
		"  sales.order_line.flag_note is sales.every_flag_holds per sales.order.lines\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-every-flag.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}

	// 5. The selection, and the application naming the rule so the read asks it.
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if containsBytes(content, "languages:") {
		t.Fatal("the reference workspace already selects a language; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n"+
			"  - operator: sales.every_flag_holds\n    use: tei.truth.every-is-true\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	appContent, err := os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	editedApp := replaceOnce(t, string(appContent), anchor,
		anchor+"    uses validation sales.order.lines.every_flag_is_true\n")
	if err := os.WriteFile(application, []byte(editedApp), 0o644); err != nil {
		t.Fatal(err)
	}
}
