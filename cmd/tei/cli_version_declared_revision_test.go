package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// TestBuildIdentityReportsOnlyADeclaredRevision pins the -ldflags path. A caller
// that builds with -buildvcs=false outside a checkout has no toolchain stamp,
// so it may inject a revision; the executable then says so with its own `source`
// value, and the value never borrows the name `vcsRevision` (#3558).
func TestBuildIdentityReportsOnlyADeclaredRevision(t *testing.T) {
	original := teiDeclaredRevision
	t.Cleanup(func() { teiDeclaredRevision = original })
	teiDeclaredRevision = "8247aab8c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6"

	identity := buildIdentity(&debug.BuildInfo{}, true)
	if identity.Source != "ldflags" {
		t.Fatalf("source=%q, want ldflags", identity.Source)
	}
	if identity.VCSRevision != "" {
		t.Fatalf("a declared revision was reported as toolchain evidence: %q", identity.VCSRevision)
	}
	if identity.DeclaredRevision != "8247aab8c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6" {
		t.Fatalf("declared revision lost: %q", identity.DeclaredRevision)
	}
}

// The toolchain stamp is the stronger evidence: when both are present, the
// declared value is kept but the reported source stays `build-info`.
func TestBuildIdentityKeepsTheToolchainStampStronger(t *testing.T) {
	original := teiDeclaredRevision
	t.Cleanup(func() { teiDeclaredRevision = original })
	teiDeclaredRevision = "declaredonly"

	identity := buildIdentity(&debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"},
	}}, true)
	if identity.Source != "build-info" || identity.VCSRevision == "" {
		t.Fatalf("toolchain evidence was replaced: %+v", identity)
	}
	if identity.DeclaredRevision != "declaredonly" {
		t.Fatalf("declared revision was dropped: %+v", identity)
	}
}

// No injection and no stamp keeps the existing contract: `unavailable`, with
// neither revision field. A declared revision must never appear by default.
func TestBuildIdentityStaysUnavailableWithoutEvidence(t *testing.T) {
	original := teiDeclaredRevision
	t.Cleanup(func() { teiDeclaredRevision = original })
	teiDeclaredRevision = ""

	identity := buildIdentity(nil, false)
	if identity.Source != "unavailable" || identity.VCSRevision != "" || identity.DeclaredRevision != "" {
		t.Fatalf("unexpected identity without evidence: %+v", identity)
	}
}

// The text form must name the declaration as self-reported, so a reader cannot
// mistake it for the toolchain revision it sits next to.
func TestVersionTextSeparatesTheDeclaredRevision(t *testing.T) {
	original := teiDeclaredRevision
	t.Cleanup(func() { teiDeclaredRevision = original })
	teiDeclaredRevision = "abc123"

	var out bytes.Buffer
	if code, err := runVersionCommand(nil, &out); err != nil || code != 0 {
		t.Fatalf("version: code=%d err=%v out=%q", code, err, out.String())
	}
	text := out.String()
	if !strings.Contains(text, "revision=unavailable") {
		t.Fatalf("the toolchain revision line changed: %q", text)
	}
	if !strings.Contains(text, "declared revision: abc123 (self-reported by the builder)") {
		t.Fatalf("the declared revision is not marked as self-reported: %q", text)
	}
}

// The real process must carry the injected value, so the test builds with
// -buildvcs=false and checks both the JSON and the text form.
func TestVersionCommandInABuiltProcessWithInjectedRevision(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	const declared = "8247aab8c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6"
	build := exec.Command("go", "build", "-buildvcs=false",
		"-ldflags=-X main.teiDeclaredRevision="+declared, "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = append(os.Environ(), "GIT_DIR=")
		var stdout bytes.Buffer
		command.Stdout = &stdout
		if err := command.Run(); err != nil {
			t.Fatalf("run %v: %v (%q)", args, err, stdout.String())
		}
		return stdout.String()
	}

	raw := run("version", "-json")
	var identity struct {
		Source           string `json:"source"`
		VCSRevision      string `json:"vcsRevision"`
		DeclaredRevision string `json:"declaredRevision"`
	}
	if err := json.Unmarshal([]byte(raw), &identity); err != nil {
		t.Fatalf("version -json is not one JSON document: %v: %q", err, raw)
	}
	if identity.Source != "ldflags" || identity.DeclaredRevision != declared {
		t.Fatalf("injected identity not reported: %+v", identity)
	}
	if identity.VCSRevision != "" {
		t.Fatalf("a -buildvcs=false build claimed toolchain evidence: %+v", identity)
	}
	if text := run("version"); !strings.Contains(text, "declared revision: "+declared) {
		t.Fatalf("text form lacks the declaration: %q", text)
	}
}

// A value injected into the JSON field must not be mistaken for the toolchain
// stamp when the build also has one: the process case is covered above, and
// this fixes the wire names in one place.
func TestVersionWireFieldNames(t *testing.T) {
	encoded, err := json.Marshal(cliBuildIdentity{
		SchemaVersion:    "tei.cli.version/v0",
		Source:           "ldflags",
		DeclaredRevision: "abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["vcsRevision"]; ok {
		t.Fatalf("an empty vcsRevision was written: %s", encoded)
	}
	if fields["declaredRevision"] != "abc" || fields["source"] != "ldflags" {
		t.Fatalf("wire names drifted: %s", encoded)
	}
}
