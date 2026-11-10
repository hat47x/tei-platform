package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/plugin"
)

// The plan says which sources the build reads. This check asks the toolchain
// which packages the build would compile and refuses when a module-local one is
// not covered: otherwise the plan describes a build that is not the one running.
func TestCheckDependencyCoverageRefusesAnUncoveredModulePackage(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goExe, err := execLookPath(t)
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	plan := plugin.BuildPlan{Recipe: plugin.BuildRecipe{Kind: "go-command", Entrypoint: "./internal/buildintent"}}
	// Only a package that does not contain the compiled package is named.
	checked := []checkedSource{{Kind: "go-package", Locator: "tools/reference-go-build-from-plan/testdata/probe"}}
	coverage := checkDependencyCoverage(root, goExe, plan, t.TempDir(), "", checked)
	if !coverage.Checked {
		t.Fatalf("the check did not run: %s", coverage.UnperformedReason)
	}
	if len(coverage.Uncovered) == 0 {
		t.Fatal("a build of a package outside the planned source was reported as covered")
	}
	joined := strings.Join(coverage.Uncovered, "\n")
	if !strings.Contains(joined, "internal/buildintent") {
		t.Fatalf("the uncovered list does not name the compiled package:\n%s", joined)
	}
}

// A package with no module-local dependencies is covered by its own source.
func TestCheckDependencyCoverageAcceptsASelfContainedPackage(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goExe, err := execLookPath(t)
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	entry := "./tools/reference-go-build-from-plan/testdata/probe"
	plan := plugin.BuildPlan{Recipe: plugin.BuildRecipe{Kind: "go-command", Entrypoint: entry}}
	coverage := checkDependencyCoverage(root, goExe, plan, t.TempDir(), "",
		[]checkedSource{{Kind: "go-package", Locator: "tools/reference-go-build-from-plan/testdata/probe"}})
	if !coverage.Checked {
		t.Fatalf("the check did not run: %s", coverage.UnperformedReason)
	}
	if len(coverage.Uncovered) != 0 {
		t.Fatalf("a self-contained package was reported as uncovered: %v", coverage.Uncovered)
	}
}

func execLookPath(t *testing.T) (string, error) {
	t.Helper()
	return exec.LookPath("go")
}
