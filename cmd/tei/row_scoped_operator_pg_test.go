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

// A rule whose operator declares `of each row` is asked of the relation, not of
// one record, and the product entry point `tei view -environment` has to report
// its answer in perRelation. The reader-level tests decide such a rule through
// validationread.Detail; this test takes the shipped CLI.
//
// The operator is the shipped tei.number.every-at-most, and its second input is
// the constant 5 (decision sections 3 and 4: an input may be an attribute or a
// constant). Every row of the relation therefore holds that one limit, and the
// comparison is row against row.
//
// The expected answers are written from
// testdata/destination-order-change/seed.sql and from "every quantity is at
// most 5", not read back from the CLI:
//
//	order 1001: quantity 2, 0, 5 -> every row is at most 5, so the rule holds
//	order 1002: quantity 3, 12  -> 12 is above 5, so the rule does not hold
//
// The contract states `refutes on some rows`, so the witnessed counterexample
// settles 1002 whatever the coverage.

const rowScopedProductDatabase = "tei_destination_order_e2e"

const rowScopedOperatorDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.quantity_at_most:\n" +
	"  takes value: number of each row\n" +
	"  takes limit: number\n" +
	"  refutes on some rows\n" +
	"\n" +
	"rule sales.order.lines.quantity_at_most_5:\n" +
	"  sales.order_line.quantity is sales.quantity_at_most limit: 5 per sales.order.lines\n" +
	"```\n"

// rowScopedProductReport is the JSON envelope `tei view -format json` writes.
type rowScopedProductReport struct {
	OK     bool                     `json:"ok"`
	Result *rowScopedProductResult  `json:"result"`
	Error  *rowScopedProductFailure `json:"error"`
}

type rowScopedProductResult struct {
	Coverage string `json:"coverage"`
	Rows     []struct {
		Key string `json:"key"`
	} `json:"rows"`
	PerRow               map[string]map[string]string `json:"perRow"`
	PerRelation          map[string]string            `json:"perRelation"`
	PerRelationUndecided map[string]string            `json:"perRelationUndecided"`
}

type rowScopedProductFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func rowScopedProductFixture(t *testing.T) (binary, dsn string) {
	t.Helper()
	dsn = os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL destination not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != rowScopedProductDatabase {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actualName string
	if err := db.QueryRow("select current_database()").Scan(&actualName); err != nil || actualName != rowScopedProductDatabase {
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
	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("cannot seed order lines: %v", err)
	}
	return buildCLIForLifecycle(t), dsn
}

// rowScopedProductWorkspace copies the reference Workspace and adds the
// operator declaration, the rule that uses it, and the implementation the
// composition selects.
func rowScopedProductWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)

	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatalf("the reference Workspace already selects a language")
	}
	additions := "languages:\n  - tei.authoring/v1\n" +
		"implementations:\n  - operator: sales.quantity_at_most\n    use: tei.number.every-at-most\n"
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-quantity-at-most.tei.md"),
		[]byte(rowScopedOperatorDocument), 0o644); err != nil {
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
	edited := strings.Replace(string(content), anchor,
		anchor+"    uses validation sales.order.lines.quantity_at_most_5\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runRowScopedProductView(t *testing.T, binary, root, orderID string) (rowScopedProductReport, cliProcessStreams) {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+orderID,
		"-format", "json",
	)
	var report rowScopedProductReport
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
	}
	return report, got
}

// plan/NOW.md road item 5's next product change: the shipped CLI must answer a
// row-scoped rule for rows that actually came from PostgreSQL, with the answer
// in perRelation and the business keys matching the seed.
func TestRowScopedOperatorRuleIsAnsweredPerRelationFromPostgres(t *testing.T) {
	binary, dsn := rowScopedProductFixture(t)
	root := rowScopedProductWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	for _, tc := range []struct {
		order string
		keys  []string
		want  string
	}{
		// 1001 holds 2, 0 and 5: every quantity is at most 5, so the rule
		// holds. The keys are the three lines of this order.
		{order: "1001", keys: []string{"1", "2", "3"}, want: "satisfied"},
		// 1002 holds 3 and 12: 12 is above the constant limit, and the
		// contract says a witnessed counterexample is enough.
		{order: "1002", keys: []string{"4", "5"}, want: "not-satisfied"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			report, got := runRowScopedProductView(t, binary, root, tc.order)
			if got.code != 0 || !report.OK || report.Error != nil {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			read := report.Result
			if read == nil || read.Coverage != "complete" {
				t.Fatalf("the read did not cover the relation: %+v", read)
			}
			if len(read.Rows) != len(tc.keys) {
				t.Fatalf("rows=%d, want %d: %+v", len(read.Rows), len(tc.keys), read.Rows)
			}
			for index, row := range read.Rows {
				if row.Key != tc.keys[index] {
					t.Fatalf("row order: got key %q at position %d, want %q", row.Key, index, tc.keys[index])
				}
			}
			answer, present := read.PerRelation["sales.order.lines.quantity_at_most_5"]
			if !present {
				t.Fatalf("the row-scoped rule is absent from perRelation: %+v", read.PerRelation)
			}
			if answer != tc.want {
				t.Fatalf("order=%s perRelation=%q, independently expected %q", tc.order, answer, tc.want)
			}
			if len(read.PerRelationUndecided) != 0 {
				t.Fatalf("undecided relation answers where the rows hold values: %v", read.PerRelationUndecided)
			}
			// The same rule must not also be answered row by row.
			for key, answers := range read.PerRow {
				if _, present := answers["sales.order.lines.quantity_at_most_5"]; present {
					t.Fatalf("the row-scoped rule was also answered for row %s", key)
				}
			}
		})
	}
}
