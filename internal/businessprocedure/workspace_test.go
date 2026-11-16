package businessprocedure

import (
	"path/filepath"
	"testing"

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

// The index must offer only definition references as targets.
func TestDeclaredFromIndexIgnoresNonDefinitionReferences(t *testing.T) {
	root := workspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	declared, err := DeclaredFromWorkspace(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}
	for _, target := range declared {
		if target.Role != "definition" {
			t.Fatalf("a %q reference became a resolvable target: %+v", target.Role, target)
		}
		if target.Subject == "" || target.Artifact == "" {
			t.Fatalf("a declared target is missing its identity or source: %+v", target)
		}
	}
}
