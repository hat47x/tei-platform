package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

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
			// Validate the exact Workspace revision whose rules the real
			// PostgreSQL read will evaluate. A different Workspace validating
			// successfully must not stand in for this read attempt.
			validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
			if validated.code != 0 {
				t.Fatalf("stage=%s: the selected Workspace did not validate before the database read: code=%d stdout=%s stderr=%s",
					stage.name, validated.code, validated.stdout, validated.stderr)
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
								OrderLineID string `json:"key"`
								Fields      map[string]struct {
									State string `json:"state"`
									Value any    `json:"value"`
								} `json:"fields"`
							} `json:"rows"`
							PerLine           map[string]map[string]string `json:"perRow"`
							PerOrder          map[string]string            `json:"perRelation"`
							PerOrderUndecided map[string]string            `json:"perRelationUndecided"`
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
					// The HTML must come from this same real PostgreSQL and
					// Workspace revision, not from a synthetic renderer fixture.
					assertDestinationHTMLFromPostgres(t, binary, root, orderID, dsn,
						want.PerLine, report.Result.PerOrder, report.Result.PerOrderUndecided)
				})
			}
		})
	}

	// Nothing in the entry names the reference Workspace. The same rows are
	// read through an Environment declared under another name, with no
	// change to the program.
	t.Run("an Environment under another name", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		directory := filepath.Join(root, "environments", "development")
		renamed := 0
		err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".tei") {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			renamed += strings.Count(string(content), "development")
			return os.WriteFile(path, []byte(strings.ReplaceAll(string(content), "development", "staging")), 0o644)
		})
		if err != nil || renamed == 0 {
			t.Fatalf("the Environment was not renamed: renamed=%d err=%v", renamed, err)
		}
		if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
			t.Fatalf("the renamed Workspace did not validate: %s%s", validated.stdout, validated.stderr)
		}
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-environment", "staging",
			"-input", "order_id=1002", "-format", "json",
		)
		var report struct {
			OK     bool `json:"ok"`
			Result struct {
				KeyAttributes []string                     `json:"keyAttributes"`
				PerRow        map[string]map[string]string `json:"perRow"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK || got.code != 0 {
			t.Fatalf("code=%d err=%v stdout=%s stderr=%s", got.code, err, got.stdout, got.stderr)
		}
		var expected struct {
			Orders map[string]struct {
				PerLine map[string]map[string]string `json:"perLine"`
			} `json:"orders"`
		}
		raw, err := os.ReadFile(filepath.Join(fixtures, "expected-before.json"))
		if err != nil || json.Unmarshal(raw, &expected) != nil {
			t.Fatalf("cannot read the expectation: %v", err)
		}
		if !reflect.DeepEqual(report.Result.PerRow, expected.Orders["1002"].PerLine) ||
			!reflect.DeepEqual(report.Result.KeyAttributes, []string{"sales.order_line.order_line_id"}) {
			t.Fatalf("perRow=%v keyAttributes=%v", report.Result.PerRow, report.Result.KeyAttributes)
		}
	})

	// A second Pattern is declared only in this disposable Workspace. The
	// runtime must select it from the actual Application/Binding declarations,
	// not recognize just the first destination's fixed Pattern ID.
	t.Run("a second declared header-detail Pattern in a copied Workspace", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		for _, relative := range []string{
			"applications/order-entry/patterns/order-lines.tei",
			"applications/order-entry/realizations/header-detail/order-lines.tei",
		} {
			original := filepath.Join(root, filepath.FromSlash(relative))
			body, err := os.ReadFile(original)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "order_entry.order_lines") {
				t.Fatalf("Pattern ID not present in %s", relative)
			}
			newPath := strings.TrimSuffix(original, ".tei") + "-probe.tei"
			updated := strings.ReplaceAll(string(body), "order_entry.order_lines", "order_entry.order_lines_probe")
			if err := os.WriteFile(newPath, []byte(updated), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
		if validated.code != 0 {
			t.Fatalf("second Pattern Workspace did not validate: stdout=%s stderr=%s", validated.stdout, validated.stderr)
		}
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines_probe", "-environment", "development",
			"-input", "order_id=1002", "-format", "json",
		)
		var report struct {
			OK     bool `json:"ok"`
			Result struct {
				Pattern  string                       `json:"pattern"`
				Relation string                       `json:"relation"`
				Coverage string                       `json:"coverage"`
				PerRow   map[string]map[string]string `json:"perRow"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || got.code != 0 || !report.OK {
			t.Fatalf("second Pattern failed: code=%d err=%v stdout=%s stderr=%s", got.code, err, got.stdout, got.stderr)
		}
		if report.Result.Pattern != "order_entry.order_lines_probe" ||
			report.Result.Relation != "sales.order.lines" || report.Result.Coverage != "partial" {
			t.Fatalf("second Pattern lost its declared identity or coverage: %+v", report.Result)
		}
		raw, err := os.ReadFile(filepath.Join(fixtures, "expected-before.json"))
		if err != nil {
			t.Fatal(err)
		}
		var want struct {
			Orders map[string]struct {
				PerLine map[string]map[string]string `json:"perLine"`
			} `json:"orders"`
		}
		if err := json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(report.Result.PerRow, want.Orders["1002"].PerLine) {
			t.Fatalf("second Pattern changed the independent row decisions: got=%v want=%v",
				report.Result.PerRow, want.Orders["1002"].PerLine)
		}
	})

	t.Run("an input the Contract Query does not declare is refused", func(t *testing.T) {
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", filepath.Join("..", "..", "reference", "workspace"), "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-environment", "development",
			"-input", "order_id=1001", "-input", "customer_id=1", "-format", "json",
		)
		if got.code == 0 || !strings.Contains(got.stdout, "customer_id is not an input of the Contract Query") || !strings.Contains(got.stdout, "INVALID_ARGUMENT") {
			t.Fatalf("code=%d stdout=%s", got.code, got.stdout)
		}
	})

	// The Workspace owns SQL realization text. A read-only CLI must not
	// trust that source to be SELECT-only: PostgreSQL accepts a data-changing
	// CTE with a SELECT result. Exercise the real CLI and independently read
	// the same disposable database afterward.
	t.Run("an edited Contract Query cannot write through view", func(t *testing.T) {
		const editedSQL = `WITH changed AS (
    UPDATE public.order_lines SET quantity = 777 WHERE order_id = :order_id
    RETURNING order_line_id, line_number, item_code, quantity, unit_price
)
SELECT order_line_id, line_number, item_code, quantity, unit_price
FROM changed ORDER BY line_number, order_line_id`
		var before string
		if err := db.QueryRow("SELECT quantity::text FROM public.order_lines WHERE order_line_id = 1").Scan(&before); err != nil {
			t.Fatal(err)
		}
		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		sourcePath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
		if err := os.WriteFile(sourcePath, []byte(editedSQL), 0o644); err != nil {
			t.Fatal(err)
		}
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-environment", "development",
			"-input", "order_id=1001", "-format", "json",
		)
		if got.code == 0 || strings.Contains(got.stdout+got.stderr, dsn) {
			t.Fatalf("edited SQL was accepted or exposed the DSN: exit=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
		}
		var report struct {
			OK    bool `json:"ok"`
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || report.OK || report.Error.Code != "VIEW_EXECUTION_FAILED" {
			t.Fatalf("edited SQL did not fail through the runtime view contract: report=%+v err=%v output=%s", report, err, got.stdout)
		}
		var after string
		if err := db.QueryRow("SELECT quantity::text FROM public.order_lines WHERE order_line_id = 1").Scan(&after); err != nil || after != before {
			t.Fatalf("read-only view changed the stored quantity: before=%s after=%s err=%v", before, after, err)
		}

		// This is a real UPDATE, not merely malformed SQL: on the same
		// disposable database it returns three modified rows under an
		// ordinary writable transaction. Rolling back keeps other cases'
		// independent seed and expectations intact.
		probe, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer probe.Rollback()
		rows, err := probe.Query(strings.ReplaceAll(editedSQL, ":order_id", "$1"), int64(1001))
		if err != nil {
			t.Fatalf("the negative SQL was not executable in a writable transaction: %v", err)
		}
		var count int
		for rows.Next() {
			count++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		if count != 3 {
			t.Fatalf("the writable control updated %d rows, want 3", count)
		}
		if err := probe.Rollback(); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT quantity::text FROM public.order_lines WHERE order_line_id = 1").Scan(&after); err != nil || after != before {
			t.Fatalf("the writable control did not roll back: before=%s after=%s err=%v", before, after, err)
		}
	})

	// With `covers sales.order.lines` in the binding, the reference Query is
	// matched against the relation storage and the read is complete. The
	// answers about all the lines are then the independent expectations as
	// written, "satisfied" included. The declaration is the only change.
	t.Run("the declared reference Query reads every line", func(t *testing.T) {
		rawExpected, err := os.ReadFile(filepath.Join(fixtures, "expected-before.json"))
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
		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		bindingPath := filepath.Join(root, "applications", "order-entry", "realizations", "header-detail", "order-lines.tei")
		binding, err := os.ReadFile(bindingPath)
		if err != nil {
			t.Fatal(err)
		}
		const anchor = "    input sales.order.order_id to order_id\n"
		if strings.Count(string(binding), anchor) != 1 || strings.Contains(string(binding), "covers ") {
			t.Fatalf("unexpected reference binding: %s", binding)
		}
		declared := strings.Replace(string(binding), anchor, anchor+"    covers sales.order.lines\n", 1)
		if err := os.WriteFile(bindingPath, []byte(declared), 0o644); err != nil {
			t.Fatal(err)
		}
		if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
			t.Fatalf("the declared Workspace did not validate: %s %s", validated.stdout, validated.stderr)
		}
		for _, orderID := range []string{"1001", "1002", "1003"} {
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines", "-environment", "development",
				"-input", "order_id="+orderID, "-format", "json",
			)
			if got.code != 0 || strings.Contains(got.stdout+got.stderr, dsn) {
				t.Fatalf("order=%s: exit=%d stdout=%s stderr=%s", orderID, got.code, got.stdout, got.stderr)
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					Coverage          string                       `json:"coverage"`
					CoverageDeclared  string                       `json:"coverageDeclared"`
					CoverageBasis     string                       `json:"coverageBasis"`
					CoverageReason    string                       `json:"coverageReason"`
					PerLine           map[string]map[string]string `json:"perRow"`
					PerOrder          map[string]string            `json:"perRelation"`
					PerOrderUndecided map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
			}
			want := expected.Orders[orderID]
			if !report.OK || report.Result.Coverage != "complete" || report.Result.CoverageDeclared != "sales.order.lines" ||
				!strings.Contains(report.Result.CoverageBasis, "order_id = sales.order.order_id") || report.Result.CoverageReason != "" {
				t.Fatalf("order=%s: coverage=%q declared=%q basis=%q reason=%q", orderID,
					report.Result.Coverage, report.Result.CoverageDeclared, report.Result.CoverageBasis, report.Result.CoverageReason)
			}
			if !reflect.DeepEqual(report.Result.PerOrder, want.PerOrder) || len(report.Result.PerOrderUndecided) != 0 {
				t.Fatalf("order=%s perRelation=%v undecided=%v, expected independently %v",
					orderID, report.Result.PerOrder, report.Result.PerOrderUndecided, want.PerOrder)
			}
			if !reflect.DeepEqual(report.Result.PerLine, want.PerLine) {
				t.Fatalf("order=%s perRow=%v, expected independently %v", orderID, report.Result.PerLine, want.PerLine)
			}
		}
	})

	// A Contract Query can return fewer than the rows a rule needs. The
	// reader must not infer relation completeness from a successful SELECT,
	// whether that SELECT returns unique-looking rows or no rows at all.
	// The same observed duplicate, however, is enough to refute uniqueness.
	for _, tc := range []struct {
		name       string
		filter     string
		wantIDs    []string
		wantAnswer string
		// Covers only describes the author's claim, not observed completeness.
		covers bool
	}{
		{
			name:       "a filter hides the second duplicate",
			filter:     " AND order_line_id <> 3",
			wantIDs:    []string{"1", "2"},
			wantAnswer: "undecided",
		},
		{
			name:       "a false covers declaration must not prove uniqueness",
			filter:     " AND order_line_id <> 3",
			wantIDs:    []string{"1", "2"},
			wantAnswer: "undecided",
			covers:     true,
		},
		{
			name:       "a filter keeps a witnessed duplicate",
			filter:     " AND order_line_id IN (2, 3)",
			wantIDs:    []string{"2", "3"},
			wantAnswer: "not-satisfied",
		},
		{
			name:       "an empty result is not proof of uniqueness",
			filter:     " AND 1 = 0",
			wantIDs:    []string{},
			wantAnswer: "undecided",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
			sourcePath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			const predicate = "WHERE order_id = :order_id"
			if strings.Count(string(source), predicate) != 1 {
				t.Fatalf("the reference Query's parent predicate changed: %s", source)
			}
			limited := strings.Replace(string(source), predicate, predicate+tc.filter, 1)
			if err := os.WriteFile(sourcePath, []byte(limited), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.covers {
				// A plausible declaration cannot turn a filtered SQL result into
				// independent evidence that the complete Relation was observed.
				bindingPath := filepath.Join(root, "applications", "order-entry", "realizations", "header-detail", "order-lines.tei")
				binding, err := os.ReadFile(bindingPath)
				if err != nil {
					t.Fatal(err)
				}
				const anchor = "    input sales.order.order_id to order_id\n"
				if strings.Count(string(binding), anchor) != 1 || strings.Contains(string(binding), "covers ") {
					t.Fatalf("unexpected reference binding: %s", binding)
				}
				declared := strings.Replace(string(binding), anchor, anchor+"    covers sales.order.lines\n", 1)
				if err := os.WriteFile(bindingPath, []byte(declared), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines", "-environment", "development",
				"-input", "order_id=1001", "-format", "json",
			)
			if got.code != 0 || strings.Contains(got.stdout+got.stderr, dsn) {
				t.Fatalf("narrowed Query failed or exposed the DSN: exit=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			var report struct {
				OK     bool `json:"ok"`
				Result struct {
					Coverage         string `json:"coverage"`
					CoverageDeclared string `json:"coverageDeclared"`
					CoverageReason   string `json:"coverageReason"`
					Rows             []struct {
						OrderLineID string `json:"key"`
					} `json:"rows"`
					PerLine           map[string]map[string]string `json:"perRow"`
					PerOrder          map[string]string            `json:"perRelation"`
					PerOrderUndecided map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
				t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
			}
			observedIDs := make([]string, 0, len(report.Result.Rows))
			for _, row := range report.Result.Rows {
				observedIDs = append(observedIDs, row.OrderLineID)
			}
			if !report.OK || report.Result.Coverage != "partial" ||
				!reflect.DeepEqual(observedIDs, tc.wantIDs) || len(report.Result.PerLine) != len(tc.wantIDs) {
				t.Fatalf("Query returned the wrong observed range: ok=%v coverage=%s IDs=%v perLine=%v, want IDs=%v",
					report.OK, report.Result.Coverage, observedIDs, report.Result.PerLine, tc.wantIDs)
			}
			// The report says why the read is partial. With the declaration,
			// it is the extra condition the matcher does not read, not a
			// missing declaration.
			wantReason, wantDeclared := "does not state", ""
			if tc.covers {
				wantReason, wantDeclared = "not a plain read", "sales.order.lines"
			}
			if report.Result.CoverageDeclared != wantDeclared || !strings.Contains(report.Result.CoverageReason, wantReason) {
				t.Fatalf("coverage declared=%q reason=%q, want declared=%q reason containing %q",
					report.Result.CoverageDeclared, report.Result.CoverageReason, wantDeclared, wantReason)
			}
			const uniqueness = "sales.order.lines.line_number_unique"
			answer := report.Result.PerOrder[uniqueness]
			reason := report.Result.PerOrderUndecided[uniqueness]
			if answer != tc.wantAnswer || (answer == "undecided" && !strings.Contains(reason, "relation-partially-read")) ||
				(answer == "not-satisfied" && reason != "") {
				t.Fatalf("the filtered Query changed uniqueness improperly: answer=%q reason=%q, want %q", answer, reason, tc.wantAnswer)
			}
		})
	}

	// A second real PostgreSQL target must exercise a different semantic Data
	// and Relation through the same public CLI entry. Renaming only a Pattern
	// tests selection; this also changes the semantic Attribute IDs and the
	// physical table while preserving independently specified values.
	t.Run("another Data and Relation read from another PostgreSQL table", func(t *testing.T) {
		if _, err := db.Exec("drop table if exists public.inventory_entries"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("create table public.inventory_entries (like public.order_lines including all)"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := db.Exec("drop table if exists public.inventory_entries"); err != nil {
				t.Errorf("cannot clean up disposable inventory table: %v", err)
			}
		})
		if _, err := db.Exec("insert into public.inventory_entries select * from public.order_lines"); err != nil {
			t.Fatal(err)
		}
		// Deliberately differ from the reference table: a hidden fallback to
		// public.order_lines must fail the independently fixed expectation.
		if _, err := db.Exec("update public.inventory_entries set quantity = 0 where order_line_id = 4"); err != nil {
			t.Fatal(err)
		}

		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		changedFiles := 0
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || (!strings.HasSuffix(path, ".tei") && !strings.HasSuffix(path, ".sql")) {
				return nil
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			before := string(raw)
			after := strings.ReplaceAll(before, "sales.order.lines", "sales.order.inventory_entries")
			after = strings.ReplaceAll(after, "sales.order_line", "sales.inventory_entry")
			after = strings.ReplaceAll(after, "relation lines to sales.inventory_entry", "relation inventoryEntries to sales.inventory_entry")
			after = strings.ReplaceAll(after, "data OrderLine {", "data InventoryEntry {")
			after = strings.ReplaceAll(after, "type OrderLineId from", "type InventoryEntryId from")
			after = strings.ReplaceAll(after, "public.order_lines", "public.inventory_entries")
			if before != after {
				changedFiles++
				return os.WriteFile(path, []byte(after), 0o644)
			}
			return nil
		})
		if err != nil || changedFiles < 7 {
			t.Fatalf("alternate Workspace source rewrite failed: changed=%d err=%v", changedFiles, err)
		}
		validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
		if validated.code != 0 {
			t.Fatalf("alternate semantic Data/Relation Workspace did not validate: code=%d stdout=%s stderr=%s",
				validated.code, validated.stdout, validated.stderr)
		}

		const quantityRule = "sales.inventory_entry.quantity_positive"
		const positionRule = "sales.inventory_entry.line_number_positive"
		const uniqueRule = "sales.order.inventory_entries.line_number_unique"
		for _, tc := range []struct {
			orderID string
			ids     []string
			amounts []string
			unique  string
		}{
			{orderID: "1001", ids: []string{"1", "2", "3"}, amounts: []string{"satisfied", "not-satisfied", "satisfied"}, unique: "not-satisfied"},
			{orderID: "1002", ids: []string{"4", "5"}, amounts: []string{"not-satisfied", "satisfied"}, unique: "undecided"},
			{orderID: "1003", ids: []string{}, amounts: []string{}, unique: "undecided"},
		} {
			t.Run(tc.orderID, func(t *testing.T) {
				got := runCLIProcessWithStreams(t, binary,
					"view", "-workspace", root, "-application", "order_entry",
					"-pattern", "order_entry.order_lines", "-environment", "development",
					"-input", "order_id="+tc.orderID, "-format", "json")
				if got.code != 0 || strings.Contains(got.stdout+got.stderr, dsn) {
					t.Fatalf("alternate Data/Relation read failed or exposed DSN: code=%d stdout=%s stderr=%s",
						got.code, got.stdout, got.stderr)
				}
				var report struct {
					OK     bool `json:"ok"`
					Result struct {
						Relation      string   `json:"relation"`
						KeyAttributes []string `json:"keyAttributes"`
						Coverage      string   `json:"coverage"`
						Rows          []struct {
							Key    string `json:"key"`
							Fields map[string]struct {
								State string `json:"state"`
							} `json:"fields"`
						} `json:"rows"`
						PerRow               map[string]map[string]string `json:"perRow"`
						PerRelation          map[string]string            `json:"perRelation"`
						PerRelationUndecided map[string]string            `json:"perRelationUndecided"`
					} `json:"result"`
				}
				if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
					t.Fatalf("alternate Result is not JSON: %v (%s)", err, got.stdout)
				}
				if !report.OK || report.Result.Relation != "sales.order.inventory_entries" ||
					report.Result.Coverage != "partial" ||
					!reflect.DeepEqual(report.Result.KeyAttributes, []string{"sales.inventory_entry.order_line_id"}) {
					t.Fatalf("alternate Result lost semantic identity or claimed full coverage: %+v", report.Result)
				}
				observedIDs := make([]string, 0, len(report.Result.Rows))
				for i, row := range report.Result.Rows {
					observedIDs = append(observedIDs, row.Key)
					if row.Fields["sales.inventory_entry.quantity"].State != "present" ||
						report.Result.PerRow[row.Key][quantityRule] != tc.amounts[i] ||
						report.Result.PerRow[row.Key][positionRule] != "satisfied" {
						t.Fatalf("alternate data rule mismatch on row %s: fields=%v decisions=%v",
							row.Key, row.Fields, report.Result.PerRow[row.Key])
					}
				}
				if !reflect.DeepEqual(observedIDs, tc.ids) ||
					report.Result.PerRelation[uniqueRule] != tc.unique {
					t.Fatalf("alternate Data/Relation changed independent expected identity/answers: rows=%v unique=%s",
						observedIDs, report.Result.PerRelation[uniqueRule])
				}
				if tc.unique == "undecided" && report.Result.PerRelationUndecided[uniqueRule] == "" {
					t.Fatal("unproven relation completeness was silently treated as full coverage")
				}
			})
		}
	})

	t.Run("different attributes, Query input and composite primary key", func(t *testing.T) {
		checkDestinationAlternateModel(t, db, binary, dsn)
	})

}

