package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
)

// This file reads example records and what rules answer for them:
//
//	<!-- tei: example of sales.order_line expecting sales.order.lines.line_number_unique not-satisfied -->
//	| lineNumber | quantity | sales.order_line.quantity_positive |
//	| --- | --- | --- |
//	| 1 | 2 | satisfied |
//	| 1 | 0 | not-satisfied |
//
// One row is one record of the Data the mark names. A column whose header is
// an item name holds values. A column whose header is a qualified rule id
// holds what that rule answers for the record of the row. A rule about all
// the rows together is written in the mark, after "expecting".
//
// The table is not a definition and holds no records of the running system.
// It states what the author expects, so a rule that answers differently is
// found where the expectation is written.
//
// An empty value cell means the record holds no value for the item. It is
// not zero and not an empty text. An empty cell in a rule column means the
// author states nothing about that rule for that row.

// RuleExampleExpect is an expected answer of a rule.
type RuleExampleExpect string

const (
	RuleExampleSatisfied    RuleExampleExpect = "satisfied"
	RuleExampleNotSatisfied RuleExampleExpect = "not-satisfied"
	// RuleExampleUndecided: the rule cannot be decided from the record, for
	// instance because the value it reads is empty.
	RuleExampleUndecided RuleExampleExpect = "undecided"
)

// RuleExampleExpectation is one expected answer and where it is written.
// Row is the index of the record it is about, or -1 for all the rows.
type RuleExampleExpectation struct {
	Rule       core.CanonicalID
	RuleSpan   Anchor
	Row        int
	Expect     RuleExampleExpect
	ExpectSpan Anchor
}

// RuleExampleAuthoring is one parsed example table. Whether the Data, its
// items and the rules exist is decided against the whole Workspace, not here.
type RuleExampleAuthoring struct {
	Mark     Anchor
	Data     core.CanonicalID
	DataSpan Anchor
	// Items are the headers of the value columns, in table order.
	Items []MarkdownCell
	// Rows hold, per record, one cell per item.
	Rows         [][]MarkdownCell
	Expectations []RuleExampleExpectation
}

func parseRuleExampleExpect(cell MarkdownCell) (RuleExampleExpect, error) {
	switch RuleExampleExpect(cell.Text) {
	case RuleExampleSatisfied, RuleExampleNotSatisfied, RuleExampleUndecided:
		return RuleExampleExpect(cell.Text), nil
	}
	return "", cellError(cell, fmt.Sprintf("an expected answer is written satisfied, not-satisfied or undecided, not %q", cell.Text))
}

func parseRuleExampleRule(cell MarkdownCell) (core.CanonicalID, error) {
	if !strings.Contains(cell.Text, ".") {
		return "", cellError(cell, fmt.Sprintf("rule %q is not a qualified canonical id; write it with its namespace", cell.Text))
	}
	id, err := core.ParseCanonicalID(cell.Text)
	if err != nil {
		return "", cellError(cell, err.Error())
	}
	return id, nil
}

// ParseRuleExampleTable reads a marked table as example records.
func ParseRuleExampleTable(table MarkdownTable) (RuleExampleAuthoring, error) {
	out := RuleExampleAuthoring{Mark: table.Mark}
	markError := func(message string) error {
		return &ParseError{Pos: table.Mark.Start, End: table.Mark.End, Msg: message}
	}
	words := table.Words
	if len(words) < 3 || words[0].Text != "example" || words[1].Text != "of" {
		return out, markError("unknown tei mark; example records are marked <!-- tei: example of <data> --> or <!-- tei: example of <data> expecting <rule> <answer> -->")
	}
	reference := words[2]
	if !strings.Contains(reference.Text, ".") {
		return out, cellError(reference, fmt.Sprintf("data %q is not a qualified canonical id; write it with its namespace", reference.Text))
	}
	data, err := core.ParseCanonicalID(reference.Text)
	if err != nil {
		return out, cellError(reference, err.Error())
	}
	out.Data, out.DataSpan = data, reference.Anchor

	stated := map[core.CanonicalID]bool{}
	if len(words) > 3 {
		if words[3].Text != "expecting" {
			return out, cellError(words[3], fmt.Sprintf("after the data, write expecting <rule> <answer>, not %q", words[3].Text))
		}
		pairs := words[4:]
		if len(pairs) == 0 || len(pairs)%2 != 0 {
			return out, markError("after expecting, write pairs of a rule and its answer")
		}
		for i := 0; i < len(pairs); i += 2 {
			rule, err := parseRuleExampleRule(pairs[i])
			if err != nil {
				return out, err
			}
			expect, err := parseRuleExampleExpect(pairs[i+1])
			if err != nil {
				return out, err
			}
			if stated[rule] {
				return out, cellError(pairs[i], fmt.Sprintf("rule %s is expected more than once; one rule has one expected answer", rule))
			}
			stated[rule] = true
			out.Expectations = append(out.Expectations, RuleExampleExpectation{
				Rule: rule, RuleSpan: pairs[i].Anchor, Row: -1, Expect: expect, ExpectSpan: pairs[i+1].Anchor,
			})
		}
	}

	// A header with a namespace is a rule; a header without one is an item.
	type ruleColumn struct {
		index int
		rule  core.CanonicalID
		span  Anchor
	}
	var itemIndexes []int
	var ruleColumns []ruleColumn
	seenItems := map[string]bool{}
	for i, header := range table.Header {
		if header.Text == "" {
			return out, cellError(header, "a column has no header; write an item name or a rule")
		}
		if strings.Contains(header.Text, ".") {
			rule, err := parseRuleExampleRule(header)
			if err != nil {
				return out, err
			}
			if stated[rule] {
				return out, cellError(header, fmt.Sprintf("rule %s is expected more than once; one rule has one place for its expected answers", rule))
			}
			stated[rule] = true
			ruleColumns = append(ruleColumns, ruleColumn{index: i, rule: rule, span: header.Anchor})
			continue
		}
		if seenItems[header.Text] {
			return out, cellError(header, fmt.Sprintf("item %q appears more than once", header.Text))
		}
		seenItems[header.Text] = true
		itemIndexes = append(itemIndexes, i)
		out.Items = append(out.Items, header)
	}
	if len(out.Items) == 0 {
		return out, markError("an example needs at least one item column")
	}

	for rowIndex, row := range table.Rows {
		cells := make([]MarkdownCell, 0, len(itemIndexes))
		for _, i := range itemIndexes {
			cells = append(cells, row[i])
		}
		out.Rows = append(out.Rows, cells)
		for _, column := range ruleColumns {
			cell := row[column.index]
			if cell.Text == "" {
				continue
			}
			expect, err := parseRuleExampleExpect(cell)
			if err != nil {
				return out, err
			}
			out.Expectations = append(out.Expectations, RuleExampleExpectation{
				Rule: column.rule, RuleSpan: column.span, Row: rowIndex, Expect: expect, ExpectSpan: cell.Anchor,
			})
		}
	}
	for _, column := range ruleColumns {
		used := false
		for _, expectation := range out.Expectations {
			used = used || (expectation.Row >= 0 && expectation.Rule == column.rule)
		}
		if !used {
			return out, &ParseError{Pos: column.span.Start, End: column.span.End,
				Msg: fmt.Sprintf("the column of rule %s states no expected answer in any row", column.rule)}
		}
	}
	if len(out.Expectations) == 0 {
		return out, markError("the example states no expected answer; add a rule column, or expecting <rule> <answer> in the mark")
	}
	return out, nil
}
