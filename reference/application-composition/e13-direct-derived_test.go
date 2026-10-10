// E13 tests: the new transport remains a projection of the existing Derived Index.
// These tests require a trusted full TEI checkout and are not validation receipts.
package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/index"
	"tei/internal/workspace"
)

func TestE13DirectIndexUsesExistingWorldProjection(t *testing.T) {
	world, _ := e5PatternTargetsFixture()
	got := e13FromWorld(world)
	fromCanonicalIndex := index.Build(world)
	if !reflect.DeepEqual(got.SemanticElements, fromCanonicalIndex.SemanticElements) ||
		!reflect.DeepEqual(got.SemanticDependencies, fromCanonicalIndex.SemanticDependencies) ||
		!reflect.DeepEqual(got.DeclaredDependencies, fromCanonicalIndex.DeclaredDependencies) ||
		!reflect.DeepEqual(got.ApplicationPatterns, fromCanonicalIndex.ApplicationPatterns) ||
		!reflect.DeepEqual(got.Queries, fromCanonicalIndex.Queries) {
		t.Fatal("direct evidence differs from the existing Derived Index")
	}
	if len(got.ApplicationPatterns) == 0 {
		t.Fatal("existing Application Pattern lost")
	}
}

func TestE13LegacyEnvelopeDoesNotExposeDerivedWithoutOptIn(t *testing.T) {
	world, srcIndex := e5Fixture()
	report := inspectE5(world, srcIndex, "order_entry")
	envelope := e5PairedEvidence(report, srcIndex)
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["derived"]; ok {
		t.Fatal("default E5 envelope changed without opt-in")
	}
	if decoded["report"] == nil || decoded["index"] == nil {
		t.Fatal("existing E7 paired fields missing")
	}
}

func TestE13DerivedEvidenceIsBoundToTheSameInMemoryWorld(t *testing.T) {
	world, srcIndex := e5PatternTargetsFixture()
	report := inspectE5(world, srcIndex, "order_entry")
	envelope := e5PairedEvidence(report, srcIndex)
	narrowed := e13FromWorld(world)
	envelope.Derived = &narrowed
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	var projected map[string]json.RawMessage
	if err := json.Unmarshal(decoded["derived"], &projected); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"semanticElements", "semanticDependencies", "declaredDependencies", "applicationPatterns", "queries"} {
		if _, ok := projected[name]; !ok {
			t.Fatalf("missing explicit derived field %s", name)
		}
	}
	for _, forbidden := range []string{"permissionGrants", "resourceBindings", "environmentBindings", "plugins", "bundles", "releasePlans"} {
		if _, ok := projected[forbidden]; ok {
			t.Fatalf("unexpected non-direct field %s", forbidden)
		}
	}
	if envelope.Qualification != "read-only-paired-input-not-an-authority" {
		t.Fatal("projection must not acquire authority")
	}
}

func TestE13ReferenceWorkspaceApplicationPatternMembership(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	report := inspectE5(loaded.World, sources, "order_entry")
	if report.Status != "observed" {
		t.Fatalf("real E5 source observation unresolved: %+v", report.Issues)
	}
	patterns := map[string]bool{}
	for _, dependency := range report.Dependencies {
		if dependency.Relation == "application-pattern" {
			patterns[dependency.Subject.ID] = true
		}
	}
	derived := e13FromWorld(loaded.World)
	indexPatterns := map[string]bool{}
	for _, pattern := range derived.ApplicationPatterns {
		if string(pattern.Application) == "order_entry" {
			id := string(pattern.Pattern)
			if indexPatterns[id] {
				t.Fatalf("duplicate derived Application Pattern %s", id)
			}
			indexPatterns[id] = true
		}
	}
	if len(patterns) == 0 || !reflect.DeepEqual(patterns, indexPatterns) {
		t.Fatalf("E5 and existing Derived Index disagree on order_entry Pattern membership: E5=%v Derived=%v", patterns, indexPatterns)
	}
}

func TestE13SUIStage0RemainsNonApplication(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "sui-adoption", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load SUI Stage 0: %v", err)
	}
	derived := e13FromWorld(loaded.World)
	for _, pattern := range derived.ApplicationPatterns {
		if string(pattern.Application) == "sui_reference_adoption_stage0" {
			t.Fatalf("semantic-only SUI Stage 0 acquired an Application Pattern: %+v", pattern)
		}
	}
}
