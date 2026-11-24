package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file carries the parts of the row-scoped operator design that are
// shared by the evaluator, the readers and the no-overlap rule:
// the three-valued answer, the reading of one input over a relation's rows,
// and the one place where a three-valued answer becomes a Decision.
//
// Decision section 15 fixes the shape. A row-scoped input is read once, for
// all the rows of one relation, so every input of one rule comes from one
// Read Attempt. An implementation is a pure function: it receives the states
// of the rows that were read and answers TriYes, TriNo or TriMaybe; it never
// sees a Source. Whether a partly read relation may still be answered
// not-satisfied is decided by the operator's own `refutes on some rows`
// statement, and that composition happens here and nowhere else.

// Tri is an answer that may depend on something that was not read.
//
// TriYes: the rows that were read state that the rule holds.
// TriNo: the rows that were read contain a counterexample.
// TriMaybe: whether the rows that were read hold the rule depends on a value
// that was not read.
type Tri int

const (
	TriNo Tri = iota
	TriYes
	TriMaybe
)

// TriAnd combines two answers that must both hold. One certain counterexample
// settles it; it is certain only when both are certain.
func TriAnd(a, b Tri) Tri {
	switch {
	case a == TriNo || b == TriNo:
		return TriNo
	case a == TriYes && b == TriYes:
		return TriYes
	}
	return TriMaybe
}

// NumberRowsReading is the value of one input over the rows of a relation,
// with the state of every row and whether those rows are all the rows.
type NumberRowsReading struct {
	Coverage Coverage
	Rows     []NumberReading
}

// StatefulRowNumberSource is implemented by a reader that can read one
// attribute over every row of a relation. A reader that cannot is refused
// rather than asked row by row: a rule whose input reads all the rows cannot
// be answered from one row.
type StatefulRowNumberSource interface {
	ReadRelationNumbers(relation, attribute core.CanonicalID) (NumberRowsReading, error)
}

// DecideRowScoped is the one place a row-scoped operator answer, the read
// Coverage and the operator's refutation statement become a Decision. It is
// exported so a caller that already holds those three facts (the product read
// path, or a test written from the design's table) uses the same composition
// the evaluator uses, instead of writing a second one.
func DecideRowScoped(rule semantic.Validation, coverage Coverage, refutesOnSomeRows bool, answer Tri, rowReason *Undecided) (Decision, error) {
	return decideRowScoped(rule, coverage, refutesOnSomeRows, answer, rowReason)
}

// decideRowScoped writes one row-scoped operator answer, the read Coverage and
// the operator's refutation statement into one Decision.
//
// The order matters and is the same order the no-overlap rule already uses:
// a reason inside the rows (a value that is absent, or was not read) is
// reported before the coverage, because reading more rows cannot settle it.
// A counterexample that was seen is a decision whatever the coverage, but only
// when the operator says that a counterexample among the rows read is enough;
// otherwise a partially read relation is undecided.
func decideRowScoped(rule semantic.Validation, coverage Coverage, refutesOnSomeRows bool, answer Tri, rowReason *Undecided) (Decision, error) {
	if coverage != CoverageComplete && coverage != CoveragePartial {
		return Decision{}, fmt.Errorf("semantic validation %s: the reader reports relation %s with the unknown coverage %q", rule.ID, rule.Relation, coverage)
	}
	// A reason inside the rows is reported before the coverage, because
	// reading more rows cannot settle a value that is absent or was not read
	// in a row that was read. It is meaningful only when the operator could
	// not decide; an implementation that answers certainly while a row holds
	// no value would have decided around a value it never saw.
	switch answer {
	case TriMaybe:
		if rowReason != nil {
			return Decision{Undecided: rowReason}, nil
		}
		return undecided(UndecidedValueNotFetched, rule.Relation, rule.Attribute)
	case TriNo, TriYes:
		if rowReason != nil {
			return Decision{}, fmt.Errorf("semantic validation %s: the operator answered %d while a row holds %s; a decided answer cannot be reported together with a reason inside the rows",
				rule.ID, int(answer), rowReason.Reason)
		}
	default:
		return Decision{}, fmt.Errorf("semantic validation %s: the operator answered %d, which is not a row-scoped answer", rule.ID, int(answer))
	}
	switch answer {
	case TriNo:
		if coverage == CoverageComplete || refutesOnSomeRows {
			return decided(ResultNotSatisfied)
		}
		return undecided(UndecidedRelationPartiallyRead, rule.Relation, rule.Attribute)
	default:
		if coverage == CoverageComplete {
			return decided(ResultSatisfied)
		}
		return undecided(UndecidedRelationPartiallyRead, rule.Relation, rule.Attribute)
	}
}
