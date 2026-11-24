package validationeval

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides an operator whose inputs read every row of a relation.
//
// The values of one such rule are not the values of one record: each input is
// a column, and the answer may depend on a row that was not read. The
// implementation receives the states of the rows (NumberReading already carries
// present / null / not-fetched) and answers Tri. The evaluator is the only
// place that turns that answer, the read Coverage and the operator's
// `refutes on some rows` statement into a Decision, because the same three
// facts must compose the same way wherever they are composed (#4359,
// decision section 15.1 and 15.2).
//
// An input read over all the rows is read once for the whole relation, so its
// rows and the subject's rows come from one read attempt. A constant input
// needs no rows: it is the same value whatever was read, so it is not passed
// through the rows map.
func decideRowScopedOperator(rule semantic.Validation, contract semantic.Operator, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	rowsSource, readsRows := source.(StatefulRowNumberSource)
	if !readsRows {
		return Decision{}, fmt.Errorf("semantic validation %s reads the rows of a relation, which this reader cannot read", rule.ID)
	}
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf("semantic validation %s declares a row-scoped operator %s but names no relation; the rows it reads are unknown", rule.ID, rule.Operator)
	}
	inputs := contractInputs(contract)
	rows := make(map[string][]NumberReading, len(inputs))
	var (
		coverage Coverage
		stated   bool
		reason   *Undecided
	)
	for index, input := range inputs {
		attribute, _ := rowInputAttribute(rule, input, index)
		if attribute == "" {
			// A constant input: it is the same value for every row, so it is
			// not part of the rows the implementation reads.
			continue
		}
		reading, err := rowsSource.ReadRelationNumbers(rule.Relation, attribute)
		if err != nil {
			return Decision{}, fmt.Errorf("semantic validation %s cannot read %s over relation %s: %w", rule.ID, attribute, rule.Relation, err)
		}
		for rowIndex, row := range reading.Rows {
			switch row.State {
			case ValuePresent:
				if row.Value == nil {
					return Decision{}, fmt.Errorf("semantic validation %s: the reader reports %s of row %d as present without a value", rule.ID, attribute, rowIndex)
				}
			case ValueNull, ValueNotFetched:
				if row.Value != nil {
					return Decision{}, fmt.Errorf("semantic validation %s: the reader reports %s of row %d as %s together with a value", rule.ID, attribute, rowIndex, row.State)
				}
			default:
				return Decision{}, fmt.Errorf("semantic validation %s: the reader reports %s of row %d in the unknown state %q", rule.ID, attribute, rowIndex, row.State)
			}
		}
		// A value the row certainly does not hold is read under the mapping the
		// declaration states for that input, exactly as a per-record input is.
		// An unread value is not: nothing is known about it, so no mapping
		// applies and the row stays unread.
		if input.AbsentAs != "" && implementation.Absent != nil {
			for i := range reading.Rows {
				if reading.Rows[i].State != ValueNull {
					continue
				}
				mapped, ok := implementation.Absent(input.AbsentAs)
				if !ok {
					continue
				}
				reading.Rows[i] = NumberReading{State: ValuePresent, Value: new(big.Rat).Set(mapped)}
			}
		}
		if !stated {
			coverage, stated = reading.Coverage, true
		} else if reading.Coverage != coverage {
			// Two inputs of one rule read from one relation must agree on
			// whether the rows are all there. Otherwise the answer would depend
			// on which input happened to be read first.
			return Decision{}, fmt.Errorf("semantic validation %s reads %s and another input of the same relation with different coverage (%s and %s)",
				rule.ID, attribute, coverage, reading.Coverage)
		}
		rows[input.Name] = reading.Rows
		for _, row := range reading.Rows {
			switch row.State {
			case ValueNull:
				if reason == nil {
					reason = &Undecided{Reason: UndecidedValueNull, Relation: rule.Relation, Attribute: attribute}
				}
			case ValueNotFetched:
				reason = &Undecided{Reason: UndecidedValueNotFetched, Relation: rule.Relation, Attribute: attribute}
			}
		}
	}
	if !stated {
		return Decision{}, fmt.Errorf("semantic validation %s declares a row-scoped operator %s but reads no attribute of relation %s; no rows were read", rule.ID, rule.Operator, rule.Relation)
	}
	answer, err := implementation.NumberRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, reason)
}

// rowInputAttribute returns the attribute one row-scoped input reads. The
// subject supplies the first input from the attribute the rule constrains; the
// others are assigned by name. An assignment whose value is a constant yields
// an empty attribute: such an input has no rows.
func rowInputAttribute(rule semantic.Validation, input semantic.OperatorInputDeclaration, index int) (core.CanonicalID, bool) {
	if index == 0 {
		return rule.Attribute, true
	}
	assignment := findInputAssignment(rule, input.Name)
	if assignment == nil {
		return "", false
	}
	if !isAttributeReference(assignment.Value) {
		return "", false
	}
	attribute, err := core.ParseCanonicalID(assignment.Value)
	if err != nil {
		return "", false
	}
	return attribute, true
}
