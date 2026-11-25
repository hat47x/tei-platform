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
	// <!-- tei: cases of <operator> expecting <answer> over no rows -->
	//
	// Section 8.2: the empty relation is stated by the clause `over no rows`
	// in the mark, not by a table whose body the author left unwritten.
	noRows := saysOverNoRows(words)
	if (len(words) != 5 && !noRows) || words[0].Text != "cases" || words[1].Text != "of" || words[3].Text != "expecting" {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "unknown tei mark; the example of an operator that reads every row is marked <!-- tei: cases of <operator> expecting <answer> -->, and an empty relation adds the clause `over no rows`"}
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
	// Section 8.2: an empty relation is one example, written as this table's
	// header and delimiter with no body row and the mark's `over no rows`.
	// It is not read as a row that holds no value: the example states that
	// there are no rows at all. The clause and the body must agree: a table
	// with rows under the clause, and an empty table without it, are refused
	// here so neither is silently read as the other.
	//
	// Rows is made non-nil before the loop so an example with no rows is kept
	// as an empty collection rather than as "no example".
	if noRows {
		if len(table.Rows) > 0 {
			first := table.Rows[0]
			at := table.Mark
			if len(first) > 0 {
				at = first[0].Anchor
			}
			return out, &ParseError{Pos: at.Start, End: at.End,
				Msg: "the mark says `over no rows`, but the table states rows; write one or the other"}
		}
	} else if len(table.Rows) == 0 {
		return out, &ParseError{Pos: table.Mark.Start, End: table.Mark.End,
			Msg: "a row example needs at least one row; an empty relation is written in the mark with `over no rows`"}
	}
	names := make([]string, 0, len(columns))
	for name := range columns {
		names = append(names, name)
	}
	sortStrings(names)
	// The type of each column comes from the operator's declaration, so a cell
	// is read the way its input says (section 8.3). This file does not read the
	// declaration, so the values are kept as written here and read by the
	// Workspace, which knows the signature. What this file does is keep the
	// three states apart: a cell that holds no value, a cell nobody read, and
	// a cell that states a value.
	out.Case.Rows = make([]map[string]string, 0, len(table.Rows))
	for index, row := range table.Rows {
		values := map[string]string{}
		spans := map[string]Anchor{}
		for _, name := range names {
			cell := row[columns[name]]
			// A cell states a value of its input's type, the empty text (the
			// row holds no value), or the word `not-fetched` (the row was not
			// read). The three states are written, not inferred: an empty cell
			// is a value holder that is empty, and a row nobody read is said
			// so. A value of a text input is written with quotes, so the empty
			// text is not the same as an empty text value (section 8.3).
			switch cell.Text {
			case semantic.RowCellNoValue, semantic.RowCellNotFetched:
			default:
				if _, err := semantic.ParseOperatorRowCellValue(cell.Text); err != nil {
					return out, cellError(cell, fmt.Sprintf("input %q of row %d: %v", name, index+1, err))
				}
			}
			values[name] = cell.Text
			spans[name] = cell.Anchor
		}
		out.Case.Rows = append(out.Case.Rows, values)
		out.CellSpans = append(out.CellSpans, spans)
	}
	return out, nil
}
