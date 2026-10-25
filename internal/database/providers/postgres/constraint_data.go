package postgres

import (
	"context"
	"fmt"
	"strings"

	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

// Adding a primary key, a unique constraint or a foreign key to a table that
// already exists can fail on, or be blocked by, the rows already stored. The
// PostgreSQL provider renders such an addition only after a provider-owned
// observation has counted the rows that would violate it (#1803 P4). The
// observation carries counts only, never the stored values.

type ConstraintDataCheckKind string

const (
	ConstraintDataPrimaryKey ConstraintDataCheckKind = "primary-key"
	ConstraintDataUnique     ConstraintDataCheckKind = "unique-constraint"
	ConstraintDataForeignKey ConstraintDataCheckKind = "foreign-key"
)

// ConstraintDataCheck is the observed existing-data fact for one constraint
// addition to an existing table. Observed=false means the counts are unknown:
// they are never read as zero.
type ConstraintDataCheck struct {
	Kind              ConstraintDataCheckKind `json:"kind"`
	Table             string                  `json:"table"`
	Columns           []string                `json:"columns"`
	ReferencedTable   string                  `json:"referencedTable,omitempty"`
	ReferencedColumns []string                `json:"referencedColumns,omitempty"`
	// ReferencedTableCreated records that the referenced table is created by the
	// same diff and is therefore empty when the foreign key is added.
	ReferencedTableCreated bool   `json:"referencedTableCreated,omitempty"`
	Observed               bool   `json:"observed"`
	Reason                 string `json:"reason,omitempty"`
	// DuplicateKeys is the number of non-NULL key values stored more than once;
	// DuplicateRows is the number of rows carrying those values.
	DuplicateKeys int64 `json:"duplicateKeys"`
	DuplicateRows int64 `json:"duplicateRows"`
	// NullKeyRows counts rows with a NULL in any primary-key column. It is not
	// observed for unique constraints, where NULLs are distinct.
	NullKeyRows int64 `json:"nullKeyRows"`
	// MissingReferenceRows counts rows whose foreign-key columns are all
	// non-NULL and have no referenced row (MATCH SIMPLE).
	MissingReferenceRows int64 `json:"missingReferenceRows"`
}

func (c ConstraintDataCheck) identity() string {
	return strings.Join([]string{
		string(c.Kind), c.Table, strings.Join(c.Columns, "\x1f"),
		c.ReferencedTable, strings.Join(c.ReferencedColumns, "\x1f"),
	}, "\x1e")
}

// KeyViolationCounts are the counts for a primary key or unique constraint.
type KeyViolationCounts struct {
	DuplicateKeys int64
	DuplicateRows int64
	NullKeyRows   int64
}

// ConstraintDataReader is an optional extension to CatalogReader that counts
// existing rows violating a constraint about to be added. Implementations
// return counts only.
type ConstraintDataReader interface {
	CountKeyViolations(ctx context.Context, table RelationIdentity, columns []string, countNullKeys bool) (KeyViolationCounts, error)
	CountMissingReferences(ctx context.Context, table RelationIdentity, columns []string, referenced RelationIdentity, referencedColumns []string, referencedEmpty bool) (int64, error)
}

// existingTableConstraintAdditions returns the constraint additions in diff
// that target a table not created by the same diff, as unobserved checks. A
// primary-key change is an addition only when the table has no primary key;
// replacing or dropping one (P5) is not covered here.
func existingTableConstraintAdditions(diff schemadiff.Diff) ([]ConstraintDataCheck, error) {
	created := createdTablesOf(diff)
	var out []ConstraintDataCheck
	for _, change := range diff.Changes {
		check, ok, err := constraintAdditionCheck(change, created)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, check)
		}
	}
	return out, nil
}

