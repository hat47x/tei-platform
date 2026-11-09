package main

import (
	"strings"
	"testing"
)

// This reference test establishes only the distinction between a published
// logical Contract and an executable Provider. Conformance evidence below is
// deliberately synthetic: it does not certify a real SQLite driver, issuer,
// artifact, or production Host acceptance.
func TestContractOnlyCanExistBeforeAnyProviderImplementation(t *testing.T) {
	contract := Contract{
		ID:                 "reference.database.sqlite.read-binding",
		Version:            "1.0.0",
		RequestSemantics:   "read-only parameterized query with explicit identifiers",
		ResultSemantics:    "typed rows with nulls preserved",
		FailureSemantics:   "unsupported and execution failure are not success",
		ConformanceProfile: "reference.sqlite.read-binding/v0",
	}
	requirement := Requirement{ContractID: contract.ID, CompatibleMajor: 1}
	matching, err := compatibleContracts([]Contract{contract}, requirement)
	if err != nil || len(matching) != 1 || fingerprint(matching[0]) != fingerprint(contract) {
		t.Fatalf("contract-only publication must remain independently inspectable: contracts=%v err=%v", matching, err)
	}

	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: requirement,
		Contracts:   []Contract{contract},
	}
	if _, err := resolveCapability(input); err == nil ||
		!strings.Contains(err.Error(), "no provider implementation") {
		t.Fatalf("contract-only request should report missing realization, got %v", err)
	}

	// Providing self-declared code metadata does not imply conformance.
	candidate := sourceAdapterProvider(
		contract, ScopeApplicationRuntime,
		"reference.synthetic.sqlite", "read-binding-v0",
	)
	input.Providers = []Provider{candidate}
	if _, err := resolveCapability(input); err == nil ||
		!strings.Contains(err.Error(), "no conformant provider") {
		t.Fatalf("unqualified candidate was selected: %v", err)
	}

	// This is a synthetic resolver fixture, NOT independent conformance,
	// Host trust, actual DB execution, or authorization to invoke the code.
	input.Evidence = []ConformanceEvidence{
		evidenceFor(candidate, contract, "synthetic-resolver-fixture-only"),
	}
	resolved, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ContractFingerprint != fingerprint(contract) ||
		resolved.Provider != candidate ||
		resolved.Contract.ID != contract.ID {
		t.Fatalf("qualified synthetic candidate lost exact logical binding: %+v", resolved)
	}
}

func TestContractOnlyExplicitWiringStillDoesNotAuthorizeMissingProvider(t *testing.T) {
	contract := Contract{
		ID:                 "reference.database.sqlite.read-binding",
		Version:            "1.0.0",
		RequestSemantics:   "read-only query",
		ResultSemantics:    "typed rows",
		FailureSemantics:   "reject unsupported",
		ConformanceProfile: "reference.sqlite.read-binding/v0",
	}
	input := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Wirings: []Wiring{{
			ContractID:       contract.ID,
			PluginID:         "reference.synthetic.sqlite",
			PluginVersion:    "1.0.0",
			ImplementationID: "read-binding-v0",
		}},
	}
	if _, err := resolveCapability(input); err == nil ||
		!strings.Contains(err.Error(), "wired provider") {
		t.Fatalf("explicit wiring without provider must fail closed, got %v", err)
	}
}
