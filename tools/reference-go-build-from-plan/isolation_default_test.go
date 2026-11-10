package main

import (
	"path/filepath"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// §4.47: when the plan names its module context, an isolated build needs nothing
// from the working tree, so it is the default. A caller can still turn it off, and
// the report says which tree was compiled either way.

func planWithModuleContext(t *testing.T, root, locator, entrypoint string) plugin.BuildPlan {
	t.Helper()
	plan := planFor(t, root, locator, entrypoint, "runtime-artifact")
	for _, name := range []string{"go.mod", "go.sum"} {
		resolved, err := (buildintent.DirSourceResolver{Root: root}).Resolve(buildintent.IntentSource{
			Kind: "go-module", ID: name, Ref: name,
		})
		if err != nil {
			t.Fatalf("resolve %s: %v", name, err)
		}
		plan.Sources = append(plan.Sources, plugin.BuildSource{
			Kind: "go-module", ID: name, Locator: name, Digest: resolved.Source.Digest,
		})
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("the plan does not validate: %v", err)
	}
	return plan
}

func TestIsolationIsTheDefaultWhenThePlanNamesTheModuleContext(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planWithModuleContext(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe")
	planPath := writePlan(t, dir, plan)

	rep, err := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if !rep.BuildOK {
		t.Fatalf("buildSucceeded = false: %s", rep.BuildError)
	}
	if rep.BuildSource != buildSourcePlanOnly {
		t.Fatalf("buildSource = %q, want %q", rep.BuildSource, buildSourcePlanOnly)
	}
	if rep.IsolatedTree == nil {
		t.Fatal("no isolated tree was reported")
	}
	if len(rep.IsolatedTree.ModuleFiles) != 0 {
		t.Fatalf("the tree took module files from the working tree: %v", rep.IsolatedTree.ModuleFiles)
	}
}

// A plan without the module context cannot be built from the plan alone, so the
// default stays the working tree and the report says so.
func TestIsolationStaysOffWhenThePlanNamesNoModuleContext(t *testing.T) {
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
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if rep.BuildSource != buildSourceWorkingTree {
		t.Fatalf("buildSource = %q, want %q", rep.BuildSource, buildSourceWorkingTree)
	}
}

// The caller can still choose: naming the flag wins over the default.
func TestIsolationCanBeTurnedOffExplicitly(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planWithModuleContext(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe")
	planPath := writePlan(t, dir, plan)

	rep, err := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
		"-isolate=false",
	)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if rep.BuildSource != buildSourceWorkingTree {
		t.Fatalf("buildSource = %q, want %q", rep.BuildSource, buildSourceWorkingTree)
	}
	if rep.IsolatedTree != nil {
		t.Fatal("an isolated tree was staged although isolation was refused")
	}
}
