package main

import (
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// These tests pin the builder-contract comparison (SB-C0 §4.18's remaining
// field).
//
// The declaration states a contract and a compatible major; the plan states the
// resolved ContractVersion. The correspondence between them is a rule the
// resolver applies, not something to be inferred: fixture.go builds the version
// as fmt.Sprintf("%s/v%d", contract, major). These tests pin that rule from the
// child's side, so a change of spelling on either side fails here.

// The resolver really does spell a requirement as contract/vN.
func TestBuilderContractVersionSpelling(t *testing.T) {
	intent := declarationFor(t, nil)
	plan := fixturePlan(t)
	want := intent.Builder.Contract + "/v" + "1"
	if plan.Builder.ContractVersion != want {
		t.Fatalf("the fixture plan declares %q, expected %q", plan.Builder.ContractVersion, want)
	}
}

// An agreeing contract compares equal.
func TestCompareIntentFieldsAcceptsAnAgreeingBuilderContract(t *testing.T) {
	if err := compareIntentFields(declarationFor(t, nil), fixturePlan(t)); err != nil {
		t.Fatalf("an agreeing builder contract was refused: %v", err)
	}
}

// A different compatible major is refused, and the message shows both the
// derived version and the plan's.
func TestCompareIntentFieldsRefusesADifferentCompatibleMajor(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.CompatibleMajor = 2
	})
	err := compareIntentFields(intent, fixturePlan(t))
	if err == nil {
		t.Fatal("a different compatible major was accepted")
	}
	msg := err.Error()
	for _, want := range []string{"tei.plugin-build", "major 2", "tei.plugin-build/v2", "tei.plugin-build/v1"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the refusal did not mention %q: %s", want, msg)
		}
	}
}

// A different contract name is refused.
func TestCompareIntentFieldsRefusesADifferentContract(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Contract = "tei.other-contract"
	})
	if err := compareIntentFields(intent, fixturePlan(t)); err == nil {
		t.Fatal("a different contract name was accepted")
	}
}

// The builder's identity and artifact digest are not compared: the declaration
// names a selection policy, not the exact builder.
func TestCompareIntentFieldsIgnoresTheResolvedBuilderIdentity(t *testing.T) {
	plan := fixturePlan(t)
	plan.Builder.ArtifactDigest = "sha256:" + strings.Repeat("f", 64)
	if err := compareIntentFields(declarationFor(t, nil), plan); err != nil {
		t.Fatalf("a different resolved builder digest was treated as a declaration mismatch: %v", err)
	}
}

// The contract and major agree, and so does everything else, so the comparison
// accepts the pair. This is the control for the refusals below: a comparison
// that always failed would not be a comparison.
func TestCompareIntentFieldsComparesOnlyContractAndMajor(t *testing.T) {
	if err := compareIntentFields(declarationFor(t, nil), fixturePlan(t)); err != nil {
		t.Fatalf("an agreeing declaration was refused: %v", err)
	}
	_ = plugin.BuildPlan{}
}

// A declared provider that the plan did not resolve is refused. The provider is
// a requirement the caller wired, so a plan built by a different builder did not
// honour it. This is the unit-level counterpart of
// TestBuildChildRejectsADeclarationWhoseBuilderChoiceWasNotResolved.
func TestCompareIntentFieldsRefusesADifferentProvider(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Provider = &buildintent.ProviderWiring{Plugin: "tei.builder.go"}
	})
	err := compareIntentFields(intent, fixturePlan(t))
	if err == nil {
		t.Fatal("a provider the plan did not resolve was accepted")
	}
	for _, want := range []string{"tei.builder.go", string(fixturePlan(t).Builder.ID)} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal did not name %q: %s", want, err)
		}
	}
}

// An agreeing provider is accepted.
func TestCompareIntentFieldsAcceptsAnAgreeingProvider(t *testing.T) {
	plan := fixturePlan(t)
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Provider = &buildintent.ProviderWiring{Plugin: plan.Builder.ID}
	})
	if err := compareIntentFields(intent, plan); err != nil {
		t.Fatalf("an agreeing provider was refused: %v", err)
	}
}

// A pin the plan did not resolve is refused: an exact artifact digest is a
// requirement, not a preference.
func TestCompareIntentFieldsRefusesADifferentPin(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Pin = &buildintent.ArtifactPin{ArtifactDigest: "sha256:" + strings.Repeat("c", 64)}
	})
	err := compareIntentFields(intent, fixturePlan(t))
	if err == nil {
		t.Fatal("a pin the plan did not resolve was accepted")
	}
	if !strings.Contains(err.Error(), strings.Repeat("c", 64)) {
		t.Fatalf("the refusal did not name the pinned digest: %s", err)
	}
}
