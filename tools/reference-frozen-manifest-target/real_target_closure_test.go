package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// goTargetClosure observes the actual Go package graph of the two TEI
// executables in this checkout. Unlike synthetic fixtures, this test can
// detect an authoring dependency accidentally pulled into the frozen entry.
func goTargetClosure(t *testing.T, entry string) map[string]bool {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "list", "-deps", "-json", entry)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("observe actual Go closure %s: %v\n%s", entry, err, raw)
	}
	packages := make(map[string]bool)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	for {
		var pkg struct {
			ImportPath string
			Name       string
			Module     *struct{ Main bool }
			Error      *struct{ Err string }
			Incomplete bool
		}
		err := decoder.Decode(&pkg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode actual target closure %s: %v", entry, err)
		}
		if pkg.Error != nil || pkg.Incomplete {
			t.Fatalf("incomplete Go package %s while observing %s", pkg.ImportPath, entry)
		}
		if pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		if packages[pkg.ImportPath] {
			t.Fatalf("duplicate package in %s: %s", entry, pkg.ImportPath)
		}
		packages[pkg.ImportPath] = true
	}
	if len(packages) == 0 {
		t.Fatalf("no TEI packages in actual closure of %s", entry)
	}
	return packages
}

// This is a source-level E0 comparison against the actual First Web Target,
// not an artifact-size, symbol-removal, authorization, or runtime test.
func TestRealFirstWebTargetVsFrozenManifestGoClosure(t *testing.T) {
	first := goTargetClosure(t, "./tools/reference-first-web-target")
	frozen := goTargetClosure(t, "./tools/reference-frozen-manifest-target")

	const frozenEntry = "tei/tools/reference-frozen-manifest-target"
	const snapshot = "tei/tools/reference-runtime-snapshot"
	if len(frozen) != 2 || !frozen[frozenEntry] || !frozen[snapshot] {
		t.Fatalf("frozen runtime gained an unapproved TEI dependency: %v", frozen)
	}
	for _, required := range []string{
		"tei/tools/reference-first-web-target",
		"tei/internal/core",
		"tei/internal/workspace",
		"tei/internal/driver",
	} {
		if !first[required] {
			t.Fatalf("First Web Target missing expected authoring/driver dependency %s: %v", required, first)
		}
		if frozen[required] {
			t.Fatalf("authoring/driver dependency %s leaked into frozen executable", required)
		}
	}
	if first[frozenEntry] || first[snapshot] {
		t.Fatalf("existing First Web Target now imports the frozen Reference: %v", first)
	}
}
