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
	return renderResolvedMigrationSQL(diff, nil)
}

func validateResolvedMigrationDiff(diff schemadiff.Diff) error {
	if len(diff.Unresolved) > 0 {
		return fmt.Errorf("migration SQL rendering requires a fully resolved schema diff: %d unresolved comparison(s)", len(diff.Unresolved))
	}
	for _, change := range diff.Changes {
		switch change.Kind {
		case schemadiff.AddUniqueConstraint:
			if change.UniqueConstraint != nil {
				if err := validateDeclaredConstraintName("unique constraint", change.Table, change.UniqueConstraint.Name); err != nil {
					return err
				}
			}
		case schemadiff.AddForeignKey:
			if change.ForeignKey != nil {
				if err := validateDeclaredConstraintName("foreign key", change.Table, change.ForeignKey.Name); err != nil {
					return err
				}
			}
		case schemadiff.RenameConstraint:
			name := ""
			if change.UniqueConstraint != nil {
				name = change.UniqueConstraint.Name
			}
			if change.ForeignKey != nil {
				name = change.ForeignKey.Name
			}
			if err := validateDeclaredConstraintName("renamed constraint", change.Table, name); err != nil {
				return err
			}
		}
	}
	return nil
}

// maxPostgresIdentifierBytes is PostgreSQL's NAMEDATALEN-1. Longer identifiers
// are silently truncated by PostgreSQL, which would make a declared name
// differ from the observed one and break convergence, so the provider refuses
// them instead of truncating.
const maxPostgresIdentifierBytes = 63

func validateDeclaredConstraintName(kind, table, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if size := len([]byte(name)); size > maxPostgresIdentifierBytes {
		return fmt.Errorf("%s on %s: declared name %q is %d bytes; PostgreSQL truncates identifiers to %d bytes, so it is refused rather than silently truncated",
			kind, table, name, size, maxPostgresIdentifierBytes)
	}
	return nil
}

