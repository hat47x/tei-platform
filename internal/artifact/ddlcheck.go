package artifact

import (
	"fmt"
	"strings"

	"tei/internal/core"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/semantic"
)

// TableDefinitionSnapshot renders one persistence mapping as a physical schema
// snapshot. It bridges the readable table definition and the existing schema
// diff engine, so both read the same source instead of keeping two lists.
//
// Indexes carry their column names and uniqueness. Key expressions, collations
// and predicates are not compared yet: the persistence mapping does not declare
// them, so they are left empty instead of being guessed.
func TableDefinitionSnapshot(mapping persistence.Mapping, element semantic.Data) physical.SchemaSnapshot {
	_ = element
	table := physical.Table{
		Name: mapping.Table,
		Kind: physical.TableKindTable,
	}
	if schema, name, ok := strings.Cut(mapping.Table, "."); ok {
		table.Schema, table.Name = schema, name
	}
	for _, key := range mapping.PrimaryKey {
		table.PrimaryKey = append(table.PrimaryKey, string(key))
	}
	for _, column := range mapping.Columns {
		table.Columns = append(table.Columns, physical.Column{
			Name:         column.Column,
			PhysicalType: string(column.Storage),
			Nullable:     column.Nullability == persistence.Optional,
			Ordinal:      len(table.Columns) + 1,
		})
	}
	columnOf := map[core.CanonicalID]string{}
	for _, column := range mapping.Columns {
		columnOf[column.Attribute] = column.Column
	}
	for _, index := range mapping.Indexes {
		entry := physical.Index{Name: index.Name, Unique: index.Unique}
		for _, attribute := range index.Attributes {
			name, ok := columnOf[attribute]
			if !ok {
				name = string(attribute)
			}
			entry.Keys = append(entry.Keys, physical.IndexKey{Column: name, Order: physical.IndexOrderAsc})
		}
		table.Indexes = append(table.Indexes, entry)
	}
	return physical.SchemaSnapshot{
		Database: mapping.Database,
		Tables:   []physical.Table{table},
	}
}

// CompareTableDefinition compares the schema the table definition implies
// against a target snapshot (a desired schema or an observed one). No changes
// means the artifact and the schema agree; the diff is returned, never silently
// repaired.
func CompareTableDefinition(mapping persistence.Mapping, element semantic.Data, target physical.SchemaSnapshot) (schemadiff.Diff, error) {
	diff, err := schemadiff.Compare(TableDefinitionSnapshot(mapping, element).Normalize(), target.Normalize())
	if err != nil {
		return schemadiff.Diff{}, fmt.Errorf("compare table definition with the target schema: %w", err)
	}
	return diff, nil
}
