package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator declares a presence input.
//
// A presence input asks whether an attribute holds a value, not which value it
// holds (decision section 3). The reader answers with one of three states: the
// record holds a value, the record holds none, or the query did not return the
// attribute. The first two are the answers the implementation receives; the
// third is not an answer at all, so the rule stays undecided with the reader's
// own reason rather than telling the implementation that nothing is there.
//
// The value never travels in semantic.OperatorValue: a truth is what a column
// says and presence is whether the column says anything, and giving presence a
// field there would let an implementation answer a question about a value from a
// record that holds none.

// declaresPresenceInput reports whether some declared input reads whether an
// attribute holds a value.
func declaresPresenceInput(inputs []semantic.OperatorInputDeclaration) bool {
	for _, input := range inputs {
		if input.Type == semantic.OperatorInputPresence {
			return true
		}
	}
	return false
}

// presenceInputName returns the name of the first presence input, so a refusal
// can name the declaration the author has to change.
func presenceInputName(inputs []semantic.OperatorInputDeclaration) string {
	for _, input := range inputs {
		if input.Type == semantic.OperatorInputPresence {
			if input.Name == "" {
				return "value"
			}
			return input.Name
		}
	}
	return ""
}

// ParsePresenceExampleValue reads the cell of an example for a presence input.
// Section 8.3 fixes the two words: `present` when the record holds a value and
// `absent` when it holds none. The word `not-fetched` is not a cell this reader
// can hand an implementation: it states that nobody read the value, which is
// not the same fact as the record holding none.
func ParsePresenceExampleValue(text string) (bool, error) {
	switch text {
	case semantic.OperatorCasePresencePresent:
		return true, nil
	case semantic.OperatorCasePresenceAbsent:
		return false, nil
	case semantic.RowCellNotFetched:
		return false, fmt.Errorf("%q states that the value was not read, so it is not an answer a presence implementation receives; write %s or %s to state whether the record holds a value",
			text, semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent)
	}
	return false, fmt.Errorf("%q is not a value of a presence input; write %s or %s",
		text, semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent)
}

// parsePresenceRowExampleCell reads the cell of a row example for a presence
// input read over every row of a relation (section 8.6).
//
// Section 8.6 fixes three words for the cell, the same three words a row cell
// may write (section 8.3): `present` when the row holds a value, `absent` when
// it holds none, and `not-fetched` when nobody read the row. The third word is
// a cell here while a one-record example cannot write it, because a row example
// states rows nobody read and the one-record reader hands an implementation
// answers, not the fact that nobody looked.
//
// An empty cell is refused. Section 8.1 lets a value input write "the row holds
// no value" as an empty cell, and a presence input writes that state as
// `absent`. Accepting both would give one state two writings, so the empty cell
// is refused rather than read as one of them.
func parsePresenceRowExampleCell(text string) (PresenceReading, error) {
	switch text {
	case semantic.OperatorCasePresencePresent:
		return PresenceReading{State: ValuePresent}, nil
	case semantic.OperatorCasePresenceAbsent:
		return PresenceReading{State: ValueNull}, nil
	case semantic.RowCellNotFetched:
		return PresenceReading{State: ValueNotFetched}, nil
	case semantic.RowCellNoValue:
		return PresenceReading{}, fmt.Errorf("the cell is empty; a presence input states one of %s, %s or %s, and a row that holds no value is written %s rather than left empty",
			semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent, semantic.RowCellNotFetched, semantic.OperatorCasePresenceAbsent)
	}
	return PresenceReading{}, fmt.Errorf("%q is not a state of a presence input; write %s, %s or %s",
		text, semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent, semantic.RowCellNotFetched)
}