// constraintAdditionCheck returns the unobserved check that change requires,
// and false when change is not a constraint addition to an existing table.
func constraintAdditionCheck(change schemadiff.Change, created map[string]bool) (ConstraintDataCheck, bool, error) {
	if created[change.Table] {
		return ConstraintDataCheck{}, false, nil
	}
	switch change.Kind {
	case schemadiff.ChangePrimaryKey:
		if len(change.BeforePrimaryKey) != 0 || len(change.PrimaryKey) == 0 {
			return ConstraintDataCheck{}, false, nil
		}
		return ConstraintDataCheck{Kind: ConstraintDataPrimaryKey, Table: change.Table, Columns: append([]string(nil), change.PrimaryKey...)}, true, nil
	case schemadiff.AddUniqueConstraint:
		if change.UniqueConstraint == nil || len(change.UniqueConstraint.Columns) == 0 {
			return ConstraintDataCheck{}, false, fmt.Errorf("add-unique-constraint %s has no columns", change.Table)
		}
		return ConstraintDataCheck{Kind: ConstraintDataUnique, Table: change.Table, Columns: append([]string(nil), change.UniqueConstraint.Columns...)}, true, nil
	case schemadiff.AddForeignKey:
		fk := change.ForeignKey
		if fk == nil || len(fk.Columns) == 0 || len(fk.ReferencedColumns) != len(fk.Columns) || strings.TrimSpace(fk.ReferencedTable) == "" {
			return ConstraintDataCheck{}, false, fmt.Errorf("add-foreign-key %s has an incomplete definition", change.Table)
		}
		if strings.TrimSpace(fk.ReferencedSchema) == "" {
			return ConstraintDataCheck{}, false, fmt.Errorf("add-foreign-key %s has no referenced schema; the referenced table is not resolved by search_path", change.Table)
		}
		referenced := fk.ReferencedSchema + "." + fk.ReferencedTable
		return ConstraintDataCheck{
			Kind: ConstraintDataForeignKey, Table: change.Table, Columns: append([]string(nil), fk.Columns...),
			ReferencedTable: referenced, ReferencedColumns: append([]string(nil), fk.ReferencedColumns...),
			ReferencedTableCreated: created[referenced],
		}, true, nil
	}
	return ConstraintDataCheck{}, false, nil
}

func createdTablesOf(diff schemadiff.Diff) map[string]bool {
	created := map[string]bool{}
	for _, change := range diff.Changes {
		if change.Kind == schemadiff.CreateTable {
			created[change.Table] = true
		}
	}
	return created
}

// ObserveConstraintData counts, on the configured database, the existing rows
// that would violate each constraint the diff adds to an existing table, and
// returns observation with ConstraintData replaced by the result. A catalog
// reader that cannot count, or a count that fails (for example without the
// privilege to read the table), leaves that check unobserved with the reason.
// Only a malformed diff is returned as an error.
func (p Provider) ObserveConstraintData(ctx context.Context, observation SchemaObservation, diff schemadiff.Diff) (SchemaObservation, error) {
	checks, err := existingTableConstraintAdditions(diff)
	if err != nil {
		return observation, err
	}
	reader, ok := p.Catalog.(ConstraintDataReader)
	for i := range checks {
		check := &checks[i]
		if !ok {
			check.Reason = "the catalog reader cannot count existing rows"
			continue
		}
		table, err := relationIdentityFromQualifiedName(check.Table)
		if err != nil {
			return observation, fmt.Errorf("%s %s: %w", check.Kind, check.Table, err)
		}
		switch check.Kind {
		case ConstraintDataPrimaryKey, ConstraintDataUnique:
			counts, err := reader.CountKeyViolations(ctx, table, check.Columns, check.Kind == ConstraintDataPrimaryKey)
			if err != nil {
				check.Reason = "counting existing rows failed: " + err.Error()
				continue
			}
			check.DuplicateKeys, check.DuplicateRows, check.NullKeyRows = counts.DuplicateKeys, counts.DuplicateRows, counts.NullKeyRows
		case ConstraintDataForeignKey:
			referenced, err := relationIdentityFromQualifiedName(check.ReferencedTable)
			if err != nil {
				return observation, fmt.Errorf("%s %s: %w", check.Kind, check.Table, err)
			}
			missing, err := reader.CountMissingReferences(ctx, table, check.Columns, referenced, check.ReferencedColumns, check.ReferencedTableCreated)
			if err != nil {
				check.Reason = "counting existing rows failed: " + err.Error()
				continue
			}
			check.MissingReferenceRows = missing
		}
		check.Observed = true
	}
	observation.ConstraintData = checks
	return observation, nil
}

type ConstraintDataReadinessReason string

const (
	// ConstraintDataUnobserved: the existing data was not counted. The addition
	// is unresolved; it is not treated as having no violating rows.
	ConstraintDataUnobserved ConstraintDataReadinessReason = "existing-data-unobserved"
	// ConstraintDataViolations: rows that violate the constraint exist. No
	// declared handling (fill, fix, exception) exists yet, so it is refused.
	ConstraintDataViolations ConstraintDataReadinessReason = "existing-data-violations"
	// ConstraintDataNullablePrimaryKey: a primary-key column stays nullable in
	// the desired schema; PostgreSQL would make it NOT NULL and drift.
	ConstraintDataNullablePrimaryKey ConstraintDataReadinessReason = "primary-key-column-nullable"
)

type ConstraintAdditionReadiness struct {
	Check      ConstraintDataCheck           `json:"check"`
	Executable bool                          `json:"executable"`
	Reason     ConstraintDataReadinessReason `json:"reason,omitempty"`
	Detail     string                        `json:"detail,omitempty"`
}

