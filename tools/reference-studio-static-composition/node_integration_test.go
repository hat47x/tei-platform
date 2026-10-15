package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func nodeForPoCTest(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js unavailable; run node --test separately")
	}
	return path
}

func TestPoCBActualNodeUITests(t *testing.T) {
	node := nodeForPoCTest(t)
	cmd := exec.Command(node, "--test", "compose-ui.test.mjs", "connect-workbench.test.mjs")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Node UI/adapter tests failed: %v\n%s", err, output)
	}
	t.Logf("Node UI/adapter tests: %s", output)
}

func TestPoCBActualGeneratedUIFilesReproducible(t *testing.T) {
	node := nodeForPoCTest(t)
	dirs := []string{t.TempDir(), t.TempDir()}
	for _, dir := range dirs {
		cmd := exec.Command(node, "compose-ui.mjs", "fixture.json", dir)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("actual CLI generation failed: %v\n%s", err, output)
		}
	}
	for _, name := range []string{"ui.generated.mjs", "ui.generated.d.mts", "ui.provenance.json"} {
		first, err := os.ReadFile(filepath.Join(dirs[0], name))
		if err != nil {
			t.Fatal(err)
		}
		second, err := os.ReadFile(filepath.Join(dirs[1], name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("different bytes for %s", name)
		}
	}
	raw, err := os.ReadFile(filepath.Join(dirs[0], "ui.provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta struct {
		FixtureSHA256      string `json:"fixtureSha256"`
		GeneratedSHA256    string `json:"generatedSha256"`
		DeclarationsSHA256 string `json:"declarationsSha256"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ name, expected string }{
		{"fixture.json", meta.FixtureSHA256},
		{"ui.generated.mjs", meta.GeneratedSHA256},
		{"ui.generated.d.mts", meta.DeclarationsSHA256},
	} {
		path := filepath.Join(dirs[0], item.name)
		if item.name == "fixture.json" {
			path = item.name
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(contents)
		if hex.EncodeToString(sum[:]) != item.expected {
			t.Errorf("incorrect actual digest for %s", item.name)
		}
	}
}
