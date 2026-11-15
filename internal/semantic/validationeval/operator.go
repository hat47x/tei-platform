package validationeval

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
)

// OperatorImplementation is one implementation of a declared operator.
//
// An implementation is a pure function of the value the declaration says the
// operator takes. It receives no Source, so it cannot read another attribute,
// another row, a clock, or anything else: what a rule reads is decided by the
// operator declaration, never by the code that implements it.
type OperatorImplementation struct {
	// Operator is the declared operator this implements.
	Operator core.CanonicalID
	// Input is the input the implementation was written for. It must equal the
	// declared input, or the implementation is refused.
	Input semantic.OperatorInput
	// Number decides a rule for OperatorInputNumber. It receives a private
	// copy of the value.
	Number func(value *big.Rat) (bool, error)
}

// Operators is what an evaluation knows about declared operators: the
// declarations of the loaded model and the implementations the composition
// selected. Neither list is searched by position; identity decides.
type Operators struct {
	Declared        []semantic.Operator
	Implementations []OperatorImplementation
}

// EvaluateWithOperators evaluates rule like Evaluate and additionally decides
// rules of kind operator.
//
// An operator rule is evaluated only when exactly one declaration and exactly
// one implementation name its operator and both state the same input. A
// missing, duplicated, or mismatched implementation is an error: it is never
// replaced by another one and never reported as satisfied or not satisfied.
func EvaluateWithOperators(rule semantic.Validation, source Source, operators Operators) (Result, error) {
	if rule.Kind != semantic.ValidationOperator {
		return Evaluate(rule, source)
	}
	if source == nil {
		return "", fmt.Errorf("no Source is specified to evaluate semantic validation %s", rule.ID)
	}
	if rule.Operator == "" {
		return "", fmt.Errorf("semantic validation %s names no operator", rule.ID)
	}

	var declared []semantic.Operator
	for _, operator := range operators.Declared {
		if operator.ID == rule.Operator {
			declared = append(declared, operator)
		}
	}
	switch {
	case len(declared) == 0:
		return "", fmt.Errorf("semantic validation %s uses operator %s, which is not declared", rule.ID, rule.Operator)
	case len(declared) > 1:
		return "", fmt.Errorf("semantic validation %s uses operator %s, which is declared %d times", rule.ID, rule.Operator, len(declared))
	}
	contract := declared[0]

	var implementations []OperatorImplementation
	for _, implementation := range operators.Implementations {
		if implementation.Operator == rule.Operator {
			implementations = append(implementations, implementation)
		}
	}
	switch {
	case len(implementations) == 0:
		return "", fmt.Errorf("semantic validation %s uses operator %s, which has no selected implementation", rule.ID, rule.Operator)
	case len(implementations) > 1:
		return "", fmt.Errorf("semantic validation %s uses operator %s, which has %d implementations; exactly one must be selected", rule.ID, rule.Operator, len(implementations))
	}
	implementation := implementations[0]
	if implementation.Input != contract.Input {
		return "", fmt.Errorf("operator %s is declared to take %s, but its implementation takes %s", rule.Operator, contract.Input, implementation.Input)
	}

	switch contract.Input {
	case semantic.OperatorInputNumber:
		if implementation.Number == nil {
			return "", fmt.Errorf("the implementation of operator %s cannot take a number", rule.Operator)
		}
		value, err := source.Number(rule.Attribute)
		if err != nil {
			return "", fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, rule.Attribute, err)
		}
		if value == nil {
			// No value is not zero. The implementation is not asked.
			return "", fmt.Errorf("semantic validation %s has no evaluable number for attribute %s", rule.ID, rule.Attribute)
		}
		satisfied, err := implementation.Number(new(big.Rat).Set(value))
		if err != nil {
			return "", fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
		}
		if satisfied {
			return ResultSatisfied, nil
		}
		return ResultNotSatisfied, nil
	default:
		return "", fmt.Errorf("operator %s declares input %q, which has no evaluator", rule.Operator, contract.Input)
	}
}
