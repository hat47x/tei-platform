package main

import (
	"reflect"
	"strings"
	"testing"
)

// No real authentication implementation is installed or invoked. These inputs
// isolate cross-phase metadata checks from actual behavior and publisher trust.
func contributionFixture() ([]ContributionRequirement, []ContributionOffer, []Contract, []ConformanceEvidence) {
	meaning := MeaningPin{ID: "reference.identity.session", Version: "2.0.0", Fingerprint: "synthetic-session-meaning-v2"}
	var requirements []ContributionRequirement
	var offers []ContributionOffer
	var contracts []Contract
	var evidence []ConformanceEvidence
	for i, phase := range []struct {
		slot   string
		scope  AuthorityScope
		target string
	}{
		{"studio", ScopeAuthoringTool, "browser"},
		{"generator", ScopeBuildTool, "linux-amd64"},
		{"runtime", ScopeApplicationRuntime, "browser-wasm"},
		{"test", ScopeTargetLocal, "linux-amd64"},
	} {
		contract := Contract{
			ID: "reference.session." + phase.slot, Version: "1.0.0",
			RequestSemantics: phase.slot + " input", ResultSemantics: phase.slot + " output",
			FailureSemantics: "unsupported is an error", ConformanceProfile: "reference-only/v0",
		}
		provider := Provider{
			PluginID: "reference.session." + phase.slot, PluginVersion: "1.0.0",
			ImplementationID: phase.slot + "-implementation", ArtifactDigest: "synthetic-" + phase.slot,
			Scope: phase.scope, ContractID: contract.ID, ContractVersion: contract.Version,
			ContractFingerprint: fingerprint(contract), ConformanceProfile: contract.ConformanceProfile,
		}
		// A runtime provider may have an independent package release cadence.
		if i == 2 {
			provider.PluginVersion = "4.2.0"
		}
		contracts = append(contracts, contract)
		offers = append(offers, ContributionOffer{Meaning: meaning, Target: phase.target, Provider: provider})
		evidence = append(evidence, evidenceFor(provider, contract, "synthetic-"+phase.slot))
		requirements = append(requirements, ContributionRequirement{
			Slot: phase.slot, Meaning: meaning, Target: phase.target, Scope: phase.scope,
			ContractFingerprint: fingerprint(contract),
			Requirement:         Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		})
	}
	return requirements, offers, contracts, evidence
}