type ConstraintDataReadiness struct {
	Executable bool                          `json:"executable"`
	Additions  []ConstraintAdditionReadiness `json:"additions,omitempty"`
}

type ConstraintDataReadinessError struct {
	Readiness ConstraintDataReadiness
}

func (e *ConstraintDataReadinessError) Error() string {
	var blocked []string
	for _, addition := range e.Readiness.Additions {
		if addition.Executable {
			continue
		}
		c := addition.Check
		blocked = append(blocked, fmt.Sprintf("%s %s (%s): %s: %s", c.Kind, c.Table, strings.Join(c.Columns, ", "), addition.Reason, addition.Detail))
	}
	return fmt.Sprintf("postgres existing-table constraint readiness blocked %d addition(s): %s", len(blocked), strings.Join(blocked, "; "))
}

// AssessConstraintDataReadiness decides, for every constraint the diff adds to
// an existing table, whether the observation allows rendering it. An addition
// is executable only when its existing data was observed and every violation
// count is zero.
func (o SchemaObservation) AssessConstraintDataReadiness(diff schemadiff.Diff) (ConstraintDataReadiness, error) {
	readiness := ConstraintDataReadiness{Executable: true}
	required, err := existingTableConstraintAdditions(diff)
	if err != nil {
		return ConstraintDataReadiness{}, err
	}
	if len(required) == 0 {
		return readiness, nil
	}
	if string(o.Snapshot.Database) != diff.Database {
		return ConstraintDataReadiness{}, fmt.Errorf("postgres observation database %s differs from diff database %s", o.Snapshot.Database, diff.Database)
	}
	observed := make(map[string]ConstraintDataCheck, len(o.ConstraintData))
	for _, check := range o.ConstraintData {
		observed[check.identity()] = check
	}
	for _, want := range required {
		assessment := ConstraintAdditionReadiness{Check: want, Executable: true}
		check, found := observed[want.identity()]
		switch {
		case !found:
			assessment.Executable = false
			assessment.Reason = ConstraintDataUnobserved
			assessment.Detail = "existing rows were not counted; unresolved, not treated as zero"
		case !check.Observed || check.ReferencedTableCreated != want.ReferencedTableCreated:
			assessment.Check = check
			assessment.Executable = false
			assessment.Reason = ConstraintDataUnobserved
			assessment.Detail = "existing rows were not counted; unresolved, not treated as zero"
			if check.Observed {
				assessment.Detail = "the observation was taken for a different diff; unresolved"
			} else if check.Reason != "" {
				assessment.Detail += " (" + check.Reason + ")"
			}
		default:
			assessment.Check = check
			if violations := violationSummary(check); violations != "" {
				assessment.Executable = false
				assessment.Reason = ConstraintDataViolations
				assessment.Detail = violations + "; no declared handling exists, so the constraint is not added"
			}
		}
		if assessment.Executable && want.Kind == ConstraintDataPrimaryKey {
			if column, nullable, err := o.nullablePrimaryKeyColumn(diff, want); err != nil {
				return ConstraintDataReadiness{}, err
			} else if nullable {
				assessment.Executable = false
				assessment.Reason = ConstraintDataNullablePrimaryKey
				assessment.Detail = fmt.Sprintf("column %s stays nullable in the desired schema; PostgreSQL would make it NOT NULL and the declared schema would drift", column)
			}
		}
		if !assessment.Executable {
			readiness.Executable = false
		}
		readiness.Additions = append(readiness.Additions, assessment)
	}
	return readiness, nil
}

func violationSummary(c ConstraintDataCheck) string {
	var parts []string
	switch c.Kind {
	case ConstraintDataPrimaryKey, ConstraintDataUnique:
		if c.DuplicateRows > 0 || c.DuplicateKeys > 0 {
			parts = append(parts, fmt.Sprintf("%d row(s) share %d duplicated key value(s)", c.DuplicateRows, c.DuplicateKeys))
		}
		if c.Kind == ConstraintDataPrimaryKey && c.NullKeyRows > 0 {
			parts = append(parts, fmt.Sprintf("%d row(s) have NULL in a key column", c.NullKeyRows))
		}
	case ConstraintDataForeignKey:
		if c.MissingReferenceRows > 0 {
			parts = append(parts, fmt.Sprintf("%d row(s) have no referenced row", c.MissingReferenceRows))
		}
	}
	return strings.Join(parts, ", ")
}

