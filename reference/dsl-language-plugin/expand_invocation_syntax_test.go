package languagereference

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/semantic"
)

func TestParseExpandInvocationKeepsSourceBytesAndNamedArguments(t *testing.T) {
	source := " \n  std.op.expand(  null_key: no-match, relation: sales.order.customer, unmatched: retain-absence ) \t"
	syntax, err := ParseExpandInvocation(source)
	if err != nil {
		t.Fatal(err)
	}
	if syntax.RelationID != "sales.order.customer" ||
		syntax.Policy.Unmatched != UnmatchedRetainAbsent || syntax.Policy.NullKey != NullKeyNoMatch {
		t.Fatalf("wrong parsed arguments: %#v", syntax)
	}
	relationStart := strings.Index(source, "sales.order.customer")
	if syntax.RelationSpan.Start != relationStart ||
		syntax.RelationSpan.End != relationStart+len("sales.order.customer") {
		t.Fatalf("lost relation byte location: %#v", syntax.RelationSpan)
	}
	nullStart := strings.Index(source, "no-match")
	if syntax.PolicySpans["null_key"] != (PipelineSpan{Start: nullStart, End: nullStart + len("no-match")}) {
		t.Fatalf("lost policy byte location: %#v", syntax.PolicySpans)
	}
	if syntax.CallSpan.Start != strings.Index(source, "std.op.expand") {
		t.Fatalf("wrong invocation start: %#v", syntax.CallSpan)
	}
}

func TestParseExpandInvocationRejectsAmbiguityAndIncompletePolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		cause  error
	}{
		{"missing unmatched", "std.op.expand(relation: sales.order.customer, null_key: reject)", nil},
		{"missing null policy", "std.op.expand(relation: sales.order.customer, unmatched: reject)", nil},
		{"duplicate argument", "std.op.expand(relation: x.y, relation: x.z, unmatched: reject, null_key: reject)", nil},
		{"unknown argument", "std.op.expand(relation: x.y, unmatched: reject, null_key: reject, distinct: yes)", nil},
		{"invalid null policy", "std.op.expand(relation: x.y, unmatched: reject, null_key: equals)", ErrMatchPolicyInvalid},
		{"invalid unmatched policy", "std.op.expand(relation: x.y, unmatched: first, null_key: reject)", ErrMatchPolicyInvalid},
		{"blank value", "std.op.expand(relation: x.y, unmatched: , null_key: reject)", nil},
		{"unnamed", "std.op.expand(relation: x.y, reject, null_key: reject)", nil},
		{"trailing comma", "std.op.expand(relation: x.y, unmatched: reject, null_key: reject,)", nil},
		{"unclosed", "std.op.expand(relation: x.y, unmatched: reject, null_key: reject", nil},
		{"wrong operator", "std.op.join(relation: x.y, unmatched: reject, null_key: reject)", nil},
		{"empty", " \n ", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseExpandInvocation(tc.source)
			if err == nil {
				t.Fatal("invalid syntax was accepted")
			}
			var diag *PipelineDiagnostic
			if !errors.As(err, &diag) ||
				diag.Span.Start < 0 || diag.Span.End > len(tc.source) || diag.Span.End < diag.Span.Start {
				t.Fatalf("invalid or unlocated diagnostic: %v", err)
			}
			if tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("cause lost: %v", err)
			}
		})
	}
}

func TestLowerPinnedExpandMeaningIsInsensitiveToWhitespaceAndArgumentOrder(t *testing.T) {
	w := crossNamespaceRelationFixture()
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "generic-v1",
		Effect: "pure", InputType: expandSourceTypeVariable, OutputType: expandTargetTypeVariable,
	}
	call := PinnedCall{Contract: pin, InputType: "rows/sales.order"}
	var first BoundExpandMeaning
	for i, src := range []string{
		"std.op.expand(relation: sales.order.customer, unmatched: reject, null_key: no-match)",
		"  std.op.expand( null_key: no-match , unmatched: reject , relation: sales.order.customer )",
	} {
		syntax, err := ParseExpandInvocation(src)
		if err != nil {
			t.Fatal(err)
		}
		meaning, err := LowerPinnedExpandMeaning(w, "sales.order", syntax, call, []OperatorContract{pin})
		if err != nil {
			t.Fatal(err)
		}
		if len(meaning.Fingerprint) != 64 || VerifyBoundExpandMeaning(meaning) != nil {
			t.Fatalf("invalid meaning fingerprint: %#v", meaning)
		}
		if i == 0 {
			first = meaning
		} else {
			if meaning.Fingerprint != first.Fingerprint {
				t.Fatalf("same meaning changed with syntax: %s vs %s", meaning.Fingerprint, first.Fingerprint)
			}
			if meaning.SourceSpan == first.SourceSpan {
				t.Fatal("syntax provenance should retain original location")
			}
		}
	}
	result, err := ProbeBoundExpandMeaning(first, []ProbeKeyPart{probeValue("key")}, [][]ProbeKeyPart{{probeValue("key")}})
	if err != nil || result.State != MatchFound {
		t.Fatalf("bound oracle failed: %#v, %v", result, err)
	}
}

