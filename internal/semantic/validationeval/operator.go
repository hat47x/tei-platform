package validationeval

import (
	"fmt"
	"math/big"
	"sort"
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
	// NumberRows decides a rule whose operator reads every row of a relation.
	// It receives, per input name, the states of the rows that were read in
	// the order they were read, and answers TriYes, TriNo or TriMaybe. It sees
	// no Source, so what a rule reads stays decided by the declaration.
	NumberRows func(rows map[string][]NumberReading) (Tri, error)
	// Absent reads an absent value under the mapping the rule named. A rule may
	// write `or <name>` after an input, and the declaration may state
	// `absent as <name>`; the mapping says how an absent value is to be read.
	//
	// It is asked only when the input held no value and the rule named a
	// mapping. A nil Absent, or ok=false for that mapping, means this
	// implementation cannot read the absence that way, and the decision stays
	// undecided instead of reading the absence as zero or as present.
	Absent func(mapping string) (*big.Rat, bool)
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

	// An operator that reads every row of a relation is decided by a row
	// function, and an operator that reads one record is decided by a value
	// function. The one that does not apply is not asked for: a
	// single-value implementation cannot answer a rule about all the rows, and
	// a row function has no value to be handed.
	if contract.ReadsEveryRow() {
		if implementation.NumberRows == nil {
			return none, fmt.Errorf("the implementation of operator %s cannot decide a rule that reads every row of a relation", rule.Operator)
		}
		// An example states one value, so it can state the meaning of a
		// single-input operator only. A contract that reads every row has no
		// example this shape can state, and the row function is not asked
		// about one it never saw.
		if len(contract.Cases) > 0 {
			return none, fmt.Errorf("operator %s reads every row of a relation and states %d examples; an example states one value, so it cannot state a rule that is decided from the rows together", rule.Operator, len(contract.Cases))
		}
		return implementation, nil
	}
	if implementation.NumberRows != nil && implementation.Number == nil && implementation.Numbers == nil {
		return none, fmt.Errorf("the implementation of operator %s decides a rule that reads every row of a relation, but the operator reads one record", rule.Operator)
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
	inputs := contractInputs(contract)
	multi := len(inputs) > 1
	for _, example := range contract.Cases {
		// An example states either one value or one value per declared input.
		// Which form it uses must match the signature: a multi-input example
		// read as a single value would leave the other inputs unstated, and a
		// single-input operator given per-input columns would be read as
		// something the author never wrote.
		if example.Inputs != nil {
			if !multi {
				// The design says the table's columns are the operator's input
				// names, so a single-input operator may write `| <name> | expect |`.
				// That is the single-value form with the input named by its own
				// name, not a multi-input example.
				if len(example.Inputs) == 1 {
					if value, ok := example.Inputs[inputs[0].Name]; ok {
						example = semantic.OperatorCase{Value: value, Satisfied: example.Satisfied}
					} else {
						return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares the input %q, but an example states %s",
							contract.ID, inputs[0].Name, operatorExampleNames(example.Inputs))
					}
				} else {
					return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares one input, but an example states the inputs %s",
						contract.ID, operatorExampleNames(example.Inputs))
				}
			} else {
				for _, input := range inputs {
					if _, stated := example.Inputs[input.Name]; !stated {
						return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s example states %s, but input %q is not stated; an example that leaves an input unread cannot be checked",
							contract.ID, operatorExampleNames(example.Inputs), input.Name)
					}
				}
				for name := range example.Inputs {
					known := false
					for _, input := range inputs {
						if input.Name == name {
							known = true
						}
					}
					if !known {
						return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s example states %q, which is not one of its inputs (%s)",
							contract.ID, name, operatorInputNames(inputs))
					}
				}
			}
		} else if multi {
			return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares %d inputs (%s), but its example carries one value; a multi-input example states one column per input",
				contract.ID, len(inputs), operatorInputNames(inputs))
		}

		values := map[string]*big.Rat{}
		if example.Inputs != nil {
			for name, text := range example.Inputs {
				value, err := semantic.ParseOperatorCaseValue(text)
				if err != nil {
					return fmt.Errorf("operator %s has an unreadable example (%s): %w", contract.ID, name, err)
				}
				values[name] = value
			}
		} else {
			value, err := semantic.ParseOperatorCaseValue(example.Value)
			if err != nil {
				return fmt.Errorf("operator %s has an unreadable example: %w", contract.ID, err)
			}
			// A single-input operator may name its input (`takes amount: number`).
			// The example's one value belongs to that input, whatever it is
			// called, so the implementation is never handed a nil value.
			values[inputs[0].Name] = value
		}

		var (
			got bool
			err error
		)
		if implementation.Numbers != nil {
			got, err = implementation.Numbers(values)
		} else {
			got, err = implementation.Number(values[inputs[0].Name])
		}
		if err != nil {
			return fmt.Errorf("the implementation of operator %s could not decide the contract's example %s: %w", contract.ID, operatorExampleText(example), err)
		}
		if got != example.Satisfied {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that %s is %s, but the selected implementation answers %s",
				contract.ID, operatorExampleText(example), operatorCaseWord(example.Satisfied), operatorCaseWord(got))
		}
	}
	return nil
}

// operatorExampleNames lists the inputs an example states, in declaration
// order when the contract names them, for a stable diagnostic.
func operatorExampleNames(stated map[string]string) string {
	names := make([]string, 0, len(stated))
	for name := range stated {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// operatorExampleText renders one example for a diagnostic.
func operatorExampleText(example semantic.OperatorCase) string {
	if example.Inputs == nil {
		return example.Value
	}
	return "(" + operatorExampleNames(example.Inputs) + ")"
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
