//go:build !tei_order_only && !tei_repair_only

package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// A multi-input operator rule is decided against real PostgreSQL rows here.
// plan/NOW.md road item 5 says the next product change is to pass
// `within(value, low, high)` through `tei view -environment` over real rows and
// check the per-key answers against an independent expectation; #5850 wired the
// selected implementations into detail.Decide but nothing read a real row with
// them.
//
// The expectations below are written by hand from the seed and the contract the
// design fixes (low <= value <= high, the subject supplies value, low and high
// are assigned by name). They are not read back from the run:
//
//	order_line_id | order_id | line_number | quantity | unit_price
//	1             | 1001     | 1           | 2        | 100
//	2             | 1001     | 2           | 0        | 250
//	3             | 1001     | 2           | 5        | 0
//	4             | 1002     | 1           | 3        | 100
//	5             | 1002     | 2           | 12       | 250
//
// With `low: 0 high: unit_price`, row 3 (quantity 5, unit_price 0) is the only
// row where the value is above its upper bound, so it is the only
// not-satisfied answer.
func TestWithinOperatorDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("refusing to modify the database without the disposable-database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
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
	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatal(err)
	}

	// A copy of the reference Workspace that declares the operator, selects the
	// shipped implementation for it, and uses it in one rule. The reference
	// Workspace itself is unchanged: its accepted state only moves through the
	// shared acceptance path.
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\nimplementations:\n  - operator: sales.within\n    use: tei.number.within\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	ruleDoc := "```tei\nnamespace sales\n\noperator sales.within:\n" +
		"  takes value: number\n  takes low: number\n  takes high: number\n\n" +
		"rule sales.order_line.quantity_within_price:\n" +
		"  sales.order_line.quantity is sales.within low: 0 high: sales.order_line.unit_price\n```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "quantity-within-price.tei.md"), []byte(ruleDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	appContent, err := os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	edited := replaceOnce(t, string(appContent), anchor,
		anchor+"    uses validation sales.order_line.quantity_within_price\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	binary := buildCLIForLifecycle(t)
	validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
	if validated.code != 0 {
		t.Fatalf("the Workspace did not validate: %s%s", validated.stdout, validated.stderr)
	}
	t.Setenv("TEI_VIEW_PG_DSN", dsn)

	for _, tc := range []struct {
		order   string
		answers map[string]string
	}{
		{
			// Only row 3 has quantity above its unit_price.
			order:   "1001",
			answers: map[string]string{"1": "satisfied", "2": "satisfied", "3": "not-satisfied"},
		},
		{
			order:   "1002",
			answers: map[string]string{"4": "satisfied", "5": "satisfied"},
		},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines", "-environment", "development",
				"-input", "order_id="+tc.order, "-format", "json",
			)
			if got.code != 0 {
				t.Fatalf("view failed: %s%s", got.stdout, got.stderr)
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					PerRow map[string]map[string]string `json:"perRow"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK {
				t.Fatalf("cannot read the report: %v\n%s", err, got.stdout)
			}
			if len(report.Result.PerRow) != len(tc.answers) {
				t.Fatalf("rows=%d, want %d: %+v", len(report.Result.PerRow), len(tc.answers), report.Result.PerRow)
			}
			for key, want := range tc.answers {
				rows, ok := report.Result.PerRow[key]
				if !ok {
					t.Fatalf("order %s has no row %s: %+v", tc.order, key, report.Result.PerRow)
				}
				if gotAnswer := rows["sales.order_line.quantity_within_price"]; gotAnswer != want {
					t.Fatalf("order %s row %s: within=%q, want %q", tc.order, key, gotAnswer, want)
				}
			}
		})
	}
}
