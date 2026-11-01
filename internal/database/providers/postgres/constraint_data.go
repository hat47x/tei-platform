package postgres

import (
	"context"
	"fmt"
	"slices"
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
	// ConstraintDataPrimaryKeyReplace replaces the primary key of an existing
	// table that already has one (#1803 P5). It additionally carries the
	// observed constraint name and the foreign keys that depend on the old key.
	ConstraintDataPrimaryKeyReplace ConstraintDataCheckKind = "primary-key-replace"
	// ConstraintDataPrimaryKeyRemove removes the primary key of an existing
	// table. It carries the observed name and the referencing foreign keys;
	// removing a key that a foreign key depends on is refused, never cascaded.
	ConstraintDataPrimaryKeyRemove ConstraintDataCheckKind = "primary-key-remove"
	// ConstraintDataRenameConstraint renames a unique constraint or foreign key
	// of an existing table whose declared name differs from the observed one
	// (#1803). It carries no row counts: it records whether the observed name
	// still exists and whether the new name is free.
	ConstraintDataRenameConstraint ConstraintDataCheckKind = "rename-constraint"
)

// ConstraintRenameKind names the constraint family of a rename.
type ConstraintRenameKind string

const (
	ConstraintRenameUnique     ConstraintRenameKind = "unique"
	ConstraintRenameForeignKey ConstraintRenameKind = "foreign-key"
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
	// BeforeColumns is, for a primary-key replacement, the primary key the diff
	// expects to replace. Columns holds the new key.
	BeforeColumns []string `json:"beforeColumns,omitempty"`
	// PrimaryKeyName is the real name of the existing primary key constraint,
	// read from the catalog. It is never derived from the table name.
	PrimaryKeyName string `json:"primaryKeyName,omitempty"`
	// ReferencingForeignKeys are the foreign keys that depend on the existing
	// primary key's index. Dropping the key would fail or require CASCADE.
	ReferencingForeignKeys []ReferencingForeignKey `json:"referencingForeignKeys,omitempty"`
	// RenameKind, BeforeName and NewName identify a constraint rename.
	// SourceFound records that a constraint of that family named BeforeName
	// exists on the table; NameOccupiedBy describes what already holds NewName
	// (empty when it is free).
	RenameKind     ConstraintRenameKind `json:"renameKind,omitempty"`
	BeforeName     string               `json:"beforeName,omitempty"`
	NewName        string               `json:"newName,omitempty"`
	SourceFound    bool                 `json:"sourceFound,omitempty"`
	NameOccupiedBy string               `json:"nameOccupiedBy,omitempty"`
}

func (c ConstraintDataCheck) identity() string {
	return strings.Join([]string{
		string(c.Kind), c.Table, strings.Join(c.Columns, "\x1f"),
		c.ReferencedTable, strings.Join(c.ReferencedColumns, "\x1f"),
		strings.Join(c.BeforeColumns, "\x1f"),
		string(c.RenameKind), c.BeforeName, c.NewName,
	}, "\x1e")
}

// ConstraintRenameFacts are what the catalog says about one rename.
type ConstraintRenameFacts struct {
	SourceFound    bool
	NameOccupiedBy string
}

