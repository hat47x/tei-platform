package main

import (
	"strings"
	"testing"

	"tei/internal/subject"
)

func TestSemanticLockRejectsCanonicalizationPolicyDrift(t *testing.T) {
	descriptor := semanticLockDescriptor("1.0.0", "policy.example")
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	lock.CanonicalizationPolicy = "different-canonicalization/v1"
	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, []ExtensionDescriptor{descriptor}); err == nil ||
		!strings.Contains(err.Error(), "canonicalization policy") {
		t.Fatalf("canonicalization policy drift was not rejected: %v", err)
	}
}

func TestSemanticLockRejectsCoreSubjectDrift(t *testing.T) {
	core := CoreCatalogInput{
		Subjects: []subject.Subject{{
			Kind: "semantic-data",
			ID:   "customer.customer",
		}},
		SubjectFingerprint: "sha256:" + strings.Repeat("1", 64),
	}
	descriptor := semanticLockDescriptor("1.0.0", "policy.example")
	catalog, err := resolveSemanticCatalog(
		core,
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	changedCore := core
	changedCore.SubjectFingerprint = "sha256:" + strings.Repeat("2", 64)
	if err := verifySemanticLockReplay(lock, changedCore, []ExtensionDescriptor{descriptor}); err == nil ||
		!strings.Contains(err.Error(), "core subject fingerprint is stale") {
		t.Fatalf("Core subject drift was not rejected: %v", err)
	}
}

func TestSemanticLockRejectsInterfaceWiringTamper(t *testing.T) {
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
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.InterfaceWirings) != 1 {
		t.Fatalf("fixture wiring=%+v", lock.InterfaceWirings)
	}

	lock.InterfaceWirings[0].ProviderExtension = "tei.reference.provider-missing"
	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, descriptors); err == nil {
		t.Fatal("tampered interface provider wiring replayed successfully")
	}
}

func TestSemanticLockRejectsInterfaceContractFingerprintTamper(t *testing.T) {
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
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	lock.InterfaceWirings[0].ContractFingerprint =
		"sha256:" + strings.Repeat("f", 64)
	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, descriptors); err == nil {
		t.Fatal("tampered interface contract fingerprint replayed successfully")
	}
}

func TestSemanticLockRejectsNamespaceOwnerTamper(t *testing.T) {
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
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.NamespaceAuthorities) != 1 {
		t.Fatalf("fixture namespace authorities=%+v", lock.NamespaceAuthorities)
	}

	lock.NamespaceAuthorities[0].Extension = "third.party.reference"
	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, []ExtensionDescriptor{descriptor}); err == nil {
		t.Fatal("tampered namespace owner replayed successfully")
	}
}

func TestSemanticLockRejectsSemanticSetFingerprintTamper(t *testing.T) {
	descriptor := semanticLockDescriptor("1.0.0", "policy.example")
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}

	lock.SemanticSetFingerprint = "sha256:" + strings.Repeat("f", 64)
	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, []ExtensionDescriptor{descriptor}); err == nil ||
		!strings.Contains(err.Error(), "semantic set fingerprint is stale") {
		t.Fatalf("semantic set fingerprint tamper was not rejected: %v", err)
	}
}
