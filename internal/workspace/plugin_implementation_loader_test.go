package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func TestPluginImplementationLoaderRejectsMissingPluginDefinition(t *testing.T) {
	root := t.TempDir()
	writePluginImplementationWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: implementation_test
sources:
  - plugin-implementations
`)
	writePluginImplementationWorkspaceFile(t, root, "plugin-implementations/missing/1.0.0/implementation.tei", `plugin-implementation tei.missing v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.missing
    operation execute json { {"attemptId":"a","outcome":"accepted"} }
}
`)

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "has no matching plugin definition") {
		t.Fatalf("expected missing Plugin definition error, got %v", err)
	}
}

func TestPluginImplementationLoaderRejectsCapabilityMismatch(t *testing.T) {
	root := t.TempDir()
	writePluginImplementationWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: implementation_test
sources:
  - plugins
  - plugin-implementations
`)
	writePluginImplementationWorkspaceFile(t, root, "plugins/example/1.0.0/plugin.tei", `plugin tei.example v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.expected {
        action application-start
    }
}
`)
	writePluginImplementationWorkspaceFile(t, root, "plugin-implementations/example/1.0.0/implementation.tei", `plugin-implementation tei.example v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.other
    operation execute json { {"attemptId":"a","outcome":"accepted"} }
}
`)

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "must provide capability") {
		t.Fatalf("expected Capability mismatch error, got %v", err)
	}
}

func writePluginImplementationWorkspaceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPluginImplementationLoaderRejectsPredeclaredArtifact(t *testing.T) {
	root := t.TempDir()
	writePluginImplementationWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: implementation_test
sources:
  - plugins
  - plugin-implementations
`)
	writePluginImplementationWorkspaceFile(t, root, "plugins/example/1.0.0/plugin.tei", `plugin tei.example v1.0.0 {
    runtime builtin
    artifact {
        digest "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    }
    provides deployment-executor runtime.expected {
        action application-start
    }
}
`)
	writePluginImplementationWorkspaceFile(t, root, "plugin-implementations/example/1.0.0/implementation.tei", `plugin-implementation tei.example v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.expected
    operation execute json { {"attemptId":"a","outcome":"accepted"} }
}
`)

	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "artifact-free authoritative Plugin definition") {
		t.Fatalf("expected predeclared artifact to fail, got %v", err)
	}
}

func TestPluginImplementationLoaderRejectsAmbiguousGenericRoot(t *testing.T) {
	root := t.TempDir()
	writePluginImplementationWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: implementation_test
sources:
  - implementations
`)
	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "use plugin-implementations") {
		t.Fatalf("expected ambiguous generic implementation root to fail, got %v", err)
	}
}
