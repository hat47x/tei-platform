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

// `plan/NOW.md` road item 5's next named change is to connect `of each row` and
// `refutes on some rows` to the **product** path with a real-database
// counterexample. The evaluator side is in place and the conversion now accepts
// these declarations, but nothing yet shows that `tei view -environment` answers
// a row-scoped rule at all: the product read path was only exercised with
// per-record rules.
//
// The rows below are inserted by this test and the expected answers are
// written from them, not read back from the run:
//
//	order 1004: quantity 2 and 0, unit_price 10 for both rows
//	order 1005: quantity 12 and 3, unit_price 10 for both rows
//
// The rule asks `every-at-most` over `sales.order.lines` with `limit:
// sales.order_line.unit_price`. The limit is read from the same row as the quantity;
// both orders here happen to use the same price 10:
//
//	1004 → satisfied     (2 ≤ 10 and 0 ≤ 10)
//	1005 → not-satisfied (12 > 10 is a counterexample)
//
// A second operator states the same rule without `refutes on some rows`. With
// the whole relation read a witnessed counterexample is decisive without any
// statement, so it answers the same way; the statement only matters when the
// relation was partly read (decision sections 15.2 and 6).
func TestRowScopedRuleDecidedFromPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL row-scoped read not run")
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
		quantity numeric(12,3),
		unit_price numeric(12,2) not null
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })

	// The reference seed supplies the ordinary orders; the two below state the
	// cases this test asks about, with price 10 on each row.
	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("cannot seed order lines: %v", err)
	}
	if _, err := db.Exec(`insert into public.order_lines
		(order_line_id, order_id, line_number, item_code, quantity, unit_price) values
		(101, 1004, 1, 'A-100', 2, 10),
		(102, 1004, 2, 'B-200', 0, 10),
		(103, 1005, 1, 'A-100', 12, 10),
		(104, 1005, 2, 'B-200', 3, 10)`); err != nil {
		t.Fatalf("cannot insert the row-scoped cases: %v", err)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installRowScopedRules(t, root)

	// The exact revision whose rules will be evaluated must validate first.
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the row-scoped rules did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const (
		refutingRule = "sales.order_line.quantity_at_most_price"
		plainRule    = "sales.order_line.quantity_at_most_price_without_statement"
	)
	for _, tc := range []struct {
		order  string
		answer string
		lines  int
	}{
		{order: "1004", answer: "satisfied", lines: 2},
		// A counterexample was read: 12 is above the limit 10.
		{order: "1005", answer: "not-satisfied", lines: 2},
	} {
		t.Run(tc.order, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+tc.order,
				"-format", "json",
			)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					Coverage    string                       `json:"coverage"`
					PerRow      map[string]map[string]string `json:"perRow"`
					PerRelation map[string]string            `json:"perRelation"`
					Undecided   map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK {
				t.Fatalf("cannot read the report: %v\n%s", err, got.stdout)
			}
			// The reference binding states `covers sales.order.lines` and its
			// Query matches the storage, so every row of this order was read.
			if report.Result.Coverage != "complete" {
				t.Fatalf("coverage=%q, want complete", report.Result.Coverage)
			}
			if len(report.Result.PerRow) != tc.lines {
				t.Fatalf("lines read=%d, want %d: %v", len(report.Result.PerRow), tc.lines, report.Result.PerRow)
			}
			// A row-scoped rule is answered for the relation, never for a row:
			// an answer in perRow would be a per-record decision of a rule that
			// reads every row.
			for key, answers := range report.Result.PerRow {
				for _, ruleID := range []string{refutingRule, plainRule} {
					if _, present := answers[ruleID]; present {
						t.Fatalf("order=%s line=%s: the row-scoped rule %s was answered per row: %v",
							tc.order, key, ruleID, answers)
					}
				}
			}
			for _, ruleID := range []string{refutingRule, plainRule} {
				if got := report.Result.PerRelation[ruleID]; got != tc.answer {
					t.Fatalf("order=%s: %s=%q, want %q (all answers: %v)",
						tc.order, ruleID, got, tc.answer, report.Result.PerRelation)
				}
				if reason, held := report.Result.Undecided[ruleID]; held {
					t.Fatalf("order=%s: %s is undecided (%q) although every row was read",
						tc.order, ruleID, reason)
				}
			}
		})
	}
	// Keep the covers declaration but narrow the SQL. A visible counterexample
	// may settle only the operator that explicitly declares refutation on some
	// rows; an invisible counterexample cannot settle either rule.
	for _, tc := range []struct {
		name         string
		excluded     string
		key          string
		wantRefuting string
	}{
		{name: "partial_visible_counterexample", excluded: "104", key: "103", wantRefuting: "not-satisfied"},
		{name: "partial_hidden_counterexample", excluded: "103", key: "104", wantRefuting: "undecided"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limitedRoot := t.TempDir()
			copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), limitedRoot)
			installRowScopedRules(t, limitedRoot)
			sourcePath := filepath.Join(limitedRoot, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			const predicate = "WHERE order_id = :order_id"
			limited := replaceOnce(t, string(source), predicate, predicate+" AND order_line_id <> "+tc.excluded)
			if err := os.WriteFile(sourcePath, []byte(limited), 0o644); err != nil {
				t.Fatal(err)
			}
			if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", limitedRoot); validated.code != 0 {
				t.Fatalf("the limited Workspace did not validate: %s%s", validated.stdout, validated.stderr)
			}
			report, got := runRowScopedProductView(t, binary, limitedRoot, "1005")
			if got.code != 0 || !report.OK || report.Error != nil || report.Result == nil {
				t.Fatalf("limited CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			read := report.Result
			if read.Coverage != "partial" || len(read.Rows) != 1 || read.Rows[0].Key != tc.key {
				t.Fatalf("want partial read of key %s, got %+v", tc.key, read)
			}
			for _, expected := range []struct {
				rule   string
				answer string
			}{
				{rule: refutingRule, answer: tc.wantRefuting},
				{rule: plainRule, answer: "undecided"},
			} {
				answer, reason := read.PerRelation[expected.rule], read.PerRelationUndecided[expected.rule]
				if answer != expected.answer {
					t.Fatalf("%s: answer=%q, want %q", expected.rule, answer, expected.answer)
				}
				if expected.answer == "undecided" && !strings.Contains(reason, "relation-partially-read") {
					t.Fatalf("%s: want partial-read reason, got %q", expected.rule, reason)
				}
				if expected.answer == "not-satisfied" && reason != "" {
					t.Fatalf("%s: witnessed counterexample has undecided reason %q", expected.rule, reason)
				}
				for key, answers := range read.PerRow {
					if _, present := answers[expected.rule]; present {
						t.Fatalf("%s was answered for row %s", expected.rule, key)
					}
				}
			}
		})
	}

}

