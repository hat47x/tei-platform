// E14 confirms read-only cross-view declarations and Source Index locations.
// This is not modelref Impact execution, accepted business meaning or authority.
package main

import (
	"path/filepath"
	"testing"

	"tei/internal/index"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func e14HasDeclaredEdge(deps []index.DeclaredDependency, fromKind, fromID, relation, toKind, toID string) bool {
	for _, edge := range deps {
		if edge.FromKind == fromKind && string(edge.From) == fromID &&
			edge.Relation == relation && edge.ToKind == toKind && string(edge.To) == toID {
			return true
		}
	}
	return false
}

func e14HasSemanticEdge(deps []index.SemanticDependency, fromKind, fromID, relation, toKind, toID string) bool {
	for _, edge := range deps {
		if edge.FromKind == fromKind && string(edge.From) == fromID &&
			edge.Relation == relation && edge.ToKind == toKind && string(edge.To) == toID {
			return true
		}
	}
	return false
}

func e14UniqueDefinition(indexed sourceindex.Index, kind, id, path string) bool {
	matches := 0
	artifacts := 0
	for _, ref := range indexed.References {
		if ref.Subject.Kind == kind && ref.Subject.ID == id && ref.Role == sourceindex.RoleDefinition {
			matches++
			if ref.ArtifactPath != path {
				return false
			}
		}
	}
	for _, artifact := range indexed.Artifacts {
		if artifact.Path == path {
			artifacts++
		}
	}
	return matches == 1 && artifacts == 1
}

func TestE14OrderEntryCrossViewReferenceChainFromOneWorld(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build source index from loaded workspace: %v", err)
	}
	report := inspectE5(loaded.World, sources, "order_entry")
	if report.Status != "observed" {
		t.Fatalf("application source observation unresolved: %+v", report.Issues)
	}
	derived := e13FromWorld(loaded.World)
	if !e14HasDeclaredEdge(derived.DeclaredDependencies, "application", "order_entry",
		"owns-pattern", "application-pattern", "order_entry.customer_reference") {
		t.Fatal("missing existing Application -> Pattern ownership edge")
	}
	if !e14HasDeclaredEdge(derived.DeclaredDependencies, "application-pattern",
		"order_entry.customer_reference", "relation", "semantic-relation", "sales.order.customer") {
		t.Fatal("missing existing Pattern -> Relation edge")
	}
	if !e14HasSemanticEdge(derived.SemanticDependencies, "semantic-relation",
		"sales.order.customer", "targets", "semantic-data", "customer.customer") {
		t.Fatal("missing existing Relation -> Data target edge")
	}
	if !e14UniqueDefinition(sources, "application-pattern",
		"order_entry.customer_reference", "applications/order-entry/patterns/customer-reference.tei") ||
		!e14UniqueDefinition(sources, "semantic-relation",
			"sales.order.customer", "semantic/sales/order.tei") {
		t.Fatal("required direct reference Source anchors are absent or ambiguous")
	}
}

func TestE14ReferenceCandidateMustNotInventOtherApplicationOwnership(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	derived := index.Build(loaded.World)
	// This check is on one known Application and one known Pattern only.
	// The existence of a Source/Pattern name cannot create a new owner.
	if e14HasDeclaredEdge(derived.DeclaredDependencies, "application", "repair_intake",
		"owns-pattern", "application-pattern", "order_entry.customer_reference") {
		t.Fatal("another Application acquired the order_entry Pattern")
	}
}

func TestE14SingleWorldProjectionRetainsGenericCapabilityInputKind(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	derived := e13FromWorld(loaded.World)
	if !e14HasSemanticEdge(derived.SemanticDependencies, "semantic-capability",
		"sales.order.submit", "input", "semantic-element", "sales.order") {
		t.Fatal("existing Capability input reference missing or incorrectly narrowed")
	}
	// A generic semantic-element is not implicitly a validated type,
	// accepted action, runtime permission, or complete dependency closure.
}

func TestE14AmbiguousOrDerivedOnlySourceIsNotADefinition(t *testing.T) {
	const src = "semantic/sales/order.tei"
	artifact := sourceindex.Artifact{
		Path: src, Fingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	def := sourceindex.Reference{
		Subject: sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"},
		ArtifactPath: src, Role: sourceindex.RoleDefinition,
	}
	for _, test := range []struct {
		name string
		index sourceindex.Index
	}{
		{"empty", sourceindex.Index{}},
		{"derived-only", sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact},
			References: []sourceindex.Reference{{Subject: def.Subject, ArtifactPath: src,
				Role: sourceindex.RoleDerived}}}},
		{"duplicate-definition", sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact},
			References: []sourceindex.Reference{def, def}}},
		{"duplicate-artifact", sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact, artifact},
			References: []sourceindex.Reference{def}}},
		{"wrong-path", sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact},
			References: []sourceindex.Reference{{
				Subject: def.Subject, ArtifactPath: "semantic/other.tei", Role: sourceindex.RoleDefinition,
			}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if e14UniqueDefinition(test.index, def.Subject.Kind, def.Subject.ID, src) {
				t.Fatal("ambiguous or untrusted source anchor treated as a unique definition")
			}
		})
	}
}
