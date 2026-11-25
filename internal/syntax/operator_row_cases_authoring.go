package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file reads the table that states a row-scoped operator's example.
//
// Decision section 8.1: the answer of a row-scoped operator is not written per
// row but in the mark, the same way the record example table writes it:
//
//	<!-- tei: cases of sales.every_line_at_most expecting satisfied -->
//	| value | limit |
//	| --- | --- |
//	| 2 | 10 |
//	| 5 | 5 |
//
// The columns are the operator's input names. The same operator may have
// several such tables, one example each; the old `cases of <operator>` table
// with its `expect` column stays what it was, one row per example, and mixing
// the two forms for one operator is refused.
//
// The first change reads tables whose numbers are all present and whose
// answer is satisfied or not-satisfied. An empty cell and the `undecided`
// answer belong to the change after this one (section 8.1), so they are
// refused here rather than accepted and left uncheckable.

// OperatorRowCasesAuthoring is one parsed row-example table: one example.
type OperatorRowCasesAuthoring struct {
	Operator     core.CanonicalID
	OperatorSpan Anchor
	// Case is the one example this table states.
	Case semantic.OperatorRowCase
	// AnswerSpan is the cell in the mark that states the answer.
	AnswerSpan Anchor
	// InputHeaderSpans holds the header cell of each input column, keyed by
	// the name the header gave, so a column that names no declared input is
	// reported at its own cell.
	InputHeaderSpans map[string]Anchor
	// CellSpans holds the span of each row's cells, keyed by row then name.
	CellSpans []map[string]Anchor
}

// ParseOperatorRowCasesTable reads a marked table as one example of an
// operator that reads every row.
func ParseOperatorRowCasesTable(table MarkdownTable) (OperatorRowCasesAuthoring, error) {
	var out OperatorRowCasesAuthoring
	words := table.Words
	// <!-- tei: cases of <operator> expecting <answer> -->
	if len(words) != 5 || words[0].Text != "cases" || words[1].Text != "of" || words[3].Text != "expecting" {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "unknown tei mark; the example of an operator that reads every row is marked <!-- tei: cases of <operator> expecting <answer> -->"}
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

	answerCell := words[4]
	answer := semantic.OperatorRowExpect(answerCell.Text)
	if !answer.Valid() {
		return out, cellError(answerCell, fmt.Sprintf(
			"expecting is written satisfied, not-satisfied or undecided, not %q", answerCell.Text))
	}
	out.Case.Answer, out.AnswerSpan = answer, answerCell.Anchor

	header := table.Header
	if len(header) == 0 {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a row example needs at least one input column"}
	}
	out.InputHeaderSpans = map[string]Anchor{}
	columns := map[string]int{}
	for i, cell := range header {
		if cell.Text == "" {
			return out, cellError(cell, "a column header is empty; name one of the operator's inputs")
		}
		if cell.Text == "expect" {
			return out, cellError(cell,
				"a row example has no expect column; the answer is in the mark, because it is about the rows together")
		}
		if _, duplicate := columns[cell.Text]; duplicate {
			return out, cellError(cell, fmt.Sprintf("column %q appears more than once", cell.Text))
		}
		if _, err := core.ParseLocalName(cell.Text); err != nil {
			return out, cellError(cell, fmt.Sprintf("column %q is not a usable input name", cell.Text))
		}
		columns[cell.Text] = i
		out.InputHeaderSpans[cell.Text] = cell.Anchor
	}
	if len(table.Rows) == 0 {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a row example needs at least one row; an empty table is not an empty collection"}
	}

	names := make([]string, 0, len(columns))
	for name := range columns {
		names = append(names, name)
	}
	sortStrings(names)
	for index, row := range table.Rows {
		values := map[string]string{}
		spans := map[string]Anchor{}
		for _, name := range names {
			cell := row[columns[name]]
			// An empty cell states that the row holds no value; the word
			// `not-fetched` states that the value was not read. They are kept
			// apart from a number and from each other (section 8.1). A missing
			// column is never used as a stand-in for "not read".
			state, text, err := semantic.ReadOperatorRowCell(cell.Text)
			if err != nil {
				return out, cellError(cell, fmt.Sprintf("input %q of row %d: %v", name, index+1, err))
			}
			if state == semantic.OperatorRowCellNoValue {
				values[name] = semantic.OperatorRowNoValue
			} else if state == semantic.OperatorRowCellNotFetched {
				values[name] = semantic.OperatorRowNotFetched
			} else {
				values[name] = text
			}
			spans[name] = cell.Anchor
		}
		out.Case.Rows = append(out.Case.Rows, values)
		out.CellSpans = append(out.CellSpans, spans)
	}
	return out, nil
}
