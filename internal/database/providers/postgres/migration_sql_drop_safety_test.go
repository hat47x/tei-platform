package postgres_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

func TestRenderMigrationSQLRejectsDropTableWithoutObservation(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropTable,
			Table: "public.audit_log",
		}},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "requires complete provider partition observation") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("blocked rendering returned statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsEntireMixedDiffBeforeUnobservedDropTable(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			{Kind: schemadiff.DropTable, Table: "public.audit_log"},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "requires complete provider partition observation") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("mixed diff must fail before returning partial SQL, got=%#v", statements)
	}
}

func TestRenderMigrationSQLWithObservationRejectsUnresolvedBeforeReadiness(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropTable,
			Table: "public.audit_log",
		}},
		Unresolved: []schemadiff.UnresolvedComparison{{
			Kind:   schemadiff.UnresolvedProviderIdentifier,
			Table:  "public.audit_log",
			Reason: schemadiff.UnresolvedNormalizationRequired,
		}},
	}

	statements, err := postgres.RenderMigrationSQLWithObservation(diff, postgres.SchemaObservation{})
	if err == nil || !strings.Contains(err.Error(), "fully resolved schema diff") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("unresolved rendering returned statements=%#v", statements)
	}
}

func TestRenderMigrationSQLWithObservationDoesNotRequirePartitionObservationWithoutDropTable(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.CreateTable,
			Table: "public.new_table",
		}},
	}

	statements, err := postgres.RenderMigrationSQLWithObservation(diff, postgres.SchemaObservation{})
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 || statements[0] != `CREATE TABLE "public"."new_table" ();` {
		t.Fatalf("statements=%#v", statements)
	}
}

func TestRenderMigrationSQLWithObservationRendersMixedDiffAfterDropTableReadiness(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			{Kind: schemadiff.DropTable, Table: "public.audit_log"},
		},
	}

	statements, err := postgres.RenderMigrationSQLWithObservation(diff, completeObservation())
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 2 || statements[0] != `CREATE TABLE "public"."new_table" ();` || statements[1] != `DROP TABLE "public"."audit_log";` {
		t.Fatalf("statements=%#v", statements)
	}
}

func TestRenderMigrationSQLWithObservationRejectsWholeDiffWhenAnyDropTableIsUnsafe(t *testing.T) {
	observation := postgres.SchemaObservation{
		Snapshot: physical.SchemaSnapshot{Database: "sales_core"},
		Partitions: []postgres.PartitionRelation{{
			Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
			Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
		}},
		PartitionObservationComplete: true,
	}
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.DropTable, Table: "public.audit_log"},
			{Kind: schemadiff.DropTable, Table: "public.orders_2026"},
		},
	}

	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err == nil {
		t.Fatal("expected unsafe partition child to block the whole destructive diff")
	}
	if statements != nil {
		t.Fatalf("blocked destructive diff returned partial statements=%#v", statements)
	}
	var blocked *postgres.DestructiveReadinessError
	if !errors.As(err, &blocked) {
		t.Fatalf("err type=%T err=%v", err, err)
	}
	if blocked.Readiness.Executable || len(blocked.Readiness.DropTables) != 2 {
		t.Fatalf("readiness=%#v", blocked.Readiness)
	}
	if !blocked.Readiness.DropTables[0].Executable {
		t.Fatalf("ordinary table should be individually executable: %#v", blocked.Readiness.DropTables[0])
	}
	if blocked.Readiness.DropTables[1].Executable || blocked.Readiness.DropTables[1].Reason != postgres.DropTableTargetIsPartitionChild {
		t.Fatalf("partition child readiness=%#v", blocked.Readiness.DropTables[1])
	}
}

func TestRenderMigrationSQLAddsPrimaryKeyToNewTableAfterItsColumns(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table", PrimaryKey: []string{"tenant", "id"}},
			{Kind: schemadiff.AddColumn, Table: "public.new_table", Column: &physical.Column{Name: "tenant", PhysicalType: "text"}},
			{Kind: schemadiff.AddColumn, Table: "public.new_table", Column: &physical.Column{Name: "id", PhysicalType: "bigint"}},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`CREATE TABLE "public"."new_table" ();`,
		`ALTER TABLE "public"."new_table" ADD COLUMN "tenant" text NOT NULL;`,
		`ALTER TABLE "public"."new_table" ADD COLUMN "id" bigint NOT NULL;`,
		`ALTER TABLE "public"."new_table" ADD PRIMARY KEY ("tenant", "id");`,
	}
	if !reflect.DeepEqual(statements, want) {
		t.Fatalf("statements=%#v want=%#v", statements, want)
	}
	for _, statement := range statements {
		if strings.Contains(statement, "CONSTRAINT") {
			t.Fatalf("primary key must stay unnamed so PostgreSQL chooses the name: %q", statement)
		}
	}
}

