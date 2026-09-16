package postgres_test

import (
	"errors"
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

func TestRenderMigrationSQLRejectsCreateTablePrimaryKeyUntilConstraintIdentityIsRealizable(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table", PrimaryKey: []string{"id"}},
			{Kind: schemadiff.AddColumn, Table: "public.new_table", Column: &physical.Column{Name: "id", PhysicalType: "bigint"}},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "does not support primary key on newly created table") || !strings.Contains(err.Error(), "constraint identity") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("unrealized create-table primary key returned partial statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsPrimaryKeyChangeUntilConstraintIdentityIsRealizable(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{
			{Kind: schemadiff.CreateTable, Table: "public.new_table"},
			{Kind: schemadiff.ChangePrimaryKey, Table: "public.orders"},
		},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "does not support primary-key change") || !strings.Contains(err.Error(), "constraint identity") {
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
	kinds := []schemadiff.ChangeKind{
		schemadiff.AddUniqueConstraint,
		schemadiff.DropUniqueConstraint,
		schemadiff.AddForeignKey,
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