func TestLowerPinnedExpandMeaningSeparatesPolicyFromOperatorIdentity(t *testing.T) {
	w := crossNamespaceRelationFixture()
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "generic-v1",
		Effect: "pure", InputType: expandSourceTypeVariable, OutputType: expandTargetTypeVariable,
	}
	call := PinnedCall{Contract: pin, InputType: "rows/sales.order"}
	var fingerprints []string
	for _, unmatched := range []UnmatchedPolicy{UnmatchedReject, UnmatchedDiscard, UnmatchedRetainAbsent} {
		syntax, err := ParseExpandInvocation(
			"std.op.expand(relation: sales.order.customer, unmatched: " + string(unmatched) + ", null_key: no-match)")
		if err != nil {
			t.Fatal(err)
		}
		meaning, err := LowerPinnedExpandMeaning(w, "sales.order", syntax, call, []OperatorContract{pin})
		if err != nil {
			t.Fatal(err)
		}
		fingerprints = append(fingerprints, meaning.Fingerprint)
		res, probeErr := ProbeBoundExpandMeaning(meaning,
			[]ProbeKeyPart{probeValue("absent")}, [][]ProbeKeyPart{{probeValue("other")}})
		switch unmatched {
		case UnmatchedReject:
			if !errors.Is(probeErr, ErrMatchUnmatched) {
				t.Fatalf("reject meaning did not reject: %v", probeErr)
			}
		case UnmatchedDiscard:
			if probeErr != nil || res.State != MatchDiscarded {
				t.Fatalf("discard meaning changed: %#v, %v", res, probeErr)
			}
		case UnmatchedRetainAbsent:
			if probeErr != nil || res.State != MatchRetainedAbsent {
				t.Fatalf("retain meaning changed: %#v, %v", res, probeErr)
			}
		}
	}
	if fingerprints[0] == fingerprints[1] || fingerprints[1] == fingerprints[2] ||
		fingerprints[0] == fingerprints[2] {
		t.Fatalf("different unmatched policies share a fingerprint: %#v", fingerprints)
	}
}

func TestBoundExpandMeaningRejectsMutationOrIncorrectPins(t *testing.T) {
	w := crossNamespaceRelationFixture()
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "generic-v1",
		Effect: "pure", InputType: expandSourceTypeVariable, OutputType: expandTargetTypeVariable,
	}
	call := PinnedCall{Contract: pin, InputType: "rows/sales.order"}
	syntax, err := ParseExpandInvocation(
		"std.op.expand(relation: sales.order.customer, unmatched: reject, null_key: reject)")
	if err != nil {
		t.Fatal(err)
	}
	meaning, err := LowerPinnedExpandMeaning(w, "sales.order", syntax, call, []OperatorContract{pin})
	if err != nil {
		t.Fatal(err)
	}

	changedPolicy := meaning
	changedPolicy.Policy.Unmatched = UnmatchedRetainAbsent
	if err := VerifyBoundExpandMeaning(changedPolicy); !errors.Is(err, ErrExpandMeaningDrift) {
		t.Fatalf("policy drift accepted: %v", err)
	}
	changedRelation := meaning
	changedRelation.Expansion.RelationID = "sales.order.other"
	if err := VerifyBoundExpandMeaning(changedRelation); !errors.Is(err, ErrExpandMeaningDrift) {
		t.Fatalf("relation drift accepted: %v", err)
	}
	changedKey := meaning
	changedKey.Expansion.Keys = append([]semantic.RelationKey(nil), meaning.Expansion.Keys...)
	changedKey.Expansion.Keys[0].TargetAttribute = "customer.customer.other_id"
	if err := VerifyBoundExpandMeaning(changedKey); !errors.Is(err, ErrExpandMeaningDrift) {
		t.Fatalf("key drift accepted: %v", err)
	}
	if _, err := ProbeBoundExpandMeaning(changedPolicy, []ProbeKeyPart{probeValue("a")}, nil); !errors.Is(err, ErrExpandMeaningDrift) {
		t.Fatalf("oracle accepted stale bound policy: %v", err)
	}
	// A display-source move does not change the authored semantic contract.
	changedSpan := meaning
	changedSpan.SourceSpan.Start += 5
	if err := VerifyBoundExpandMeaning(changedSpan); err != nil {
		t.Fatalf("source location became part of meaning: %v", err)
	}
	w.Semantics[0].Data[0].Relations[0].Keys[0].TargetAttribute = "customer.customer.modified"
	if err := VerifyBoundExpandMeaning(meaning); err != nil {
		t.Fatalf("candidate should retain detached keys: %v", err)
	}

	w = crossNamespaceRelationFixture()
	bad := pin
	bad.Fingerprint = "unknown"
	if _, err := LowerPinnedExpandMeaning(w, "sales.order", syntax,
		PinnedCall{Contract: bad, InputType: call.InputType}, []OperatorContract{pin}); !errors.Is(err, ErrOperatorMissing) {
		t.Fatalf("unaccepted pin was selected: %v", err)
	}
	missingSyntax := syntax
	missingSyntax.RelationID = "sales.order.unknown"
	missingSyntax.RelationSpan = PipelineSpan{Start: 14, End: 33}
	if _, err := LowerPinnedExpandMeaning(w, "sales.order", missingSyntax, call, []OperatorContract{pin}); err == nil {
		t.Fatal("unresolved relation was accepted")
	} else {
		var diagnostic *PipelineDiagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Span != missingSyntax.RelationSpan {
			t.Fatalf("lost relation source span: %v", err)
		}
	}
}
