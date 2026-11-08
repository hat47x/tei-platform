package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// A leading UTF-8 BOM is what a Windows editor or PowerShell redirect writes
// when it saves "UTF-8". The manifest parser ignores one BOM at the start (the
// .tei lexer does the same), while the manifest digest stays over the raw
// bytes, so `manifest.sha256` still identifies the file as stored (#3556).
const manifestBOM = "\ufeff"

func TestParseWorkspaceManifestAcceptsOneLeadingBOM(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{name: "without BOM", src: "workspace: bom\nformatVersion: 1\nsources:\n  - semantic\n"},
		{name: "with BOM", src: manifestBOM + "workspace: bom\nformatVersion: 1\nsources:\n  - semantic\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := parseWorkspaceManifestYAML(tc.src)
			if err != nil {
				t.Fatalf("manifest was rejected: %v", err)
			}
			if manifest.Name != "bom" || manifest.FormatVersion != 1 || len(manifest.Sources) != 1 {
				t.Fatalf("manifest parsed incorrectly: %+v", manifest)
			}
		})
	}
}

// A BOM in the middle is not a prefix. It stays an unsupported entry instead of
// being silently stripped or treated as whitespace.
func TestParseWorkspaceManifestRejectsBOMAfterFirstLine(t *testing.T) {
	src := "workspace: bom\n" + manifestBOM + "formatVersion: 1\nsources:\n  - semantic\n"
	if _, err := parseWorkspaceManifestYAML(src); err == nil ||
		!strings.Contains(err.Error(), "unsupported workspace manifest entry") {
		t.Fatalf("inner BOM was accepted: %v", err)
	}
}

func TestLoadAcceptsLeadingBOMInManifestAndTEIFile(t *testing.T) {
	for _, tc := range []struct {
		name        string
		manifestBOM bool
		teiBOM      bool
	}{
		{name: "neither", manifestBOM: false, teiBOM: false},
		{name: "manifest only", manifestBOM: true, teiBOM: false},
		{name: "tei only", manifestBOM: false, teiBOM: true},
		{name: "both", manifestBOM: true, teiBOM: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest := "workspace: bom\nformatVersion: 1\nsources:\n  - semantic\n"
			if tc.manifestBOM {
				manifest = manifestBOM + manifest
			}
			semantic := "namespace test\n"
			if tc.teiBOM {
				semantic = manifestBOM + semantic
			}
			root := writeFormatVersionWorkspace(t, manifest)
			if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(semantic), 0o644); err != nil {
				t.Fatal(err)
			}
			result, err := Load(root)
			if err != nil {
				t.Fatalf("workspace was rejected: %v", err)
			}
			if got := result.Identity().Name; got != "bom" {
				t.Fatalf("workspace name=%q want %q", got, "bom")
			}
			// The digest covers the raw bytes, so a BOM changes it. That is the
			// point: the identity of the file as stored is not the parsed text.
			raw, err := os.ReadFile(filepath.Join(root, "tei.workspace.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Identity().Manifest.SHA256; got != sourceindex.Fingerprint(raw) {
				t.Fatalf("manifest digest %q does not cover the raw manifest bytes", got)
			}
		})
	}
}

// Two workspaces that differ only by the BOM are different files, so they
// report different manifest digests even though both load.
func TestManifestBOMChangesTheReportedDigest(t *testing.T) {
	manifest := "workspace: bom\nformatVersion: 1\nsources:\n  - semantic\n"
	plain := writeFormatVersionWorkspace(t, manifest)
	withBOM := writeFormatVersionWorkspace(t, manifestBOM+manifest)
	plainResult, err := Load(plain)
	if err != nil {
		t.Fatal(err)
	}
	bomResult, err := Load(withBOM)
	if err != nil {
		t.Fatal(err)
	}
	if plainResult.Identity().Manifest.SHA256 == bomResult.Identity().Manifest.SHA256 {
		t.Fatal("manifest digests are equal despite different raw bytes")
	}
}
