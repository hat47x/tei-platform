// Package derivationbatch is a D13 read-only reference boundary.
// It fixes the relation between per-row derivation outcomes and the whole
// batch result: a row that cannot be evaluated must not be hidden behind a
// NULL, a 0, a dropped row or a complete Query result.
//
// It does not define the D1 error set, the D2 query planner, the D5
// disclosure decision or the D7 explanation. It only decides, from already
// evaluated rows, whether a result over them can be complete, can be an
// explicitly partial display projection, or must fail.
package derivationbatch

import (
	"errors"
	"fmt"
	"strings"

	"tei/internal/semantic/derivationref"
)

var (
	ErrIncomplete = errors.New("derivation batch result is not provably complete")
	ErrInvalidRow = errors.New("derivation batch row is invalid")
)

// RowOutcome is the per-row result class. It is read from the D1 result, not a
// second D1 error enum: a legitimate NULL is not a failure, and an unavailable
// input or a failed evaluation is never a NULL.
type RowOutcome string

const (
	RowValue            RowOutcome = "value"
	RowNull             RowOutcome = "null"
	RowInputUnavailable RowOutcome = "input-unavailable"
	RowEvaluationFailed RowOutcome = "evaluation-failed"
)

// RowOutcomeOf classifies one already evaluated row using the D1 result and
// error. Missing input, an absent relation and a cardinality violation are all
// "input unavailable"; a type, operator, cycle or impure failure is an
// "evaluation failure". Authorization denial is owned by D5, not decided here.
func RowOutcomeOf(value derivationref.Input, err error) RowOutcome {
	switch {
	case err == nil && value.State == derivationref.Present:
		return RowValue
	case err == nil && value.State == derivationref.Null:
		return RowNull
	case errors.Is(err, derivationref.ErrInput):
		return RowInputUnavailable
	case err != nil:
		return RowEvaluationFailed
	default:
		return RowEvaluationFailed
	}
}

// Usage is how a derivation participates in a Query over its rows. DisplayOnly
// and CountSourceRows do not let an unevaluated row corrupt a set result;
// every other usage does.
type Usage string

const (
	DisplayOnly     Usage = "display-only"      // optional column, not in set operations
	Filter          Usage = "filter"            // WHERE on the derived value
	Sort            Usage = "sort"              // ORDER BY the derived value
	Aggregate       Usage = "aggregate"         // SUM/AVG over the derived value
	Group           Usage = "group"             // GROUP BY the derived value
	CountDerived    Usage = "count-derived"     // COUNT after a derived filter
	CountSourceRows Usage = "count-source-rows" // COUNT of source rows only
	PageCursor      Usage = "page-cursor"       // keyset/cursor on the derived value
)

// usesDerivation reports whether an unevaluated row makes the usage's result
// unprovable. A source-row count does not read the derived value at all.
func usesDerivation(usage Usage) bool {
	return usage != CountSourceRows
}

// Row is one already evaluated source row.
type Row struct {
	ID     string
	Result derivationref.Input
	Err    error
}

// RowView projects one row. Value is set only for a real value; an
// unevaluated row keeps an empty Value so it is not mistaken for 0 or NULL.
type RowView struct {
	ID      string     `json:"id"`
	Outcome RowOutcome `json:"outcome"`
	Value   string     `json:"value,omitempty"`
}

// Decision is the batch result contract. Complete means the whole result is
// proven for the requested usage. Partial means only an explicitly permitted
// display projection was produced and must be labelled as such. The zero
// Decision with ErrIncomplete carries no row data, so a failed batch does not
// leak a partial result.
type Decision struct {
	Complete    bool      `json:"complete"`
	Partial     bool      `json:"partial"`
	Rows        []RowView `json:"rows,omitempty"`
	Unevaluated []string  `json:"unevaluated,omitempty"`
}

// Plan decides whether the batch over rows can satisfy usage. A usage that
// reads the derived value fails as a whole when any row is unevaluated, even
// when partial display output is otherwise allowed. Only DisplayOnly with an
// explicit allowance returns a partial projection that keeps every row and
// marks the unevaluated ones.
func Plan(usage Usage, allowPartial bool, rows []Row) (Decision, error) {
	if !knownUsage(usage) {
		return Decision{}, fmt.Errorf("%w: usage %q", ErrIncomplete, usage)
	}
	views := make([]RowView, 0, len(rows))
	unevaluated := make([]string, 0)
	for _, row := range rows {
		if row.ID == "" || row.ID != strings.TrimSpace(row.ID) {
			return Decision{}, fmt.Errorf("%w: id", ErrInvalidRow)
		}
		outcome := RowOutcomeOf(row.Result, row.Err)
		view := RowView{ID: row.ID, Outcome: outcome}
		if outcome == RowValue {
			view.Value = row.Result.Decimal
		}
		views = append(views, view)
		if outcome == RowInputUnavailable || outcome == RowEvaluationFailed {
			unevaluated = append(unevaluated, row.ID)
		}
	}
	if len(unevaluated) == 0 || !usesDerivation(usage) {
		return Decision{Complete: true, Rows: views}, nil
	}
	if usage == DisplayOnly && allowPartial {
		return Decision{Partial: true, Rows: views, Unevaluated: unevaluated}, nil
	}
	return Decision{}, fmt.Errorf("%w: %s with %d unevaluated row(s)", ErrIncomplete, usage, len(unevaluated))
}

func knownUsage(usage Usage) bool {
	switch usage {
	case DisplayOnly, Filter, Sort, Aggregate, Group, CountDerived, CountSourceRows, PageCursor:
		return true
	default:
		return false
	}
}
