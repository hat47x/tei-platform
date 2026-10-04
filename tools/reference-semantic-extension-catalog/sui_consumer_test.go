package main

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/subject"
	"tei/internal/workspace"
)

func suiReferenceCoreCatalog(t *testing.T) (CoreCatalogInput, subject.Subject) {
	t.Helper()
	root := repositoryRoot(t)
	workspaceRoot := filepath.Join(root, "reference", "sui-adoption", "workspace")
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(workspaceRoot, loaded)
	if err != nil {
		t.Fatal(err)
	}
	core := coreCatalogFromSourceIndex(index)
	var target subject.Subject
	for _, item := range core.Subjects {
		if item.ID == "sui.island" {
			if !target.IsZero() {
				t.Fatalf("sui.island is not unique in Source Index: first=%+v second=%+v", target, item)
			}
			target = item
		}
	}
	if target.IsZero() {
		t.Fatalf("sui.island was not found in SUI Reference Source Index: %+v", core.Subjects)
	}
	return core, target
}

func suiHierarchySemanticExtensionFixture(
	target subject.Subject,
) (
	SemanticInterfaceContract,
	ExtensionDescriptor,
	ExtensionDescriptor,
	SemanticInterfaceConformance,
) {
	contract := SemanticInterfaceContract{
		ID:                     "reference.hierarchy.integrity.evaluator",
		Version:                "1.0.0",
		InputSemantics:         []string{"hierarchy relation snapshot"},
		OutputSemantics:        []string{"pass fail unknown diagnostics"},
		UnknownSemantics:       "incomplete hierarchy input remains unknown",
		FailureSemantics:       "violated hierarchy integrity is blocking",
		ProviderConformanceRef: "reference.conformance.sui-hierarchy-v1",
	}
	hierarchy := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.hierarchy-integrity",
		Namespace: "reference.hierarchy.integrity",
		Version:   "1.0.0",
		Interfaces: []SemanticInterfaceContract{contract},
		RequiresInterfaces: []SemanticInterfaceRequirement{{
			Interface:       contract.ID,
			CompatibleMajor: 1,
		}},
		Constraints: []ExtensionConstraint{{
			ID:                 "sui-island-hierarchy-integrity",
			Subject:            target,
			ValidatorRef:       "reference.hierarchy.integrity.v1",
			EvaluatorInterface: contract.ID,
		}},
	}
	provider := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.sui-hierarchy-shadow-evaluator",
		Namespace: "reference.hierarchy.evaluator",
		Version:   "1.0.0",
		ProvidesInterfaces: []SemanticInterfaceProvider{{
			Interface:           contract.ID,
			ContractVersion:     contract.Version,
			ContractFingerprint: semanticInterfaceFingerprint(contract),
		}},
	}
	conformance := SemanticInterfaceConformance{
		ProviderExtension:   provider.ID,
		Interface:           contract.ID,
		ContractFingerprint: semanticInterfaceFingerprint(contract),
		Profile:             contract.ProviderConformanceRef,
		EvidenceFingerprint: "sha256:" + strings.Repeat("a", 64),
	}
	return contract, hierarchy, provider, conformance
}

func TestSUIHierarchyConstraintUsesSemanticInterfaceInsteadOfCoreValidationKind(t *testing.T) {
	core, target := suiReferenceCoreCatalog(t)
	contract, hierarchy, provider, conformance := suiHierarchySemanticExtensionFixture(target)

	catalog, err := resolveSemanticCatalogWithAllInputs(
		core,
		[]ExtensionDescriptor{provider, hierarchy},
		[]ExtensionRequirement{
			{Extension: hierarchy.ID, CompatibleMajor: 1},
			{Extension: provider.ID, CompatibleMajor: 1},
		},
		nil,
		[]SemanticInterfaceConformance{conformance},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("SUI Semantic Extension fixture unresolved=%+v", catalog.Unresolved)
	}
	if len(catalog.Constraints) != 1 {
		t.Fatalf("constraints=%+v", catalog.Constraints)
	}
	constraint := catalog.Constraints[0]
	if constraint.Subject != target ||
		constraint.EvaluatorInterface != contract.ID ||
		constraint.EvaluatorProviderExtension != provider.ID ||
		constraint.EvaluatorContractFingerprint != semanticInterfaceFingerprint(contract) {
		t.Fatalf("constraint is not bound to exact SUI evaluator contract: %+v", constraint)
	}
	if constraint.ValidatorRef != "reference.hierarchy.integrity.v1" {
		t.Fatalf("validatorRef=%q", constraint.ValidatorRef)
	}

	lock, err := buildSemanticLock(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifySemanticLockReplay(
		lock,
		core,
		[]ExtensionDescriptor{hierarchy, provider},
	); err != nil {
		t.Fatalf("SUI Semantic Set lock did not replay: %v", err)
	}
}

func TestSUIHierarchyConstraintDoesNotActivateWithoutEvaluatorConformance(t *testing.T) {
	core, target := suiReferenceCoreCatalog(t)
	_, hierarchy, provider, _ := suiHierarchySemanticExtensionFixture(target)

	catalog, err := resolveSemanticCatalog(
		core,
		[]ExtensionDescriptor{hierarchy, provider},
		[]ExtensionRequirement{
			{Extension: hierarchy.ID, CompatibleMajor: 1},
			{Extension: provider.ID, CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Constraints) != 0 {
		t.Fatalf("constraint became active without evaluator conformance: %+v", catalog.Constraints)
	}
	if !catalogHasUnresolvedCode(catalog, "interface-provider-conformance-unverified") ||
		!catalogHasUnresolvedCode(catalog, "constraint-evaluator-unresolved") {
		t.Fatalf("missing evaluator dependency was not explicit: %+v", catalog.Unresolved)
	}
	if _, err := buildSemanticLock(catalog); err == nil {
		t.Fatal("unresolved SUI evaluator dependency produced Semantic Lock")
	}
}

func TestConstraintEvaluatorMustBeDeclaredAsSemanticInterfaceRequirement(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.invalid-evaluator-use",
		Namespace: "reference.invalid-evaluator-use",
		Version:   "1.0.0",
		Constraints: []ExtensionConstraint{{
			ID:                 "invalid",
			Subject:            subject.Subject{Kind: "semantic-data", ID: "sui.island"},
			ValidatorRef:       "reference.invalid",
			EvaluatorInterface: "reference.hierarchy.integrity.evaluator",
		}},
	}
	_, err := normalizeDescriptor(descriptor)
	if err == nil || !strings.Contains(err.Error(), "must be declared in requiresInterfaces") {
		t.Fatalf("implicit evaluator dependency was not rejected: %v", err)
	}
}
