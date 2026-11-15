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
}

// OperatorCasesAuthoring is one parsed case table. Operator is the authored
// reference; whether that operator is declared is decided against the whole
// Workspace, not here.
type OperatorCasesAuthoring struct {
	Operator     core.CanonicalID
	OperatorSpan Anchor
	Cases        []OperatorCaseAuthoring
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

	columns := map[string]int{}
	for i, header := range table.Header {
		switch header.Text {
		case "value", "expect", "note":
		default:
			return out, cellError(header, fmt.Sprintf("unknown column %q; a case table has the columns value, expect, note", header.Text))
		}
		if _, duplicate := columns[header.Text]; duplicate {
			return out, cellError(header, fmt.Sprintf("column %q appears more than once", header.Text))
		}
		columns[header.Text] = i
	}
	for _, required := range []string{"value", "expect"} {
		if _, present := columns[required]; !present {
			return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
				Msg: fmt.Sprintf("a case table needs the column %q", required)}
		}
	}

	for _, row := range table.Rows {
		valueCell, expectCell := row[columns["value"]], row[columns["expect"]]
		if valueCell.Text == "" {
			return out, cellError(valueCell, "the value cell is empty; an example without a value is not read as zero or as \"no value\"")
		}
		if _, err := semantic.ParseOperatorCaseValue(valueCell.Text); err != nil {
			return out, cellError(valueCell, err.Error())
		}
		var satisfied bool
		switch expectCell.Text {
		case "satisfied":
			satisfied = true
		case "not-satisfied":
		case "":
			return out, cellError(expectCell, fmt.Sprintf("the example for %s has no expected result; write satisfied or not-satisfied", valueCell.Text))
		default:
			return out, cellError(expectCell, fmt.Sprintf("expect is written satisfied or not-satisfied, not %q", expectCell.Text))
		}
		example := semantic.OperatorCase{Value: valueCell.Text, Satisfied: satisfied}
		for _, prior := range out.Cases {
			if semantic.SameOperatorCaseValue(prior.Value.Value, example.Value) {
				return out, cellError(valueCell, fmt.Sprintf("the value %s is written more than once (first as %s at %d:%d); one value has one expected result",
					valueCell.Text, prior.Value.Value, prior.ValueSpan.Start.Line, prior.ValueSpan.Start.Column))
			}
		}
		out.Cases = append(out.Cases, OperatorCaseAuthoring{Value: example, ValueSpan: valueCell.Anchor, ExpectSpan: expectCell.Anchor})
	}
	return out, nil
}
