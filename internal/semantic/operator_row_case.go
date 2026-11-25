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
}

// OperatorRowExpect is the answer a row example states.
//
// It has the three values a row-scoped operator can give, so an author states
// "undecided" instead of having to write an answer that is not true (section
// 8.1, "the three-valued example").
type OperatorRowExpect string

const (
	OperatorRowSatisfied    OperatorRowExpect = "satisfied"
	OperatorRowNotSatisfied OperatorRowExpect = "not-satisfied"
	OperatorRowUndecided    OperatorRowExpect = "undecided"
)

// Valid reports whether this is an answer an example may state.
func (e OperatorRowExpect) Valid() bool {
	switch e {
	case OperatorRowSatisfied, OperatorRowNotSatisfied, OperatorRowUndecided:
		return true
	}
	return false
}

// A row cell states one of three things, and they are kept apart because
// decision section 8.1 requires it: a number, a value the row certainly does
// not hold, and a value that was not read. A missing column is never used as a
// stand-in for "not read", and an empty cell is never read as zero.
const (
	// OperatorRowNoValue is written as an empty cell. The row holds no value.
	OperatorRowNoValue = ""
	// OperatorRowNotFetched is written as this word in a cell. The value was
	// not read, so nothing is known about it, including whether it is empty.
	OperatorRowNotFetched = "not-fetched"
)

// OperatorRowCellState says what one cell of a row example states.
type OperatorRowCellState int

const (
	// OperatorRowCellNumber: the cell holds a decimal.
	OperatorRowCellNumber OperatorRowCellState = iota
	// OperatorRowCellNoValue: the cell is empty; the row holds no value.
	OperatorRowCellNoValue
	// OperatorRowCellNotFetched: the cell says the value was not read.
	OperatorRowCellNotFetched
)

// ReadOperatorRowCell reads one cell of a row example. The text is kept exactly
// as written so no precision is lost or invented.
func ReadOperatorRowCell(text string) (OperatorRowCellState, string, error) {
	switch text {
	case OperatorRowNoValue:
		return OperatorRowCellNoValue, "", nil
	case OperatorRowNotFetched:
		return OperatorRowCellNotFetched, "", nil
	}
	if _, err := ParseOperatorCaseValue(text); err != nil {
		return 0, "", err
	}
	return OperatorRowCellNumber, text, nil
}

// SameOperatorRowCaseStates reports whether two row examples state the same
// rows, however they are written. Section 8.1: two tables whose numbers, input
// names, row order and multiplicity are equal are the same example and are
// refused as a duplicate even when their answers differ; a permuted column
// order is the same example, and a permuted row order is a different one.
func SameOperatorRowCaseStates(left, right OperatorRowCase) bool {
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
			if !present || !SameOperatorCaseValue(value, other) {
				return false
			}
		}
	}
	return true
}
