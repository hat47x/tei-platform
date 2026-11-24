package validationread

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// Decisions is the answer of one rule for the rows of a Detail.
//
// A rule about all the rows of the relation has one answer, in Relation. A
// rule about one record has one answer for every row, in Rows, in the order
// of the rows. Exactly one of the two is set; which one is decided by the
// kind of the rule and never by what the rows happen to hold.
type Decisions struct {
	Rule     core.CanonicalID
	Relation *validationeval.Decision
	Rows     []validationeval.Decision
}

// Decide asks rule of the rows.
//
// A rule about one record is asked of every row that was read. When the rows
// are not all the rows of the relation, its answers still speak only of the
// rows read: nothing is said about a row that was not returned, and no row
// means no answer, not a satisfied rule.
//
// A rule whose operator reads every row of the relation is asked once, of the
// Detail, and has one answer in Relation. That is decided by the declaration
// and never by what the rows happen to hold: the same rule asked of one row
// would be answered row by row while its Source says otherwise, and a reader
// that cannot read the rows of a relation refuses it instead of answering from
// one row.
//
// A kind this reader does not know is refused. It is not asked of the rows in
// either way by default: a new kind that is about all the rows would
// otherwise be answered row by row, and that answer would look decided.
func (d *Detail) Decide(rule semantic.Validation, operators validationeval.Operators) (Decisions, error) {
	decisions := Decisions{Rule: rule.ID}
	switch rule.Kind {
	case semantic.ValidationRelationUnique, semantic.ValidationNoOverlap:
		decision, err := validationeval.Decide(rule, d, operators)
		if err != nil {
			return Decisions{}, err
		}
		decisions.Relation = &decision
		return decisions, nil

	case semantic.ValidationPositive, semantic.ValidationOperator, semantic.ValidationNotAfter, semantic.ValidationRequiredIf:
		if rule.Kind == semantic.ValidationOperator && operatorReadsEveryRow(operators, rule.Operator) {
			decision, err := validationeval.Decide(rule, d, operators)
			if err != nil {
				return Decisions{}, err
			}
			decisions.Relation = &decision
			return decisions, nil
		}
		decisions.Rows = make([]validationeval.Decision, 0, len(d.rows))
		for index := range d.rows {
			decision, err := validationeval.Decide(rule, Row{detail: d, index: index}, operators)
			if err != nil {
				// A value that cannot be read names its row itself.
				return Decisions{}, err
			}
			decisions.Rows = append(decisions.Rows, decision)
		}
		return decisions, nil

	default:
		return Decisions{}, fmt.Errorf("semantic validation %s is of kind %q; this reader does not know whether that is asked of one row or of all the rows", rule.ID, rule.Kind)
	}
}

// operatorReadsEveryRow reports whether the operator a rule names is declared
// to read every row of a relation. An operator that is not declared is false
// here: the evaluator refuses the rule for the missing declaration, which is
// the diagnosis the author needs, rather than this dispatch choosing a
// different reading for it.
func operatorReadsEveryRow(operators validationeval.Operators, id core.CanonicalID) bool {
	for _, operator := range operators.Declared {
		if operator.ID == id {
			return operator.ReadsEveryRow()
		}
	}
	return false
}

func (d *Detail) readPresence(index int, attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	if _, err := d.kindOf(attribute); err != nil {
		return validationeval.PresenceReading{}, err
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.PresenceReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.PresenceReading{State: validationeval.ValueNull}, nil
	}
	return validationeval.PresenceReading{State: validationeval.ValuePresent}, nil
}

// ReadPresence reads whether the row holds a value for attribute. The value
// is not parsed: an empty text is a value, and only SQL NULL is none.
func (r Row) ReadPresence(attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	return r.detail.readPresence(r.index, attribute)
}
