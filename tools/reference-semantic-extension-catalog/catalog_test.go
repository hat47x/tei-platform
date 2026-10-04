package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/internal/subject"
	"tei/internal/workspace"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source pathを取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func referenceCoreCatalog(t *testing.T) CoreCatalogInput {
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
	return coreCatalogFromSourceIndex(index)
}

func referenceDescriptors(t *testing.T) []ExtensionDescriptor {
	t.Helper()
	descriptors, err := loadExtensionDescriptors(filepath.Join(repositoryRoot(t), "tools", "reference-semantic-extension-catalog", "testdata", "extensions"))
	if err != nil {
		t.Fatal(err)
	}
	return descriptors
}

func TestReferenceCatalogCombinesCoreAndOpenEndedExtensionSubjects(t *testing.T) {
	catalog, err := resolveSemanticCatalog(
		referenceCoreCatalog(t),
		referenceDescriptors(t),
		[]ExtensionRequirement{{Extension: "tei.reference.service-level-policy", CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("unexpected unresolved: %+v", catalog.Unresolved)
	}
	for _, want := range []subject.Subject{
		{Kind: "semantic-data", ID: "repair_intake.intake"},
		{Kind: "service-level-policy", ID: "support.case.standard-response"},
		{Kind: "temporal-window", ID: "support.window.business-hours"},
	} {
		if !catalogHasSubject(catalog, want) {
			t.Fatalf("catalog does not contain %+v", want)
		}
	}
	coreEntry, ok := catalogSubject(catalog, subject.Subject{Kind: "semantic-data", ID: "repair_intake.intake"})
	if !ok || !catalogSourcesContain(coreEntry.Sources, "semantic/repair-intake/intake.tei") {
		t.Fatalf("core subject source Evidence is missing: %+v", coreEntry)
	}
	extensionEntry, ok := catalogSubject(catalog, subject.Subject{Kind: "service-level-policy", ID: "support.case.standard-response"})
	if !ok || !catalogSourcesContain(extensionEntry.Sources, "service-level-1.1.0.json") {
		t.Fatalf("extension subject source Evidence is missing: %+v", extensionEntry)
	}

	if !catalogHasRelation(
		catalog,
		subject.Subject{Kind: "service-level-policy", ID: "support.case.standard-response"},
		"governs",
		subject.Subject{Kind: "semantic-data", ID: "repair_intake.intake"},
	) {
		t.Fatalf("extension-to-core relation is missing: %+v", catalog.Relations)
	}
	if !catalogHasRelation(
		catalog,
		subject.Subject{Kind: "service-level-policy", ID: "support.case.standard-response"},
		"uses-window",
		subject.Subject{Kind: "temporal-window", ID: "support.window.business-hours"},
	) {
		t.Fatalf("extension-to-extension relation is missing: %+v", catalog.Relations)
	}
	if len(catalog.Extensions) != 2 ||
		catalog.Extensions[0].ID != "tei.reference.service-level-policy" ||
		catalog.Extensions[1].ID != "tei.reference.temporal-semantics" {
		t.Fatalf("dependency closure is not deterministic: %+v", catalog.Extensions)
	}
	serviceLevel := catalog.Extensions[0]
	if serviceLevel.Namespace != "support.service-level" ||
		!strings.HasPrefix(serviceLevel.DescriptorFingerprint, "sha256:") ||
		serviceLevel.Source != "service-level-1.1.0.json" ||
		!strings.HasPrefix(serviceLevel.SourceFingerprint, "sha256:") ||
		!reflect.DeepEqual(serviceLevel.AppliesToKinds, []string{"application-pattern", "semantic-data"}) ||
		len(serviceLevel.Requires) != 1 ||
		serviceLevel.Requires[0].Extension != "tei.reference.temporal-semantics" ||
		serviceLevel.Requires[0].CompatibleMajor != 1 {
		t.Fatalf("extension discovery metadata is incomplete: %+v", serviceLevel)
	}
}

func TestReferenceCatalogSelectsHighestCompatibleVersionDeterministically(t *testing.T) {
	makeDescriptor := func(version string) ExtensionDescriptor {
		return ExtensionDescriptor{
			Schema:    extensionSchema,
			ID:        "tei.reference.policy",
			Namespace: "policy",
			Version:   version,
			Concepts: []ExtensionConcept{{
				Subject: subject.Subject{Kind: "policy-view", ID: "policy.example"},
			}},
		}
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{makeDescriptor("1.0.0"), makeDescriptor("1.2.0"), makeDescriptor("2.0.0")},
		[]ExtensionRequirement{{Extension: "tei.reference.policy", CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Extensions) != 1 || catalog.Extensions[0].Version != "1.2.0" {
		t.Fatalf("resolved extensions=%+v", catalog.Extensions)
	}
}

func TestReferenceCatalogBacktracksFromGreedyFalseConflict(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.x",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.y",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.x",
			Namespace: "x",
			Version:   "1.2.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 2,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.x",
			Namespace: "x",
			Version:   "1.1.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.y",
			Namespace: "y",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.z",
			Namespace: "z",
			Version:   "2.0.0",
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.z",
			Namespace: "z",
			Version:   "1.0.0",
		},
	}

	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{
			{Extension: "tei.reference.a", CompatibleMajor: 1},
			{Extension: "tei.reference.b", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	versions := map[string]string{}
	for _, extension := range catalog.Extensions {
		versions[extension.ID] = extension.Version
	}
	if got := versions["tei.reference.x"]; got != "1.1.0" {
		t.Fatalf("resolver kept greedy x version %q; want 1.1.0", got)
	}
	if got := versions["tei.reference.z"]; got != "1.0.0" {
		t.Fatalf("resolver kept incompatible z version %q; want 1.0.0", got)
	}
}

func TestReferenceCatalogBacktrackingIsIndependentOfInputOrder(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.x",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.y",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.x",
			Namespace: "x",
			Version:   "1.2.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 2,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.x",
			Namespace: "x",
			Version:   "1.1.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.y",
			Namespace: "y",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.z",
			Namespace: "z",
			Version:   "2.0.0",
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.z",
			Namespace: "z",
			Version:   "1.0.0",
		},
	}
	roots := []ExtensionRequirement{
		{Extension: "tei.reference.a", CompatibleMajor: 1},
		{Extension: "tei.reference.b", CompatibleMajor: 1},
	}
	left, err := resolveSemanticCatalog(CoreCatalogInput{}, descriptors, roots)
	if err != nil {
		t.Fatal(err)
	}

	reversedDescriptors := append([]ExtensionDescriptor(nil), descriptors...)
	for i, j := 0, len(reversedDescriptors)-1; i < j; i, j = i+1, j-1 {
		reversedDescriptors[i], reversedDescriptors[j] = reversedDescriptors[j], reversedDescriptors[i]
	}
	reversedRoots := []ExtensionRequirement{roots[1], roots[0]}
	right, err := resolveSemanticCatalog(CoreCatalogInput{}, reversedDescriptors, reversedRoots)
	if err != nil {
		t.Fatal(err)
	}

	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf(
			"backtracking depends on input order: %s != %s",
			left.ResolutionFingerprint,
			right.ResolutionFingerprint,
		)
	}
	if !reflect.DeepEqual(left.Extensions, right.Extensions) {
		t.Fatalf("resolved extension set depends on input order: left=%+v right=%+v", left.Extensions, right.Extensions)
	}
}

func TestReferenceCatalogBacktracksAroundVersionSpecificDependencyCycle(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.1.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.b",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.a",
				CompatibleMajor: 1,
			}},
		},
	}

	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{{
			Extension:       "tei.reference.a",
			CompatibleMajor: 1,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Extensions) != 1 ||
		catalog.Extensions[0].ID != "tei.reference.a" ||
		catalog.Extensions[0].Version != "1.0.0" {
		t.Fatalf("resolver did not backtrack away from dependency cycle: %+v", catalog.Extensions)
	}
}

