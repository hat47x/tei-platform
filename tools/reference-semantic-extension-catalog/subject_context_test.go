package main

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/knowledgecontext"
	"tei/internal/subject"
	"tei/internal/workspace"
)

func knowledgeContextForWorkspace(
	t *testing.T,
	workspaceRoot string,
	target subject.Subject,
) knowledgecontext.Context {
	t.Helper()
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(workspaceRoot, loaded)
	if err != nil {
		t.Fatal(err)
	}
	return knowledgecontext.Build(loaded.World, index, target)
}

func subjectExtensionUseByID(
	values []SubjectExtensionUse,
	id string,
) (SubjectExtensionUse, bool) {
	for _, value := range values {
		if value.Extension.ID == id {
			return value, true
		}
	}
	return SubjectExtensionUse{}, false
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestSubjectSemanticContextConnectsSUIKnowledgeToExtensionValidationRail(t *testing.T) {
	core, target := suiReferenceCoreCatalog(t)
	consumer := acyclicDomainConsumer(t, "tei.reference.sui-acyclic", target)
	catalog, contract, provider, conformance := resolveAcyclicDomainCatalog(
		t,
		core,
		consumer,
	)
	implementation := acyclicEvaluatorImplementation(contract, provider)
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{{
		ConstraintExtension: consumer.ID,
		ConstraintID:        acyclicConstraintID,
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}})
	if err != nil {
		t.Fatal(err)
	}

	root := repositoryRoot(t)
	workspaceRoot := filepath.Join(root, "reference", "sui-adoption", "workspace")
	base := knowledgeContextForWorkspace(t, workspaceRoot, target)
	ctx, err := buildSubjectSemanticContext(base, catalog, &plan, target)
	if err != nil {
		t.Fatal(err)
	}

	if ctx.Schema != subjectSemanticContextSchema ||
		ctx.Subject != target ||
		ctx.SemanticSetFingerprint != catalog.ResolutionFingerprint ||
		ctx.CoreSourceIndexFingerprint != catalog.CoreSourceIndexFingerprint {
		t.Fatalf("subject semantic context identity mismatch: %+v", ctx)
	}
	if ctx.ContextFingerprint == "" ||
		ctx.ContextFingerprint != subjectSemanticContextFingerprint(ctx) {
		t.Fatalf("subject context fingerprint is not reproducible: %q", ctx.ContextFingerprint)
	}
	if len(ctx.Core.Artifacts) == 0 || len(ctx.Core.References) == 0 {
		t.Fatalf("core Knowledge Context was not preserved: %+v", ctx.Core)
	}
	if len(ctx.Constraints) != 1 ||
		ctx.Constraints[0].Extension.ID != consumer.ID ||
		ctx.Constraints[0].EvaluatorInterface != contract.ID ||
		ctx.Constraints[0].EvaluatorProviderExtension != provider.ID {
		t.Fatalf("subject constraint rail=%+v", ctx.Constraints)
	}
	if len(ctx.Interfaces) != 1 ||
		ctx.Interfaces[0].ID != contract.ID ||
		ctx.Interfaces[0].ContractFingerprint != semanticInterfaceFingerprint(contract) {
		t.Fatalf("subject interfaces=%+v", ctx.Interfaces)
	}
	if len(ctx.InterfaceWirings) != 1 ||
		ctx.InterfaceWirings[0].ConsumerExtension != consumer.ID ||
		ctx.InterfaceWirings[0].ProviderExtension != provider.ID {
		t.Fatalf("subject interface wirings=%+v", ctx.InterfaceWirings)
	}

	consumerUse, ok := subjectExtensionUseByID(ctx.Extensions, consumer.ID)
	if !ok {
		t.Fatalf("consumer extension is not discoverable: %+v", ctx.Extensions)
	}
	if !hasString(consumerUse.Roles, "constraint") ||
		!hasString(consumerUse.Roles, "kind-applicable") ||
		consumerUse.Extension.Source == "" ||
		consumerUse.Extension.SourceFingerprint == "" ||
		len(consumerUse.RootPaths) == 0 {
		t.Fatalf("consumer extension lacks re-entry information: %+v", consumerUse)
	}
	providerUse, ok := subjectExtensionUseByID(ctx.Extensions, provider.ID)
	if !ok || !hasString(providerUse.Roles, "evaluator-provider") {
		t.Fatalf("evaluator provider is not discoverable: %+v", ctx.Extensions)
	}

	if ctx.ValidationPlan == nil ||
		ctx.ValidationPlan.SemanticSetFingerprint != catalog.ResolutionFingerprint ||
		ctx.ValidationPlan.PlanFingerprint != plan.PlanFingerprint ||
		len(ctx.ValidationPlan.Bindings) != 1 ||
		ctx.ValidationPlan.Bindings[0].ImplementationID != implementation.ID ||
		ctx.ValidationPlan.Bindings[0].ArtifactDigest != implementation.ArtifactDigest {
		t.Fatalf("subject validation plan=%+v", ctx.ValidationPlan)
	}
	for _, limit := range []string{
		"derived-context-is-not-canonical-authority",
		"extension-selection-is-not-acceptance",
		"kind-applicable-extension-is-candidate-not-acceptance",
		"validation-plan-binding-is-not-runtime-authority",
	} {
		if !hasString(ctx.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q is missing: %+v", limit, ctx.AuthorityLimits)
		}
	}
}