// assertDestinationHTMLFromPostgres reads the real database a second time
// through the same product CLI entry, using HTML rather than JSON. The expected
// row/rule answers come from hand-authored fixture files, not generated HTML.
// This is a product CLI HTML check, not evidence of a browser or Web Target E2E.
func assertDestinationHTMLFromPostgres(
	t *testing.T,
	binary, root, orderID, dsn string,
	perLine map[string]map[string]string,
	perRelation map[string]string,
	undecided map[string]string,
) {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+orderID,
		"-format", "html")
	if got.code != 0 {
		t.Fatalf("order=%s HTML CLI failed: code=%d stderr=%s", orderID, got.code, got.stderr)
	}
	page := got.stdout
	if !strings.Contains(page, "<!doctype html>") ||
		!strings.Contains(page, "coverage: partial") ||
		!strings.Contains(page, "rows: "+strconv.Itoa(len(perLine))) {
		t.Fatalf("order=%s HTML missing document, observed coverage or row count", orderID)
	}
	if strings.Contains(page+got.stderr, dsn) {
		t.Fatal("real PostgreSQL connection string leaked into HTML or diagnostics")
	}
	if len(perLine) == 0 {
		if !strings.Contains(page, "明細の行はない。") || strings.Contains(page, "<h3>") {
			t.Fatalf("order=%s HTML did not preserve the empty row set", orderID)
		}
	}
	for key, rules := range perLine {
		marker := "<h3>" + html.EscapeString(key) + "</h3>"
		if strings.Count(page, marker) != 1 {
			t.Fatalf("order=%s row key %s is missing or duplicated in HTML", orderID, key)
		}
		start := strings.Index(page, marker) + len(marker)
		end := len(page)
		if next := strings.Index(page[start:], "<h3>"); next >= 0 {
			end = start + next
		}
		if next := strings.Index(page[start:end], "<h2>"); next >= 0 {
			end = start + next
		}
		rowHTML := page[start:end]
		for rule, answer := range rules {
			cells := "<td>" + html.EscapeString(rule) + "</td><td>" + html.EscapeString(answer) + "</td>"
			if !strings.Contains(rowHTML, cells) {
				t.Fatalf("order=%s row=%s rule=%s: HTML answer is not %s on this exact row", orderID, key, rule, answer)
			}
		}
	}
	marker := "<h2>関係全体の答え</h2>"
	off := strings.Index(page, marker)
	if off < 0 {
		t.Fatalf("order=%s HTML has no separate Relation answers", orderID)
	}
	relationHTML := page[off+len(marker):]
	for rule, answer := range perRelation {
		cells := "<td>" + html.EscapeString(rule) + "</td><td>" + html.EscapeString(answer) + "</td>"
		if !strings.Contains(relationHTML, cells) {
			t.Fatalf("order=%s Relation=%s: HTML answer is not %s", orderID, rule, answer)
		}
		if reason := undecided[rule]; reason != "" && !strings.Contains(relationHTML, html.EscapeString(reason)) {
			t.Fatalf("order=%s Relation=%s: HTML lost undecided reason", orderID, rule)
		}
	}
	if browser := os.Getenv("TEI_VIEW_BROWSER_BINARY"); browser != "" {
		assertDestinationChromeDOMFromPostgres(t, browser, page, orderID, dsn, perLine, perRelation, undecided)
	}
}