func TestRenderMigrationSQLRejectsNewTablePrimaryKeyColumnThatIsMissingOrNullable(t *testing.T) {
	missing := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.AddColumn, Table: "public.new_table", Column: &physical.Column{Name: "other", PhysicalType: "bigint"}},
		},
	}
	statements, err := postgres.RenderMigrationSQL(missing)
	if err == nil || !strings.Contains(err.Error(), "is not added by this diff") || statements != nil {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}

	nullable := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.AddColumn, Table: "public.new_table", Column: &physical.Column{Name: "id", PhysicalType: "bigint", Nullable: true}},
		},
	}
	statements, err = postgres.RenderMigrationSQL(nullable)
	if err == nil || !strings.Contains(err.Error(), "is nullable") || statements != nil {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
}

func TestRenderMigrationSQLKeepsEachNewTablePrimaryKeyWithItsOwnTable(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.a", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.CreateTable, Table: "public.b", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.AddColumn, Table: "public.a", Column: &physical.Column{Name: "id", PhysicalType: "bigint"}},
			{Kind: schemadiff.AddColumn, Table: "public.b", Column: &physical.Column{Name: "other", PhysicalType: "bigint"}},
		},
	}
	// Table b has no id column in this diff: a's column must not satisfy b's key.
	if _, err := postgres.RenderMigrationSQL(diff); err == nil || !strings.Contains(err.Error(), "public.b") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLRendersUniqueAndForeignKeyOfNewTablesAfterPrimaryKeys(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.child", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.AddColumn, Table: "public.child", Column: &physical.Column{Name: "id", PhysicalType: "bigint"}},
			{Kind: schemadiff.AddColumn, Table: "public.child", Column: &physical.Column{Name: "parent_id", PhysicalType: "bigint"}},
			{Kind: schemadiff.AddColumn, Table: "public.child", Column: &physical.Column{Name: "code", PhysicalType: "text"}},
			{Kind: schemadiff.AddForeignKey, Table: "public.child", ForeignKey: &physical.ForeignKey{
				Name: "child_parent_fk", Columns: []string{"parent_id"}, ReferencedSchema: "public", ReferencedTable: "parent", ReferencedColumns: []string{"id"}}},
			{Kind: schemadiff.AddUniqueConstraint, Table: "public.child", UniqueConstraint: &physical.UniqueConstraint{Name: "child_code_key", Columns: []string{"code"}}},
			{Kind: schemadiff.AddUniqueConstraint, Table: "public.child", UniqueConstraint: &physical.UniqueConstraint{Name: "child_parent_id_code_key", Columns: []string{"parent_id", "code"}}},
			{Kind: schemadiff.CreateTable, Table: "public.parent", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.AddColumn, Table: "public.parent", Column: &physical.Column{Name: "id", PhysicalType: "bigint"}},
		},
	}
	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatal(err)
	}
	// Order: tables and columns, then primary keys, then unique constraints, and
	// foreign keys last so that the referenced table exists.
	want := []string{
		`CREATE TABLE "public"."child" ();`,
		`ALTER TABLE "public"."child" ADD COLUMN "id" bigint NOT NULL;`,
		`ALTER TABLE "public"."child" ADD COLUMN "parent_id" bigint NOT NULL;`,
		`ALTER TABLE "public"."child" ADD COLUMN "code" text NOT NULL;`,
		`CREATE TABLE "public"."parent" ();`,
		`ALTER TABLE "public"."parent" ADD COLUMN "id" bigint NOT NULL;`,
		`ALTER TABLE "public"."child" ADD PRIMARY KEY ("id");`,
		`ALTER TABLE "public"."parent" ADD PRIMARY KEY ("id");`,
		`ALTER TABLE "public"."child" ADD CONSTRAINT "child_code_key" UNIQUE ("code");`,
		`ALTER TABLE "public"."child" ADD CONSTRAINT "child_parent_id_code_key" UNIQUE ("parent_id", "code");`,
		`ALTER TABLE "public"."child" ADD CONSTRAINT "child_parent_fk" FOREIGN KEY ("parent_id") REFERENCES "public"."parent" ("id");`,
	}
	if !reflect.DeepEqual(statements, want) {
		t.Fatalf("statements=%#v want=%#v", statements, want)
	}
}