func TestSubjectSemanticContextFindsSameAcyclicRailFromMigrationSubject(t *testing.T) {
	core, target, _ := referenceMigrationAcyclicInput(t)
	consumer := acyclicDomainConsumer(t, "tei.reference.migration-acyclic", target)
	catalog, contract, provider, _ := resolveAcyclicDomainCatalog(
		t,
		core,
		consumer,
	)

	root := repositoryRoot(t)
	workspaceRoot := filepath.Join(root, "reference", "workspace")
	base := knowledgeContextForWorkspace(t, workspaceRoot, target)
	ctx, err := buildSubjectSemanticContext(base, catalog, nil, target)
	if err != nil {
		t.Fatal(err)
	}

	if len(ctx.Constraints) != 1 ||
		ctx.Constraints[0].EvaluatorInterface != contract.ID ||
		ctx.Constraints[0].EvaluatorProviderExtension != provider.ID {
		t.Fatalf("migration semantic rail=%+v", ctx.Constraints)
	}
	consumerUse, ok := subjectExtensionUseByID(ctx.Extensions, consumer.ID)
	if !ok || !hasString(consumerUse.Roles, "constraint") ||
		!hasString(consumerUse.Roles, "kind-applicable") {
		t.Fatalf("migration extension use=%+v", consumerUse)
	}
	if ctx.ValidationPlan != nil {
		t.Fatalf("Validation Plan was invented without accepted bindings: %+v", ctx.ValidationPlan)
	}
}

func TestSubjectSemanticContextKeepsGlobalUnresolvedVisibleWithoutMisattributingIt(t *testing.T) {
	core, target := suiReferenceCoreCatalog(t)
	consumer := acyclicDomainConsumer(t, "tei.reference.sui-acyclic", target)
	catalog, _, _, _ := resolveAcyclicDomainCatalog(t, core, consumer)
	unrelated := subject.Subject{Kind: "semantic-data", ID: "unrelated.item"}
	catalog.Unresolved = append(catalog.Unresolved, CatalogUnresolved{
		Code:    "synthetic-global-pressure",
		Subject: &unrelated,
		Detail:  "unrelated unresolved remains part of Semantic Set health",
	})

	root := repositoryRoot(t)
	workspaceRoot := filepath.Join(root, "reference", "sui-adoption", "workspace")
	base := knowledgeContextForWorkspace(t, workspaceRoot, target)
	ctx, err := buildSubjectSemanticContext(base, catalog, nil, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.SemanticSetUnresolved) == 0 {
		t.Fatal("global Semantic Set unresolved was hidden from subject context")
	}
	for _, unresolved := range ctx.SubjectUnresolved {
		if unresolved.Subject != nil && unresolved.Subject.ID == unrelated.ID {
			t.Fatalf("unrelated unresolved was misattributed to target subject: %+v", unresolved)
		}
	}
	if !strings.Contains(ctx.SemanticSetUnresolved[0].Detail, "Semantic Set health") {
		t.Fatalf("global unresolved detail was lost: %+v", ctx.SemanticSetUnresolved)
	}
}

