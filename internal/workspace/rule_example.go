package workspace

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// RuleExample is one example table of a loaded Workspace: records of one Data
// and the answers the author expects rules to give for them.
//
// An example is not part of the World. It defines nothing, so adding,
// changing or removing an example leaves every definition and its fingerprint
// as it was. It is an input and an expectation, kept beside the World so the
// rules of that World can be checked against it.
type RuleExample struct {
	// File is the Workspace-relative path of the document, with slashes.
	File string
	Data core.CanonicalID
	// Items are the value columns, resolved to items of Data.
	Items []RuleExampleItem
	// Rows hold, per record, one cell per item.
	Rows         [][]RuleExampleCell
	Expectations []RuleExampleExpectation
}

// RuleExampleItem is a value column.
type RuleExampleItem struct {
	Attribute core.CanonicalID
	Symbol    string
	// Numeric is true when the item has a number type. Its cells were read as
	// exact decimals, and two cells with the same number are the same value
	// however they are written.
	Numeric bool
	// Date is true when the item is a calendar date. Its cells were read as
	// YYYY-MM-DD.
	Date bool
}

// RuleExampleCell is one value as written. An empty Text means the record
// holds no value for the item.
type RuleExampleCell struct {
	Text  string
	Range sourceindex.Range
}

// RuleExampleExpectation is one expected answer. Row is the index of the
// record it is about, or -1 when the rule is about all the rows.
type RuleExampleExpectation struct {
	Rule   core.CanonicalID
	Row    int
	Expect syntax.RuleExampleExpect
	// Range is where the expected answer is written.
	Range sourceindex.Range
}

// pendingRuleExample is an example table read during Load and not yet
// resolved against the Data and the rules it names.
type pendingRuleExample struct {
	filename string
	relative string
	table    syntax.RuleExampleAuthoring
}

func sourceRange(anchor syntax.Anchor) sourceindex.Range {
	return sourceindex.Range{
		Start: sourceindex.Position{Offset: anchor.Start.Offset, Line: anchor.Start.Line, Column: anchor.Start.Column},
		End:   sourceindex.Position{Offset: anchor.End.Offset, Line: anchor.End.Line, Column: anchor.End.Column},
	}
}

