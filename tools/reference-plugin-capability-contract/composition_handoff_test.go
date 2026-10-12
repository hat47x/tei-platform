package main

import (
	"strings"
	"testing"

	"tei/internal/composition"
	handoff "tei/reference/composition-evidence-handoff"
)

func handoffSourceFixture(t *testing.T) (
	*composition.Registry, composition.WiringSetFinding,
	Contract, Provider, ResolveInput, CompositionResolutionExpectation,
) {
	t.Helper()
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.xead.source", "xead.reader.v0")
	r := composition.NewRegistry()
	_, err := r.Apply(0, []composition.Provider{{
		ID: "source-reader", Scope: "project-a",
		Offers: composition.Contract{
			ID: contract.ID, Version: contract.Version,
			// The composition Shape is a separate declared interface identity:
			// NEVER equate it with #951's full ContractFingerprint.
			Shape: "source-adapter-request-result-failure-v0",
		},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	set := r.InspectWiringSet([]composition.WiringRequirement{{
		Scope: "project-a",
		Contract: composition.Contract{
			ID: contract.ID, Version: contract.Version,
			Shape: "source-adapter-request-result-failure-v0",
		},
	}})
	if set.Verdict != composition.ProposalCandidate || len(set.Findings) != 1 ||
		len(set.Findings[0].VerificationTargets) != 1 {
		t.Fatalf("unexpected registry candidate: %+v", set)
	}
	input := ResolveInput{
		Scope:       ScopeAuthoringTool,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
		Evidence:    []ConformanceEvidence{evidenceFor(provider, contract, "reference-evidence-a")},
	}
	expected := CompositionResolutionExpectation{
		Target: set.Findings[0].VerificationTargets[0],
		Provider: provider,
	}
	return r, set, contract, provider, input, expected
}

func TestCompositionHandoffUsesRealCapabilityResolution(t *testing.T) {
	registry, set, contract, provider, input, expected := handoffSourceFixture(t)
	resolved, err := resolveCapability(input)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := resolveCompositionEvidenceReference(input, expected)
	if err != nil {
		t.Fatal(err)
	}
	if ref.ContractFingerprint != fingerprint(contract) ||
		ref.ContractFingerprint != resolved.ContractFingerprint ||
		ref.ResolutionFingerprint != resolved.ResolutionFingerprint ||
		ref.EvidenceFingerprint != resolved.EvidenceFingerprint ||
		ref.ArtifactDigest != provider.ArtifactDigest ||
		ref.AuthorityScope != string(ScopeAuthoringTool) {
		t.Fatalf("real capability resolution was not preserved: %+v", ref)
	}
	report := handoff.ReconcileCurrent(registry, set, []handoff.EvidenceReference{ref})
	if report.Blocked || len(report.Subjects) != 1 ||
		report.Subjects[0].State != handoff.ReferenceLinked ||
		report.Subjects[0].Reference == nil {
		t.Fatalf("reference-only handoff failed: %+v", report)
	}
}

func TestCompositionHandoffRejectsUnconformantAndScopeDrift(t *testing.T) {
	_, set, contract, provider, input, expected := handoffSourceFixture(t)
	cases := []struct {
		name   string
		mutate func(*ResolveInput)
	}{
		{"without-conformance", func(i *ResolveInput) { i.Evidence = nil }},
		{"wrong-authority-scope", func(i *ResolveInput) { i.Scope = ScopeApplicationRuntime }},
		{"changed-contract-semantics", func(i *ResolveInput) {
			i.Contracts[0].ResultSemantics = "different-result"
		}},
		{"different-provider", func(i *ResolveInput) {
			different := sourceAdapterProvider(contract, ScopeAuthoringTool, "reference.compat.other.source", "different-reader")
			i.Providers = []Provider{different}
			i.Evidence = []ConformanceEvidence{evidenceFor(different, contract, "reference-evidence-b")}
		}},
		{"wrong-major", func(i *ResolveInput) { i.Requirement.CompatibleMajor = 2 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := input
			in.Contracts = append([]Contract(nil), input.Contracts...)
			in.Providers = append([]Provider(nil), input.Providers...)
			in.Evidence = append([]ConformanceEvidence(nil), input.Evidence...)
			tc.mutate(&in)
			if got, err := resolveCompositionEvidenceReference(in, expected); err == nil {
				t.Fatalf("invalid crosswalk returned a reference: %+v", got)
			}
			report := handoff.Reconcile(set, nil)
			if report.Blocked || report.Subjects[0].State != handoff.ReferenceMissing {
				t.Fatalf("missing legitimate reference became accepted: %+v", report)
			}
		})
	}
	_ = provider
}

func TestCompositionHandoffPinsExactVersionArtifactAndFingerprint(t *testing.T) {
	_, _, _, provider, input, expected := handoffSourceFixture(t)
	for _, change := range []struct {
		name   string
		mutate func(*CompositionResolutionExpectation)
	}{
		{"version", func(e *CompositionResolutionExpectation) { e.Target.Contract.Version = "1.1.0" }},
		{"artifact", func(e *CompositionResolutionExpectation) {
			e.Provider.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
		}},
		{"contract-fingerprint", func(e *CompositionResolutionExpectation) {
			e.Provider.ContractFingerprint = "sha256:" + strings.Repeat("0", 64)
		}},
		{"implementation", func(e *CompositionResolutionExpectation) { e.Provider.ImplementationID = "other" }},
	} {
		t.Run(change.name, func(t *testing.T) {
			b := expected
			change.mutate(&b)
			if candidate, err := resolveCompositionEvidenceReference(input, b); err == nil {
				t.Fatalf("drifted expected binding was accepted: %+v", candidate)
			}
		})
	}
	_ = provider
}

func TestCompositionHandoffFreshEvidenceAndRegistryEpochsRemainSeparate(t *testing.T) {
	registry, set, contract, provider, input, expected := handoffSourceFixture(t)
	first, err := resolveCompositionEvidenceReference(input, expected)
	if err != nil {
		t.Fatal(err)
	}
	input.Evidence = []ConformanceEvidence{evidenceFor(provider, contract, "reference-evidence-rotated")}
	rotated, err := resolveCompositionEvidenceReference(input, expected)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.EvidenceFingerprint == first.EvidenceFingerprint ||
		rotated.ResolutionFingerprint != first.ResolutionFingerprint {
		t.Fatalf("conformance evidence rotation altered semantic resolution identity: %+v %+v", first, rotated)
	}
	report := handoff.ReconcileCurrent(registry, set, []handoff.EvidenceReference{first, rotated})
	if report.Subjects[0].State != handoff.ReferenceAmbiguous {
		t.Fatalf("two exact references silently selected: %+v", report)
	}
	// Removing and registering an identical logical provider makes the
	// previous Composition selection stale without touching #951 evidence.
	removed, err := registry.Apply(set.Revision, nil, []string{"source-reader"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.Apply(removed.Revision, []composition.Provider{{
		ID: "source-reader", Scope: "project-a",
		Offers: composition.Contract{ID: contract.ID, Version: contract.Version, Shape: "source-adapter-request-result-failure-v0"},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	next := registry.InspectWiringSet([]composition.WiringRequirement{{
		Scope: "project-a", Contract: composition.Contract{ID: contract.ID, Version: contract.Version, Shape: "source-adapter-request-result-failure-v0"},
	}})
	if next.Verdict != composition.ProposalCandidate ||
		next.Findings[0].VerificationTargets[0].Epoch == expected.Target.Epoch {
		t.Fatalf("provider reinstallation did not change declaration epoch: %+v", next)
	}
	if old := handoff.ReconcileCurrent(registry, set, []handoff.EvidenceReference{first}); !old.Blocked {
		t.Fatalf("obsolete pre-reinstallation Registry proposal remained current: %+v", old)
	}
	stale := handoff.ReconcileCurrent(registry, next, []handoff.EvidenceReference{first})
	if stale.Blocked || stale.Subjects[0].State != handoff.ReferenceStale {
		t.Fatalf("stale exact reference was accepted: %+v", stale)
	}
}

func TestCompositionHandoffRefusesAmbiguousCapabilityArtifacts(t *testing.T) {
	_, _, contract, provider, input, expected := handoffSourceFixture(t)
	second := provider
	second.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	input.Providers = []Provider{provider, second}
	input.Evidence = []ConformanceEvidence{
		evidenceFor(provider, contract, "first-artifact"),
		evidenceFor(second, contract, "second-artifact"),
	}
	if got, err := resolveCompositionEvidenceReference(input, expected); err == nil {
		t.Fatalf("physical artifact ambiguity accepted: %+v", got)
	}
}
