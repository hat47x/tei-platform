package postgres_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

type partitionCatalog struct{ fakeCatalog }

func (partitionCatalog) Tables(context.Context) ([]postgres.TableMeta, error) {
	return []postgres.TableMeta{
		{Schema: "public", Name: "orders", Kind: "table"},
		{Schema: "public", Name: "orders_2026", Kind: "table"},
		{Schema: "public", Name: "audit_log", Kind: "table"},
	}, nil
}

func (partitionCatalog) Partitions(context.Context) ([]postgres.PartitionRelation, error) {
	return []postgres.PartitionRelation{
		{
			Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
			Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
		},
	}, nil
}

func TestObserveSchemaPreservesPartitionRelationOutsideGenericSnapshotContract(t *testing.T) {
	instance := environment.DatabaseInstance{
		ID:         core.MustCanonicalID("sales_core.dev"),
		Definition: core.MustCanonicalID("sales_core"),
		Provider:   environment.ProviderPostgres,
	}
	observation, err := (postgres.Provider{Catalog: partitionCatalog{}}).ObserveSchema(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.Partitions) != 1 {
		t.Fatalf("partitions=%#v", observation.Partitions)
	}
	edge := observation.Partitions[0]
	if edge.Child.Schema != "public" || edge.Child.Name != "orders_2026" || edge.Parent.Schema != "public" || edge.Parent.Name != "orders" {
		t.Fatalf("partition edge=%#v", edge)
	}
	if _, ok := observation.Snapshot.Table("public.orders_2026"); !ok {
		t.Fatal("provider observationからpartition child relation自体が失われました")
	}
}

func TestPartitionChildDoesNotBecomeGenericDropButOrdinaryUnexpectedTableStillDoes(t *testing.T) {
	databaseID := core.MustCanonicalID("sales_core")
	observation := postgres.SchemaObservation{
		Snapshot: physical.SchemaSnapshot{
			Database: databaseID,
			Tables: []physical.Table{
				{Schema: "public", Name: "orders", Kind: physical.TableKindTable},
				{Schema: "public", Name: "orders_2026", Kind: physical.TableKindTable},
				{Schema: "public", Name: "audit_log", Kind: physical.TableKindTable},
			},
		},
		Partitions: []postgres.PartitionRelation{
			{
				Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
				Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
			},
		},
	}
	desired := physical.SchemaSnapshot{
		Database: databaseID,
		Tables:   []physical.Table{{Schema: "public", Name: "orders", Kind: physical.TableKindTable}},
	}

	actualForConvergence, err := observation.ProjectForConvergence(desired)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := actualForConvergence.Table("public.orders_2026"); ok {
		t.Fatal("known partition childがgeneric convergence subjectに残りました")
	}
	if _, ok := observation.Snapshot.Table("public.orders_2026"); !ok {
		t.Fatal("projectionが元のprovider observationを破壊しました")
	}

	diff, err := schemadiff.Compare(desired.Normalize(), actualForConvergence)
	if err != nil {
		t.Fatal(err)
	}
	foundOrdinaryDrop := false
	for _, change := range diff.Changes {
		if change.Kind != schemadiff.DropTable {
			continue
		}
		switch change.Table {
		case "public.orders_2026":
			t.Fatal("partition childからgeneric DropTableを導出しました")
		case "public.audit_log":
			foundOrdinaryDrop = true
		}
	}
	if !foundOrdinaryDrop {
		t.Fatalf("ordinary actual-only tableのDropTableが失われました: %#v", diff.Changes)
	}
}

func TestDesiredStandaloneTableConflictingWithObservedPartitionChildFailsClosed(t *testing.T) {
	databaseID := core.MustCanonicalID("sales_core")
	observation := postgres.SchemaObservation{
		Snapshot: physical.SchemaSnapshot{
			Database: databaseID,
			Tables: []physical.Table{
				{Schema: "public", Name: "orders", Kind: physical.TableKindTable},
				{Schema: "public", Name: "orders_2026", Kind: physical.TableKindTable},
			},
		},
		Partitions: []postgres.PartitionRelation{
			{
				Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
				Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
			},
		},
	}
	desired := physical.SchemaSnapshot{
		Database: databaseID,
		Tables: []physical.Table{
			{Schema: "public", Name: "orders", Kind: physical.TableKindTable},
			{Schema: "public", Name: "orders_2026", Kind: physical.TableKindTable},
		},
	}

	_, err := observation.ProjectForConvergence(desired)
	if err == nil {
		t.Fatal("desired standalone tableとactual partition childを黙って同一視しました")
	}
	if !strings.Contains(err.Error(), "public.orders_2026") || !strings.Contains(err.Error(), "partition child") {
		t.Fatalf("unexpected conflict diagnostic: %v", err)
	}
}

func TestPartitionObservationKeepsSubpartitionEdgeChain(t *testing.T) {
	observation := postgres.SchemaObservation{
		Partitions: []postgres.PartitionRelation{
			{
				Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
				Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
			},
			{
				Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026_q1"},
				Parent: postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
			},
		},
	}
	if len(observation.Partitions) != 2 {
		t.Fatalf("partitions=%#v", observation.Partitions)
	}
	if observation.Partitions[0].Child != observation.Partitions[1].Parent {
		t.Fatalf("sub-partition relationをchild/parent edge chainとして保持できません: %#v", observation.Partitions)
	}
}
