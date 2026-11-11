package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestMultipleSyntaxFindingsMatchCLI(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"tei.workspace.yaml": "workspace: syntax\nformatVersion: 1\nsources:\n  - semantic\n",
		"semantic/a.tei":     "namespace a\ntype Bad from\n",
		"semantic/b.tei":     "namespace b\ntype Bad from\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cliPath := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", cliPath, "../../cmd/tei").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	cmd := exec.Command(cliPath, "validate", "-workspace", root, "-json")
	var cliOut, cliErr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &cliOut, &cliErr
	err := cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || cliErr.Len() != 0 {
		t.Fatalf("CLI err=%v stderr=%s stdout=%s", err, &cliErr, &cliOut)
	}
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}` + "\n")
	var mcpOut bytes.Buffer
	if err := serve(bytes.NewReader(request), &mcpOut, root); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Content []textContent `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(mcpOut.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("unexpected MCP response: %s", &mcpOut)
	}
	var cliDoc, mcpDoc map[string]any
	if err := json.Unmarshal(cliOut.Bytes(), &cliDoc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &mcpDoc); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cliDoc, mcpDoc) {
		t.Fatalf("CLI and MCP differ: CLI=%v MCP=%v", cliDoc, mcpDoc)
	}
	findings, ok := cliDoc["diagnostics"].([]any)
	incomplete, hasIncomplete := cliDoc["incomplete"].(map[string]any)
	if !ok || len(findings) != 2 || !hasIncomplete || incomplete["reason"] != "syntax-coverage-unverified" || cliDoc["ok"] != false {
		t.Fatalf("unexpected findings: %v", cliDoc)
	}
}
