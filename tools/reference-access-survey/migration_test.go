package main

import (
	"os"
	"path/filepath"
	"testing"
)

// migrated records the tool directories this repository claims are on the
// guarded path: every file there that touches the outside world goes through
// internal/access. The list is a claim, and the test below is what makes it a
// checkable one.
//
// The permission adoption plan says adapters that do not yet use the guarded
// layer must be recorded as such. Without this test the claim lives only in prose:
// a new os.WriteFile added to a "migrated" tool would not fail anything, and the
// prose would go on saying the tool is migrated.
var migrated = []string{
	"tools/reference-first-web-frozen-world-generate",
}

// repoRoot walks up from the test's working directory until it finds go.mod. A
// test that hard-codes "." or a relative path would pass or fail depending on
// where it is run from, which is worse than either outcome.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod found above %s", dir)
		}
		dir = parent
	}
}

// TestMigratedToolsActuallyUseTheGuardedLayer recomputes the survey over each
// declared tool and fails when a file that touches the outside world does not
// import the guarded layer.
//
// It does not check that the guarded layer is *effective* (that is what the
// access tests and the frozen-world build check are for). It checks the weaker,
// mechanical claim the migration list makes: nothing in these tools reaches the
// outside world by a path that bypasses the layer.
func TestMigratedToolsActuallyUseTheGuardedLayer(t *testing.T) {
	root := repoRoot(t)
	for _, tool := range migrated {
		report, err := survey([]string{filepath.Join(root, tool)}, false)
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		for _, file := range report.Files {
			if file.Guarded {
				continue
			}
			t.Errorf("%s touches the outside world without the guarded layer: %s (%v)", tool, file.Path, file.Calls)
		}
	}
}

// TestMigratedToolsAreStillPresent keeps the list from outliving the tools it
// names: a deleted tool that stays declared would make the test pass while
// checking nothing.
func TestMigratedToolsAreStillPresent(t *testing.T) {
	root := repoRoot(t)
	for _, tool := range migrated {
		report, err := survey([]string{filepath.Join(root, tool)}, false)
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		if report.Totals.FilesScanned == 0 {
			t.Errorf("%s is declared migrated but has no Go files", tool)
		}
	}
}
