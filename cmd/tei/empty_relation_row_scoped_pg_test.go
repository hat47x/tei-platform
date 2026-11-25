package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// `plan/NOW.md` road item 5's next named change is to let a row-set example state
// an EMPTY relation and to fix, on real PostgreSQL, that `refutes on some rows`
// does not change the answer for an empty relation. The notation is not designed
// yet, so that part is not attempted here. What this test does is state the
// current behaviour for an empty relation at the product path, so a later
// notation change has a fact to preserve or to change deliberately.
//
// The rows come from the reference seed, which deliberately leaves order 1003
// without lines. The expected answers are taken from the decision table, not from
// the run:
//
//	every row of the relation is within its limit
//	→ no row exists, so nothing refutes it
//	→ complete coverage: satisfied
//
// and the refutation statement is irrelevant here, because there is no row it
// could apply to. A partly read relation is a different case and stays undecided.
func TestARowScopedRuleOverAnEmptyRelationIsSatisfied(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL empty-relation read not run")
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
		t.Fatalf("cannot seed order lines: %v", err)
	}
	// The seed states that order 1003 holds no lines; this test depends on
	// that fact rather than inserting one.
	var lines int
	if err := db.QueryRow("select count(*) from public.order_lines where order_id = 1003").Scan(&lines); err != nil {
		t.Fatal(err)
	}
	if lines != 0 {
		t.Fatalf("order 1003 holds %d lines; the seed no longer states an empty relation", lines)
	}

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installEmptyRelationRules(t, root)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace carrying the row-scoped rules did not validate: %s%s",
			validated.stdout, validated.stderr)
	}

	const (
		refutingRule = "sales.order_line.every_quantity_at_most_price"
		plainRule    = "sales.order_line.every_quantity_at_most_price_without_statement"
	)
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id=1003",
		"-format", "json",
	)
	if got.code != 0 {
		t.Fatalf("CLI failed on an empty relation: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
	}
	var report struct {
		OK     bool `json:"ok"`
		Result *struct {
			Coverage             string                       `json:"coverage"`
			Rows                 []json.RawMessage            `json:"rows"`
			PerRelation          map[string]string            `json:"perRelation"`
			PerRelationUndecided map[string]string            `json:"perRelationUndecided"`
			PerRow               map[string]map[string]string `json:"perRow"`
		} `json:"result"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil || !report.OK || report.Error != nil || report.Result == nil {
		t.Fatalf("cannot read the report: %v %s", err, got.stdout)
	}
	read := report.Result
	// An empty relation read completely states that there are no rows; the
	// coverage declaration and the Query agree, so this is complete.
	if read.Coverage != "complete" {
		t.Fatalf("coverage=%q, want complete: an empty relation read completely is still complete", read.Coverage)
	}
	if len(read.Rows) != 0 {
		t.Fatalf("rows=%d, want none for order 1003", len(read.Rows))
	}
	// No row exists, so nothing refutes the rule and the statement about a
	// counterexample among the rows has nothing to apply to.
	for _, ruleID := range []string{refutingRule, plainRule} {
		answer := read.PerRelation[ruleID]
		if answer != "satisfied" {
			t.Fatalf("%s=%q, want satisfied: an empty relation has no row that could break the rule",
				ruleID, answer)
		}
		if reason := read.PerRelationUndecided[ruleID]; reason != "" {
			t.Fatalf("%s is undecided (%q) although the relation was read completely", ruleID, reason)
		}
		for key, answers := range read.PerRow {
			if _, present := answers[ruleID]; present {
				t.Fatalf("%s was answered for row %s of an empty relation", ruleID, key)
			}
		}
	}
}

// installEmptyRelationRules adds two row-scoped operators over the rows of
// sales.order.lines, one with `refutes on some rows` and one without. The rules
// ask that every quantity is at most its own unit price.
func installEmptyRelationRules(t *testing.T, root string) {
	t.Helper()
	document := "# 明細すべての数量\n\n" +
		"明細のどの行も、数量がその行の単価以下であることを述べる。行が無い場合は反例も無い。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.every_at_most_price:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row\n" +
		"  refutes on some rows\n\n" +
		"operator sales.every_at_most_price_plain:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row\n\n" +
		"rule sales.order_line.every_quantity_at_most_price:\n" +
		"  sales.order_line.quantity is sales.every_at_most_price limit: sales.order_line.unit_price per sales.order.lines\n\n" +
		"rule sales.order_line.every_quantity_at_most_price_without_statement:\n" +
		"  sales.order_line.quantity is sales.every_at_most_price_plain limit: sales.order_line.unit_price per sales.order.lines\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "empty-relation.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if containsBytes(content, "languages:") {
		t.Fatal("the reference workspace already selects a language; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n"+
			"  - operator: sales.every_at_most_price\n    use: tei.number.every-at-most\n"+
			"  - operator: sales.every_at_most_price_plain\n    use: tei.number.every-at-most\n")...)
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
			"    uses validation sales.order_line.every_quantity_at_most_price\n"+
			"    uses validation sales.order_line.every_quantity_at_most_price_without_statement\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsBytes(haystack []byte, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == needle {
			return true
		}
	}
	return false
}
