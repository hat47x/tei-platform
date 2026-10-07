package main

import (
	"strings"
	"testing"
)

func semanticInterfaceFixture() (
	SemanticInterfaceContract,
	ExtensionDescriptor,
	ExtensionDescriptor,
	ExtensionDescriptor,
) {
	contract := SemanticInterfaceContract{
		ID:                     "reference.interface.acyclic",
		Version:                "1.0.0",
		InputSemantics:         []string{"directed relation"},
		OutputSemantics:        []string{"acyclic verdict"},
		UnknownSemantics:       "insufficient graph remains unknown",
		FailureSemantics:       "validation failure is blocking",
		ProviderConformanceRef: "reference.conformance.acyclic-v1",
	}
	fingerprint := semanticInterfaceFingerprint(contract)
	owner := ExtensionDescriptor{
		Schema:     extensionSchema,
		ID:         "tei.reference.interface-contract",
		Namespace:  "reference.interface",
		Version:    "1.0.0",
		Interfaces: []SemanticInterfaceContract{contract},
	}
	consumer := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.consumer",
		Namespace: "reference.consumer",
		Version:   "1.0.0",
		RequiresInterfaces: []SemanticInterfaceRequirement{{
			Interface:       contract.ID,
			CompatibleMajor: 1,
		}},
	}
	provider := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.provider-a",
		Namespace: "reference.provider-a",
		Version:   "1.0.0",
		ProvidesInterfaces: []SemanticInterfaceProvider{{
			Interface:           contract.ID,
			ContractVersion:     contract.Version,
			ContractFingerprint: fingerprint,
		}},
	}
	return contract, owner, consumer, provider
}

func semanticInterfaceRoots(ids ...string) []ExtensionRequirement {
	out := make([]ExtensionRequirement, 0, len(ids))
	for _, id := range ids {
		out = append(out, ExtensionRequirement{Extension: id, CompatibleMajor: 1})
	}
	return out
}

func semanticInterfaceConformanceFor(
	contract SemanticInterfaceContract,
	providerID string,
	evidenceHex string,
) SemanticInterfaceConformance {
	return SemanticInterfaceConformance{
		ProviderExtension:   providerID,
		Interface:           contract.ID,
		ContractFingerprint: semanticInterfaceFingerprint(contract),
		Profile:             contract.ProviderConformanceRef,
		EvidenceFingerprint: "sha256:" + strings.Repeat(evidenceHex, 64),
	}
}

func TestSemanticInterfaceRequiresIndependentConformanceEvidence(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	catalog, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{owner, consumer, provider},
		semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID),
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.InterfaceWirings) != 0 {
		t.Fatalf("provider self-claim created wiring without conformance Evidence: %+v", catalog.InterfaceWirings)
	}
	if !catalogHasUnresolvedCode(catalog, "interface-provider-conformance-unverified") {
		t.Fatalf("missing independent conformance was not surfaced: %+v", catalog.Unresolved)
	}
	if !catalogHasUnresolvedCode(catalog, "interface-provider-unavailable") {
		t.Fatalf("consumer did not remain unresolved: %+v", catalog.Unresolved)
	}
	_ = contract
}

func TestSemanticInterfaceUniqueConformantProviderResolvesDeterministically(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	conformance := semanticInterfaceConformanceFor(contract, provider.ID, "a")
	catalog, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{provider, consumer, owner},
		semanticInterfaceRoots(consumer.ID, provider.ID, owner.ID),
		nil,
		[]SemanticInterfaceConformance{conformance},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("unexpected unresolved: %+v", catalog.Unresolved)
	}
	if len(catalog.Interfaces) != 1 || len(catalog.InterfaceWirings) != 1 {
		t.Fatalf("interface resolution is incomplete: interfaces=%+v wiring=%+v", catalog.Interfaces, catalog.InterfaceWirings)
	}
	wiring := catalog.InterfaceWirings[0]
	if wiring.ConsumerExtension != consumer.ID ||
		wiring.ProviderExtension != provider.ID ||
		wiring.Interface != contract.ID ||
		wiring.Selection != "unique-provider" ||
		wiring.ProviderConformanceProfile != contract.ProviderConformanceRef ||
		wiring.ConformanceEvidence != conformance.EvidenceFingerprint {
		t.Fatalf("wiring=%+v", wiring)
	}
}

