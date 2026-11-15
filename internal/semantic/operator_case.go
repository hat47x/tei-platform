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
	Value     string `json:"value"`
	Satisfied bool   `json:"satisfied"`
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
