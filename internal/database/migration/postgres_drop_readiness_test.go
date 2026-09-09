package migration_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func TestRenderPostgresDropTableRequiresProviderPartitionObservation(t *testing.T) {
	diff := postgresDropDiff("public.audit_log")

	statements, err := migration.Render(environment.ProviderPostgres, diff)
	if err == nil || !strings.Contains(err.Error(), "requires complete provider partition observation") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("blocked rendering returned statements=%#v", statements)
	}
}

func TestPlanPostgresWithObservationAllowsOrdinaryDropAfterReadinessPass(t *testing.T) {
	observation := postgres.SchemaObservation{
		Snapshot:                     physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")},
		PartitionObservationComplete: true,
	}

	plan, err := migration.PlanPostgresWithObservation(
		"20260909_001",
		core.MustCanonicalID("sales_core"),
		"fp-ordinary",
		postgresDropDiff("public.audit_log"),
		observation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Realizations) != 1 || len(plan.Realizations[0].Statements) != 1 {
		t.Fatalf("realizations=%#v", plan.Realizations)
	}
	if plan.Realizations[0].Statements[0] != `DROP TABLE "public"."audit_log";` {
		t.Fatalf("statements=%#v", plan.Realizations[0].Statements)
	}
	if strings.Contains(strings.ToUpper(plan.Realizations[0].Statements[0]), "CASCADE") {
		t.Fatalf("statement must not contain CASCADE: %s", plan.Realizations[0].Statements[0])
	}
}

func TestPlanPostgresWithObservationBlocksPartitionParent(t *testing.T) {
	observation := postgres.SchemaObservation{
		Snapshot:                     physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")},
		PartitionObservationComplete: true,
		Partitions: []postgres.PartitionRelation{{
			Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
			Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
		}},
	}

	plan, err := migration.PlanPostgresWithObservation(
		"20260909_002",
		core.MustCanonicalID("sales_core"),
		"fp-partitioned",
		postgresDropDiff("public.orders"),
		observation,
	)
	if err == nil {
		t.Fatalf("plan=%#v", plan)
	}
	if len(plan.Realizations) != 0 || len(plan.Steps) != 0 {
		t.Fatalf("blocked planning must not return a partial definition: %#v", plan)
	}
}

func postgresDropDiff(table string) schemadiff.Diff {
	return schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropTable,
			Table: table,
		}},
	}
}
