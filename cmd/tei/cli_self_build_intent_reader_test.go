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

// These tests pin SB-C0 §4.13 (I-1a): the entry now derives the declaration's
// intent fingerprint itself, through the package the reference resolver shares.
//
// What this adds: the record carries a reading the entry made, not only one a
// child reported. What it does NOT add: resolution. No BuildPlan is derived, so
// L2-b is untouched.

type declarationResult struct {
	OK           bool `json:"ok"`
	Qualified    bool `json:"qualified"`
	IntentRecord *struct {
		Path              string `json:"path"`
		Digest            string `json:"digest"`
		IntentFingerprint string `json:"intentFingerprint"`
		StrictlyRead      bool   `json:"strictlyRead"`
		StrictReadFailed  string `json:"strictReadFailure"`
		FormatVersion     int    `json:"formatVersion"`
		SubjectID         string `json:"subjectId"`
		Resolved          bool   `json:"resolved"`
	} `json:"intentDeclaration"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// The entry's own fingerprint equals the reference resolver's derivation for the
// same bytes.
func TestIntentDeclarationMatchesTheReferenceDerivation(t *testing.T) {
	dir := t.TempDir()
	declaration := filepath.Join(dir, "intent.json")
	body, err := os.ReadFile(filepath.Join("..", "..", "tools", "reference-build-authoring-intent", "testdata", "intent.json"))
	if err != nil {
		t.Fatalf("read the reference fixture: %v", err)
	}
	if err := os.WriteFile(declaration, body, 0o600); err != nil {
		t.Fatal(err)
	}

	record, err := readIntentDeclaration(declaration)
	if err != nil {
		t.Fatalf("the entry refused a declaration the resolver accepts: %v", err)
	}
	if !record.StrictlyRead {
		t.Fatalf("the shared reader refused the fixture: %s", record.StrictReadFailedNote)
	}
	if record.IntentFingerprint == "" {
		t.Fatal("the entry derived no intent fingerprint")
	}
	if record.Resolved {
		t.Fatal("the entry reported the declaration as resolved")
	}

	// The same bytes, through the reference binary, must agree.
	resolver := buildDeclarationResolver(t, dir)
	cmd := exec.Command(resolver,
		"-intent", declaration,
		"-catalog", filepath.Join("..", "..", "tools", "reference-build-authoring-intent", "testdata", "catalog.json"),
		"-source-root", filepath.Join("..", "..", "tools", "reference-build-authoring-intent", "testdata", "src"))
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
	if receipt.IntentFingerprint != record.IntentFingerprint {
		t.Fatalf("the entry derived %q, the resolver derived %q",
			record.IntentFingerprint, receipt.IntentFingerprint)
	}
}

// A declaration the strict reader refuses is still recorded, with the reason and
// no fingerprint, rather than being read as agreement.
func TestIntentDeclarationRecordsWhyAStrictReadFailed(t *testing.T) {
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
	if err != nil {
		t.Fatalf("the entry rejected the file outright instead of recording the refusal: %v", err)
	}
	if record.StrictlyRead {
		t.Fatal("an authored derived value was accepted by the strict reader")
	}
	if record.IntentFingerprint != "" {
		t.Fatal("a fingerprint was derived from a declaration the reader refused")
	}
	if record.StrictReadFailedNote == "" {
		t.Fatal("the strict-read refusal had no stated reason")
	}
	if record.Digest == "" {
		t.Fatal("the byte digest must still be recorded: the entry held those bytes")
	}
}

// A declaration with no formatVersion is still a hard error: the entry cannot
// describe what it read at all.
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
