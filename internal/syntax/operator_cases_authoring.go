package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file reads the examples of an operator, written as a table next to its
// declaration:
//
//	<!-- tei: cases of sales.non_negative -->
//	| value | expect | note |
//	| --- | --- | --- |
//	| 300 | satisfied | |
//	| 0 | satisfied | 値引きなしは正しい入力 |
//	| -0.01 | not-satisfied | |
//
// One row is one example: for this value, the operator answers this. The
// examples are part of the contract. They are written by the person who knows
// what the operator means, not derived from an implementation, and an
// implementation that disagrees with one of them is not used.
//
// The note column is documentation and is not kept.

// OperatorCaseAuthoring is one row and the spans of its cells.
type OperatorCaseAuthoring struct {
	Value      semantic.OperatorCase
	ValueSpan  Anchor
	ExpectSpan Anchor
	// InputSpans holds the span of each named input's cell, keyed by the name
	// the header gave. A single-input row leaves it empty: its one value is
	// named by the fixed `value` column, whose span is ValueSpan.
	InputSpans map[string]Anchor
}

// OperatorCasesAuthoring is one parsed case table. Operator is the authored
// reference; whether that operator is declared is decided against the whole
// Workspace, not here.
type OperatorCasesAuthoring struct {
	Operator     core.CanonicalID
	OperatorSpan Anchor
	Cases        []OperatorCaseAuthoring
	// InputHeaderSpans holds the header cell of each input column, keyed by the
	// name the header gave. A column that names no declared input is a mistake
	// in the header, so the diagnostic points there rather than at a body cell.
	InputHeaderSpans map[string]Anchor
}

