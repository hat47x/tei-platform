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
		anchor := projected.Sources[0].Range
		if anchor.Start.Offset >= anchor.End.Offset ||
			anchor.Start.Offset < 0 || anchor.End.Offset > len(invalid) ||
			invalid[anchor.Start.Offset:anchor.End.Offset] != "priority" {
			t.Fatalf("semantic type error must select the edited attribute identifier, not a file-level placeholder: %+v", anchor)
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
	repairedIndex, err := workspace.BuildSourceIndex(userRoot, repaired)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(changedIndex, repairedIndex) {
		t.Fatal("repair did not restore the same source lineage and fingerprint")
	}
}

// A missing Data relation is a common first step toward standard CRUD.
// It must have a stable model identity and an actionable exact Source token.
func TestFirstModelChangeRelationTargetDiagnosticAndRepair(t *testing.T) {
	base := filepath.Join(repoRoot(t), filepath.FromSlash(firstModelRoot))
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

	path := filepath.Join(userRoot, filepath.FromSlash(firstModelSemantic))
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const attribute = "    attribute status type std.code\n"
	const relation = "    relation parent to intake.unknown_request one\n"
	if strings.Count(string(src), attribute) != 1 {
		t.Fatal("first-workspace status declaration must be unique")
	}
	brokenSource := strings.Replace(string(src), attribute, attribute+relation, 1)
	if err := os.WriteFile(path, []byte(brokenSource), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatalf("unknown related Data must remain an authorable validation finding: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(userRoot, broken)
	if err != nil {
		t.Fatal(err)
	}
	var located bool
	for _, result := range workspace.ProjectDiagnostics(validation.Validate(broken.World), idx) {
		d := result.Diagnostic
		if d.Code != "UNKNOWN_SEMANTIC_RELATION_TARGET" {
			continue
		}
		if d.Severity != validation.Error || d.Subject == nil ||
			*d.Subject != (sourceindex.Subject{Kind: "semantic-relation", ID: "intake.request.parent"}) {
			t.Fatalf("missing relation target lacks a stable semantic Subject: %+v", d)
		}
		if len(result.Sources) != 1 || result.Sources[0].ArtifactPath != firstModelSemantic {
			t.Fatalf("missing relation target lost its source location: %+v", result.Sources)
		}
		anchor := result.Sources[0].Range
		if anchor.Start.Offset < 0 || anchor.Start.Offset >= anchor.End.Offset ||
			anchor.End.Offset > len(brokenSource) ||
			brokenSource[anchor.Start.Offset:anchor.End.Offset] != "parent" {
			t.Fatalf("missing relation target must point at its declaration identifier: %+v", anchor)
		}
		located = true
	}
	if !located {
		t.Fatal("missing relation target was silently accepted")
	}

	repairedSource := strings.Replace(brokenSource, "intake.unknown_request", "intake.request", 1)
	if err := os.WriteFile(path, []byte(repairedSource), 0o644); err != nil {
		t.Fatal(err)
	}
	repaired, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatal(err)
	}
	if ds := validation.Validate(repaired.World); validation.HasErrors(ds) {
		t.Fatalf("a corrected self-relation must validate: %+v", ds)
	}
	repairedIndex, err := workspace.BuildSourceIndex(userRoot, repaired)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstModelReference(t, repairedIndex, "semantic-relation", "intake.request.parent", firstModelSemantic)
	if err := idx.Validate(func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(userRoot, filepath.FromSlash(rel)))
	}); err == nil {
		t.Fatal("a Source Index for the incorrect relation was accepted after repair")
	}
}

