package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runCLICommand builds the CLI and runs it as a real process, so the test sees
// the same stdout/exit boundary a caller does.
func runCLICommand(t *testing.T, args ...string) []byte {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("argv=%q failed: %v stderr=%q stdout=%q", args, err, stderr.String(), stdout.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("argv=%q wrote stderr on the success path: %q", args, stderr.String())
	}
	return stdout.Bytes()
}

// An empty reference list must serialize as [] and the whole-index form must
// name the Workspace, so a caller can iterate the index and tell which
// Workspace produced it (#3455, #3456).
func TestSourceIndexWholeIndexUsesArraysAndIdentity(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if os.PathSeparator == '\\' {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tei.workspace.yaml"),
		[]byte("workspace: empty_probe\nformatVersion: 1\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "semantic", "probe.tei"),
		[]byte("namespace probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binary, "source-index", "-workspace", dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("source-index failed: %v stderr=%q", err, stderr.String())
	}

	var envelope struct {
		OK     bool   `json:"ok"`
		Mode   string `json:"mode"`
		Result struct {
			Artifacts  []any `json:"artifacts"`
			References []any `json:"references"`
			Workspace  *struct {
				Name     string `json:"name"`
				Manifest struct {
					SHA256 string `json:"sha256"`
				} `json:"manifest"`
			} `json:"workspace"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("not one JSON object: %v (%q)", err, stdout.String())
	}
	if !envelope.OK || envelope.Mode != "index" {
		t.Fatalf("unexpected envelope: %s", stdout.String())
	}
	if envelope.Result.References == nil {
		t.Fatalf("empty reference list serialized as null: %s", stdout.String())
	}
	if envelope.Result.Artifacts == nil {
		t.Fatalf("empty artifact list serialized as null: %s", stdout.String())
	}
	if envelope.Result.Workspace == nil || envelope.Result.Workspace.Name != "empty_probe" {
		t.Fatalf("Workspace identity missing: %s", stdout.String())
	}
	if envelope.Result.Workspace.Manifest.SHA256 == "" {
		t.Fatalf("manifest fingerprint missing: %s", stdout.String())
	}
}

// Every read command that reports a Workspace read must name the Workspace in
// a machine-readable field, so a caller joining several responses does not
// have to record the path itself (#3455).
func TestReadCommandsReportTheWorkspaceIdentity(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")

	cases := []struct {
		name string
		args []string
	}{
		{"view", []string{"view", "-workspace", reference, "-application", "order_entry", "-format", "json"}},
		{"source-index", []string{"source-index", "-workspace", reference}},
		{"source-index-subject", []string{"source-index", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}},
		{"knowledge-context", []string{"knowledge-context", "-workspace", reference, "-subject-kind", "semantic-data", "-subject-id", "sales.order"}},
		{"requirements", []string{"requirements", "-workspace", reference}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := runCLICommand(t, tc.args...)
			var envelope map[string]any
			if err := json.Unmarshal(out, &envelope); err != nil {
				t.Fatalf("not one JSON object: %v (%q)", err, out)
			}
			if ok, _ := envelope["ok"].(bool); !ok {
				t.Fatalf("call failed: %q", out)
			}
			if path := findWorkspaceIdentity(envelope); path == "" {
				t.Fatalf("no workspace identity in response: %q", out)
			}
		})
	}
}

// findWorkspaceIdentity returns the JSON path of the first object that looks
// like a Workspace identity.
func findWorkspaceIdentity(value any) string {
	m, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	if raw, ok := m["workspace"]; ok {
		if probe, ok := raw.(map[string]any); ok {
			if manifest, ok := probe["manifest"].(map[string]any); ok && manifest["sha256"] != nil {
				return "/workspace"
			}
		}
	}
	for key, child := range m {
		if found := findWorkspaceIdentity(child); found != "" {
			return "/" + key + found
		}
	}
	return ""
}
