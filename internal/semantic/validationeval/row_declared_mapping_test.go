package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// Decision sections 3-5 and 8.1's table fix whose word decides how an absent
// value is read: the DECLARATION names the mappings it can offer, and the RULE
// names the one it wants with `or <name>`. A declaration that states
//
//	takes limit: number of each row absent as zero
//
// therefore says "a rule of this operator MAY ask for zero"; it does not say
// that every rule of this operator reads every absence as zero. Section 8.1
// states the difference in one row of its table:
//
//	NULL | orなし（宣言のみ） | NULLのまま／一件では判定不能
//
// The per-record path already reads it that way (the assignment's mapping
// decides, #5890). The row-scoped path used the input's DECLARED mapping as
// its default, so the same operator answered a row whose limit is NULL as
// not-satisfied (limit read as zero) in a rule that asked for no mapping,
// while an unreadable mapping in the same rule left that same row undecided.
//
// The expectations below are written from that table, not read back from the
// code: a row that holds no value and whose input states no mapping is
// undecided with the absence's own reason, and the same row is read as the
// named value only when the rule names the mapping the declaration offers.

// declaredMappingContract is a row-scoped operator whose second input offers
// the mapping `zero`.
func declaredMappingContract() semantic.Operator {
	return semantic.Operator{
		ID:     "sales.at_most_with_declared_mapping",
		Symbol: "at_most_with_declared_mapping",
		Input:  semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true, AbsentAs: "zero"},
		},
		RefutesOnSomeRows: true,
	}
}

// declaredMappingRule is the same rule twice: once without `or`, once naming
// the mapping the declaration offers.
func declaredMappingRule(absent string) semantic.Validation {
	assignment := semantic.ValidationInputAssignment{Name: "limit", Value: "sales.order_line.unit_price"}
	if absent != "" {
		assignment.AbsentAs = absent
	}
	return semantic.Validation{
		ID:        "sales.order_line.quantity_at_most_price",
		Symbol:    "quantity_at_most_price",
		Kind:      semantic.ValidationOperator,
		Attribute: "sales.order_line.quantity",
		Relation:  "sales.order.lines",
		Operator:  "sales.at_most_with_declared_mapping",
		Inputs:    []semantic.ValidationInputAssignment{assignment},
	}
}

// declaredMappingImplementation is a plain reading of "every row's value is at
// most its limit", with zero read for an absence under `zero`.
func declaredMappingImplementation() validationeval.OperatorImplementation {
	zero := new(big.Rat)
	return validationeval.OperatorImplementation{
		Operator: "sales.at_most_with_declared_mapping",
		Input:    semantic.OperatorInputNumber,
		Absent: func(mapping string) (*big.Rat, bool) {
			if mapping != "zero" {
				return nil, false
			}
			return zero, true
		},
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			values, held := rows["value"]
			if !held {
				return validationeval.TriMaybe, nil
			}
			limits, held := rows["limit"]
			if !held || len(values) != len(limits) {
				return validationeval.TriMaybe, nil
			}
			for i := range values {
				if values[i].State != validationeval.ValuePresent || limits[i].State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if values[i].Value.Cmp(limits[i].Value) > 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
}

func declareMappingRows(values, limits []validationeval.NumberReading) *rowsSource {
	return &rowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			"sales.order_line.quantity":   values,
			"sales.order_line.unit_price": limits,
		},
	}
}

func decideDeclaredMapping(t *testing.T, rule semantic.Validation, source *rowsSource) validationeval.Decision {
	t.Helper()
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{declaredMappingContract()},
		Implementations: []validationeval.OperatorImplementation{declaredMappingImplementation()},
	})
	if err != nil {
		t.Fatalf("%s: %v", rule.ID, err)
	}
	return decision
}

// Without `or`, the declaration's mapping is not used: the row keeps its own
// reason, so a quantity above a NULL limit is NOT reported as not-satisfied.
func TestADeclaredMappingTheRuleDoesNotNameLeavesTheRowUndecided(t *testing.T) {
	// Order 1002's second line: quantity 12, unit_price NULL. If the NULL were
	// read as zero the answer would be not-satisfied, which is exactly the
	// reading this test refuses.
	source := declareMappingRows(
		[]validationeval.NumberReading{present(3), present(12)},
		[]validationeval.NumberReading{present(100), {State: validationeval.ValueNull}},
	)
	decision := decideDeclaredMapping(t, declaredMappingRule(""), source)
	if decision.Result != "" || decision.Undecided == nil {
		t.Fatalf("decision=%+v, want undecided: the declaration offers a mapping, it does not apply one", decision)
	}
	if decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("reason=%q, want %q", decision.Undecided.Reason, validationeval.UndecidedValueNull)
	}
	if decision.Undecided.Attribute != "sales.order_line.unit_price" {
		t.Fatalf("attribute=%q, want the attribute that holds no value", decision.Undecided.Attribute)
	}
}

// Naming the mapping the declaration offers reads the absence as that value,
// so the same rows are decided.
func TestADeclaredMappingTheRuleNamesReadsTheAbsence(t *testing.T) {
	source := declareMappingRows(
		[]validationeval.NumberReading{present(3), present(12)},
		[]validationeval.NumberReading{present(100), {State: validationeval.ValueNull}},
	)
	decision := decideDeclaredMapping(t, declaredMappingRule("zero"), source)
	// 12 is above the mapped limit 0: the same counterexample the declared
	// mapping would have produced, but reached only because the rule asked.
	if decision.Result != validationeval.ResultNotSatisfied || decision.Undecided != nil {
		t.Fatalf("decision=%+v, want not-satisfied once the rule names the mapping", decision)
	}
}

// A rule may name a mapping the declaration never offered for that input; the
// syntax refuses it, so the evaluator is not asked to invent one. An
// implementation that cannot read the named mapping leaves the decision
// undecided instead of reading the absence as zero.
func TestAMappingTheImplementationCannotReadStaysUndecided(t *testing.T) {
	implementation := declaredMappingImplementation()
	implementation.Absent = nil
	source := declareMappingRows(
		[]validationeval.NumberReading{present(3)},
		[]validationeval.NumberReading{{State: validationeval.ValueNull}},
	)
	decision, err := validationeval.Decide(declaredMappingRule("zero"), source, validationeval.Operators{
		Declared:        []semantic.Operator{declaredMappingContract()},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if decision.Result != "" || decision.Undecided == nil ||
		decision.Undecided.Reason != validationeval.UndecidedValueNull {
		t.Fatalf("decision=%+v, want undecided value-null", decision)
	}
}
