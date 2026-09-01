package persistence_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
)

func TestRealizeDesiredConvertsSimpleIndexToOrderedColumnKeys(t *testing.T) {
	m := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("sales.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("sales.customer.tenant_id"), Column: "tenant_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("sales.customer.email"), Column: "email", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
		Indexes: []persistence.IndexMapping{{
			Name:       "customers_identity_uidx",
			Attributes: []core.CanonicalID{core.MustCanonicalID("sales.customer.tenant_id"), core.MustCanonicalID("sales.customer.email")},
			Unique:     true,
		}},
	}

	snapshot, err := persistence.RealizeDesired([]persistence.Mapping{m}, "sales_core", postgres.RealizationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.customers")
	if !ok || len(table.Indexes) != 1 {
		t.Fatalf("table=%#v", table)
	}
	index := table.Indexes[0]
	if len(index.Keys) != 2 || index.Keys[0].Column != "tenant_id" || index.Keys[1].Column != "email" {
		t.Fatalf("keys=%#v", index.Keys)
	}
	for _, key := range index.Keys {
		if key.Expression != nil || key.Order != physical.IndexOrderDefault || key.NullsOrder != physical.IndexNullsOrderDefault {
			t.Fatalf("key=%#v", key)
		}
	}
	if !index.Unique || index.UniqueNullTreatment != physical.UniqueNullTreatmentDistinct {
		t.Fatalf("unique semantics=%#v", index)
	}
	if index.State != nil {
		t.Fatalf("desired index must not carry actual state: %#v", index.State)
	}
}
