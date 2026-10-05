package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/subject"
	"tei/internal/workspace"
	suihierarchy "tei/tools/reference-sui-adoption/hierarchy"
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


func suiHierarchyEvaluatorImplementation(
	contract SemanticInterfaceContract,
	provider ExtensionDescriptor,
) SemanticEvaluatorImplementation {
	return SemanticEvaluatorImplementation{
		ID:                  "reference.sui-hierarchy-shadow-evaluator/v1",
		ProviderExtension:   provider.ID,
		Interface:           contract.ID,
		ContractFingerprint: semanticInterfaceFingerprint(contract),
		ArtifactDigest:      "sha256:" + strings.Repeat("b", 64),
		Evaluate: func(ctx context.Context, input []byte) (SemanticEvaluationOutcome, error) {
			if err := ctx.Err(); err != nil {
				return SemanticEvaluationOutcome{}, err
			}
			doc, err := suihierarchy.Decode(input)
			if err != nil {
				return SemanticEvaluationOutcome{}, err
			}
			diagnostics := suihierarchy.Validate(doc)
			out := SemanticEvaluationOutcome{Status: SemanticEvaluationPass}
			if len(diagnostics) != 0 {
				out.Status = SemanticEvaluationFail
			}
			for _, diagnostic := range diagnostics {
				message := diagnostic.Message
				if diagnostic.Island != "" {
					message = diagnostic.Island + ": " + message
				}
				out.Diagnostics = append(out.Diagnostics, SemanticEvaluationDiagnostic{
					Code:    diagnostic.Code,
					Message: message,
				})
			}
			return out, nil
		},
	}
}

func resolvedSUIHierarchySemanticCatalog(
	t *testing.T,
) (
	SemanticCatalog,
	SemanticInterfaceContract,
	ExtensionDescriptor,
	ExtensionDescriptor,
	SemanticInterfaceConformance,
) {
	t.Helper()
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
	return catalog, contract, hierarchy, provider, conformance
}

func TestSUIHierarchySemanticEvaluatorMatchesPinnedFixtures(t *testing.T) {
	catalog, contract, hierarchy, provider, conformance := resolvedSUIHierarchySemanticCatalog(t)
	implementation := suiHierarchyEvaluatorImplementation(contract, provider)
	binding := SemanticEvaluatorBinding{
		ConstraintExtension: hierarchy.ID,
		ConstraintID:        "sui-island-hierarchy-integrity",
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{binding})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		fixture string
		status  SemanticEvaluationStatus
		code    string
	}{
		{fixture: "M1-root.valid.json", status: SemanticEvaluationPass},
		{fixture: "M2-three-level.valid.json", status: SemanticEvaluationPass},
		{fixture: "M3-missing-parent.invalid.json", status: SemanticEvaluationFail, code: "SUI_HIERARCHY_PARENT_MISSING"},
		{fixture: "M4-cycle.invalid.json", status: SemanticEvaluationFail, code: "SUI_HIERARCHY_CYCLE"},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "sui-adoption", tc.fixture))
			if err != nil {
				t.Fatal(err)
			}
			evidence, err := evaluateSemanticConstraintWithPlan(
				context.Background(),
				catalog,
				plan,
				hierarchy.ID,
				"sui-island-hierarchy-integrity",
				implementation,
				[]SemanticInterfaceConformance{conformance},
				input,
			)
			if err != nil {
				t.Fatal(err)
			}
			if evidence.Status != tc.status {
				t.Fatalf("status=%q want=%q evidence=%+v", evidence.Status, tc.status, evidence)
			}
			if evidence.SemanticSetFingerprint != catalog.ResolutionFingerprint ||
				evidence.EvaluatorImplementationID != implementation.ID ||
				evidence.EvaluatorArtifactDigest != implementation.ArtifactDigest ||
				evidence.ConformanceEvidenceFingerprint != conformance.EvidenceFingerprint ||
				!validSHA256Fingerprint(evidence.InputFingerprint) ||
				!validSHA256Fingerprint(evidence.EvidenceFingerprint) {
				t.Fatalf("Evaluation Evidence is incomplete: %+v", evidence)
			}
			if tc.code != "" && !semanticEvaluationHasDiagnostic(evidence.Diagnostics, tc.code) {
				t.Fatalf("diagnostics=%+v want=%s", evidence.Diagnostics, tc.code)
			}
		})
	}
}

func TestSemanticEvaluatorCannotBeReplacedByEnvironmentLikeAlternative(t *testing.T) {
	catalog, contract, hierarchy, provider, conformance := resolvedSUIHierarchySemanticCatalog(t)
	accepted := suiHierarchyEvaluatorImplementation(contract, provider)
	binding := SemanticEvaluatorBinding{
		ConstraintExtension: hierarchy.ID,
		ConstraintID:        "sui-island-hierarchy-integrity",
		ImplementationID:    accepted.ID,
		ArtifactDigest:      accepted.ArtifactDigest,
	}
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "sui-adoption", "M2-three-level.valid.json"))
	if err != nil {
		t.Fatal(err)
	}

	alternative := accepted
	alternative.ID = "environment.selected.other-evaluator/v9"
	alternative.ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
	if _, err := evaluateSemanticConstraintWithPlan(
		context.Background(),
		catalog,
		plan,
		hierarchy.ID,
		"sui-island-hierarchy-integrity",
		alternative,
		[]SemanticInterfaceConformance{conformance},
		input,
	); err == nil || !strings.Contains(err.Error(), "explicit Validation binding") {
		t.Fatalf("environment-like evaluator replacement was accepted: %v", err)
	}
}

