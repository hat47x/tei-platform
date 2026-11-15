package functioncontext

import (
	"tei/internal/core"
	"tei/internal/driver"
)

// ColumnProvenance is what the Provider's statement description says about ONE result
// column: which relation column it is. Relation is empty when the column is not a plain
// relation column (an expression, a constant, an aggregate, a function result).
// It is an observation of the statement text the host prepared, not of the rows.
type ColumnProvenance struct {
	Column   string `json:"column"`
	Relation string `json:"relation,omitempty"`
	RelKind  string `json:"relKind,omitempty"`
	Attname  string `json:"attname,omitempty"`
}

const (
	// ProvDeclared: the result column is the declared physical column of the declared
	// relation for its Semantic attribute.
	ProvDeclared = "declared-column"
	// ProvOtherColumn: a plain table column, but not the declared one (an alias swapped it,
	// or another relation supplied it).
	ProvOtherColumn = "other-column"
	// ProvComputed: an expression, constant, aggregate or function result.
	ProvComputed = "computed"
	// ProvNonTable: the column belongs to a view, partitioned table, foreign table or the like.
	ProvNonTable = "non-table-relation"
	// ProvUnchecked: a plain table column, but nothing declared to compare it against.
	ProvUnchecked = "base-column-unchecked"
	// ProvUnobserved: no provenance was observed for this column.
	ProvUnobserved = "unobserved"

	NullOriginStored        = "stored"
	NullOriginUnestablished = "unestablished"
	NullOriginNone          = "no-null"

	GrainEntity        = "entity"
	GrainUnestablished = "unestablished"
)

// columnProvenance classifies each mapped column. Provenance is matched by position and the
// names must agree; any mismatch means nothing was observed for the whole result.
func columnProvenance(declared driver.DeclaredBrowseRelations, columns []Column, provenance []ColumnProvenance) []string {
	out := make([]string, len(columns))
	aligned := len(provenance) == len(columns)
	for i := range columns {
		if aligned && provenance[i].Column != columns[i].Raw {
			aligned = false
		}
	}
	physical := map[core.CanonicalID]string{}
	for _, column := range declared.Columns {
		physical[column.Attribute] = column.Column
	}
	declaredTable := map[string]bool{}
	for _, table := range declared.Tables {
		declaredTable[table] = true
	}
	for i, column := range columns {
		if !aligned {
			out[i] = ProvUnobserved
			continue
		}
		p := provenance[i]
		switch {
		case p.Relation == "":
			out[i] = ProvComputed
		case p.RelKind != "r":
			out[i] = ProvNonTable
		case declared.Status != driver.DeclaredRelationsResolved:
			out[i] = ProvUnchecked
		case declaredTable[p.Relation] && physical[column.Attribute] == p.Attname && p.Attname != "":
			out[i] = ProvDeclared
		default:
			out[i] = ProvOtherColumn
		}
	}
	return out
}

// nullOrigin says "stored" only when every column holding a NULL is the declared column of the
// declared relation AND the plan is a single scan of one relation. The second condition is
// what stops an outer join from passing: with a join, a base column can be NULL because no row
// matched, which is not a stored value.
func nullOrigin(columns []Column, rows [][]Cell, scan string) string {
	hasNull := false
	for i := range columns {
		for _, row := range rows {
			if row[i].State != StateNull {
				continue
			}
			hasNull = true
			if columns[i].Provenance != ProvDeclared || scan != ScanSingle {
				return NullOriginUnestablished
			}
		}
	}
	if !hasNull {
		return NullOriginNone
	}
	return NullOriginStored
}

// grainLevel says "entity" when each row is a row of the declared relation (single scan, the
// declared key columns are the declared columns) and the key is unique among the rows read.
func grainLevel(declared driver.DeclaredBrowseRelations, columns []Column, scan, key string) string {
	if scan != ScanSingle || key != KeyUnique || declared.Status != driver.DeclaredRelationsResolved || len(declared.PrimaryKey) == 0 {
		return GrainUnestablished
	}
	for _, attribute := range declared.PrimaryKey {
		ok := false
		for _, column := range columns {
			if column.Attribute == attribute && column.Provenance == ProvDeclared {
				ok = true
			}
		}
		if !ok {
			return GrainUnestablished
		}
	}
	return GrainEntity
}
