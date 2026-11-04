package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/workspace"
)

func writeSignFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const providerSemantic = "namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n}\n"

func plainWorkspace(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "ws")
	writeSignFixture(t, filepath.Join(root, "tei.workspace.yaml"), "workspace: plain\nformatVersion: 1\nsources:\n  - semantic\n")
	writeSignFixture(t, filepath.Join(root, "semantic", "os", "process.tei"), providerSemantic)
	return root
}

func importingWorkspace(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	writeSignFixture(t, filepath.Join(base, "provider", "tei.workspace.yaml"), "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	writeSignFixture(t, filepath.Join(base, "provider", "semantic", "os", "process.tei"), providerSemantic)
	consumer := filepath.Join(base, "consumer")
	writeSignFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeSignFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	return consumer
}

func signArgs(ws, key, out string) []string {
	return []string{"sign", "-workspace", ws, "-application", "order_entry", "-decision-ref", "d1",
		"-authority-revision", "r1", "-issuer", "iss", "-key-id", "k1", "-private-key", key, "-out", out}
}

func newKey(t *testing.T) string {
	t.Helper()
	key := filepath.Join(t.TempDir(), "issuer.key")
	if err := run(keygenArgs(key), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestSignRefusesAWorkspaceThatDeclaresImportsAndWritesNoOutput(t *testing.T) {
	key := newKey(t)
	out := filepath.Join(t.TempDir(), "decision.json")
	err := run(signArgs(importingWorkspace(t), key, out), &bytes.Buffer{})
	if err == nil {
		t.Fatal("sign accepted a Workspace that declares imports")
	}
	if !strings.Contains(err.Error(), "SOURCE_INDEX_BUILD_FAILED") || !strings.Contains(err.Error(), "WORKSPACE_IMPORTS_NOT_BOUND") {
		t.Fatalf("message does not name the refusal: %v", err)
	}
	if _, statErr := os.Stat(out); statErr == nil {
		t.Fatal("an output file was written for a refused Workspace")
	}
	raw, _ := os.ReadFile(key)
	if strings.Contains(err.Error(), strings.TrimSpace(string(raw))) || strings.Contains(err.Error(), key) {
		t.Fatal("error exposes private key material or the key path")
	}
}

func readDecision(t *testing.T, path string) hoststore.Decision {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var d hoststore.Decision
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSignSignsLoadTimeBytesEvenIfTheDiskChangesAfterLoad(t *testing.T) {
	key := newKey(t)
	pristine := filepath.Join(t.TempDir(), "a.json")
	if err := run(signArgs(plainWorkspace(t), key, pristine), &bytes.Buffer{}); err != nil {
		t.Fatalf("ordinary Workspace must still sign: %v", err)
	}

	root := plainWorkspace(t)
	orig := loadWorkspace
	t.Cleanup(func() { loadWorkspace = orig })
	loadWorkspace = func(r string) (workspace.Result, error) {
		res, err := workspace.Load(r)
		if err == nil {
			writeSignFixture(t, filepath.Join(r, "semantic", "os", "process.tei"), providerSemantic+"\n// changed after Load\n")
		}
		return res, err
	}
	mutated := filepath.Join(t.TempDir(), "b.json")
	if err := run(signArgs(root, key, mutated), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	a, b := readDecision(t, pristine), readDecision(t, mutated)
	if !reflect.DeepEqual(a.Record.Artifacts, b.Record.Artifacts) {
		t.Fatalf("signed members differ from Load-time bytes:\n%v\n%v", a.Record.Artifacts, b.Record.Artifacts)
	}
}