// ConstraintNameReader is an optional extension to CatalogReader that reads,
// for a constraint rename, whether the observed constraint still exists and
// whether the new name is free in the namespaces PostgreSQL enforces: the
// schema's relations (a unique constraint's index shares them) and the
// constraints of the table.
type ConstraintNameReader interface {
	ConstraintRenameFacts(ctx context.Context, table RelationIdentity, kind ConstraintRenameKind, oldName, newName string) (ConstraintRenameFacts, error)
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
// primary-key change is an addition when the table has no primary key, a
// replacement (P5) when it has one and keeps one, and a removal when the
// desired key is empty and the table has one.
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
		if len(change.PrimaryKey) == 0 {
			if len(change.BeforePrimaryKey) == 0 {
				return ConstraintDataCheck{}, false, fmt.Errorf("change-primary-key %s removes a primary key but names no current key", change.Table)
			}
			return ConstraintDataCheck{Kind: ConstraintDataPrimaryKeyRemove, Table: change.Table,
				BeforeColumns: append([]string(nil), change.BeforePrimaryKey...)}, true, nil
		}
		if len(change.BeforePrimaryKey) != 0 {
			return ConstraintDataCheck{Kind: ConstraintDataPrimaryKeyReplace, Table: change.Table,
				Columns: append([]string(nil), change.PrimaryKey...), BeforeColumns: append([]string(nil), change.BeforePrimaryKey...)}, true, nil
		}
		return ConstraintDataCheck{Kind: ConstraintDataPrimaryKey, Table: change.Table, Columns: append([]string(nil), change.PrimaryKey...)}, true, nil
	case schemadiff.RenameConstraint:
		check := ConstraintDataCheck{Kind: ConstraintDataRenameConstraint, Table: change.Table, BeforeName: strings.TrimSpace(change.BeforeName)}
		switch {
		case change.UniqueConstraint != nil && change.ForeignKey == nil:
			check.RenameKind = ConstraintRenameUnique
			check.Columns = append([]string(nil), change.UniqueConstraint.Columns...)
			check.NewName = strings.TrimSpace(change.UniqueConstraint.Name)
		case change.ForeignKey != nil && change.UniqueConstraint == nil:
			check.RenameKind = ConstraintRenameForeignKey
			check.Columns = append([]string(nil), change.ForeignKey.Columns...)
			check.NewName = strings.TrimSpace(change.ForeignKey.Name)
		default:
			return ConstraintDataCheck{}, false, fmt.Errorf("rename-constraint %s must carry exactly one of a unique constraint or a foreign key", change.Table)
		}
		if check.BeforeName == "" || check.NewName == "" || check.BeforeName == check.NewName {
			return ConstraintDataCheck{}, false, fmt.Errorf("rename-constraint %s needs an observed name and a different declared name", change.Table)
		}
		return check, true, nil
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
		if check.Kind == ConstraintDataRenameConstraint {
			p.observeConstraintRename(ctx, check)
			continue
		}
		if !ok {
			check.Reason = "the catalog reader cannot count existing rows"
			continue
		}
		table, err := relationIdentityFromQualifiedName(check.Table)
		if err != nil {
			return observation, fmt.Errorf("%s %s: %w", check.Kind, check.Table, err)
		}
		switch check.Kind {
		case ConstraintDataPrimaryKeyRemove:
			identityReader, canRead := p.Catalog.(PrimaryKeyIdentityReader)
			if !canRead {
				check.Reason = "the catalog reader cannot read the existing primary key constraint"
				continue
			}
			pk, err := identityReader.PrimaryKeyIdentity(ctx, table)
			if err != nil {
				check.Reason = "reading the existing primary key constraint failed: " + err.Error()
				continue
			}
			if strings.TrimSpace(pk.Name) == "" || !slices.Equal(pk.Columns, check.BeforeColumns) {
				check.Reason = fmt.Sprintf("the observed primary key (%s) differs from the diff's current key (%s)", strings.Join(pk.Columns, ", "), strings.Join(check.BeforeColumns, ", "))
				continue
			}
			check.PrimaryKeyName = pk.Name
			check.ReferencingForeignKeys = pk.ReferencingForeignKeys
			check.Observed = true
			continue
		case ConstraintDataPrimaryKeyReplace:
			identityReader, canRead := p.Catalog.(PrimaryKeyIdentityReader)
			if !canRead {
				check.Reason = "the catalog reader cannot read the existing primary key constraint"
				continue
			}
			pk, err := identityReader.PrimaryKeyIdentity(ctx, table)
			if err != nil {
				check.Reason = "reading the existing primary key constraint failed: " + err.Error()
				continue
			}
			if strings.TrimSpace(pk.Name) == "" || !slices.Equal(pk.Columns, check.BeforeColumns) {
				check.Reason = fmt.Sprintf("the observed primary key (%s) differs from the diff's current key (%s)", strings.Join(pk.Columns, ", "), strings.Join(check.BeforeColumns, ", "))
				continue
			}
			check.PrimaryKeyName = pk.Name
			check.ReferencingForeignKeys = pk.ReferencingForeignKeys
			counts, err := reader.CountKeyViolations(ctx, table, check.Columns, true)
			if err != nil {
				check.Reason = "counting existing rows failed: " + err.Error()
				continue
			}
			check.DuplicateKeys, check.DuplicateRows, check.NullKeyRows = counts.DuplicateKeys, counts.DuplicateRows, counts.NullKeyRows
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

func (p Provider) observeConstraintRename(ctx context.Context, check *ConstraintDataCheck) {
	reader, ok := p.Catalog.(ConstraintNameReader)
	if !ok {
		check.Reason = "the catalog reader cannot read constraint names"
		return
	}
	table, err := relationIdentityFromQualifiedName(check.Table)
	if err != nil {
		check.Reason = err.Error()
		return
	}
	facts, err := reader.ConstraintRenameFacts(ctx, table, check.RenameKind, check.BeforeName, check.NewName)
	if err != nil {
		check.Reason = "reading constraint names failed: " + err.Error()
		return
	}
	check.SourceFound, check.NameOccupiedBy = facts.SourceFound, facts.NameOccupiedBy
	check.Observed = true
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
	// ConstraintDataReferencedByForeignKeys: foreign keys depend on the primary
	// key being replaced. They are named and the replacement is refused; the
	// provider never cascades.
	ConstraintDataReferencedByForeignKeys ConstraintDataReadinessReason = "primary-key-referenced-by-foreign-keys"
	// ConstraintRenameSourceMissing: no constraint of the family carries the
	// observed name any more; the plan is stale and is not applied.
	ConstraintRenameSourceMissing ConstraintDataReadinessReason = "constraint-rename-source-missing"
	// ConstraintRenameNameTaken: another relation or constraint already holds
	// the new name, so the constraint is unresolved rather than renamed.
	ConstraintRenameNameTaken ConstraintDataReadinessReason = "constraint-rename-name-taken"
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
		case want.Kind == ConstraintDataRenameConstraint:
			assessment.Check = check
			switch {
			case !check.SourceFound:
				assessment.Executable = false
				assessment.Reason = ConstraintRenameSourceMissing
				assessment.Detail = fmt.Sprintf("no %s constraint %q exists on the table any more; the observation is stale", check.RenameKind, check.BeforeName)
			case check.NameOccupiedBy != "":
				assessment.Executable = false
				assessment.Reason = ConstraintRenameNameTaken
				assessment.Detail = fmt.Sprintf("the name %q is already used by %s; unresolved, not renamed", check.NewName, check.NameOccupiedBy)
			}
		default:
			assessment.Check = check
			if violations := violationSummary(check); violations != "" {
				assessment.Executable = false
				assessment.Reason = ConstraintDataViolations
				assessment.Detail = violations + "; no declared handling exists, so the constraint is not added"
			} else if want.Kind == ConstraintDataPrimaryKeyReplace {
				switch {
				case strings.TrimSpace(check.PrimaryKeyName) == "":
					assessment.Executable = false
					assessment.Reason = ConstraintDataUnobserved
					assessment.Detail = "the existing primary key constraint name was not observed; it is never assumed from the table name"
				case len(check.ReferencingForeignKeys) > 0:
					assessment.Executable = false
					assessment.Reason = ConstraintDataReferencedByForeignKeys
					assessment.Detail = "referenced by " + describeReferencingForeignKeys(check.ReferencingForeignKeys) + "; the primary key is not replaced"
				}
			} else if want.Kind == ConstraintDataPrimaryKeyRemove {
				switch {
				case strings.TrimSpace(check.PrimaryKeyName) == "":
					assessment.Executable = false
					assessment.Reason = ConstraintDataUnobserved
					assessment.Detail = "the existing primary key constraint name was not observed; it is never assumed from the table name"
				case len(check.ReferencingForeignKeys) > 0:
					assessment.Executable = false
					assessment.Reason = ConstraintDataReferencedByForeignKeys
					assessment.Detail = "referenced by " + describeReferencingForeignKeys(check.ReferencingForeignKeys) + "; the primary key is not removed"
				}
			}
		}
		if assessment.Executable && (want.Kind == ConstraintDataPrimaryKey || want.Kind == ConstraintDataPrimaryKeyReplace) {
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
	case ConstraintDataPrimaryKey, ConstraintDataPrimaryKeyReplace, ConstraintDataUnique:
		if c.DuplicateRows > 0 || c.DuplicateKeys > 0 {
			parts = append(parts, fmt.Sprintf("%d row(s) share %d duplicated key value(s)", c.DuplicateRows, c.DuplicateKeys))
		}
		if c.Kind != ConstraintDataUnique && c.NullKeyRows > 0 {
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

const constraintSourceQuery = `
SELECT count(*)
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class cls ON cls.oid = con.conrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
WHERE ns.nspname = $1 AND cls.relname = $2 AND con.conname = $3 AND con.contype::text = $4`

// constraintNameOccupantQuery lists what holds the new name: a relation of the
// schema (tables, indexes, the primary key's index, sequences, views) and a
// constraint of the same table. Only the kinds are returned, never data.
const constraintNameOccupantQuery = `
SELECT 'relation of kind ' || cls.relkind::text
FROM pg_catalog.pg_class cls
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
WHERE ns.nspname = $1 AND cls.relname = $3
UNION ALL
SELECT 'constraint of type ' || con.contype::text || ' on the same table'
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class cls ON cls.oid = con.conrelid
JOIN pg_catalog.pg_namespace ns ON ns.oid = cls.relnamespace
WHERE ns.nspname = $1 AND cls.relname = $2 AND con.conname = $3`

func (r SQLCatalogReader) ConstraintRenameFacts(ctx context.Context, table RelationIdentity, kind ConstraintRenameKind, oldName, newName string) (ConstraintRenameFacts, error) {
	var contype string
	switch kind {
	case ConstraintRenameUnique:
		contype = "u"
	case ConstraintRenameForeignKey:
		contype = "f"
	default:
		return ConstraintRenameFacts{}, fmt.Errorf("unknown constraint rename kind %q", kind)
	}
	var facts ConstraintRenameFacts
	var found int
	if err := r.DB.QueryRowContext(ctx, constraintSourceQuery, table.Schema, table.Name, oldName, contype).Scan(&found); err != nil {
		return ConstraintRenameFacts{}, err
	}
	facts.SourceFound = found == 1
	rows, err := r.DB.QueryContext(ctx, constraintNameOccupantQuery, table.Schema, table.Name, newName)
	if err != nil {
		return ConstraintRenameFacts{}, err
	}
	defer rows.Close()
	var occupants []string
	for rows.Next() {
		var what string
		if err := rows.Scan(&what); err != nil {
			return ConstraintRenameFacts{}, err
		}
		// A foreign key is not a relation: only constraint names of its own
		// table can collide with it.
		if kind == ConstraintRenameForeignKey && !strings.HasPrefix(what, "constraint") {
			continue
		}
		occupants = append(occupants, what)
	}
	if err := rows.Err(); err != nil {
		return ConstraintRenameFacts{}, err
	}
	facts.NameOccupiedBy = strings.Join(occupants, " and ")
	return facts, nil
}
