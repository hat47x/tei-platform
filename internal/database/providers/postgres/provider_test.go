package postgres_test

import (
	"context"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/provider"
	"tei/internal/database/providers/postgres"
	"tei/internal/environment"
)

type fakeCatalog struct{}

func (fakeCatalog) Tables(context.Context) ([]postgres.TableMeta, error) {
	return []postgres.TableMeta{{Schema: "public", Name: "orders", Kind: "table"}, {Schema: "reporting", Name: "orders", Kind: "view"}}, nil
}
func (fakeCatalog) Columns(context.Context) ([]postgres.ColumnMeta, error) {
	return []postgres.ColumnMeta{
		{Schema: "public", Table: "orders", Name: "order_id", PhysicalType: "pg_catalog.int8", Nullable: false, Ordinal: 1},
		{Schema: "public", Table: "orders", Name: "customer_id", PhysicalType: "pg_catalog.int8", Nullable: false, Ordinal: 2},
		{Schema: "public", Table: "orders", Name: "note", PhysicalType: "pg_catalog.text", Nullable: true, Ordinal: 3},
		{Schema: "reporting", Table: "orders", Name: "order_id", PhysicalType: "pg_catalog.int8", Nullable: false, Ordinal: 1},
	}, nil
}
func (fakeCatalog) PrimaryKeys(context.Context) ([]postgres.PrimaryKeyMeta, error) {
	return []postgres.PrimaryKeyMeta{{Schema: "public", Table: "orders", Columns: []string{"order_id"}}}, nil
}
func (fakeCatalog) UniqueConstraints(context.Context) ([]postgres.UniqueConstraintMeta, error) {
	return []postgres.UniqueConstraintMeta{{Schema: "public", Table: "orders", Name: "orders_customer_key", Columns: []string{"customer_id"}}}, nil
}
func (fakeCatalog) ForeignKeys(context.Context) ([]postgres.ForeignKeyMeta, error) {
	return []postgres.ForeignKeyMeta{{Schema: "public", Table: "orders", Name: "orders_customer_fk", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"customer_id"}}}, nil
}
func (fakeCatalog) Indexes(context.Context) ([]postgres.IndexMeta, error) {
	return []postgres.IndexMeta{
		{
			Schema: "public", Table: "orders", Name: "orders_customer_idx", Method: "btree", Valid: true, Ready: true, Live: true,
			Keys: []postgres.IndexKeyMeta{{Column: "customer_id", Order: "default", NullsOrder: "default"}},
		},
		{
			Schema: "public", Table: "orders", Name: "orders_note_expr_idx", Method: "btree", Valid: false, Ready: true, Live: true,
			Keys:            []postgres.IndexKeyMeta{{Expression: "lower(note)", Order: "desc", NullsOrder: "default", Collation: "pg_catalog.C", OperatorClass: "public.custom_text_ops"}},
			IncludedColumns: []string{"order_id"},
			Predicate:       "note IS NOT NULL",
		},
	}, nil
}

func TestPostgresProviderIntrospectsPhysicalSchema(t *testing.T) {
	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}
	fixed := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	p := postgres.Provider{Catalog: fakeCatalog{}, Now: func() time.Time { return fixed }}
	snap, err := provider.Introspect(context.Background(), p, inst)
	if err != nil {
		t.Fatal(err)
	}
	if snap.CapturedAt != "2026-08-27T00:00:00Z" {
		t.Fatalf("capturedAt=%q", snap.CapturedAt)
	}
	if len(snap.Tables) != 2 {
		t.Fatalf("tables=%#v", snap.Tables)
	}
	publicOrders, ok := snap.Table("public.orders")
	if !ok {
		t.Fatal("public.orders not resolved")
	}
	if len(publicOrders.PrimaryKey) != 1 || publicOrders.PrimaryKey[0] != "order_id" {
		t.Fatalf("pk=%#v", publicOrders.PrimaryKey)
	}
	if len(publicOrders.UniqueConstraints) != 1 || publicOrders.UniqueConstraints[0].Name != "orders_customer_key" {
		t.Fatalf("unique constraints=%#v", publicOrders.UniqueConstraints)
	}
	if len(publicOrders.ForeignKeys) != 1 || len(publicOrders.Indexes) != 2 {
		t.Fatalf("physical metadata=%#v", publicOrders)
	}
	advanced := publicOrders.Indexes[1]
	if advanced.Name != "orders_note_expr_idx" || len(advanced.Keys) != 1 || advanced.Keys[0].Expression == nil {
		t.Fatalf("advanced index=%#v", advanced)
	}
	if advanced.Keys[0].Expression.Provider != string(environment.ProviderPostgres) || advanced.Keys[0].Expression.Dialect != "postgresql" {
		t.Fatalf("expression provenance=%#v", advanced.Keys[0].Expression)
	}
	if advanced.Predicate == nil || advanced.Predicate.Source != "note IS NOT NULL" || len(advanced.IncludedColumns) != 1 || advanced.IncludedColumns[0] != "order_id" {
		t.Fatalf("advanced index=%#v", advanced)
	}
	if advanced.Keys[0].Order != physical.IndexOrderDesc || advanced.Keys[0].NullsOrder != physical.IndexNullsOrderDefault {
		t.Fatalf("key options=%#v", advanced.Keys[0])
	}
	if advanced.State == nil || advanced.State.Valid || !advanced.State.Ready || !advanced.State.Live {
		t.Fatalf("state=%#v", advanced.State)
	}
	if _, ok := snap.Table("orders"); ok {
		t.Fatal("unqualified orders must be ambiguous across schemas")
	}
}

func TestPostgresProviderAdvertisesWorkbenchCapabilities(t *testing.T) {
	p := postgres.Provider{Catalog: fakeCatalog{}}
	caps, err := p.Capabilities(context.Background(), environment.DatabaseInstance{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []provider.Capability{provider.CapabilityMetadataRead, provider.CapabilityQuery, provider.CapabilitySchemaChange, provider.CapabilityExplain} {
		if !caps.Has(c) {
			t.Fatalf("missing capability %s", c)
		}
	}
}
