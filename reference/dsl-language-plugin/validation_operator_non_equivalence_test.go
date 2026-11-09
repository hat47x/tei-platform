package languagereference

import (
	"errors"
	"math/big"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// comparisonSource returns test values only. It is not an Operator evaluator
// or a replacement for the accepted Semantic Validation data-source boundary.
type comparisonSource struct {
	number *big.Rat
	keys   []string
	err    error
}

func (s comparisonSource) Number(core.CanonicalID) (*big.Rat, error) {
	return s.number, s.err
}

func (s comparisonSource) RelationKeys(core.CanonicalID, core.CanonicalID) ([]string, error) {
	return s.keys, s.err
}

// relation-unique validates uniqueness across an authored relation scope.
// A cardinality-many expansion may legitimately preserve multiple matches.
// Treating both as one generic "unique/join" Operator would change meaning.
func TestRelationUniqueAndExpandAreNotInterchangeable(t *testing.T) {
	rule := semantic.Validation{
		ID: "sales.order.customer_unique", Kind: semantic.ValidationRelationUnique,
		Relation: "sales.order.customer", Attribute: "customer.customer.id",
	}
	got, err := validationeval.Evaluate(rule, comparisonSource{keys: []string{"id-7", "id-7"}})
	if err != nil || got != validationeval.ResultNotSatisfied {
		t.Fatalf("canonical relation-unique duplicate semantics: %q, %v", got, err)
	}

	candidate := RelationExpansionCandidate{
		SourceID: "sales.order", RelationID: "sales.order.customer",
		TargetID: "customer.customer", Cardinality: semantic.CardinalityMany,
		Keys: []semantic.RelationKey{{
			SourceAttribute: "sales.order.customer_id",
			TargetAttribute: "customer.customer.id",
		}},
		Operator: OperatorContract{
			ID: "std.op.expand", Version: "1", Fingerprint: "comparison-pin",
			Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/customer.customer",
		},
		OutputType: "rows/customer.customer",
	}
	policy := RelationMatchPolicy{Unmatched: UnmatchedReject, NullKey: NullKeyNoMatch}
	value := []ProbeKeyPart{{Kind: ProbeValue, Canonical: "id-7"}}
	matches, err := ProbePinnedExpansion(candidate, policy, value, [][]ProbeKeyPart{value, value})
	if err != nil || matches.State != MatchFound || !reflect.DeepEqual(matches.TargetIndices, []int{0, 1}) {
		t.Fatalf("cardinality-many expansion improperly deduplicated: %#v, %v", matches, err)
	}

	// The same absence is vacuously unique, but expansion needs an explicit
	// missing-target policy; there is no implicit semantic equivalence.
	unique, err := validationeval.Evaluate(rule, comparisonSource{keys: nil})
	if err != nil || unique != validationeval.ResultSatisfied {
		t.Fatalf("canonical empty-scope uniqueness: %q, %v", unique, err)
	}
	if _, err := ProbePinnedExpansion(candidate, policy, value, nil); !errors.Is(err, ErrMatchUnmatched) {
		t.Fatalf("missing-target policy disappeared: %v", err)
	}
}

func TestValidationOutcomeKeepsNotSatisfiedSeparateFromEvaluationFailure(t *testing.T) {
	rule := semantic.Validation{
		ID: "sales.order_line.quantity_positive", Kind: semantic.ValidationPositive,
		Attribute: "sales.order_line.quantity",
	}
	for _, tc := range []struct {
		name    string
		source  comparisonSource
		want    validationeval.Result
		wantErr bool
	}{
		{"positive rational", comparisonSource{number: big.NewRat(1, 3)}, validationeval.ResultSatisfied, false},
		{"zero", comparisonSource{number: big.NewRat(0, 1)}, validationeval.ResultNotSatisfied, false},
		{"negative", comparisonSource{number: big.NewRat(-2, 1)}, validationeval.ResultNotSatisfied, false},
		{"unavailable value", comparisonSource{}, "", true},
		{"failed observation", comparisonSource{err: errors.New("upstream unavailable")}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validationeval.Evaluate(rule, tc.source)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("result=%q error=%v; expected %q error=%t", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
