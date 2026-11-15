package main

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/scenariotest"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

type scenarioValidationSource struct {
	fixture       scenariotest.ValidationFixture
	dateReads     map[string]bool
	numberReads   map[string]bool
	relationReads map[string]bool
}

// The Fixture says for each input whether a value is there and, if not, why.
// The source hands that state to the evaluator unchanged; it never turns a
// missing value into zero or a partly read relation into a whole one.

func (s *scenarioValidationSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	id := string(attribute)
	if state, stated := s.fixture.NumberStates[id]; stated {
		if _, both := s.fixture.Numbers[id]; both {
			return validationeval.NumberReading{}, fmt.Errorf("Validation numeric Attribute %s has both a value and the state %s", id, state)
		}
		s.numberReads[id] = true
		switch state {
		case scenariotest.ValidationStateNull:
			return validationeval.NumberReading{State: validationeval.ValueNull}, nil
		case scenariotest.ValidationStateNotFetched:
			return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
		default:
			return validationeval.NumberReading{}, fmt.Errorf("invalid state %q of Validation numeric Attribute %s", state, id)
		}
	}
	value, ok := s.fixture.Numbers[id]
	if !ok {
		return validationeval.NumberReading{}, fmt.Errorf("missing Validation numeric Attribute %s", id)
	}
	s.numberReads[id] = true
	n, ok := new(big.Rat).SetString(value)
	if !ok {
		return validationeval.NumberReading{}, fmt.Errorf("invalid Validation numeric Attribute %s", id)
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: n}, nil
}

func (s *scenarioValidationSource) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	id := string(attribute)
	if state, stated := s.fixture.DateStates[id]; stated {
		if _, both := s.fixture.Dates[id]; both {
			return validationeval.DateReading{}, fmt.Errorf("Validation date Attribute %s has both a value and the state %s", id, state)
		}
		s.dateReads[id] = true
		switch state {
		case scenariotest.ValidationStateNull:
			return validationeval.DateReading{State: validationeval.ValueNull}, nil
		case scenariotest.ValidationStateNotFetched:
			return validationeval.DateReading{State: validationeval.ValueNotFetched}, nil
		default:
			return validationeval.DateReading{}, fmt.Errorf("invalid state %q of Validation date Attribute %s", state, id)
		}
	}
	value, ok := s.fixture.Dates[id]
	if !ok {
		return validationeval.DateReading{}, fmt.Errorf("missing Validation date Attribute %s", id)
	}
	s.dateReads[id] = true
	date, err := semantic.ParseCalendarDate(value)
	if err != nil {
		return validationeval.DateReading{}, fmt.Errorf("invalid Validation date Attribute %s: %w", id, err)
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: date}, nil
}

func (s *scenarioValidationSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	key := string(relation) + "\x00" + string(attribute)
	for _, declaration := range s.fixture.RelationKeys {
		if declaration.Relation != string(relation) || declaration.Attribute != string(attribute) {
			continue
		}
		s.relationReads[key] = true
		reading := validationeval.RelationReading{Coverage: validationeval.CoverageComplete}
		switch declaration.Coverage {
		case "", scenariotest.ValidationCoverageComplete:
		case scenariotest.ValidationCoveragePartial:
			reading.Coverage = validationeval.CoveragePartial
		default:
			return validationeval.RelationReading{}, fmt.Errorf("invalid coverage %q of Validation Relation input %s/%s", declaration.Coverage, relation, attribute)
		}
		if declaration.NullKeys < 0 || declaration.NotFetchedKeys < 0 {
			return validationeval.RelationReading{}, fmt.Errorf("negative key count of Validation Relation input %s/%s", relation, attribute)
		}
		for _, present := range declaration.Keys {
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValuePresent, Key: present})
		}
		for i := 0; i < declaration.NullKeys; i++ {
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNull})
		}
		for i := 0; i < declaration.NotFetchedKeys; i++ {
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNotFetched})
		}
		return reading, nil
	}
	return validationeval.RelationReading{}, fmt.Errorf("missing Validation Relation input %s/%s", relation, attribute)
}

// executeScenarioValidations never accepts a rule list from the Scenario.
// Only the Application's existing Canonical selection can supply rules.
//
// A rule decided by a declared operator is evaluated with the implementation
// the Workspace selected. The second result names, per rule, the
// implementation that was actually called; a rule of a built-in kind has no
// entry.
//
// A rule that cannot be decided from what the Fixture says was read stops the
// step with TEI_VALIDATION_UNDECIDED and the reason. It is not reported as
// satisfied or as not satisfied, and it cannot be expected as either.
func executeScenarioValidations(resolver driver.Resolver, appID core.CanonicalID, step scenariotest.Step, fixture scenariotest.ValidationFixture, selections []validationeval.ImplementationSelection, available []validationeval.NamedImplementation) (map[string]string, map[string]string, error) {
	results, called, err := evaluateScenarioValidations(resolver, appID, step, fixture, selections, available)
	return results, called, err
}

