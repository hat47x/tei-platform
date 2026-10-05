package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/subject"
	"tei/internal/workspace"
	"tei/tools/reference-semantic-extension-catalog/acyclicgraph"
	suihierarchy "tei/tools/reference-sui-adoption/hierarchy"
)

const (
	acyclicInterfaceID = "reference.graph.acyclic.evaluate"
	acyclicConstraintID = "acyclic"
)

func acyclicSemanticInterfaceFixture() (
	SemanticInterfaceContract,
	ExtensionDescriptor,
	ExtensionDescriptor,
	SemanticInterfaceConformance,
) {
	contract := SemanticInterfaceContract{
		ID:                     acyclicInterfaceID,
		Version:                "1.0.0",
		InputSemantics:         []string{"normalized directed relation graph"},
		OutputSemantics:        []string{"acyclic pass fail unknown diagnostics"},
		UnknownSemantics:       "incomplete normalized graph remains unknown",
		FailureSemantics:       "directed cycle is blocking",
		ProviderConformanceRef: "reference.conformance.acyclic-graph-v1",
	}
	owner := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.acyclic-contract",
		Namespace: "reference.graph.acyclic",
		Version:   "1.0.0",
		Interfaces: []SemanticInterfaceContract{contract},
	}
	provider := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.acyclic-evaluator",
		Namespace: "reference.graph.evaluator",
		Version:   "1.0.0",
		ProvidesInterfaces: []SemanticInterfaceProvider{{
			Interface:           contract.ID,
			ContractVersion:     contract.Version,
			ContractFingerprint: semanticInterfaceFingerprint(contract),
		}},
	}
	conformance := semanticInterfaceConformanceFor(contract, provider.ID, "7")
	return contract, owner, provider, conformance
}

func acyclicDomainConsumer(
	id string,
	namespace string,
	target subject.Subject,
) ExtensionDescriptor {
	return ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        id,
		Namespace: namespace,
		Version:   "1.0.0",
		RequiresInterfaces: []SemanticInterfaceRequirement{{
			Interface:       acyclicInterfaceID,
			CompatibleMajor: 1,
		}},
		Constraints: []ExtensionConstraint{{
			ID:                 acyclicConstraintID,
			Subject:            target,
			ValidatorRef:       "reference.graph.acyclic.v1",
			EvaluatorInterface: acyclicInterfaceID,
		}},
	}
}

func resolveAcyclicDomainCatalog(
	t *testing.T,
	core CoreCatalogInput,
	consumer ExtensionDescriptor,
) (
	SemanticCatalog,
	SemanticInterfaceContract,
	ExtensionDescriptor,
	SemanticInterfaceConformance,
) {
	t.Helper()
	contract, owner, provider, conformance := acyclicSemanticInterfaceFixture()
	catalog, err := resolveSemanticCatalogWithAllInputs(
		core,
		[]ExtensionDescriptor{consumer, provider, owner},
		semanticInterfaceRoots(consumer.ID, provider.ID, owner.ID),
		nil,
		[]SemanticInterfaceConformance{conformance},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("acyclic domain catalog unresolved=%+v", catalog.Unresolved)
	}
	if len(catalog.Constraints) != 1 {
		t.Fatalf("acyclic domain constraints=%+v", catalog.Constraints)
	}
	constraint := catalog.Constraints[0]
	if constraint.EvaluatorInterface != contract.ID ||
		constraint.EvaluatorProviderExtension != provider.ID ||
		constraint.EvaluatorContractFingerprint != semanticInterfaceFingerprint(contract) {
		t.Fatalf("acyclic constraint is not bound to generic contract: %+v", constraint)
	}
	return catalog, contract, provider, conformance
}

