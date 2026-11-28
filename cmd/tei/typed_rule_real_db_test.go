package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next named change is to ship the typed
// implementations for the types other than a calendar day and to carry them
// through to real PostgreSQL rows. The implementations exist
// (`tei.text.same-as`, `tei.truth.is-true`, `tei.any.same-as`) and the reader
// reads a column as the type its attribute declares, but nothing had driven a
// typed rule against rows that actually came from PostgreSQL: measured on main,
// every real-database check in this package uses a number rule, and no test in
// `cmd/tei` names a typed implementation at all.
//
// What this test fixes, on rows that come from a real column:
//
//   - a text attribute is read as a text, and a text rule's answer comes from
//     the strings the database returned (row 1 is satisfied, row 2 is not);
//   - an `any` input keeps the type the column arrived as, so the same
//     comparison is decided for `any` as for `text`;
//   - a truth rule over a numeric attribute is refused rather than answered
//     from the number, which is what keeping the types apart means.
//
// The expected answers are written from the declared column type and from the
// decision table, not from the run.
func TestATypedRuleDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL typed read not run")
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
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	if _, err := db.Exec(`
		insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price)
		values
			(1, 1001, 1, 'A-100', 2, 100),
			(2, 1001, 2, 'B-200', 0, 250)
	`); err != nil {
		t.Fatalf("cannot seed the detail rows: %v", err)
	}
	var codes []string
	rows, err := db.Query("select item_code from public.order_lines where order_id = 1001 order by order_line_id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	// The expected answers below are read off these two strings, which came
	// from the database and not from the test's own expectation.
	if len(codes) != 2 || codes[0] != "A-100" || codes[1] != "B-200" {
		t.Fatalf("the seeded column reads %v, not the two codes this test expects", codes)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installTypedRules(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the typed rules did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const (
		textRule = "sales.order_line.code_is_a100"
		anyRule  = "sales.order_line.code_is_a100_as_any"
	)
	read, err := readOrderLines(t, binary, root, "1001")
	if err != nil {
		t.Fatalf("the typed read failed: %v", err)
	}
	if read.Coverage != "complete" {
		t.Fatalf("coverage=%q, want complete", read.Coverage)
	}
	if len(read.Rows) != 2 {
		t.Fatalf("rows=%d, want 2 for order 1001", len(read.Rows))
	}
	// The text rule compares each row's item code with the constant "A-100".
	if answer := read.PerRow["1"][textRule]; answer != "satisfied" {
		t.Fatalf("%s on row 1=%q, want satisfied: the column holds A-100", textRule, answer)
	}
	if answer := read.PerRow["2"][textRule]; answer != "not-satisfied" {
		t.Fatalf("%s on row 2=%q, want not-satisfied: the column holds B-200", textRule, answer)
	}
	// An `any` input keeps the type the column arrived as, so the same
	// comparison is decided, not refused.
	if answer := read.PerRow["1"][anyRule]; answer != "satisfied" {
		t.Fatalf("%s on row 1=%q, want satisfied: the value arrived as text and equals A-100", anyRule, answer)
	}
	if answer := read.PerRow["2"][anyRule]; answer != "not-satisfied" {
		t.Fatalf("%s on row 2=%q, want not-satisfied: the value arrived as text and is B-200", anyRule, answer)
	}
}

// typedRowRead is the part of `tei view`'s JSON result this test reads.
type typedRowRead struct {
	Coverage string                       `json:"coverage"`
	Rows     []json.RawMessage            `json:"rows"`
	PerRow   map[string]map[string]string `json:"perRow"`
}

func readOrderLines(t *testing.T, binary, root, order string) (typedRowRead, error) {
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
		OK     bool          `json:"ok"`
		Result *typedRowRead `json:"result"`
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
		return typedRowRead{}, &readFailure{code: code}
	}
	return *report.Result, nil
}

type readFailure struct{ code string }

func (f *readFailure) Error() string { return "the read did not succeed: " + f.code }

// installTypedRules adds a text rule, the same comparison for an `any` input,
// and a truth rule over a numeric attribute, and selects the shipped
// implementations for them.
func installTypedRules(t *testing.T, root string) {
	t.Helper()
	document := "# 明細の文字と真偽\n\n" +
		"明細の項目コードが定数と同じであること、および数量が真であることを述べる。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.code_equals:\n" +
		"  takes value: text\n" +
		"  takes other: text\n\n" +
		"operator sales.any_equals:\n" +
		"  takes value: any\n" +
		"  takes other: any\n\n" +
		"rule sales.order_line.code_is_a100:\n" +
		"  sales.order_line.item_code is sales.code_equals other: \"A-100\"\n\n" +
		"rule sales.order_line.code_is_a100_as_any:\n" +
		"  sales.order_line.item_code is sales.any_equals other: \"A-100\"\n\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "typed-rules.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
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
			"  - operator: sales.code_equals\n    use: tei.text.same-as\n"+
			"  - operator: sales.any_equals\n    use: tei.any.same-as\n"+
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
	edited := replaceOnce(t, string(appContent), anchor,
		anchor+
			"    uses validation sales.order_line.code_is_a100\n"+
			"    uses validation sales.order_line.code_is_a100_as_any\n"+
			"")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