// assertDestinationChromeDOMFromPostgres opens the HTML produced from the
// actual PostgreSQL read in Chrome. It checks DOM receipt against independent
// fixture expectations, not against the CLI's own JSON serialization.
// This does not qualify the separate Web Target's HTTP/Provider connection.
func assertDestinationChromeDOMFromPostgres(
	t *testing.T,
	browser, page, orderID, dsn string,
	perLine map[string]map[string]string,
	perRelation map[string]string,
	undecided map[string]string,
) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "order-lines.html")
	if err := os.WriteFile(path, []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, browser,
		"--headless", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
		"--no-first-run", "--dump-dom", "file://"+filepath.ToSlash(path))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real Chrome failed to open the generated page: %v", err)
	}
	dom := string(output)
	if !strings.Contains(dom, "<html") ||
		!strings.Contains(dom, "coverage: partial") ||
		!strings.Contains(dom, "rows: "+strconv.Itoa(len(perLine))) ||
		strings.Contains(dom, dsn) {
		t.Fatalf("order %s: Chrome DOM lost read metadata or exposed a DB secret", orderID)
	}
	// The template uses h3 only for row keys. Chrome can reserialize HTML
	// entities in text nodes, so compare decoded heading contents.
	sections := strings.Split(dom, "<h3>")
	rows := make(map[string]string, len(sections)-1)
	for _, section := range sections[1:] {
		close := strings.Index(section, "</h3>")
		if close < 0 {
			t.Fatal("Chrome DOM has a row heading without a closing tag")
		}
		key := html.UnescapeString(section[:close])
		if _, duplicate := rows[key]; duplicate {
			t.Fatalf("Chrome DOM repeated row key %s", key)
		}
		body := section[close+len("</h3>"):]
		if end := strings.Index(body, "<h2>"); end >= 0 {
			body = body[:end]
		}
		rows[key] = html.UnescapeString(body)
	}
	if len(rows) != len(perLine) {
		t.Fatalf("order %s: Chrome DOM rows=%d; independent expected=%d", orderID, len(rows), len(perLine))
	}
	if len(perLine) == 0 && !strings.Contains(dom, "明細の行はない。") {
		t.Fatal("Chrome lost the zero-row message")
	}
	for key, rules := range perLine {
		body, ok := rows[key]
		if !ok {
			t.Fatalf("order %s: Chrome lost row key %s", orderID, key)
		}
		for rule, answer := range rules {
			cells := "<td>" + rule + "</td><td>" + answer + "</td>"
			if !strings.Contains(body, cells) {
				t.Fatalf("order %s row %s: Chrome lost decision %s=%s", orderID, key, rule, answer)
			}
		}
	}
	marker := "<h2>関係全体の答え</h2>"
	off := strings.Index(dom, marker)
	if off < 0 {
		t.Fatal("Chrome lost the separate relation answers")
	}
	relationHTML := html.UnescapeString(dom[off+len(marker):])
	for rule, answer := range perRelation {
		cells := "<td>" + rule + "</td><td>" + answer + "</td>"
		if !strings.Contains(relationHTML, cells) {
			t.Fatalf("order %s: Chrome lost relation decision %s=%s", orderID, rule, answer)
		}
		if reason := undecided[rule]; reason != "" && !strings.Contains(relationHTML, reason) {
			t.Fatalf("order %s: Chrome lost undecided reason for %s", orderID, rule)
		}
	}
	t.Logf("actual Chrome DOM verified for order %s", orderID)
}
