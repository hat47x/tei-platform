package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next named change asks for the shipped
// implementations of the types other than a calendar day to be carried through
// to real PostgreSQL rows. The text and `any` implementations now are
// (typed_rule_real_db_test.go); the calendar-day implementation had only a
// validation-time check of its examples. Measured on main: `tei.day.not-before`
// is named by no test under cmd/tei at all.
//
// What this test fixes, on rows whose day comes from a real `date` column:
//
//   - a calendar day is read as a day: `tei.day.not-before` answers from the two
//     dates the database returned;
//   - a row dated before the bound is not-satisfied and a row on the bound is
//     satisfied, which the column itself decides.
//
// The expectation is stated from the two days read back out of the database, not
// from this test's own guess.
func TestADayRuleDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL day read not run")
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
	// The column is `date`, not `timestamp`: decision section 7 says a moment in
	// time is not a day until the reader is told which zone to read it in, so a
	// timestamp column would be refused rather than read as a day.
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null,
		shipped_on date not null
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	if _, err := db.Exec(`
		insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price, shipped_on)
		values
			(1, 1001, 1, 'A-100', 2, 100, '2026-03-31'),
			(2, 1001, 2, 'B-200', 0, 250, '2026-04-01')
	`); err != nil {
		t.Fatalf("cannot seed the day column: %v", err)
	}
	var shipped []string
	rows, err := db.Query("select to_char(shipped_on, 'YYYY-MM-DD') from public.order_lines where order_id = 1001 order by order_line_id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			t.Fatal(err)
		}
		shipped = append(shipped, day)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	// The expectation below is stated from these two days: the first is before
	// 2026-04-01, the second is on it.
	if len(shipped) != 2 || shipped[0] != "2026-03-31" || shipped[1] != "2026-04-01" {
		t.Fatalf("the seeded column reads %v, not the two days this test expects", shipped)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installDayRule(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the day rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order_line.shipped_not_before_april"
	read, err := readDayRows(t, binary, root, "1001")
	if err != nil {
		t.Fatalf("the day read failed: %v", err)
	}
	if read.Coverage != "complete" {
		t.Fatalf("coverage=%q, want complete", read.Coverage)
	}
	// 2026-03-31 is before 2026-04-01, so the rule is not-satisfied there; 2026-04-01
	// is not before itself, so it is satisfied.
	if answer := read.PerRow["1"][ruleID]; answer != "not-satisfied" {
		t.Fatalf("%s on row 1=%q, want not-satisfied: the column holds 2026-03-31", ruleID, answer)
	}
	if answer := read.PerRow["2"][ruleID]; answer != "satisfied" {
		t.Fatalf("%s on row 2=%q, want satisfied: the column holds 2026-04-01", ruleID, answer)
	}
}

type dayRowRead struct {
	Coverage string                       `json:"coverage"`
	Rows     []map[string]any             `json:"rows"`
	PerRow   map[string]map[string]string `json:"perRow"`
}

func readDayRows(t *testing.T, binary, root, order string) (dayRowRead, error) {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+order,
		"-format", "json",
	)
	if got.code != 0 {
		t.Fatalf("order=%s: CLI failed: code=%d stdout=%s stderr=%s", order, got.code, got.stdout, got.stderr)
	}
	var report struct {
		OK     bool        `json:"ok"`
		Result *dayRowRead `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("order=%s: cannot read the report: %v %s", order, err, got.stdout)
	}
	if !report.OK || report.Result == nil {
		code := ""
		if report.Error != nil {
			code = report.Error.Code + ": " + report.Error.Message
		}
		return dayRowRead{}, &readFailure{code: code}
	}
	return *report.Result, nil
}

// installDayRule adds a calendar-day attribute to the order line, makes the
// detail Query return the column, maps it, and puts a date rule over it with the
// shipped `tei.day.not-before`.
func installDayRule(t *testing.T, root string) {
	t.Helper()
	// 1. The attribute, declared as a calendar day rather than as a moment.
	dataPath := filepath.Join(root, "semantic", "sales", "order.tei")
	data, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	const attributes = "    attribute unitPrice type std.money\n"
	edited := replaceOnce(t, string(data), attributes,
		attributes+"    attribute shippedOn type std.date\n")
	if err := os.WriteFile(dataPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. The Query returns the column. Without this the reader would report the
	// attribute as not read, which is a different state from a day.
	sqlPath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
	sqlText, err := os.ReadFile(sqlPath)
	if err != nil {
		t.Fatal(err)
	}
	const column = "    unit_price\n"
	editedSQL := replaceOnce(t, string(sqlText), column, "    unit_price,\n    shipped_on\n")
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
		mapping+"        column shipped_on -> sales.order_line.shipped_on\n")
	if err := os.WriteFile(realizationPath, []byte(editedRealization), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. The operator and the rule.
	document := "# 明細の発送日\n\n" +
		"明細の発送日が、2026年4月1日より前でないことを述べる。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.not_before_april:\n" +
		"  takes value: date\n" +
		"  takes deadline: date\n\n" +
		"rule sales.order_line.shipped_not_before_april:\n" +
		"  sales.order_line.shipped_on is sales.not_before_april deadline: 2026-04-01\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "day-rule.tei.md"), []byte(document), 0o644); err != nil {
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
			"  - operator: sales.not_before_april\n    use: tei.day.not-before\n")...)
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
		anchor+"    uses validation sales.order_line.shipped_not_before_april\n")
	if err := os.WriteFile(application, []byte(editedApp), 0o644); err != nil {
		t.Fatal(err)
	}
}
