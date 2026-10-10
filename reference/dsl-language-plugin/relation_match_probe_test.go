package languagereference

import (
	"errors"
	"reflect"
	"testing"

	"tei/internal/semantic"
)

func probeCandidate(cardinality semantic.Cardinality, width int) RelationExpansionCandidate {
	c := RelationExpansionCandidate{
		SourceID: "sales.order", RelationID: "sales.order.customer", TargetID: "customer.customer",
		Cardinality: cardinality,
		Operator: OperatorContract{
			ID: "std.op.expand", Version: "1", Fingerprint: "expand-v1",
			Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/customer.customer",
		},
		OutputType: "rows/customer.customer",
	}
	for i := 0; i < width; i++ {
		c.Keys = append(c.Keys, semantic.RelationKey{})
	}
	return c
}

func probeValue(s string) ProbeKeyPart { return ProbeKeyPart{Kind: ProbeValue, Canonical: s} }
func probeNull() ProbeKeyPart          { return ProbeKeyPart{Kind: ProbeNull} }
func probeUnavailable() ProbeKeyPart   { return ProbeKeyPart{Kind: ProbeUnavailable} }

func TestProbePinnedExpansionPreservesMultiplicityAndCompositeKeys(t *testing.T) {
	candidate := probeCandidate(semantic.CardinalityMany, 2)
	policy := RelationMatchPolicy{Unmatched: UnmatchedReject, NullKey: NullKeyNoMatch}
	result, err := ProbePinnedExpansion(candidate, policy, []ProbeKeyPart{probeValue("a"), probeValue("b")}, [][]ProbeKeyPart{
		{probeValue("a"), probeValue("c")},
		{probeValue("a"), probeValue("b")},
		{probeValue("a"), probeValue("b")},
		{probeValue("a"), probeValue("B")},
	})
	if err != nil || result.State != MatchFound || !reflect.DeepEqual(result.TargetIndices, []int{1, 2}) {
		t.Fatalf("lost duplicate matches or joined an incorrect composite key: %#v, %v", result, err)
	}
}

func TestProbePinnedExpansionRejectsOneToManyViolation(t *testing.T) {
	candidate := probeCandidate(semantic.CardinalityOne, 1)
	result, err := ProbePinnedExpansion(candidate,
		RelationMatchPolicy{Unmatched: UnmatchedDiscard, NullKey: NullKeyNoMatch},
		[]ProbeKeyPart{probeValue("key")}, [][]ProbeKeyPart{{probeValue("key")}, {probeValue("key")}})
	if !errors.Is(err, ErrMatchCardinality) || result.State != "" {
		t.Fatalf("one-to-many mismatch silently picked one: %#v, %v", result, err)
	}
}

func TestProbePinnedExpansionNoMatchDecisions(t *testing.T) {
	candidate := probeCandidate(semantic.CardinalityMany, 1)
	for _, tc := range []struct {
		policy	UnmatchedPolicy
		state	RelationMatchState
		fail	error
	}{
		{UnmatchedReject, "", ErrMatchUnmatched},
		{UnmatchedDiscard, MatchDiscarded, nil},
		{UnmatchedRetainAbsent, MatchRetainedAbsent, nil},
	} {
		t.Run(string(tc.policy), func(t *testing.T) {
			result, err := ProbePinnedExpansion(candidate,
				RelationMatchPolicy{Unmatched: tc.policy, NullKey: NullKeyNoMatch},
				[]ProbeKeyPart{probeValue("missing-target")}, [][]ProbeKeyPart{{probeValue("other")}})
			if !errors.Is(err, tc.fail) || result.State != tc.state || len(result.TargetIndices) != 0 {
				t.Fatalf("wrong missing target semantics: %#v, %v", result, err)
			}
		})
	}
}

func TestProbePinnedExpansionNullIsNotUnavailable(t *testing.T) {
	candidate := probeCandidate(semantic.CardinalityMany, 1)
	for _, tc := range []struct {
		name	string
		source	[]ProbeKeyPart
		targets	[][]ProbeKeyPart
		null	NullKeyPolicy
		want	RelationMatchState
		wantErr	error
	}{
		{"null source no-match", []ProbeKeyPart{probeNull()}, [][]ProbeKeyPart{{probeValue("x")}}, NullKeyNoMatch, MatchRetainedAbsent, nil},
		{"null target no-match", []ProbeKeyPart{probeValue("x")}, [][]ProbeKeyPart{{probeNull()}}, NullKeyNoMatch, MatchRetainedAbsent, nil},
		{"null never joins null", []ProbeKeyPart{probeNull()}, [][]ProbeKeyPart{{probeNull()}}, NullKeyNoMatch, MatchRetainedAbsent, nil},
		{"null source rejected", []ProbeKeyPart{probeNull()}, nil, NullKeyReject, "", ErrMatchNullRejected},
		{"null target rejected", []ProbeKeyPart{probeValue("x")}, [][]ProbeKeyPart{{probeNull()}}, NullKeyReject, "", ErrMatchNullRejected},
		{"unavailable source", []ProbeKeyPart{probeUnavailable()}, nil, NullKeyNoMatch, "", ErrMatchKeyUnavailable},
		{"unavailable target", []ProbeKeyPart{probeValue("x")}, [][]ProbeKeyPart{{probeUnavailable()}}, NullKeyNoMatch, "", ErrMatchKeyUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ProbePinnedExpansion(candidate,
				RelationMatchPolicy{Unmatched: UnmatchedRetainAbsent, NullKey: tc.null}, tc.source, tc.targets)
			if !errors.Is(err, tc.wantErr) || result.State != tc.want {
				t.Fatalf("NULL/unavailable were conflated: %#v, %v", result, err)
			}
		})
	}
}

