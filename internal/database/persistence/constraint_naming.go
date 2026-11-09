package persistence

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/database/physical"
)

// ConstraintNameKind identifies the constraint whose name a provider policy is
// asked to derive.
type ConstraintNameKind string

const (
	ConstraintNamePrimaryKey ConstraintNameKind = "primary-key"
	ConstraintNameUnique     ConstraintNameKind = "unique"
	ConstraintNameForeignKey ConstraintNameKind = "foreign-key"
)

// ConstraintNamingPolicy is an optional part of PhysicalRealizationPolicy
// (ADR-0013). A provider that implements it gives every unique constraint and
// foreign key without a declared Name a deterministic name during Desired
// realization, so the Desired Physical Schema carries the name the provider
// will create. The primary key stays unnamed in the Desired schema; its
// predicted name is used only to detect collisions.
type ConstraintNamingPolicy interface {
	ConstraintName(kind ConstraintNameKind, table string, columns []string) string
	// ConstraintNameCollisions returns, for every generated name among the
	// given tables, the collisions with another name in a namespace the
	// provider enforces. Collisions between two declared names are not
	// reported here.
	ConstraintNameCollisions(tables []ConstraintNameTable) []ConstraintNameCollision
}

// ConstraintNameTable is the naming view of one realized table.
type ConstraintNameTable struct {
	Schema string
	Table  string
	// Relation names that already occupy the schema namespace besides the
	// constraints below (the table itself and its declared indexes).
	Indexes []string
	// PrimaryKey is the predicted provider name of the unnamed primary key, or
	// empty when the table has no primary key.
	PrimaryKey string
	Unique     []ConstraintNameEntry
	Foreign    []ConstraintNameEntry
}

// ConstraintNameEntry is one constraint name and whether realization
// generated it (true) or the mapping declared it (false).
type ConstraintNameEntry struct {
	Name      string
	Generated bool
}

// ConstraintNameCollision reports a generated constraint name the provider
// would reject or rename because another name in the same namespace is equal.
type ConstraintNameCollision struct {
	// Table is the qualified name of the table that owns the generated name.
	Table  string
	Name   string
	Reason string
}

// UnresolvedTable is a realized table whose constraint names could not be
// decided. Callers must not render it.
type UnresolvedTable struct {
	Table  string
	Reason string
}

// ConstraintNamingError is returned by RealizeDesired when generated
// constraint names collide.
type ConstraintNamingError struct {
	Unresolved []UnresolvedTable
}

func (e *ConstraintNamingError) Error() string {
	parts := make([]string, len(e.Unresolved))
	for i, u := range e.Unresolved {
		parts[i] = u.Table + ": " + u.Reason
	}
	return "constraint names could not be decided: " + strings.Join(parts, "; ")
}

// assignConstraintNames fills the empty unique constraint and foreign key
// names of the realized tables when the policy implements
// ConstraintNamingPolicy. Generated names that collide are cleared again and
// their tables are reported as unresolved, sorted by table name.
func assignConstraintNames(tables []physical.Table, policy PhysicalRealizationPolicy) []UnresolvedTable {
	namer, ok := policy.(ConstraintNamingPolicy)
	if !ok {
		return nil
	}
	views := make([]ConstraintNameTable, len(tables))
	for i := range tables {
		t := &tables[i]
		view := ConstraintNameTable{Schema: t.Schema, Table: t.Name}
		for _, idx := range t.Indexes {
			view.Indexes = append(view.Indexes, idx.Name)
		}
		if len(t.PrimaryKey) > 0 {
			view.PrimaryKey = namer.ConstraintName(ConstraintNamePrimaryKey, t.Name, t.PrimaryKey)
		}
		for j := range t.UniqueConstraints {
			uc := &t.UniqueConstraints[j]
			generated := strings.TrimSpace(uc.Name) == ""
			if generated {
				uc.Name = namer.ConstraintName(ConstraintNameUnique, t.Name, uc.Columns)
				uc.NameGenerated = uc.Name != ""
			}
			view.Unique = append(view.Unique, ConstraintNameEntry{Name: uc.Name, Generated: generated})
		}
		for j := range t.ForeignKeys {
			fk := &t.ForeignKeys[j]
			generated := strings.TrimSpace(fk.Name) == ""
			if generated {
				fk.Name = namer.ConstraintName(ConstraintNameForeignKey, t.Name, fk.Columns)
				fk.NameGenerated = fk.Name != ""
			}
			view.Foreign = append(view.Foreign, ConstraintNameEntry{Name: fk.Name, Generated: generated})
		}
		views[i] = view
	}
	collisions := namer.ConstraintNameCollisions(views)
	if len(collisions) == 0 {
		return nil
	}
	reasons := map[string][]string{}
	for _, c := range collisions {
		reasons[c.Table] = append(reasons[c.Table], c.Reason)
	}
	var out []UnresolvedTable
	for i := range tables {
		qualified := tables[i].QualifiedName()
		rs, ok := reasons[qualified]
		if !ok {
			continue
		}
		// A table whose names are not decided must not carry a guessed name.
		for j := range tables[i].UniqueConstraints {
			if views[i].Unique[j].Generated {
				tables[i].UniqueConstraints[j].Name = ""
				tables[i].UniqueConstraints[j].NameGenerated = false
			}
		}
		for j := range tables[i].ForeignKeys {
			if views[i].Foreign[j].Generated {
				tables[i].ForeignKeys[j].Name = ""
				tables[i].ForeignKeys[j].NameGenerated = false
			}
		}
		sort.Strings(rs)
		out = append(out, UnresolvedTable{Table: qualified, Reason: fmt.Sprintf("generated constraint name collision: %s", strings.Join(rs, "; "))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Table < out[j].Table })
	return out
}
