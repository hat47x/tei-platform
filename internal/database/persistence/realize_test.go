package persistence_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/environment"
)

func TestRealizeDesiredIsProviderSpecific(t *testing.T) {
	m := persistence.Mapping{
		ID: core.MustCanonicalID("sales_core.order"), Database: core.MustCanonicalID("sales_core"), SemanticData: core.MustCanonicalID("sales.order"), Table: "public.orders",
		Columns:    []persistence.ColumnMapping{{Attribute: core.MustCanonicalID("sales.order.order_id"), Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required}},
		PrimaryKey: []core.CanonicalID{core.MustCanonicalID("sales.order.order_id")},
	}
	pg, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if got := pg.Tables[0].Columns[0].PhysicalType; got != "pg_catalog.int8" {
		t.Fatalf("postgres type: %s", got)
	}
	ora, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", environment.ProviderOracle)
	if err != nil {
		t.Fatal(err)
	}
	if got := ora.Tables[0].Columns[0].PhysicalType; got != "NUMBER(19)" {
		t.Fatalf("oracle type: %s", got)
	}
}
