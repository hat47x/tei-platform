// Package numbering issues sequence numbers under a declared contract. It is a
// reference implementation: the state is supplied by the caller, so the package
// does not read a database or a clock.
//
// X-TEA's Session.getNextNumber read a row, formatted the current value with a
// digit count and a prefix, optionally appended a modulus-10 check digit, then
// counted up and wrote the row back with an optimistic check on UPDCOUNTER whose
// result was never inspected (D:Session.java L1980–2009). An exception was
// swallowed and an empty string was returned (同 L2004–2007), so a failure
// looked like "no number" rather than a failure. On the digit limit the counter
// wrapped to 1 (countUpNumber, 同 L2061–2100), which reissues numbers that were
// already used. Here the contract names uniqueness, continuity, gaps, and the
// overflow behaviour, and the default overflow behaviour is to stop.
package numbering

import (
	"errors"
	"fmt"
	"strconv"
)

// Overflow names what happens when the counter reaches the digit limit.
type Overflow string

const (
	// OverflowStop refuses the issuance. It is the default, because wrapping
	// reissues numbers that were already used.
	OverflowStop Overflow = "stop"
	// OverflowWrap returns to 1. X-TEA always did this
	// (D:Session.java L2061–2100); it is available only when a declaration
	// asks for it, because it breaks uniqueness.
	OverflowWrap Overflow = "wrap"
)

// CheckDigit names the check digit a number carries.
type CheckDigit string

const (
	// CheckDigitNone carries no check digit.
	CheckDigitNone CheckDigit = ""
	// CheckDigitModulus10Weighted adds one digit: each odd-positioned digit is
	// multiplied by three, the sum's remainder modulo ten is taken, and the
	// check digit is ten minus that remainder. X-TEA did exactly that
	// (D:Session.java L2044–2057).
	CheckDigitModulus10Weighted CheckDigit = "modulus10-weighted"
)

// ErrNumbering reports an issuance the contract refuses. It is a configuration
// or state failure, not a business result.
var ErrNumbering = errors.New("numbering cannot issue")

// Contract is the declared behaviour of one numbering rule.
type Contract struct {
	// Digits is the width of the numeric part, 1..9. X-TEA counted up with a
	// separate comparison per width (D:Session.java L2061–2100); the limit is
	// the same: ten to the power of the width.
	Digits int
	// Prefix is written before the number.
	Prefix string
	// CheckDigit names the check digit.
	CheckDigit CheckDigit
	// Overflow is what happens at the limit. An empty value means OverflowStop.
	Overflow Overflow
}

// State is the current counter a caller holds.
type State struct {
	// Current is the last issued value, or zero when nothing is issued.
	Current int
}

func (c Contract) validate() error {
	if c.Digits < 1 || c.Digits > 9 {
		return fmt.Errorf("%w: digit width %d is outside 1..9", ErrNumbering, c.Digits)
	}
	switch c.CheckDigit {
	case CheckDigitNone, CheckDigitModulus10Weighted:
	default:
		return fmt.Errorf("%w: unsupported check digit %q", ErrNumbering, c.CheckDigit)
	}
	switch c.Overflow {
	case "", OverflowStop, OverflowWrap:
	default:
		return fmt.Errorf("%w: unsupported overflow behaviour %q", ErrNumbering, c.Overflow)
	}
	return nil
}

// Limit is the largest value the width holds.
func (c Contract) Limit() int {
	limit := 1
	for i := 0; i < c.Digits; i++ {
		limit *= 10
	}
	return limit - 1
}

// Next issues the next number from the state and returns the new state.
//
// The returned value is the counter, and the formatted number is what a record
// stores. A caller that cannot commit the returned state must not use the
// formatted number: the issuance and the state change belong to the same unit of
// work, which X-TEA's read-and-update did not guarantee (同 L1995–2002).
func (c Contract) Next(state State) (string, State, error) {
	if err := c.validate(); err != nil {
		return "", state, err
	}
	if state.Current < 0 {
		return "", state, fmt.Errorf("%w: the counter is negative", ErrNumbering)
	}
	next := state.Current + 1
	if next > c.Limit() {
		if c.Overflow == OverflowWrap {
			next = 1
		} else {
			return "", state, fmt.Errorf("%w: counter %d reached the limit %d for %d digits", ErrNumbering, state.Current, c.Limit(), c.Digits)
		}
	}
	formatted, err := c.Format(next)
	if err != nil {
		return "", state, err
	}
	return formatted, State{Current: next}, nil
}

// Format renders a counter value without changing the counter.
func (c Contract) Format(value int) (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}
	if value < 0 || value > c.Limit() {
		return "", fmt.Errorf("%w: value %d does not fit %d digits", ErrNumbering, value, c.Digits)
	}
	digits := strconv.Itoa(value)
	for len(digits) < c.Digits {
		digits = "0" + digits
	}
	if c.CheckDigit == CheckDigitNone {
		return c.Prefix + digits, nil
	}
	check, err := modulus10Weighted(digits)
	if err != nil {
		return "", err
	}
	return c.Prefix + digits + strconv.Itoa(check), nil
}

// modulus10Weighted returns the check digit for the digits.
//
// X-TEA appended the raw difference "10 - sum mod 10" (D:Session.java L2053),
// so a remainder of zero produced the two-character string "10" and the number
// carried one more character than its width. This implementation takes the
// difference modulo ten, so the check digit is always one character. The
// difference is visible in a test rather than smoothed away silently.
func modulus10Weighted(digits string) (int, error) {
	sum := 0
	for index := 0; index < len(digits); index++ {
		value, err := strconv.Atoi(string(digits[index]))
		if err != nil {
			return 0, fmt.Errorf("%w: %q is not a digit", ErrNumbering, digits[index])
		}
		if value%2 == 1 {
			value *= 3
		}
		sum += value
	}
	return (10 - sum%10) % 10, nil
}

// NextValue reports the counter a state would move to, without formatting. It
// exists so a caller can reserve the value before it knows the prefix, and it
// applies the same overflow rule as Next.
func (c Contract) NextValue(state State) (int, State, error) {
	if err := c.validate(); err != nil {
		return 0, state, err
	}
	next := state.Current + 1
	if next > c.Limit() {
		if c.Overflow == OverflowWrap {
			return 1, State{Current: 1}, nil
		}
		return 0, state, fmt.Errorf("%w: counter %d reached the limit %d for %d digits", ErrNumbering, state.Current, c.Limit(), c.Digits)
	}
	return next, State{Current: next}, nil
}