// renderResolvedMigrationSQL is package-private because it can emit destructive
// DROP TABLE after its caller has established the provider-specific safety
// preconditions. Exported callers must go through RenderMigrationSQL or
// RenderMigrationSQLWithObservation.
//
// checked holds the identities of constraint additions to existing tables
// whose existing data has been observed with no violating row; any other such
// addition is refused.
func renderResolvedMigrationSQL(diff schemadiff.Diff, checked map[string]ConstraintDataCheck) ([]string, error) {
	var out []string
	// A primary key on a newly created table is added unnamed, after the table's
	// columns exist, so PostgreSQL chooses the name and the constraint is observed
	// by its table rather than by an assumed name. An existing table without a
	// primary key gets one only after its existing data is checked (below);
	// replacing an existing primary key additionally uses its observed constraint
	// name (#1803 P5); removing one stays unsupported.
	var createdWithPrimaryKey []schemadiff.Change
	addedColumns := map[string]map[string]physical.Column{}
	// Unique constraints and foreign keys on tables created in this same diff
	// need no check: those tables are empty. Adding a primary key, a unique
	// constraint or a foreign key to an existing table is rendered only when
	// checked says its existing data was counted with no violating row
	// (#1803 P4). They follow the primary keys, and foreign keys come last so
	// every table in the diff exists by then.
	createdTables := createdTablesOf(diff)
	existingAdditionChecked := func(change schemadiff.Change) (ConstraintDataCheck, error) {
		check, ok, err := constraintAdditionCheck(change, createdTables)
		if err != nil || !ok {
			return ConstraintDataCheck{}, err
		}
		observed, found := checked[check.identity()]
		if !found {
			return ConstraintDataCheck{}, fmt.Errorf("postgres migration SQL rendering does not render schema change kind %q on existing table %s without an existing-data observation: unresolved; use RenderMigrationSQLWithObservation with ObserveConstraintData", change.Kind, change.Table)
		}
		return observed, nil
	}
	droppedColumns := map[string]map[string]bool{}
	for _, change := range diff.Changes {
		if change.Kind == schemadiff.DropColumn && change.BeforeColumn != nil {
			if droppedColumns[change.Table] == nil {
				droppedColumns[change.Table] = map[string]bool{}
			}
			droppedColumns[change.Table][change.BeforeColumn.Name] = true
		}
	}
	var existingPrimaryKeyStatements, uniqueStatements, foreignKeyStatements, renameStatements []string
	if err := refuseRenameTargetConflicts(diff); err != nil {
		return nil, err
	}
	for _, change := range diff.Changes {
		switch change.Kind {
		case schemadiff.CreateTable:
			if len(change.PrimaryKey) > 0 {
				createdWithPrimaryKey = append(createdWithPrimaryKey, change)
			}
			// Columns are emitted by following AddColumn changes to retain one common change model.
			out = append(out, fmt.Sprintf("CREATE TABLE %s ();", quoteQualifiedName(change.Table)))
		case schemadiff.DropTable:
			out = append(out, fmt.Sprintf("DROP TABLE %s;", quoteQualifiedName(change.Table)))
		case schemadiff.AddColumn:
			if change.Column == nil {
				return nil, fmt.Errorf("add-column %s has no column", change.Table)
			}
			if addedColumns[change.Table] == nil {
				addedColumns[change.Table] = map[string]physical.Column{}
			}
			addedColumns[change.Table][change.Column.Name] = *change.Column
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
		case schemadiff.RenameConstraint:
			// Only a constraint of an existing table, renamed from the name
			// observed in the catalog, and only after the observation says the
			// name still exists and the new one is free (#1803). The primary
			// key is unnamed in the model and is never renamed.
			check, err := existingAdditionChecked(change)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(check.BeforeName) == "" || strings.TrimSpace(check.NewName) == "" {
				return nil, fmt.Errorf("rename-constraint %s has no observed or declared name", change.Table)
			}
			renameStatements = append(renameStatements, fmt.Sprintf("ALTER TABLE %s RENAME CONSTRAINT %s TO %s;",
				quoteQualifiedName(change.Table), quoteIdentifier(check.BeforeName), quoteIdentifier(check.NewName)))
		case schemadiff.AddUniqueConstraint:
			if _, err := existingAdditionChecked(change); err != nil {
				return nil, err
			}
			if change.UniqueConstraint == nil || len(change.UniqueConstraint.Columns) == 0 {
				return nil, fmt.Errorf("add-unique-constraint %s has no columns", change.Table)
			}
			// Names are decided by the realization policy (#1803); an unnamed
			// constraint here means the name was not decided, so it is refused
			// instead of being left to PostgreSQL.
			if strings.TrimSpace(change.UniqueConstraint.Name) == "" {
				return nil, fmt.Errorf("add-unique-constraint %s has no constraint name; the realization policy must decide it", change.Table)
			}
			uniqueStatements = append(uniqueStatements, fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);",
				quoteQualifiedName(change.Table), quoteIdentifier(change.UniqueConstraint.Name), joinQuotedIdentifiers(change.UniqueConstraint.Columns)))
		case schemadiff.AddForeignKey:
			if _, err := existingAdditionChecked(change); err != nil {
				return nil, err
			}
			fk := change.ForeignKey
			if fk == nil || len(fk.Columns) == 0 || len(fk.ReferencedColumns) != len(fk.Columns) || strings.TrimSpace(fk.ReferencedTable) == "" {
				return nil, fmt.Errorf("add-foreign-key %s has an incomplete definition", change.Table)
			}
			if strings.TrimSpace(fk.ReferencedSchema) == "" {
				return nil, fmt.Errorf("add-foreign-key %s has no referenced schema; the referenced table is not resolved by search_path", change.Table)
			}
			if strings.TrimSpace(fk.Name) == "" {
				return nil, fmt.Errorf("add-foreign-key %s has no constraint name; the realization policy must decide it", change.Table)
			}
			foreignKeyStatements = append(foreignKeyStatements, fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s);",
				quoteQualifiedName(change.Table), quoteIdentifier(fk.Name), joinQuotedIdentifiers(fk.Columns), quoteQualifiedName(fk.ReferencedSchema+"."+fk.ReferencedTable), joinQuotedIdentifiers(fk.ReferencedColumns)))
		case schemadiff.ChangePrimaryKey:
			// Adding a primary key to an existing table that has none is P4;
			// replacing one is P5; removing one drops the observed constraint.
			// In every case the observed constraint name is used, never a name
			// derived from the table name.
			if len(change.PrimaryKey) == 0 {
				if len(change.BeforePrimaryKey) == 0 {
					return nil, fmt.Errorf("primary-key change on %s removes the primary key but names no current key", change.Table)
				}
				observed, err := existingAdditionChecked(change)
				if err != nil {
					return nil, err
				}
				if strings.TrimSpace(observed.PrimaryKeyName) == "" {
					return nil, fmt.Errorf("primary-key removal on %s has no observed constraint name; the name is never assumed from the table name", change.Table)
				}
				for _, name := range change.BeforePrimaryKey {
					if droppedColumns[change.Table][name] {
						return nil, fmt.Errorf("primary-key removal on %s: column %s of the current primary key is dropped by the same diff, which removes the key implicitly", change.Table, name)
					}
				}
				existingPrimaryKeyStatements = append(existingPrimaryKeyStatements, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", quoteQualifiedName(change.Table), quoteIdentifier(observed.PrimaryKeyName)))
				continue
			}
			observed, err := existingAdditionChecked(change)
			if err != nil {
				return nil, err
			}
			if len(change.BeforePrimaryKey) != 0 {
				if strings.TrimSpace(observed.PrimaryKeyName) == "" {
					return nil, fmt.Errorf("primary-key change on %s has no observed constraint name; the name is never assumed from the table name", change.Table)
				}
				for _, name := range change.BeforePrimaryKey {
					if droppedColumns[change.Table][name] {
						return nil, fmt.Errorf("primary-key change on %s: column %s of the current primary key is dropped by the same diff, which removes the key implicitly", change.Table, name)
					}
				}
				// One ALTER TABLE with two actions is atomic: a failing new key
				// (for example a duplicate written after the count) leaves the old
				// primary key in place. The new key is added unnamed (design 4.2 C).
				existingPrimaryKeyStatements = append(existingPrimaryKeyStatements, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s, ADD PRIMARY KEY (%s);", quoteQualifiedName(change.Table), quoteIdentifier(observed.PrimaryKeyName), joinQuotedIdentifiers(change.PrimaryKey)))
				continue
			}
			existingPrimaryKeyStatements = append(existingPrimaryKeyStatements, fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s);", quoteQualifiedName(change.Table), joinQuotedIdentifiers(change.PrimaryKey)))
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
	for _, created := range createdWithPrimaryKey {
		columns := addedColumns[created.Table]
		for _, name := range created.PrimaryKey {
			column, ok := columns[name]
			if !ok {
				return nil, fmt.Errorf("primary key column %s of newly created table %s is not added by this diff", name, created.Table)
			}
			if column.Nullable {
				return nil, fmt.Errorf("primary key column %s of newly created table %s is nullable; PostgreSQL would make it NOT NULL and the declared schema would drift", name, created.Table)
			}
		}
		out = append(out, fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s);", quoteQualifiedName(created.Table), joinQuotedIdentifiers(created.PrimaryKey)))
	}
	out = append(out, existingPrimaryKeyStatements...)
	out = append(out, renameStatements...)
	out = append(out, uniqueStatements...)
	out = append(out, foreignKeyStatements...)
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

// refuseRenameTargetConflicts refuses a diff in which a rename's new name is
// also the target of another rename or of an addition in the same diff. The
// catalog observation cannot see names that only this diff introduces. A
// rename onto a name that another rename of the same diff vacates is refused
// too: the observation finds the name taken.
func refuseRenameTargetConflicts(diff schemadiff.Diff) error {
	// Unique constraints share the schema's relation namespace; a foreign key
	// only meets the constraints of its own table.
	owner := map[string]string{}
	claim := func(change schemadiff.Change, kind, name string) error {
		schema := ""
		if i := strings.LastIndex(change.Table, "."); i >= 0 {
			schema = change.Table[:i]
		}
		key := schema + "\x00" + name
		if kind == "fk" {
			key = "fk\x00" + change.Table + "\x00" + name
		}
		if prior, ok := owner[key]; ok {
			return fmt.Errorf("constraint name %q is claimed twice in one diff (%s on %s and %s)", name, prior, change.Table, change.Kind)
		}
		owner[key] = string(change.Kind) + " on " + change.Table
		return nil
	}
	for _, change := range diff.Changes {
		switch change.Kind {
		case schemadiff.RenameConstraint, schemadiff.AddUniqueConstraint, schemadiff.AddForeignKey:
			switch {
			case change.UniqueConstraint != nil:
				if err := claim(change, "unique", strings.TrimSpace(change.UniqueConstraint.Name)); err != nil {
					return err
				}
			case change.ForeignKey != nil:
				if err := claim(change, "fk", strings.TrimSpace(change.ForeignKey.Name)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