func TestRenderMigrationSQLDoesNotAddUniqueOrForeignKeyToExistingTables(t *testing.T) {
	for _, change := range []schemadiff.Change{
		{Kind: schemadiff.AddUniqueConstraint, Table: "public.orders", UniqueConstraint: &physical.UniqueConstraint{Columns: []string{"code"}}},
		{Kind: schemadiff.AddForeignKey, Table: "public.orders", ForeignKey: &physical.ForeignKey{
			Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}}},
	} {
		diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			change,
		}}
		statements, err := postgres.RenderMigrationSQL(diff)
		if err == nil || !strings.Contains(err.Error(), string(change.Kind)) || !strings.Contains(err.Error(), "without an existing-data observation") || statements != nil {
			t.Fatalf("kind=%s statements=%#v err=%v", change.Kind, statements, err)
		}
	}
}

func TestRenderMigrationSQLRejectsIncompleteOrUnresolvedForeignKey(t *testing.T) {
	base := func(fk physical.ForeignKey) schemadiff.Diff {
		return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.child"},
			{Kind: schemadiff.AddForeignKey, Table: "public.child", ForeignKey: &fk},
		}}
	}
	noSchema := base(physical.ForeignKey{Columns: []string{"a"}, ReferencedTable: "parent", ReferencedColumns: []string{"id"}})
	if _, err := postgres.RenderMigrationSQL(noSchema); err == nil || !strings.Contains(err.Error(), "no referenced schema") {
		t.Fatalf("err=%v", err)
	}
	mismatch := base(physical.ForeignKey{Columns: []string{"a", "b"}, ReferencedSchema: "public", ReferencedTable: "parent", ReferencedColumns: []string{"id"}})
	if _, err := postgres.RenderMigrationSQL(mismatch); err == nil || !strings.Contains(err.Error(), "incomplete definition") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLRejectsPrimaryKeyRemovalWithoutCurrentKey(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			{Kind: schemadiff.ChangePrimaryKey, Table: "public.orders"},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "removes the primary key but names no current key") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("unrealized primary-key change returned partial statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsAlterColumnDefaultChange(t *testing.T) {
	before := physical.Column{Name: "status", PhysicalType: "text", Nullable: false, Default: "'pending'"}
	after := before
	after.Default = "'ready'"
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			{Kind: schemadiff.AlterColumn, Table: "public.orders", Column: &after, BeforeColumn: &before},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "does not support column default change") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("unsupported default change returned partial statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsAlterColumnWithoutBeforeColumn(t *testing.T) {
	target := physical.Column{Name: "status", PhysicalType: "text", Nullable: false}
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:   schemadiff.AlterColumn,
			Table:  "public.orders",
			Column: &target,
		}},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "has no before column") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("alter-column without before state returned statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsUnsupportedSchemaChangeKinds(t *testing.T) {
	// Additions to existing tables need an existing-data observation; see
	// TestRenderMigrationSQLDoesNotAddUniqueOrForeignKeyToExistingTables.
	kinds := []schemadiff.ChangeKind{
		schemadiff.DropUniqueConstraint,
		schemadiff.DropForeignKey,
	}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			diff := schemadiff.Diff{
				Database: "sales_core",
				Changes: []schemadiff.Change{
					{Kind: schemadiff.CreateTable, Table: "public.new_table"},
					{Kind: kind, Table: "public.orders"},
				},
			}

			statements, err := postgres.RenderMigrationSQL(diff)
			if err == nil || !strings.Contains(err.Error(), "does not support schema change kind") {
				t.Fatalf("kind=%q statements=%#v err=%v", kind, statements, err)
			}
			if statements != nil {
				t.Fatalf("unsupported kind %q returned partial statements=%#v", kind, statements)
			}
		})
	}
}

func TestRenderMigrationSQLRejectsUnknownSchemaChangeKind(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			{Kind: schemadiff.ChangeKind("future-change"), Table: "public.orders"},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "does not support schema change kind") || !strings.Contains(err.Error(), "future-change") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("unknown change kind returned partial statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsDropIndexWithoutSchema(t *testing.T) {
	index := physical.Index{Name: "customers_name_idx"}
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropIndex,
			Table: "customers",
			Index: &index,
		}},
	}

	_, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "not schema-qualified") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLRejectsUnnamedUniqueAndForeignKey(t *testing.T) {
	for _, change := range []schemadiff.Change{
		{Kind: schemadiff.AddUniqueConstraint, Table: "public.child", UniqueConstraint: &physical.UniqueConstraint{Columns: []string{"code"}}},
		{Kind: schemadiff.AddForeignKey, Table: "public.child", ForeignKey: &physical.ForeignKey{
			Columns: []string{"parent_id"}, ReferencedSchema: "public", ReferencedTable: "parent", ReferencedColumns: []string{"id"}}},
	} {
		diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.child"},
			change,
		}}
		statements, err := postgres.RenderMigrationSQL(diff)
		if err == nil || !strings.Contains(err.Error(), "no constraint name") || statements != nil {
			t.Fatalf("kind=%s statements=%#v err=%v", change.Kind, statements, err)
		}
	}
}
