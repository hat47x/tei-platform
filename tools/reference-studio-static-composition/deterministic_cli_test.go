package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestPoCBTwoEmptyDirectoriesHaveIdenticalAssetManifest exercises the real
// generator CLIs, rather than only comparing their in-memory return values.
// It normalizes absolute output paths away, but includes the filename and
// content SHA-256 of every emitted asset in the ordered manifest digest.
func TestPoCBTwoEmptyDirectoriesHaveIdenticalAssetManifest(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("Node.js is required for the combined Go/UI reproducibility test: %v", err)
	}
	dirs := []string{t.TempDir(), t.TempDir()}
	const expectedGeneratedGoSHA = "867501c5a98d71337178d8bd7995e2e4eb6d276accd25151a2c18d8cb5adee50"
	var manifests []map[string]string
	var manifestHashes []string
	for _, dir := range dirs {
		initial, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(initial) != 0 {
			t.Fatal("generation destination must initially be empty")
		}
		for _, command := range []*exec.Cmd{
			exec.Command("go", "run", ".", "-fixture", "fixture.json", "-out", dir),
			exec.Command(node, "compose-ui.mjs", "fixture.json", dir),
		} {
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("actual Go/UI generation failed: %v\n%s", err, output)
			}
		}
		files, err := os.ReadDir(dir) // os.ReadDir sorts by filename.
		if err != nil {
			t.Fatal(err)
		}
		want := []string{
			"services.generated.go",
			"services.provenance.json",
			"ui.generated.d.mts",
			"ui.generated.mjs",
			"ui.provenance.json",
		}
		if len(files) != len(want) {
			t.Fatalf("generated asset count: got %d, want %d", len(files), len(want))
		}
		assetHashes := make(map[string]string, len(want))
		var manifest strings.Builder
		for i, file := range files {
			if file.Name() != want[i] || !file.Type().IsRegular() {
				t.Fatalf("unexpected asset %q at position %d", file.Name(), i)
			}
			content, err := os.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(content)
			digest := hex.EncodeToString(sum[:])
			assetHashes[file.Name()] = digest
			fmt.Fprintf(&manifest, "%s\t%s\n", file.Name(), digest)
		}
		if assetHashes["services.generated.go"] != expectedGeneratedGoSHA {
			t.Fatalf("generated Go SHA differs from fixed #1833/#1832 evidence: %s",
				assetHashes["services.generated.go"])
		}
		m := sha256.Sum256([]byte(manifest.String()))
		manifests = append(manifests, assetHashes)
		manifestHashes = append(manifestHashes, hex.EncodeToString(m[:]))
	}
	if !reflect.DeepEqual(manifests[0], manifests[1]) ||
		manifestHashes[0] != manifestHashes[1] {
		t.Fatalf("generation depends on output directory: asset manifests %#v vs %#v; SHA %s vs %s",
			manifests[0], manifests[1], manifestHashes[0], manifestHashes[1])
	}
	t.Logf("PASS: generated Go SHA-256=%s; sorted asset manifest SHA-256=%s; 5 files per empty directory",
		manifests[0]["services.generated.go"], manifestHashes[0])
}
