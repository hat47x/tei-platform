package main

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/realization"
	"tei/internal/environment"
	"tei/internal/workspace"
)

// `tei desired-schema -format sql` answers the question docs/first-change.md
// used to tell the reader to answer by hand: what SQL creates the declared
// tables. The expectation below is written from
// reference/workspace/databases/sales-core/persistence/order-line.tei and the
// relation-storage beside it, and from the columns docs/first-change.md listed
// for public.order_lines, not from the renderer (#2586).
func TestDesiredSchemaSQLMatchesTheDeclaredOrderLinesTable(t *testing.T) {
	statements := desiredSchemaSQLForReference(t, "sales_core")
	sql := strings.Join(statements, "\n")

	// Five columns come from the persistence mapping, order_id from
	// relation-storage. The storage type each one states decides the SQL type.
	for _, want := range []struct {
		column string
		pgType string
	}{
		{"order_line_id", "pg_catalog.int8"},
		{"order_id", "pg_catalog.int8"},
		{"line_number", "pg_catalog.int8"},
		{"item_code", "pg_catalog.varchar"},
		{"quantity", "pg_catalog.numeric"},
		{"unit_price", "pg_catalog.numeric"},
	} {
		line := `ALTER TABLE "public"."order_lines" ADD COLUMN "` + want.column + `" ` + want.pgType
		if !strings.Contains(sql, line) {
			t.Errorf("missing column declaration %q in:\n%s", line, sql)
		}
	}

	if !strings.Contains(sql, `ADD PRIMARY KEY ("order_line_id")`) {
		t.Errorf("the declared primary key is missing:\n%s", sql)
	}
	if !strings.Contains(sql, `CREATE TABLE "public"."order_lines"`) {
		t.Errorf("the table is not created:\n%s", sql)
	}
	// Each statement must end with exactly one semicolon: the renderer already
	// terminates them, and a second would be a syntax error when piped to psql.
	for _, statement := range statements {
		if strings.HasSuffix(statement, ";;") {
			t.Errorf("statement ends with a doubled semicolon: %q", statement)
		}
		if !strings.HasSuffix(statement, ";") {
			t.Errorf("statement is not terminated: %q", statement)
		}
	}
}

// The JSON snapshot and the SQL are two renderings of one definition, so a
// column the snapshot lists must appear in the statements.
func TestDesiredSchemaSQLCoversEverySnapshotColumn(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := realization.RealizeDesired(loaded.World.Persistence, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	statements, err := desiredSchemaSQL(snap, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.Join(statements, "\n")
	for _, table := range snap.Tables {
		for _, column := range table.Columns {
			if !strings.Contains(sql, `"`+column.Name+`"`) {
				t.Errorf("column %s.%s is in the snapshot but not in the SQL", table.Name, column.Name)
			}
		}
	}
}

// A provider that cannot render SQL must fail rather than print nothing.
func TestDesiredSchemaSQLRefusesAnUnsupportedProvider(t *testing.T) {
	snap := physical.SchemaSnapshot{Database: "sales_core"}
	if _, err := desiredSchemaSQL(snap, environment.ProviderOracle); err == nil {
		t.Fatal("an unsupported provider was accepted")
	}
}

func desiredSchemaSQLForReference(t *testing.T, database string) []string {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := realization.RealizeDesired(loaded.World.Persistence, database, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	statements, err := desiredSchemaSQL(snap, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	return statements
}
