package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A row-scoped input may declare `absent as <name>`, and the rule may name that
// mapping. The mapping says how a value the row certainly does not hold is
// read. It does not say anything about a column the query never returned:
// nothing is known about that value, including whether it is empty (decision
// section 5, the table's "読んでいない" row).
//
// withAbsentMapping rewrote both, so a rule over a column the Query omitted was
// decided from a value that was never read: measured before the fix, a
// not-fetched limit column answered not-satisfied instead of undecided (#4359).
func TestTheRowMappingReadsOnlyRowsThatCertainlyHoldNoValue(t *testing.T) {
	const (
		quantity = "sales.order_line.quantity"
		limit    = "sales.order_line.limit"
	)
	data := semantic.Data{ID: "sales.order_line", Symbol: "order_line", Attributes: []semantic.Attribute{
		{ID: quantity, Symbol: "quantity", TypeRef: "sales.amount"},
		{ID: limit, Symbol: "limit", TypeRef: "sales.amount"},
	}}
	relation := semantic.Relation{
		ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line",
		Cardinality: semantic.CardinalityMany,
	}
	namespaces := []semantic.Namespace{{ID: "sales",
		Types: []semantic.Type{{ID: "sales.amount", Symbol: "amount", Kind: semantic.TypeAlias, Underlying: "std.decimal"}},
		Data:  []semantic.Data{data},
	}}

	contract := semantic.Operator{
		ID: "sales.at_most", Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true, AbsentAs: "open"},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.at_most",
		Input:    semantic.OperatorInputNumber,
		Absent: func(mapping string) (*big.Rat, bool) {
			// The mapping `open` reads an absent limit as zero.
			if mapping == "open" {
				return new(big.Rat), true
			}
			return nil, false
		},
		NumberRows: func(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			limits, held := rows["limit"]
			if !held {
				return validationeval.TriMaybe, nil
			}
			values := rows["value"]
			for i := range limits {
				if limits[i].State != validationeval.ValuePresent {
					// One row whose limit could not be read leaves the
					// question open rather than assumed.
					return validationeval.TriMaybe, nil
				}
				if i < len(values) && values[i].State == validationeval.ValuePresent &&
					values[i].Value.Sign() > 0 && limits[i].Value.Sign() == 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_at_most_limit", Kind: semantic.ValidationOperator,
		Attribute: quantity, Relation: "sales.order.lines", Operator: "sales.at_most",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "limit", Value: limit, AbsentAs: "open"},
		},
	}
	operators := validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}

	build := func(rows []driver.ContractQueryRow) *validationread.Detail {
		d, err := validationread.NewDetail(namespaces, relation, data, rows, validationeval.CoverageComplete)
		if err != nil {
			t.Fatalf("cannot build the Detail: %v", err)
		}
		return d
	}

	t.Run("a limit the row holds as empty is read under the mapping", func(t *testing.T) {
		// Both rows hold the limit as SQL NULL: certainly no value.
		detail := build([]driver.ContractQueryRow{
			{Attributes: map[core.CanonicalID]any{quantity: "2", limit: nil}},
			{Attributes: map[core.CanonicalID]any{quantity: "5", limit: nil}},
		})
		decisions, err := detail.Decide(rule, operators)
		if err != nil {
			t.Fatalf("the mapped read failed: %v", err)
		}
		if decisions.Relation == nil || decisions.Relation.Result != validationeval.ResultNotSatisfied {
			t.Fatalf("the mapping was not used: %+v", decisions.Relation)
		}
	})

	t.Run("a limit the query did not return stays undecided", func(t *testing.T) {
		// The column is absent from the rows: it was not fetched.
		detail := build([]driver.ContractQueryRow{
			{Attributes: map[core.CanonicalID]any{quantity: "2"}},
			{Attributes: map[core.CanonicalID]any{quantity: "5"}},
		})
		decisions, err := detail.Decide(rule, operators)
		if err != nil {
			t.Fatalf("an unread limit must not be an error: %v", err)
		}
		if decisions.Relation == nil || decisions.Relation.Undecided == nil {
			t.Fatalf("an unread limit was decided: %+v", decisions.Relation)
		}
		if decisions.Relation.Undecided.Reason != validationeval.UndecidedValueNotFetched {
			t.Fatalf("reason=%q, want %q", decisions.Relation.Undecided.Reason,
				validationeval.UndecidedValueNotFetched)
		}
	})
}
