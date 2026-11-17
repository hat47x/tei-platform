package tabledef

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/semantic"
)

func mapping() persistence.Mapping {
	return persistence.Mapping{
		ID:           "sales_core.order",
		Database:     "sales_core",
		SemanticData: "sales.order",
		Table:        "public.orders",
		PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
		Columns: []persistence.ColumnMapping{
			{Attribute: "sales.order.order_id", Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: "sales.order.customer_id", Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
		},
	}
}

// The primary key is physical metadata: it names the physical columns, resolved
// through the declared column map, not the semantic attribute identifiers the
// mapping wrote.
func TestTableDefinitionSnapshotResolvesPrimaryKeyToPhysicalColumns(t *testing.T) {
	snapshot := TableDefinitionSnapshot(mapping(), semantic.Data{ID: "sales.order"})
	if len(snapshot.Tables) != 1 {
		t.Fatalf("tables=%d, want 1", len(snapshot.Tables))
	}
	got := snapshot.Tables[0].PrimaryKey
	if len(got) != 1 || got[0] != "order_id" {
		t.Fatalf("primary key=%v, want [order_id]", got)
	}
}

// A key the mapping does not declare as a column keeps its declared identifier
// rather than being silently dropped.
func TestTableDefinitionSnapshotKeepsUndeclaredPrimaryKeyIdentifier(t *testing.T) {
	m := mapping()
	m.PrimaryKey = []core.CanonicalID{"sales.order.missing"}
	got := TableDefinitionSnapshot(m, semantic.Data{ID: "sales.order"}).Tables[0].PrimaryKey
	if len(got) != 1 || got[0] != "sales.order.missing" {
		t.Fatalf("primary key=%v, want [sales.order.missing]", got)
	}
}
