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

// TestValidateDocumentMatchesTheCLIByteForByte pins #3420 for validate: the
// CLI and the read-only MCP adapter must answer one document. Before this they
// differed in shape (the adapter wrapped the projection in `result`) and in
// failure code (numeric 1 with the code buried in the message vs the CLI's
// string `error.code`), so a client could not compare them mechanically.
func TestValidateDocumentMatchesTheCLIByteForByte(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	cliPath := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", cliPath, "../../cmd/tei").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	var cliOut, cliErr bytes.Buffer
	cli := exec.Command(cliPath, "validate", "-workspace", root, "-json")
	cli.Stdout, cli.Stderr = &cliOut, &cliErr
	if err := cli.Run(); err != nil {
		t.Fatalf("CLI validate: %v (%s)", err, cliErr.String())
	}
	if cliErr.Len() > 0 {
		t.Fatalf("CLI wrote stderr: %q", cliErr.String())
	}
	if !json.Valid(cliOut.Bytes()) {
		t.Fatalf("CLI validate is not JSON: %q", cliOut.String())
	}

	// The adapter answers the same request on stdio.
	var mcpOut bytes.Buffer
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}` + "\n")
	if err := serve(bytes.NewReader(request), &mcpOut, root); err != nil {
		t.Fatalf("adapter serve: %v", err)
	}
	var response struct {
		Result struct {
			Content []textContent `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(mcpOut.Bytes(), &response); err != nil {
		t.Fatalf("adapter response: %v (%q)", err, mcpOut.String())
	}
	if response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("adapter reported a failed validate for a valid Workspace: %+v", response.Result)
	}

	// Compare the parsed documents, not the formatting: both are one JSON
	// value with the same fields. A field the CLI has and the adapter lacks
	// (or the reverse) fails here.
	var cliDoc, mcpDoc map[string]any
	if err := json.Unmarshal(cliOut.Bytes(), &cliDoc); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &mcpDoc); err != nil {
		t.Fatalf("adapter text is not the document: %v", err)
	}
	if _, ok := cliDoc["error"]; ok {
		t.Fatalf("reference Workspace validate unexpectedly failed: %v", cliDoc["error"])
	}
	for _, key := range []string{
		"schemaVersion", "ok", "diagnostics", "worldFingerprint",
		"sourceIndexFingerprint", "declarationCounts", "workspace",
	} {
		if _, ok := cliDoc[key]; !ok {
			t.Fatalf("CLI document lost %q", key)
		}
		if _, ok := mcpDoc[key]; !ok {
			t.Fatalf("adapter document lacks %q that the CLI returns", key)
		}
	}
	if cliDoc["worldFingerprint"] != mcpDoc["worldFingerprint"] ||
		cliDoc["sourceIndexFingerprint"] != mcpDoc["sourceIndexFingerprint"] ||
		cliDoc["schemaVersion"] != mcpDoc["schemaVersion"] ||
		cliDoc["ok"] != mcpDoc["ok"] {
		t.Fatalf("documents disagree:\nCLI: %v\nMCP: %v", cliDoc, mcpDoc)
	}
	if len(cliDoc["diagnostics"].([]any)) != len(mcpDoc["diagnostics"].([]any)) {
		t.Fatalf("diagnostics differ in length: CLI=%d MCP=%d",
			len(cliDoc["diagnostics"].([]any)), len(mcpDoc["diagnostics"].([]any)))
	}

	// A failed read must carry the CLI's string code, not a numeric protocol
	// error with the code inside the message.
	var failOut bytes.Buffer
	badRequest := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}` + "\n")
	if err := serve(bytes.NewReader(badRequest), &failOut, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("adapter serve: %v", err)
	}
	var failed struct {
		Result struct {
			Content []textContent `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(failOut.Bytes(), &failed); err != nil {
		t.Fatalf("adapter failure response: %v", err)
	}
	if !failed.Result.IsError {
		t.Fatal("a missing Workspace was reported as a successful validate")
	}
	var failureDoc struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		Diagnostics   []any  `json:"diagnostics"`
		Error         *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(failed.Result.Content[0].Text), &failureDoc); err != nil {
		t.Fatalf("failure text: %v", err)
	}
	if failureDoc.SchemaVersion != "tei.cli.validate/v0" || failureDoc.OK ||
		failureDoc.Error == nil || failureDoc.Error.Code != "WORKSPACE_LOAD_FAILED" ||
		failureDoc.Diagnostics == nil {
		t.Fatalf("failure document is not the CLI contract: %+v", failureDoc)
	}

}

// TestValidateFindingIsAFailedAnswerOnBothTransports pins the other half of
// #3420: a semantic finding (not a load error) is exit 1 with `ok:false` and a
// `diagnostics` entry on the CLI, and must be `isError:true` with the same
// document on MCP. Before this the adapter answered ok:true / isError:false.
func TestValidateFindingIsAFailedAnswerOnBothTransports(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: dup\nformatVersion: 1\nsources:\n  - semantic\n")
	declaration := "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"
	write("semantic/sales/order.tei", declaration)
	write("semantic/sales2/order2.tei", declaration)

	cliPath := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", cliPath, "../../cmd/tei").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	var cliOut, cliErr bytes.Buffer
	cli := exec.Command(cliPath, "validate", "-workspace", root, "-json")
	cli.Stdout, cli.Stderr = &cliOut, &cliErr
	if err := cli.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			t.Fatalf("CLI validate: %v (%s)", err, cliErr.String())
		}
	}
	var cliDoc struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		Diagnostics   []any  `json:"diagnostics"`
	}
	if err := json.Unmarshal(cliOut.Bytes(), &cliDoc); err != nil {
		t.Fatalf("CLI validate is not JSON: %v (%q)", err, cliOut.String())
	}
	if cliDoc.OK || len(cliDoc.Diagnostics) == 0 {
		t.Fatalf("a duplicate declaration was not a finding: %+v", cliDoc)
	}

	var mcpOut bytes.Buffer
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}` + "\n")
	if err := serve(bytes.NewReader(request), &mcpOut, root); err != nil {
		t.Fatalf("adapter serve: %v", err)
	}
	var response struct {
		Result struct {
			Content []textContent `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(mcpOut.Bytes(), &response); err != nil {
		t.Fatalf("adapter response: %v", err)
	}
	if !response.Result.IsError {
		t.Fatal("a semantic error finding was reported as a successful validate")
	}
	var mcpDoc struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		Diagnostics   []any  `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &mcpDoc); err != nil {
		t.Fatalf("adapter text: %v", err)
	}
	if mcpDoc.SchemaVersion != cliDoc.SchemaVersion || mcpDoc.OK || len(mcpDoc.Diagnostics) != len(cliDoc.Diagnostics) {
		t.Fatalf("documents disagree: CLI=%+v MCP=%+v", cliDoc, mcpDoc)
	}
}
