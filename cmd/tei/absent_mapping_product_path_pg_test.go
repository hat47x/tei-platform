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

// `plan/NOW.md` road item 5 still lists "a shipped implementation reads an
// absent value" as unimplemented. That is now false: `tei.number.every-at-most`
// states the mapping `zero`. What was never shown is that the mapping reaches the
// decision through the PRODUCT path — a real read, a real Workspace, the shipped
// CLI — rather than only through a hand-built implementation in a unit test.
//
// The rows are inserted here and the expected answers are written from them:
//
//	order 1010: quantity 2 and 0, unit_price NULL on both rows
//
// With `limit: sales.order_line.unit_price or zero` and the declaration stating
// `absent as zero`, an absent price is the value zero. Quantity 2 is then above
// it, so the rule does not hold:
//
//	1010 → not-satisfied
//
// Without the declaration and the rule's `or zero`, the same rows leave the
// question open, because a value the row does not hold is not zero:
//
//	1010 → undecided: value-null
func TestAbsentPriceIsReadAsZeroOnTheProductPath(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL absent-mapping read not run")
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
	// unit_price is nullable here: the point of the test is a row that holds no
	// price at all.
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3),
		unit_price numeric(12,2)
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	if _, err := db.Exec(`insert into public.order_lines
		(order_line_id, order_id, line_number, item_code, quantity, unit_price) values
		(201, 1010, 1, 'A-100', 2, null),
		(202, 1010, 2, 'B-200', 0, null)`); err != nil {
		t.Fatalf("cannot insert the absent-price rows: %v", err)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installAbsentMappingRules(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the absent mapping did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	for _, tc := range []struct {
		rule   string
		answer string
		reason string
	}{
		// The mapping is declared on the input and named by the rule.
		{rule: "sales.order_line.quantity_at_most_price_mapped", answer: "not-satisfied"},
		// The same rows without either half of the mapping: nothing states
		// what an absent price means, so it stays undecided.
		{rule: "sales.order_line.quantity_at_most_price_unmapped", answer: "undecided", reason: "value-null"},
	} {
		t.Run(tc.rule, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id=1010",
				"-format", "json",
			)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					Coverage    string            `json:"coverage"`
					Rows        []json.RawMessage `json:"rows"`
					PerRelation map[string]string `json:"perRelation"`
					Undecided   map[string]string `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK {
				t.Fatalf("cannot read the report: %v\n%s", err, got.stdout)
			}
			if report.Result.Coverage != "complete" {
				t.Fatalf("coverage=%q, want complete", report.Result.Coverage)
			}
			if len(report.Result.Rows) != 2 {
				t.Fatalf("rows=%d, want the two rows of order 1010", len(report.Result.Rows))
			}
			if answer := report.Result.PerRelation[tc.rule]; answer != tc.answer {
				t.Fatalf("%s=%q, want %q (all answers: %v)", tc.rule, answer, tc.answer, report.Result.PerRelation)
			}
			reason := report.Result.Undecided[tc.rule]
			if tc.reason == "" {
				if reason != "" {
					t.Fatalf("%s has an undecided reason %q although it was decided", tc.rule, reason)
				}
				return
			}
			if !strings.Contains(reason, tc.reason) {
				t.Fatalf("%s: reason=%q, want it to name %q", tc.rule, reason, tc.reason)
			}
		})
	}
}

// installAbsentMappingRules adds the two operators and their rules. One states
// `absent as zero` on its limit input and the rule names that mapping; the other
// states nothing, so an absent price keeps its own reason.
func installAbsentMappingRules(t *testing.T, root string) {
	t.Helper()
	document := "# 単価がない明細\n\n" +
		"単価が無い行の数量を、単価0と比べるかどうかだけが違う二つの規則。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.at_most_price:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row absent as zero\n" +
		"  refutes on some rows\n\n" +
		"operator sales.at_most_price_unmapped:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row\n" +
		"  refutes on some rows\n\n" +
		"rule sales.order_line.quantity_at_most_price_mapped:\n" +
		"  sales.order_line.quantity is sales.at_most_price limit: sales.order_line.unit_price or zero per sales.order.lines\n\n" +
		"rule sales.order_line.quantity_at_most_price_unmapped:\n" +
		"  sales.order_line.quantity is sales.at_most_price_unmapped limit: sales.order_line.unit_price per sales.order.lines\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "absent-price.tei.md"), []byte(document), 0o644); err != nil {
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
			"  - operator: sales.at_most_price\n    use: tei.number.every-at-most\n"+
			"  - operator: sales.at_most_price_unmapped\n    use: tei.number.every-at-most\n")...)
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
			"    uses validation sales.order_line.quantity_at_most_price_mapped\n"+
			"    uses validation sales.order_line.quantity_at_most_price_unmapped\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
