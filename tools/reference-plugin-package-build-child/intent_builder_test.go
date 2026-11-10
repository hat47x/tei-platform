package main

import (
	"strings"
	"testing"

	"tei/internal/buildintent"
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

// A wired provider is a requirement the child does check (#3670): if the plan
// resolved a different builder, the declaration's choice was not honoured.
// This used to assert the opposite, which contradicted main_test.go's
// end-to-end provider case and left the package failing (#3697 follow-up).
func TestCompareIntentFieldsRefusesADifferentWiredProvider(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Provider = &buildintent.ProviderWiring{Plugin: "tei.builder.other.seed"}
	})
	err := compareIntentFields(intent, fixturePlan(t))
	if err == nil {
		t.Fatal("a provider the plan did not resolve was accepted")
	}
	if !strings.Contains(err.Error(), "tei.builder.other.seed") {
		t.Fatalf("the refusal does not name the wired provider: %v", err)
	}
}

// An agreeing provider is not a mismatch: a comparison that always fails is not
// a comparison.
func TestCompareIntentFieldsAcceptsTheResolvedWiredProvider(t *testing.T) {
	plan := fixturePlan(t)
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Provider = &buildintent.ProviderWiring{Plugin: plan.Builder.ID}
	})
	if err := compareIntentFields(intent, plan); err != nil {
		t.Fatalf("the resolved provider was refused: %v", err)
	}
}

// provider.implementation has no counterpart in plugin.BuilderIdentity, so it is
// not compared. Naming the gap keeps a later reader from assuming it is checked.
func TestCompareIntentFieldsIgnoresTheProviderImplementation(t *testing.T) {
	plan := fixturePlan(t)
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Builder.Provider = &buildintent.ProviderWiring{Plugin: plan.Builder.ID, Implementation: "some.other.implementation"}
	})
	if err := compareIntentFields(intent, plan); err != nil {
		t.Fatalf("an unmatched provider.implementation was treated as a mismatch: %v", err)
	}
}