func TestSemanticEvaluatorRequiresCurrentConformanceAndDoesNotFailOpen(t *testing.T) {
	catalog, contract, hierarchy, provider, _ := resolvedSUIHierarchySemanticCatalog(t)
	implementation := suiHierarchyEvaluatorImplementation(contract, provider)
	binding := SemanticEvaluatorBinding{
		ConstraintExtension: hierarchy.ID,
		ConstraintID:        "sui-island-hierarchy-integrity",
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "sui-adoption", "M1-root.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	if evidence, err := evaluateSemanticConstraintWithPlan(
		context.Background(),
		catalog,
		plan,
		hierarchy.ID,
		"sui-island-hierarchy-integrity",
		implementation,
		nil,
		input,
	); err == nil || evidence.Status == SemanticEvaluationPass {
		t.Fatalf("missing Conformance Evidence failed open: evidence=%+v err=%v", evidence, err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if evidence, err := evaluateSemanticConstraintWithPlan(
		cancelled,
		catalog,
		plan,
		hierarchy.ID,
		"sui-island-hierarchy-integrity",
		implementation,
		[]SemanticInterfaceConformance{semanticInterfaceConformanceFor(contract, provider.ID, "d")},
		input,
	); err == nil || evidence.Status == SemanticEvaluationPass {
		t.Fatalf("cancelled evaluation failed open: evidence=%+v err=%v", evidence, err)
	}
}

func TestSemanticEvaluatorPreservesUnknownAsDistinctOutcome(t *testing.T) {
	catalog, contract, hierarchy, provider, conformance := resolvedSUIHierarchySemanticCatalog(t)
	implementation := suiHierarchyEvaluatorImplementation(contract, provider)
	implementation.Evaluate = func(context.Context, []byte) (SemanticEvaluationOutcome, error) {
		return SemanticEvaluationOutcome{
			Status: SemanticEvaluationUnknown,
			Diagnostics: []SemanticEvaluationDiagnostic{{
				Code:    "REFERENCE_INPUT_INCOMPLETE",
				Message: "hierarchy snapshot is incomplete",
			}},
		}, nil
	}
	binding := SemanticEvaluatorBinding{
		ConstraintExtension: hierarchy.ID,
		ConstraintID:        "sui-island-hierarchy-integrity",
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := evaluateSemanticConstraintWithPlan(
		context.Background(),
		catalog,
		plan,
		hierarchy.ID,
		"sui-island-hierarchy-integrity",
		implementation,
		[]SemanticInterfaceConformance{conformance},
		[]byte("{}"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != SemanticEvaluationUnknown {
		t.Fatalf("unknown was collapsed: %+v", evidence)
	}
}

func semanticEvaluationHasDiagnostic(values []SemanticEvaluationDiagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}


func TestSemanticValidationPlanRejectsMissingBindingAndSemanticSetDrift(t *testing.T) {
	catalog, contract, hierarchy, provider, _ := resolvedSUIHierarchySemanticCatalog(t)
	implementation := suiHierarchyEvaluatorImplementation(contract, provider)
	if _, err := buildSemanticValidationPlan(catalog, nil); err == nil ||
		!strings.Contains(err.Error(), "missing evaluator binding") {
		t.Fatalf("incomplete Validation Plan was accepted: %v", err)
	}

	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{{
		ConstraintExtension: hierarchy.ID,
		ConstraintID:        "sui-island-hierarchy-integrity",
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}})
	if err != nil {
		t.Fatal(err)
	}
	drifted := catalog
	drifted.ResolutionFingerprint = "sha256:" + strings.Repeat("e", 64)
	if err := verifySemanticValidationPlan(plan, drifted); err == nil ||
		!strings.Contains(err.Error(), "bound to Semantic Set") {
		t.Fatalf("Semantic Set drift did not invalidate Validation Plan: %v", err)
	}
}

func TestSemanticValidationPlanIsStableAcrossConformanceEvidenceRefresh(t *testing.T) {
	catalog, contract, hierarchy, provider, _ := resolvedSUIHierarchySemanticCatalog(t)
	implementation := suiHierarchyEvaluatorImplementation(contract, provider)
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{{
		ConstraintExtension: hierarchy.ID,
		ConstraintID:        "sui-island-hierarchy-integrity",
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}})
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "sui-adoption", "M1-root.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	left, err := evaluateSemanticConstraintWithPlan(
		context.Background(),
		catalog,
		plan,
		hierarchy.ID,
		"sui-island-hierarchy-integrity",
		implementation,
		[]SemanticInterfaceConformance{semanticInterfaceConformanceFor(contract, provider.ID, "1")},
		input,
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := evaluateSemanticConstraintWithPlan(
		context.Background(),
		catalog,
		plan,
		hierarchy.ID,
		"sui-island-hierarchy-integrity",
		implementation,
		[]SemanticInterfaceConformance{semanticInterfaceConformanceFor(contract, provider.ID, "2")},
		input,
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlanFingerprint == "" || !validSHA256Fingerprint(plan.PlanFingerprint) {
		t.Fatalf("Validation Plan fingerprint is invalid: %+v", plan)
	}
	if left.ConformanceEvidenceFingerprint == right.ConformanceEvidenceFingerprint ||
		left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatalf("Conformance Evidence refresh was not captured: left=%+v right=%+v", left, right)
	}
	if left.SemanticSetFingerprint != right.SemanticSetFingerprint ||
		left.SemanticSetFingerprint != plan.SemanticSetFingerprint {
		t.Fatalf("Evidence refresh changed semantic binding: left=%+v right=%+v plan=%+v", left, right, plan)
	}
}
