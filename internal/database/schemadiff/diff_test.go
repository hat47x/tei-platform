package schemadiff_test

import (
	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"testing"
)

func TestCompareDetectsOnlyAddedColumn(t *testing.T) {
	actual := physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core"), Tables: []physical.Table{{Schema: "public", Name: "orders", Kind: physical.TableKindTable, Columns: []physical.Column{{Name: "order_id", PhysicalType: "pg_catalog.int8", Nullable: false}}, PrimaryKey: []string{"order_id"}}}}
	desired := actual
	desired.Tables = append([]physical.Table(nil), actual.Tables...)
	desired.Tables[0].Columns = append([]physical.Column(nil), actual.Tables[0].Columns...)
	desired.Tables[0].Columns = append(desired.Tables[0].Columns, physical.Column{Name: "external_id", PhysicalType: "pg_catalog.varchar", Nullable: true})
	d, err := schemadiff.Compare(desired.Normalize(), actual.Normalize())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Changes) != 1 || d.Changes[0].Kind != schemadiff.AddColumn || d.Changes[0].Column.Name != "external_id" {
		t.Fatalf("unexpected diff: %#v", d)
	}
}
