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
	numberReads   map[string]bool
	relationReads map[string]bool
}

func (s *scenarioValidationSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	id := string(attribute)
	value, ok := s.fixture.Numbers[id]
	if !ok {
		return nil, fmt.Errorf("missing Validation numeric Attribute %s", id)
	}
	s.numberReads[id] = true
	n, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("invalid Validation numeric Attribute %s", id)
	}
	return n, nil
}

func (s *scenarioValidationSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	key := string(relation) + "\x00" + string(attribute)
	for _, declaration := range s.fixture.RelationKeys {
		if declaration.Relation == string(relation) && declaration.Attribute == string(attribute) {
			s.relationReads[key] = true
			return append([]string(nil), declaration.Keys...), nil
		}
	}
	return nil, fmt.Errorf("missing Validation Relation input %s/%s", relation, attribute)
}

// executeScenarioValidations never accepts a rule list from the Scenario.
// Only the Application's existing Canonical selection can supply rules.
//
// A rule decided by a declared operator is evaluated with the implementation
// the Workspace selected. The second result names, per rule, the
// implementation that was actually called; a rule of a built-in kind has no
// entry.
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
	results, err := runScenarioValidations(resolver, appID, step, fixture, func(rule semantic.Validation, source validationeval.Source) (validationeval.Result, error) {
		before := len(calls)
		got, err := validationeval.EvaluateWithOperators(rule, source, operators)
		if err != nil {
			return "", err
		}
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

func runScenarioValidations(resolver driver.Resolver, appID core.CanonicalID, step scenariotest.Step, fixture scenariotest.ValidationFixture, evaluate func(semantic.Validation, validationeval.Source) (validationeval.Result, error)) (map[string]string, error) {
	rules, err := resolver.ResolveApplicationValidations(appID)
	if err != nil {
		return nil, err
	}
	if len(rules) != len(step.ExpectValidations) {
		return nil, fmt.Errorf("Scenario expects %d Validations but Application declares %d", len(step.ExpectValidations), len(rules))
	}
	source := &scenarioValidationSource{fixture: fixture, numberReads: map[string]bool{}, relationReads: map[string]bool{}}
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
	if len(source.numberReads) != len(fixture.Numbers) || len(source.relationReads) != len(fixture.RelationKeys) {
		return nil, fmt.Errorf("Validation Fixture has unused numeric or relation inputs")
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