// A semantic Action declaration is still only a model; its parameter
// errors must identify the authored capability before any binding exists.
func TestFirstModelChangeCapabilityTypeDiagnosticAndRepair(t *testing.T) {
	base := filepath.Join(repoRoot(t), filepath.FromSlash(firstModelRoot))
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

	path := filepath.Join(userRoot, filepath.FromSlash(firstModelSemantic))
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const action = `
capability intake.request.submit {
    input request intake.unknown_request
    output intake.unknown_result
}
`
	brokenSource := string(source) + action
	if err := os.WriteFile(path, []byte(brokenSource), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatalf("unknown Action parameter types should reach semantic validation: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(userRoot, broken)
	if err != nil {
		t.Fatal(err)
	}
	missing := map[string]bool{
		"UNKNOWN_SEMANTIC_CAPABILITY_INPUT":  false,
		"UNKNOWN_SEMANTIC_CAPABILITY_OUTPUT": false,
	}
	for _, projected := range workspace.ProjectDiagnostics(validation.Validate(broken.World), idx) {
		d := projected.Diagnostic
		if _, expect := missing[d.Code]; !expect {
			continue
		}
		if d.Severity != validation.Error || d.Subject == nil ||
			*d.Subject != (sourceindex.Subject{Kind: "semantic-capability", ID: "intake.request.submit"}) {
			t.Fatalf("invalid Action parameter has no stable capability Subject: %+v", d)
		}
		if len(projected.Sources) != 1 || projected.Sources[0].ArtifactPath != firstModelSemantic {
			t.Fatalf("invalid Action parameter lost its source location: %+v", projected.Sources)
		}
		anchor := projected.Sources[0].Range
		if anchor.Start.Offset < 0 || anchor.Start.Offset >= anchor.End.Offset ||
			anchor.End.Offset > len(brokenSource) ||
			brokenSource[anchor.Start.Offset:anchor.End.Offset] != "intake.request.submit" {
			t.Fatalf("invalid Action parameter must locate its capability declaration: %+v", anchor)
		}
		missing[d.Code] = true
	}
	for code, observed := range missing {
		if !observed {
			t.Fatalf("unknown Action type did not emit %s", code)
		}
	}

	repairedSource := strings.ReplaceAll(brokenSource, "intake.unknown_request", "intake.request")
	repairedSource = strings.ReplaceAll(repairedSource, "intake.unknown_result", "intake.request")
	if err := os.WriteFile(path, []byte(repairedSource), 0o644); err != nil {
		t.Fatal(err)
	}
	repaired, err := workspace.Load(userRoot)
	if err != nil {
		t.Fatal(err)
	}
	if ds := validation.Validate(repaired.World); validation.HasErrors(ds) {
		t.Fatalf("corrected semantic Action does not validate: %+v", ds)
	}
	if len(repaired.World.Bindings) != 0 || len(repaired.World.Applications[0].Patterns) != 0 {
		t.Fatal("validating Action meaning must not fabricate a runnable function or UI")
	}
	index, err := workspace.BuildSourceIndex(userRoot, repaired)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstModelReference(t, index, "semantic-capability", "intake.request.submit", firstModelSemantic)
}

// The default Validator must check typed relation-key mappings, not only
// existence of the target Data. This guards future Data-backed CRUD derivation.
func TestFirstModelChangeRelationKeyTypeMismatchAndRepair(t *testing.T) {
	base := filepath.Join(repoRoot(t), filepath.FromSlash(firstModelRoot))
	root := t.TempDir()
	for _, file := range []string{
		"tei.workspace.yaml",
		firstModelSemantic,
		"applications/request-log/application.tei",
	} {
		content, err := os.ReadFile(filepath.Join(base, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, filepath.FromSlash(firstModelSemantic))
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const status = "    attribute status type std.code\n"
	const relation = "    relation parent to intake.request one\n        key intake.request.summary -> intake.request.request_id\n"
	if strings.Count(string(source), status) != 1 {
		t.Fatal("cannot find the unique first-workspace insertion point")
	}
	invalid := strings.Replace(string(source), status, status+relation, 1)
	if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, broken)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, projected := range workspace.ProjectDiagnostics(validation.Validate(broken.World), index) {
		d := projected.Diagnostic
		if d.Code != "SEMANTIC_RELATION_KEY_TYPE_MISMATCH" {
			continue
		}
		if d.Severity != validation.Error || d.Subject == nil ||
			*d.Subject != (sourceindex.Subject{Kind: "semantic-relation", ID: "intake.request.parent"}) {
			t.Fatalf("relation-key mismatch has no typed Subject: %+v", d)
		}
		if len(projected.Sources) != 1 || projected.Sources[0].ArtifactPath != firstModelSemantic {
			t.Fatalf("relation-key mismatch lost its source: %+v", projected.Sources)
		}
		anchor := projected.Sources[0].Range
		if anchor.Start.Offset < 0 || anchor.Start.Offset >= anchor.End.Offset ||
			anchor.End.Offset > len(invalid) ||
			invalid[anchor.Start.Offset:anchor.End.Offset] != "parent" {
			t.Fatalf("relation-key mismatch does not locate its Relation: %+v", anchor)
		}
		found = true
	}
	if !found {
		t.Fatal("default Validation silently accepted mismatched Relation keys")
	}

	corrected := strings.Replace(invalid, "key intake.request.summary ->", "key intake.request.request_id ->", 1)
	if err := os.WriteFile(path, []byte(corrected), 0o644); err != nil {
		t.Fatal(err)
	}
	repaired, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if ds := validation.Validate(repaired.World); validation.HasErrors(ds) {
		t.Fatalf("repaired Relation key mapping should be valid: %+v", ds)
	}
	if len(repaired.World.Persistence) != 0 || len(repaired.World.Databases) != 0 {
		t.Fatal("semantic relation-key validation must not provision physical storage")
	}
	repairedIndex, err := workspace.BuildSourceIndex(root, repaired)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstModelReference(t, repairedIndex, "semantic-relation", "intake.request.parent", firstModelSemantic)
	if err := index.Validate(func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	}); err == nil {
		t.Fatal("outdated Source Index accepted a repaired relation-key mapping")
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
