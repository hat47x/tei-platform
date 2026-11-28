package validationeval

import (
	"fmt"
	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides an operator rule whose inputs read whether an attribute
// holds a value. It is the one-record path; a row-scoped presence input is
// refused rather than decided from one record.
//
// Decision section 5 fixes the shape: the evaluator reads the state and hands
// all three of them to the implementation, because a rule about presence is
// decided by whether a value is there. "The record holds none" and "the
// attribute was not read" are answers, not reasons to stop: the implementation
// is asked about both, so a rule that would answer differently has to be
// refused by its own example table rather than guessed here.

// declaresPresenceInput reports whether any input reads whether a value is
// there.
func declaresPresenceInput(inputs []semantic.OperatorInputDeclaration) bool {
	for _, input := range inputs {
		if input.Type == semantic.OperatorInputPresence {
			return true
		}
	}
	return false
}

// decidePresenceOperator decides one operator rule about whether the record's
// attributes hold values.
func decidePresenceOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if implementation.Presence == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads whether a value is there, but its selected implementation does not decide from presence",
			rule.Operator)
	}
	// A presence input is of one record. A row-scoped presence input would ask
	// about every row at once, which this path does not read; refusing it is
	// better than answering it from the subject's own record.
	for _, input := range inputs {
		if input.OfEachRow {
			return Decision{}, fmt.Errorf(
				"semantic validation %s reads operator %s input %q as presence over every row of a relation, which this path does not read",
				rule.ID, rule.Operator, input.Name)
		}
	}
	reader, ok := source.(StatefulPresenceSource)
	if !ok {
		return Decision{}, fmt.Errorf(
			"semantic validation %s reads whether attribute %s holds a value, but this reader cannot read presence",
			rule.ID, rule.Attribute)
	}
	// The rule subject supplies the first input, and only it.
	for _, assignment := range rule.Inputs {
		if assignment.Name == inputs[0].Name {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies", rule.ID, rule.Operator, assignment.Name)
		}
	}
	values := make(map[string]bool, len(inputs))
	origins := []inputOrigin{dataOrigin(inputs[0].Name, string(rule.Attribute), source)}
	reading, err := readPresence(rule, reader, rule.Attribute)
	if err != nil {
		return Decision{}, err
	}
	// Section 8.5's table: a present attribute is true, an attribute that holds
	// no value is false, and an attribute nobody read is not handed to the
	// implementation at all — the rule stays undecided with its own reason.
	switch reading.State {
	case ValuePresent:
		values[inputs[0].Name] = true
	case ValueNull:
		values[inputs[0].Name] = false
	default:
		return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: rule.Attribute}}, nil
	}
	for _, input := range inputs[1:] {
		assignment := findInputAssignment(rule, input.Name)
		if assignment == nil {
			return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
		}
		if !isAttributeReference(assignment.Value) {
			// A constant is not a presence state: it names no attribute, and
			// writing a word for "there" or "not there" would be a new spelling
			// the design does not state.
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q; a presence input is read from an attribute, and a constant states no presence",
				rule.ID, rule.Operator, input.Name, assignment.Value)
		}
		attribute, aerr := core.ParseCanonicalID(assignment.Value)
		if aerr != nil {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q", rule.ID, rule.Operator, input.Name, assignment.Value)
		}
		origins = append(origins, dataOrigin(input.Name, assignment.Value, source))
		got, rerr := readPresence(rule, reader, attribute)
		if rerr != nil {
			return Decision{}, rerr
		}
		switch got.State {
		case ValuePresent:
			values[input.Name] = true
		case ValueNull:
			values[input.Name] = false
		default:
			return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}}, nil
		}
	}
	unknown, aerr := requireSameAttempt(rule, origins)
	if aerr != nil {
		return Decision{}, aerr
	}
	if unknown != nil {
		return Decision{Undecided: unknown}, nil
	}
	satisfied, err := implementation.Presence(values)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decided(satisfiedResult(satisfied))
}

// presenceIsPresent is the shipped presence implementation's decision: an
// input named "value" holds a value. It is the generic check the standard's
// presence input states, and it reads nothing else.
//
// A state that is neither present nor absent is not something a reader
// produced, so it is refused rather than read as either.
func presenceIsPresent(values map[string]bool) (bool, error) {
	present, ok := values["value"]
	if !ok {
		return false, fmt.Errorf("present reads an input named \"value\"; the rule supplied no value input")
	}
	return present, nil
}
