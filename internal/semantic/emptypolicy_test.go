package semantic

import "testing"

func TestEmptyPolicyValidity(t *testing.T) {
	for _, policy := range []EmptyPolicy{EmptyPolicyAllow, EmptyPolicyRequiresEvidence, EmptyPolicyDeny} {
		if !policy.Valid() {
			t.Fatalf("%q should be valid", policy)
		}
	}
	for _, policy := range []EmptyPolicy{"", "reject", "ALLOW"} {
		if policy.Valid() {
			t.Fatalf("%q should be invalid", policy)
		}
	}
}

func TestDefaultEmptyPolicyByCondition(t *testing.T) {
	cases := map[EmptyPolicyCondition]EmptyPolicy{
		EmptyPolicyUniversal: EmptyPolicyRequiresEvidence,
		EmptyPolicyAbsence:   EmptyPolicyAllow,
		EmptyPolicyExistence: EmptyPolicyDeny,
		EmptyPolicyAggregate: EmptyPolicyAllow,
		EmptyPolicyHandoff:   EmptyPolicyRequiresEvidence,
	}
	for kind, want := range cases {
		if got := DefaultEmptyPolicy(kind); got != want {
			t.Fatalf("kind %q default=%q want %q", kind, got, want)
		}
	}
	if got := DefaultEmptyPolicy("unknown"); got != "" {
		t.Fatalf("unknown kind default=%q, want no default", got)
	}
}

func TestEffectiveEmptyPolicyUsesTheActionDeclaration(t *testing.T) {
	capability := Capability{}
	if got := capability.EffectiveEmptyPolicy(EmptyPolicyUniversal); got != EmptyPolicyRequiresEvidence {
		t.Fatalf("universal default=%q", got)
	}
	capability.EmptyPolicy = EmptyPolicyAllow
	if got := capability.EffectiveEmptyPolicy(EmptyPolicyUniversal); got != EmptyPolicyAllow {
		t.Fatalf("declared policy did not override the default: %q", got)
	}
	if got := capability.EffectiveEmptyPolicy(EmptyPolicyExistence); got != EmptyPolicyAllow {
		t.Fatalf("declared policy did not apply to another condition kind: %q", got)
	}
}
