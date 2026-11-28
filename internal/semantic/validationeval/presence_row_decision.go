package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// PresenceRowsReading is what a reader knows about whether one attribute holds a
// value in every row of a relation: the rows in the order they were read, each
// state kept, and whether those rows are all the rows.
//
// The three states are kept apart, because "this row holds no value" and
// "nobody read this row" answer the presence question differently: the first is
// an answer, the second is not one.
type PresenceRowsReading struct {
	Coverage Coverage
	Rows     []PresenceReading
}

// StatefulPresenceRowSource is implemented by a reader that can read whether one
// attribute holds a value in every row of a relation. A rule whose operator reads
// presence of every row is refused by a reader that cannot; it is not answered
// from one record (decision section 15.1).
type StatefulPresenceRowSource interface {
	ReadRelationPresences(relation, attribute core.CanonicalID) (PresenceRowsReading, error)
}

// readRelationPresences reads one input's presence column over the rows of a
// relation. A reader that cannot read presence over rows is an error rather than
// a value of another type: reading a column as numbers would answer a question
// about a value nobody asked about (#4359).
func readRelationPresences(rule semantic.Validation, source StatefulSource, relation, attribute core.CanonicalID) (PresenceRowsReading, error) {
	reader, ok := source.(StatefulPresenceRowSource)
	if !ok {
		return PresenceRowsReading{}, fmt.Errorf("semantic validation %s reads whether attribute %s holds a value in every row of %s, but this reader cannot read presence over rows",
			rule.ID, attribute, relation)
	}
	reading, err := reader.ReadRelationPresences(relation, attribute)
	if err != nil {
		return PresenceRowsReading{}, fmt.Errorf("semantic validation %s cannot read whether attribute %s holds a value over the rows of %s: %w",
			rule.ID, attribute, relation, err)
	}
	return reading, nil
}

// decidePresenceRowScopedOperator decides a rule whose operator reads whether an
// attribute holds a value in every row of a relation.
//
// The implementation receives, per input name, the state of every row that was
// read, so it can answer the three-valued question itself. The read Coverage and
// the operator's refutation statement are composed in the one place that decides
// a row-scoped answer (decideRowScoped, row_scope.go), exactly as they are for
// the numeric and typed row paths.
//
// A constant belongs to no record, so a presence input assigned a constant is
// refused rather than read: there is no row whose value it could ask about.
func decidePresenceRowScopedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}
	if implementation.PresenceRows == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads whether an attribute holds a value in every row, but its selected implementation does not decide presence over rows",
			rule.Operator)
	}
	rows := make(map[string][]PresenceReading, len(inputs))
	var coverage Coverage
	var reason *Undecided
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the constant %q, but a presence input asks a row whether it holds a value; assign an attribute",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
		}
		reading, readErr := readRelationPresences(rule, source, core.CanonicalID(rule.Relation), attribute)
		if readErr != nil {
			return Decision{}, readErr
		}
		// Every input of this one rule is read from this one relation, so all of
		// them report the same coverage. Two coverages would mean two reads, and
		// one answer must not mix them.
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
		rows[input.Name] = reading.Rows
		// A row nobody read keeps its own reason. The first such row in read order
		// is reported; the implementation is still asked, so a counterexample
		// found in another row is not thrown away.
		if reason == nil {
			reason = firstPresenceRowReason(attribute, reading.Rows)
		}
	}
	answer, err := implementation.PresenceRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, reason)
}

// firstPresenceRowReason reports the first row nobody read, in read order.
//
// A row that holds no value is not a reason here: it is an answer to the presence
// question, and the implementation is the one that decides what that answer
// means for the operator.
func firstPresenceRowReason(attribute core.CanonicalID, rows []PresenceReading) *Undecided {
	for _, row := range rows {
		if row.State == ValueNotFetched {
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
		if row.State != ValuePresent && row.State != ValueNull {
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
	}
	return nil
}
