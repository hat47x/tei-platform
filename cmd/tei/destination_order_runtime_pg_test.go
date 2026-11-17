package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// This test uses an isolated, disposable PostgreSQL database and executes the
// actual tei CLI. The expected answers are read from NOW.md's independent
// fixtures, never copied from the product's decision output.
func TestDestinationOrderChangeReadViaCLIFromPostgres(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL destination not run")
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
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
			t.Errorf("cannot clean up disposable order lines: %v", err)
		}
	})
	fixtures := filepath.Join("..", "..", "testdata", "destination-order-change")
	seed, err := os.ReadFile(filepath.Join(fixtures, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("cannot seed order lines: %v", err)
	}
	editedRule, err := os.ReadFile(filepath.Join(fixtures, "edit", "unit-price.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	binary := buildCLIForLifecycle(t)

	for _, stage := range []struct {
		name     string
		expected string
		edited   bool
	}{
		{name: "before", expected: "expected-before.json"},
		{name: "after", expected: "expected-after.json", edited: true},
	} {
		t.Run(stage.name, func(t *testing.T) {
			root := t.TempDir()
			copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
			if stage.edited {
				applyDestinationEdit(t, root, string(editedRule))
			}
			rawExpected, err := os.ReadFile(filepath.Join(fixtures, stage.expected))
			if err != nil {
				t.Fatal(err)
			}
			var expected struct {
				Orders map[string]struct {
					PerLine  map[string]map[string]string `json:"perLine"`
					PerOrder map[string]string            `json:"perOrder"`
				} `json:"orders"`
			}
			if err := json.Unmarshal(rawExpected, &expected); err != nil {
				t.Fatal(err)
			}
			for _, orderID := range []string{"1001", "1002", "1003"} {
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
								OrderLineID string `json:"orderLineId"`
								Fields      map[string]struct {
									State string `json:"state"`
									Value any    `json:"value"`
								} `json:"fields"`
							} `json:"rows"`
							PerLine           map[string]map[string]string `json:"perLine"`
							PerOrder          map[string]string            `json:"perOrder"`
							PerOrderUndecided map[string]string            `json:"perOrderUndecided"`
						} `json:"result"`
					}
					if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
						t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
					}
					if !report.OK || report.Result.Coverage != "partial" {
						t.Fatalf("CLI claimed complete coverage or failed: %+v", report)
					}
					want := expected.Orders[orderID]
					if !reflect.DeepEqual(report.Result.PerLine, want.PerLine) {
						t.Fatalf("order=%s perLine=%v, expected independently %v", orderID, report.Result.PerLine, want.PerLine)
					}
					if len(report.Result.Rows) != len(want.PerLine) {
						t.Fatalf("order=%s rows=%d, want=%d", orderID, len(report.Result.Rows), len(want.PerLine))
					}
					seen := map[string]bool{}
					for _, row := range report.Result.Rows {
						if _, err := strconv.ParseInt(row.OrderLineID, 10, 64); err != nil || seen[row.OrderLineID] {
							t.Fatalf("order=%s repeated or invalid line identity %s", orderID, row.OrderLineID)
						}
						seen[row.OrderLineID] = true
						if row.Fields["sales.order_line.quantity"].State != "present" {
							t.Fatalf("order=%s line=%s has no observed quantity", orderID, row.OrderLineID)
						}
					}
					for rule, answer := range want.PerOrder {
						gotAnswer := report.Result.PerOrder[rule]
						if answer == "not-satisfied" {
							// A witnessed duplicate is decisive even if there
							// may be more rows not yet read.
							if gotAnswer != answer {
								t.Fatalf("order=%s rule=%s: %s, want %s", orderID, rule, gotAnswer, answer)
							}
						} else if gotAnswer != "undecided" || report.Result.PerOrderUndecided[rule] == "" {
							// A successful query does not prove a complete
							// relation; until the Contract binding says so,
							// a positive uniqueness claim must stay open.
							t.Fatalf("order=%s rule=%s: should be undecided without coverage proof, got %s", orderID, rule, gotAnswer)
						}
					}
				})
			}
		})
	}
}
