package semantic

// This file states the example of an operator whose input reads every row of
// a relation.
//
// Decision section 8, 8.1: for such an operator `one table is one example`.
// Its meaning is not "for this value it answers this"; it is "for this
// collection of rows it answers this". So the example is the rows themselves
// and one expected answer written in the mark, and the columns are the
// operator's input names.
//
// The example is kept apart from OperatorCase on purpose. OperatorCase states
// one record's values and a boolean answer, and its JSON is unchanged; an
// example here states many rows and an explicit answer, so a legacy `false`
// cannot be mistaken for the answer of a row example.

// OperatorRowCase is one example of a row-scoped operator: the rows, and the
// answer the contract states for them.
type OperatorRowCase struct {
	// Rows states one value per input name per row, in the spelling the
	// declaration gave. Row order and multiplicity are preserved, because a
	// collection with two equal rows is not the same collection as one with
	// one.
	Rows []map[string]string `json:"rows"`
	// Answer is what the contract states for those rows. It is explicit: the
	// zero value of the field means no answer was stated, and that is refused
	// rather than read as not-satisfied.
	Answer OperatorRowExpect `json:"answer"`
	// Empty states that the example is the relation with no rows at all,
	// rather than a relation whose rows could not be read. The two are
	// different examples: no row can break a rule about none, while a row
	// that was not read might. Rows is then empty, and Empty is what tells
	// "stated no row" from "stated rows that happen to be empty".
	Empty bool `json:"empty,omitempty"`
}

// OperatorRowExpect is the answer a row example states.
//
// A row states a decimal, an empty cell (the row holds no value), or the word
// `not-fetched` (the row was not read). Section 8.1's table of three-valued
// answers therefore has three answers, and each is written rather than left
// to a zero value: an example that states no answer is refused, not read as
// not-satisfied.
type OperatorRowExpect string

const (
	OperatorRowSatisfied    OperatorRowExpect = "satisfied"
	OperatorRowNotSatisfied OperatorRowExpect = "not-satisfied"
	// OperatorRowUndecided is the answer of an example whose rows cannot be
	// compared: a row holds no value, or was not read, with no row that
	// refutes the operator.
	OperatorRowUndecided OperatorRowExpect = "undecided"
)

// RowCellNoValue is the text of a row cell that states the row holds no value.
// An empty cell writes it: the cell is present and holds the empty text, which
// is not the same as a missing column.
const RowCellNoValue = ""

// RowCellNotFetched is the word a row cell writes when the row was not read.
// It is a word rather than an empty cell so a value nobody fetched is never
// written as a value nobody holds.
const RowCellNotFetched = "not-fetched"

// Valid reports whether this is an answer an example may state.
func (e OperatorRowExpect) Valid() bool {
	switch e {
	case OperatorRowSatisfied, OperatorRowNotSatisfied, OperatorRowUndecided:
		return true
	}
	return false
}

// SameOperatorRowValue reports whether two row cells state the same value: two
// spellings of one number are the same ("2" and "2.0"), an empty cell is the
// same as an empty cell, `not-fetched` is the same as `not-fetched`, and a
// number is never the same as no value or as an unread row.
func SameOperatorRowValue(left, right string) bool {
	switch {
	case left == RowCellNotFetched || right == RowCellNotFetched:
		return left == RowCellNotFetched && right == RowCellNotFetched
	case left == RowCellNoValue || right == RowCellNoValue:
		return left == RowCellNoValue && right == RowCellNoValue
	}
	return SameOperatorCaseValue(left, right)
}

// SameOperatorRowCaseStates reports whether two row examples state the same
// rows, however they are written. Section 8.1: two tables whose numbers, input
// names, row order and multiplicity are equal are the same example and are
// refused as a duplicate even when their answers differ; a permuted column
// order is the same example, and a permuted row order is a different one.
func SameOperatorRowCaseStates(left, right OperatorRowCase) bool {
	// An empty relation is the same example as another empty relation, and a
	// different example from any table that states a row.
	if left.Empty != right.Empty {
		return false
	}
	if len(left.Rows) != len(right.Rows) {
		return false
	}
	for i := range left.Rows {
		a, b := left.Rows[i], right.Rows[i]
		if len(a) != len(b) {
			return false
		}
		for name, value := range a {
			other, present := b[name]
			if !present || !SameOperatorRowValue(value, other) {
				return false
			}
		}
	}
	return true
}
