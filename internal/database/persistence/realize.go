package persistence

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/environment"
)

// RealizeDesired turns provider-neutral relational storage types into a
// provider-specific desired physical schema. The mapping remains canonical;
// this snapshot is a deterministic projection and can be regenerated.
func RealizeDesired(mappings []Mapping, databaseID string, provider environment.DatabaseProvider) (physical.SchemaSnapshot, error) {
	out := physical.SchemaSnapshot{Database: mustDatabaseID(databaseID)}
	mappingByID := make(map[core.CanonicalID]Mapping, len(mappings))
	for _, m := range mappings {
		mappingByID[m.ID] = m
	}

	for _, m := range mappings {
		if string(m.Database) != databaseID {
			continue
		}
		schema, table := splitQualified(m.Table)
		t := physical.Table{Schema: schema, Name: table, Kind: physical.TableKindTable}
		columnByAttribute := columnNamesByAttribute(m)
		for i, c := range m.Columns {
			pt, err := physicalType(provider, c.Storage)
			if err != nil {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s attribute %s: %w", m.ID, c.Attribute, err)
			}
			t.Columns = append(t.Columns, physical.Column{Name: c.Column, PhysicalType: pt, Nullable: c.Nullability == Optional, Ordinal: i + 1})
		}
		for _, attr := range m.PrimaryKey {
			name, ok := columnByAttribute[attr]
			if !ok {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s primary key attribute %s has no column mapping", m.ID, attr)
			}
			t.PrimaryKey = append(t.PrimaryKey, name)
		}
		for _, constraint := range m.UniqueConstraints {
			uc, err := realizeUniqueConstraint(m, constraint, columnByAttribute)
			if err != nil {
				return physical.SchemaSnapshot{}, err
			}
			t.UniqueConstraints = append(t.UniqueConstraints, uc)
		}
		for _, idx := range m.Indexes {
			pi := physical.Index{Name: idx.Name, Unique: idx.Unique, Method: defaultIndexMethod(provider)}
			for _, attr := range idx.Attributes {
				name, ok := columnByAttribute[attr]
				if !ok {
					return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s index %s attribute %s has no column mapping", m.ID, idx.Name, attr)
				}
				pi.Columns = append(pi.Columns, name)
			}
			t.Indexes = append(t.Indexes, pi)
		}
		for _, fk := range m.ForeignKeys {
			pfk, err := realizeForeignKey(m, fk, mappingByID, columnByAttribute)
			if err != nil {
				return physical.SchemaSnapshot{}, err
			}
			t.ForeignKeys = append(t.ForeignKeys, pfk)
		}
		out.Tables = append(out.Tables, t)
	}
	return out.Normalize(), nil
}

func realizeUniqueConstraint(source Mapping, constraint UniqueConstraintMapping, columns map[core.CanonicalID]string) (physical.UniqueConstraint, error) {
	if len(constraint.Attributes) == 0 {
		return physical.UniqueConstraint{}, fmt.Errorf("mapping %s unique constraint %s has no attributes", source.ID, constraint.Name)
	}
	out := physical.UniqueConstraint{Name: constraint.Name}
	for _, attr := range constraint.Attributes {
		name, ok := columns[attr]
		if !ok {
			return physical.UniqueConstraint{}, fmt.Errorf("mapping %s unique constraint %s attribute %s has no column mapping", source.ID, constraint.Name, attr)
		}
		out.Columns = append(out.Columns, name)
	}
	return out, nil
}

func realizeForeignKey(source Mapping, fk ForeignKeyMapping, mappings map[core.CanonicalID]Mapping, sourceColumns map[core.CanonicalID]string) (physical.ForeignKey, error) {
	if len(fk.Attributes) == 0 {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s has no source attributes", source.ID, fk.Name)
	}
	if len(fk.Attributes) != len(fk.ReferencedAttributes) {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s source/target attribute count mismatch: %d != %d", source.ID, fk.Name, len(fk.Attributes), len(fk.ReferencedAttributes))
	}
	target, ok := mappings[fk.ReferencedMapping]
	if !ok {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s references unknown mapping %s", source.ID, fk.Name, fk.ReferencedMapping)
	}
	if target.Database != source.Database {
		return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s crosses database boundary from %s to %s", source.ID, fk.Name, source.Database, target.Database)
	}
	targetColumns := columnNamesByAttribute(target)
	refSchema, refTable := splitQualified(target.Table)
	out := physical.ForeignKey{Name: fk.Name, ReferencedSchema: refSchema, ReferencedTable: refTable}
	for i, attr := range fk.Attributes {
		name, ok := sourceColumns[attr]
		if !ok {
			return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s source attribute %s has no column mapping", source.ID, fk.Name, attr)
		}
		refAttr := fk.ReferencedAttributes[i]
		refName, ok := targetColumns[refAttr]
		if !ok {
			return physical.ForeignKey{}, fmt.Errorf("mapping %s foreign key %s target attribute %s has no column mapping in %s", source.ID, fk.Name, refAttr, target.ID)
		}
		out.Columns = append(out.Columns, name)
		out.ReferencedColumns = append(out.ReferencedColumns, refName)
	}
	return out, nil
}

func columnNamesByAttribute(m Mapping) map[core.CanonicalID]string {
	out := make(map[core.CanonicalID]string, len(m.Columns))
	for _, c := range m.Columns {
		out[c.Attribute] = c.Column
	}
	return out
}

func physicalType(provider environment.DatabaseProvider, storage StorageType) (string, error) {
	switch provider {
	case environment.ProviderPostgres:
		switch storage {
		case StorageInt64:
			return "pg_catalog.int8", nil
		case StorageVarchar:
			return "pg_catalog.varchar", nil
		case StorageText:
			return "pg_catalog.text", nil
		case StorageDateTimeTZ:
			return "pg_catalog.timestamptz", nil
		case StorageDecimal:
			return "pg_catalog.numeric", nil
		case StorageJSON:
			return "pg_catalog.jsonb", nil
		}
	case environment.ProviderOracle:
		switch storage {
		case StorageInt64:
			return "NUMBER(19)", nil
		case StorageVarchar:
			return "VARCHAR2(255)", nil
		case StorageText:
			return "CLOB", nil
		case StorageDateTimeTZ:
			return "TIMESTAMP WITH TIME ZONE", nil
		case StorageDecimal:
			return "NUMBER", nil
		case StorageJSON:
			return "CLOB", nil
		}
	default:
		return "", fmt.Errorf("unsupported database provider %q", provider)
	}
	return "", fmt.Errorf("unsupported storage type %q for provider %q", storage, provider)
}

func defaultIndexMethod(provider environment.DatabaseProvider) string {
	if provider == environment.ProviderPostgres {
		return "btree"
	}
	return ""
}

func mustDatabaseID(s string) core.CanonicalID { return core.MustCanonicalID(s) }

func splitQualified(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}
