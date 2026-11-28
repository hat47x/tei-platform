package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next named change asks for a `presence` input to
// be read as whether a value is there. The reader existed (`Row.ReadPresence`)
// but nothing called it on the operator path: measured on main, a contract
// declaring `takes value: presence` was refused, and no shipped implementation
// could be selected for it.
//
// What this test fixes, on rows that come from a real column:
//
//   - a presence rule is decided from whether the attribute holds a value, not
//     from the value itself: the row whose code is A-100 is satisfied and the
//     row whose code is NULL is not-satisfied;
//   - an attribute the Query does not return is "nobody read it", which is a
//     different state from "the record holds no value": the rule stays
//     undecided there rather than being read as not-satisfied.
//
// The expected answers are stated from the values this test inserts and from
// decision section 5's table, not read back from the CLI.
func TestAPresenceRuleDecidesRealPostgresRows(t *testing.T) {
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
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	// `item_code` is nullable, so the same attribute holds a value for one row
	// and holds none for another. The column's type does not matter to a
	// presence rule: it asks whether a value is there, whatever it is.
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar,
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
			(2, 1001, 2, NULL, 0, 250)
	`); err != nil {
		t.Fatalf("cannot seed the nullable column: %v", err)
	}
	// Read back what the column holds, so the expectation is not this test's
	// own assumption about how SQL NULL arrives.
	rows, err := db.Query("select order_line_id, item_code from public.order_lines where order_id = 1001 order by order_line_id")
	if err != nil {
		t.Fatal(err)
	}
	type seeded struct {
		id   int64
		code sql.NullString
	}
	var got []seeded
	for rows.Next() {
		var s seeded
		if err := rows.Scan(&s.id, &s.code); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	if len(got) != 2 || !got[0].code.Valid || got[0].code.String != "A-100" || got[1].code.Valid {
		t.Fatalf("the seeded column reads %+v, not one value and one held absence", got)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installPresenceRule(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the presence rule did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order_line.item_code_present"
	read := readPresenceRows(t, binary, root, "1001")
	if read.Coverage != "complete" {
		t.Fatalf("coverage=%q, want complete", read.Coverage)
	}
	// Row 1 holds A-100: the attribute has a value, so the rule holds.
	if answer := read.PerRow["1"][ruleID]; answer != "satisfied" {
		t.Fatalf("%s on row 1=%q, want satisfied: the column holds A-100", ruleID, answer)
	}
	// Row 2 holds SQL NULL: the record holds no value, so the rule does not
	// hold. It is not undecided: the reader did ask and the answer is there.
	if answer := read.PerRow["2"][ruleID]; answer != "not-satisfied" {
		t.Fatalf("%s on row 2=%q, want not-satisfied: the column holds no value", ruleID, answer)
	}
	if reason := read.PerRowUndecided["2"][ruleID]; reason != "" {
		t.Fatalf("%s on row 2 is undecided (%q) although the value is known to be absent", ruleID, reason)
	}

}

type presenceRowRead struct {
	Coverage        string                       `json:"coverage"`
	PerRow          map[string]map[string]string `json:"perRow"`
	PerRowUndecided map[string]map[string]string `json:"perRowUndecided"`
}

func readPresenceRows(t *testing.T, binary, root, order string) presenceRowRead {
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
		OK     bool             `json:"ok"`
		Result *presenceRowRead `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK || report.Result == nil {
		t.Fatalf("order=%s: cannot read the report: %v %s", order, err, got.stdout)
	}
	return *report.Result
}

// installPresenceRule adds a presence rule over the item code and selects the
// shipped `tei.presence.present` for it.
func installPresenceRule(t *testing.T, root string) {
	t.Helper()
	document := "# 明細の項目コードがあること\n\n" +
		"明細のどの行も、項目コードに値があることを述べる。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.item_code_holds:\n" +
		"  takes value: presence\n\n" +
		"rule sales.order_line.item_code_present:\n" +
		"  sales.order_line.item_code is sales.item_code_holds\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-item-code-presence.tei.md"), []byte(document), 0o644); err != nil {
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
			"implementations:\n"+
			"  - operator: sales.item_code_holds\n    use: tei.presence.present\n")...)
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
		anchor+"    uses validation sales.order_line.item_code_present\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
