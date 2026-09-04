package postgres_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/environment"
)

type advancedIndexCatalog struct{ fakeCatalog }

func (advancedIndexCatalog) Indexes(context.Context) ([]postgres.IndexMeta, error) {
	return []postgres.IndexMeta{{
		Schema: "public",
		Table:  "orders",
		Name:   "orders_advanced_idx",
		Keys: []postgres.IndexKeyMeta{
			{Column: "customer_id", OperatorClass: "public.int8_ops", Order: "desc", NullsOrder: "first"},
			{Expression: "lower(customer_name)", Collation: "public.case_insensitive", Order: "asc", NullsOrder: "last"},
		},
		IncludedColumns:  []string{"order_id"},
		Predicate:        "customer_id > 0",
		Unique:           true,
		NullsNotDistinct: true,
		Method:           "btree",
		Valid:            false,
		Ready:            true,
		Live:             true,
	}}, nil
}

func TestPostgresProviderPreservesAdvancedIndexMetadata(t *testing.T) {
	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}
	p := postgres.Provider{Catalog: advancedIndexCatalog{}}
	snapshot, err := provider.Introspect(context.Background(), p, inst)
	if err != nil {
		t.Fatal(err)
	}
	table, ok := snapshot.Table("public.orders")
	if !ok || len(table.Indexes) != 1 {
		t.Fatalf("indexes=%#v", table.Indexes)
	}
	index := table.Indexes[0]
	if index.Name != "orders_advanced_idx" || !index.Unique || index.Method != "btree" || index.UniqueNullTreatment != physical.UniqueNullTreatmentNotDistinct {
		t.Fatalf("index identity=%#v", index)
	}
	if len(index.Keys) != 2 {
		t.Fatalf("keys=%#v", index.Keys)
	}
	if key := index.Keys[0]; key.Column != "customer_id" || key.Expression != nil || key.OperatorClass != "public.int8_ops" || key.Order != physical.IndexOrderDesc || key.NullsOrder != physical.IndexNullsOrderFirst {
		t.Fatalf("column key=%#v", key)
	}
	if key := index.Keys[1]; key.Column != "" || key.Expression == nil || key.Expression.Provider != string(environment.ProviderPostgres) || key.Expression.Source != "lower(customer_name)" || key.Collation != "public.case_insensitive" || key.Order != physical.IndexOrderAsc || key.NullsOrder != physical.IndexNullsOrderLast {
		t.Fatalf("expression key=%#v", key)
	}
	if len(index.IncludedColumns) != 1 || index.IncludedColumns[0] != "order_id" {
		t.Fatalf("included columns=%#v", index.IncludedColumns)
	}
	if index.Predicate == nil || index.Predicate.Provider != string(environment.ProviderPostgres) || index.Predicate.Source != "customer_id > 0" {
		t.Fatalf("predicate=%#v", index.Predicate)
	}
	if index.State == nil || index.State.Valid || !index.State.Ready || !index.State.Live || index.State.Usable() {
		t.Fatalf("state=%#v", index.State)
	}
}