func evaluateScenarioValidations(resolver driver.Resolver, appID core.CanonicalID, step scenariotest.Step, fixture scenariotest.ValidationFixture, selections []validationeval.ImplementationSelection, available []validationeval.NamedImplementation) (map[string]string, map[string]string, error) {
	var calls []validationeval.ImplementationCall
	implementations, err := validationeval.ResolveImplementations(selections, available, func(call validationeval.ImplementationCall) {
		calls = append(calls, call)
	})
	if err != nil {
		return nil, nil, err
	}
	operators := validationeval.Operators{Declared: resolver.Operators(), Implementations: implementations}
	called := map[string]string{}
	results, err := runScenarioValidations(resolver, appID, step, fixture, func(rule semantic.Validation, source validationeval.StatefulSource) (validationeval.Result, error) {
		before := len(calls)
		decision, err := validationeval.Decide(rule, source, operators)
		if err != nil {
			return "", err
		}
		if decision.Undecided != nil {
			return "", fmt.Errorf("TEI_VALIDATION_UNDECIDED: Canonical Validation %s is neither satisfied nor not-satisfied: %s", rule.ID, decision.Undecided)
		}
		got := decision.Result
		if rule.Kind == semantic.ValidationOperator {
			// The implementation is asked about the contract's examples first
			// and about the fixture value last, so one rule makes one or more
			// calls. Every one of them must be the selected implementation of
			// this rule's operator.
			made := calls[before:]
			if len(made) == 0 {
				return "", fmt.Errorf("Canonical Validation %s was decided without calling the implementation selected for operator %s", rule.ID, rule.Operator)
			}
			for _, call := range made {
				if call.Operator != rule.Operator || call.Implementation != made[len(made)-1].Implementation {
					return "", fmt.Errorf("Canonical Validation %s was decided with a call that does not belong to operator %s", rule.ID, rule.Operator)
				}
			}
			called[string(rule.ID)] = made[len(made)-1].Implementation
		}
		return got, nil
	})
	if err != nil {
		return nil, nil, err
	}
	return results, called, nil
}

func runScenarioValidations(resolver driver.Resolver, appID core.CanonicalID, step scenariotest.Step, fixture scenariotest.ValidationFixture, evaluate func(semantic.Validation, validationeval.StatefulSource) (validationeval.Result, error)) (map[string]string, error) {
	rules, err := resolver.ResolveApplicationValidations(appID)
	if err != nil {
		return nil, err
	}
	if len(rules) != len(step.ExpectValidations) {
		return nil, fmt.Errorf("Scenario expects %d Validations but Application declares %d", len(step.ExpectValidations), len(rules))
	}
	source := &scenarioValidationSource{fixture: fixture, dateReads: map[string]bool{}, numberReads: map[string]bool{}, relationReads: map[string]bool{}}
	results := make(map[string]string, len(rules))
	for _, rule := range rules {
		expected, ok := step.ExpectValidations[string(rule.ID)]
		if !ok {
			return nil, fmt.Errorf("Scenario has no expected result for Canonical Validation %s", rule.ID)
		}
		got, err := evaluate(rule, source)
		if err != nil {
			return nil, err
		}
		if string(got) != expected {
			return nil, fmt.Errorf("Canonical Validation %s result=%s, expected=%s", rule.ID, got, expected)
		}
		results[string(rule.ID)] = string(got)
	}
	if len(source.numberReads) != len(fixture.Numbers)+len(fixture.NumberStates) || len(source.relationReads) != len(fixture.RelationKeys) {
		return nil, fmt.Errorf("Validation Fixture has unused numeric or relation inputs")
	}
	if len(source.dateReads) != len(fixture.Dates)+len(fixture.DateStates) {
		return nil, fmt.Errorf("Validation Fixture has unused date inputs")
	}
	return results, nil
}

// scenarioImplementationSelections carries the Workspace's choices to the
// evaluator unchanged.
func scenarioImplementationSelections(selected []workspace.OperatorImplementationSelection) []validationeval.ImplementationSelection {
	out := make([]validationeval.ImplementationSelection, 0, len(selected))
	for _, selection := range selected {
		out = append(out, validationeval.ImplementationSelection{
			Operator: core.CanonicalID(selection.Operator), Implementation: selection.Implementation,
		})
	}
	return out
}
