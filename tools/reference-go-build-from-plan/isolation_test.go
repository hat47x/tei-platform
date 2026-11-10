package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// §4.39 案B: with -isolate the build runs in a tree made only of the planned
// sources, so nothing can change between the check and the compile. These tests
// pin the two properties that make that worth doing: the staged copy is the
// planned copy, and a change to the working tree after staging cannot reach the
// build.

func TestStageSourcesCopiesExactlyThePlannedBytes(t *testing.T) {
	root := repoRoot(t)
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planFor(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe", "runtime-artifact")
	checked, err := checkSources(root, plan)
	if err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	tree, err := stageSources(root, stage, plan, checked)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if len(tree.Members) != 1 {
		t.Fatalf("members = %d, want 1", len(tree.Members))
	}
	member := tree.Members[0]
	if member.Changed || member.Staged != member.Planned {
		t.Fatalf("the staged copy is not the planned copy: %+v", member)
	}
	// The staged tree keeps the locator, so the tree mirrors the plan rather than
	// flattening it.
	staged := filepath.Join(stage, filepath.FromSlash(locator), "main.go")
	if _, err := os.Stat(staged); err != nil {
		t.Fatalf("the staged tree does not keep the locator (%s): %v", staged, err)
	}
	if len(tree.ModuleFiles) == 0 {
		t.Fatal("the module context was not recorded")
	}
	for _, entry := range tree.ModuleFiles {
		if !strings.HasPrefix(entry, "go.mod:") && !strings.HasPrefix(entry, "go.sum:") {
			t.Fatalf("unexpected module file record: %q", entry)
		}
	}
}

// The staged tree is re-hashed; a source that does not hash to the planned value
// is refused rather than built.
func TestStageSourcesRefusesACopyThatDoesNotMatchThePlan(t *testing.T) {
	root := repoRoot(t)
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planFor(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe", "runtime-artifact")
	checked, err := checkSources(root, plan)
	if err != nil {
		t.Fatal(err)
	}
	// Pretend the plan named a different digest: staging must notice that the copy
	// it made does not hash to what the plan says.
	plan.Sources[0].Digest = "sha256:" + strings.Repeat("d", 64)
	if _, err := stageSources(root, t.TempDir(), plan, checked); err == nil {
		t.Fatal("a copy that does not match the plan was accepted")
	} else if !strings.Contains(err.Error(), "does not hash to the planned digest") {
		t.Fatalf("the refusal does not say what disagreed: %v", err)
	}
}

// The whole point: an isolated build produces the artifact from the staged tree,
// and the report says so.
func TestRunWithIsolateBuildsFromTheStagedTree(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planFor(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe", "runtime-artifact")
	planPath := writePlan(t, dir, plan)

	rep, err := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
		"-isolate",
	)
	if err != nil {
		t.Fatalf("isolated build failed: %v", err)
	}
	if !rep.BuildOK {
		t.Fatalf("buildSucceeded = false: %s", rep.BuildError)
	}
	if rep.IsolatedTree == nil {
		t.Fatal("the report does not say it built from an isolated tree")
	}
	if rep.IsolatedTreeDigest == "" {
		t.Fatal("the report does not record the isolated tree digest")
	}
	if rep.IsolatedTree.Boundary == "" {
		t.Fatal("the isolated tree has no boundary")
	}
	if rep.DependencyCoverage == nil || !rep.DependencyCoverage.Checked {
		t.Fatalf("the coverage check did not run: %+v", rep.DependencyCoverage)
	}
	if len(rep.DependencyCoverage.Uncovered) != 0 {
		t.Fatalf("the staged tree has uncovered packages: %v", rep.DependencyCoverage.Uncovered)
	}
}

var _ = plugin.BuildPlan{}

// §4.43: when the plan names the module context itself, the isolated tree is made
// only of planned sources — nothing is taken from the working tree — so "what was
// checked" and "what was built" are the same tree with no exception.
func TestIsolatedTreeTakesNoModuleFileFromTheWorkingTreeWhenThePlanNamesThem(t *testing.T) {
	root := repoRoot(t)
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planFor(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe", "runtime-artifact")

	// Add the module context as planned sources.
	resolver := buildintent.DirSourceResolver{Root: root}
	for _, name := range []string{"go.mod", "go.sum"} {
		resolved, err := resolver.Resolve(buildintent.IntentSource{Kind: "go-module", ID: name, Ref: name})
		if err != nil {
			t.Fatalf("resolve %s: %v", name, err)
		}
		plan.Sources = append(plan.Sources, plugin.BuildSource{
			Kind: "go-module", ID: name, Locator: name, Digest: resolved.Source.Digest,
		})
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("the plan with a module context does not validate: %v", err)
	}

	dir := t.TempDir()
	planPath := writePlan(t, dir, plan)
	rep, err := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
		"-isolate",
	)
	if err != nil {
		t.Fatalf("isolated build failed: %v", err)
	}
	if !rep.BuildOK {
		t.Fatalf("buildSucceeded = false: %s", rep.BuildError)
	}
	if rep.IsolatedTree == nil {
		t.Fatal("no isolated tree was reported")
	}
	if len(rep.IsolatedTree.ModuleFiles) != 0 {
		t.Fatalf("the isolated tree still took module files from the working tree: %v", rep.IsolatedTree.ModuleFiles)
	}
	// Every staged member is a planned source, module context included.
	if len(rep.IsolatedTree.Members) != 3 {
		t.Fatalf("staged members = %d, want 3 (the package, go.mod, go.sum)", len(rep.IsolatedTree.Members))
	}
}
