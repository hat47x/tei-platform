package tabledef

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/semantic"
)

// TestRelationStorageComponentsBecomeColumns pins the gap that made the
// generated DDL unusable against the workspace's own seed data.
//
// A relation's support column is declared under `relation-storage`, not under
// `column`, because it is not an attribute of the target Data: it exists so the
// relation can be stored. `reference/workspace` declares `sales.order.order_id`
// as such a component of `sales_core.order_line`, and the real seed data inserts
// `order_id` into `public.order_lines`. Before this change the derivation
// emitted only the `column` entries, so the generated table had no `order_id`
// column and the seed INSERT could not run.
func TestRelationStorageComponentsBecomeColumns(t *testing.T) {
	mapping := persistence.Mapping{
		Database: mustID(t, "sales_core"),
		Table:    "public.order_lines",
		Columns: []persistence.ColumnMapping{
			{Attribute: mustID(t, "sales.order_line.order_line_id"), Column: "order_line_id",
				Storage: persistence.StorageInt64, Nullability: persistence.Required},
			{Attribute: mustID(t, "sales.order_line.line_number"), Column: "line_number",
				Storage: persistence.StorageInt64, Nullability: persistence.Required},
		},
		RelationStorage: []persistence.RelationStorageMapping{
			{
				Relation: mustID(t, "sales.order.lines"),
				Components: []persistence.RelationStorageComponentMapping{
					{SourceAttribute: mustID(t, "sales.order.order_id"), Column: "order_id",
						Storage: persistence.StorageInt64, Nullability: persistence.Required},
				},
			},
		},
		PrimaryKey: []core.CanonicalID{mustID(t, "sales.order_line.order_line_id")},
	}

	snapshot := TableDefinitionSnapshot(mapping, semantic.Data{})
	table, ok := snapshot.Table("order_lines")
	if !ok && len(snapshot.Tables) == 1 {
		table = snapshot.Tables[0]
		ok = true
	}
	if !ok {
		t.Fatalf("no table in the snapshot: %+v", snapshot.Tables)
	}

	columns := map[string]physical.Column{}
	for _, column := range table.Columns {
		columns[column.Name] = column
	}
	if _, present := columns["order_id"]; !present {
		t.Fatalf("the relation-storage component is not a column: got %v",
			columnNames(table.Columns))
	}
	if got := columns["order_id"].PhysicalType; got != string(persistence.StorageInt64) {
		t.Fatalf("order_id type = %q, want %q", got, persistence.StorageInt64)
	}
	if columns["order_id"].Nullable {
		t.Fatal("order_id is declared required, but the column is nullable")
	}

	// The declared columns stay, and stay in their declared order first, so the
	// support column does not reshuffle what the author wrote.
	if len(table.Columns) != 3 {
		t.Fatalf("column count = %d, want 3 (two declared + one support): %v",
			len(table.Columns), columnNames(table.Columns))
	}
	if table.Columns[0].Name != "order_line_id" || table.Columns[1].Name != "line_number" {
		t.Fatalf("declared columns moved: %v", columnNames(table.Columns))
	}

	// The primary key stays resolved to the physical column it names.
	if len(table.PrimaryKey) != 1 || table.PrimaryKey[0] != "order_line_id" {
		t.Fatalf("primary key = %v, want [order_line_id]", table.PrimaryKey)
	}
}

func columnNames(columns []physical.Column) []string {
	names := make([]string, 0, len(columns))
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return names
}

func mustID(t *testing.T, value string) core.CanonicalID {
	t.Helper()
	id, err := core.ParseCanonicalID(value)
	if err != nil {
		t.Fatalf("canonical id %q: %v", value, err)
	}
	return id
}