func TestProbePinnedExpansionRejectsPartialContracts(t *testing.T) {
	candidate := probeCandidate(semantic.CardinalityMany, 1)
	for _, policy := range []RelationMatchPolicy{
		{}, {Unmatched: UnmatchedReject}, {NullKey: NullKeyNoMatch},
		{Unmatched: "first-only", NullKey: NullKeyNoMatch},
		{Unmatched: UnmatchedDiscard, NullKey: "match-null"},
	} {
		if _, err := ProbePinnedExpansion(candidate, policy, []ProbeKeyPart{probeValue("a")}, nil); !errors.Is(err, ErrMatchPolicyInvalid) {
			t.Fatalf("implicit/unsupported policy %#v was accepted: %v", policy, err)
		}
	}
	policy := RelationMatchPolicy{Unmatched: UnmatchedDiscard, NullKey: NullKeyNoMatch}
	for _, source := range [][]ProbeKeyPart{
		nil, {{Kind: "unknown"}}, {{Kind: ProbeNull, Canonical: "value"}}, {{Kind: ProbeUnavailable}},
	} {
		if _, err := ProbePinnedExpansion(candidate, policy, source, nil); !errors.Is(err, ErrMatchKeyUnavailable) {
			t.Fatalf("invalid fixture key %#v was accepted: %v", source, err)
		}
	}
}

func TestProbePinnedExpansionEmptyCanonicalValueIsNotNull(t *testing.T) {
	candidate := probeCandidate(semantic.CardinalityOne, 1)
	policy := RelationMatchPolicy{Unmatched: UnmatchedReject, NullKey: NullKeyReject}
	result, err := ProbePinnedExpansion(candidate, policy, []ProbeKeyPart{probeValue("")}, [][]ProbeKeyPart{{probeValue("")}})
	if err != nil || result.State != MatchFound || !reflect.DeepEqual(result.TargetIndices, []int{0}) {
		t.Fatalf("empty value incorrectly handled as null: %#v, %v", result, err)
	}
}

// A valid Canonical World must supply the Relation and the exact operator pin.
// The probe evaluates fixtures only after the existing validator is consulted.
func TestProbePinnedExpansionUsesCanonicalWorldCandidate(t *testing.T) {
	w := crossNamespaceRelationFixture()
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "cross-namespace-v1",
		Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/customer.customer",
	}
	candidate, err := InspectPinnedRelationExpansion(w, "sales.order", "sales.order.customer",
		PinnedCall{Contract: pin, InputType: pin.InputType}, []OperatorContract{pin})
	if err != nil {
		t.Fatalf("cannot inspect canonical relation: %v", err)
	}
	result, err := ProbePinnedExpansion(candidate,
		RelationMatchPolicy{Unmatched: UnmatchedReject, NullKey: NullKeyReject},
		[]ProbeKeyPart{probeValue("c-42")}, [][]ProbeKeyPart{{probeValue("c-42")}})
	if err != nil || result.State != MatchFound || !reflect.DeepEqual(result.TargetIndices, []int{0}) {
		t.Fatalf("validated candidate did not match fixture: %#v, %v", result, err)
	}
}

func TestProbePinnedExpansionWithParametricOperatorPin(t *testing.T) {
	w := crossNamespaceRelationFixture()
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "expand-parametric-v1",
		Effect: "pure", InputType: expandSourceTypeVariable, OutputType: expandTargetTypeVariable,
	}
	candidate, err := InspectPinnedRelationExpansion(w, "sales.order", "sales.order.customer",
		PinnedCall{Contract: pin, InputType: "rows/sales.order"}, []OperatorContract{pin})
	if err != nil || candidate.OutputType != "rows/customer.customer" || candidate.Operator != pin {
		t.Fatalf("cannot instantiate accepted template at canonical endpoints: %#v, %v", candidate, err)
	}
	result, err := ProbePinnedExpansion(candidate,
		RelationMatchPolicy{Unmatched: UnmatchedReject, NullKey: NullKeyNoMatch},
		[]ProbeKeyPart{probeValue("customer-1")}, [][]ProbeKeyPart{{probeValue("customer-1")}})
	if err != nil || result.State != MatchFound || !reflect.DeepEqual(result.TargetIndices, []int{0}) {
		t.Fatalf("parametric pinned candidate did not match: %#v, %v", result, err)
	}
}
