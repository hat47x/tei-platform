package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLCatalogReader is the standard-library adapter for a PostgreSQL connection.
// TEI does not choose the concrete database/sql driver; applications/editor
// assembly may use pgx/stdlib or another compatible driver.
type SQLCatalogReader struct{ DB *sql.DB }

func (r SQLCatalogReader) Tables(ctx context.Context) ([]TableMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT n.nspname, c.relname,
       CASE c.relkind WHEN 'v' THEN 'view' WHEN 'm' THEN 'materialized view' ELSE 'table' END
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r','p','v','m')
  AND n.nspname NOT IN ('pg_catalog','information_schema')
  AND n.nspname !~ '^pg_toast'
ORDER BY n.nspname, c.relname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TableMeta
	for rows.Next() {
		var v TableMeta
		if err := rows.Scan(&v.Schema, &v.Name, &v.Kind); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r SQLCatalogReader) Columns(ctx context.Context) ([]ColumnMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT ns.nspname, cls.relname, att.attname,
       typ_ns.nspname || '.' || typ.typname,
       NOT att.attnotnull,
       COALESCE(pg_catalog.pg_get_expr(def.adbin, def.adrelid), ''),
       att.attnum
FROM pg_catalog.pg_class cls
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
JOIN pg_catalog.pg_attribute att ON att.attrelid = cls.oid
JOIN pg_catalog.pg_type typ ON typ.oid = att.atttypid
JOIN pg_catalog.pg_namespace typ_ns ON typ_ns.oid = typ.typnamespace
LEFT JOIN pg_catalog.pg_attrdef def
  ON def.adrelid = cls.oid AND def.adnum = att.attnum
WHERE cls.relkind IN ('r','p','v','m')
  AND ns.nspname NOT IN ('pg_catalog','information_schema')
  AND ns.nspname !~ '^pg_toast'
  AND att.attnum > 0
  AND NOT att.attisdropped
ORDER BY ns.nspname, cls.relname, att.attnum`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ColumnMeta
	for rows.Next() {
		var v ColumnMeta
		if err := rows.Scan(&v.Schema, &v.Table, &v.Name, &v.PhysicalType, &v.Nullable, &v.Default, &v.Ordinal); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r SQLCatalogReader) PrimaryKeys(ctx context.Context) ([]PrimaryKeyMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT ns.nspname, cls.relname, att.attname
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class cls ON cls.oid = con.conrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
JOIN generate_subscripts(con.conkey, 1) AS pos(i) ON true
JOIN pg_catalog.pg_attribute att
  ON att.attrelid = cls.oid AND att.attnum = con.conkey[pos.i]
WHERE con.contype = 'p'
ORDER BY ns.nspname, cls.relname, pos.i`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PrimaryKeyMeta
	byKey := map[string]int{}
	for rows.Next() {
		var schema, table, column string
		if err := rows.Scan(&schema, &table, &column); err != nil {
			return nil, err
		}
		k := schema + "." + table
		i, ok := byKey[k]
		if !ok {
			out = append(out, PrimaryKeyMeta{Schema: schema, Table: table})
			i = len(out) - 1
			byKey[k] = i
		}
		out[i].Columns = append(out[i].Columns, column)
	}
	return out, rows.Err()
}

func (r SQLCatalogReader) UniqueConstraints(ctx context.Context) ([]UniqueConstraintMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT ns.nspname, cls.relname, con.conname, att.attname, ind.indnullsnotdistinct
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class cls ON cls.oid = con.conrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
JOIN pg_catalog.pg_index ind ON ind.indexrelid = con.conindid
JOIN generate_subscripts(con.conkey, 1) AS pos(i) ON true
JOIN pg_catalog.pg_attribute att
  ON att.attrelid = cls.oid AND att.attnum = con.conkey[pos.i]
WHERE con.contype = 'u'
ORDER BY ns.nspname, cls.relname, con.conname, pos.i`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UniqueConstraintMeta
	byKey := map[string]int{}
	for rows.Next() {
		var schema, table, name, column string
		var nullsNotDistinct bool
		if err := rows.Scan(&schema, &table, &name, &column, &nullsNotDistinct); err != nil {
			return nil, err
		}
		if nullsNotDistinct {
			return nil, fmt.Errorf("unique constraint %s.%s.%s uses NULLS NOT DISTINCT, which M0.28 does not model", schema, table, name)
		}
		k := schema + "." + table + "." + name
		i, ok := byKey[k]
		if !ok {
			out = append(out, UniqueConstraintMeta{Schema: schema, Table: table, Name: name})
			i = len(out) - 1
			byKey[k] = i
		}
		out[i].Columns = append(out[i].Columns, column)
	}
	return out, rows.Err()
}

func (r SQLCatalogReader) ForeignKeys(ctx context.Context) ([]ForeignKeyMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT src_ns.nspname, src.relname, con.conname, src_att.attname,
       dst_ns.nspname, dst.relname, dst_att.attname
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class src ON src.oid = con.conrelid
JOIN pg_catalog.pg_namespace src_ns ON src_ns.oid = src.relnamespace
JOIN pg_catalog.pg_class dst ON dst.oid = con.confrelid
JOIN pg_catalog.pg_namespace dst_ns ON dst_ns.oid = dst.relnamespace
JOIN generate_subscripts(con.conkey, 1) AS pos(i) ON true
JOIN pg_catalog.pg_attribute src_att
  ON src_att.attrelid = src.oid AND src_att.attnum = con.conkey[pos.i]
JOIN pg_catalog.pg_attribute dst_att
  ON dst_att.attrelid = dst.oid AND dst_att.attnum = con.confkey[pos.i]
WHERE con.contype = 'f'
ORDER BY src_ns.nspname, src.relname, con.conname, pos.i`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ForeignKeyMeta
	byKey := map[string]int{}
	for rows.Next() {
		var schema, table, name, column, refSchema, refTable, refColumn string
		if err := rows.Scan(&schema, &table, &name, &column, &refSchema, &refTable, &refColumn); err != nil {
			return nil, err
		}
		k := schema + "." + table + "." + name
		i, ok := byKey[k]
		if !ok {
			out = append(out, ForeignKeyMeta{Schema: schema, Table: table, Name: name, ReferencedSchema: refSchema, ReferencedTable: refTable})
			i = len(out) - 1
			byKey[k] = i
		}
		out[i].Columns = append(out[i].Columns, column)
		out[i].ReferencedColumns = append(out[i].ReferencedColumns, refColumn)
	}
	return out, rows.Err()
}

func (r SQLCatalogReader) Indexes(ctx context.Context) ([]IndexMeta, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT ns.nspname, tbl.relname, idx.relname, ind.indisunique, am.amname,
       pos.ord, ind.indnkeyatts,
       COALESCE(att.attname, ''),
       CASE WHEN pos.ord <= ind.indnkeyatts AND pos.attnum = 0
            THEN pg_catalog.pg_get_indexdef(ind.indexrelid, pos.ord::int, true)
            ELSE '' END,
       CASE WHEN pos.ord <= ind.indnkeyatts AND coll.oid <> 0
                 AND (att.attname IS NULL OR coll.oid <> att.attcollation)
            THEN coll_ns.nspname || '.' || coll.collname ELSE '' END,
       CASE WHEN pos.ord <= ind.indnkeyatts AND opc.oid <> 0 AND NOT opc.opcdefault
            THEN opc_ns.nspname || '.' || opc.opcname ELSE '' END,
       CASE WHEN pos.ord <= ind.indnkeyatts THEN COALESCE(pg_catalog.pg_index_column_has_property(ind.indexrelid, pos.ord::int, 'desc'), false) ELSE false END,
       CASE WHEN pos.ord <= ind.indnkeyatts THEN COALESCE(pg_catalog.pg_index_column_has_property(ind.indexrelid, pos.ord::int, 'nulls_first'), false) ELSE false END,
       COALESCE(pg_catalog.pg_get_expr(ind.indpred, ind.indrelid, true), ''),
       ind.indnullsnotdistinct, ind.indisvalid, ind.indisready, ind.indislive
FROM pg_catalog.pg_index ind
JOIN pg_catalog.pg_class tbl ON tbl.oid = ind.indrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = tbl.relnamespace
JOIN pg_catalog.pg_class idx ON idx.oid = ind.indexrelid
JOIN pg_catalog.pg_am am ON am.oid = idx.relam
CROSS JOIN LATERAL unnest(ind.indkey) WITH ORDINALITY AS pos(attnum, ord)
LEFT JOIN pg_catalog.pg_attribute att ON att.attrelid = tbl.oid AND att.attnum = pos.attnum
LEFT JOIN pg_catalog.pg_collation coll ON coll.oid = CASE WHEN pos.ord <= ind.indnkeyatts THEN ind.indcollation[(pos.ord - 1)::int] ELSE 0 END
LEFT JOIN pg_catalog.pg_namespace coll_ns ON coll_ns.oid = coll.collnamespace
LEFT JOIN pg_catalog.pg_opclass opc ON opc.oid = CASE WHEN pos.ord <= ind.indnkeyatts THEN ind.indclass[(pos.ord - 1)::int] ELSE 0 END
LEFT JOIN pg_catalog.pg_namespace opc_ns ON opc_ns.oid = opc.opcnamespace
WHERE ns.nspname NOT IN ('pg_catalog','information_schema')
  AND ns.nspname !~ '^pg_toast'
  AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_constraint con
      WHERE con.conindid = ind.indexrelid AND con.contype IN ('p','u')
  )
ORDER BY ns.nspname, tbl.relname, idx.relname, pos.ord`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndexMeta
	byKey := map[string]int{}
	for rows.Next() {
		var schema, table, name, method, column, expression, collation, opclass, predicate string
		var unique, descending, nullsFirst, nullsNotDistinct, valid, ready, live bool
		var ordinal, keyCount int
		if err := rows.Scan(&schema, &table, &name, &unique, &method, &ordinal, &keyCount, &column, &expression, &collation, &opclass, &descending, &nullsFirst, &predicate, &nullsNotDistinct, &valid, &ready, &live); err != nil {
			return nil, err
		}
		k := schema + "." + table + "." + name
		i, ok := byKey[k]
		if !ok {
			out = append(out, IndexMeta{Schema: schema, Table: table, Name: name, Unique: unique, NullsNotDistinct: nullsNotDistinct, Method: method, Predicate: predicate, Valid: valid, Ready: ready, Live: live})
			i = len(out) - 1
			byKey[k] = i
		}
		if ordinal <= keyCount {
			out[i].Keys = append(out[i].Keys, IndexKeyMeta{Column: column, Expression: expression, Collation: collation, OperatorClass: opclass, Descending: descending, NullsFirst: nullsFirst})
		} else if column != "" {
			out[i].IncludedColumns = append(out[i].IncludedColumns, column)
		}
	}
	return out, rows.Err()
}
