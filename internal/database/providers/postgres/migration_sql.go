package postgres

import (
	"fmt"
	"strings"

	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

// RenderMigrationSQL renders one fully resolved Schema Diff as PostgreSQL DDL
// when it contains no DropTable. DropTable requires provider-owned partition
// observation and must use RenderMigrationSQLWithObservation instead.
//
// Provider-native expressions are treated as trusted database-owned source
// fragments. This renderer validates their provenance, but deliberately does
// not rewrite or heuristically sanitize SQL text.
func RenderMigrationSQL(diff schemadiff.Diff) ([]string, error) {
	if err := validateResolvedMigrationDiff(diff); err != nil {
		return nil, err
	}
	for _, change := range diff.Changes {
		if change.Kind == schemadiff.DropTable {
			return nil, fmt.Errorf("postgres drop-table rendering requires complete provider partition observation")
		}
	}
	return renderResolvedMigrationSQL(diff)
}

func validateResolvedMigrationDiff(diff schemadiff.Diff) error {
	if len(diff.Unresolved) > 0 {
		return fmt.Errorf("migration SQL rendering requires a fully resolved schema diff: %d unresolved comparison(s)", len(diff.Unresolved))
	}
	return nil
}

// renderResolvedMigrationSQL is package-private because it can emit destructive
// DROP TABLE after its caller has established the provider-specific safety
// preconditions. Exported callers must go through RenderMigrationSQL or
// RenderMigrationSQLWithObservation.
func renderResolvedMigrationSQL(diff schemadiff.Diff) ([]string, error) {
	var out []string
	for _, change := range diff.Changes {
		switch change.Kind {
		case schemadiff.CreateTable:
			if len(change.PrimaryKey) > 0 {
				return nil, fmt.Errorf("postgres migration SQL rendering does not support primary key on newly created table %s until constraint identity is realizable", change.Table)
			}
			// Columns are emitted by following AddColumn changes to retain one common change model.
			out = append(out, fmt.Sprintf("CREATE TABLE %s ();", quoteQualifiedName(change.Table)))
		case schemadiff.DropTable:
			out = append(out, fmt.Sprintf("DROP TABLE %s;", quoteQualifiedName(change.Table)))
		case schemadiff.AddColumn:
			if change.Column == nil {
				return nil, fmt.Errorf("add-column %s has no column", change.Table)
			}
			stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", quoteQualifiedName(change.Table), quoteIdentifier(change.Column.Name), change.Column.PhysicalType)
			if !change.Column.Nullable {
				stmt += " NOT NULL"
			}
			if strings.TrimSpace(change.Column.Default) != "" {
				stmt += " DEFAULT " + change.Column.Default
			}
			out = append(out, stmt+";")
		case schemadiff.DropColumn:
			if change.BeforeColumn == nil {
				return nil, fmt.Errorf("drop-column %s has no before column", change.Table)
			}
			out = append(out, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", quoteQualifiedName(change.Table), quoteIdentifier(change.BeforeColumn.Name)))
		case schemadiff.AlterColumn:
			if change.Column == nil {
				return nil, fmt.Errorf("alter-column %s has no target column", change.Table)
			}
			if change.BeforeColumn == nil {
				return nil, fmt.Errorf("alter-column %s has no before column", change.Table)
			}
			if strings.TrimSpace(change.Column.Default) != strings.TrimSpace(change.BeforeColumn.Default) {
				return nil, fmt.Errorf("postgres migration SQL rendering does not support column default change on %s.%s", change.Table, change.Column.Name)
			}
			out = append(out, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", quoteQualifiedName(change.Table), quoteIdentifier(change.Column.Name), change.Column.PhysicalType))
			if change.Column.Nullable {
				out = append(out, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;", quoteQualifiedName(change.Table), quoteIdentifier(change.Column.Name)))
			} else {
				out = append(out, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;", quoteQualifiedName(change.Table), quoteIdentifier(change.Column.Name)))
			}
		case schemadiff.ChangePrimaryKey:
			return nil, fmt.Errorf("postgres migration SQL rendering does not support primary-key change on %s until constraint identity is realizable", change.Table)
		case schemadiff.AddIndex:
			if change.Index == nil {
				return nil, fmt.Errorf("add-index %s has no index", change.Table)
			}
			stmt, err := renderCreateIndex(change.Table, change.Index)
			if err != nil {
				return nil, fmt.Errorf("add-index %s.%s: %w", change.Table, change.Index.Name, err)
			}
			out = append(out, stmt)
		case schemadiff.DropIndex:
			if change.Index == nil {
				return nil, fmt.Errorf("drop-index %s has no index", change.Table)
			}
			name, err := qualifiedIndexName(change.Table, change.Index.Name)
			if err != nil {
				return nil, fmt.Errorf("drop-index %s.%s: %w", change.Table, change.Index.Name, err)
			}
			out = append(out, "DROP INDEX "+name+";")
		case schemadiff.IndexStateDrift:
			return nil, fmt.Errorf("index-state-drift on %s requires an operational recovery decision; migration SQL is not generated automatically", change.Table)
		default:
			return nil, fmt.Errorf("postgres migration SQL rendering does not support schema change kind %q on %s", change.Kind, change.Table)
		}
	}
	return out, nil
}

func renderCreateIndex(table string, index *physical.Index) (string, error) {
	name := strings.TrimSpace(index.Name)
	if name == "" {
		return "", fmt.Errorf("index name is empty")
	}
	if len(index.Keys) == 0 {
		return "", fmt.Errorf("index has no keys")
	}

	method := strings.TrimSpace(index.Method)
	if method == "" {
		method = "btree"
	}

	keys := make([]string, 0, len(index.Keys))
	for position, key := range index.Keys {
		rendered, err := renderIndexKey(key, method)
		if err != nil {
			return "", fmt.Errorf("key %d: %w", position+1, err)
		}
		keys = append(keys, rendered)
	}

	var b strings.Builder
	b.WriteString("CREATE ")
	if index.Unique {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("INDEX ")
	b.WriteString(quoteIdentifier(name))
	b.WriteString(" ON ")
	b.WriteString(quoteQualifiedName(table))
	if !strings.EqualFold(method, "btree") {
		if strings.Contains(method, ".") {
			return "", fmt.Errorf("index method %q must be an unqualified PostgreSQL access method name", method)
		}
		b.WriteString(" USING ")
		b.WriteString(quoteIdentifier(method))
	}
	b.WriteString(" (")
	b.WriteString(strings.Join(keys, ", "))
	b.WriteString(")")

	if len(index.IncludedColumns) > 0 {
		b.WriteString(" INCLUDE (")
		b.WriteString(joinQuotedIdentifiers(index.IncludedColumns))
		b.WriteString(")")
	}

	switch index.UniqueNullTreatment {
	case "", physical.UniqueNullTreatmentDistinct:
		// DISTINCT is PostgreSQL's default and need not be emitted.
	case physical.UniqueNullTreatmentNotDistinct:
		if !index.Unique {
			return "", fmt.Errorf("NULLS NOT DISTINCT is valid only for a unique index")
		}
		b.WriteString(" NULLS NOT DISTINCT")
	default:
		return "", fmt.Errorf("unknown unique NULL treatment %q", index.UniqueNullTreatment)
	}

	if index.Predicate != nil {
		source, err := renderProviderExpression(index.Predicate, "predicate")
		if err != nil {
			return "", err
		}
		b.WriteString(" WHERE (")
		b.WriteString(source)
		b.WriteString(")")
	}
	b.WriteString(";")
	return b.String(), nil
}

func renderIndexKey(key physical.IndexKey, method string) (string, error) {
	hasColumn := strings.TrimSpace(key.Column) != ""
	hasExpression := key.Expression != nil
	if hasColumn == hasExpression {
		return "", fmt.Errorf("exactly one column or expression is required")
	}

	var b strings.Builder
	if hasColumn {
		b.WriteString(quoteIdentifier(strings.TrimSpace(key.Column)))
	} else {
		source, err := renderProviderExpression(key.Expression, "index key expression")
		if err != nil {
			return "", err
		}
		b.WriteString("(")
		b.WriteString(source)
		b.WriteString(")")
	}

	if collation := strings.TrimSpace(key.Collation); collation != "" {
		quoted, err := quoteProviderIdentifier(collation)
		if err != nil {
			return "", fmt.Errorf("collation: %w", err)
		}
		b.WriteString(" COLLATE ")
		b.WriteString(quoted)
	}
	if opclass := strings.TrimSpace(key.OperatorClass); opclass != "" {
		quoted, err := quoteProviderIdentifier(opclass)
		if err != nil {
			return "", fmt.Errorf("operator class: %w", err)
		}
		b.WriteString(" ")
		b.WriteString(quoted)
	}

	if !strings.EqualFold(method, "btree") && (key.Order != physical.IndexOrderDefault || key.NullsOrder != physical.IndexNullsOrderDefault) {
		return "", fmt.Errorf("sort direction and NULL ordering are supported only for PostgreSQL B-tree indexes")
	}
	switch key.Order {
	case "", physical.IndexOrderDefault:
	case physical.IndexOrderAsc:
		b.WriteString(" ASC")
	case physical.IndexOrderDesc:
		b.WriteString(" DESC")
	default:
		return "", fmt.Errorf("unknown index order %q", key.Order)
	}
	switch key.NullsOrder {
	case "", physical.IndexNullsOrderDefault:
	case physical.IndexNullsOrderFirst:
		b.WriteString(" NULLS FIRST")
	case physical.IndexNullsOrderLast:
		b.WriteString(" NULLS LAST")
	default:
		return "", fmt.Errorf("unknown index NULL ordering %q", key.NullsOrder)
	}
	return b.String(), nil
}

func renderProviderExpression(expression *physical.ProviderExpression, field string) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("%s is missing", field)
	}
	if !strings.EqualFold(strings.TrimSpace(expression.Provider), string(environment.ProviderPostgres)) {
		return "", fmt.Errorf("%s provider %q is not PostgreSQL", field, expression.Provider)
	}
	if !strings.EqualFold(strings.TrimSpace(expression.Dialect), "postgresql") {
		return "", fmt.Errorf("%s dialect %q is not postgresql", field, expression.Dialect)
	}
	source := strings.TrimSpace(expression.Source)
	if source == "" {
		return "", fmt.Errorf("%s source is empty", field)
	}
	switch expression.Representation {
	case "", physical.ProviderExpressionAuthored:
	case physical.ProviderExpressionProviderNormalized:
		if expression.Normalization == nil || strings.TrimSpace(expression.Normalization.Authority) == "" {
			return "", fmt.Errorf("%s provider-normalized source has no normalization authority", field)
		}
	default:
		return "", fmt.Errorf("%s has unknown representation %q", field, expression.Representation)
	}
	return source, nil
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func quoteQualifiedName(value string) string {
	parts := strings.Split(value, ".")
	for i := range parts {
		parts[i] = quoteIdentifier(parts[i])
	}
	return strings.Join(parts, ".")
}

func quoteProviderIdentifier(value string) (string, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) < 1 || len(parts) > 2 {
		return "", fmt.Errorf("%q must be an unqualified or schema-qualified identifier", value)
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return "", fmt.Errorf("%q contains an empty identifier component", value)
		}
		parts[i] = quoteIdentifier(parts[i])
	}
	return strings.Join(parts, "."), nil
}

func joinQuotedIdentifiers(values []string) string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = quoteIdentifier(value)
	}
	return strings.Join(out, ", ")
}

func qualifiedIndexName(table, index string) (string, error) {
	indexName := strings.TrimSpace(index)
	if indexName == "" {
		return "", fmt.Errorf("index name is empty")
	}

	parts := strings.Split(strings.TrimSpace(table), ".")
	switch len(parts) {
	case 1:
		if strings.TrimSpace(parts[0]) == "" {
			return "", fmt.Errorf("table name is empty")
		}
		return "", fmt.Errorf("table name %q is not schema-qualified; refusing to render DROP INDEX without a deterministic schema", table)
	case 2:
		if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("table name %q contains an empty identifier component", table)
		}
		return quoteIdentifier(strings.TrimSpace(parts[0])) + "." + quoteIdentifier(indexName), nil
	default:
		return "", fmt.Errorf("table name %q cannot determine PostgreSQL index schema", table)
	}
}
