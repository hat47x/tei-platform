package validationeval

import (
	"fmt"
	"math/big"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// OperatorImplementation is one implementation of a declared operator.
//
// An implementation is a pure function of the values the declaration says the
// operator takes. It receives no Source, so it cannot read another attribute,
// another row, a clock, or anything else: what a rule reads is decided by the
// operator declaration, never by the code that implements it.
//
// For a single-input operator, Number is used. For a multi-input operator,
// Numbers is used. Exactly one of them is non-nil for any implementation.
type OperatorImplementation struct {
	// Operator is the declared operator this implements.
	Operator core.CanonicalID
	// Input is the first input's type, kept for backward compatibility with
	// single-input operators.
	Input semantic.OperatorInput
	// Number decides a rule for a single-input operator. It receives a private
	// copy of the value.
	Number func(value *big.Rat) (bool, error)
	// Numbers decides a rule for a multi-input operator. It receives the input
	// name to value map. Values are non-nil only for inputs that are present.
	Numbers func(values map[string]*big.Rat) (bool, error)
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
// one implementation name its operator. A missing, duplicated, or mismatched
// implementation is an error: it is never replaced by another one and never
// reported as satisfied or not satisfied.
func EvaluateWithOperators(rule semantic.Validation, source Source, operators Operators) (Result, error) {
	if rule.Kind != semantic.ValidationOperator {
		return Evaluate(rule, source)
	}
	if source == nil {
		return "", fmt.Errorf("no Source is specified to evaluate semantic validation %s", rule.ID)
	}
	decision, err := decide(rule, legacySource{source}, operators, true)
	if err != nil {
		return "", err
	}
	return decision.Result, nil
}

// selectOperatorImplementation finds the one implementation an operator rule
// is decided with. It reads no real data.
func selectOperatorImplementation(rule semantic.Validation, operators Operators) (OperatorImplementation, error) {
	none := OperatorImplementation{}
	if rule.Operator == "" {
		return none, fmt.Errorf("semantic validation %s names no operator", rule.ID)
	}

	var declared []semantic.Operator
	for _, operator := range operators.Declared {
		if operator.ID == rule.Operator {
			declared = append(declared, operator)
		}
	}
	switch {
	case len(declared) == 0:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which is not declared", rule.ID, rule.Operator)
	case len(declared) > 1:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which is declared %d times", rule.ID, rule.Operator, len(declared))
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
		return none, fmt.Errorf("semantic validation %s uses operator %s, which has no selected implementation", rule.ID, rule.Operator)
	case len(implementations) > 1:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which has %d implementations; exactly one must be selected", rule.ID, rule.Operator, len(implementations))
	}
	implementation := implementations[0]
	if implementation.Input != contract.Input {
		return none, fmt.Errorf("operator %s is declared to take %s, but its implementation takes %s", rule.Operator, contract.Input, implementation.Input)
	}

	// Reject an implementation that does not match the arity of the contract.
	hasMulti := implementation.Numbers != nil
	hasSingle := implementation.Number != nil
	if !hasMulti && !hasSingle {
		return none, fmt.Errorf("the implementation of operator %s has no decision function", rule.Operator)
	}
	if hasMulti && hasSingle {
		return none, fmt.Errorf("the implementation of operator %s provides both single and multi-input functions", rule.Operator)
	}
	multiInput := len(contract.Inputs) > 1
	if multiInput && !hasMulti {
		return none, fmt.Errorf("operator %s declares %d inputs but its implementation takes a single value", rule.Operator, len(contract.Inputs))
	}
	if !multiInput && hasMulti {
		return none, fmt.Errorf("operator %s declares one input but its implementation takes multiple values", rule.Operator)
	}

	// The contract's own examples come first. An implementation that
	// answers one of them differently is not an implementation of this
	// operator, so it is not asked about real data.
	if err := checkOperatorCases(contract, implementation); err != nil {
		return none, err
	}
	return implementation, nil
}

// checkOperatorCases asks the implementation about every example the contract
// states and refuses the implementation at the first different answer.
//
// An example carries one value, so it can state the meaning of a single-input
// operator only. A multi-input contract with examples is refused rather than
// read as if its one value bound the first input: the remaining inputs would
// go unstated and the example would say something the author never wrote.
func checkOperatorCases(contract semantic.Operator, implementation OperatorImplementation) error {
	if len(contract.Cases) > 0 && len(contract.Inputs) > 1 {
		return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares %d inputs (%s), but its examples carry one value; an example can state a single-input operator only",
			contract.ID, len(contract.Inputs), operatorInputNames(contract.Inputs))
	}
	for _, example := range contract.Cases {
		value, err := semantic.ParseOperatorCaseValue(example.Value)
		if err != nil {
			return fmt.Errorf("operator %s has an unreadable example: %w", contract.ID, err)
		}
		var got bool
		if implementation.Number != nil {
			got, err = implementation.Number(value)
		} else {
			got, err = implementation.Numbers(map[string]*big.Rat{"value": value})
		}
		if err != nil {
			return fmt.Errorf("the implementation of operator %s could not decide the contract's example %s: %w", contract.ID, example.Value, err)
		}
		if got != example.Satisfied {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that %s is %s, but the selected implementation answers %s",
				contract.ID, example.Value, operatorCaseWord(example.Satisfied), operatorCaseWord(got))
		}
	}
	return nil
}

func operatorCaseWord(satisfied bool) Result {
	if satisfied {
		return ResultSatisfied
	}
	return ResultNotSatisfied
}

// operatorInputNames lists the declared input names in signature order, so a
// diagnostic can name the inputs an example would have to state.
func operatorInputNames(inputs []semantic.OperatorInputDeclaration) string {
	names := make([]string, 0, len(inputs))
	for _, input := range inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
