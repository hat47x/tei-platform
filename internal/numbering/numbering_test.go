package numbering_test

import (
	"errors"
	"testing"

	"tei/internal/numbering"
)

// TestNextIssuesInOrder checks that the counter moves by one and the formatted
// number carries the declared width and prefix.
func TestNextIssuesInOrder(t *testing.T) {
	contract := numbering.Contract{Digits: 5, Prefix: "ORD-"}
	first, state, err := contract.Next(numbering.State{})
	if err != nil {
		t.Fatal(err)
	}
	if first != "ORD-00001" || state.Current != 1 {
		t.Fatalf("first=%q state=%+v", first, state)
	}
	second, state, err := contract.Next(state)
	if err != nil {
		t.Fatal(err)
	}
	if second != "ORD-00002" || state.Current != 2 {
		t.Fatalf("second=%q state=%+v", second, state)
	}
}

// TestOverflowStopsByDefault checks that the default is to stop at the limit.
// X-TEA wrapped to 1 (D:Session.java L2061–2100), which reissues used numbers.
func TestOverflowStopsByDefault(t *testing.T) {
	contract := numbering.Contract{Digits: 1}
	if _, _, err := contract.Next(numbering.State{Current: 9}); !errors.Is(err, numbering.ErrNumbering) {
		t.Fatalf("err=%v", err)
	}
}

// TestOverflowWrapIsChosen checks that wrapping is available only when a
// declaration asks for it, and that asking for it is visible in the state.
func TestOverflowWrapIsChosen(t *testing.T) {
	contract := numbering.Contract{Digits: 1, Overflow: numbering.OverflowWrap}
	value, state, err := contract.Next(numbering.State{Current: 9})
	if err != nil {
		t.Fatal(err)
	}
	if value != "1" || state.Current != 1 {
		t.Fatalf("value=%q state=%+v", value, state)
	}
}

// TestCheckDigitIsOneCharacter checks the modulus-10 weighted digit and the
// difference from X-TEA's raw subtraction.
//
// X-TEA appended "10 - sum mod 10" (D:Session.java L2053), so a remainder of
// zero produced "10" and the number grew by a character. The test pins the
// single-character result.
func TestCheckDigitIsOneCharacter(t *testing.T) {
	contract := numbering.Contract{Digits: 2, CheckDigit: numbering.CheckDigitModulus10Weighted}
	// 07: 0 stays 0, 7 is odd so 7*3=21, sum=21, remainder 1, check 9.
	value, err := contract.Format(7)
	if err != nil {
		t.Fatal(err)
	}
	if value != "079" {
		t.Fatalf("value=%q", value)
	}
	// 00: sum=0, remainder 0, check 0 (X-TEA would have produced "0010").
	zero, err := contract.Format(0)
	if err != nil {
		t.Fatal(err)
	}
	if zero != "000" {
		t.Fatalf("zero=%q", zero)
	}
}

// TestFormatRefusesAValueWiderThanTheDigits checks that a value that does not
// fit is refused instead of printed wider than the declared width.
func TestFormatRefusesAValueWiderThanTheDigits(t *testing.T) {
	contract := numbering.Contract{Digits: 2}
	if _, err := contract.Format(100); !errors.Is(err, numbering.ErrNumbering) {
		t.Fatalf("err=%v", err)
	}
}

// TestContractWithoutAUsableWidthIsRefused checks that a width outside 1..9 or
// an unknown rule is refused.
func TestContractWithoutAUsableWidthIsRefused(t *testing.T) {
	cases := []numbering.Contract{
		{Digits: 0},
		{Digits: 10},
		{Digits: 4, CheckDigit: "luhn"},
		{Digits: 4, Overflow: "reset"},
	}
	for index, contract := range cases {
		if _, _, err := contract.Next(numbering.State{}); !errors.Is(err, numbering.ErrNumbering) {
			t.Fatalf("case %d err=%v", index, err)
		}
	}
}

// TestNegativeCounterIsRefused checks that a corrupt counter is reported.
func TestNegativeCounterIsRefused(t *testing.T) {
	contract := numbering.Contract{Digits: 4}
	if _, _, err := contract.Next(numbering.State{Current: -1}); !errors.Is(err, numbering.ErrNumbering) {
		t.Fatalf("err=%v", err)
	}
}

// TestNextValueReservesWithoutFormatting checks that a caller can reserve the
// counter before it knows the prefix, under the same overflow rule.
func TestNextValueReservesWithoutFormatting(t *testing.T) {
	contract := numbering.Contract{Digits: 3}
	value, state, err := contract.NextValue(numbering.State{Current: 41})
	if err != nil {
		t.Fatal(err)
	}
	if value != 42 || state.Current != 42 {
		t.Fatalf("value=%d state=%+v", value, state)
	}
	stop := numbering.Contract{Digits: 1}
	if _, _, err := stop.NextValue(numbering.State{Current: 9}); !errors.Is(err, numbering.ErrNumbering) {
		t.Fatalf("err=%v", err)
	}
}
