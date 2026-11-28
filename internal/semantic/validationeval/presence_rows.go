package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator declares a presence input read over
// every row of a relation (decision sections 15.1 and 3):
//
//	operator sales.every_line_has_a_price:
//	  takes value: presence of each row
//	  refutes on some rows
//
// A presence input asks whether a row holds a value, not which value it holds.
// The answer is one of three states per row: the row holds a value, the row
// holds none, or the query did not return the column. Only the first two are
// readings of the row; the third is the read saying it did not look, so it is
// not turned into "no value".
//
// The rows are read once, for the whole relation, so every input of one rule
// comes from one Read Attempt (section 14.2 B-10). The operator is a pure
// function: it receives the three-valued state of each row and answers
// TriYes, TriNo or TriMaybe. The composition with Coverage and the operator's
// `refutes on some rows` statement happens in the one place that decides it
// (decideRowScoped, row_scope.go).

// PresenceRowsReading is what a reader knows about one presence input over the
// rows of a relation: the state of every row, in the order the rows were read,
// and whether those rows are every row of the relation.
type PresenceRowsReading struct {
	Coverage Coverage
	Rows     []PresenceReading
}

// StatefulPresenceRowSource is implemented by a reader that can answer whether
// every row of a relation holds a value for one attribute. A reader that cannot
// is refused rather than asked row by row: a rule whose input reads all the rows
// cannot be answered from one row.
type StatefulPresenceRowSource interface {
	ReadRelationPresence(relation, attribute core.CanonicalID) (PresenceRowsReading, error)
}

// readPresenceRows reads one presence input over the rows of a relation. A
// reader that cannot is an error, not an undecided rule: the reader lacking the
// read is a property of the reader, while an unread row is a property of this
// relation.
func readPresenceRows(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource, relation, attribute core.CanonicalID) (PresenceRowsReading, error) {
	reader, ok := source.(StatefulPresenceRowSource)
	if !ok {
		return PresenceRowsReading{}, fmt.Errorf("semantic validation %s reads operator %s input %q as presence over the rows of %s, but this reader cannot answer whether a row holds a value",
			rule.ID, rule.Operator, input.Name, relation)
	}
	reading, err := reader.ReadRelationPresence(relation, attribute)
	if err != nil {
		return PresenceRowsReading{}, fmt.Errorf("semantic validation %s cannot ask whether attribute %s of relation %s holds a value: %w",
			rule.ID, attribute, relation, err)
	}
	return reading, nil
}

// decidePresenceRowScopedOperator decides a rule whose operator reads whether
// every row of a relation holds a value.
//
// The first input's column is the rule's subject attribute. Every later input
// must be assigned by name to an attribute of the same relation. A constant is
// refused rather than expanded to every row: a constant belongs to no record,
// so a presence question about each row cannot be asked of it.
func decidePresenceRowScopedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}
	if implementation.PresenceRows == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads whether every row of a relation holds a value, but its selected implementation does not decide over rows",
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
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the constant %q, but a presence input asks the rows of relation %s whether they hold a value; assign an attribute",
					rule.ID, rule.Operator, input.Name, assignment.Value, rule.Relation)
			}
			attribute = core.CanonicalID(assignment.Value)
		}
		reading, err := readPresenceRows(rule, input, source, core.CanonicalID(rule.Relation), attribute)
		if err != nil {
			return Decision{}, err
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
		// A row that holds no value, or was not read, keeps its own reason. The
		// first such row in read order is reported; the implementation is still
		// asked, so a counterexample found in another row is not thrown away.
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

// firstPresenceRowReason reports the first row that holds no value or was not
// read, in read order. A decided answer must not be reported together with it
// (decideRowScoped checks that).
func firstPresenceRowReason(attribute core.CanonicalID, rows []PresenceReading) *Undecided {
	for _, row := range rows {
		switch row.State {
		case ValueNull:
			return &Undecided{Reason: UndecidedValueNull, Attribute: attribute}
		case ValueNotFetched:
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
	}
	return nil
}
