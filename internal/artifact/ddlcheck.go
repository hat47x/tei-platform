package artifact

import (
	"fmt"
	"sort"
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
	return TableDefinitionSnapshotWith(mapping, element, nil)
}

// TableDefinitionSnapshotWith also resolves the referenced mappings, so foreign
// keys carry the table and columns they point at. A nil resolver leaves them to
// the declared identifiers.
func TableDefinitionSnapshotWith(mapping persistence.Mapping, element semantic.Data, resolve func(core.CanonicalID) (persistence.Mapping, bool)) physical.SchemaSnapshot {
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
	for _, unique := range mapping.UniqueConstraints {
		entry := physical.UniqueConstraint{Name: unique.Name}
		for _, attribute := range unique.Attributes {
			if column, ok := columnOf[attribute]; ok {
				entry.Columns = append(entry.Columns, column)
			} else {
				entry.Columns = append(entry.Columns, string(attribute))
			}
		}
		table.UniqueConstraints = append(table.UniqueConstraints, entry)
	}

	foreignKeys := append([]persistence.ForeignKeyMapping(nil), mapping.ForeignKeys...)
	sort.Slice(foreignKeys, func(i, j int) bool { return foreignKeys[i].Name < foreignKeys[j].Name })
	for _, foreignKey := range foreignKeys {
		entry := physical.ForeignKey{Name: foreignKey.Name}
		for _, attribute := range foreignKey.Attributes {
			if column, ok := columnOf[attribute]; ok {
				entry.Columns = append(entry.Columns, column)
			} else {
				entry.Columns = append(entry.Columns, string(attribute))
			}
		}
		referenced, ok := persistence.Mapping{}, false
		if resolve != nil {
			referenced, ok = resolve(foreignKey.ReferencedMapping)
		}
		tableName := string(foreignKey.ReferencedMapping)
		if ok {
			tableName = referenced.Table
		}
		if schema, name, cut := strings.Cut(tableName, "."); cut {
			entry.ReferencedSchema, entry.ReferencedTable = schema, name
		} else {
			entry.ReferencedTable = tableName
		}
		for _, attribute := range foreignKey.ReferencedAttributes {
			if ok {
				for _, column := range referenced.Columns {
					if column.Attribute == attribute {
						entry.ReferencedColumns = append(entry.ReferencedColumns, column.Column)
					}
				}
			} else {
				entry.ReferencedColumns = append(entry.ReferencedColumns, string(attribute))
			}
		}
		table.ForeignKeys = append(table.ForeignKeys, entry)
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
	return CompareTableDefinitionWith(mapping, element, target, nil)
}

// CompareTableDefinitionWith resolves referenced mappings, so foreign keys are
// compared by the table and columns they point at.
func CompareTableDefinitionWith(mapping persistence.Mapping, element semantic.Data, target physical.SchemaSnapshot, resolve func(core.CanonicalID) (persistence.Mapping, bool)) (schemadiff.Diff, error) {
	diff, err := schemadiff.Compare(TableDefinitionSnapshotWith(mapping, element, resolve).Normalize(), target.Normalize())
	if err != nil {
		return schemadiff.Diff{}, fmt.Errorf("compare table definition with the target schema: %w", err)
	}
	return diff, nil
}
