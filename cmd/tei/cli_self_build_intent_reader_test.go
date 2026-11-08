package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// These tests pin what SB-C0 §4.13 (I-1a) actually bought: the entry derives the
// declaration's intent fingerprint itself, through the package the reference
// resolver shares, so the record carries a reading the entry made rather than
// one a child reported.
//
// What this adds: an independent basis for a later byte-level comparison. What
// it does NOT add: resolution. No BuildPlan is derived here, so L2-b is
// untouched.

// The entry's own fingerprint equals the reference resolver's derivation for the
// same bytes. Without this, "the entry's reading" could drift from the
// resolver's while both still looked plausible.
func TestIntentDeclarationMatchesTheReferenceDerivation(t *testing.T) {
	dir := t.TempDir()
	fixtureDir := filepath.Join("..", "..", "tools", "reference-build-authoring-intent", "testdata")
	body, err := os.ReadFile(filepath.Join(fixtureDir, "intent.json"))
	if err != nil {
		t.Fatalf("read the reference fixture: %v", err)
	}
	declaration := filepath.Join(dir, "intent.json")
	if err := os.WriteFile(declaration, body, 0o600); err != nil {
		t.Fatal(err)
	}

	record, err := readIntentDeclaration(declaration)
	if err != nil {
		t.Fatalf("the entry refused a declaration the resolver accepts: %v", err)
	}
	if record.Fingerprint == "" {
		t.Fatal("the entry derived no intent fingerprint")
	}
	if record.Digest == "" {
		t.Fatal("the byte digest was not recorded")
	}
	if record.Digest == record.Fingerprint {
		t.Fatal("the byte digest and the declaration fingerprint must be different readings")
	}
	if record.Resolved {
		t.Fatal("the entry reported the declaration as resolved")
	}

	resolver := buildDeclarationResolver(t, dir)
	cmd := exec.Command(resolver,
		"-intent", declaration,
		"-catalog", filepath.Join(fixtureDir, "catalog.json"),
		"-source-root", filepath.Join(fixtureDir, "src"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("reference resolver failed: %v: %s", err, stderr.String())
	}
	var receipt struct {
		IntentFingerprint string `json:"intentFingerprint"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &receipt); err != nil {
		t.Fatalf("receipt is not JSON: %v", err)
	}
	if receipt.IntentFingerprint != record.Fingerprint {
		t.Fatalf("the entry derived %q, the resolver derived %q",
			record.Fingerprint, receipt.IntentFingerprint)
	}
}

// A declaration the shared reader refuses stops the entry: the entry cannot
// describe a declaration it cannot read, and a fingerprint derived from bytes it
// rejected would be a reading it does not stand behind.
func TestIntentDeclarationRefusesWhatTheSharedReaderRefuses(t *testing.T) {
	dir := t.TempDir()
	declaration := filepath.Join(dir, "intent.json")
	// Looks like a declaration but authors a derived value.
	body := `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.weak", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "planFingerprint": "deadbeef",
  "target": "host/linux/amd64"
}`
	if err := os.WriteFile(declaration, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	record, err := readIntentDeclaration(declaration)
	if err == nil {
		t.Fatalf("an authored derived value was accepted: %+v", record)
	}
	if record.Fingerprint != "" {
		t.Fatal("a fingerprint was derived from a declaration the reader refused")
	}
}

// A declaration with no formatVersion is refused: the entry cannot describe what
// it read at all.
func TestIntentDeclarationStillRequiresFormatVersion(t *testing.T) {
	dir := t.TempDir()
	declaration := filepath.Join(dir, "intent.json")
	if err := os.WriteFile(declaration, []byte(`{"subject": {"id": "tei.plugin.x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readIntentDeclaration(declaration); err == nil {
		t.Fatal("a declaration without formatVersion was accepted")
	}
}

func buildDeclarationResolver(t *testing.T, dir string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	binary := filepath.Join(dir, "resolver")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command(goExe, "build", "-trimpath", "-o", binary,
		"../../tools/reference-build-authoring-intent")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build the reference resolver: %v: %s", err, out)
	}
	return binary
}
