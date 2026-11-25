package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// A row-scoped rule whose SECOND input reads an attribute of another relation's
// Data is refused by `tei validate` (SEMANTIC_VALIDATION_OPERATOR_INPUT_OUTSIDE_RELATION_TARGET).
// `tei view -environment` runs no validator, so the product read path must not
// answer such a rule from whatever the Query happened to return: a limit read
// from a different table's column would compare rows that do not belong
// together. This test asks the product path directly, with no validator in front
// of it, and states what it must do.
func TestARowScopedRuleReadingAnotherRelationsAttributeIsRefusedByTheRead(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL row-scoped read not run")
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

	binary := buildCLIForLifecycle(t)
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	installForeignInputRule(t, root)

	// `tei validate` refuses the rule, which is the static gate.
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code == 0 {
		t.Fatal("the static validator accepted a row-scoped input read from another relation")
	} else if !strings.Contains(validated.stdout+validated.stderr, "OUTSIDE_RELATION_TARGET") {
		t.Fatalf("the validator refused for a different reason: %s%s", validated.stdout, validated.stderr)
	} else {
		// A workspace that does not validate must not be read either: the
		// read path is not a second, looser gate.
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines",
			"-environment", "development", "-input", "order_id=1001",
			"-format", "json",
		)
		if got.code == 0 {
			t.Fatalf("the read answered a rule its validator refuses: %s", got.stdout)
		}
		// The refusal names the attribute and states that it is not an item of
		// the relation's rows, which is the fact the validator also reported.
		for _, want := range []string{
			"customer.customer.credit_limit",
			"is not an item of sales.order_line",
		} {
			if !strings.Contains(got.stdout+got.stderr, want) {
				t.Fatalf("the read refusal does not name %q: %s%s", want, got.stdout, got.stderr)
			}
		}
		return
	}
}

// installForeignInputRule adds an operator whose second row-scoped input reads
// an attribute that is not a column of the relation's rows, plus the rule that
// uses it, and declares the Application uses that rule.
func installForeignInputRule(t *testing.T, root string) {
	t.Helper()
	document := "# 顧客の与信で明細を絞る（意図的に誤り）\n\n" +
		"明細の数量を顧客の与信と比べる。与信は明細の行の列ではないため、この規則は拒否されるべきである。\n\n" +
		"```tei\n" +
		"namespace sales\n\n" +
		"operator sales.at_most_credit:\n" +
		"  takes value: number of each row\n" +
		"  takes limit: number of each row\n\n" +
		"rule sales.order_line.quantity_at_most_credit:\n" +
		"  sales.order_line.quantity is sales.at_most_credit limit: customer.customer.credit_limit per sales.order.lines\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "foreign-input.tei.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatal("the reference workspace already selects a language; this test would not describe the change it makes")
	}
	content = append(content, []byte(
		"languages:\n  - tei.authoring/v1\n"+
			"implementations:\n"+
			"  - operator: sales.at_most_credit\n    use: tei.number.every-at-most\n")...)
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
		anchor+"    uses validation sales.order_line.quantity_at_most_credit\n")
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}
