package postgres_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

func TestDestructiveReadinessBlocksParentWithPartitionDescendants(t *testing.T) {
	observation := postgres.SchemaObservation{
		PartitionObservationComplete: true,
		Partitions: []postgres.PartitionRelation{
			{Child: postgres.RelationIdentity{Schema: "public", Name: "orders_2026"}, Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"}},
			{Child: postgres.RelationIdentity{Schema: "public", Name: "orders_2026_q1"}, Parent: postgres.RelationIdentity{Schema: "public", Name: "orders_2026"}},
		},
	}
	diff := dropTableDiff("public.orders")

	readiness, err := observation.AssessDestructiveReadiness(diff)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Executable || len(readiness.DropTables) != 1 {
		t.Fatalf("readiness=%#v", readiness)
	}
	drop := readiness.DropTables[0]
	if drop.Reason != postgres.DropTablePartitionTreePresent {
		t.Fatalf("reason=%q", drop.Reason)
	}
	if len(drop.RelatedRelations) != 2 || drop.RelatedRelations[0].Name != "orders_2026" || drop.RelatedRelations[1].Name != "orders_2026_q1" {
		t.Fatalf("related=%#v", drop.RelatedRelations)
	}

	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err == nil {
		t.Fatal("expected partition tree to block rendering")
	}
	if statements != nil {
		t.Fatalf("blocked rendering returned statements=%#v", statements)
	}
	var blocked *postgres.DestructiveReadinessError
	if !errors.As(err, &blocked) {
		t.Fatalf("err type=%T err=%v", err, err)
	}
}

func TestDestructiveReadinessBlocksTargetThatIsPartitionChild(t *testing.T) {
	observation := postgres.SchemaObservation{
		PartitionObservationComplete: true,
		Partitions: []postgres.PartitionRelation{{
			Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
			Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
		}},
	}

	readiness, err := observation.AssessDestructiveReadiness(dropTableDiff("public.orders_2026"))
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Executable || readiness.DropTables[0].Reason != postgres.DropTableTargetIsPartitionChild {
		t.Fatalf("readiness=%#v", readiness)
	}
	if len(readiness.DropTables[0].RelatedRelations) != 1 || readiness.DropTables[0].RelatedRelations[0].Name != "orders" {
		t.Fatalf("related=%#v", readiness.DropTables[0].RelatedRelations)
	}
}

func TestDestructiveReadinessBlocksWhenPartitionObservationIsUnavailable(t *testing.T) {
	observation := postgres.SchemaObservation{}

	readiness, err := observation.AssessDestructiveReadiness(dropTableDiff("public.audit_log"))
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Executable || readiness.DropTables[0].Reason != postgres.DropTablePartitionObservationMissing {
		t.Fatalf("readiness=%#v", readiness)
	}
	if _, err := postgres.RenderMigrationSQLWithObservation(dropTableDiff("public.audit_log"), observation); err == nil {
		t.Fatal("rendering must fail without complete partition observation")
	}
}

func TestDestructiveReadinessAllowsOrdinaryTableOnlyWithCompleteObservation(t *testing.T) {
	observation := postgres.SchemaObservation{
		PartitionObservationComplete: true,
		Partitions: []postgres.PartitionRelation{{
			Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
			Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
		}},
	}
	diff := dropTableDiff("public.audit_log")

	readiness, err := observation.AssessDestructiveReadiness(diff)
	if err != nil {
		t.Fatal(err)
	}
	if !readiness.Executable || len(readiness.DropTables) != 1 || !readiness.DropTables[0].Executable {
		t.Fatalf("readiness=%#v", readiness)
	}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 || statements[0] != `DROP TABLE "public"."audit_log";` {
		t.Fatalf("statements=%#v", statements)
	}
	if strings.Contains(strings.ToUpper(statements[0]), "CASCADE") {
		t.Fatalf("destructive rendering must not add CASCADE: %s", statements[0])
	}
}

func TestDestructiveReadinessRejectsConflictingPartitionParents(t *testing.T) {
	observation := postgres.SchemaObservation{
		PartitionObservationComplete: true,
		Partitions: []postgres.PartitionRelation{
			{Child: postgres.RelationIdentity{Schema: "public", Name: "orders_2026"}, Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"}},
			{Child: postgres.RelationIdentity{Schema: "public", Name: "orders_2026"}, Parent: postgres.RelationIdentity{Schema: "archive", Name: "orders"}},
		},
	}

	_, err := observation.AssessDestructiveReadiness(dropTableDiff("public.orders"))
	if err == nil || !strings.Contains(err.Error(), "multiple observed parents") {
		t.Fatalf("err=%v", err)
	}
}

func dropTableDiff(table string) schemadiff.Diff {
	return schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropTable,
			Table: table,
		}},
	}
}
