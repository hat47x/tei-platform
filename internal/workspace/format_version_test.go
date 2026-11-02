package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWorkspaceManifestFormatVersion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		src     string
		version int
		wantErr string
	}{
		{name: "version one", src: "workspace: test\nformatVersion: 1\nsources:\n  - semantic\n", version: 1},
		{name: "version missing is transitional", src: "workspace: test\nsources:\n  - semantic\n", version: 0},
		{name: "zero rejected", src: "workspace: test\nformatVersion: 0\nsources:\n  - semantic\n", wantErr: "positive integer"},
		{name: "negative rejected", src: "workspace: test\nformatVersion: -1\nsources:\n  - semantic\n", wantErr: "positive integer"},
		{name: "non numeric rejected", src: "workspace: test\nformatVersion: one\nsources:\n  - semantic\n", wantErr: "positive integer"},
		{name: "duplicate rejected", src: "workspace: test\nformatVersion: 1\nformatVersion: 1\nsources:\n  - semantic\n", wantErr: "duplicate formatVersion"},
		{name: "after sources rejected", src: "workspace: test\nsources:\n  - semantic\nformatVersion: 1\n", wantErr: "before sources and imports"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := parseWorkspaceManifestYAML(tc.src)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want %q, got manifest=%+v err=%v", tc.wantErr, manifest, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if manifest.FormatVersion != tc.version {
				t.Fatalf("version=%d want %d", manifest.FormatVersion, tc.version)
			}
		})
	}
}

func writeFormatVersionWorkspace(t *testing.T, manifest string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte("namespace test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadRejectsUnsupportedWorkspaceFormatVersion(t *testing.T) {
	root := writeFormatVersionWorkspace(t, "workspace: test\nformatVersion: 2\nsources:\n  - semantic\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported workspace formatVersion 2") {
		t.Fatalf("unsupported version was accepted: %v", err)
	}
}

func TestLoadAcceptsVersionOneAndMissingDuringMigration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		manifest string
	}{
		{name: "version one", manifest: "workspace: test\nformatVersion: 1\nsources:\n  - semantic\n"},
		{name: "missing transitional", manifest: "workspace: test\nsources:\n  - semantic\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeFormatVersionWorkspace(t, tc.manifest)
			if _, err := Load(root); err != nil {
				t.Fatalf("Load failed: %v", err)
			}
		})
	}
}
