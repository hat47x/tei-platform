package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file reads a row-set example of an operator, written as a table next to
// its declaration:
//
//	<!-- tei: cases of sales.every_line_at_most expecting not-satisfied -->
//	| value | limit |
//	| --- | --- |
//	| 2 | 10 |
//	| 5 | 4 |
//
// One table is one example. The columns are the operator's declared input names
// and the body rows are the rows of one relation: the operator reads all of
// them at once, so an example has to state a set of rows rather than one value.
// The expected answer is written once, in the mark, because it belongs to the
// set and not to any one row.
//
// The first change keeps to examples whose values are all present. A blank cell
// is refused instead of being read as zero or as "no value", and an answer the
// reader does not know is refused instead of being read as one of the two.

// parseRowSetOperatorCases reads <!-- tei: cases of <operator> expecting <answer> -->.
func parseRowSetOperatorCases(table MarkdownTable, words []MarkdownCell) (OperatorCasesAuthoring, error) {
	var out OperatorCasesAuthoring
	if len(words) != 5 || words[3].Text != "expecting" {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a row-set example is marked <!-- tei: cases of <operator> expecting <answer> -->"}
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

	expect := words[4]
	switch expect.Text {
	case semantic.RowExpectSatisfied, semantic.RowExpectNotSatisfied:
	case "":
		return out, cellError(expect, "the mark states no expected answer; write satisfied or not-satisfied")
	default:
		return out, cellError(expect, fmt.Sprintf(
			"expecting is written satisfied or not-satisfied, not %q", expect.Text))
	}

	// The columns are the operator's input names, in whatever order the table
	// writes them. A `value` column is not the one-value form here: a row-set
	// operator's first input may be named `value`, and its cell states the
	// value of that input in this row, not an example's single value.
	columns := map[string]int{}
	out.InputHeaderSpans = map[string]Anchor{}
	for i, cell := range table.Header {
		if cell.Text == "" {
			return out, cellError(cell, "a column header is empty; name one of the operator's inputs")
		}
		if cell.Text == "expect" || cell.Text == "note" {
			return out, cellError(cell, fmt.Sprintf(
				"a row-set example states its answer once in the mark, not in a %q column; remove the column", cell.Text))
		}
		if _, repeated := columns[cell.Text]; repeated {
			return out, cellError(cell, fmt.Sprintf("column %q appears more than once", cell.Text))
		}
		columns[cell.Text] = i
		out.InputHeaderSpans[cell.Text] = cell.Anchor
	}
	if len(columns) == 0 {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a row-set example needs one column per declared input"}
	}

	// Read the body rows into one example. An empty cell is refused here rather
	// than carried as a value: a row whose value is absent says something this
	// reader does not yet state, and reading it as zero would make an example
	// the author never wrote.
	example := semantic.OperatorCase{RowExpect: expect.Text}
	example.Rows = make([]map[string]string, 0, len(table.Rows))
	spans := make([][]Anchor, 0, len(table.Rows))
	for _, row := range table.Rows {
		stated := make(map[string]string, len(columns))
		rowSpans := make([]Anchor, 0, len(columns))
		for name, index := range columns {
			cell := row[index]
			if cell.Text == "" {
				return out, cellError(cell, fmt.Sprintf(
					"the cell for input %q is empty; every row states a value for every input it names, and an absent value is not written as blank", name))
			}
			if _, err := semantic.ParseOperatorCaseValue(cell.Text); err != nil {
				return out, cellError(cell, err.Error())
			}
			stated[name] = cell.Text
			rowSpans = append(rowSpans, cell.Anchor)
		}
		example.Rows = append(example.Rows, stated)
		spans = append(spans, rowSpans)
	}

	// One table is one example, so two row-set tables for one operator are two
	// examples and stay in Cases. Whether they state the same example is
	// decided where the operator is known: this reader has no signature.
	out.Cases = append(out.Cases, OperatorCaseAuthoring{
		Value: example, ValueSpan: expect.Anchor, ExpectSpan: expect.Anchor, RowSpans: spans,
	})
	return out, nil
}
