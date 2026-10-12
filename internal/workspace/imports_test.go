package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func writeExternalFixture(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func externalPair(t *testing.T) (consumer, provider string) {
	t.Helper()
	base := t.TempDir()
	consumer = filepath.Join(base, "consumer")
	provider = filepath.Join(base, "provider")
	writeExternalFixture(t, filepath.Join(provider, "tei.workspace.yaml"), "workspace: provider\nsources:\n  - semantic\n")
	writeExternalFixture(t, filepath.Join(provider, "semantic", "os", "process.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeExternalFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	return consumer, provider
}

func findNamespace(r workspace.Result, namespace string) bool {
	for _, ns := range r.World.Semantics {
		if string(ns.ID) == namespace {
			return true
		}
	}
	return false
}

func TestExternalNamespaceImportPreservesProviderIdentityAndSourceOwnership(t *testing.T) {
	consumer, provider := externalPair(t)
	before, err := workspace.Load(consumer)
	if err != nil {
		t.Fatalf("load explicit imported namespace: %v", err)
	}
	if !findNamespace(before, "os") || !findNamespace(before, "monitoring") {
		t.Fatalf("missing provider or consumer namespace: %#v", before.World.Semantics)
	}
	if len(before.Imports) != 1 || before.Imports[0].Namespace != "os" ||
		before.Imports[0].Path != "../provider" ||
		!strings.HasPrefix(before.Imports[0].SemanticDigest, "sha256:") {
		t.Fatalf("unexpected import provenance: %#v", before.Imports)
	}
	if _, found := before.Sources["semantic:os"]; found {
		t.Fatal("imported provider source must not become a consumer-owned Source")
	}
	if len(before.World.Plugins) != 0 {
		t.Fatal("semantic import must not import a provider's plugin implementations")
	}
	idx, err := workspace.BuildSourceIndex(consumer, before)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range idx.Artifacts {
		if strings.Contains(artifact.Path, "provider") || strings.HasPrefix(artifact.Path, "../") {
			t.Fatalf("provider source must not be indexed as a consumer file: %+v", artifact)
		}
	}
	original := filepath.Join(provider, "semantic", "os", "process.tei")
	moved := filepath.Join(provider, "semantic", "os", "renamed.tei")
	if err := os.Rename(original, moved); err != nil {
		t.Fatal(err)
	}
	after, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if after.Imports[0].SemanticDigest != before.Imports[0].SemanticDigest {
		t.Fatalf("physical source move changed the semantic identity: %v -> %v", before.Imports, after.Imports)
	}
}

func TestExternalNamespaceImportRejectsUnsafeOrAmbiguousProviders(t *testing.T) {
	t.Run("missing-provider", func(t *testing.T) {
		consumer, _ := externalPair(t)
		writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../missing\n    namespace: os\n")
		if _, err := workspace.Load(consumer); err == nil {
			t.Fatal("missing provider was accepted")
		}
	})
	t.Run("namespace-not-owned", func(t *testing.T) {
		consumer, _ := externalPair(t)
		writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: absent\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "not owned") {
			t.Fatalf("undeclared namespace must fail: %v", err)
		}
	})
	t.Run("duplicate-alias", func(t *testing.T) {
		consumer, _ := externalPair(t)
		writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n  - alias: os\n    path: ../provider\n    namespace: second\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "duplicate import alias") {
			t.Fatalf("duplicate alias must fail: %v", err)
		}
	})
	t.Run("conflicting-local-namespace", func(t *testing.T) {
		consumer, _ := externalPair(t)
		writeExternalFixture(t, filepath.Join(consumer, "semantic", "local.tei"),
			"namespace os\n\ntype LocalId from std.identifier\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "conflicts") {
			t.Fatalf("namespace conflict must fail: %v", err)
		}
	})
	t.Run("cycle", func(t *testing.T) {
		consumer, provider := externalPair(t)
		writeExternalFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
			"workspace: provider\nsources:\n  - semantic\nimports:\n  - alias: consumer\n    path: ../consumer\n    namespace: monitoring\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "cyclic Workspace import") {
			t.Fatalf("cycle must fail: %v", err)
		}
	})
	t.Run("source-root-escape-remains-denied", func(t *testing.T) {
		consumer, _ := externalPair(t)
		writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			"workspace: consumer\nsources:\n  - ../provider/semantic\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "Workspace-relative") {
			t.Fatalf("managed-source escape must fail: %v", err)
		}
	})
	t.Run("imported-provider-symlink-escape", func(t *testing.T) {
		consumer, provider := externalPair(t)
		original := filepath.Join(provider, "semantic", "os", "process.tei")
		if err := os.Remove(original); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(filepath.Dir(provider), "outside.tei")
		writeExternalFixture(t, outside, "namespace os\n")
		if err := os.Symlink(outside, original); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "outside the Workspace") {
			t.Fatalf("provider source boundary escape must fail: %v", err)
		}
	})
	t.Run("absolute-import-path", func(t *testing.T) {
		consumer, _ := externalPair(t)
		writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: /tmp/provider\n    namespace: os\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "clean relative") {
			t.Fatalf("absolute import path must fail: %v", err)
		}
	})
}

func TestExternalNamespaceImportDoesNotReexportTransitiveDependency(t *testing.T) {
	consumer, provider := externalPair(t)
	third := filepath.Join(filepath.Dir(consumer), "third")
	writeExternalFixture(t, filepath.Join(third, "tei.workspace.yaml"), "workspace: third\nsources:\n  - semantic\n")
	writeExternalFixture(t, filepath.Join(third, "semantic", "common.tei"), "namespace common\n\ntype CommonId from std.identifier\n")
	writeExternalFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\nimports:\n  - alias: common\n    path: ../third\n    namespace: common\n")
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: common\n    path: ../provider\n    namespace: common\n")
	if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("transitive import must not become implicit provider export: %v", err)
	}
}
