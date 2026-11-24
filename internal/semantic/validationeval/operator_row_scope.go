package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file connects a declaration whose input reads every row to the one
// composition in row_scope.go.
//
// A rule with such an input is asked of the rows of the relation, not of the
// subject's one row (decision section 15.4). The values of every input are read
// once each, for all the rows, so every input of one rule comes from one reading
// (section 15.1). The implementation is then called once with the states of the
// rows that were read and answers TriYes, TriNo or TriMaybe; whether a partly
// read relation may still be answered not-satisfied is the operator's own
// statement, applied in decideRowScoped and nowhere else.

// declaresRowScopedInput reports whether some declared input reads every row.
func declaresRowScopedInput(contract semantic.Operator) bool {
	for _, input := range contractInputs(contract) {
		if input.OfEachRow {
			return true
		}
	}
	return false
}

// decideRowScopedOperator reads every row-scoped input once and asks the
// implementation about the rows as a collection.
func decideRowScopedOperator(
	rule semantic.Validation,
	source StatefulSource,
	contract semantic.Operator,
	implementation OperatorImplementation,
) (Decision, error) {
	reader, readsRows := source.(StatefulRowNumberSource)
	if !readsRows {
		return Decision{}, fmt.Errorf(
			"semantic validation %s reads every row of relation %s, which this reader cannot read",
			rule.ID, rule.Relation)
	}
	if implementation.NumberRows == nil {
		return Decision{}, fmt.Errorf(
			"the selected implementation of operator %s cannot decide a rule over the rows of a relation",
			rule.Operator)
	}
	inputs := contractInputs(contract)
	columns := make(map[string][]NumberReading, len(inputs))
	var coverage Coverage
	var rowReason *Undecided
	for _, input := range inputs {
		// A row-scoped input is always an attribute of the relation: a
		// constant has one value, not one per row, and would be the same
		// whatever the rows hold.
		attribute, err := rowScopedInputAttribute(rule, input, inputs[0])
		if err != nil {
			return Decision{}, err
		}
		reading, err := reader.ReadRelationNumbers(rule.Relation, attribute)
		if err != nil {
			return Decision{}, fmt.Errorf(
				"semantic validation %s cannot evaluate relation %s: %w", rule.ID, rule.Relation, err)
		}
		switch reading.Coverage {
		case CoverageComplete, CoveragePartial:
		default:
			return Decision{}, fmt.Errorf(
				"semantic validation %s: the reader reports relation %s with the unknown coverage %q",
				rule.ID, rule.Relation, reading.Coverage)
		}
		if coverage == "" {
			coverage = reading.Coverage
		} else if reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s: input %q was read with coverage %q but an earlier input with %q; one rule reads one set of rows",
				rule.ID, input.Name, reading.Coverage, coverage)
		}
		if len(columns[inputs[0].Name]) > 0 && len(reading.Rows) != len(columns[inputs[0].Name]) {
			return Decision{}, fmt.Errorf(
				"semantic validation %s: input %q was read with %d rows but input %q with %d; one rule reads one set of rows",
				rule.ID, inputs[0].Name, len(columns[inputs[0].Name]), input.Name, len(reading.Rows))
		}
		columns[input.Name] = reading.Rows
		for _, row := range reading.Rows {
			// A reason inside the rows keeps its own meaning: it is not read
			// as zero, and it is reported before the coverage because reading
			// more rows cannot settle it.
			switch row.State {
			case ValuePresent:
				if row.Value == nil {
					return Decision{}, fmt.Errorf(
						"semantic validation %s: input %q reports a row as present without a value",
						rule.ID, input.Name)
				}
			case ValueNull, ValueNotFetched:
				// The reason is kept and passed to the composition, which
				// reports it for TriMaybe and refuses a decided answer that
				// was made around a row without a value.
				if rowReason == nil {
					reason := UndecidedValueNull
					if row.State == ValueNotFetched {
						reason = UndecidedValueNotFetched
					}
					rowReason = &Undecided{Reason: reason, Attribute: attribute, Relation: rule.Relation}
				}
			default:
				return Decision{}, fmt.Errorf(
					"semantic validation %s: input %q reports a row with the unknown state %q",
					rule.ID, input.Name, row.State)
			}
		}
	}
	answer, err := implementation.NumberRows(columns)
	if err != nil {
		return Decision{}, fmt.Errorf(
			"operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, rowReason)
}

// rowScopedInputAttribute is the attribute a row-scoped input is read from. The
// first input is the rule subject; the others must be assigned an attribute.
func rowScopedInputAttribute(
	rule semantic.Validation,
	input semantic.OperatorInputDeclaration,
	first semantic.OperatorInputDeclaration,
) (core.CanonicalID, error) {
	if input.Name == first.Name {
		return rule.Attribute, nil
	}
	assignment := findInputAssignment(rule, input.Name)
	if assignment == nil {
		return "", fmt.Errorf(
			"semantic validation %s does not assign operator %s input %q; every input after the first must be assigned",
			rule.ID, rule.Operator, input.Name)
	}
	if !isAttributeReference(assignment.Value) {
		return "", fmt.Errorf(
			"semantic validation %s assigns a constant to operator %s input %q, but an input read over all rows is an attribute",
			rule.ID, rule.Operator, input.Name)
	}
	attribute, err := core.ParseCanonicalID(assignment.Value)
	if err != nil {
		return "", fmt.Errorf(
			"semantic validation %s assigns %q to operator %s input %q, which is not an attribute: %w",
			rule.ID, assignment.Value, rule.Operator, input.Name, err)
	}
	return attribute, nil
}
