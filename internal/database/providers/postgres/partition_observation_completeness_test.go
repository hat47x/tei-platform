package postgres_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/database/providers/postgres"
	"tei/internal/environment"
)

type partitionAwareCatalog struct{ fakeCatalog }

func (partitionAwareCatalog) Partitions(context.Context) ([]postgres.PartitionRelation, error) {
	return []postgres.PartitionRelation{{
		Child:  postgres.RelationIdentity{Schema: "public", Name: "orders_2026"},
		Parent: postgres.RelationIdentity{Schema: "public", Name: "orders"},
	}}, nil
}

func TestObserveSchemaDistinguishesCompletePartitionObservation(t *testing.T) {
	instance := environment.DatabaseInstance{Definition: core.MustCanonicalID("sales_core")}

	withoutReader, err := (postgres.Provider{Catalog: fakeCatalog{}}).ObserveSchema(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}
	if withoutReader.PartitionObservationComplete {
		t.Fatal("CatalogReader without partition capability must not claim complete partition observation")
	}

	withReader, err := (postgres.Provider{Catalog: partitionAwareCatalog{}}).ObserveSchema(context.Background(), instance)
	if err != nil {
		t.Fatal(err)
	}
	if !withReader.PartitionObservationComplete {
		t.Fatal("successful PartitionCatalogReader must mark the observation complete")
	}
	if len(withReader.Partitions) != 1 || withReader.Partitions[0].Child.Name != "orders_2026" {
		t.Fatalf("partitions=%#v", withReader.Partitions)
	}
}
