package main

import (
	"strings"
	"testing"

	"tei/internal/subject"
	authoritytrust "tei/tools/reference-authority-evidence-trust/trust"
)

func namespaceAuthorityEvidence(hex string) string {
	return "sha256:" + strings.Repeat(hex, 64)
}

func semanticNamespacePublisherForTest() authoritytrust.Issuer {
	return authoritytrust.Issuer{
		ID:      "tei.reference.semantic-authority-publisher",
		Version: "1.0.0",
	}
}

func semanticNamespaceAuthorityForTest(
	t *testing.T,
	namespace string,
	extension string,
	evidenceHex string,
	policyHex string,
	active bool,
) (SemanticNamespaceAuthority, []PublisherTrustGrant) {
	t.Helper()
	authority := SemanticNamespaceAuthority{
		Namespace:           namespace,
		Extension:           extension,
		Issuer:              semanticNamespacePublisherForTest(),
		EvidenceFingerprint: namespaceAuthorityEvidence(evidenceHex),
	}
	grant, err := authoritytrust.NewGrant(
		semanticNamespacePublisherAuthorityClass,
		semanticNamespaceTrustScope(namespace),
		authority.Issuer,
		namespaceAuthorityEvidence(policyHex),
		active,
	)
	if err != nil {
		t.Fatal(err)
	}
	return authority, []PublisherTrustGrant{grant}
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
	authority, publisherTrust := semanticNamespaceAuthorityForTest(
		t,
		descriptor.Namespace,
		descriptor.ID,
		"a",
		"1",
		true,
	)

	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{authority},
		publisherTrust,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("accepted standard authority remained unresolved: %+v", catalog.Unresolved)
	}
	if len(catalog.NamespaceAuthorities) != 1 ||
		catalog.NamespaceAuthorities[0].Namespace != descriptor.Namespace ||
		catalog.NamespaceAuthorities[0].Extension != descriptor.ID ||
		catalog.NamespaceAuthorities[0].TrustQualificationFingerprint == "" {
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
	authority, publisherTrust := semanticNamespaceAuthorityForTest(
		t,
		left.Namespace,
		right.ID,
		"b",
		"2",
		true,
	)

	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{right, left},
		[]ExtensionRequirement{
			{Extension: left.ID, CompatibleMajor: 1},
			{Extension: right.ID, CompatibleMajor: 1},
		},
		nil,
		nil,
		[]SemanticNamespaceAuthority{authority},
		publisherTrust,
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
	authority, publisherTrust := semanticNamespaceAuthorityForTest(
		t,
		descriptor.Namespace,
		descriptor.ID,
		"c",
		"3",
		true,
	)
	catalog, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{Subjects: []subject.Subject{coreNamespace}},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{authority},
		publisherTrust,
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
		authority, publisherTrust := semanticNamespaceAuthorityForTest(
			t,
			descriptor.Namespace,
			descriptor.ID,
			hex,
			"4",
			true,
		)
		catalog, err := resolveSemanticCatalogWithAllInputs(
			CoreCatalogInput{},
			[]ExtensionDescriptor{descriptor},
			[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
			nil,
			nil,
			[]SemanticNamespaceAuthority{authority},
			publisherTrust,
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

func TestNamespaceAuthorityRequiresTrustedPublisherEvenWithAcceptedEvidence(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.validation",
		Namespace: "std.validation",
		Version:   "1.0.0",
	}
	authority, _ := semanticNamespaceAuthorityForTest(
		t,
		descriptor.Namespace,
		descriptor.ID,
		"a",
		"5",
		true,
	)
	_, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{authority},
	)
	if err == nil || !strings.Contains(err.Error(), "publisher trust") {
		t.Fatalf("accepted Namespace Authority Evidence bypassed publisher trust: %v", err)
	}
}

func TestNamespaceAuthorityRejectsRevokedPublisherTrust(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.validation",
		Namespace: "std.validation",
		Version:   "1.0.0",
	}
	authority, publisherTrust := semanticNamespaceAuthorityForTest(
		t,
		descriptor.Namespace,
		descriptor.ID,
		"a",
		"6",
		false,
	)
	_, err := resolveSemanticCatalogWithAllInputs(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
		nil,
		nil,
		[]SemanticNamespaceAuthority{authority},
		publisherTrust,
	)
	if err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked namespace publisher remained trusted: %v", err)
	}
}

func TestNamespacePublisherTrustRefreshDoesNotChangeSemanticIdentity(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.std.validation",
		Namespace: "std.validation",
		Version:   "1.0.0",
	}
	authority, leftTrust := semanticNamespaceAuthorityForTest(
		t,
		descriptor.Namespace,
		descriptor.ID,
		"a",
		"7",
		true,
	)
	_, rightTrust := semanticNamespaceAuthorityForTest(
		t,
		descriptor.Namespace,
		descriptor.ID,
		"a",
		"8",
		true,
	)
	resolve := func(publisherTrust []PublisherTrustGrant) SemanticCatalog {
		catalog, err := resolveSemanticCatalogWithAllInputs(
			CoreCatalogInput{},
			[]ExtensionDescriptor{descriptor},
			[]ExtensionRequirement{{Extension: descriptor.ID, CompatibleMajor: 1}},
			nil,
			nil,
			[]SemanticNamespaceAuthority{authority},
			publisherTrust,
		)
		if err != nil {
			t.Fatal(err)
		}
		return catalog
	}
	left := resolve(leftTrust)
	right := resolve(rightTrust)
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatal("publisher trust refresh changed Semantic Set identity")
	}
	if left.NamespaceAuthorities[0].EvidenceFingerprint !=
		right.NamespaceAuthorities[0].EvidenceFingerprint {
		t.Fatal("publisher trust refresh changed Namespace Authority Evidence identity")
	}
	if left.NamespaceAuthorities[0].TrustQualificationFingerprint ==
		right.NamespaceAuthorities[0].TrustQualificationFingerprint {
		t.Fatal("publisher trust refresh was not observable")
	}
	if left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatal("publisher trust refresh was not reflected in catalog Evidence identity")
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
