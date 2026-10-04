package main

import (
	"strings"
	"testing"
)

func TestSemanticLockEvidenceBindingRequiresCurrentConformanceEvidence(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	descriptors := []ExtensionDescriptor{owner, consumer, provider}
	conformance := semanticInterfaceConformanceFor(contract, provider.ID, "a")
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		descriptors,
		semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID),
		nil,
		[]SemanticInterfaceConformance{conformance},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("fixture unresolved=%+v", catalog.Unresolved)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, descriptors); err != nil {
		t.Fatalf("semantic replay failed without current Evidence: %v", err)
	}
	if err := verifySemanticLockAcceptedEvidenceBindings(lock, nil, nil); err == nil ||
		!strings.Contains(err.Error(), "accepted conformance Evidence") {
		t.Fatalf("missing current Conformance Evidence was not detected: %v", err)
	}
	if err := verifySemanticLockAcceptedEvidenceBindings(
		lock,
		[]SemanticInterfaceConformance{conformance},
		nil,
	); err != nil {
		t.Fatalf("exact Conformance Evidence did not qualify locked wiring: %v", err)
	}
}

func TestSemanticLockEvidenceBindingAllowsConformanceEvidenceRefresh(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	initial := semanticInterfaceConformanceFor(contract, provider.ID, "a")
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{owner, consumer, provider},
		semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID),
		nil,
		[]SemanticInterfaceConformance{initial},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	refreshed := semanticInterfaceConformanceFor(contract, provider.ID, "b")
	if refreshed.EvidenceFingerprint == initial.EvidenceFingerprint {
		t.Fatal("fixture did not refresh Evidence fingerprint")
	}
	if err := verifySemanticLockAcceptedEvidenceBindings(
		lock,
		[]SemanticInterfaceConformance{refreshed},
		nil,
	); err != nil {
		t.Fatalf("Evidence refresh for the same exact contract/profile was rejected: %v", err)
	}
}

func TestSemanticLockEvidenceBindingRejectsWrongConformanceContract(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	conformance := semanticInterfaceConformanceFor(contract, provider.ID, "a")
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{owner, consumer, provider},
		semanticInterfaceRoots(owner.ID, consumer.ID, provider.ID),
		nil,
		[]SemanticInterfaceConformance{conformance},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	wrong := conformance
	wrong.ContractFingerprint = "sha256:" + strings.Repeat("f", 64)
	if err := verifySemanticLockAcceptedEvidenceBindings(
		lock,
		[]SemanticInterfaceConformance{wrong},
		nil,
	); err == nil || !strings.Contains(err.Error(), "accepted conformance Evidence") {
		t.Fatalf("wrong contract Conformance Evidence was accepted: %v", err)
	}
}

func TestSemanticLockEvidenceBindingRequiresCurrentNamespaceAuthority(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.reference",
		Namespace: "std.reference",
		Version:   "1.0.0",
	}
	authority := SemanticNamespaceAuthority{
		Namespace:           descriptor.Namespace,
		Extension:           descriptor.ID,
		EvidenceFingerprint: namespaceAuthorityEvidence("a"),
	}
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{authority},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("fixture unresolved=%+v", catalog.Unresolved)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, []ExtensionDescriptor{descriptor}); err != nil {
		t.Fatalf("semantic replay failed without current Authority Evidence: %v", err)
	}
	if err := verifySemanticLockAcceptedEvidenceBindings(lock, nil, nil); err == nil ||
		!strings.Contains(err.Error(), "namespace Authority Evidence") {
		t.Fatalf("missing current namespace Authority Evidence was not detected: %v", err)
	}
	if err := verifySemanticLockAcceptedEvidenceBindings(
		lock,
		nil,
		[]SemanticNamespaceAuthority{authority},
	); err != nil {
		t.Fatalf("exact namespace Authority Evidence did not qualify locked owner: %v", err)
	}
}

func TestSemanticLockEvidenceBindingRejectsDifferentNamespaceOwner(t *testing.T) {
	lock := SemanticLock{
		NamespaceAuthorities: []SemanticLockNamespaceAuthority{{
			Namespace: "std.validation",
			Extension: "tei.std.validation",
		}},
	}
	wrong := SemanticNamespaceAuthority{
		Namespace:           "std.validation",
		Extension:           "third.party.validation",
		EvidenceFingerprint: namespaceAuthorityEvidence("a"),
	}
	if err := verifySemanticLockAcceptedEvidenceBindings(
		lock,
		nil,
		[]SemanticNamespaceAuthority{wrong},
	); err == nil || !strings.Contains(err.Error(), "but accepted Authority Evidence names") {
		t.Fatalf("different namespace owner was accepted: %v", err)
	}
}
