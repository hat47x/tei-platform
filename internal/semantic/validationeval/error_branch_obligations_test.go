package validationeval_test

import (
	"errors"
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// pathProbeSource is an independent test double for Source outcomes. It does
// not imply that a real database or application Driver was exercised.
type pathProbeSource struct {
	number      *big.Rat
	numberErr   error
	keys        []string
	keysErr     error
	numberCalls int
	keyCalls    int
}

func (s *pathProbeSource) Number(core.CanonicalID) (*big.Rat, error) {
	s.numberCalls++
	return s.number, s.numberErr
}

func (s *pathProbeSource) RelationKeys(core.CanonicalID, core.CanonicalID) ([]string, error) {
	s.keyCalls++
	return s.keys, s.keysErr
}

// These cases establish a bounded set of evaluator decision outcomes. They do
// not measure Go C1: statement coverage and branch-outcome coverage are distinct.
func TestEvaluationErrorsAreDistinctFromBusinessRejections(t *testing.T) {
	positive := semantic.Validation{
		ID: "test.positive", Kind: semantic.ValidationPositive, Attribute: "test.quantity",
	}
	unique := semantic.Validation{
		ID: "test.unique", Kind: semantic.ValidationRelationUnique,
		Relation: "test.lines", Attribute: "test.line_number",
	}
	numberFailure := errors.New("number source failed")
	relationFailure := errors.New("relation source failed")
	cases := []struct {
		name        string
		rule        semantic.Validation
		source      *pathProbeSource
		want        validationeval.Result
		wantError   bool
		wrapped     error
		numberCalls int
		keyCalls    int
	}{
		{name: "source missing", rule: positive, wantError: true},
		{name: "number source error", rule: positive, source: &pathProbeSource{numberErr: numberFailure}, wantError: true, wrapped: numberFailure, numberCalls: 1},
		{name: "number value unavailable without source error", rule: positive, source: &pathProbeSource{}, wantError: true, numberCalls: 1},
		{name: "zero is business rejection", rule: positive, source: &pathProbeSource{number: big.NewRat(0, 1)}, want: validationeval.ResultNotSatisfied, numberCalls: 1},
		{name: "positive is satisfied", rule: positive, source: &pathProbeSource{number: big.NewRat(1, 2)}, want: validationeval.ResultSatisfied, numberCalls: 1},
		{name: "relation source error", rule: unique, source: &pathProbeSource{keysErr: relationFailure}, wantError: true, wrapped: relationFailure, keyCalls: 1},
		{name: "duplicate keys are business rejection", rule: unique, source: &pathProbeSource{keys: []string{"1", "2", "2"}}, want: validationeval.ResultNotSatisfied, keyCalls: 1},
		{name: "unique keys are satisfied", rule: unique, source: &pathProbeSource{keys: []string{"1", "2"}}, want: validationeval.ResultSatisfied, keyCalls: 1},
		{name: "empty keys are satisfied in a fully supplied slice", rule: unique, source: &pathProbeSource{keys: []string{}}, want: validationeval.ResultSatisfied, keyCalls: 1},
		{name: "unsupported kind does not call source", rule: semantic.Validation{ID: "test.unsupported", Kind: semantic.ValidationKind("unsupported")}, source: &pathProbeSource{}, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var source validationeval.Source
			if tc.source != nil {
				source = tc.source
			}
			got, err := validationeval.Evaluate(tc.rule, source)
			if tc.wantError {
				if err == nil || got != "" {
					t.Fatalf("expected technical error and empty business result, got result=%q err=%v", got, err)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("expected business result %q without technical error, got result=%q err=%v", tc.want, got, err)
			}
			if tc.wrapped != nil && !errors.Is(err, tc.wrapped) {
				t.Fatalf("expected source error to remain inspectable: got %v", err)
			}
			if tc.source != nil && (tc.source.numberCalls != tc.numberCalls || tc.source.keyCalls != tc.keyCalls) {
				t.Fatalf("wrong Source invocation path: number=%d keys=%d; want number=%d keys=%d", tc.source.numberCalls, tc.source.keyCalls, tc.numberCalls, tc.keyCalls)
			}
		})
	}
}
