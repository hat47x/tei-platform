package artifact

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/semantic"
)

func ddlCheckMapping() persistence.Mapping {
	return persistence.Mapping{
		ID:           "sales_core.order",
		Database:     "sales_core",
		SemanticData: "sales.order",
		Table:        "public.orders",
		PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
		Columns: []persistence.ColumnMapping{
			{Attribute: "sales.order.order_id", Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: "sales.order.status", Column: "status", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
		},
	}
}

func TestCompareTableDefinitionDetectsADifference(t *testing.T) {
	target := TableDefinitionSnapshot(ddlCheckMapping(), semanticData())
	target.Tables[0].Columns = target.Tables[0].Columns[:1]

	diff, err := CompareTableDefinition(ddlCheckMapping(), semanticData(), target)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(diff.Changes) == 0 {
		t.Fatal("a missing column must produce a change")
	}
}

func TestCompareTableDefinitionIsSilentWhenSchemasAgree(t *testing.T) {
	target := TableDefinitionSnapshot(ddlCheckMapping(), semanticData())

	diff, err := CompareTableDefinition(ddlCheckMapping(), semanticData(), target)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("identical schemas must produce no change: %+v", diff.Changes)
	}
}

func TestTableDefinitionSnapshotDescribesTheTable(t *testing.T) {
	snapshot := TableDefinitionSnapshot(ddlCheckMapping(), semanticData())
	if snapshot.Database != "sales_core" || len(snapshot.Tables) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	table := snapshot.Tables[0]
	if table.Schema != "public" || table.Name != "orders" {
		t.Fatalf("table identity lost: %+v", table)
	}
	if len(table.Columns) != 2 || table.Columns[0].Name != "order_id" || table.Columns[0].PhysicalType != "int64" {
		t.Fatalf("columns lost: %+v", table.Columns)
	}
	if table.Columns[0].Nullable || len(table.PrimaryKey) != 1 {
		t.Fatalf("nullability or key lost: %+v", table)
	}
}

func semanticData() semantic.Data {
	return semantic.Data{ID: "sales.order"}
}
func TestTableDefinitionSnapshotCarriesIndexes(t *testing.T) {
	mapping := ddlCheckMapping()
	indexes := []persistence.IndexMapping{{Name: "orders_status_idx", Attributes: []core.CanonicalID{"sales.order.status"}, Unique: true}}
	mapping.Indexes = indexes

	snapshot := TableDefinitionSnapshot(mapping, semanticData())
	table := snapshot.Tables[0]
	if len(table.Indexes) != 1 {
		t.Fatalf("index lost: %+v", table.Indexes)
	}
	index := table.Indexes[0]
	if index.Name != "orders_status_idx" || !index.Unique {
		t.Fatalf("index identity lost: %+v", index)
	}
	if len(index.Keys) != 1 || index.Keys[0].Column != "status" {
		t.Fatalf("the attribute must be resolved to its column: %+v", index.Keys)
	}
}

