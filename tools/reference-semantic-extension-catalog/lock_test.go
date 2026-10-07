package main

import (
	"strings"
	"testing"

	"tei/internal/subject"
)

func semanticLockDescriptor(version, conceptID string) ExtensionDescriptor {
	return ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.locked-policy",
		Namespace: "reference.locked-policy",
		Version:   version,
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "policy-view", ID: conceptID},
		}},
	}
}

func TestSemanticLockKeepsExactVersionWhenCatalogAddsNewerCompatibleVersion(t *testing.T) {
	v100 := semanticLockDescriptor("1.0.0", "policy.v1")
	initial, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{v100},
		[]ExtensionRequirement{{Extension: v100.ID, CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := buildSemanticLock(initial)
	if err != nil {
		t.Fatal(err)
	}

	v110 := semanticLockDescriptor("1.1.0", "policy.v1_1")
	available := []ExtensionDescriptor{v110, v100}
	if err := verifySemanticLockReplay(lock, CoreCatalogInput{}, available); err != nil {
		t.Fatalf("new catalog version invalidated exact lock: %v", err)
	}

	reresolved, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		available,
		[]ExtensionRequirement{{Extension: v100.ID, CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if reresolved.ResolutionFingerprint == lock.SemanticSetFingerprint {
		t.Fatalf("candidate re-resolution did not observe newer semantic version")
	}
}

func TestSemanticLockRejectsSameVersionDeclarationDrift(t *testing.T) {
	descriptor := semanticLockDescriptor("1.0.0", "policy.original")
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

	changed := semanticLockDescriptor("1.0.0", "policy.changed")
	err = verifySemanticLockReplay(lock, CoreCatalogInput{}, []ExtensionDescriptor{changed})
	if err == nil || !strings.Contains(err.Error(), "declaration fingerprint is stale") {
		t.Fatalf("same-version semantic drift was not rejected: %v", err)
	}
}

func TestSemanticLockRejectsUnresolvedCatalog(t *testing.T) {
	descriptor := semanticLockDescriptor("1.0.0", "policy.example")
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{{
			Namespace:           "unrelated.namespace",
			Extension:           descriptor.ID,
			EvidenceFingerprint: namespaceAuthorityEvidence("a"),
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) == 0 {
		t.Fatal("fixture did not create unresolved catalog")
	}
	if _, err := buildSemanticLock(catalog); err == nil {
		t.Fatal("unresolved catalog produced a semantic lock")
	}
}

func TestSemanticLockReconstructsInterfaceWiringWithoutEvidenceIdentity(t *testing.T) {
	contract, owner, consumer, provider := semanticInterfaceFixture()
	descriptors := []ExtensionDescriptor{owner, consumer, provider}
	conformance := semanticInterfaceConformanceFor(contract, provider.ID, "b")
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
		t.Fatalf("semantic lock did not reconstruct interface wiring: %v", err)
	}
}

func TestSemanticLockReconstructsAcceptedNamespaceAuthorityWithoutEvidenceIdentity(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.reference",
		Namespace: "std.reference",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "standard-view", ID: "std.reference.example"},
		}},
	}
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{{
			Namespace:           descriptor.Namespace,
			Extension:           descriptor.ID,
			EvidenceFingerprint: namespaceAuthorityEvidence("c"),
		}},
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
		t.Fatalf("semantic lock did not reconstruct namespace authority: %v", err)
	}
}

func TestSemanticLockDetectsRootAndPolicyDrift(t *testing.T) {
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

	rootDrift := lock
	rootDrift.Requirements = append(rootDrift.Requirements, ExtensionRequirement{
		Extension:       "tei.reference.missing",
		CompatibleMajor: 1,
	})
	if err := verifySemanticLockReplay(rootDrift, CoreCatalogInput{}, []ExtensionDescriptor{descriptor}); err == nil ||
		!strings.Contains(err.Error(), "root requirements fingerprint is stale") {
		t.Fatalf("root requirement drift was not detected: %v", err)
	}

	policyDrift := lock
	policyDrift.ResolutionPolicy = "different-policy/v1"
	if err := verifySemanticLockReplay(policyDrift, CoreCatalogInput{}, []ExtensionDescriptor{descriptor}); err == nil ||
		!strings.Contains(err.Error(), "resolution policy") {
		t.Fatalf("resolution policy drift was not detected: %v", err)
	}
}