// ParseOperatorCasesTable reads a marked table as the examples of an operator.
func ParseOperatorCasesTable(table MarkdownTable) (OperatorCasesAuthoring, error) {
	var out OperatorCasesAuthoring
	words := table.Words
	if len(words) != 3 || words[0].Text != "cases" || words[1].Text != "of" {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "unknown tei mark; the examples of an operator are marked <!-- tei: cases of <operator> -->"}
	}
	reference := words[2]
	if !strings.Contains(reference.Text, ".") {
		return out, cellError(reference, fmt.Sprintf("operator %q is not a qualified canonical id; write it with its namespace", reference.Text))
	}
	operator, err := core.ParseCanonicalID(reference.Text)
	if err != nil {
		return out, cellError(reference, err.Error())
	}
	out.Operator, out.OperatorSpan = operator, reference.Anchor

	// A table states either one value, named by the fixed `value` column, or one
	// value per declared input, named by the input's own name. Which form it is
	// depends on the operator's signature, which this file does not know: it
	// reads a `value` column as the single-value form and every other column as
	// an input name, and leaves the signature check to the Workspace.
	header := table.Header
	expectIndex, hasExpect := columnIndex(header, "expect")
	if !hasExpect {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a case table needs the column \"expect\""}
	}
	valueIndex, singleValue := columnIndex(header, "value")
	columns := map[string]int{}
	out.InputHeaderSpans = map[string]Anchor{}
	seenHeader := map[string]bool{}
	for i, cell := range header {
		if cell.Text == "" {
			return out, cellError(cell, "a column header is empty; name the value column or one of the operator's inputs")
		}
		if seenHeader[cell.Text] {
			return out, cellError(cell, fmt.Sprintf("column %q appears more than once", cell.Text))
		}
		seenHeader[cell.Text] = true
		switch cell.Text {
		case "expect", "note", "value":
			continue
		}
		columns[cell.Text] = i
		out.InputHeaderSpans[cell.Text] = cell.Anchor
	}
	if singleValue && len(columns) > 0 {
		// The first input of a multi-input operator may itself be named "value"
		// (the design's within(value, low, high) is). A `value` column beside
		// other input columns therefore names that input rather than selecting
		// the single-value form.
		columns["value"] = valueIndex
		singleValue = false
	}
	if !singleValue && len(columns) == 0 {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a case table needs a value column or one column per declared input"}
	}

	for _, row := range table.Rows {
		expectCell := row[expectIndex]
		satisfied, expectErr := caseExpectation(expectCell)
		if expectErr != nil {
			return out, expectErr
		}
		var example semantic.OperatorCase
		var valueSpan Anchor
		inputSpans := map[string]Anchor{}
		repeatedAt := valueSpan
		if singleValue {
			valueCell := row[valueIndex]
			if valueCell.Text == "" {
				return out, cellError(valueCell, "the value cell is empty; an example without a value is not read as zero or as \"no value\"")
			}
			// The single-value form names the one input by its own name, and
			// that input may be a presence input, whose cell states one of two
			// words rather than a decimal. The word is carried here and read
			// against the input's type at the Workspace, where the signature is
			// known.
			if semantic.IsOperatorCasePresenceWord(valueCell.Text) {
				example = semantic.OperatorCase{Inputs: map[string]string{"value": valueCell.Text}, Satisfied: satisfied}
				inputSpans["value"] = valueCell.Anchor
				// The repeated example is reported at the cell the author wrote,
				// so the span is carried here as it is for the multi-input form.
				repeatedAt = valueCell.Anchor
			} else if semantic.IsOperatorCaseNotFetchedWord(valueCell.Text) {
				// Section 8.3: the writing for "not read" does not depend on the
				// input's type, so a cell of this form may write the same word
				// the row-set form writes. It is not a decimal, so it is carried
				// as written and read against the input's type at the Workspace.
				example = semantic.OperatorCase{Inputs: map[string]string{"value": valueCell.Text}, Satisfied: satisfied}
				inputSpans["value"] = valueCell.Anchor
				repeatedAt = valueCell.Anchor
			} else {
				if _, err := semantic.ParseOperatorCaseValue(valueCell.Text); err != nil {
					return out, cellError(valueCell, err.Error())
				}
				example = semantic.OperatorCase{Value: valueCell.Text, Satisfied: satisfied}
				valueSpan = valueCell.Anchor
				repeatedAt = valueCell.Anchor
			}
		} else {
			example = semantic.OperatorCase{Inputs: map[string]string{}, Satisfied: satisfied}
			names := make([]string, 0, len(columns))
			for name := range columns {
				names = append(names, name)
			}
			sortStrings(names)
			for _, name := range names {
				cell := row[columns[name]]
				if cell.Text == "" {
					return out, cellError(cell, fmt.Sprintf("the cell for input %q is empty; every input the table names states a value, and an unread value is not written as blank", name))
				}
				// A presence input reads whether the attribute holds a value,
				// not the value itself, so its cell states one of two words
				// rather than a decimal. A number input keeps the decimal
				// spelling, and a word in a number cell is refused at the
				// Workspace, where the signature is known.
				if semantic.IsOperatorCasePresenceWord(cell.Text) {
					example.Inputs[name] = cell.Text
					inputSpans[name] = cell.Anchor
					continue
				}
				// Section 8.3: the writing for "not read" does not depend on the
				// input's type, so a cell of this form may write the same word
				// the row-set form writes. It is not a decimal, so it is carried
				// as written and read against the input's type at the Workspace.
				if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
					example.Inputs[name] = cell.Text
					inputSpans[name] = cell.Anchor
					continue
				}
				if _, err := semantic.ParseOperatorCaseValue(cell.Text); err != nil {
					return out, cellError(cell, err.Error())
				}
				example.Inputs[name] = cell.Text
				inputSpans[name] = cell.Anchor
			}
		}
		if !singleValue {
			names := make([]string, 0, len(columns))
			for name := range columns {
				names = append(names, name)
			}
			sortStrings(names)
			repeatedAt = inputSpans[names[0]]
		}
		for _, prior := range out.Cases {
			if semantic.SameOperatorCaseStates(prior.Value, example) {
				return out, &ParseError{Pos: repeatedAt.Start, End: repeatedAt.End,
					Msg: fmt.Sprintf("the same example is written more than once (the first is at %d:%d); one example has one expected result",
						prior.ExpectSpan.Start.Line, prior.ExpectSpan.Start.Column)}
			}
		}
		out.Cases = append(out.Cases, OperatorCaseAuthoring{
			Value: example, ValueSpan: valueSpan, ExpectSpan: expectCell.Anchor, InputSpans: inputSpans,
		})
	}
	return out, nil
}

// columnIndex returns the index of one header word, if present.
func columnIndex(header []MarkdownCell, text string) (int, bool) {
	for i, cell := range header {
		if cell.Text == text {
			return i, true
		}
	}
	return 0, false
}

// caseExpectation reads the expected result of one row.
func caseExpectation(cell MarkdownCell) (bool, error) {
	switch cell.Text {
	case "satisfied":
		return true, nil
	case "not-satisfied":
		return false, nil
	case "":
		return false, cellError(cell, "the example has no expected result; write satisfied or not-satisfied")
	default:
		return false, cellError(cell, fmt.Sprintf("expect is written satisfied or not-satisfied, not %q", cell.Text))
	}
}

// sortStrings orders names so a diagnostic lists them the same way every time.
func sortStrings(names []string) {
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
}
