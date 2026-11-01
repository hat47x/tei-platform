package postgres

import (
	"context"
	"fmt"
	"strings"
)

// A primary key is unnamed in the declared schema (design 4.2, option C), so
// replacing an existing one needs the constraint's real name from the catalog.
// PostgreSQL's default name is not always "<table>_pkey": a collision adds a
// number, long names are truncated, and renaming the table keeps the old name.
// The name is therefore observed, never derived (#1803 P5).

// ReferencingForeignKey is a foreign key that depends on a primary key's index.
type ReferencingForeignKey struct {
	Constraint string   `json:"constraint"`
	Table      string   `json:"table"`
	Columns    []string `json:"columns"`
}

// PrimaryKeyIdentity is the observed primary key of one table.
type PrimaryKeyIdentity struct {
	Name                   string                  `json:"name"`
	Columns                []string                `json:"columns"`
	ReferencingForeignKeys []ReferencingForeignKey `json:"referencingForeignKeys,omitempty"`
}

// PrimaryKeyIdentityReader is an optional extension to CatalogReader that reads
// the real primary key constraint of a table and the foreign keys depending on
// it. It returns an error when the table has no primary key.
type PrimaryKeyIdentityReader interface {
	PrimaryKeyIdentity(ctx context.Context, table RelationIdentity) (PrimaryKeyIdentity, error)
}

func describeReferencingForeignKeys(fks []ReferencingForeignKey) string {
	parts := make([]string, len(fks))
	for i, fk := range fks {
		parts[i] = fmt.Sprintf("foreign key %q on %s (%s)", fk.Constraint, fk.Table, strings.Join(fk.Columns, ", "))
	}
	return strings.Join(parts, ", ")
}

const primaryKeyColumnsQuery = `
SELECT con.conname, att.attname
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class cls ON cls.oid = con.conrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
JOIN generate_subscripts(con.conkey, 1) AS pos(i) ON true
JOIN pg_catalog.pg_attribute att
  ON att.attrelid = cls.oid AND att.attnum = con.conkey[pos.i]
WHERE con.contype = 'p' AND con.conparentid = 0 AND ns.nspname = $1 AND cls.relname = $2
ORDER BY pos.i`

// referencingForeignKeysQuery selects the foreign keys whose referenced index
// is the primary key's own index. A foreign key that references another unique
// constraint of the table does not depend on the primary key.
const referencingForeignKeysQuery = `
SELECT fk.conname, fns.nspname, fcls.relname, att.attname
FROM pg_catalog.pg_constraint pk
JOIN pg_catalog.pg_class pcls ON pcls.oid = pk.conrelid
JOIN pg_catalog.pg_namespace pns ON pns.oid = pcls.relnamespace
JOIN pg_catalog.pg_constraint fk
  ON fk.contype = 'f' AND fk.confrelid = pk.conrelid AND fk.conindid = pk.conindid
JOIN pg_catalog.pg_class fcls ON fcls.oid = fk.conrelid
JOIN pg_catalog.pg_namespace fns ON fns.oid = fcls.relnamespace
JOIN generate_subscripts(fk.conkey, 1) AS pos(i) ON true
JOIN pg_catalog.pg_attribute att
  ON att.attrelid = fk.conrelid AND att.attnum = fk.conkey[pos.i]
WHERE pk.contype = 'p' AND pk.conparentid = 0 AND fk.conparentid = 0
  AND pns.nspname = $1 AND pcls.relname = $2
ORDER BY fns.nspname, fcls.relname, fk.conname, pos.i`

func (r SQLCatalogReader) PrimaryKeyIdentity(ctx context.Context, table RelationIdentity) (PrimaryKeyIdentity, error) {
	var out PrimaryKeyIdentity
	rows, err := r.DB.QueryContext(ctx, primaryKeyColumnsQuery, table.Schema, table.Name)
	if err != nil {
		return PrimaryKeyIdentity{}, err
	}
	for rows.Next() {
		var name, column string
		if err := rows.Scan(&name, &column); err != nil {
			rows.Close()
			return PrimaryKeyIdentity{}, err
		}
		out.Name = name
		out.Columns = append(out.Columns, column)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PrimaryKeyIdentity{}, err
	}
	rows.Close()
	if out.Name == "" {
		return PrimaryKeyIdentity{}, fmt.Errorf("table %s.%s has no primary key constraint", table.Schema, table.Name)
	}

	fkRows, err := r.DB.QueryContext(ctx, referencingForeignKeysQuery, table.Schema, table.Name)
	if err != nil {
		return PrimaryKeyIdentity{}, err
	}
	defer fkRows.Close()
	byKey := map[string]int{}
	for fkRows.Next() {
		var name, schema, child, column string
		if err := fkRows.Scan(&name, &schema, &child, &column); err != nil {
			return PrimaryKeyIdentity{}, err
		}
		k := schema + "." + child + "." + name
		i, ok := byKey[k]
		if !ok {
			out.ReferencingForeignKeys = append(out.ReferencingForeignKeys, ReferencingForeignKey{Constraint: name, Table: schema + "." + child})
			i = len(out.ReferencingForeignKeys) - 1
			byKey[k] = i
		}
		out.ReferencingForeignKeys[i].Columns = append(out.ReferencingForeignKeys[i].Columns, column)
	}
	return out, fkRows.Err()
}
