package main

import (
	"strings"
	"testing"

	"tei/internal/subject"
)

func namespaceAuthorityEvidence(hex string) string {
	return "sha256:" + strings.Repeat(hex, 64)
}

func TestSemanticExtensionCannotFirstClaimStandardNamespace(t *testing.T) {
	for _, namespace := range []string{"std", "std.validation", "std.data.custom"} {
		descriptor := ExtensionDescriptor{
			Schema:    extensionSchema,
			ID:        "third.party.extension",
			Namespace: namespace,
			Version:   "1.0.0",
		}
		catalog, err := resolveSemanticCatalog(
			CoreCatalogInput{},
			[]ExtensionDescriptor{descriptor},
			[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if !catalogHasUnresolvedCode(catalog, "namespace-authority-missing") {
			t.Fatalf("namespace %q was not protected: %+v", namespace, catalog.Unresolved)
		}
	}
}

func TestAcceptedAuthorityAllowsStandardExtensionThroughSameContract(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.validation",
		Namespace: "std.validation",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "validation-policy", ID: "std.validation.required"},
		}},
	}
	grant := SemanticNamespaceAuthority{
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
		[]SemanticNamespaceAuthority{grant},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("accepted standard authority remained unresolved: %+v", catalog.Unresolved)
	}
	if len(catalog.NamespaceAuthorities) != 1 ||
		catalog.NamespaceAuthorities[0].Namespace != descriptor.Namespace ||
		catalog.NamespaceAuthorities[0].Extension != descriptor.ID {
		t.Fatalf("namespace authority was not preserved: %+v", catalog.NamespaceAuthorities)
	}
	if !catalogHasSubject(catalog, descriptor.Concepts[0].Subject) {
		t.Fatalf("authorized standard extension did not contribute its concept: %+v", catalog.Subjects)
	}
}

func TestNamespaceAuthoritySelectsOneOwnerWithoutLoadOrderWinner(t *testing.T) {
	left := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.left",
		Namespace: "shared.policy",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "policy-view", ID: "policy.left"},
		}},
	}
	right := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.right",
		Namespace: "shared.policy",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "policy-view", ID: "policy.right"},
		}},
	}
	grant := SemanticNamespaceAuthority{
		Namespace:           left.Namespace,
		Extension:           right.ID,
		EvidenceFingerprint: namespaceAuthorityEvidence("b"),
	}

	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{right, left},
		[]ExtensionRequirement{
			{Extension: left.ID, CompatibleMajor: 1},
			{Extension: right.ID, CompatibleMajor: 1},
		},
		nil,
		nil,
		[]SemanticNamespaceAuthority{grant},
	)
	if err != nil {
		t.Fatal(err)
	}
	if catalogHasSubject(catalog, left.Concepts[0].Subject) {
		t.Fatalf("unauthorized namespace claimant contributed meaning: %+v", catalog.Subjects)
	}
	if !catalogHasSubject(catalog, right.Concepts[0].Subject) {
		t.Fatalf("authorized namespace owner was blocked: %+v", catalog.Subjects)
	}
	if !catalogHasUnresolvedCode(catalog, "namespace-authority-mismatch") {
		t.Fatalf("unauthorized claimant was not surfaced: %+v", catalog.Unresolved)
	}
	if catalogHasUnresolvedCode(catalog, "namespace-ownership-conflict") {
		t.Fatalf("explicit authority should replace load-order conflict: %+v", catalog.Unresolved)
	}
}

func TestNamespaceAuthorityCannotOverrideCoreOwnership(t *testing.T) {
	coreNamespace := subject.Subject{Kind: "semantic", ID: "shared.policy"}
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.extension",
		Namespace: "shared.policy",
		Version:   "1.0.0",
	}
	grant := SemanticNamespaceAuthority{
		Namespace:           descriptor.Namespace,
		Extension:           descriptor.ID,
		EvidenceFingerprint: namespaceAuthorityEvidence("c"),
	}
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{Subjects: []subject.Subject{coreNamespace}},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{grant},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !catalogHasUnresolvedCode(catalog, "namespace-ownership-conflict") {
		t.Fatalf("authority grant overrode core ownership: %+v", catalog.Unresolved)
	}
}

func TestNamespaceAuthorityEvidenceDoesNotChangeSemanticIdentity(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.validation",
		Namespace: "std.validation",
		Version:   "1.0.0",
	}
	resolve := func(hex string) SemanticCatalog {
		catalog, err := resolveSemanticCatalogWithAllInputs(
			CoreCatalogInput{},
			[]ExtensionDescriptor{descriptor},
			[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
			nil,
			nil,
			[]SemanticNamespaceAuthority{{
				Namespace:           descriptor.Namespace,
				Extension:           descriptor.ID,
				EvidenceFingerprint: namespaceAuthorityEvidence(hex),
			}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(catalog.Unresolved) != 0 {
			t.Fatalf("unresolved=%+v", catalog.Unresolved)
		}
		return catalog
	}

	left := resolve("d")
	right := resolve("e")
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf(
			"authority Evidence changed semantic identity: %s != %s",
			left.ResolutionFingerprint,
			right.ResolutionFingerprint,
		)
	}
	if left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatalf("authority Evidence change was not reflected in Evidence identity: %s", left.EvidenceFingerprint)
	}
}

func TestSemanticExtensionReferenceNamespaceRemainsAvailable(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.example",
		Namespace: "reference.example",
		Version:   "1.0.0",
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("reference namespace was rejected: %+v", catalog.Unresolved)
	}
}
