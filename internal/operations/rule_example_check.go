package operations

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/syntax"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// Diagnostic codes of the example check.
const (
	// RuleExampleMismatchCode: a rule answers differently from what the
	// example expects. Either the rule or the example is wrong; the check
	// cannot tell which, and reports the place the expectation is written.
	RuleExampleMismatchCode = "RULE_EXAMPLE_MISMATCH"
	// RuleExampleNotCheckedCode: the rule could not be asked at all, for
	// instance because no implementation is selected for its operator. An
	// example that was not checked is not reported as agreeing.
	RuleExampleNotCheckedCode = "RULE_EXAMPLE_NOT_CHECKED"
)

// exampleSource reads one example the way a rule reads records.
//
// For a rule about one record it answers from that row. For a rule about all
// the rows it answers with every row, and says the rows are all there are:
// the author wrote the whole example. An item the table has no column for was
// not written, so it is reported as not fetched, never as empty.
type exampleSource struct {
	example workspace.RuleExample
	row     int
}

func (s exampleSource) column(attribute core.CanonicalID) (int, bool) {
	for i, item := range s.example.Items {
		if item.Attribute == attribute {
			return i, true
		}
	}
	return 0, false
}

func (s exampleSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	if s.row < 0 {
		return validationeval.NumberReading{}, fmt.Errorf("a rule about all the rows asked for the single value %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	if cell.Text == "" {
		return validationeval.NumberReading{State: validationeval.ValueNull}, nil
	}
	value, err := semantic.ParseOperatorCaseValue(cell.Text)
	if err != nil {
		return validationeval.NumberReading{}, err
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: value}, nil
}

func (s exampleSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	if s.row >= 0 {
		return validationeval.RelationReading{}, fmt.Errorf("a rule about one record asked for the rows of relation %s", relation)
	}
	reading := validationeval.RelationReading{Coverage: validationeval.CoverageComplete}
	column, written := s.column(attribute)
	for _, row := range s.example.Rows {
		switch {
		case !written:
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNotFetched})
		case row[column].Text == "":
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNull})
		default:
			key, err := exampleKey(s.example.Items[column], row[column].Text)
			if err != nil {
				return validationeval.RelationReading{}, err
			}
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValuePresent, Key: key})
		}
	}
	return reading, nil
}

// exampleKey is what two cells are compared by. Numbers are compared as
// numbers, so 1 and 1.0 are the same key; anything else is compared as
// written.
func exampleKey(item workspace.RuleExampleItem, text string) (string, error) {
	if !item.Numeric {
		return "text:" + text, nil
	}
	value, err := semantic.ParseOperatorCaseValue(text)
	if err != nil {
		return "", err
	}
	return "number:" + new(big.Rat).Set(value).RatString(), nil
}

// CheckRuleExamples asks every rule an example names about the example's
// records and reports each answer that differs from the expected one.
//
// It reads the examples and the World of one Load and nothing else. A rule
// decided by a declared operator is asked through the implementation the
// Workspace selected among available. The result is empty when every example
// was checked and agrees.
func CheckRuleExamples(loaded workspace.Result, available []validationeval.NamedImplementation) []workspace.ProjectedDiagnostic {
	if len(loaded.RuleExamples) == 0 {
		return nil
	}
	rules := map[core.CanonicalID]semantic.Validation{}
	operators := validationeval.Operators{}
	for _, namespace := range loaded.World.Semantics {
		for _, rule := range namespace.Validations {
			rules[rule.ID] = rule
		}
		operators.Declared = append(operators.Declared, namespace.Operators...)
	}
	selections := make([]validationeval.ImplementationSelection, 0, len(loaded.OperatorImplementations))
	for _, selection := range loaded.OperatorImplementations {
		selections = append(selections, validationeval.ImplementationSelection{
			Operator: core.CanonicalID(selection.Operator), Implementation: selection.Implementation,
		})
	}
	implementations, selectionErr := validationeval.ResolveImplementations(selections, available, nil)
	if selectionErr == nil {
		operators.Implementations = implementations
	}

	var out []workspace.ProjectedDiagnostic
	report := func(example workspace.RuleExample, expectation workspace.RuleExampleExpectation, code, message string) {
		ruleSubject := subject.Subject{Kind: "semantic-validation", ID: string(expectation.Rule)}
		out = append(out, workspace.ProjectedDiagnostic{
			Diagnostic: validation.Diagnostic{Code: code, Severity: validation.Error, Message: message, Subject: &ruleSubject},
			Sources: []sourceindex.Reference{{
				Subject:      ruleSubject,
				ArtifactPath: example.File,
				Range:        expectation.Range,
				Role:         sourceindex.RoleAuthoring,
			}},
		})
	}
	for _, example := range loaded.RuleExamples {
		for _, expectation := range example.Expectations {
			rule, known := rules[expectation.Rule]
			if !known {
				report(example, expectation, RuleExampleNotCheckedCode,
					fmt.Sprintf("the example expects an answer of rule %s, which the loaded model does not have", expectation.Rule))
				continue
			}
			if rule.Kind == semantic.ValidationOperator && selectionErr != nil {
				report(example, expectation, RuleExampleNotCheckedCode,
					fmt.Sprintf("rule %s was not asked about the example: %v", rule.ID, selectionErr))
				continue
			}
			decision, err := validationeval.Decide(rule, exampleSource{example: example, row: expectation.Row}, operators)
			if err != nil {
				report(example, expectation, RuleExampleNotCheckedCode,
					fmt.Sprintf("rule %s was not asked about the example: %v", rule.ID, err))
				continue
			}
			answer, detail := syntax.RuleExampleExpect(decision.Result), ""
			if decision.Undecided != nil {
				answer, detail = syntax.RuleExampleUndecided, fmt.Sprintf(" (%s)", decision.Undecided)
			}
			if answer != expectation.Expect {
				report(example, expectation, RuleExampleMismatchCode,
					fmt.Sprintf("the example expects %s, but rule %s answers %s%s", expectation.Expect, rule.ID, answer, detail))
			}
		}
	}
	return out
}
