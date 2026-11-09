package derivationvalidation

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/derivationref"
	"tei/internal/semantic/validationeval"
)

var errFixtureDerivedNull = errors.New("derived NULL cannot be evaluated as positive")

// canonicalDerivedFixtureSource is test-only. It bridges an explicitly named
// derived attribute to the existing Canonical Validation Source interface.
// It does not resolve a Binding, grant authority or supply a database snapshot.
type canonicalDerivedFixtureSource struct {
	target     core.CanonicalID
	value      derivationref.Input
	derivedErr error
}

func (s canonicalDerivedFixtureSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	if attribute != s.target {
		return nil, fmt.Errorf("derived attribute %s is not available for %s", s.target, attribute)
	}
	if s.derivedErr != nil {
		return nil, s.derivedErr
	}
	switch s.value.State {
	case derivationref.Null:
		return nil, errFixtureDerivedNull
	case derivationref.Present:
		value, ok := new(big.Rat).SetString(s.value.Decimal)
		if !ok {
			return nil, fmt.Errorf("%w: invalid derived decimal", derivationref.ErrOperator)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("%w: derived state %s", derivationref.ErrInput, s.value.State)
	}
}

func (canonicalDerivedFixtureSource) RelationKeys(_, _ core.CanonicalID) ([]string, error) {
	return nil, errors.New("relation values are not available in this scalar fixture")
}

// D15 M2 partial integration: call the actual D1 operator, then the existing
// Canonical positive Validation evaluator. No second validation expression,
// action, commit or accepted rule is introduced.
func TestD15D1OutputFlowsIntoExistingCanonicalValidation(t *testing.T) {
	definition := derivationref.Definition{
		ID:       "fixture.line_amount",
		Inputs:   []string{"fixture.order_line.quantity", "fixture.order_line.unit_price"},
		Operator: "multiply",
		Pure:     true,
	}
	if err := derivationref.ValidateGraph([]derivationref.Definition{definition}, nil); err != nil {
		t.Fatal(err)
	}
	binding := derivationref.Binding{
		RuleID: definition.ID, TargetID: "fixture.order_line.amount", Scope: "semantic",
	}
	// A synthetic instance of the *existing* positive Validation kind, not an
	// assertion that X-TEA source meaning or this Binding was accepted.
	rule := semantic.Validation{
		ID:        "fixture.order_line.amount_positive",
		Kind:      semantic.ValidationPositive,
		Attribute: core.CanonicalID(binding.TargetID),
	}
	tests := []struct {
		name    string
		qty     derivationref.Input
		price   derivationref.Input
		want    validationeval.Result
		wantErr error
	}{
		{name: "amount_600", qty: present("2"), price: present("300"), want: validationeval.ResultSatisfied},
		{name: "amount_0", qty: present("0"), price: present("300"), want: validationeval.ResultNotSatisfied},
		{name: "negative_amount", qty: present("-1"), price: present("300"), want: validationeval.ResultNotSatisfied},
		{name: "input_unavailable", qty: derivationref.Input{State: derivationref.Missing}, price: present("300"), wantErr: derivationref.ErrInput},
		{name: "declared_null", qty: derivationref.Input{State: derivationref.Null}, price: present("300"), wantErr: errFixtureDerivedNull},
		{name: "invalid_decimal", qty: present("invalid"), price: present("300"), wantErr: derivationref.ErrOperator},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			derived, derivedErr := derivationref.Multiply(tc.qty, tc.price)
			got, err := validationeval.Evaluate(rule, canonicalDerivedFixtureSource{
				target: rule.Attribute, value: derived, derivedErr: derivedErr,
			})
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error=%v want=%v", err, tc.wantErr)
				}
				if got == validationeval.ResultNotSatisfied || got == validationeval.ResultSatisfied {
					t.Fatalf("unavailable derivation became a business validation verdict: %q", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("result=%q err=%v want=%q", got, err, tc.want)
			}
		})
	}
}

func TestD15CanonicalValidationDoesNotInferSameNamedAttribute(t *testing.T) {
	derived, err := derivationref.Multiply(present("2"), present("300"))
	if err != nil {
		t.Fatal(err)
	}
	// Both have the display label "amount", but distinct typed attributes.
	const actualAttribute core.CanonicalID = "fixture.order_line.amount"
	rule := semantic.Validation{
		ID: "fixture.other_line.amount_positive", Kind: semantic.ValidationPositive,
		Attribute: "fixture.other_line.amount",
	}
	got, err := validationeval.Evaluate(rule, canonicalDerivedFixtureSource{
		target: actualAttribute, value: derived,
	})
	if err == nil || got != "" {
		t.Fatalf("different Semantic Attribute ID must not reuse derived value: result=%q err=%v", got, err)
	}
}
