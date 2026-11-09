package runtimesnapshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
)

// The frozen-manifest transport must not accidentally reintroduce an import
// of the TEI Workspace, Driver, DSL parser, or other main-module machinery.
// This is a source import-closure guard, not evidence of final linked bytes.
func TestFrozenProjectionHasNoAuthoringPackageDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-json", ".")
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect package import closure: %v\n%s", err, out)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	const expected = "tei/tools/reference-runtime-snapshot"
	found := false
	for {
		var listed struct {
			ImportPath string
			Module     *struct{ Main bool }
		}
		err := dec.Decode(&listed)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode go list: %v", err)
		}
		if listed.Module != nil && listed.Module.Main {
			if listed.ImportPath != expected {
				t.Fatalf("runtime snapshot transitively imports another TEI package %s", listed.ImportPath)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("runtime snapshot's main-module entry was absent from go list")
	}
}