func TestReferenceCatalogRejectsDependencyCycleWhenNoAlternativeExists(t *testing.T) {
	a := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.a",
		Namespace: "a",
		Version:   "1.0.0",
		Requires: []ExtensionRequirement{{
			Extension:       "tei.reference.b",
			CompatibleMajor: 1,
		}},
	}
	b := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.b",
		Namespace: "b",
		Version:   "1.0.0",
		Requires: []ExtensionRequirement{{
			Extension:       "tei.reference.a",
			CompatibleMajor: 1,
		}},
	}

	_, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{a, b},
		[]ExtensionRequirement{{
			Extension:       "tei.reference.a",
			CompatibleMajor: 1,
		}},
	)
	if err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("dependency cycle was not rejected: %v", err)
	}
}

func TestReferenceCatalogRejectsIncompatibleDependency(t *testing.T) {
	root := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.root",
		Namespace: "root",
		Version:   "1.0.0",
		Requires: []ExtensionRequirement{{
			Extension:       "tei.reference.dependency",
			CompatibleMajor: 2,
		}},
	}
	dependency := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.dependency",
		Namespace: "dependency",
		Version:   "1.0.0",
	}
	_, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{root, dependency},
		[]ExtensionRequirement{{Extension: "tei.reference.root", CompatibleMajor: 1}},
	)
	if err == nil || !strings.Contains(err.Error(), "compatible with major 2") {
		t.Fatalf("incompatible dependency was not rejected: %v", err)
	}
}

