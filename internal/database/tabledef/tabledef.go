// Package tabledef derives the physical table a persistence mapping declares.
//
// It is the single source that turns one database-owned persistence.Mapping
// (plus the semantic.Data it realizes) into a physical.SchemaSnapshot: columns,
// primary key, unique constraints, foreign keys and indexes. It deliberately
// sits below both internal/artifact and internal/operations so each can read
// the same derivation without importing the other. internal/artifact depends on
// internal/operations, so a projection that must live in operations (model DDL)
// cannot reach this logic through artifact; moving it here keeps one derivation
// instead of two.
package tabledef

import (
	"sort"
	"strings"

	"tei/internal/core"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
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
	columnOf := map[core.CanonicalID]string{}
	for _, column := range mapping.Columns {
		table.Columns = append(table.Columns, physical.Column{
			Name:         column.Column,
			PhysicalType: string(column.Storage),
			Nullable:     column.Nullability == persistence.Optional,
			Ordinal:      len(table.Columns) + 1,
		})
		columnOf[column.Attribute] = column.Column
	}
	// A relation's support column is declared under relation-storage, not
	// under column, because it is not an attribute of the target Data: it
	// exists so the relation can be stored. It is still a physical column of
	// this table, so a table built from the mapping has to carry it; without
	// it the table cannot hold what the relation writes into it.
	// Components come after the declared columns, so the author's order is
	// not reshuffled by a storage detail.
	for _, storage := range mapping.RelationStorage {
		for _, component := range storage.Components {
			table.Columns = append(table.Columns, physical.Column{
				Name:         component.Column,
				PhysicalType: string(component.Storage),
				Nullable:     component.Nullability == persistence.Optional,
				Ordinal:      len(table.Columns) + 1,
			})
			columnOf[component.SourceAttribute] = component.Column
		}
	}
	// The primary key is physical metadata, so it names the physical columns,
	// resolved through the same column map the unique constraints and foreign
	// keys below use. A key the mapping does not declare as a column keeps its
	// declared identifier rather than being dropped.
	for _, key := range mapping.PrimaryKey {
		if column, ok := columnOf[key]; ok {
			table.PrimaryKey = append(table.PrimaryKey, column)
		} else {
			table.PrimaryKey = append(table.PrimaryKey, string(key))
		}
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
