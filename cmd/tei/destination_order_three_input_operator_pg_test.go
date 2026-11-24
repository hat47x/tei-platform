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

// A rule decided by a three-input operator must be answered per row from the
// real PostgreSQL values, through the same `tei view -environment` entry the
// guide uses.
//
// plan/NOW.md road item 5 says the named inputs of a rule are handed to the
// existing meaning and evaluation path, starting with `within(value, low,
// high)`. The operator declarations, the selection wiring and the evaluator
// exist; what this test adds is the one thing none of them is checked against:
// an actual row in an actual database.
//
// The expected answers below were written from the seeded values, not copied
// from the CLI output:
//
//	order 1001 quantities 2, 0, 5   inside [0, 10] -> satisfied, satisfied, satisfied
//	order 1002 quantities 3, 12     inside [0, 10] -> satisfied, not-satisfied
//
// The second value of each order is the one that discriminates: 12 is outside
// the range while being positive, so a rule that ignored `low`/`high` and
// answered "is the quantity positive" would call it satisfied.
func TestDestinationRowAnswersAThreeInputOperatorFromPostgres(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL three-input operator not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actualName string
	if err := db.QueryRow("select current_database()").Scan(&actualName); err != nil || actualName != databaseName {
		t.Fatalf("isolated PostgreSQL database check failed: actual=%q err=%v", actualName, err)
	}

	// This test owns only the detail table. The parent rows the browse pattern
	// reads are left to the suite that seeds them, so this does not disturb
	// another suite's fixture.
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })

	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("the seeded rows did not fit the table: %v", err)
	}

	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	applyThreeInputOperatorRule(t, root)

	binary := buildCLIForLifecycle(t)
	for _, tc := range []struct {
		orderID string
		answers map[string]string
	}{
		{orderID: "1001", answers: map[string]string{"1": "satisfied", "2": "satisfied", "3": "satisfied"}},
		{orderID: "1002", answers: map[string]string{"4": "satisfied", "5": "not-satisfied"}},
	} {
		t.Run(tc.orderID, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+tc.orderID,
				"-format", "json",
			)
			if got.code != 0 {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					PerRow map[string]map[string]string `json:"perRow"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
			}
			if !report.OK {
				t.Fatalf("view failed: %s", got.stdout)
			}
			const rule = "sales.order_line.quantity_within"
			if len(report.Result.PerRow) != len(tc.answers) {
				t.Fatalf("order %s answered %d rows, want %d: %+v",
					tc.orderID, len(report.Result.PerRow), len(tc.answers), report.Result.PerRow)
			}
			for key, want := range tc.answers {
				answers, ok := report.Result.PerRow[key]
				if !ok {
					t.Fatalf("order %s did not answer order-line %s", tc.orderID, key)
				}
				if got := answers[rule]; got != want {
					t.Fatalf("order %s line %s: %s=%q, the seeded values say %q",
						tc.orderID, key, rule, got, want)
				}
			}
		})
	}
}

// applyThreeInputOperatorRule edits the copied Workspace: select the authoring
// language, add the operator and its rule, select the implementation the
// running binary carries, and declare the rule in the Application.
//
// Every step refuses when the copy is not what this test assumes, so a silent
// no-op cannot make the assertions above pass for the wrong reason.
func applyThreeInputOperatorRule(t *testing.T, root string) {
	t.Helper()
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatalf("the reference Workspace already selects a language, so this edit would be a no-op")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n  - operator: sales.within\n    use: tei.number.within\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}

	const ruleDocument = "# 範囲の規則\n\n数量は 0 以上 10 以下である。\n\n```tei\n" +
		"namespace sales\n\noperator sales.within:\n" +
		"  takes value: number\n  takes low: number\n  takes high: number\n\n" +
		"rule sales.order_line.quantity_within:\n" +
		"  sales.order_line.quantity is sales.within low: 0 high: 10\n```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "quantity-within.tei.md"),
		[]byte(ruleDocument), 0o644); err != nil {
		t.Fatal(err)
	}

	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	content, err = os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order.lines.line_number_unique\n"
	if strings.Count(string(content), anchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(content), anchor,
		anchor+"    uses validation sales.order_line.quantity_within\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
