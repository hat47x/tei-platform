package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator asks, for every row of a relation,
// whether that row holds a value (decision section 8.6).
//
// The reader reports one state per row. A row that holds a value and a row that
// holds none are both answers, and both are handed to the implementation as
// they are: folding "holds none" into a boolean here would decide, for the
// implementation, a question the implementation is meant to answer from the
// rows it receives. A row nobody read is not an answer at all, so the evaluator
// reports that row's own reason and never calls the implementation.
//
// The three-valued answer, the read Coverage and the operator's refutation
// statement are composed in the one place that decides it (decideRowScoped),
// exactly as they are for the other row paths.

// decidePresenceRowScopedOperator decides a rule whose operator reads whether
// every row of a relation holds a value.
func decidePresenceRowScopedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}
	reader, ok := source.(StatefulPresenceRowSource)
	if !ok {
		return Decision{}, fmt.Errorf(
			"semantic validation %s asks whether every row of relation %s holds a value, but this reader cannot read the presence of a relation's rows",
			rule.ID, rule.Relation)
	}

	rows := make(map[string][]PresenceReading, len(inputs))
	var coverage Coverage
	var reason *Undecided
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s does not assign operator %s input %q; every input after the first must be assigned",
					rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				// A constant belongs to no row, so there is no row whose
				// presence it could state. Reading the constant's spelling as
				// an answer would decide a question about rows nobody read.
				return Decision{}, fmt.Errorf(
					"semantic validation %s assigns operator %s input %q the constant %q, but a presence input asks the rows of a relation whether they hold a value; assign an attribute",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
		}
		reading, readErr := reader.ReadRelationPresence(core.CanonicalID(rule.Relation), attribute)
		if readErr != nil {
			return Decision{}, fmt.Errorf(
				"semantic validation %s cannot read whether every row of relation %s holds a value for attribute %s: %w",
				rule.ID, rule.Relation, attribute, readErr)
		}
		// Every input of this one rule is read from this one relation, so all
		// of them report the same coverage. Two coverages would mean two reads,
		// and one answer must not mix them.
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
		rows[input.Name] = reading.Rows
		// A row nobody read is not an answer to the presence question (section
		// 8.6): the implementation is not called for it, and the rule is
		// undecided for that row's own reason. Folding the unread row into
		// "holds none" would decide a question about a row nobody looked at.
		if reason == nil {
			reason = firstPresenceRowReason(attribute, reading.Rows)
		}
		if reason != nil {
			return Decision{Undecided: reason}, nil
		}
	}

	answer, err := implementation.PresenceRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, nil)
}

// firstPresenceRowReason reports the first row in read order that nobody read,
// so the rule is undecided for that row's own reason rather than told that the
// row holds nothing.
func firstPresenceRowReason(attribute core.CanonicalID, rows []PresenceReading) *Undecided {
	for _, row := range rows {
		if row.State == ValueNotFetched {
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
	}
	return nil
}
