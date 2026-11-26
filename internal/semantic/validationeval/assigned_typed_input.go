package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file reads an input that is assigned by name, in the type its
// declaration states.
//
// The subject is read by the readers that understand its type (readNumber,
// readDate, readTextSubject, readTruthSubject). An input assigned by name is
// read the same way, so a text is a text whether it is the rule's subject or a
// bound the rule names: reading it as a number would answer a question about a
// different value (decision section 8.4).

// readAssignedTypedInput reads one named input in the type its declaration
// states, whether the rule assigned it an attribute or a constant. A constant is
// read in the spelling that type is written in, and it belongs to no record, so
// the reader is not asked about it.
//
// The second and third results are the value and the reason there is none.
func readAssignedTypedInput(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource) (semantic.OperatorValue, *Undecided, error) {
	assignment := findInputAssignment(rule, input.Name)
	if assignment == nil {
		return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
	}
	if !isAttributeReference(assignment.Value) {
		value, err := readTypedConstant(rule, input, assignment.Value)
		if err != nil {
			return semantic.OperatorValue{}, nil, err
		}
		return value, nil, nil
	}
	attribute, err := core.ParseCanonicalID(assignment.Value)
	if err != nil {
		return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q",
			rule.ID, rule.Operator, input.Name, assignment.Value)
	}
	return readAssignedTypedValue(rule, input, source, attribute)
}

// readTypedConstant reads one input's constant in the spelling its declared type
// is written in: a calendar day is YYYY-MM-DD, a text is in double quotes, and a
// truth is the word true or the word false. A value of another type, or a text
// that spells none of them, is refused at the assignment's own position rather
// than read as a number.
func readTypedConstant(rule semantic.Validation, input semantic.OperatorInputDeclaration, text string) (semantic.OperatorValue, error) {
	switch input.Type {
	case semantic.OperatorInputDate:
		day, err := semantic.ParseCalendarDate(text)
		if err != nil {
			return semantic.OperatorValue{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is neither an attribute nor a calendar day: %w",
				rule.ID, rule.Operator, input.Name, text, err)
		}
		return semantic.OperatorValue{Date: &day}, nil
	case semantic.OperatorInputText:
		spelled, ok := semantic.ReadSpelledValue(text)
		if !ok || spelled.Kind != semantic.SpelledKindText {
			return semantic.OperatorValue{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is not a text; write it in double quotes",
				rule.ID, rule.Operator, input.Name, text)
		}
		value, err := spelled.TextOf()
		if err != nil {
			return semantic.OperatorValue{}, err
		}
		return semantic.OperatorValue{Text: &value}, nil
	case semantic.OperatorInputTruth:
		spelled, ok := semantic.ReadSpelledValue(text)
		if !ok || spelled.Kind != semantic.SpelledKindTruth {
			return semantic.OperatorValue{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is not a truth; write true or false",
				rule.ID, rule.Operator, input.Name, text)
		}
		value, err := spelled.TruthOf()
		if err != nil {
			return semantic.OperatorValue{}, err
		}
		return semantic.OperatorValue{Truth: &value}, nil
	case semantic.OperatorInputAny:
		spelled, ok := semantic.ReadSpelledValue(text)
		if !ok {
			return semantic.OperatorValue{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is not a value this program reads",
				rule.ID, rule.Operator, input.Name, text)
		}
		return anyValueOf(spelled)
	}
	return semantic.OperatorValue{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the constant %q, which this program does not read as %s",
		rule.ID, rule.Operator, input.Name, text, input.Type)
}

// anyValueOf carries a spelled value as a value of any type, keeping the kind it
// was read as.
func anyValueOf(spelled semantic.SpelledValue) (semantic.OperatorValue, error) {
	any := semantic.AnyValue{Type: semantic.OperatorInput(spelled.Kind)}
	switch spelled.Kind {
	case semantic.SpelledKindNumber:
		value, err := spelled.NumberOf()
		if err != nil {
			return semantic.OperatorValue{}, err
		}
		any.Number = value
	case semantic.SpelledKindDate:
		value, err := spelled.DateOf()
		if err != nil {
			return semantic.OperatorValue{}, err
		}
		any.Date = &value
	case semantic.SpelledKindText:
		value, err := spelled.TextOf()
		if err != nil {
			return semantic.OperatorValue{}, err
		}
		any.Text = &value
	case semantic.SpelledKindTruth:
		value, err := spelled.TruthOf()
		if err != nil {
			return semantic.OperatorValue{}, err
		}
		any.Truth = &value
	default:
		return semantic.OperatorValue{}, fmt.Errorf("%q is spelled as a kind no input accepts", spelled.Text)
	}
	return semantic.OperatorValue{Any: &any}, nil
}

// readAssignedTypedValue reads one named input in the type its declaration
// states. The second and third results are the value and the reason there is
// none: a value the record does not hold and a value the reader did not fetch
// are different states, and neither is the zero of its type.
//
// A reader that cannot read the declared type is an error, so a rule is refused
// rather than answered from a value it never held.
func readAssignedTypedValue(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource, attribute core.CanonicalID) (semantic.OperatorValue, *Undecided, error) {
	switch input.Type {
	case semantic.OperatorInputText:
		reader, ok := source.(StatefulTextSource)
		if !ok {
			return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as text, but this reader cannot read text",
				rule.ID, rule.Operator, input.Name)
		}
		reading, err := reader.ReadText(attribute)
		if err != nil {
			return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
		}
		return presentTypedReading(rule, attribute, reading.State, func() semantic.OperatorValue {
			text := reading.Value
			return semantic.OperatorValue{Text: &text}
		})
	case semantic.OperatorInputTruth:
		reader, ok := source.(StatefulTruthSource)
		if !ok {
			return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as truth, but this reader cannot read truth",
				rule.ID, rule.Operator, input.Name)
		}
		reading, err := reader.ReadTruth(attribute)
		if err != nil {
			return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
		}
		return presentTypedReading(rule, attribute, reading.State, func() semantic.OperatorValue {
			truth := reading.Value
			return semantic.OperatorValue{Truth: &truth}
		})
	case semantic.OperatorInputAny:
		reader, ok := source.(StatefulAnySource)
		if !ok {
			return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as a value of any type, but this reader cannot read one",
				rule.ID, rule.Operator, input.Name)
		}
		reading, err := reader.ReadAny(attribute)
		if err != nil {
			return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
		}
		return presentTypedReading(rule, attribute, reading.State, func() semantic.OperatorValue {
			if !reading.Value.Valid() {
				// A value no type reads is not a value: it is reported here,
				// where the reader that produced it is named, rather than being
				// handed to an implementation.
				return semantic.OperatorValue{}
			}
			value := reading.Value
			return semantic.OperatorValue{Any: &value}
		})
	}
	return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as %s, which this program does not read a value as",
		rule.ID, rule.Operator, input.Name, input.Type)
}

// presentTypedReading turns a reading's state into either the value the reader
// read or the reason there is none. Exactly one of the first two results is set
// when err is nil.
func presentTypedReading(rule semantic.Validation, attribute core.CanonicalID, state ValueState, value func() semantic.OperatorValue) (semantic.OperatorValue, *Undecided, error) {
	switch state {
	case ValuePresent:
		return value(), nil, nil
	case ValueNull:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNull, Attribute: attribute}, nil
	case ValueNotFetched:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}, nil
	default:
		return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, state)
	}
}
