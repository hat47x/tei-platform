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

// The executable must not pull the TEI authoring, Driver, or runtime
// selection machinery back into a frozen, read-only process. go list is a
// source dependency check, not a linked-binary or security certificate.
func TestFrozenTargetImportsOnlySnapshotWithinTEI(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-json", ".")
	cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect executable source dependencies: %v\n%s", err, out)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	got := make(map[string]bool)
	for {
		var item struct {
			ImportPath string
			Name       string
			Module     *struct{ Main bool }
		}
		err := dec.Decode(&item)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode go list: %v", err)
		}
		if item.Module == nil || !item.Module.Main {
			continue
		}
		if got[item.ImportPath] {
			t.Fatalf("duplicate main-module package %s", item.ImportPath)
		}
		got[item.ImportPath] = true
	}
	for _, expected := range []string{"tei/tools/reference-runtime-snapshot", "tei/tools/reference-frozen-manifest-target"} {
		if !got[expected] {
			t.Fatalf("missing expected main-module package %s", expected)
		}
	}
	if len(got) != 2 {
		t.Fatalf("read-only target imports unexpected main-module packages: %v", got)
	}
}
