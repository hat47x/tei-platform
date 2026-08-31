package persistence_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/environment"
)

func TestRealizeDesiredForeignKeyByMappingAndAttributeIdentity(t *testing.T) {
	mappings := foreignKeyMappings()
	snapshot, err := persistence.RealizeDesired(mappings, "sales_core", environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	orders, ok := snapshot.Table("public.orders")
	if !ok {
		t.Fatal("orders table not found")
	}
	if len(orders.ForeignKeys) != 1 {
		t.Fatalf("foreign keys=%#v", orders.ForeignKeys)
	}
	fk := orders.ForeignKeys[0]
	if fk.Name != "orders_customer_fk" || len(fk.Columns) != 1 || fk.Columns[0] != "customer_id" || fk.ReferencedSchema != "public" || fk.ReferencedTable != "customers" || len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "id" {
		t.Fatalf("foreign key=%#v", fk)
	}
}

func TestRealizeDesiredForeignKeyRejectsMismatchedAttributeCounts(t *testing.T) {
	mappings := foreignKeyMappings()
	mappings[1].ForeignKeys[0].ReferencedAttributes = nil
	_, err := persistence.RealizeDesired(mappings, "sales_core", environment.ProviderPostgres)
	if err == nil || !strings.Contains(err.Error(), "source/target attribute count mismatch") {
		t.Fatalf("error=%v", err)
	}
}

func TestRealizeDesiredForeignKeyRejectsCrossDatabaseReference(t *testing.T) {
	mappings := foreignKeyMappings()
	mappings[0].Database = core.MustCanonicalID("master_data")
	_, err := persistence.RealizeDesired(mappings, "sales_core", environment.ProviderPostgres)
	if err == nil || !strings.Contains(err.Error(), "crosses database boundary") {
		t.Fatalf("error=%v", err)
	}
}

func TestRealizeDesiredForeignKeyRejectsUnknownTargetAttribute(t *testing.T) {
	mappings := foreignKeyMappings()
	mappings[1].ForeignKeys[0].ReferencedAttributes[0] = core.MustCanonicalID("sales.customer.missing")
	_, err := persistence.RealizeDesired(mappings, "sales_core", environment.ProviderPostgres)
	if err == nil || !strings.Contains(err.Error(), "has no column mapping") {
		t.Fatalf("error=%v", err)
	}
}

func foreignKeyMappings() []persistence.Mapping {
	customers := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.customer"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("sales.customer"),
		Table:        "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("sales.customer.customer_id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
		},
		PrimaryKey: []core.CanonicalID{core.MustCanonicalID("sales.customer.customer_id")},
	}
	orders := persistence.Mapping{
		ID:           core.MustCanonicalID("sales_core.order"),
		Database:     core.MustCanonicalID("sales_core"),
		SemanticData: core.MustCanonicalID("sales.order"),
		Table:        "public.orders",
		Columns: []persistence.ColumnMapping{
			{Attribute: core.MustCanonicalID("sales.order.order_id"), Column: "id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: core.MustCanonicalID("sales.order.customer_id"), Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
		},
		PrimaryKey: []core.CanonicalID{core.MustCanonicalID("sales.order.order_id")},
		ForeignKeys: []persistence.ForeignKeyMapping{{
			Name:                 "orders_customer_fk",
			Attributes:           []core.CanonicalID{core.MustCanonicalID("sales.order.customer_id")},
			ReferencedMapping:    core.MustCanonicalID("sales_core.customer"),
			ReferencedAttributes: []core.CanonicalID{core.MustCanonicalID("sales.customer.customer_id")},
		}},
	}
	return []persistence.Mapping{customers, orders}
}