func TestReferenceCatalogReportsSubjectConflictWithoutLoadOrderWinner(t *testing.T) {
	core := subject.Subject{Kind: "semantic-data", ID: "repair_intake.intake"}
	extension := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.conflict",
		Namespace: "conflict",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: core,
		}},
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{Subjects: []subject.Subject{core}},
		[]ExtensionDescriptor{extension},
		[]ExtensionRequirement{{Extension: "tei.reference.conflict", CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if catalogHasSubject(catalog, core) {
		t.Fatalf("conflicting subject was silently selected: %+v", catalog.Subjects)
	}
	if !catalogHasUnresolved(catalog, "subject-conflict", core) {
		t.Fatalf("subject conflict is missing: %+v", catalog.Unresolved)
	}
}

func TestReferenceCatalogProtectsCoreNamespaceOwnership(t *testing.T) {
	coreNamespace := subject.Subject{Kind: "semantic", ID: "shared.policy"}
	extensionConcept := subject.Subject{Kind: "policy-view", ID: "policy.extension"}
	extension := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.extension",
		Namespace: "shared.policy",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: extensionConcept,
		}},
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{Subjects: []subject.Subject{coreNamespace}},
		[]ExtensionDescriptor{extension},
		[]ExtensionRequirement{{Extension: "tei.reference.extension", CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !catalogHasSubject(catalog, coreNamespace) {
		t.Fatalf("core namespace disappeared: %+v", catalog.Subjects)
	}
	if catalogHasSubject(catalog, extensionConcept) {
		t.Fatalf("extension silently took ownership of a core namespace: %+v", catalog.Subjects)
	}
	var found bool
	for _, unresolved := range catalog.Unresolved {
		if unresolved.Code == "namespace-ownership-conflict" &&
			strings.Contains(unresolved.Detail, "core") &&
			strings.Contains(unresolved.Detail, "tei.reference.extension") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("core namespace ownership conflict is missing: %+v", catalog.Unresolved)
	}
}

func TestReferenceCatalogKeepsSharedNamespaceOwnershipUnresolved(t *testing.T) {
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
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{right, left},
		[]ExtensionRequirement{
			{Extension: "tei.reference.left", CompatibleMajor: 1},
			{Extension: "tei.reference.right", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if catalogHasSubject(catalog, left.Concepts[0].Subject) ||
		catalogHasSubject(catalog, right.Concepts[0].Subject) {
		t.Fatalf("ambiguous namespace contributions were published: %+v", catalog.Subjects)
	}
	var found bool
	for _, unresolved := range catalog.Unresolved {
		if unresolved.Code == "namespace-ownership-conflict" &&
			strings.Contains(unresolved.Detail, "shared.policy") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("namespace ownership conflict is missing: %+v", catalog.Unresolved)
	}
}

func TestReferenceCatalogKeepsUnknownRelationTargetUnresolved(t *testing.T) {
	from := subject.Subject{Kind: "policy-view", ID: "policy.example"}
	missing := subject.Subject{Kind: "semantic-data", ID: "missing.data"}
	extension := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.policy",
		Namespace: "policy",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: from,
		}},
		Relations: []ExtensionRelation{{
			From: from, Relation: "governs", To: missing,
		}},
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{extension},
		[]ExtensionRequirement{{Extension: "tei.reference.policy", CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Relations) != 0 {
		t.Fatalf("relation with unknown target was published: %+v", catalog.Relations)
	}
	if !catalogHasUnresolved(catalog, "relation-target-unavailable", missing) {
		t.Fatalf("unknown relation target is not unresolved: %+v", catalog.Unresolved)
	}
}

func TestReferenceCatalogResolutionPolicyChoosesCompleteSolutionByExtensionID(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.1.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 2,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.1.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 2,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension:       "tei.reference.z",
				CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.z",
			Namespace: "z",
			Version:   "2.0.0",
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.z",
			Namespace: "z",
			Version:   "1.0.0",
		},
	}

	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{
			{Extension: "tei.reference.a", CompatibleMajor: 1},
			{Extension: "tei.reference.b", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	versions := map[string]string{}
	for _, extension := range catalog.Extensions {
		versions[extension.ID] = extension.Version
	}
	if got := versions["tei.reference.a"]; got != "1.1.0" {
		t.Fatalf("resolution policy selected a=%q want=1.1.0", got)
	}
	if got := versions["tei.reference.b"]; got != "1.0.0" {
		t.Fatalf("resolution policy selected b=%q want=1.0.0", got)
	}
	if got := versions["tei.reference.z"]; got != "1.0.0" {
		t.Fatalf("resolution policy selected z=%q want=1.0.0", got)
	}
}

func TestReferenceCatalogExposesAndFingerprintsResolutionPolicy(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.policy",
		Namespace: "policy",
		Version:   "1.0.0",
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{Extension: "tei.reference.policy", CompatibleMajor: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.ResolutionPolicy != semanticResolutionPolicy {
		t.Fatalf("resolution policy=%q want=%q", catalog.ResolutionPolicy, semanticResolutionPolicy)
	}

	changed := catalog
	changed.ResolutionFingerprint = ""
	changed.EvidenceFingerprint = ""
	changed.ResolutionPolicy = "different-policy/v0"
	if got := semanticCatalogFingerprint(changed); got == catalog.ResolutionFingerprint {
		t.Fatalf("resolution policy change did not change semantic resolution fingerprint: %s", got)
	}
}

func TestReferenceCatalogNormalizesRootRequirementsAndFingerprintsResolution(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Concepts: []ExtensionConcept{{
				Subject: subject.Subject{Kind: "view-a", ID: "a.example"},
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Concepts: []ExtensionConcept{{
				Subject: subject.Subject{Kind: "view-b", ID: "b.example"},
			}},
		},
	}
	left, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{
			{Extension: "tei.reference.b", CompatibleMajor: 1},
			{Extension: "tei.reference.a", CompatibleMajor: 1},
			{Extension: "tei.reference.a", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{
			{Extension: "tei.reference.a", CompatibleMajor: 1},
			{Extension: "tei.reference.b", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(left.ResolutionFingerprint, "sha256:") ||
		!strings.HasPrefix(left.EvidenceFingerprint, "sha256:") {
		t.Fatalf("catalog fingerprints: resolution=%q evidence=%q", left.ResolutionFingerprint, left.EvidenceFingerprint)
	}
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf("root order changed resolution identity: %s != %s", left.ResolutionFingerprint, right.ResolutionFingerprint)
	}
	if !reflect.DeepEqual(left.Requirements, right.Requirements) || len(left.Requirements) != 2 {
		t.Fatalf("root requirements are not normalized: left=%+v right=%+v", left.Requirements, right.Requirements)
	}
}

func TestReferenceCatalogRejectsConflictingRootMajors(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.policy",
		Namespace: "policy",
		Version:   "1.0.0",
	}
	_, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{
			{Extension: "tei.reference.policy", CompatibleMajor: 1},
			{Extension: "tei.reference.policy", CompatibleMajor: 2},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "incompatible majors") {
		t.Fatalf("conflicting root majors were not rejected: %v", err)
	}
}

func TestReferenceCatalogSeparatesCoreMeaningFromSourceIndexProvenance(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.policy",
		Namespace: "policy",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "policy-view", ID: "policy.example"},
		}},
	}
	root := []ExtensionRequirement{{Extension: "tei.reference.policy", CompatibleMajor: 1}}
	coreSubjects := []subject.Subject{{Kind: "semantic-data", ID: "customer.customer"}}

	left, err := resolveSemanticCatalog(
		CoreCatalogInput{
			Subjects:               coreSubjects,
			SubjectFingerprint:     "sha256:" + strings.Repeat("1", 64),
			SourceIndexFingerprint: "sha256:" + strings.Repeat("a", 64),
		},
		[]ExtensionDescriptor{descriptor},
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolveSemanticCatalog(
		CoreCatalogInput{
			Subjects:               coreSubjects,
			SubjectFingerprint:     "sha256:" + strings.Repeat("1", 64),
			SourceIndexFingerprint: "sha256:" + strings.Repeat("b", 64),
		},
		[]ExtensionDescriptor{descriptor},
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf("source provenance changed semantic resolution identity: %s != %s", left.ResolutionFingerprint, right.ResolutionFingerprint)
	}
	if left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatalf("Source Index provenance change did not change Evidence identity: %s", left.EvidenceFingerprint)
	}

	changedMeaning, err := resolveSemanticCatalog(
		CoreCatalogInput{
			Subjects: []subject.Subject{
				{Kind: "semantic-data", ID: "customer.customer"},
				{Kind: "semantic-data", ID: "sales.order"},
			},
			SubjectFingerprint:     "sha256:" + strings.Repeat("2", 64),
			SourceIndexFingerprint: "sha256:" + strings.Repeat("c", 64),
		},
		[]ExtensionDescriptor{descriptor},
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	if left.ResolutionFingerprint == changedMeaning.ResolutionFingerprint {
		t.Fatalf("core subject-set change did not change semantic resolution identity: %s", left.ResolutionFingerprint)
	}
}

func TestReferenceCatalogResolutionIdentityIgnoresRawDescriptorProvenance(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.policy",
		Namespace: "policy",
		Version:   "1.0.0",
		Concepts: []ExtensionConcept{{
			Subject: subject.Subject{Kind: "policy-view", ID: "policy.example"},
		}},
	}
	leftDescriptor := descriptor
	leftDescriptor.sourcePath = "first/policy.json"
	leftDescriptor.sourceFingerprint = "sha256:" + strings.Repeat("a", 64)
	rightDescriptor := descriptor
	rightDescriptor.sourcePath = "moved/policy.json"
	rightDescriptor.sourceFingerprint = "sha256:" + strings.Repeat("b", 64)
	root := []ExtensionRequirement{{Extension: "tei.reference.policy", CompatibleMajor: 1}}

	left, err := resolveSemanticCatalog(CoreCatalogInput{}, []ExtensionDescriptor{leftDescriptor}, root)
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolveSemanticCatalog(CoreCatalogInput{}, []ExtensionDescriptor{rightDescriptor}, root)
	if err != nil {
		t.Fatal(err)
	}
	if left.ResolutionFingerprint != right.ResolutionFingerprint {
		t.Fatalf("raw descriptor provenance changed semantic resolution identity: %s != %s", left.ResolutionFingerprint, right.ResolutionFingerprint)
	}
	if left.EvidenceFingerprint == right.EvidenceFingerprint {
		t.Fatalf("raw descriptor provenance did not change Evidence identity: %s", left.EvidenceFingerprint)
	}
	if left.Extensions[0].Source == right.Extensions[0].Source ||
		left.Extensions[0].SourceFingerprint == right.Extensions[0].SourceFingerprint {
		t.Fatalf("raw provenance was not preserved separately: left=%+v right=%+v", left.Extensions[0], right.Extensions[0])
	}
	if left.Extensions[0].DescriptorFingerprint != right.Extensions[0].DescriptorFingerprint {
		t.Fatalf("equivalent normalized descriptors produced different descriptor identities: %s != %s", left.Extensions[0].DescriptorFingerprint, right.Extensions[0].DescriptorFingerprint)
	}
}

func TestReferenceCatalogIsIndependentOfDescriptorLoadOrder(t *testing.T) {
	descriptors := referenceDescriptors(t)
	reversed := append([]ExtensionDescriptor(nil), descriptors...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	roots := []ExtensionRequirement{{Extension: "tei.reference.service-level-policy", CompatibleMajor: 1}}
	left, err := resolveSemanticCatalog(referenceCoreCatalog(t), descriptors, roots)
	if err != nil {
		t.Fatal(err)
	}
	right, err := resolveSemanticCatalog(referenceCoreCatalog(t), reversed, roots)
	if err != nil {
		t.Fatal(err)
	}
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	if !reflect.DeepEqual(leftJSON, rightJSON) {
		t.Fatalf("catalog depends on descriptor load order\nleft=%s\nright=%s", leftJSON, rightJSON)
	}
}

func catalogHasSubject(catalog SemanticCatalog, want subject.Subject) bool {
	for _, entry := range catalog.Subjects {
		if entry.Subject == want {
			return true
		}
	}
	return false
}

func catalogHasRelation(catalog SemanticCatalog, from subject.Subject, relation string, to subject.Subject) bool {
	for _, item := range catalog.Relations {
		if item.From == from && item.Relation == relation && item.To == to {
			return true
		}
	}
	return false
}

func catalogHasUnresolved(catalog SemanticCatalog, code string, want subject.Subject) bool {
	for _, item := range catalog.Unresolved {
		if item.Code == code && item.Subject != nil && *item.Subject == want {
			return true
		}
	}
	return false
}


func catalogSubject(catalog SemanticCatalog, want subject.Subject) (CatalogSubject, bool) {
	for _, entry := range catalog.Subjects {
		if entry.Subject == want {
			return entry, true
		}
	}
	return CatalogSubject{}, false
}

func catalogSourcesContain(values []CatalogSource, path string) bool {
	for _, value := range values {
		if value.Path == path && value.Fingerprint != "" {
			return true
		}
	}
	return false
}