// nullablePrimaryKeyColumn reports a primary-key column whose desired state is
// nullable: the diff's alter-column target when there is one, otherwise the
// observed column, which the diff leaves unchanged.
func (o SchemaObservation) nullablePrimaryKeyColumn(diff schemadiff.Diff, check ConstraintDataCheck) (string, bool, error) {
	desired := map[string]physical.Column{}
	for _, change := range diff.Changes {
		if change.Table == check.Table && change.Kind == schemadiff.AlterColumn && change.Column != nil {
			desired[change.Column.Name] = *change.Column
		}
	}
	var actual *physical.Table
	for i := range o.Snapshot.Tables {
		if o.Snapshot.Tables[i].QualifiedName() == check.Table {
			actual = &o.Snapshot.Tables[i]
			break
		}
	}
	for _, name := range check.Columns {
		if column, ok := desired[name]; ok {
			if column.Nullable {
				return name, true, nil
			}
			continue
		}
		found := false
		if actual != nil {
			for _, column := range actual.Columns {
				if column.Name == name {
					found = true
					if column.Nullable {
						return name, true, nil
					}
				}
			}
		}
		if !found {
			return "", false, fmt.Errorf("primary key column %s of existing table %s is not in the observed schema", name, check.Table)
		}
	}
	return "", false, nil
}

// keyViolationQuery counts, without returning any stored value, duplicated
// non-NULL key values, the rows carrying them and, for a primary key, rows
// with a NULL key column.
func keyViolationQuery(table RelationIdentity, columns []string, countNullKeys bool) string {
	quoted := make([]string, len(columns))
	notNull := make([]string, len(columns))
	isNull := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quoteIdentifier(column)
		notNull[i] = quoted[i] + " IS NOT NULL"
		isNull[i] = quoted[i] + " IS NULL"
	}
	relation := quoteIdentifier(table.Schema) + "." + quoteIdentifier(table.Name)
	nullRows := "0::bigint"
	if countNullKeys {
		nullRows = fmt.Sprintf("(SELECT count(*) FROM %s WHERE %s)", relation, strings.Join(isNull, " OR "))
	}
	return fmt.Sprintf("SELECT count(*), COALESCE(sum(d.n), 0)::bigint, %s FROM (SELECT count(*) AS n FROM %s WHERE %s GROUP BY %s HAVING count(*) > 1) AS d",
		nullRows, relation, strings.Join(notNull, " AND "), strings.Join(quoted, ", "))
}

// missingReferenceQuery counts rows whose foreign-key columns are all non-NULL
// and match no referenced row. When the referenced table is created by the
// same diff it is empty, and every such row is missing its reference.
func missingReferenceQuery(table RelationIdentity, columns []string, referenced RelationIdentity, referencedColumns []string, referencedEmpty bool) (string, error) {
	if len(columns) == 0 || len(columns) != len(referencedColumns) {
		return "", fmt.Errorf("foreign key columns and referenced columns differ in number")
	}
	notNull := make([]string, len(columns))
	match := make([]string, len(columns))
	for i, column := range columns {
		notNull[i] = "child." + quoteIdentifier(column) + " IS NOT NULL"
		match[i] = "parent." + quoteIdentifier(referencedColumns[i]) + " = child." + quoteIdentifier(column)
	}
	relation := quoteIdentifier(table.Schema) + "." + quoteIdentifier(table.Name) + " AS child"
	if referencedEmpty {
		return fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", relation, strings.Join(notNull, " AND ")), nil
	}
	return fmt.Sprintf("SELECT count(*) FROM %s WHERE %s AND NOT EXISTS (SELECT 1 FROM %s AS parent WHERE %s)",
		relation, strings.Join(notNull, " AND "),
		quoteIdentifier(referenced.Schema)+"."+quoteIdentifier(referenced.Name), strings.Join(match, " AND ")), nil
}

func (r SQLCatalogReader) CountKeyViolations(ctx context.Context, table RelationIdentity, columns []string, countNullKeys bool) (KeyViolationCounts, error) {
	if len(columns) == 0 {
		return KeyViolationCounts{}, fmt.Errorf("key has no columns")
	}
	var counts KeyViolationCounts
	err := r.DB.QueryRowContext(ctx, keyViolationQuery(table, columns, countNullKeys)).Scan(&counts.DuplicateKeys, &counts.DuplicateRows, &counts.NullKeyRows)
	return counts, err
}

func (r SQLCatalogReader) CountMissingReferences(ctx context.Context, table RelationIdentity, columns []string, referenced RelationIdentity, referencedColumns []string, referencedEmpty bool) (int64, error) {
	query, err := missingReferenceQuery(table, columns, referenced, referencedColumns, referencedEmpty)
	if err != nil {
		return 0, err
	}
	var missing int64
	err = r.DB.QueryRowContext(ctx, query).Scan(&missing)
	return missing, err
}