// installRowScopedRules adds two row-scoped operators and their rules to the
// copied Workspace and declares that the Application uses them. Nothing else
// in the Workspace changes.
func installRowScopedRules(t *testing.T, root string) {
	t.Helper()
	document := "# 明細ごとの数量と単価\n\n" +
		"明細のどの行も、数量がその行の単価以下であることを述べる。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.at_most:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row\n" +
		"  refutes on some rows\n\n" +
		"operator sales.at_most_without_statement:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row\n\n" +
		"rule sales.order_line.quantity_at_most_price:\n" +
		"  sales.order_line.quantity is sales.at_most limit: sales.order_line.unit_price per sales.order.lines\n\n" +
		"rule sales.order_line.quantity_at_most_price_without_statement:\n" +
		"  sales.order_line.quantity is sales.at_most_without_statement limit: sales.order_line.unit_price per sales.order.lines\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "row-scoped.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatal("the reference workspace already selects a language; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n"+
			"  - operator: sales.at_most\n    use: tei.number.every-at-most\n"+
			"  - operator: sales.at_most_without_statement\n    use: tei.number.every-at-most\n")...)
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
			"    uses validation sales.order_line.quantity_at_most_price\n"+
			"    uses validation sales.order_line.quantity_at_most_price_without_statement\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
