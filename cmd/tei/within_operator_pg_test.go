package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// This test asks the question `plan/NOW.md` road item 5 still leaves open: does
// a rule whose decision needs SEVERAL named operator inputs get the right
// answer from the rows a real PostgreSQL returns?
//
// The expected answers below are written by hand from the seed rows
// (testdata/destination-order-change/seed.sql), not read from the product's
// output. `within` includes both bounds, so a quantity exactly on a bound is
// satisfied.
//
//	order 1001: line 1 qty 2, line 2 qty 0, line 3 qty 5
//	order 1002: line 1 qty 3, line 2 qty 12
//
// The rule is `low: 3 high: 100`:
//
//	1001/1 qty 2  -> not-satisfied   (below low)
//	1001/2 qty 0  -> not-satisfied   (below low)
//	1001/3 qty 5  -> satisfied
//	1002/1 qty 3  -> satisfied       (exactly low: the bound is included)
//	1002/2 qty 12 -> satisfied
func TestWithinRuleDecidedFromPostgresRows(t *testing.T) {
	const databaseName = "tei_within_operator_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL within-operator read not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatalf("refusing to modify database without explicit disposable database confirmation %q", databaseName)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize the isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actualName string
	if err := db.QueryRow("select current_database()").Scan(&actualName); err != nil || actualName != databaseName {
		t.Fatalf("isolated PostgreSQL database check failed: actual=%q err=%v", actualName, err)
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
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
			t.Errorf("cannot clean up the disposable order lines: %v", err)
		}
	})
	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("cannot seed order lines: %v", err)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installWithinRule(t, root)

	// The exact Workspace revision whose rule will be evaluated must validate
	// first; a different revision validating is not evidence for this read.
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the within rule did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	const ruleID = "sales.order_line.quantity_in_range"
	// The key is the row identity the Query returns, which is the order line
	// id (`order_line_id`), not the line number inside the order.
	//	1001: order_line_id 1 qty 2, 2 qty 0, 3 qty 5
	//	1002: order_line_id 4 qty 3, 5 qty 12
	want := map[string]map[string]string{
		"1001": {"1": "not-satisfied", "2": "not-satisfied", "3": "satisfied"},
		"1002": {"4": "satisfied", "5": "satisfied"},
	}
	for _, orderID := range []string{"1001", "1002"} {
		t.Run(orderID, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines",
				"-environment", "development", "-input", "order_id="+orderID,
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
					Coverage string `json:"coverage"`
					Rows     []struct {
						Key string `json:"key"`
					} `json:"rows"`
					PerLine           map[string]map[string]string `json:"perRow"`
					PerRelation       map[string]string            `json:"perRelation"`
					PerRelationReason map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
			}
			if !report.OK {
				t.Fatalf("the read failed: %s", got.stdout)
			}
			// The reference binding states `covers sales.order.lines` and its
			// Query matches the relation storage, so this read is complete and
			// every answer must be the independently written one.
			if report.Result.Coverage != "complete" {
				t.Fatalf("order=%s coverage=%q, want complete", orderID, report.Result.Coverage)
			}

			lines := want[orderID]
			if len(report.Result.PerLine) != len(lines) {
				t.Fatalf("order=%s lines=%d, expected %d independently: %v",
					orderID, len(report.Result.PerLine), len(lines), report.Result.PerLine)
			}
			if len(report.Result.Rows) != len(lines) {
				t.Fatalf("order=%s rows=%d, expected %d", orderID, len(report.Result.Rows), len(lines))
			}
			seen := map[string]bool{}
			for _, row := range report.Result.Rows {
				if _, err := strconv.ParseInt(row.Key, 10, 64); err != nil || seen[row.Key] {
					t.Fatalf("order=%s repeated or invalid line identity %q", orderID, row.Key)
				}
				seen[row.Key] = true
			}
			for lineID, expected := range lines {
				answers, held := report.Result.PerLine[lineID]
				if !held {
					t.Fatalf("order=%s line=%s was not read (perRow=%v)", orderID, lineID, report.Result.PerLine)
				}
				if answers[ruleID] != expected {
					t.Fatalf("order=%s line=%s %s=%q, expected independently %q (all answers: %v)",
						orderID, lineID, ruleID, answers[ruleID], expected, answers)
				}
			}
			// The within rule is about one row, so it contributes nothing to
			// the relation answers. A rule decided per relation must not be
			// silently answered here.
			if _, held := report.Result.PerRelation[ruleID]; held {
				t.Fatalf("the per-row within rule was answered per relation: %v", report.Result.PerRelation)
			}
		})
	}

	// A row whose quantity is NULL cannot be decided: the rule reads a value
	// the row does not hold. It must answer undecided with a reason, never
	// not-satisfied, so a missing value is not recorded as a failed check.
	t.Run("a row with no quantity is undecided", func(t *testing.T) {
		if _, err := db.Exec(`insert into public.order_lines (order_line_id, order_id, line_number, item_code, quantity, unit_price)
			values (6, 1003, 1, 'D-400', null, 10)`); err != nil {
			t.Skipf("the table does not accept a null quantity: %v", err)
		}
		t.Cleanup(func() {
			_, _ = db.Exec("delete from public.order_lines where order_line_id = 6")
		})
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines",
			"-environment", "development", "-input", "order_id=1003",
			"-format", "json",
		)
		if got.code != 0 {
			t.Fatalf("CLI failed on a null value: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
		}
		var report struct {
			OK     bool `json:"ok"`
			Result struct {
				PerLine map[string]map[string]string `json:"perRow"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
			t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
		}
		if !report.OK {
			t.Fatalf("the read failed: %s", got.stdout)
		}
		answers, held := report.Result.PerLine["6"]
		if !held {
			t.Fatalf("line 6 was not read: %v", report.Result.PerLine)
		}
		if answers[ruleID] != "undecided" {
			t.Fatalf("a null quantity answered %q; a value the row does not hold is undecided, not a failed check (answers: %v)",
				answers[ruleID], answers)
		}
	})
}

// installWithinRule adds a declared multi-input operator, the rule that uses it,
// the implementation selection the operator needs, and the application's
// declaration that it uses the rule. Nothing else in the Workspace changes.
func installWithinRule(t *testing.T, root string) {
	t.Helper()
	document := "# 数量の範囲\n\n" +
		"数量が仕様の範囲にあることを述べる。範囲は両端を含む。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.within:\n" +
		"  takes value: number\n" +
		"  takes low: number\n" +
		"  takes high: number\n\n" +
		"rule sales.order_line.quantity_in_range:\n" +
		"  sales.order_line.quantity is sales.within low: 3 high: 100\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "quantity-range.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") || strings.Contains(string(content), "implementations:") {
		t.Fatalf("the reference manifest already declares a language or an implementation selection; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n  - operator: sales.within\n    use: tei.number.within\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
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
	edited := replaceOnce(t, string(content), anchor, anchor+"    uses validation sales.order_line.quantity_in_range\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
