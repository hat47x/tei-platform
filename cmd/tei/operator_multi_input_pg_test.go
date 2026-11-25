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

// These tests take plan/NOW.md road item 5's next product change: a rule
// decided by the multi-input operator tei.number.within must be answered for
// rows that actually came from PostgreSQL, and the answers must match values
// decided independently from the operator's meaning and the seed data.
//
// The expected values below are written from the seed
// (testdata/destination-order-change/seed.sql) and from "low <= value <=
// high". They are not read back from the CLI's output.
//
// Order 1001 has three lines:
//
//	key 1: quantity 2,   unit_price 100
//	key 2: quantity 0,   unit_price 250
//	key 3: quantity 5,   unit_price 0
//
// Order 1002 has two lines:
//
//	key 4: quantity 3,   unit_price 100
//	key 5: quantity 12,  unit_price 250

const operatorMultiInputDatabase = "tei_destination_order_e2e"

// withinOperatorDocument declares the operator and two rules: one whose bounds
// are constants, and one whose high bound is read from the same row's
// unit_price. Both are ordinary `*.tei.md` rules; no new validation kind is
// introduced.
const withinOperatorDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.within_value:\n" +
	"  takes value: number\n" +
	"  takes low: number\n" +
	"  takes high: number\n" +
	"\n" +
	"rule sales.order_line.unit_price_in_range:\n" +
	"  sales.order_line.unit_price is sales.within_value low: 0 high: 200\n" +
	"\n" +
	"rule sales.order_line.quantity_under_unit_price:\n" +
	"  sales.order_line.quantity is sales.within_value low: 0 high: sales.order_line.unit_price\n" +
	"```\n"

// crossRowRuleDocument names the same operator but takes the high bound from
// the same row's quantity. On order 1001 key 2 the row holds unit_price 250 and
// quantity 0, so the range's lower bound (250) is above its upper bound (0):
// that is a wrong range specification, not an out-of-range value.
const crossRowRuleDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.within_value:\n" +
	"  takes value: number\n" +
	"  takes low: number\n" +
	"  takes high: number\n" +
	"\n" +
	"rule sales.order_line.quantity_in_price_bounds:\n" +
	"  sales.order_line.line_number is sales.within_value low: sales.order_line.unit_price high: sales.order_line.quantity\n" +
	"```\n"

// operatorMultiInputViewReport is the JSON envelope `tei view -format json`
// writes. Only the fields these tests judge are named.
type operatorMultiInputViewReport struct {
	OK     bool                           `json:"ok"`
	Result *operatorMultiInputViewResult  `json:"result"`
	Error  *operatorMultiInputViewFailure `json:"error"`
}

type operatorMultiInputViewResult struct {
	Coverage        string                       `json:"coverage"`
	KeyAttributes   []string                     `json:"keyAttributes"`
	Rows            []operatorMultiInputViewRow  `json:"rows"`
	PerRow          map[string]map[string]string `json:"perRow"`
	PerRowUndecided map[string]map[string]string `json:"perRowUndecided"`
	// PerRelation is the answer of a rule asked of the rows of the relation
	// rather than of one row, keyed by rule.
	PerRelation          map[string]string `json:"perRelation"`
	PerRelationUndecided map[string]string `json:"perRelationUndecided"`
}

type operatorMultiInputViewRow struct {
	Key string `json:"key"`
}

type operatorMultiInputViewFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// operatorMultiInputFixture prepares the disposable database and returns the
// built CLI. It refuses to touch a database it was not told to expect.
func operatorMultiInputFixture(t *testing.T) (binary, fixtures string) {
	t.Helper()
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL destination not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != operatorMultiInputDatabase {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actualName string
	if err := db.QueryRow("select current_database()").Scan(&actualName); err != nil || actualName != operatorMultiInputDatabase {
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
	fixtures = filepath.Join("..", "..", "testdata", "destination-order-change")
	seed, err := os.ReadFile(filepath.Join(fixtures, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("cannot seed order lines: %v", err)
	}
	return buildCLIForLifecycle(t), fixtures
}

// operatorMultiInputWorkspace copies the reference Workspace and adds the
// operator declaration, the rules that use it, and optionally the
// implementation selection.
func operatorMultiInputWorkspace(t *testing.T, document string, selectImplementation bool) string {
	addition := ""
	if selectImplementation {
		addition = "implementations:\n  - operator: sales.within_value\n    use: tei.number.within\n"
	}
	return operatorMultiInputWorkspaceWithManifest(t, document, addition)
}

// operatorMultiInputWorkspaceWithManifest is the same Workspace with the exact
// manifest addition the caller names, so a test can select another
// implementation without duplicating this helper.
func operatorMultiInputWorkspaceWithManifest(t *testing.T, document, addition string) string {
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
	additions := "languages:\n  - tei.authoring/v1\n" + addition
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-within.tei.md"), []byte(document), 0o644); err != nil {
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
	rules := []string{}
	for _, line := range strings.Split(document, "\n") {
		if strings.HasPrefix(line, "rule ") {
			rules = append(rules, strings.TrimSuffix(strings.TrimPrefix(line, "rule "), ":"))
		}
	}
	if len(rules) == 0 {
		t.Fatal("the document declares no rule")
	}
	registration := ""
	for _, rule := range rules {
		registration += "    uses validation " + rule + "\n"
	}
	edited := strings.Replace(string(content), anchor, anchor+registration, 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runOperatorMultiInputView(t *testing.T, binary, root, orderID string) (operatorMultiInputViewReport, cliProcessStreams) {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+orderID,
		"-format", "json",
	)
	var report operatorMultiInputViewReport
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
	}
	return report, got
}

// TestMultipleInputOperatorDecidesRealPostgresRows is plan/NOW.md road item 5's
// next product change: the answers for rows that came from PostgreSQL must
// match values decided independently, per business key.
func TestMultipleInputOperatorDecidesRealPostgresRows(t *testing.T) {
	binary, _ := operatorMultiInputFixture(t)
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	root := operatorMultiInputWorkspace(t, withinOperatorDocument, true)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	cases := []struct {
		order string
		// want is the answer of each rule for each business key.
		want map[string]map[string]string
	}{
		{
			order: "1001",
			want: map[string]map[string]string{
				"1": {
					// 100 is inside [0, 200]; 2 is inside [0, 100].
					"sales.order_line.unit_price_in_range":       "satisfied",
					"sales.order_line.quantity_under_unit_price": "satisfied",
				},
				"2": {
					// 250 is outside [0, 200]; 0 is inside [0, 250].
					"sales.order_line.unit_price_in_range":       "not-satisfied",
					"sales.order_line.quantity_under_unit_price": "satisfied",
				},
				"3": {
					// 0 is inside [0, 200]. 5 is outside [0, 0]: the high
					// bound is this same row's unit_price. A row that read the
					// bound from another row (100 or 250) would answer
					// satisfied here, so this answer separates a same-row read
					// from a cross-row one.
					"sales.order_line.unit_price_in_range":       "satisfied",
					"sales.order_line.quantity_under_unit_price": "not-satisfied",
				},
			},
		},
		{
			order: "1002",
			want: map[string]map[string]string{
				"4": {
					// 100 is inside [0, 200]; 3 is inside [0, 100].
					"sales.order_line.unit_price_in_range":       "satisfied",
					"sales.order_line.quantity_under_unit_price": "satisfied",
				},
				"5": {
					// 250 is outside [0, 200]; 12 is inside [0, 250].
					"sales.order_line.unit_price_in_range":       "not-satisfied",
					"sales.order_line.quantity_under_unit_price": "satisfied",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.order, func(t *testing.T) {
			report, got := runOperatorMultiInputView(t, binary, root, tc.order)
			if got.code != 0 || !report.OK || report.Error != nil {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			read := report.Result
			if read == nil {
				t.Fatal("the report carries no result")
			}
			if read.Coverage != "complete" {
				t.Fatalf("the read did not cover the relation: coverage=%q", read.Coverage)
			}
			if len(read.KeyAttributes) != 1 || read.KeyAttributes[0] != "sales.order_line.order_line_id" {
				t.Fatalf("key attributes=%v", read.KeyAttributes)
			}
			// The rows are named by their business key, not by their position.
			seen := map[string]bool{}
			for _, row := range read.Rows {
				if seen[row.Key] {
					t.Fatalf("business key %q appears twice", row.Key)
				}
				seen[row.Key] = true
			}
			if len(seen) != len(tc.want) {
				t.Fatalf("rows=%v, want business keys of %v", seen, tc.want)
			}
			for key, want := range tc.want {
				if !seen[key] {
					t.Fatalf("the read did not return business key %q", key)
				}
				for rule, answer := range want {
					got, present := read.PerRow[key][rule]
					if !present {
						t.Fatalf("order=%s key=%s has no answer for %s", tc.order, key, rule)
					}
					if got != answer {
						t.Fatalf("order=%s key=%s %s=%q, independently expected %q", tc.order, key, rule, got, answer)
					}
				}
			}
			// A wrong range is an error; an absent value is undecided. Neither
			// may be reported as a satisfied or not-satisfied answer here.
			if len(read.PerRowUndecided) != 0 {
				t.Fatalf("undecided answers where the data holds values: %v", read.PerRowUndecided)
			}
		})
	}
}

// TestMultipleInputOperatorDataRangeIsAnEvaluationError checks plan/NOW.md road
// item 5's other half: a range whose bounds come from data and end up
// inverted is refused as a wrong range specification. It is not answered as
// not-satisfied, and the bounds are not swapped to make it decidable.
func TestMultipleInputOperatorDataRangeIsAnEvaluationError(t *testing.T) {
	binary, _ := operatorMultiInputFixture(t)
	root := operatorMultiInputWorkspace(t, crossRowRuleDocument, true)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}
	report, got := runOperatorMultiInputView(t, binary, root, "1001")
	if got.code == 0 || report.OK || report.Error == nil {
		t.Fatalf("an inverted data range was not refused: code=%d stdout=%s", got.code, got.stdout)
	}
	if report.Error.Code != "VIEW_EXECUTION_FAILED" {
		t.Fatalf("error code=%q, want VIEW_EXECUTION_FAILED", report.Error.Code)
	}
	if !strings.Contains(report.Error.Message, "greater than high") {
		t.Fatalf("the refusal does not name the inverted range: %s", report.Error.Message)
	}
	// Key 2 is the row whose unit_price (250) exceeds its quantity (0). It
	// must not appear as a decided answer of this rule.
	if report.Result != nil {
		if answers, present := report.Result.PerRow["2"]["sales.order_line.quantity_in_price_bounds"]; present {
			t.Fatalf("an inverted range was answered %q for business key 2", answers)
		}
	}
}

// TestMultipleInputOperatorWithoutASelectionIsRefused checks that a Workspace
// which declares the operator but selects no implementation for it is refused:
// the read does not fall back to another implementation, and it does not answer
// the rule satisfied or not-satisfied.
func TestMultipleInputOperatorWithoutASelectionIsRefused(t *testing.T) {
	binary, _ := operatorMultiInputFixture(t)
	root := operatorMultiInputWorkspace(t, withinOperatorDocument, false)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}
	report, got := runOperatorMultiInputView(t, binary, root, "1001")
	if got.code == 0 || report.OK || report.Error == nil {
		t.Fatalf("a rule with no selected implementation was answered: code=%d stdout=%s", got.code, got.stdout)
	}
	if !strings.Contains(report.Error.Message, "no selected implementation") {
		t.Fatalf("the refusal does not name the missing selection: %s", report.Error.Message)
	}
	if report.Result != nil && len(report.Result.PerRow) != 0 {
		t.Fatalf("row answers were produced without a selected implementation: %v", report.Result.PerRow)
	}
}
