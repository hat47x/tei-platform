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
	for _, m := range mappings {
		if string(m.Database) != databaseID {
			continue
		}
		schema, table := splitQualified(m.Table)
		t := physical.Table{Schema: schema, Name: table, Kind: physical.TableKindTable}
		columnByAttribute := map[string]string{}
		for i, c := range m.Columns {
			pt, err := physicalType(provider, c.Storage)
			if err != nil {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s attribute %s: %w", m.ID, c.Attribute, err)
			}
			t.Columns = append(t.Columns, physical.Column{Name: c.Column, PhysicalType: pt, Nullable: c.Nullability == Optional, Ordinal: i + 1})
			columnByAttribute[string(c.Attribute)] = c.Column
		}
		for _, attr := range m.PrimaryKey {
			name, ok := columnByAttribute[string(attr)]
			if !ok {
				return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s primary key attribute %s has no column mapping", m.ID, attr)
			}
			t.PrimaryKey = append(t.PrimaryKey, name)
		}
		for _, idx := range m.Indexes {
			pi := physical.Index{Name: idx.Name, Unique: idx.Unique, Method: defaultIndexMethod(provider)}
			for _, attr := range idx.Attributes {
				name, ok := columnByAttribute[string(attr)]
				if !ok {
					return physical.SchemaSnapshot{}, fmt.Errorf("mapping %s index %s attribute %s has no column mapping", m.ID, idx.Name, attr)
				}
				pi.Columns = append(pi.Columns, name)
			}
			t.Indexes = append(t.Indexes, pi)
		}
		out.Tables = append(out.Tables, t)
	}
	return out.Normalize(), nil
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