func TestSubjectSemanticContextSurfacesOnlyRelevantExtensionConnections(t *testing.T) {
	target := subject.Subject{Kind: "semantic-data", ID: "example.item"}
	descriptors := []ExtensionDescriptor{
		{
			Schema:         extensionSchema,
			ID:             "tei.reference.relevant",
			Namespace:      "reference.relevant",
			Version:        "1.0.0",
			AppliesToKinds: []string{"semantic-data"},
			Constraints: []ExtensionConstraint{{
				ID:           "example-rule",
				Subject:      target,
				ValidatorRef: "reference.example.rule",
			}},
			Connections: []ExtensionConnection{{
				Kind: "capability",
				Ref:  "std.example.evaluate",
			}},
		},
		{
			Schema:         extensionSchema,
			ID:             "tei.reference.unrelated",
			Namespace:      "reference.unrelated",
			Version:        "1.0.0",
			AppliesToKinds: []string{"application"},
			Connections: []ExtensionConnection{{
				Kind: "capability",
				Ref:  "std.unrelated.execute",
			}},
		},
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{Subjects: []subject.Subject{target}},
		descriptors,
		[]ExtensionRequirement{
			{Extension: "tei.reference.relevant", CompatibleMajor: 1},
			{Extension: "tei.reference.unrelated", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	base := knowledgecontext.Context{
		Subject:         target,
		AuthorityLimits: []string{"derived-context-is-not-canonical-authority"},
	}
	ctx, err := buildSubjectSemanticContext(base, catalog, nil, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Connections) != 1 ||
		ctx.Connections[0].Extension.ID != "tei.reference.relevant" ||
		ctx.Connections[0].Kind != "capability" ||
		ctx.Connections[0].Ref != "std.example.evaluate" {
		t.Fatalf("subject connections=%+v", ctx.Connections)
	}
	for _, connection := range ctx.Connections {
		if connection.Extension.ID == "tei.reference.unrelated" {
			t.Fatalf("unrelated extension connection leaked into subject context: %+v", connection)
		}
	}
}

func TestSubjectSemanticContextRejectsStaleValidationPlan(t *testing.T) {
	core, target := suiReferenceCoreCatalog(t)
	consumer := acyclicDomainConsumer(t, "tei.reference.sui-acyclic", target)
	catalog, contract, provider, _ := resolveAcyclicDomainCatalog(t, core, consumer)
	implementation := acyclicEvaluatorImplementation(contract, provider)
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{{
		ConstraintExtension: consumer.ID,
		ConstraintID:        acyclicConstraintID,
		ImplementationID:    implementation.ID,
		ArtifactDigest:      implementation.ArtifactDigest,
	}})
	if err != nil {
		t.Fatal(err)
	}
	plan.SemanticSetFingerprint = "sha256:" + strings.Repeat("f", 64)

	root := repositoryRoot(t)
	workspaceRoot := filepath.Join(root, "reference", "sui-adoption", "workspace")
	base := knowledgeContextForWorkspace(t, workspaceRoot, target)
	if _, err := buildSubjectSemanticContext(base, catalog, &plan, target); err == nil {
		t.Fatal("stale Validation Plan was accepted into subject context")
	}
}
