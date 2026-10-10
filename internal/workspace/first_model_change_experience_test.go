package workspace_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const firstModelRoot = "examples/first-workspace"
const firstModelSemantic = "semantic/intake/request.tei"

func TestFirstModelChangeIndependentWorkspace(t *testing.T) {
	root := filepath.Join(repoRoot(t), filepath.FromSlash(firstModelRoot))
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load first workspace: %v", err)
	}
	if ds := validation.Validate(loaded.World); validation.HasErrors(ds) {
		t.Fatalf("first workspace must validate: %#v", ds)
	}
	if len(loaded.World.Applications) != 1 || loaded.World.Applications[0].ID != "request_log" {
		t.Fatalf("unexpected applications: %#v", loaded.World.Applications)
	}
	if len(loaded.World.Databases) != 0 || len(loaded.World.Persistence) != 0 ||
		len(loaded.World.Migrations) != 0 || len(loaded.World.Environments) != 0 ||
		len(loaded.World.Plugins) != 0 || len(loaded.World.Queries) != 0 ||
		len(loaded.World.Bindings) != 0 || len(loaded.World.Applications[0].Patterns) != 0 {
		t.Fatal("model-only first experience must not imply function, database, or runtime readiness")
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build first workspace source index: %v", err)
	}
	assertFirstModelReference(t, idx, "semantic-data", "intake.request", firstModelSemantic)
	assertFirstModelReference(t, idx, "semantic-attribute", "intake.request.status", firstModelSemantic)
	assertFirstModelReference(t, idx, "application", "request_log", "applications/request-log/application.tei")
}

func TestFirstModelChangeEditValidationAndRepair(t *testing.T) {
	base := filepath.Join(repoRoot(t), filepath.FromSlash(firstModelRoot))
	start, err := workspace.Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if ds := validation.Validate(start.World); validation.HasErrors(ds) {
		t.Fatalf("baseline is invalid: %#v", ds)
	}
	startIndex, err := workspace.BuildSourceIndex(base, start)
	if err != nil {
		t.Fatal(err)
	}

	// The task must work in an independently copied user Workspace.
	userRoot := t.TempDir()
	for _, file := range []string{
		"tei.workspace.yaml",
		firstModelSemantic,
		"applications/request-log/application.tei",
	} {
		content, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(userRoot, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	semanticPath := filepath.Join(userRoot, filepath.FromSlash(firstModelSemantic))
	original, err := os.ReadFile(semanticPath)
	if err != nil {
		t.Fatal(err)
	}
	const existing = "    attribute status type std.code\n"
	const added = existing + "    attribute priority type std.code\n"
	if strings.Count(string(original), existing) != 1 {
		t.Fatal("sample source changed: status attribute anchor is not unique")
	}
	updated := strings.Replace(string(original), existing, added, 1)
	if err := os.WriteFile(semanticPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatalf("reload changed workspace: %v", err)
	}
	if ds := validation.Validate(changed.World); validation.HasErrors(ds) {
		t.Fatalf("valid additive model change was rejected: %#v", ds)
	}
	if reflect.DeepEqual(start.World, changed.World) {
		t.Fatal("adding a business attribute did not change canonical meaning")
	}
	if !reflect.DeepEqual(start.World.Applications, changed.World.Applications) {
		t.Fatal("adding an attribute must not change application identity or definition")
	}
	changedIndex, err := workspace.BuildSourceIndex(userRoot, changed)
	if err != nil {
		t.Fatal(err)
	}
	readUserSource := func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(userRoot, filepath.FromSlash(rel)))
	}
	if err := startIndex.Validate(readUserSource); err == nil {
		t.Fatal("the original Source Index accepted an edited source without a fingerprint mismatch")
	}
	if err := changedIndex.Validate(readUserSource); err != nil {
		t.Fatalf("the rebuilt Source Index must reflect the current source bytes: %v", err)
	}
	assertFirstModelReference(t, changedIndex, "semantic-data", "intake.request", firstModelSemantic)
	assertFirstModelReference(t, changedIndex, "semantic-attribute", "intake.request.priority", firstModelSemantic)
	if firstModelFingerprint(t, startIndex, firstModelSemantic) == firstModelFingerprint(t, changedIndex, firstModelSemantic) {
		t.Fatal("edited authoritative source retained its old fingerprint")
	}
	if firstModelFingerprint(t, startIndex, "applications/request-log/application.tei") != firstModelFingerprint(t, changedIndex, "applications/request-log/application.tei") {
		t.Fatal("unchanged application source was unexpectedly modified")
	}

	invalid := strings.Replace(updated, "priority type std.code", "priority type std.not_defined", 1)
	if err := os.WriteFile(semanticPath, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatalf("malformed type must reach validation rather than silently disappearing: %v", err)
	}
	brokenIndex, err := workspace.BuildSourceIndex(userRoot, broken)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, projected := range workspace.ProjectDiagnostics(validation.Validate(broken.World), brokenIndex) {
		if projected.Diagnostic.Severity != validation.Error || projected.Diagnostic.Code != "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE" {
			continue
		}
		if projected.Diagnostic.Subject == nil ||
			projected.Diagnostic.Subject.Kind != "semantic-attribute" ||
			projected.Diagnostic.Subject.ID != "intake.request.priority" {
			t.Fatalf("invalid type has no stable semantic subject: %+v", projected.Diagnostic)
		}
		if len(projected.Sources) != 1 || projected.Sources[0].ArtifactPath != firstModelSemantic {
			t.Fatalf("invalid type has no actionable source location: %+v", projected.Sources)
		}
		found = true
	}
	if !found {
		t.Fatal("unknown business attribute type did not produce a validation error")
	}

	if err := os.WriteFile(semanticPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	repaired, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatal(err)
	}
	if ds := validation.Validate(repaired.World); validation.HasErrors(ds) {
		t.Fatalf("repair did not restore validity: %#v", ds)
	}
	if !reflect.DeepEqual(changed.World, repaired.World) {
		t.Fatal("repair did not restore the exact canonical meaning")
	}
}

func assertFirstModelReference(t *testing.T, idx sourceindex.Index, kind, id, path string) {
	t.Helper()
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: kind, ID: id})
	if len(refs) != 1 || refs[0].ArtifactPath != path || refs[0].Role != sourceindex.RoleDefinition {
		t.Fatalf("source index for %s:%s = %#v, want definition at %q", kind, id, refs, path)
	}
}

func firstModelFingerprint(t *testing.T, idx sourceindex.Index, path string) string {
	t.Helper()
	for _, artifact := range idx.Artifacts {
		if artifact.Path == path {
			if artifact.Fingerprint == "" {
				t.Fatalf("source %q has no fingerprint", path)
			}
			return artifact.Fingerprint
		}
	}
	t.Fatalf("source artifact %q was not indexed", path)
	return ""
}
