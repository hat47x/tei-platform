package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator declares a text input or a truth
// input.
//
// A text is read as a text and a truth as a truth. The reader reports one
// reading per attribute, and the reading states whether the value was read,
// whether the record holds no value, or what it holds. The implementation
// receives one value per input, each carrying the type it was read as
// (decision section 8.4), so it never has to ask which kind it holds and never
// sees a Source: what a rule reads stays decided by the declaration.
//
// A rule whose such input reads every row of a relation is refused rather than
// read row by row: this reader has no reader for a column of texts yet, and
// answering from one row would answer a question about the relation from one
// record of it.

// declaresTextOrTruthInput reports whether some declared input reads a text or
// a truth.
func declaresTextOrTruthInput(inputs []semantic.OperatorInputDeclaration) bool {
	for _, input := range inputs {
		if input.Type == semantic.OperatorInputText || input.Type == semantic.OperatorInputTruth {
			return true
		}
	}
	return false
}

// decideTextTruthOperator decides a rule whose operator declares at least one
// text or truth input, from the values of one record.
func decideTextTruthOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	texts, canText := source.(StatefulTextSource)
	truths, canTruth := source.(StatefulTruthSource)
	for _, input := range inputs {
		switch input.Type {
		case semantic.OperatorInputText:
			if !canText {
				return Decision{}, fmt.Errorf("semantic validation %s reads operator %s input %q as text, but this reader cannot read a text", rule.ID, rule.Operator, input.Name)
			}
		case semantic.OperatorInputTruth:
			if !canTruth {
				return Decision{}, fmt.Errorf("semantic validation %s reads operator %s input %q as truth, but this reader cannot read a truth", rule.ID, rule.Operator, input.Name)
			}
		}
		if input.OfEachRow {
			return Decision{}, fmt.Errorf(
				"semantic validation %s reads operator %s input %q over every row of relation %s, but this reader reads the %s of one record; a column of %s has no reader yet",
				rule.ID, rule.Operator, input.Name, rule.Relation, input.Type, input.Type)
		}
	}
	// The rule subject supplies the first input, and only it.
	for _, assignment := range rule.Inputs {
		if assignment.Name == inputs[0].Name {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies", rule.ID, rule.Operator, assignment.Name)
		}
	}
	// Every input is read before the operator is asked, so an unread part is
	// never turned into a settled answer and a value the caller supplied is
	// never left unread.
	values := make(map[string]semantic.OperatorValue, len(inputs))
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				// A constant is read as the type the declaration gives the
				// input, so a text constant is written with quotes and a truth
				// constant is the word true or false (decision section 8.3).
				// It belongs to no record, so no attempt is collected for it.
				value, err := semantic.ParseOperatorInputValue(input.Type, assignment.Value)
				if err != nil {
					return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is neither an attribute nor a %s: %w",
						rule.ID, rule.Operator, input.Name, assignment.Value, input.Type, err)
				}
				values[input.Name] = value
				continue
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
		}
		// The value is carried with the type it was read as, so the
		// implementation is asked exactly the question a rule read would.
		var value semantic.OperatorValue
		switch input.Type {
		case semantic.OperatorInputText:
			reading, err := readText(rule, texts, attribute)
			if err != nil {
				return Decision{}, err
			}
			switch reading.State {
			case ValuePresent:
				value = semantic.OperatorValue{Text: &reading.Value}
			case ValueNull:
				// A record that holds no text for this attribute has no value
				// to ask the operator about. The decision is undecided, and
				// the reason is the record's own. The empty text is a value of
				// length zero, which is why it is not this state (section
				// 8.3).
				return Decision{Undecided: &Undecided{Reason: UndecidedValueNull, Attribute: attribute}}, nil
			case ValueNotFetched:
				return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}}, nil
			default:
				return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
			}
		case semantic.OperatorInputTruth:
			reading, err := readTruth(rule, truths, attribute)
			if err != nil {
				return Decision{}, err
			}
			switch reading.State {
			case ValuePresent:
				value = semantic.OperatorValue{Truth: &reading.Value}
			case ValueNull:
				return Decision{Undecided: &Undecided{Reason: UndecidedValueNull, Attribute: attribute}}, nil
			case ValueNotFetched:
				return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}}, nil
			default:
				return Decision{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
			}
		default:
			return Decision{}, fmt.Errorf("semantic validation %s reads operator %s input %q as %s, which this path does not read", rule.ID, rule.Operator, input.Name, input.Type)
		}
		values[input.Name] = value
	}
	// Every input came from this one record, so no two-attempt check is needed:
	// the reader states the record it read.
	if implementation.Values == nil {
		return Decision{}, fmt.Errorf("operator %s reads a %s, but its selected implementation does not decide one", rule.Operator, inputs[0].Type)
	}
	satisfied, err := implementation.Values(values)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	if satisfied {
		return Decision{Result: ResultSatisfied}, nil
	}
	return Decision{Result: ResultNotSatisfied}, nil
}

// readText reads one text attribute of one record and reports what the reader
// said about it.
func readText(rule semantic.Validation, source StatefulTextSource, attribute core.CanonicalID) (TextReading, error) {
	reading, err := source.ReadText(attribute)
	if err != nil {
		return TextReading{}, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
	}
	return reading, nil
}

// readTruth reads one truth attribute of one record and reports what the
// reader said about it.
func readTruth(rule semantic.Validation, source StatefulTruthSource, attribute core.CanonicalID) (TruthReading, error) {
	reading, err := source.ReadTruth(attribute)
	if err != nil {
		return TruthReading{}, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
	}
	return reading, nil
}
