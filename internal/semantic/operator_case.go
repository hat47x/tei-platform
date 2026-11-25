package semantic

import (
	"fmt"
	"math/big"
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
		if !present {
			return false
		}
		// A presence input's cell states one of two words, which is not a
		// decimal, so two cells state the same example when they are the same
		// word. A decimal keeps the number comparison, which reads "5" and
		// "5.0" as one number and never reads a word as one.
		if IsOperatorCasePresenceWord(value) {
			if value != other {
				return false
			}
			continue
		}
		if !SameOperatorCaseValue(value, other) {
			return false
		}
	}
	return true
}

// OperatorCasePresencePresent and OperatorCasePresenceAbsent are the two words
// a presence input's example cell states. A presence input reads whether the
// attribute holds a value, not the value itself, so its example states one of
// these two words rather than a decimal.
const (
	OperatorCasePresencePresent = "present"
	OperatorCasePresenceAbsent  = "absent"
)

// IsOperatorCasePresenceWord reports whether the text is one of the two words a
// presence input's example cell may state.
//
// It lives here, beside the decimal reader, because the example table's reader
// reads every cell before the signature is known: a word this function accepts
// is carried as the cell's text and checked against the input's type at the
// Workspace, where the declaration is in hand.
func IsOperatorCasePresenceWord(text string) bool {
	switch text {
	case OperatorCasePresencePresent, OperatorCasePresenceAbsent:
		return true
	}
	return false
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
