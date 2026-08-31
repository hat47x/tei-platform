package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func TestForeignKeyNameOnlyDifferenceIsNotDrift(t *testing.T) {
	desired := schemaWithForeignKeys(physical.ForeignKey{Name: "desired_name", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}})
	actual := schemaWithForeignKeys(physical.ForeignKey{Name: "generated_name", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}})
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("name-only difference produced drift: %#v", diff.Changes)
	}
}

func TestForeignKeyStructuralDifferenceIsDropAndAdd(t *testing.T) {
	desired := schemaWithForeignKeys(physical.ForeignKey{Name: "orders_customer_fk", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}})
	actual := schemaWithForeignKeys(physical.ForeignKey{Name: "orders_customer_fk", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "legacy_customers", ReferencedColumns: []string{"id"}})
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropForeignKey || diff.Changes[1].Kind != schemadiff.AddForeignKey {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}

func TestForeignKeyCompositeColumnOrderIsMeaningful(t *testing.T) {
	desired := schemaWithForeignKeys(physical.ForeignKey{Columns: []string{"tenant_id", "product_id"}, ReferencedSchema: "public", ReferencedTable: "products", ReferencedColumns: []string{"tenant_id", "id"}})
	actual := schemaWithForeignKeys(physical.ForeignKey{Columns: []string{"product_id", "tenant_id"}, ReferencedSchema: "public", ReferencedTable: "products", ReferencedColumns: []string{"id", "tenant_id"}})
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropForeignKey || diff.Changes[1].Kind != schemadiff.AddForeignKey {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}

func TestForeignKeyMultiplicityIsPreserved(t *testing.T) {
	fk1 := physical.ForeignKey{Name: "fk_one", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}}
	fk2 := fk1
	fk2.Name = "fk_two"
	desired := schemaWithForeignKeys(fk1)
	actual := schemaWithForeignKeys(fk1, fk2)
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.DropForeignKey || diff.Changes[0].ForeignKey == nil || diff.Changes[0].ForeignKey.Name != "fk_two" {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}

func TestCreateTableIncludesDesiredForeignKey(t *testing.T) {
	desired := schemaWithForeignKeys(physical.ForeignKey{Name: "orders_customer_fk", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}})
	actual := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")}
	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, change := range diff.Changes {
		if change.Kind == schemadiff.AddForeignKey {
			found = true
		}
	}
	if !found {
		t.Fatalf("foreign key missing from create-table diff: %#v", diff.Changes)
	}
}

func schemaWithForeignKeys(fks ...physical.ForeignKey) physical.SchemaSnapshot {
	return physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables: []physical.Table{{
			Schema:      "public",
			Name:        "orders",
			Kind:        physical.TableKindTable,
			Columns:     []physical.Column{{Name: "customer_id", PhysicalType: "pg_catalog.int8"}, {Name: "tenant_id", PhysicalType: "pg_catalog.int8"}, {Name: "product_id", PhysicalType: "pg_catalog.int8"}},
			ForeignKeys: fks,
		}},
	}
}
