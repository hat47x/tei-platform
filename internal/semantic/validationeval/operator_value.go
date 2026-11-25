package validationeval

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file is the value an operator implementation receives, decided by the
// type its declaration states (decision section 8.4).
//
// Number, Numbers and NumberRows stay what they are: a numeric implementation
// keeps its three fields and is not asked to change. What was missing was a
// way to hand a calendar date, a string or a boolean to an implementation at
// all, and a way to hand an `any` value without losing which of those it was.
// One value type carries the answer: exactly one field is set, decided by the
// type the declaration gave, so an implementation handed a value of a type it
// does not read can say so instead of reading it as a number.
//
// The state of a row is not part of the value. Whether a row holds a value,
// holds none, or was not read is carried by ValueReading.State exactly as
// NumberReading carries it, and a typed value is set only for a row that holds
// one.

// OperatorValue is the value of one input, read according to the type the
// declaration gave. Exactly one of the fields is set, decided by the type.
type OperatorValue struct {
	Number *big.Rat               // type number
	Date   *semantic.CalendarDate // type date
	Text   *string                // type text
	Truth  *bool                  // type truth
	Any    *AnyValue              // type any
}

// AnyValue is a value read for an input of type any. Exactly one of the fields
// is set, and it records the type the value was read as: the number 100 and
// the text "100" are two values, not one. Section 8.3's quoting decides which
// one a table states, and the read keeps that decision.
type AnyValue struct {
	Number *big.Rat
	Date   *semantic.CalendarDate
	Text   *string
	Truth  *bool
}

// ValueReading is one row's value of a typed input and its state. Value is set
// exactly when State is ValuePresent.
type ValueReading struct {
	State ValueState
	Value *OperatorValue
}

// ValueRowsReading is the value of one input over the rows of a relation, with
// the state of every row and whether those rows are all the rows.
type ValueRowsReading struct {
	Coverage Coverage
	Rows     []ValueReading
}

// TextReading is one string value and its state. Value is set exactly when
// State is ValuePresent.
type TextReading struct {
	State ValueState
	Value *string
}

// TruthReading is one boolean value and its state. Value is set exactly when
// State is ValuePresent.
type TruthReading struct {
	State ValueState
	Value *bool
}

// AnyReading is one value of any type and its state. Value is set exactly when
// State is ValuePresent, and it records the type the value was read as.
type AnyReading struct {
	State ValueState
	Value *OperatorValue
}

// StatefulTextSource is implemented by a reader that can read string values. A
// rule over text is refused by a reader that cannot; it is not decided from a
// value of another representation.
type StatefulTextSource interface {
	ReadText(attribute core.CanonicalID) (TextReading, error)
}

// StatefulTruthSource is implemented by a reader that can read boolean values.
type StatefulTruthSource interface {
	ReadTruth(attribute core.CanonicalID) (TruthReading, error)
}

// StatefulAnySource is implemented by a reader that can read a value of any
// type and say which type it read it as.
type StatefulAnySource interface {
	ReadAny(attribute core.CanonicalID) (AnyReading, error)
}

// StatefulRowValueSource is implemented by a reader that can read one
// attribute of any type over every row of a relation. A reader that cannot is
// refused rather than asked row by row.
type StatefulRowValueSource interface {
	ReadRelationValues(relation, attribute core.CanonicalID) (ValueRowsReading, error)
}

// OperatorValueFor builds the value of one input of the declared type from a
// cell of an example table. The cell states the type it is written as, so an
// input of type any keeps what the table wrote and every other input takes
// only its own.
func OperatorValueFor(kind semantic.OperatorInput, cell semantic.OperatorCaseCell) (OperatorValue, error) {
	if !cell.Fits(kind) {
		return OperatorValue{}, fmt.Errorf("%q is not a value of type %s", cell.Text, kind)
	}
	switch kind {
	case semantic.OperatorInputNumber:
		return OperatorValue{Number: cell.Number}, nil
	case semantic.OperatorInputDate:
		day := cell.Date
		return OperatorValue{Date: &day}, nil
	case semantic.OperatorInputText:
		text := cell.Text
		return OperatorValue{Text: &text}, nil
	case semantic.OperatorInputTruth:
		truth := cell.Truth
		return OperatorValue{Truth: &truth}, nil
	case semantic.OperatorInputAny:
		return OperatorValue{Any: anyValueFor(cell)}, nil
	}
	return OperatorValue{}, fmt.Errorf("an input of type %s states whether a value is there, so it has no value to hand an implementation", kind)
}

// anyValueFor is the `any` form of one cell: the value as it was read, with the
// type that reading gave it.
func anyValueFor(cell semantic.OperatorCaseCell) *AnyValue {
	switch cell.Kind {
	case semantic.OperatorCaseCellNumber:
		return &AnyValue{Number: cell.Number}
	case semantic.OperatorCaseCellDate:
		day := cell.Date
		return &AnyValue{Date: &day}
	case semantic.OperatorCaseCellText:
		text := cell.Text
		return &AnyValue{Text: &text}
	case semantic.OperatorCaseCellTruth:
		truth := cell.Truth
		return &AnyValue{Truth: &truth}
	}
	return &AnyValue{}
}

// absentReason is the reason a read that found no value gives. It is the same
// mapping the numeric reader applies, so an absent value is never read as zero
// and an unread one keeps its own reason.
func absentReason(state ValueState, attribute core.CanonicalID) *Undecided {
	if state == ValueNotFetched {
		return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
	}
	return &Undecided{Reason: UndecidedValueNull, Attribute: attribute}
}
