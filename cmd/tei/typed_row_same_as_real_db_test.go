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

// `plan/NOW.md` road item 5's next named change asks for the shipped
// implementations of the row-set values of the types other than a calendar day
// to be carried through to real PostgreSQL rows. Three of them exist
// (`tei.text.every-same-as`, `tei.truth.every-is-true`, `tei.any.every-same-as`)
// and #6073 drove a calendar-day row rule through the product path, but the text
// and `any` row forms are named by no test under cmd/tei at all: measured on
// main, every row-scoped real-PostgreSQL check uses a number or a day rule.
//
// What this test fixes, on rows that come from a real varchar column:
//
//   - every row's text is compared with one constant by `tei.text.every-same-as`,
//     and an `any` input keeps the type the column arrived as so the same
//     comparison is decided by `tei.any.every-same-as`;
//   - an order whose rows all hold A-100 is satisfied, an order holding a
//     different code is not-satisfied because the counterexample was read, and an
//     order with no rows is satisfied because there is no row to refute it.
//
// The expected answers are written from the seed's item codes and from the
// decision table, not read back from the CLI.
func TestARowScopedTypedTextRuleDecidesRealPostgresRows(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL typed row read not run")
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
		t.Fatalf("cannot seed order lines: %v", err)
	}
	// Order 1004 holds two lines that both carry A-100, so the same rule has an
	// order whose every row agrees with the constant.
	if _, err := db.Exec(`
		insert into public.order_lines
			(order_line_id, order_id, line_number, item_code, quantity, unit_price)
		values
			(10, 1004, 1, 'A-100', 1, 100),
			(11, 1004, 2, 'A-100', 2, 100)
	`); err != nil {
		t.Fatalf("cannot seed the agreeing order: %v", err)
	}
	// The expected answers below are read off the codes this seed holds.
	var codes []string
	rows, err := db.Query("select item_code from public.order_lines where order_id = 1004 order by order_line_id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatal(err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	if len(codes) != 2 || codes[0] != "A-100" || codes[1] != "A-100" {
		t.Fatalf("order 1004 reads %v, not the two A-100 codes this test expects", codes)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installRowTypedTextRules(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the row text rules did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const (
		textRule = "sales.order.lines.every_code_is_a100"
		anyRule  = "sales.order.lines.every_code_is_a100_as_any"
	)
	for _, tc := range []struct {
		order           string
		textAns, anyAns string
		explanation     string
	}{
		// Both lines carry A-100, so no row refutes the statement.
		{order: "1004", textAns: "satisfied", anyAns: "satisfied",
			explanation: "every row holds A-100"},
		// The seed holds A-100, B-200 and C-300: B-200 and C-300 were read and
		// differ from the constant, so the counterexample settles the answer.
		{order: "1001", textAns: "not-satisfied", anyAns: "not-satisfied",
			explanation: "B-200 and C-300 were read and differ"},
		// The seed states that order 1003 holds no lines, so there is no row a
		// counterexample could be in.
		{order: "1003", textAns: "satisfied", anyAns: "satisfied",
			explanation: "an empty relation has no row to refute it"},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			read := readTypedRowRelation(t, binary, root, tc.order)
			for _, rule := range []struct{ id, want string }{
				{id: textRule, want: tc.textAns},
				{id: anyRule, want: tc.anyAns},
			} {
				answer := read.PerRelation[rule.id]
				if answer != rule.want {
					t.Fatalf("order=%s %s=%q, want %q: %s", tc.order, rule.id, answer, rule.want, tc.explanation)
				}
				if reason := read.PerRelationUndecided[rule.id]; reason != "" {
					t.Fatalf("order=%s %s is undecided (%q) although the rows were read", tc.order, rule.id, reason)
				}
			}
			if read.Coverage != "complete" {
				t.Fatalf("order=%s coverage=%q, want complete", tc.order, read.Coverage)
			}
		})
	}
}

type typedRowRelationRead struct {
	Coverage             string            `json:"coverage"`
	PerRelation          map[string]string `json:"perRelation"`
	PerRelationUndecided map[string]string `json:"perRelationUndecided"`
}

func readTypedRowRelation(t *testing.T, binary, root, order string) typedRowRelationRead {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+order,
		"-format", "json",
	)
	if got.code != 0 {
		t.Fatalf("order=%s: CLI failed: code=%d stdout=%s stderr=%s", order, got.code, got.stdout, got.stderr)
	}
	var report struct {
		OK     bool                  `json:"ok"`
		Result *typedRowRelationRead `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK || report.Result == nil {
		t.Fatalf("order=%s: cannot read the report: %v %s", order, err, got.stdout)
	}
	return *report.Result
}

// installRowTypedTextRules adds two row-scoped operators over the item code, one
// typed `text` and one typed `any`, each comparing every row with the constant
// A-100, and selects the shipped implementations for them.
func installRowTypedTextRules(t *testing.T, root string) {
	t.Helper()
	document := "# 明細すべての項目コード\n\n" +
		"明細のどの行も、項目コードが A-100 と同じであることを述べる。行が無ければ反例も無い。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.every_code_same:\n" +
		"  takes value: text of each row\n" +
		"  takes other: text of each row\n" +
		"  refutes on some rows\n\n" +
		"operator sales.every_code_same_as_any:\n" +
		"  takes value: any of each row\n" +
		"  takes other: any of each row\n" +
		"  refutes on some rows\n\n" +
		"rule sales.order.lines.every_code_is_a100:\n" +
		"  sales.order_line.item_code is sales.every_code_same other: \"A-100\" per sales.order.lines\n\n" +
		"rule sales.order.lines.every_code_is_a100_as_any:\n" +
		"  sales.order_line.item_code is sales.every_code_same_as_any other: \"A-100\" per sales.order.lines\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-every-code.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatal("the reference Workspace already selects a language; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n"+
			"  - operator: sales.every_code_same\n    use: tei.text.every-same-as\n"+
			"  - operator: sales.every_code_same_as_any\n    use: tei.any.every-same-as\n")...)
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
			"    uses validation sales.order.lines.every_code_is_a100\n"+
			"    uses validation sales.order.lines.every_code_is_a100_as_any\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
