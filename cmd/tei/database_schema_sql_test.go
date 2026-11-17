package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestDatabaseSchemaSQLEmitsCreateStatementsFromDefinitions pins the DDL path
// NOW.md item 4 asks for: `tei` itself prints the SQL that builds the declared
// tables, without contacting a database. The expectations below are written
// out from the reference Workspace's declarations, not copied from the
// renderer's output.
func TestDatabaseSchemaSQLEmitsCreateStatementsFromDefinitions(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	workspace, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	got := runCLIProcessWithStreams(t, binary,
		"database-schema-sql", "-workspace", workspace,
		"-database", "sales_core", "-provider", "postgres",
	)
	if got.code != 0 {
		t.Fatalf("database-schema-sql failed: code=%d stderr=%s", got.code, got.stderr)
	}
	if strings.Contains(got.stdout, "TEI_VIEW_PG_DSN") {
		t.Fatal("the command asked for a database connection string; DDL must come from definitions")
	}

	for _, want := range []string{
		`CREATE TABLE "public"."order_lines"`,
		`CREATE TABLE "public"."orders"`,
		`"order_line_id" pg_catalog.int8 NOT NULL`,
		`"total_amount" pg_catalog.numeric NOT NULL`,
		`ADD PRIMARY KEY ("order_line_id")`,
	} {
		if !strings.Contains(got.stdout, want) {
			t.Fatalf("DDL is missing %q; got:\n%s", want, got.stdout)
		}
	}

	// An unknown database must be refused, not silently rendered as empty.
	gotUnknown := runCLIProcessWithStreams(t, binary,
		"database-schema-sql", "-workspace", workspace,
		"-database", "no_such_database", "-provider", "postgres",
	)
	if gotUnknown.code == 0 {
		t.Fatalf("an unknown database produced DDL instead of failing: %s", gotUnknown.stdout)
	}
	if !strings.Contains(gotUnknown.stderr, "no_such_database") {
		t.Fatalf("refusal does not name the unknown database: %s", gotUnknown.stderr)
	}
}