func TestCompareTableDefinitionDetectsAMissingIndex(t *testing.T) {
	mapping := ddlCheckMapping()
	mapping.Indexes = []persistence.IndexMapping{{Name: "orders_status_idx", Attributes: []core.CanonicalID{"sales.order.status"}}}

	withIndex := TableDefinitionSnapshot(mapping, semanticData())
	withoutIndex := TableDefinitionSnapshot(ddlCheckMapping(), semanticData())

	diff, err := CompareTableDefinition(mapping, semanticData(), withoutIndex)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if withIndex.Tables[0].Indexes == nil || len(diff.Changes) == 0 {
		t.Fatalf("a missing index must be reported: %+v", diff.Changes)
	}
}
func TestTableDefinitionSnapshotResolvesForeignKeys(t *testing.T) {
	customer := persistence.Mapping{
		ID: "sales_core.customer", Database: "sales_core", SemanticData: "sales.customer",
		Table: "public.customers",
		Columns: []persistence.ColumnMapping{
			{Attribute: "sales.customer.customer_id", Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
		},
	}
	mapping := ddlCheckMapping()
	mapping.Columns = append(mapping.Columns, persistence.ColumnMapping{
		Attribute: "sales.order.customer_id", Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required,
	})
	mapping.ForeignKeys = []persistence.ForeignKeyMapping{{
		Name:                 "orders_customer_fk",
		Attributes:           []core.CanonicalID{"sales.order.customer_id"},
		ReferencedMapping:    "sales_core.customer",
		ReferencedAttributes: []core.CanonicalID{"sales.customer.customer_id"},
	}}

	snapshot := TableDefinitionSnapshotWith(mapping, semanticData(), func(id core.CanonicalID) (persistence.Mapping, bool) {
		if id == customer.ID {
			return customer, true
		}
		return persistence.Mapping{}, false
	})
	table := snapshot.Tables[0]
	if len(table.ForeignKeys) != 1 {
		t.Fatalf("foreign key lost: %+v", table.ForeignKeys)
	}
	fk := table.ForeignKeys[0]
	if fk.Name != "orders_customer_fk" || fk.ReferencedSchema != "public" || fk.ReferencedTable != "customers" {
		t.Fatalf("reference not resolved: %+v", fk)
	}
	if len(fk.Columns) != 1 || fk.Columns[0] != "customer_id" || len(fk.ReferencedColumns) != 1 || fk.ReferencedColumns[0] != "customer_id" {
		t.Fatalf("columns not resolved: %+v", fk)
	}
}

func TestCompareTableDefinitionDetectsAMissingForeignKey(t *testing.T) {
	mapping := ddlCheckMapping()
	mapping.Columns = append(mapping.Columns, persistence.ColumnMapping{
		Attribute: "sales.order.customer_id", Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required,
	})
	withKey := mapping
	withKey.ForeignKeys = []persistence.ForeignKeyMapping{{
		Name:                 "orders_customer_fk",
		Attributes:           []core.CanonicalID{"sales.order.customer_id"},
		ReferencedMapping:    "sales_core.customer",
		ReferencedAttributes: []core.CanonicalID{"sales.customer.customer_id"},
	}}

	target := TableDefinitionSnapshot(mapping, semanticData())
	diff, err := CompareTableDefinition(withKey, semanticData(), target)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(diff.Changes) == 0 {
		t.Fatal("a missing foreign key must be reported")
	}
}
func TestTableDefinitionSnapshotCarriesUniqueConstraints(t *testing.T) {
	mapping := ddlCheckMapping()
	mapping.UniqueConstraints = []persistence.UniqueConstraintMapping{{
		Name:       "orders_status_key",
		Attributes: []core.CanonicalID{"sales.order.status"},
	}}

	snapshot := TableDefinitionSnapshot(mapping, semanticData())
	table := snapshot.Tables[0]
	if len(table.UniqueConstraints) != 1 {
		t.Fatalf("unique constraint lost: %+v", table.UniqueConstraints)
	}
	if table.UniqueConstraints[0].Name != "orders_status_key" || len(table.UniqueConstraints[0].Columns) != 1 || table.UniqueConstraints[0].Columns[0] != "status" {
		t.Fatalf("unique constraint not resolved: %+v", table.UniqueConstraints[0])
	}
}
func TestCompareTableDefinitionDetectsAMissingUniqueConstraint(t *testing.T) {
	mapping := ddlCheckMapping()
	mapping.UniqueConstraints = []persistence.UniqueConstraintMapping{{
		Name:       "orders_status_key",
		Attributes: []core.CanonicalID{"sales.order.status"},
	}}
	target := TableDefinitionSnapshot(ddlCheckMapping(), semanticData())

	diff, err := CompareTableDefinition(mapping, semanticData(), target)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if len(diff.Changes) == 0 {
		t.Fatal("a missing unique constraint must be reported")
	}
}
