package businessprocedure

import (
	"testing"

	"tei/internal/workspace"
)

// TestDeclaredTargetsMatchTheRealIndex is what keeps the hand-written target
// table honest. The table exists so the unit tests need no Workspace, but a
// hand-copied table drifts: it is only a second source of truth while nothing
// compares it with the first. This test compares the exact Source position of
// every identity, so a moved declaration fails here instead of being silently
// accepted by the unit tests.
func TestDeclaredTargetsMatchTheRealIndex(t *testing.T) {
	root := workspaceRoot()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	actual, err := DeclaredFromWorkspace(root, loaded)
	if err != nil {
		t.Fatalf("build source index: %v", err)
	}

	type key struct {
		kind ReferenceKind
		id   string
	}
	byKey := map[key]Declared{}
	for _, target := range actual {
		byKey[key{target.Kind, target.ID}] = target
	}

	for _, expected := range declaredTargets() {
		target, ok := byKey[key{expected.Kind, expected.ID}]
		if !ok {
			t.Errorf("%s %q is listed but the Workspace does not declare it", expected.Kind, expected.ID)
			continue
		}
		if target.Artifact != expected.Artifact {
			t.Errorf("%s %q: artifact %q, table says %q", expected.Kind, expected.ID, target.Artifact, expected.Artifact)
		}
		if target.Line != expected.Line || target.Column != expected.Column {
			t.Errorf("%s %q: source is %d:%d, table says %d:%d",
				expected.Kind, expected.ID, target.Line, target.Column, expected.Line, expected.Column)
		}
		if target.Subject != expected.Subject {
			t.Errorf("%s %q: subject %q, table says %q", expected.Kind, expected.ID, target.Subject, expected.Subject)
		}
		if target.Fingerprint == "" {
			t.Errorf("%s %q resolved without an artifact digest", expected.Kind, expected.ID)
		}
	}

	// Every identity the example references must be in the table, so a step
	// cannot resolve through an entry the table never pins.
	listed := map[key]bool{}
	for _, expected := range declaredTargets() {
		listed[key{expected.Kind, expected.ID}] = true
	}
	for _, step := range readFixture(t).Steps {
		if step.Function == nil {
			continue
		}
		if !listed[key{step.Function.Kind, step.Function.ID}] {
			t.Errorf("the example references %s %q, which the table does not pin",
				step.Function.Kind, step.Function.ID)
		}
	}
}
