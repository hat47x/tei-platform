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
// The first change that reads these examples accepts the two answers
// "satisfied" and "not-satisfied". An "undecided" answer needs the three-valued
// input states (empty and unread cells), which decision section 8.1 defers to
// the change after this one; accepting it now would let an author state
// something this path cannot check.
type OperatorRowExpect string

const (
	OperatorRowSatisfied    OperatorRowExpect = "satisfied"
	OperatorRowNotSatisfied OperatorRowExpect = "not-satisfied"
)

// Valid reports whether this is an answer an example may state today.
func (e OperatorRowExpect) Valid() bool {
	switch e {
	case OperatorRowSatisfied, OperatorRowNotSatisfied:
		return true
	}
	return false
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
