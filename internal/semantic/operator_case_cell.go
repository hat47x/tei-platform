package semantic

import (
	"fmt"
	"math/big"
)

// This file is the notation decision section 8.3 fixed for the cells of an
// example table: what a cell says depends on the type the operator's signature
// gives the input, and the signature is the only place that type comes from.
//
//	number  100, -0.01
//	date    2026-04-01
//	text    "A-100"
//	truth   true, false
//	any     any of the above; the quoting decides text from the rest
//
// The same notation is read here so a table, a Workspace check and an
// implementation check cannot read one cell two ways.

// OperatorCaseCellKind is the kind of value one cell of an example table
// states. It is decided by how the cell is written, never by the input it is
// written for: `any` accepts every kind and keeps the one that was written.
type OperatorCaseCellKind string

const (
	// OperatorCaseCellNumber is a decimal number.
	OperatorCaseCellNumber OperatorCaseCellKind = "number"
	// OperatorCaseCellDate is a calendar date written YYYY-MM-DD.
	OperatorCaseCellDate OperatorCaseCellKind = "date"
	// OperatorCaseCellText is a string written between double quotes.
	OperatorCaseCellText OperatorCaseCellKind = "text"
	// OperatorCaseCellTruth is the word true or the word false.
	OperatorCaseCellTruth OperatorCaseCellKind = "truth"
)

// OperatorCaseCell is one cell of an example table, read as the value it
// states. Exactly one of Number, Date, Text and Truth holds the value, decided
// by Kind. Text holds the string itself, without its quotes.
type OperatorCaseCell struct {
	Kind   OperatorCaseCellKind
	Text   string
	Number *big.Rat
	Date   CalendarDate
	Truth  bool
}

// ParseOperatorCaseCell reads one cell of an example table. It accepts the
// spellings section 8.3 lists and nothing else, so a cell that cannot be read
// as any of them is refused at the table instead of being approximated.
func ParseOperatorCaseCell(text string) (OperatorCaseCell, error) {
	if quoted, ok := unquoteCellText(text); ok {
		return OperatorCaseCell{Kind: OperatorCaseCellText, Text: quoted}, nil
	}
	switch text {
	case "true":
		return OperatorCaseCell{Kind: OperatorCaseCellTruth, Truth: true}, nil
	case "false":
		return OperatorCaseCell{Kind: OperatorCaseCellTruth}, nil
	}
	if day, err := ParseCalendarDate(text); err == nil {
		return OperatorCaseCell{Kind: OperatorCaseCellDate, Date: day}, nil
	}
	value, err := ParseOperatorCaseValue(text)
	if err != nil {
		return OperatorCaseCell{}, fmt.Errorf(
			"%q is not a decimal number, a date as YYYY-MM-DD, a text between double quotes, or the word true or false", text)
	}
	return OperatorCaseCell{Kind: OperatorCaseCellNumber, Number: value}, nil
}

// unquoteCellText reads a cell written between double quotes. The quotes must
// both be there and nothing may sit between them but the string itself, so a
// cell whose quotes do not close is refused rather than read as a shorter
// string.
func unquoteCellText(text string) (string, bool) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", false
	}
	inner := text[1 : len(text)-1]
	for i := 0; i < len(inner); i++ {
		if inner[i] == '"' {
			return "", false
		}
	}
	return inner, true
}

// Fits reports whether this cell can be the value of an input the declaration
// gave the type kind. An input of type any takes every kind and keeps the one
// that was written; an input of any other type takes only its own, so a cell
// written for another type is refused instead of being read as this one.
func (c OperatorCaseCell) Fits(kind OperatorInput) bool {
	switch kind {
	case OperatorInputAny:
		return true
	case OperatorInputNumber:
		return c.Kind == OperatorCaseCellNumber
	case OperatorInputDate:
		return c.Kind == OperatorCaseCellDate
	case OperatorInputText:
		return c.Kind == OperatorCaseCellText
	case OperatorInputTruth:
		return c.Kind == OperatorCaseCellTruth
	}
	// presence states whether a value is there, not a value, so no cell of an
	// example table can be one.
	return false
}

// CheckOperatorCaseCellFits reports whether one cell of an example table can
// be read as the value of an input of the declared type. The error names the
// input, the type and what may be written, so the author is sent to the cell
// rather than to the declaration.
func CheckOperatorCaseCellFits(text string, kind OperatorInput) error {
	cell, err := ParseOperatorCaseCell(text)
	if err != nil {
		return err
	}
	if cell.Fits(kind) {
		return nil
	}
	return fmt.Errorf("%q is not a value of type %s; %s", text, kind, operatorCaseCellWords(kind))
}

// operatorCaseCellWords says what a cell of an input of this type may hold.
func operatorCaseCellWords(kind OperatorInput) string {
	switch kind {
	case OperatorInputNumber:
		return "write a decimal number, for example 100 or -0.01"
	case OperatorInputDate:
		return "write a calendar date as YYYY-MM-DD, for example 2026-04-01"
	case OperatorInputText:
		return "write the text between double quotes, for example \"A-100\""
	case OperatorInputTruth:
		return "write the word true or the word false"
	case OperatorInputAny:
		return "write a decimal number, a date as YYYY-MM-DD, a text between double quotes, or the word true or false"
	}
	return "this input states whether a value is there, so an example table states no value for it"
}

// SameOperatorCaseCell reports whether two cells state the same value. Two
// numbers are the same number however they are written ("0" and "0.0"); two
// values of different kinds are never the same, so the number 100 and the text
// "100" are two values of an `any` input rather than one written twice.
func SameOperatorCaseCell(left, right OperatorCaseCell) bool {
	if left.Kind != right.Kind {
		return false
	}
	switch left.Kind {
	case OperatorCaseCellNumber:
		return left.Number.Cmp(right.Number) == 0
	case OperatorCaseCellDate:
		return left.Date == right.Date
	case OperatorCaseCellText:
		return left.Text == right.Text
	case OperatorCaseCellTruth:
		return left.Truth == right.Truth
	}
	return false
}