// decidePresenceOperator decides a rule whose operator declares at least one
// presence input, from the states of one record.
func decidePresenceOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	reader, ok := source.(StatefulPresenceSource)
	if !ok {
		return Decision{}, fmt.Errorf("semantic validation %s reads operator %s, which asks whether an attribute holds a value, but this reader cannot answer that",
			rule.ID, rule.Operator)
	}
	if implementation.Presence == nil {
		return Decision{}, fmt.Errorf("operator %s declares a presence input, but its selected implementation does not decide presence",
			rule.Operator)
	}
	// The rule subject supplies the first input, and only it.
	for _, assignment := range rule.Inputs {
		if assignment.Name == inputs[0].Name {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies",
				rule.ID, rule.Operator, assignment.Name)
		}
	}
	present := make(map[string]bool, len(inputs))
	// A rule that asks about more than one attribute reads more than one value,
	// and having several answers is not the same as knowing they came from one
	// read (section 14.2 B-10). The origin of each input is collected while it
	// is asked about, and compared before the implementation is asked, exactly
	// as the value path does.
	origins := make([]inputOrigin, 0, len(inputs))
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned",
					rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				// A constant belongs to no record, so there is no record whose
				// value the presence input could be asked about. Reading the
				// constant's spelling as a value would answer a question about
				// a record nobody read, so it is refused rather than guessed.
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the constant %q, but a presence input asks a record whether it holds a value; assign an attribute",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
		}
		origins = append(origins, dataOrigin(input.Name, string(attribute), source))
		reading, err := reader.ReadPresence(attribute)
		if err != nil {
			return Decision{}, fmt.Errorf("semantic validation %s cannot ask whether attribute %s holds a value: %w",
				rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			present[input.Name] = true
		case ValueNull:
			// The record holds no value. That is an answer to the presence
			// question, and it is not the same as the value being false or
			// empty: the implementation is told that nothing is there.
			present[input.Name] = false
		case ValueNotFetched:
			// Nobody read the attribute. A rule cannot be told that nothing is
			// there when what happened is that nobody looked (section 5).
			return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}}, nil
		default:
			return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q",
				rule.ID, attribute, reading.State)
		}
	}
	unknown, aerr := requireSameAttempt(rule, origins)
	if aerr != nil {
		return Decision{}, aerr
	}
	if unknown != nil {
		return Decision{Undecided: unknown}, nil
	}
	satisfied, err := implementation.Presence(present)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	if satisfied {
		return Decision{Result: ResultSatisfied}, nil
	}
	return Decision{Result: ResultNotSatisfied}, nil
}

// checkPresenceOperatorCases asks a presence implementation about every example
// the contract states, so an implementation that answers one of them differently
// is not an implementation of this operator and is never asked about real data
// (section 8.1).
func checkPresenceOperatorCases(contract semantic.Operator, implementation OperatorImplementation, inputs []semantic.OperatorInputDeclaration) error {
	for _, example := range contract.Cases {
		// An example states either one value or one value per declared input.
		// A multi-input example that leaves one unstated would ask the
		// implementation about a record nobody wrote, so it is refused.
		if len(inputs) > 1 && example.Inputs == nil {
			return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares %d inputs (%s), but its example carries one value; a multi-input example states one column per input",
				contract.ID, len(inputs), operatorInputNames(inputs))
		}
		if example.Inputs != nil && len(inputs) > 1 {
			for _, input := range inputs {
				if _, stated := example.Inputs[input.Name]; !stated {
					return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s example states %s, but input %q is not stated; an example that leaves an input unread cannot be checked",
						contract.ID, operatorExampleNames(example.Inputs), input.Name)
				}
			}
		}
		if example.Inputs != nil && len(inputs) == 1 {
			if _, ok := example.Inputs[inputs[0].Name]; !ok {
				return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares the input %q, but an example states %s",
					contract.ID, inputs[0].Name, operatorExampleNames(example.Inputs))
			}
		}
		present := make(map[string]bool, len(inputs))
		for _, input := range inputs {
			text := example.Value
			if example.Inputs != nil {
				text = example.Inputs[input.Name]
			}
			value, err := ParsePresenceExampleValue(text)
			if err != nil {
				return fmt.Errorf("operator %s has an unreadable example for input %q: %w", contract.ID, input.Name, err)
			}
			present[input.Name] = value
		}
		got, err := implementation.Presence(present)
		if err != nil {
			return fmt.Errorf("the implementation of operator %s could not decide the contract's example %s: %w",
				contract.ID, operatorExampleText(example), err)
		}
		if got != example.Satisfied {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that %s is %s, but the selected implementation answers %s",
				contract.ID, operatorExampleText(example), operatorCaseWord(example.Satisfied), operatorCaseWord(got))
		}
	}
	return nil
}
