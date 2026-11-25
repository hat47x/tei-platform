package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// Decision sections 3 and 4 say a rule may give an input either an attribute or a
// constant. A row-scoped operator's second input used to be refused when it was
// a constant ("a row-scoped input reads a column of the rows, not one value"),
// so a rule like `... is below limit: 3 per sales.order.lines` could not be
// answered at all. The expectations below are written from that rule meaning:
// the constant holds in every row, so it is one limit for the whole relation.
//
//	order 1001: quantity 2, 0, 5  with limit 3 -> row 5 is above, and 0 is not
//	order 1002: quantity 3, 12    with limit 3 -> row 12 is above
//
// A witnessed counterexample settles the rule when the operator says so.

var rowInputConstantRule = semantic.Validation{
	ID:        "sales.order.lines.quantity_at_most",
	Symbol:    "quantity_at_most",
	Kind:      semantic.ValidationOperator,
	Attribute: "sales.order_line.quantity",
	Relation:  "sales.order.lines",
	Operator:  "sales.at_most",
	Inputs: []semantic.ValidationInputAssignment{
		{Name: "limit", Value: "3"},
	},
}

func rowInputConstantContract() semantic.Operator {
	return semantic.Operator{
		ID:     "sales.at_most",
		Symbol: "at_most",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber},
		},
		RefutesOnSomeRows: true,
	}
}

// withinConstantLimit is a plain reading of the contract: every row's value is
// at most the limit, and the limit is one value for the whole relation.
func withinConstantLimit() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "sales.at_most",
		Input:    semantic.OperatorInputNumber,
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			values, ok := rows["value"]
			if !ok {
				return validationeval.TriMaybe, nil
			}
			limits, ok := rows["limit"]
			if !ok {
				return validationeval.TriMaybe, nil
			}
			if len(limits) != len(values) {
				return validationeval.TriMaybe, nil
			}
			var limit *big.Rat
			for _, row := range limits {
				if row.State != validationeval.ValuePresent || row.Value == nil {
					return validationeval.TriMaybe, nil
				}
				if limit == nil {
					limit = row.Value
					continue
				}
				if row.Value.Cmp(limit) != 0 {
					return validationeval.TriMaybe, nil
				}
			}
			if limit == nil {
				return validationeval.TriMaybe, nil
			}
			for _, row := range values {
				if row.State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if row.Value.Cmp(limit) > 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func rowInputConstantDecision(t *testing.T, coverage validationeval.Coverage, values []validationeval.NumberReading) validationeval.Decision {
	t.Helper()
	source := &rowsSource{
		coverage: coverage,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			"sales.order_line.quantity": values,
		},
	}
	decision, err := validationeval.Decide(rowInputConstantRule, source, validationeval.Operators{
		Declared:        []semantic.Operator{rowInputConstantContract()},
		Implementations: []validationeval.OperatorImplementation{withinConstantLimit()},
	})
	if err != nil {
		t.Fatalf("coverage=%q: %v", coverage, err)
	}
	return decision
}

// The constant is read once for the whole relation: the operator sees the same
// limit in every row, and only the subject's column comes from the relation.
func TestARowScopedInputMayBeAConstant(t *testing.T) {
	source := &rowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			"sales.order_line.quantity": {
				{State: validationeval.ValuePresent, Value: big.NewRat(2, 1)},
				{State: validationeval.ValuePresent, Value: big.NewRat(5, 1)},
			},
		},
	}
	decision, err := validationeval.Decide(rowInputConstantRule, source, validationeval.Operators{
		Declared:        []semantic.Operator{rowInputConstantContract()},
		Implementations: []validationeval.OperatorImplementation{withinConstantLimit()},
	})
	if err != nil {
		// The old refusal named the constant as "a row-scoped input reads a
		// column of the rows, not one value"; a constant is now read instead.
		if strings.Contains(err.Error(), "reads a column of the rows, not one value") {
			t.Fatalf("a constant row-scoped input was refused: %v", err)
		}
		t.Fatalf("err=%v", err)
	}
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied (5 is above the limit 3)", decision)
	}
	// Only the subject's attribute was read from the relation; the constant
	// needed no column.
	if len(source.reads) != 1 || source.reads[0] != "sales.order_line.quantity" {
		t.Fatalf("reads=%v, want only the subject's column", source.reads)
	}
}

// Every row is at most the constant, so the rule holds.
func TestARowScopedConstantLimitThatHolds(t *testing.T) {
	decision := rowInputConstantDecision(t, validationeval.CoverageComplete, []validationeval.NumberReading{
		{State: validationeval.ValuePresent, Value: big.NewRat(2, 1)},
		{State: validationeval.ValuePresent, Value: big.NewRat(3, 1)},
	})
	if decision.Result != validationeval.ResultSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want satisfied", decision)
	}
}

// A partly read relation with a witnessed counterexample is not-satisfied only
// because the contract states `refutes on some rows`; the constant does not
// change that rule.
func TestARowScopedConstantLimitOnAPartlyReadRelation(t *testing.T) {
	decision := rowInputConstantDecision(t, validationeval.CoveragePartial, []validationeval.NumberReading{
		{State: validationeval.ValuePresent, Value: big.NewRat(10, 1)},
	})
	if decision.Result != validationeval.ResultNotSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want not-satisfied", decision)
	}
}

// A value that is neither an attribute nor a decimal is still refused, and the
// refusal says so instead of reading the text as a number.
func TestARowScopedInputThatIsNeitherAttributeNorConstantIsRefused(t *testing.T) {
	rule := rowInputConstantRule
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "limit", Value: "not-a-number"}}
	_, err := validationeval.Decide(rule, &rowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			"sales.order_line.quantity": {{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}},
		},
	}, validationeval.Operators{
		Declared:        []semantic.Operator{rowInputConstantContract()},
		Implementations: []validationeval.OperatorImplementation{withinConstantLimit()},
	})
	if err == nil {
		t.Fatal("an unreadable value was accepted as a constant")
	}
	if !strings.Contains(err.Error(), "neither an attribute nor a decimal") {
		t.Fatalf("the refusal does not say what is wrong: %v", err)
	}
}