func acyclicEvaluatorImplementation(
	contract SemanticInterfaceContract,
	provider ExtensionDescriptor,
) SemanticEvaluatorImplementation {
	return SemanticEvaluatorImplementation{
		ID:                  "reference.graph.acyclic-evaluator/v1",
		ProviderExtension:   provider.ID,
		Interface:           contract.ID,
		ContractFingerprint: semanticInterfaceFingerprint(contract),
		ArtifactDigest:      "sha256:" + strings.Repeat("8", 64),
		Evaluate: func(ctx context.Context, input []byte) (SemanticEvaluationOutcome, error) {
			if err := ctx.Err(); err != nil {
				return SemanticEvaluationOutcome{}, err
			}
			var snapshot acyclicgraph.Snapshot
			if err := json.Unmarshal(input, &snapshot); err != nil {
				return SemanticEvaluationOutcome{}, err
			}
			diagnostics, err := acyclicgraph.ValidateAcyclic(snapshot)
			if err != nil {
				return SemanticEvaluationOutcome{}, err
			}
			out := SemanticEvaluationOutcome{Status: SemanticEvaluationPass}
			if len(diagnostics) != 0 {
				out.Status = SemanticEvaluationFail
			}
			for _, diagnostic := range diagnostics {
				message := diagnostic.Message
				if diagnostic.Node != "" {
					message = diagnostic.Node + ": " + message
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

func evaluateAcyclicSnapshot(
	t *testing.T,
	catalog SemanticCatalog,
	consumer ExtensionDescriptor,
	contract SemanticInterfaceContract,
	provider ExtensionDescriptor,
	conformance SemanticInterfaceConformance,
	snapshot acyclicgraph.Snapshot,
) SemanticEvaluationEvidence {
	t.Helper()
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
	input, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := evaluateSemanticConstraintWithPlan(
		context.Background(),
		catalog,
		plan,
		consumer.ID,
		acyclicConstraintID,
		implementation,
		[]SemanticInterfaceConformance{conformance},
		input,
	)
	if err != nil {
		t.Fatal(err)
	}
	return evidence
}

func suiAcyclicSnapshot(doc suihierarchy.Document) acyclicgraph.Snapshot {
	nodes := make([]string, 0, len(doc.Islands))
	exists := map[string]struct{}{}
	for _, island := range doc.Islands {
		nodes = append(nodes, island.ID)
		exists[island.ID] = struct{}{}
	}
	var edges []acyclicgraph.Edge
	for _, island := range doc.Islands {
		if island.ParentIslandID == nil {
			continue
		}
		if _, ok := exists[*island.ParentIslandID]; !ok {
			continue
		}
		edges = append(edges, acyclicgraph.Edge{
			From: island.ID,
			To:   *island.ParentIslandID,
		})
	}
	return acyclicgraph.Snapshot{Nodes: nodes, Edges: edges}
}

func referenceMigrationAcyclicInput(
	t *testing.T,
) (CoreCatalogInput, subject.Subject, []migration.Definition) {
	t.Helper()
	root := repositoryRoot(t)
	workspaceRoot := filepath.Join(root, "reference", "workspace")
	loaded, err := workspace.Load(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(workspaceRoot, loaded)
	if err != nil {
		t.Fatal(err)
	}
	core := coreCatalogFromSourceIndex(index)

	target := subject.Subject{Kind: "migration", ID: "sales_core/20260826_001"}
	var found bool
	for _, item := range core.Subjects {
		if item == target {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Reference migration subject %+v was not found: %+v", target, core.Subjects)
	}
	var definitions []migration.Definition
	for _, definition := range loaded.World.Migrations {
		if string(definition.Database) == "sales_core" {
			definitions = append(definitions, definition)
		}
	}
	if len(definitions) == 0 {
		t.Fatal("Reference Workspace has no sales_core migration definitions")
	}
	return core, target, definitions
}

func migrationAcyclicSnapshot(definitions []migration.Definition) acyclicgraph.Snapshot {
	nodes := make([]string, 0, len(definitions))
	exists := map[string]struct{}{}
	for _, definition := range definitions {
		nodes = append(nodes, definition.ID)
		exists[definition.ID] = struct{}{}
	}
	var edges []acyclicgraph.Edge
	for _, definition := range definitions {
		for _, parent := range definition.Parents {
			if _, ok := exists[parent]; !ok {
				continue
			}
			edges = append(edges, acyclicgraph.Edge{
				From: definition.ID,
				To:   parent,
			})
		}
	}
	return acyclicgraph.Snapshot{Nodes: nodes, Edges: edges}
}

func TestAcyclicSemanticInterfaceReusesExactContractAcrossSUIAndMigration(t *testing.T) {
	suiCore, suiTarget := suiReferenceCoreCatalog(t)
	suiConsumer := acyclicDomainConsumer(
		"tei.reference.sui-acyclic",
		"reference.sui.acyclic",
		suiTarget,
	)
	suiCatalog, suiContract, suiProvider, _ := resolveAcyclicDomainCatalog(
		t,
		suiCore,
		suiConsumer,
	)

	migrationCore, migrationTarget, _ := referenceMigrationAcyclicInput(t)
	migrationConsumer := acyclicDomainConsumer(
		"tei.reference.migration-acyclic",
		"reference.migration.acyclic",
		migrationTarget,
	)
	migrationCatalog, migrationContract, migrationProvider, _ := resolveAcyclicDomainCatalog(
		t,
		migrationCore,
		migrationConsumer,
	)

	if semanticInterfaceFingerprint(suiContract) != semanticInterfaceFingerprint(migrationContract) {
		t.Fatalf("second consumer required a different acyclic contract: SUI=%s migration=%s",
			semanticInterfaceFingerprint(suiContract),
			semanticInterfaceFingerprint(migrationContract),
		)
	}
	if suiProvider.ID != migrationProvider.ID ||
		suiProvider.Version != migrationProvider.Version {
		t.Fatalf("second consumer required a different provider identity: SUI=%s@%s migration=%s@%s",
			suiProvider.ID, suiProvider.Version,
			migrationProvider.ID, migrationProvider.Version,
		)
	}
	if got := suiCatalog.InterfaceWirings[0].ProviderExtension; got != suiProvider.ID {
		t.Fatalf("SUI wiring provider=%q want=%q", got, suiProvider.ID)
	}
	if got := migrationCatalog.InterfaceWirings[0].ProviderExtension; got != migrationProvider.ID {
		t.Fatalf("migration wiring provider=%q want=%q", got, migrationProvider.ID)
	}
}

func TestAcyclicSemanticEvaluatorHandlesIndependentSUIAndMigrationDomains(t *testing.T) {
	suiCore, suiTarget := suiReferenceCoreCatalog(t)
	suiConsumer := acyclicDomainConsumer(
		"tei.reference.sui-acyclic",
		"reference.sui.acyclic",
		suiTarget,
	)
	suiCatalog, contract, provider, conformance := resolveAcyclicDomainCatalog(
		t,
		suiCore,
		suiConsumer,
	)

	validSUI, err := suihierarchy.ReadFile(filepath.Join(
		repositoryRoot(t),
		"reference",
		"sui-adoption",
		"M2-three-level.valid.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	cyclicSUI, err := suihierarchy.ReadFile(filepath.Join(
		repositoryRoot(t),
		"reference",
		"sui-adoption",
		"M4-cycle.invalid.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	if evidence := evaluateAcyclicSnapshot(
		t, suiCatalog, suiConsumer, contract, provider, conformance, suiAcyclicSnapshot(validSUI),
	); evidence.Status != SemanticEvaluationPass {
		t.Fatalf("valid SUI hierarchy status=%q diagnostics=%+v", evidence.Status, evidence.Diagnostics)
	}
	if evidence := evaluateAcyclicSnapshot(
		t, suiCatalog, suiConsumer, contract, provider, conformance, suiAcyclicSnapshot(cyclicSUI),
	); evidence.Status != SemanticEvaluationFail ||
		!semanticEvaluationHasDiagnostic(evidence.Diagnostics, "DIRECTED_GRAPH_CYCLE") {
		t.Fatalf("cyclic SUI hierarchy evidence=%+v", evidence)
	}

	migrationCore, migrationTarget, definitions := referenceMigrationAcyclicInput(t)
	migrationConsumer := acyclicDomainConsumer(
		"tei.reference.migration-acyclic",
		"reference.migration.acyclic",
		migrationTarget,
	)
	migrationCatalog, migrationContract, migrationProvider, migrationConformance :=
		resolveAcyclicDomainCatalog(t, migrationCore, migrationConsumer)
	if evidence := evaluateAcyclicSnapshot(
		t,
		migrationCatalog,
		migrationConsumer,
		migrationContract,
		migrationProvider,
		migrationConformance,
		migrationAcyclicSnapshot(definitions),
	); evidence.Status != SemanticEvaluationPass {
		t.Fatalf("Reference migration graph status=%q diagnostics=%+v", evidence.Status, evidence.Diagnostics)
	}

	cyclicMigrations := []migration.Definition{
		{ID: "cycle-a", Database: "sales_core", Parents: []string{"cycle-b"}},
		{ID: "cycle-b", Database: "sales_core", Parents: []string{"cycle-a"}},
	}
	if evidence := evaluateAcyclicSnapshot(
		t,
		migrationCatalog,
		migrationConsumer,
		migrationContract,
		migrationProvider,
		migrationConformance,
		migrationAcyclicSnapshot(cyclicMigrations),
	); evidence.Status != SemanticEvaluationFail ||
		!semanticEvaluationHasDiagnostic(evidence.Diagnostics, "DIRECTED_GRAPH_CYCLE") {
		t.Fatalf("cyclic migration graph evidence=%+v", evidence)
	}
}

func TestAcyclicConsumersShareEvaluatorWithoutCollapsingSemanticSets(t *testing.T) {
	suiCore, suiTarget := suiReferenceCoreCatalog(t)
	suiConsumer := acyclicDomainConsumer(
		"tei.reference.sui-acyclic",
		"reference.sui.acyclic",
		suiTarget,
	)
	suiCatalog, contract, provider, _ := resolveAcyclicDomainCatalog(
		t,
		suiCore,
		suiConsumer,
	)

	migrationCore, migrationTarget, _ := referenceMigrationAcyclicInput(t)
	migrationConsumer := acyclicDomainConsumer(
		"tei.reference.migration-acyclic",
		"reference.migration.acyclic",
		migrationTarget,
	)
	migrationCatalog, migrationContract, migrationProvider, _ :=
		resolveAcyclicDomainCatalog(t, migrationCore, migrationConsumer)

	if semanticInterfaceFingerprint(contract) != semanticInterfaceFingerprint(migrationContract) {
		t.Fatalf("consumers did not share the same Interface contract")
	}
	if provider.ID != migrationProvider.ID || provider.Version != migrationProvider.Version {
		t.Fatalf("consumers did not share the same evaluator provider")
	}
	if suiCatalog.ResolutionFingerprint == migrationCatalog.ResolutionFingerprint {
		t.Fatalf(
			"independent domain consumers collapsed into one Semantic Set: %s",
			suiCatalog.ResolutionFingerprint,
		)
	}
}

func TestAcyclicSecondConsumerSharesOneEvaluatorArtifactIdentity(t *testing.T) {
	contract, _, provider, _ := acyclicSemanticInterfaceFixture()
	implementation := acyclicEvaluatorImplementation(contract, provider)

	suiConsumer := acyclicDomainConsumer(
		"tei.reference.sui-acyclic",
		"reference.sui.acyclic",
		subject.Subject{Kind: "semantic-data", ID: "sui.island"},
	)
	migrationConsumer := acyclicDomainConsumer(
		"tei.reference.migration-acyclic",
		"reference.migration.acyclic",
		subject.Subject{Kind: "migration", ID: "sales_core/20260826_001"},
	)

	if suiConsumer.ID == migrationConsumer.ID {
		t.Fatal("domain bindings unexpectedly share logical consumer identity")
	}
	if implementation.ArtifactDigest == "" ||
		implementation.ID != "reference.graph.acyclic-evaluator/v1" {
		t.Fatalf("generic evaluator implementation=%+v", implementation)
	}
	if semanticInterfaceFingerprint(contract) == "" {
		t.Fatal("generic contract fingerprint is empty")
	}
}

func TestAcyclicAdaptersDoNotLeakSUIVocabularyIntoSharedGraph(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		repositoryRoot(t),
		"reference",
		"sui-adoption",
		"M2-three-level.valid.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := suihierarchy.Decode(content)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(suiAcyclicSnapshot(doc))
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"islands", "parentIslandId", "childIslandIds", "schemaVersion"} {
		if strings.Contains(string(encoded), leaked) {
			t.Fatalf("shared graph input leaked SUI vocabulary %q: %s", leaked, encoded)
		}
	}
}
