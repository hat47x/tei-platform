package postgres

import "context"

// Partitions observes PostgreSQL inheritance edges used by declarative
// partitioning. The result keeps relation identity as schema + name and leaves
// partition strategy/bounds outside the current M0.x contract.
func (r SQLCatalogReader) Partitions(ctx context.Context) ([]PartitionRelation, error) {
	rows, err := r.DB.QueryContext(ctx, `
SELECT child_ns.nspname, child.relname,
       parent_ns.nspname, parent.relname
FROM pg_catalog.pg_inherits inh
JOIN pg_catalog.pg_class child ON child.oid = inh.inhrelid
JOIN pg_catalog.pg_namespace child_ns ON child_ns.oid = child.relnamespace
JOIN pg_catalog.pg_class parent ON parent.oid = inh.inhparent
JOIN pg_catalog.pg_namespace parent_ns ON parent_ns.oid = parent.relnamespace
WHERE child.relispartition
  AND child_ns.nspname NOT IN ('pg_catalog','information_schema')
  AND child_ns.nspname !~ '^pg_toast'
  AND parent_ns.nspname NOT IN ('pg_catalog','information_schema')
  AND parent_ns.nspname !~ '^pg_toast'
ORDER BY child_ns.nspname, child.relname, parent_ns.nspname, parent.relname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PartitionRelation
	for rows.Next() {
		var edge PartitionRelation
		if err := rows.Scan(
			&edge.Child.Schema,
			&edge.Child.Name,
			&edge.Parent.Schema,
			&edge.Parent.Name,
		); err != nil {
			return nil, err
		}
		out = append(out, edge)
	}
	return out, rows.Err()
}
