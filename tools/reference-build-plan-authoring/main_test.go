package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tei/internal/plugin"
)

// §4.49: the plan this tool writes must be one the builder can consume — it
// validates, it names the module context, and its digests are computed from the
// files rather than declared.

func TestAuthorWritesAPlanThatValidates(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "plan.json")
	code, err := run([]string{
		"-repo", root,
		"-entry", "./tools/reference-go-build-from-plan/testdata/probe",
		"-out", out,
	})
	if err != nil || code != 0 {
		t.Fatalf("author failed: code=%d err=%v", code, err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var plan plugin.BuildPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatalf("the written plan does not decode: %v", err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("the written plan does not validate: %v", err)
	}
	if plan.Recipe.Kind != "go-command" {
		t.Fatalf("recipe = %q, want go-command", plan.Recipe.Kind)
	}

	// The package and the module context, and nothing the command build would not
	// read (§4.46).
	var packages, modules int
	for _, source := range plan.Sources {
		switch source.Kind {
		case "go-package":
			packages++
		case "go-module":
			modules++
		default:
			t.Fatalf("the plan names an unexpected source kind %q", source.Kind)
		}
		if source.Digest == "" {
			t.Fatalf("source %q carries no digest", source.Locator)
		}
	}
	if packages != 1 {
		t.Fatalf("go-package sources = %d, want 1 for a self-contained probe", packages)
	}
	if modules != 2 {
		t.Fatalf("go-module sources = %d, want 2 (go.mod and go.sum)", modules)
	}
}

// A target with a module-local closure names every one of its packages.
func TestAuthorIncludesTheModuleLocalClosure(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "plan.json")
	if code, err := run([]string{"-repo", root, "-entry", "./internal/buildintent", "-out", out}); err != nil || code != 0 {
		t.Fatalf("author failed: code=%d err=%v", code, err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var plan plugin.BuildPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Sources) < 5 {
		t.Fatalf("a package with a closure produced only %d sources", len(plan.Sources))
	}
}

func TestAuthorRequiresAnOutputPath(t *testing.T) {
	if code, err := run([]string{"-repo", ".", "-entry", "."}); err == nil || code == 0 {
		t.Fatal("an attempt without -out was accepted")
	}
}
