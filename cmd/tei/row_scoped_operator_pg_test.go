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

// plan/NOW.md road item 5's next product change is to decide a rule whose
// operator reads every row of a relation, for rows that actually came from
// PostgreSQL, and to compare the answers with values decided independently.
// The reader-level test (#5912) does this through validationread.Detail; this
// test takes the product entry point: `tei view -environment` must report such
// a rule in perRelation, not in perRow, with the answers the seed rows give.
//
// The expected values below are written from
// testdata/destination-order-change/seed.sql and from "every row's value is
// below the limit", not read back from the CLI.
//
//	order 1001: quantity 2, 0, 5 -> row 2 holds zero, so the rule does not hold
//	order 1002: quantity 3, 12   -> no row holds zero, so the rule holds
//
// The operator is the shipped tei.number.rows.non-zero. A witnessed
// counterexample settles the rule only when the operator states `refutes on
// some rows`; a partly read relation with no counterexample stays undecided.

const rowScopedProductDatabase = "tei_destination_order_e2e"

// rowScopedOperatorDocument declares the operator and the rule. The rule names
// its relation with `per sales.order.lines`, which is what decision section
// 15.4 requires of a row-scoped operator.
const rowScopedOperatorDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.quantity_non_zero:\n" +
	"  takes value: number of each row\n" +
	"  refutes on some rows\n" +
	"\n" +
	"rule sales.order.lines.quantity_non_zero:\n" +
	"  sales.order_line.quantity is sales.quantity_non_zero per sales.order.lines\n" +
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

func rowScopedProductWorkspace(t *testing.T, document string, selectImplementation bool) string {
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
	additions := "languages:\n  - tei.authoring/v1\n"
	if selectImplementation {
		additions += "implementations:\n  - operator: sales.quantity_non_zero\n    use: tei.number.rows.non-zero\n"
	}
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-quantity-below.tei.md"), []byte(document), 0o644); err != nil {
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
		anchor+"    uses validation sales.order.lines.quantity_non_zero\n", 1)
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

// The product entry point must answer a row-scoped rule in perRelation. Before
// this it could not: the rule was refused on both sides of the surface, and the
// environment read answered every rule row by row.
func TestRowScopedOperatorRuleIsAnsweredPerRelationFromPostgres(t *testing.T) {
	binary, dsn := rowScopedProductFixture(t)
	root := rowScopedProductWorkspace(t, rowScopedOperatorDocument, true)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	// The order matters: 1001 has quantity 2, 0, 5 against a limit of 3, so a
	// counterexample was read and the statement makes it decisive.
	for _, tc := range []struct {
		order string
		want  string
	}{
		{"1001", "not-satisfied"},
		{"1002", "satisfied"},
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
			answer, present := read.PerRelation["sales.order.lines.quantity_non_zero"]
			if !present {
				t.Fatalf("the row-scoped rule is absent from perRelation: %+v", read.PerRelation)
			}
			if answer != tc.want {
				t.Fatalf("order=%s perRelation=%q, independently expected %q", tc.order, answer, tc.want)
			}
			if len(read.PerRelationUndecided) != 0 {
				t.Fatalf("undecided relation answers where the rows hold values: %v", read.PerRelationUndecided)
			}
			// The same rule must not also appear row by row.
			for key, answers := range read.PerRow {
				if _, present := answers["sales.order.lines.quantity_non_zero"]; present {
					t.Fatalf("the row-scoped rule was also answered for row %s", key)
				}
			}
		})
	}
}
