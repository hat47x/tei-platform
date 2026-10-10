package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
)

// Go dependency absence is structural, not proof that the observed remote
// process or the manifest has passed any TEI acceptance gate.
func TestManifestCaptureHasNoAuthoringDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-json", ".")
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect capture CLI dependencies: %v\n%s", err, out)
	}
	expected := map[string]bool{
		"tei/tools/reference-runtime-snapshot": false,
		"tei/tools/reference-first-web-target-manifest-capture": false,
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p struct {
			ImportPath string
			Module *struct { Main bool }
		}
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode go list: %v", err)
		}
		if p.Module == nil || !p.Module.Main {
			continue
		}
		found, ok := expected[p.ImportPath]
		if !ok || found {
			t.Fatalf("unexpected/duplicate main-module dependency: %s", p.ImportPath)
		}
		expected[p.ImportPath] = true
	}
	for p, seen := range expected {
		if !seen {
			t.Fatalf("expected source package missing: %s", p)
		}
	}
}
