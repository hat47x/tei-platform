package businessprocedure

import (
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// workspaceRoot is the reference Workspace the example procedure points at.
func workspaceRoot() string {
	return filepath.Join("..", "..", "reference", "workspace")
}

// TestExampleReferencesResolveAgainstTheRealWorkspace is the integration check:
// every Function reference in the public example must resolve to exactly one
// declaration in the actual reference Workspace, and that declaration must carry
// a real Source location. The unit tests use a hand-written target list; this
// one uses the Workspace's own Source Index.
func TestExampleReferencesResolveAgainstTheRealWorkspace(t *testing.T) {
	root := workspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	declared, err := DeclaredFromWorkspace(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	if len(declared) == 0 {
		t.Fatal("the Source Index yielded no declared targets; the mapping changed")
	}

	fixture := readFixture(t)
	projection := Project(fixture, Fingerprint(fixture), NewResolver(declared))
	if len(projection.Diagnostics) != 0 {
		t.Fatalf("the example did not resolve against the real Workspace: %+v", projection.Diagnostics)
	}

	for _, step := range projection.Steps {
		if step.Function == nil {
			if step.NoFunction && step.Manual {
				continue
			}
			t.Fatalf("step %s declares a Function but resolved none", step.ID)
		}
		if step.Function.Artifact == "" {
			t.Fatalf("step %s resolved %q without a source artifact", step.ID, step.Function.ID)
		}
		if step.Function.Line < 1 || step.Function.Column < 1 {
			t.Fatalf("step %s resolved %q without a source position: %+v",
				step.ID, step.Function.ID, step.Function)
		}
		if step.Function.Role != "definition" {
			t.Fatalf("step %s resolved to a %q reference, not a definition",
				step.ID, step.Function.Role)
		}
	}

	// The resolved artifact must be a file the Workspace actually manages, so a
	// source location cannot point outside the loaded workspace.
	managed := map[string]bool{}
	for _, file := range loaded.ManagedFiles {
		managed[file.Path] = true
	}
	for _, step := range projection.Steps {
		if step.Function == nil {
			continue
		}
		if !managed[step.Function.Artifact] {
			t.Fatalf("step %s resolved to %q, which the Workspace does not manage",
				step.ID, step.Function.Artifact)
		}
	}
}

// TestUnknownFunctionInTheRealWorkspaceIsReported checks that the real index
// refuses an identity it does not declare, rather than matching a similar one.
func TestUnknownFunctionInTheRealWorkspaceIsReported(t *testing.T) {
	root := workspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	declared, err := DeclaredFromWorkspace(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}

	fixture := readFixture(t)
	for i := range fixture.Steps {
		if fixture.Steps[i].ID == "BP-A06" {
			fixture.Steps[i].Function = &Reference{
				Kind: ReferencePattern,
				ID:   "order_entry.submit_order",
			}
			// The same words under the wrong kind must not resolve.
			fixture.Steps[i].Function.Kind = ReferenceCapability
		}
	}
	projection := Project(fixture, Fingerprint(fixture), NewResolver(declared))
	if codesOf(projection)[CodeReferenceUnresolved] != 1 {
		t.Fatalf("a Pattern id asked for as a capability resolved anyway: %+v", projection.Diagnostics)
	}
	if findStep(t, projection, "BP-A06").Function != nil {
		t.Fatal("a reference under the wrong kind was resolved")
	}
}

// The index must offer only definition-role references as targets. The real
// reference Workspace carries no authoring or derived reference of a mapped
// subject kind, so a fixture Index is needed: without one, removing the role
// filter would not fail any test.
func TestDeclaredFromIndexIgnoresNonDefinitionReferences(t *testing.T) {
	definition := sourceindex.Reference{
		Subject:      sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.submit_order"},
		ArtifactPath: "applications/order-entry/patterns/submit-order.tei",
		Role:         sourceindex.RoleDefinition,
	}
	authoring := sourceindex.Reference{
		Subject:      sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.submit_order"},
		ArtifactPath: "applications/order-entry/patterns/submit-order.tei",
		Role:         sourceindex.RoleAuthoring,
	}
	declared := DeclaredFromIndex(sourceindex.Index{
		Artifacts:  []sourceindex.Artifact{{Path: definition.ArtifactPath, Fingerprint: "sha256:test"}},
		References: []sourceindex.Reference{definition, authoring},
	})

	if len(declared) != 1 {
		t.Fatalf("%d targets from one definition and one authoring reference, want 1: %+v", len(declared), declared)
	}
	if declared[0].Role != "definition" {
		t.Fatalf("a %q reference became a resolvable target", declared[0].Role)
	}
	if declared[0].Fingerprint != "sha256:test" {
		t.Fatalf("the artifact digest was not carried: %+v", declared[0])
	}
}

// The real Workspace offers only definition references too; this keeps the
// synthetic case above from drifting away from the actual index.
func TestRealWorkspaceOffersOnlyDefinitionTargets(t *testing.T) {
	root := workspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	declared, err := DeclaredFromWorkspace(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	if len(declared) == 0 {
		t.Fatal("the real index yielded no targets")
	}
	for _, target := range declared {
		if target.Role != "definition" {
			t.Fatalf("a %q reference became a resolvable target: %+v", target.Role, target)
		}
		if target.Fingerprint == "" {
			t.Fatalf("a target carries no artifact digest: %+v", target)
		}
	}
}