func TestContributionPlanPreservesMeaningAcrossDifferentPhaseContracts(t *testing.T) {
	requirements, offers, contracts, evidence := contributionFixture()
	bindings, err := planContributions(requirements, offers, contracts, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 4 {
		t.Fatalf("expected all four phases, got %+v", bindings)
	}
	wantSlots := []string{"generator", "runtime", "studio", "test"}
	for i, binding := range bindings {
		if binding.Slot != wantSlots[i] || binding.Meaning != requirements[0].Meaning {
			t.Fatalf("shared meaning or deterministic slot order lost: %+v", bindings)
		}
		if binding.Resolution.Provider.ImplementationID != binding.Slot+"-implementation" {
			t.Fatalf("phase-specific implementation lost: %+v", binding)
		}
	}
	if bindings[1].Resolution.Provider.PluginVersion != "4.2.0" {
		t.Fatalf("independent package version was not preserved: %+v", bindings)
	}
	for i, j := 0, len(offers)-1; i < j; i, j = i+1, j-1 {
		offers[i], offers[j] = offers[j], offers[i]
	}
	for i, j := 0, len(requirements)-1; i < j; i, j = i+1, j-1 {
		requirements[i], requirements[j] = requirements[j], requirements[i]
	}
	reordered, err := planContributions(requirements, offers, contracts, evidence)
	if err != nil || !reflect.DeepEqual(bindings, reordered) {
		t.Fatalf("catalog/request ordering changed plan: %v", err)
	}
}

func TestContributionPlanDoesNotInventUnrequestedPhases(t *testing.T) {
	r, o, c, e := contributionFixture()
	bindings, err := planContributions(r[:1], o, c, e)
	if err != nil || len(bindings) != 1 || bindings[0].Slot != "studio" {
		t.Fatalf("an editor-only selection should not require runtime: %+v %v", bindings, err)
	}
	bindings, err = planContributions(nil, nil, nil, nil)
	if err != nil || len(bindings) != 0 {
		t.Fatalf("a data-only package need not add executable phases: %+v %v", bindings, err)
	}
}

func TestContributionPlanRejectsIncompleteCrossPhaseComposition(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func([]ContributionRequirement, []ContributionOffer, []Contract, *[]ConformanceEvidence)
		message string
	}{
		{"old runtime meaning", func(_ []ContributionRequirement, o []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			o[2].Meaning.Version = "1.0.0"
		}, "slot runtime"},
		{"same shape different meaning", func(_ []ContributionRequirement, o []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			o[2].Meaning.ID = "reference.unrelated"
		}, "slot runtime"},
		{"same meaning version changed meaning", func(_ []ContributionRequirement, o []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			o[2].Meaning.Fingerprint = "other-semantics"
		}, "slot runtime"},
		{"wrong target", func(_ []ContributionRequirement, o []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			o[2].Target = "windows-amd64"
		}, "slot runtime"},
		{"wrong scope", func(_ []ContributionRequirement, o []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			o[2].Provider.Scope = ScopeBuildTool
		}, "slot runtime"},
		{"changed phase contract", func(_ []ContributionRequirement, _ []ContributionOffer, c []Contract, _ *[]ConformanceEvidence) {
			c[2].ResultSemantics = "silently changed result"
		}, "slot runtime"},
		{"missing conformance", func(_ []ContributionRequirement, _ []ContributionOffer, _ []Contract, e *[]ConformanceEvidence) {
			*e = append((*e)[:2], (*e)[3:]...)
		}, "slot runtime"},
		{"wrong explicit provider", func(r []ContributionRequirement, _ []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			r[2].Wirings = []Wiring{{ContractID: r[2].Requirement.ContractID, PluginID: "wrong", PluginVersion: "1.0.0", ImplementationID: "runtime-implementation"}}
		}, "wired provider"},
		{"stale artifact evidence", func(_ []ContributionRequirement, o []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			o[2].Provider.ArtifactDigest = "replacement"
		}, "slot runtime"},
		{"duplicate slot", func(r []ContributionRequirement, _ []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			r[2].Slot = r[0].Slot
		}, "duplicate contribution slot"},
		{"missing meaning pin", func(r []ContributionRequirement, _ []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			r[2].Meaning.Fingerprint = ""
		}, "incomplete"},
		{"invalid meaning version", func(r []ContributionRequirement, _ []ContributionOffer, _ []Contract, _ *[]ConformanceEvidence) {
			r[2].Meaning.Version = "latest"
		}, "meaning version"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r, o, c, e := contributionFixture()
			test.mutate(r, o, c, &e)
			bindings, err := planContributions(r, o, c, e)
			if err == nil || !strings.Contains(err.Error(), test.message) || bindings != nil {
				t.Fatalf("invalid composition returned partial/successful plan: bindings=%+v err=%v", bindings, err)
			}
		})
	}
}

func TestContributionPlanAmbiguityAndExactWiring(t *testing.T) {
	r, o, c, e := contributionFixture()
	alternative := o[2]
	alternative.Provider.PluginID = "reference.alternative-runtime"
	o = append(o, alternative)
	e = append(e, evidenceFor(alternative.Provider, c[2], "synthetic-alternative"))
	if _, err := planContributions(r, o, c, e); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous provider should not depend on order: %v", err)
	}
	p := alternative.Provider
	r[2].Wirings = []Wiring{{ContractID: p.ContractID, PluginID: p.PluginID, PluginVersion: p.PluginVersion, ImplementationID: p.ImplementationID, ArtifactDigest: p.ArtifactDigest}}
	bindings, err := planContributions(r, o, c, e)
	if err != nil || bindings[1].Resolution.Provider != p {
		t.Fatalf("explicit provider was not retained: %+v %v", bindings, err)
	}
}