// resolveRuleExamples checks every example table against the loaded World and
// keeps the ones that name existing things. It runs after every source is
// read, because the Data and the rules may be declared in other documents.
//
// A table that names something that does not exist, or asks a rule about
// records the rule does not read, stops the Load at the place it is written.
// Whether a rule answers as expected is not decided here.
func resolveRuleExamples(r *Result) error {
	types := map[core.CanonicalID]semantic.Type{}
	data := map[core.CanonicalID]semantic.Data{}
	rules := map[core.CanonicalID]semantic.Validation{}
	relationTargets := map[core.CanonicalID]core.CanonicalID{}
	for _, namespace := range r.World.Semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
		for _, declared := range namespace.Data {
			data[declared.ID] = declared
			for _, relation := range declared.Relations {
				relationTargets[relation.ID] = relation.Target
			}
		}
		for _, rule := range namespace.Validations {
			rules[rule.ID] = rule
		}
	}

	for _, pending := range r.pendingRuleExamples {
		table := pending.table
		rejected := false
		reject := func(span syntax.Anchor, message string) error {
			rejected = true
			return r.collectSourceError(pending.filename, &syntax.ParseError{Pos: span.Start, End: span.End, Msg: message})
		}

		declared, known := data[table.Data]
		if !known {
			if err := reject(table.DataSpan, fmt.Sprintf("the example is of data %s, but no source declares that data", table.Data)); err != nil {
				return err
			}
			continue
		}
		example := RuleExample{File: pending.relative, Data: table.Data}
		itemOf := map[core.CanonicalID]bool{}
		for _, attribute := range declared.Attributes {
			itemOf[attribute.ID] = true
		}
		for _, header := range table.Items {
			var found *semantic.Attribute
			for i := range declared.Attributes {
				if declared.Attributes[i].Symbol == header.Text {
					found = &declared.Attributes[i]
				}
			}
			if found == nil {
				if err := reject(header.Anchor, fmt.Sprintf("data %s has no item %q", table.Data, header.Text)); err != nil {
					return err
				}
				continue
			}
			example.Items = append(example.Items, RuleExampleItem{
				Attribute: found.ID, Symbol: found.Symbol,
				Numeric: semantic.IsNumberType(types, found.TypeRef), Date: semantic.IsCalendarDateType(types, found.TypeRef),
			})
		}
		if rejected {
			continue
		}

		for _, row := range table.Rows {
			cells := make([]RuleExampleCell, 0, len(row))
			for i, cell := range row {
				// A value nobody read is written with the word `not-fetched`, and
				// section 8.3 says that writing does not depend on the type, so a
				// cell of any type's item may write it. It is not a number and
				// not a date, so both type checks are skipped for it rather than
				// reading it as a value it never was.
				unread := semantic.IsOperatorCaseNotFetchedWord(cell.Text)
				if cell.Text != "" && example.Items[i].Numeric && !unread {
					if _, err := semantic.ParseOperatorCaseValue(cell.Text); err != nil {
						if err := reject(cell.Anchor, fmt.Sprintf("item %s is a number: %v", example.Items[i].Symbol, err)); err != nil {
							return err
						}
					}
				}
				if cell.Text != "" && example.Items[i].Date && !unread {
					if _, err := semantic.ParseCalendarDate(cell.Text); err != nil {
						if err := reject(cell.Anchor, fmt.Sprintf("item %s is a date: %v", example.Items[i].Symbol, err)); err != nil {
							return err
						}
					}
				}
				cells = append(cells, RuleExampleCell{Text: cell.Text, Range: sourceRange(cell.Anchor)})
			}
			example.Rows = append(example.Rows, cells)
		}

		// A rule column is one statement about the rule. What is wrong with
		// it is reported once, not once for every row that uses it.
		refused := map[core.CanonicalID]bool{}
		for _, expectation := range table.Expectations {
			if refused[expectation.Rule] {
				continue
			}
			refused[expectation.Rule] = true
			rule, known := rules[expectation.Rule]
			if !known {
				if err := reject(expectation.RuleSpan, fmt.Sprintf("the example expects an answer of rule %s, but no source declares that rule", expectation.Rule)); err != nil {
					return err
				}
				continue
			}
			// A rule is about all the rows together when it reads the rows of a
			// relation: the uniqueness rules compare rows with each other, and an
			// operator rule whose input reads every row answers about the rows
			// that relation holds. Either way the answer is one answer about the
			// rows, so it is written in the mark and not in a column, because a
			// column states one answer per record.
			aboutAllRows := rule.Kind == semantic.ValidationRelationUnique ||
				rule.Kind == semantic.ValidationNoOverlap ||
				(rule.Kind == semantic.ValidationOperator && rule.Relation != "")
			switch {
			case aboutAllRows && expectation.Row >= 0:
				if err := reject(expectation.RuleSpan, fmt.Sprintf("rule %s is about all the rows together; write it in the mark after expecting, not as a column", rule.ID)); err != nil {
					return err
				}
				continue
			case !aboutAllRows && expectation.Row < 0:
				if err := reject(expectation.RuleSpan, fmt.Sprintf("rule %s is about one record; write it as a column, not in the mark", rule.ID)); err != nil {
					return err
				}
				continue
			}
			if !itemOf[rule.Attribute] {
				if err := reject(expectation.RuleSpan, fmt.Sprintf("rule %s reads %s, which is not an item of data %s", rule.ID, rule.Attribute, table.Data)); err != nil {
					return err
				}
				continue
			}
			if rule.Compared != "" && !itemOf[rule.Compared] {
				if err := reject(expectation.RuleSpan, fmt.Sprintf("rule %s reads %s, which is not an item of data %s", rule.ID, rule.Compared, table.Data)); err != nil {
					return err
				}
				continue
			}
			if rule.GroupBy != "" && !itemOf[rule.GroupBy] {
				if err := reject(expectation.RuleSpan, fmt.Sprintf("rule %s reads %s, which is not an item of data %s", rule.ID, rule.GroupBy, table.Data)); err != nil {
					return err
				}
				continue
			}
			if aboutAllRows && relationTargets[rule.Relation] != table.Data {
				if err := reject(expectation.RuleSpan, fmt.Sprintf("rule %s is about the rows of relation %s, which are not records of data %s", rule.ID, rule.Relation, table.Data)); err != nil {
					return err
				}
				continue
			}
			refused[expectation.Rule] = false
			example.Expectations = append(example.Expectations, RuleExampleExpectation{
				Rule: expectation.Rule, Row: expectation.Row, Expect: expectation.Expect, Range: sourceRange(expectation.ExpectSpan),
			})
		}
		if !rejected {
			r.RuleExamples = append(r.RuleExamples, example)
		}
	}
	r.pendingRuleExamples = nil
	return nil
}
