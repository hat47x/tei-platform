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

// RowValue is one row's value of any type, with the state of the read that
// produced it.
//
// The state is carried next to the value, exactly as a one-record reading does,
// so "this row holds no value" and "this row was not read" stay apart and
// neither becomes the zero of its type (decision section 5). Value is set only
// when State is ValuePresent.
type RowValue struct {
	State ValueState
	Value semantic.OperatorValue
}

// ValueRowsReading is the value of one input over the rows of a relation, read
// as the type the declaration gives the input (decision section 8.4).
//
// It is the typed counterpart of NumberRowsReading: the rows of a relation are
// read once, and the value of each row carries which type it was read as, so an
// implementation that decides a column of days receives days and not numbers.
type ValueRowsReading struct {
	Coverage Coverage
	Rows     []RowValue
}

// StatefulRowValueSource is implemented by a reader that can read one attribute
// over every row of a relation as a value of the type the declaration gives it.
// A reader that cannot is refused rather than asked row by row, and the value is
// never read as a number it was not declared as.
type StatefulRowValueSource interface {
	ReadRelationValues(relation, attribute core.CanonicalID) (ValueRowsReading, error)
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
//
// A witnessed counterexample is also a decision when another row holds no
// value: section 6 says a counterexample that was seen settles the rule
// "whatever else is unknown", and nothing an absent row could add would undo
// it. Only an answer that claims the rule holds has to account for every row,
// so a reason inside the rows is dropped for TriYes and reported for the rest.
func decideRowScoped(rule semantic.Validation, coverage Coverage, refutesOnSomeRows bool, answer Tri, rowReason *Undecided) (Decision, error) {
	if coverage != CoverageComplete && coverage != CoveragePartial {
		return Decision{}, fmt.Errorf("semantic validation %s: the reader reports relation %s with the unknown coverage %q", rule.ID, rule.Relation, coverage)
	}
	switch answer {
	case TriMaybe:
		if rowReason != nil {
			return Decision{Undecided: rowReason}, nil
		}
		return undecided(UndecidedValueNotFetched, rule.Relation, rule.Attribute)
	case TriYes:
		// Holding everywhere cannot be claimed while a row that was read
		// holds no value or was not read: that row might break the rule.
		if rowReason != nil {
			return Decision{Undecided: rowReason}, nil
		}
	case TriNo:
		// A counterexample that was seen is settled by itself.
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
