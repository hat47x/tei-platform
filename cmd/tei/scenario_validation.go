package main

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/scenariotest"
	"tei/internal/semantic/validationeval"
)

type scenarioValidationSource struct {
	fixture scenariotest.ValidationFixture
	numberReads map[string]bool
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
func executeScenarioValidations(resolver driver.Resolver, appID core.CanonicalID, step scenariotest.Step, fixture scenariotest.ValidationFixture) (map[string]string, error) {
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
		got, err := validationeval.Evaluate(rule, source)
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