func TestSemanticInterfaceRejectsProviderContractSpoof(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	provider.ProvidesInterfaces[0].ContractFingerprint = "sha256:" + strings.Repeat("0", 64)
	conformance := semanticInterfaceConformanceFor(contract, provider.ID, "b")
	catalog, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{owner, consumer, provider},
		semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID),
		nil,
		[]SemanticInterfaceConformance{conformance},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.InterfaceWirings) != 0 {
		t.Fatalf("spoofed provider was wired: %+v", catalog.InterfaceWirings)
	}
	if !catalogHasUnresolvedCode(catalog, "interface-provider-contract-mismatch") {
		t.Fatalf("contract spoof was not rejected: %+v", catalog.Unresolved)
	}
}

func TestSemanticInterfaceMultipleProvidersRequireExplicitWiring(t *testing.T) {
	contract, owner, consumer, providerA := semanticInterfaceFixture()
	providerB := providerA
	providerB.ID = "tei.reference.provider-b"
	providerB.Namespace = "reference.provider-b"
	conformanceA := semanticInterfaceConformanceFor(contract, providerA.ID, "c")
	conformanceB := semanticInterfaceConformanceFor(contract, providerB.ID, "d")
	descriptors := []ExtensionDescriptor{owner, consumer, providerA, providerB}
	roots := semanticInterfaceRoots(owner.ID, consumer.ID, providerA.ID, providerB.ID)
	conformances := []SemanticInterfaceConformance{conformanceA, conformanceB}

	ambiguous, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{}, descriptors, roots, nil, conformances,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(ambiguous.InterfaceWirings) != 0 || !catalogHasUnresolvedCode(ambiguous, "interface-provider-ambiguous") {
		t.Fatalf("multiple providers were implicitly selected: wiring=%+v unresolved=%+v", ambiguous.InterfaceWirings, ambiguous.Unresolved)
	}

	resolved, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{},
		descriptors,
		roots,
		[]SemanticInterfaceWiring{{
			ConsumerExtension: consumer.ID,
			Interface:         contract.ID,
			ProviderExtension: providerB.ID,
		}},
		conformances,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Unresolved) != 0 || len(resolved.InterfaceWirings) != 1 {
		t.Fatalf("explicit wiring did not resolve ambiguity: wiring=%+v unresolved=%+v", resolved.InterfaceWirings, resolved.Unresolved)
	}
	if got := resolved.InterfaceWirings[0]; got.ProviderExtension != providerB.ID || got.Selection != "explicit" {
		t.Fatalf("explicit wiring=%+v", got)
	}
}

func TestSemanticInterfaceSelectionPathDoesNotChangeMeaningIdentity(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	descriptors := []ExtensionDescriptor{owner, consumer, provider}
	roots := semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID)
	conformances := []SemanticInterfaceConformance{
		semanticInterfaceConformanceFor(contract, provider.ID, "9"),
	}

	unique, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{},
		descriptors,
		roots,
		nil,
		conformances,
	)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := resolveSemanticCatalogWithInterfaceInputs(
		CoreCatalogInput{},
		descriptors,
		roots,
		[]SemanticInterfaceWiring{{
			ConsumerExtension: consumer.ID,
			Interface:         contract.ID,
			ProviderExtension: provider.ID,
		}},
		conformances,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(unique.Unresolved) != 0 || len(explicit.Unresolved) != 0 {
		t.Fatalf("fixture unresolved: unique=%+v explicit=%+v", unique.Unresolved, explicit.Unresolved)
	}
	if unique.InterfaceWirings[0].Selection == explicit.InterfaceWirings[0].Selection {
		t.Fatalf("fixture did not exercise different selection paths: unique=%+v explicit=%+v", unique.InterfaceWirings, explicit.InterfaceWirings)
	}
	if unique.ResolutionFingerprint != explicit.ResolutionFingerprint {
		t.Fatalf(
			"selection path changed semantic identity: %s != %s",
			unique.ResolutionFingerprint,
			explicit.ResolutionFingerprint,
		)
	}
	if unique.EvidenceFingerprint == explicit.EvidenceFingerprint {
		t.Fatalf("selection path was not retained in Evidence identity")
	}
}

