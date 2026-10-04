package main

import (
	"reflect"
	"testing"
)

func explanationByID(
	values []CatalogDependencyExplanation,
	id string,
) (CatalogDependencyExplanation, bool) {
	for _, value := range values {
		if value.Extension.ID == id {
			return value, true
		}
	}
	return CatalogDependencyExplanation{}, false
}

func TestDependencyExplanationIncludesRequiresReverseDepsAndCanonicalRootPath(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{
				{Extension: "tei.reference.b", CompatibleMajor: 1},
				{Extension: "tei.reference.c", CompatibleMajor: 1},
			},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension: "tei.reference.d", CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.c",
			Namespace: "c",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension: "tei.reference.d", CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.d",
			Namespace: "d",
			Version:   "1.0.0",
		},
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{{
			Extension: "tei.reference.a", CompatibleMajor: 1,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Unresolved) != 0 {
		t.Fatalf("unexpected unresolved: %+v", catalog.Unresolved)
	}

	explanation, ok := explanationByID(catalog.DependencyExplanations, "tei.reference.d")
	if !ok {
		t.Fatalf("dependency explanation for d is missing: %+v", catalog.DependencyExplanations)
	}
	if got, want := explanation.RequiredBy, []ExtensionIdentity{
		{ID: "tei.reference.b", Version: "1.0.0"},
		{ID: "tei.reference.c", Version: "1.0.0"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("requiredBy=%+v want=%+v", got, want)
	}
	if len(explanation.RootPaths) != 1 {
		t.Fatalf("root paths=%+v", explanation.RootPaths)
	}
	if got, want := explanation.RootPaths[0].Path, []ExtensionIdentity{
		{ID: "tei.reference.a", Version: "1.0.0"},
		{ID: "tei.reference.b", Version: "1.0.0"},
		{ID: "tei.reference.d", Version: "1.0.0"},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical root path=%+v want=%+v", got, want)
	}
}

func TestDependencyExplanationTracksEveryRootWithoutEnumeratingEveryPath(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension: "tei.reference.shared", CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension: "tei.reference.shared", CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.shared",
			Namespace: "shared",
			Version:   "1.0.0",
		},
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		descriptors,
		[]ExtensionRequirement{
			{Extension: "tei.reference.b", CompatibleMajor: 1},
			{Extension: "tei.reference.a", CompatibleMajor: 1},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	explanation, ok := explanationByID(catalog.DependencyExplanations, "tei.reference.shared")
	if !ok {
		t.Fatal("shared dependency explanation is missing")
	}
	if len(explanation.RootPaths) != 2 {
		t.Fatalf("root paths=%+v", explanation.RootPaths)
	}
	if explanation.RootPaths[0].Root.Extension != "tei.reference.a" ||
		explanation.RootPaths[1].Root.Extension != "tei.reference.b" {
		t.Fatalf("root paths are not deterministic: %+v", explanation.RootPaths)
	}
}

func TestDependencyExplanationDoesNotChangeSemanticOrEvidenceIdentity(t *testing.T) {
	descriptor := ExtensionDescriptor{
		Schema:    extensionSchema,
		ID:        "tei.reference.a",
		Namespace: "a",
		Version:   "1.0.0",
	}
	catalog, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		[]ExtensionDescriptor{descriptor},
		[]ExtensionRequirement{{
			Extension: descriptor.ID, CompatibleMajor: 1,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	changed := catalog
	changed.DependencyExplanations = []CatalogDependencyExplanation{{
		Extension: ExtensionIdentity{ID: "synthetic.explanation", Version: "9.9.9"},
	}}
	if got := semanticCatalogFingerprint(changed); got != catalog.ResolutionFingerprint {
		t.Fatalf("derived explanation changed semantic identity: %s != %s", got, catalog.ResolutionFingerprint)
	}
	if got := semanticCatalogEvidenceFingerprint(changed); got != catalog.EvidenceFingerprint {
		t.Fatalf("derived explanation changed Evidence identity: %s != %s", got, catalog.EvidenceFingerprint)
	}
}

func TestDependencyExplanationIsIndependentOfDescriptorAndRootOrder(t *testing.T) {
	descriptors := []ExtensionDescriptor{
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.a",
			Namespace: "a",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension: "tei.reference.shared", CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.b",
			Namespace: "b",
			Version:   "1.0.0",
			Requires: []ExtensionRequirement{{
				Extension: "tei.reference.shared", CompatibleMajor: 1,
			}},
		},
		{
			Schema:    extensionSchema,
			ID:        "tei.reference.shared",
			Namespace: "shared",
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
	right, err := resolveSemanticCatalog(
		CoreCatalogInput{},
		reversedDescriptors,
		[]ExtensionRequirement{roots[1], roots[0]},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(left.DependencyExplanations, right.DependencyExplanations) {
		t.Fatalf(
			"dependency explanation depends on input order: left=%+v right=%+v",
			left.DependencyExplanations,
			right.DependencyExplanations,
		)
	}
}
