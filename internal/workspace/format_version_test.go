package workspace

import (
	"fmt"
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
		{name: "version missing parses to zero", src: "workspace: test\nsources:\n  - semantic\n", version: 0},
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
	root := writeFormatVersionWorkspace(t, "workspace: test\nformatVersion: 3\nsources:\n  - semantic\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported workspace formatVersion 3") {
		t.Fatalf("unsupported version was accepted: %v", err)
	}
	if !strings.Contains(err.Error(), "supported: 1, 2") {
		t.Fatalf("supported versions are not named: %v", err)
	}
}

func TestLoadAcceptsWorkspaceFormatVersionTwoWithExplicitSemanticHeader(t *testing.T) {
	root := writeSemanticHeaderWorkspaceWithVersion(t, 2, "formatVersion 2\nnamespace test\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("version 2 workspace was rejected: %v", err)
	}
}

func TestLoadRejectsVersionTwoWorkspaceWithoutSemanticHeader(t *testing.T) {
	root := writeSemanticHeaderWorkspaceWithVersion(t, 2, "namespace test\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "requires an explicit semantic formatVersion header") {
		t.Fatalf("version 2 workspace accepted a headerless semantic file: %v", err)
	}
}

func TestLoadRejectsSemanticVersionThatDiffersFromWorkspaceTwo(t *testing.T) {
	root := writeSemanticHeaderWorkspaceWithVersion(t, 2, "formatVersion 1\nnamespace test\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "does not match the Workspace formatVersion 2") {
		t.Fatalf("version mismatch was accepted: %v", err)
	}
}

func TestLoadAcceptsVersionOne(t *testing.T) {
	root := writeFormatVersionWorkspace(t, "workspace: test\nformatVersion: 1\nsources:\n  - semantic\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
}

func TestLoadRejectsMissingFormatVersion(t *testing.T) {
	root := writeFormatVersionWorkspace(t, "workspace: test\nsources:\n  - semantic\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "missing formatVersion") {
		t.Fatalf("missing format version was accepted: %v", err)
	}
}

func TestLoadAllowsMissingFormatVersionOnlyWithExplicitFrozenOptIn(t *testing.T) {
	t.Setenv("TEI_WORKSPACE_ALLOW_UNSPECIFIED_FORMAT", "1")
	root := writeFormatVersionWorkspace(t, "workspace: test\nsources:\n  - semantic\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("explicit frozen opt-in did not load the legacy manifest: %v", err)
	}
}

func writeSemanticHeaderWorkspace(t *testing.T, semanticBody string) string {
	t.Helper()
	return writeSemanticHeaderWorkspaceWithVersion(t, 1, semanticBody)
}

func writeSemanticHeaderWorkspaceWithVersion(t *testing.T, workspaceVersion int, semanticBody string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf("workspace: test\nformatVersion: %d\nsources:\n  - semantic\n", workspaceVersion)
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), []byte(semanticBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLoadAcceptsMatchingSemanticFormatVersion(t *testing.T) {
	root := writeSemanticHeaderWorkspace(t, "formatVersion 1\nnamespace test\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("matching semantic formatVersion was rejected: %v", err)
	}
}

func TestLoadAcceptsHeaderlessSemanticFileByInheritingTheWorkspaceVersion(t *testing.T) {
	root := writeSemanticHeaderWorkspace(t, "namespace test\n")
	if _, err := Load(root); err != nil {
		t.Fatalf("headerless semantic file did not inherit the workspace version: %v", err)
	}
}

func TestLoadRejectsUnsupportedSemanticFormatVersion(t *testing.T) {
	root := writeSemanticHeaderWorkspace(t, "formatVersion 3\nnamespace test\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "unsupported semantic formatVersion 3") {
		t.Fatalf("unsupported semantic formatVersion was accepted: %v", err)
	}
}