func TestSemanticInterfaceWiringChangesSemanticResolutionIdentity(t *testing.T) {
	contract, owner, consumer, providerA := semanticInterfaceFixture()
	providerB := providerA
	providerB.ID = "tei.reference.provider-b"
	providerB.Namespace = "reference.provider-b"
	descriptors := []ExtensionDescriptor{owner, consumer, providerA, providerB}
	roots := semanticInterfaceRoots(owner.ID, consumer.ID, providerA.ID, providerB.ID)
	conformances := []SemanticInterfaceConformance{
		semanticInterfaceConformanceFor(contract, providerA.ID, "e"),
		semanticInterfaceConformanceFor(contract, providerB.ID, "f"),
	}
	resolve := func(providerID string) SemanticCatalog {
		catalog, err := resolveSemanticCatalogWithInterfaceInputs(
			CoreCatalogInput{}, descriptors, roots,
			[]SemanticInterfaceWiring{{
				ConsumerExtension: consumer.ID,
				Interface:         contract.ID,
				ProviderExtension: providerID,
			}},
			conformances,
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(catalog.Unresolved) != 0 {
			t.Fatalf("provider %s unresolved=%+v", providerID, catalog.Unresolved)
		}
		return catalog
	}
	left := resolve(providerA.ID)
	right := resolve(providerB.ID)
	if left.ResolutionFingerprint == right.ResolutionFingerprint {
		t.Fatalf("provider wiring did not change semantic resolution identity: %s", left.ResolutionFingerprint)
	}
}

func TestSemanticInterfaceConformanceEvidenceDoesNotChangeSemanticIdentity(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	descriptors := []ExtensionDescriptor{owner, consumer, provider}
	roots := semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID)
	resolve := func(evidenceHex string) SemanticCatalog {
		catalog, err := resolveSemanticCatalogWithInterfaceInputs(
			CoreCatalogInput{}, descriptors, roots, nil,
			[]SemanticInterfaceConformance{semanticInterfaceConformanceFor(contract, provider.ID, evidenceHex)},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(catalog.Unresolved) != 0 {
			t.Fatalf("unresolved=%+v", catalog.Unresolved)
		}
		return catalog
	}
	left := resolve("1")
	right := resolve("2")
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf("conformance Evidence changed semantic identity: %s != %s", left.ResolutionFingerprint, right.ResolutionFingerprint)
	}
	if left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatalf("conformance Evidence change was not reflected in Evidence identity: %s", left.EvidenceFingerprint)
	}
}

func catalogHasUnresolvedCode(catalog SemanticCatalog, code string) bool {
	for _, unresolved := range catalog.Unresolved {
		if unresolved.Code == code {
			return true
		}
	}
	return false
}

func TestSemanticInterfaceConformanceIdentityIncludesExactContract(t *testing.T) {
	first := SemanticInterfaceConformance{
		ProviderExtension:   "tei.reference.provider-a",
		Interface:           "reference.interface.acyclic",
		ContractFingerprint: "sha256:" + strings.Repeat("a", 64),
		Profile:             "reference.conformance.acyclic-v1",
		EvidenceFingerprint: "sha256:" + strings.Repeat("c", 64),
	}
	second := SemanticInterfaceConformance{
		ProviderExtension:   first.ProviderExtension,
		Interface:           first.Interface,
		ContractFingerprint: "sha256:" + strings.Repeat("b", 64),
		Profile:             "reference.conformance.acyclic-v2",
		EvidenceFingerprint: "sha256:" + strings.Repeat("d", 64),
	}

	normalized, err := normalizeSemanticInterfaceConformance(
		[]SemanticInterfaceConformance{first, second},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized) != 2 {
		t.Fatalf("distinct contract conformance Evidence collapsed: %+v", normalized)
	}
	if got := normalized[semanticInterfaceConformanceKey(
		first.ProviderExtension,
		first.Interface,
		first.ContractFingerprint,
		first.Profile,
	)]; got != first {
		t.Fatalf("first contract conformance missing: %+v", got)
	}
	if got := normalized[semanticInterfaceConformanceKey(
		second.ProviderExtension,
		second.Interface,
		second.ContractFingerprint,
		second.Profile,
	)]; got != second {
		t.Fatalf("second contract conformance missing: %+v", got)
	}
}
