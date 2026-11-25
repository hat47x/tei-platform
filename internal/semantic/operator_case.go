package semantic

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// OperatorCase is one example of an operator contract: for this value the
// operator answers satisfied or not satisfied. Examples are stated by the
// author of the contract. They are not produced by an implementation, so an
// implementation can be checked against them.
type OperatorCase struct {
	// Value is a decimal number exactly as written, such as "300", "0", or
	// "-0.01". It is kept as text so no precision is lost or invented.
	//
	// It states the example of a SINGLE-input operator. A declaration that
	// lists more than one input states its example as one value per input name
	// in Inputs instead, because one value cannot say what the other inputs
	// held.
	Value     string `json:"value"`
	Satisfied bool   `json:"satisfied"`
	// Inputs holds one decimal text per declared input name, in the spelling
	// the declaration gave. It is set only for a multi-input example; a
	// single-input example leaves it empty and uses Value, so its JSON is
	// unchanged.
	Inputs map[string]string `json:"inputs,omitempty"`
	// Rows states an example of an operator that reads every row of a
	// relation: one table is one example, and each element of Rows is one row,
	// mapping a declared input name to the decimal text that row holds. The
	// order of the rows is the order the author wrote, and a repeated row is
	// kept: two rows that hold the same numbers are still two rows.
	//
	// Rows is set only for a row-set example. A one-value example leaves it
	// empty and keeps its JSON unchanged, so a reader that sees no Rows reads
	// Value or Inputs exactly as before.
	Rows []map[string]string `json:"rows,omitempty"`
	// RowExpect states the expected answer of a row-set example, as written:
	// RowExpectSatisfied or RowExpectNotSatisfied. It is empty for a one-value
	// example, and Rows is empty whenever it is empty.
	//
	// The answer is a word rather than a bool on purpose. A bool cannot tell
	// "this example expects not-satisfied" from "this example states no
	// expectation", and an example that lost its expectation must not be read
	// as expecting not-satisfied.
	RowExpect string `json:"rowExpect,omitempty"`
}

// The answers a row-set example can state in its mark. The design keeps the
// first change to examples whose values are all present, so `undecided` is not
// offered: a word this reader does not know is refused rather than read as
// one of these.
const (
	RowExpectSatisfied    = "satisfied"
	RowExpectNotSatisfied = "not-satisfied"
)

// IsRowSetExample reports whether this example states the rows of a relation
// rather than the values of one record.
func (c OperatorCase) IsRowSetExample() bool {
	return c.Rows != nil
}

// RowSetExampleStates renders a row-set example for a diagnostic: the answer it
// expects and the rows it states, so a reader can find the table.
func RowSetExampleStates(example OperatorCase) string {
	names := make([]string, 0, len(example.Rows))
	for index, row := range example.Rows {
		parts := make([]string, 0, len(row))
		for name, value := range row {
			parts = append(parts, name+"="+value)
		}
		sort.Strings(parts)
		names = append(names, fmt.Sprintf("row %d (%s)", index+1, strings.Join(parts, ", ")))
	}
	return fmt.Sprintf("expecting %s over %d row(s): %s", example.RowExpect, len(example.Rows), strings.Join(names, "; "))
}

// SameRowSetRows reports whether two row-set examples state the same rows in
// the same order, however the numbers and the columns are written. The columns
// are matched by name, so a table that writes limit first states the same rows
// as one that writes value first.
//
// The expected answer is not part of this comparison on purpose. Two tables
// that state the same rows are the same example written twice; if they answer
// differently, one of them is wrong, and the declaration cannot say both.
// Section 8.1 requires the duplicate to be refused whatever the two answers
// are, so the answer cannot be a reason to treat them as different examples.
//
// A repeated row inside one table is kept: this compares two tables, not two
// rows of one table.
func SameRowSetRows(left, right OperatorCase) bool {
	if !left.IsRowSetExample() || !right.IsRowSetExample() {
		return false
	}
	if len(left.Rows) != len(right.Rows) {
		return false
	}
	for index, row := range left.Rows {
		other := right.Rows[index]
		if len(row) != len(other) {
			return false
		}
		for name, value := range row {
			otherValue, present := other[name]
			if !present || !SameOperatorCaseValue(value, otherValue) {
				return false
			}
		}
	}
	return true
}

// SameOperatorCaseStates reports whether two examples state the same values,
// however they are written, so a table cannot list one example twice. A
// single-input example compares its one value; a multi-input example compares
// every input it states and requires the same input names.
func SameOperatorCaseStates(left, right OperatorCase) bool {
	if len(left.Inputs) != len(right.Inputs) {
		return false
	}
	if len(left.Inputs) == 0 {
		return SameOperatorCaseValue(left.Value, right.Value)
	}
	for name, value := range left.Inputs {
		other, present := right.Inputs[name]
		if !present || !SameOperatorCaseValue(value, other) {
			return false
		}
	}
	return true
}

// ParseOperatorCaseValue reads the decimal text of an example. It accepts an
// optional minus sign, digits, and an optional fraction, and nothing else: no
// exponent, no digit grouping, no leading plus, no surrounding space. A text
// that might be read as another number is refused instead of approximated.
func ParseOperatorCaseValue(text string) (*big.Rat, error) {
	digits := text
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	seenDigit, seenPoint, fractionDigits := false, false, 0
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		switch {
		case c >= '0' && c <= '9':
			seenDigit = true
			if seenPoint {
				fractionDigits++
			}
		case c == '.' && !seenPoint && seenDigit:
			seenPoint = true
		default:
			return nil, fmt.Errorf("%q is not a decimal number; write digits with an optional minus sign and fraction, for example -0.01", text)
		}
	}
	// "007" might be meant as 7, as an octal number, or as a code. Refuse it.
	leadingZero := len(digits) > 1 && digits[0] == '0' && digits[1] != '.'
	if !seenDigit || seenPoint && fractionDigits == 0 || leadingZero {
		return nil, fmt.Errorf("%q is not a decimal number; write digits with an optional minus sign and fraction, for example -0.01", text)
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, fmt.Errorf("%q is not a decimal number", text)
	}
	return value, nil
}

// SameOperatorCaseValue reports whether two example values are the same
// number, however they are written ("0" and "0.0").
func SameOperatorCaseValue(left, right string) bool {
	a, errA := ParseOperatorCaseValue(left)
	b, errB := ParseOperatorCaseValue(right)
	return errA == nil && errB == nil && a.Cmp(b) == 0
}
