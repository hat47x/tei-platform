//go:build !tei_order_only && !tei_repair_only

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

// An operator that reads every row of a relation, written `of each row`, and a
// declaration whose witnessed counterexample is decisive, written `refutes on
// some rows`, are decided through `tei view -environment` over real PostgreSQL
// rows here. plan/NOW.md road item 5 names this as the next product change: the
// conversion carries both statements into the meaning, Detail.Decide asks the
// Detail once instead of once per row, and the answer lands in PerRelation.
//
// The expectations below are written by hand from the seed and the contract the
// design fixes. They are not read back from the run:
//
//	order_line_id | order_id | line_number | quantity
//	1             | 1001     | 1           | 2
//	2             | 1001     | 2           | 0
//	3             | 1001     | 2           | 5
//	4             | 1002     | 1           | 3
//	5             | 1002     | 2           | 12
//
// Every quantity is non-negative, so with every row read the rule holds. The
// counterexample is a row this test inserts with a negative quantity: it
// refutes the rule whatever the coverage, but only because the declaration
// states that a witnessed counterexample is enough.
func TestRowScopedOperatorDecidesRealPostgresRows(t *testing.T) {
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
		quantity numeric(12,3),
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

	// A copy of the reference Workspace that declares the row-scoped operator,
	// selects the shipped implementation for it, and uses it in one rule. The
	// reference Workspace itself is unchanged: its accepted state only moves
	// through the shared acceptance path.
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\nimplementations:\n  - operator: sales.all_lines_non_negative\n    use: tei.number.non-negative\n")...)
	if err := os.WriteFile(manifest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	ruleDoc := "```tei\nnamespace sales\n\noperator sales.all_lines_non_negative:\n" +
		"  takes value: number of each row\n  refutes on some rows\n\n" +
		"rule sales.order_line.all_lines_non_negative:\n" +
		"  sales.order_line.quantity is sales.all_lines_non_negative per sales.order.lines\n```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "all-lines-non-negative.tei.md"), []byte(ruleDoc), 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	appContent, err := os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    uses validation sales.order_line.quantity_positive\n"
	edited := replaceOnce(t, string(appContent), anchor,
		anchor+"    uses validation sales.order_line.all_lines_non_negative\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	binary := buildCLIForLifecycle(t)
	validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
	if validated.code != 0 {
		t.Fatalf("the Workspace did not validate: %s%s", validated.stdout, validated.stderr)
	}
	t.Setenv("TEI_VIEW_PG_DSN", dsn)

	const rule = "sales.order_line.all_lines_non_negative"

	// answer reads the one answer the rule has for the relation. A rule that
	// reads every row has no per-row answer, so an answer that appears per row
	// is the dispatch being wrong rather than the data.
	answer := func(t *testing.T, workspace, order string) (string, string) {
		t.Helper()
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", workspace, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-environment", "development",
			"-input", "order_id="+order, "-format", "json",
		)
		if got.code != 0 {
			t.Fatalf("view failed: %s%s", got.stdout, got.stderr)
		}
		var report struct {
			OK     bool `json:"ok"`
			Result struct {
				Coverage             string                       `json:"coverage"`
				PerRow               map[string]map[string]string `json:"perRow"`
				PerRelation          map[string]string            `json:"perRelation"`
				PerRelationUndecided map[string]string            `json:"perRelationUndecided"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK {
			t.Fatalf("cannot read the report: %v\n%s", err, got.stdout)
		}
		for _, answers := range report.Result.PerRow {
			if _, perRow := answers[rule]; perRow {
				t.Fatalf("the rule was answered per row: %+v", report.Result.PerRow)
			}
		}
		return report.Result.PerRelation[rule], report.Result.PerRelationUndecided[rule]
	}

	for _, order := range []string{"1001", "1002"} {
		t.Run("every row read, order "+order, func(t *testing.T) {
			got, why := answer(t, root, order)
			if got != "satisfied" || why != "" {
				t.Fatalf("answer=%q why=%q, want satisfied with no reason", got, why)
			}
		})
	}

	// A row that holds no quantity is not a counterexample: the value the row
	// does not hold is reported as itself, before the coverage.
	t.Run("a row with no quantity", func(t *testing.T) {
		if _, err := db.Exec(`insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price)
			values (6, 1003, 1, 'D-400', null, 10)`); err != nil {
			t.Fatalf("cannot insert a row with no quantity: %v", err)
		}
		t.Cleanup(func() { _, _ = db.Exec("delete from public.order_lines where order_line_id = 6") })

		got, why := answer(t, root, "1003")
		if got != "undecided" || !strings.Contains(why, "value-null") {
			t.Fatalf("answer=%q why=%q, want undecided with value-null", got, why)
		}
	})

	// The witnessed counterexample, with every row read.
	t.Run("a witnessed counterexample with every row read", func(t *testing.T) {
		if _, err := db.Exec(`insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price)
			values (7, 1004, 1, 'D-500', -1, 10)`); err != nil {
			t.Fatalf("cannot insert the counterexample: %v", err)
		}
		t.Cleanup(func() { _, _ = db.Exec("delete from public.order_lines where order_line_id = 7") })

		got, why := answer(t, root, "1004")
		if got != "not-satisfied" || why != "" {
			t.Fatalf("answer=%q why=%q, want not-satisfied with no reason", got, why)
		}
	})

	// A Query that drops a row leaves the read partial. Without a witnessed
	// counterexample the rule is then undecided: an unread row may still hold
	// one, and `refutes on some rows` says a counterexample is enough, not
	// that the rows that were read are all the rows.
	t.Run("partial read, no counterexample", func(t *testing.T) {
		partial := partialReadWorkspace(t, root)
		got, why := answer(t, partial, "1001")
		if got != "undecided" || !strings.Contains(why, "relation-partially-read") {
			t.Fatalf("answer=%q why=%q, want undecided with relation-partially-read", got, why)
		}
	})

	// The same partial read with a witnessed counterexample: the declaration
	// decides it, because a counterexample among the rows that were read is
	// enough to say the rule does not hold.
	t.Run("partial read, witnessed counterexample", func(t *testing.T) {
		if _, err := db.Exec(`insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price)
			values (7, 1004, 1, 'D-500', -1, 10)`); err != nil {
			t.Fatalf("cannot insert the counterexample: %v", err)
		}
		t.Cleanup(func() { _, _ = db.Exec("delete from public.order_lines where order_line_id = 7") })

		partial := partialReadWorkspace(t, root)
		got, why := answer(t, partial, "1004")
		if got != "not-satisfied" || why != "" {
			t.Fatalf("answer=%q why=%q, want not-satisfied with no reason", got, why)
		}
	})
}

// partialReadWorkspace copies a Workspace and edits its Contract Query so the
// read returns only some of the rows. The binding still declares every row, so
// the qualification is what leaves the coverage partial.
func partialReadWorkspace(t *testing.T, root string) string {
	t.Helper()
	partial := t.TempDir()
	copyTreeForDestination(t, root, partial)
	queryPath := filepath.Join(partial, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
	source, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	const predicate = "WHERE order_id = :order_id"
	if strings.Count(string(source), predicate) != 1 {
		t.Fatalf("the reference Query's parent predicate changed: %s", source)
	}
	if err := os.WriteFile(queryPath, []byte(strings.Replace(string(source), predicate, predicate+" AND order_line_id <> 1", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	return partial
}
