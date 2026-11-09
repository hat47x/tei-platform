package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// capi2aCLI builds the CLI once for this test file. #3420 CAPI-2a asks for the
// real CLI process and the real adapter process, because a comparison against
// `operations` return values cannot show what a caller of each entrance
// actually receives.
func capi2aCLI(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", path, "../../cmd/tei").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	return path
}

// runCLIProcess runs the CLI and returns the exit code, ok flag and failure
// code, decoded from the one JSON document the entrance writes.
func runCLIProcess(t *testing.T, binary string, args ...string) (int, bool, string, any) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("argv=%q: %v (%s)", args, err, stderr.String())
		}
		code = exit.ExitCode()
	}
	var doc struct {
		OK     bool `json:"ok"`
		Result any  `json:"result"`
		Error  *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("argv=%q is not one JSON document: %v (%q)", args, err, stdout.String())
	}
	failure := ""
	if doc.Error != nil {
		failure = doc.Error.Code
	}
	return code, doc.OK, failure, doc.Result
}

// runMCPProcess starts the adapter on the fixed Workspace and sends one
// tools/call, returning the isError flag, the failure code inside the document
// and the result value.
func runMCPProcess(t *testing.T, root, tool string, arguments map[string]any) (bool, string, any) {
	t.Helper()
	adapter := filepath.Join(t.TempDir(), "tei-mcp")
	if runtime.GOOS == "windows" {
		adapter += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", adapter, ".").CombinedOutput(); err != nil {
		t.Fatalf("build adapter: %v\n%s", err, out)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": arguments},
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(adapter, "-workspace", root)
	cmd.Stdin = bytes.NewReader(append(request, '\n'))
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("adapter: %v (%s)", err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("adapter wrote stderr: %q", stderr.String())
	}
	var response struct {
		Result *struct {
			Content []textContent `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("adapter reply is not JSON: %v (%q)", err, stdout.String())
	}
	if response.Error != nil {
		t.Fatalf("adapter answered a protocol error, not a tool result: %s", stdout.String())
	}
	var doc struct {
		OK     bool `json:"ok"`
		Result any  `json:"result"`
		Error  *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &doc); err != nil {
		t.Fatalf("tool text is not the documented document: %v (%q)", err, response.Result.Content[0].Text)
	}
	failure := ""
	if doc.Error != nil {
		failure = doc.Error.Code
	}
	return response.Result.IsError, failure, doc.Result
}

// The two entrances must answer the same operation with the same value and the
// same failure code, for the successes and the failures #3420 CAPI-2a names.
func TestInspectAndTraceMatchAcrossRealProcesses(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	cli := capi2aCLI(t)

	type fixture struct {
		name      string
		cliArgs   []string
		tool      string
		arguments map[string]any
		wantCode  string
	}
	fixtures := []fixture{
		{
			name:      "inspect success",
			cliArgs:   []string{"inspect", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.order"},
			tool:      "tei_inspect",
			arguments: map[string]any{"subjectKind": "semantic-data", "subjectId": "sales.order"},
		},
		{
			name:    "trace success",
			cliArgs: []string{"trace", "-workspace", root, "-scope", "capi2a", "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-mode", "full"},
			tool:    "tei_trace",
			arguments: map[string]any{"scope": "capi2a", "subjectKind": "semantic-data",
				"subjectId": "sales.order", "mode": "full"},
		},
		{
			name:      "inspect missing subject",
			cliArgs:   []string{"inspect", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "no.such.element"},
			tool:      "tei_inspect",
			arguments: map[string]any{"subjectKind": "semantic-data", "subjectId": "no.such.element"},
			wantCode:  "SUBJECT_NOT_FOUND",
		},
		{
			name:      "inspect unknown kind",
			cliArgs:   []string{"inspect", "-workspace", root, "-subject-kind", "not-a-kind", "-subject-id", "sales.order"},
			tool:      "tei_inspect",
			arguments: map[string]any{"subjectKind": "not-a-kind", "subjectId": "sales.order"},
			wantCode:  "UNSUPPORTED_SUBJECT_KIND",
		},
		{
			name:      "trace missing subject",
			cliArgs:   []string{"trace", "-workspace", root, "-scope", "capi2a", "-subject-kind", "semantic-data", "-subject-id", "no.such.element"},
			tool:      "tei_trace",
			arguments: map[string]any{"scope": "capi2a", "subjectKind": "semantic-data", "subjectId": "no.such.element"},
			wantCode:  "SUBJECT_NOT_FOUND",
		},
		{
			name:      "trace unsupported kind",
			cliArgs:   []string{"trace", "-workspace", root, "-scope", "capi2a", "-subject-kind", "semantic-candidate-key", "-subject-id", "x"},
			tool:      "tei_trace",
			arguments: map[string]any{"scope": "capi2a", "subjectKind": "semantic-candidate-key", "subjectId": "x"},
			wantCode:  "UNSUPPORTED_SUBJECT_KIND",
		},
		{
			name:    "trace stale fingerprint",
			cliArgs: []string{"trace", "-workspace", root, "-scope", "capi2a", "-subject-kind", "semantic-data", "-subject-id", "sales.order", "-expected-fingerprint", "sha256:0000"},
			tool:    "tei_trace",
			arguments: map[string]any{"scope": "capi2a", "subjectKind": "semantic-data",
				"subjectId": "sales.order", "expectedFingerprint": "sha256:0000"},
			wantCode: "STALE_SNAPSHOT",
		},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			_, cliOK, cliCode, cliResult := runCLIProcess(t, cli, tc.cliArgs...)
			mcpIsError, mcpCode, mcpResult := runMCPProcess(t, root, tc.tool, tc.arguments)

			// A failure is a failure on both entrances, and the stable code is
			// the same string. The CLI exit code and the MCP isError flag are
			// the entrance's own way of saying it, so they are not compared.
			if cliOK == mcpIsError {
				t.Fatalf("ok/isError disagree: CLI ok=%v MCP isError=%v", cliOK, mcpIsError)
			}
			if cliCode != mcpCode {
				t.Fatalf("failure code differs: CLI=%q MCP=%q", cliCode, mcpCode)
			}
			if tc.wantCode != "" && cliCode != tc.wantCode {
				t.Fatalf("failure code=%q, want %q", cliCode, tc.wantCode)
			}
			if tc.wantCode == "" {
				// The success value must be identical, not merely similar.
				cliJSON, err := json.Marshal(cliResult)
				if err != nil {
					t.Fatal(err)
				}
				mcpJSON, err := json.Marshal(mcpResult)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(cliJSON, mcpJSON) {
					t.Fatalf("result differs:\nCLI=%s\nMCP=%s", cliJSON, mcpJSON)
				}
			}
		})
	}
}

// A missing required argument must be refused by both entrances, and the CLI
// must say so with one code regardless of which command was called (#3733).
// The remaining difference is only the layer: the CLI answers the stable string
// INVALID_ARGUMENT, the adapter answers JSON-RPC -32602. Both refuse the call
// and neither runs it, so a caller that asks "was it refused" sees the same
// answer. This test pins both answers so the gap cannot widen unnoticed, and
// #3420 still owns the decision to unify the two layers (ADR-0048 §2.3 "失敗").
func TestMissingRequiredArgumentIsRefusedByBothEntrances(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	cli := capi2aCLI(t)

	code, ok, failure, _ := runCLIProcess(t, cli,
		"inspect", "-workspace", root, "-subject-kind", "semantic-data")
	// A missing flag is an argument problem. Before #3733 the CLI answered
	// INVALID_SUBJECT here, which named the wrong thing.
	if code != 2 || ok || failure != "INVALID_ARGUMENT" {
		t.Fatalf("CLI missing id: exit=%d ok=%v code=%q", code, ok, failure)
	}

	adapter := filepath.Join(t.TempDir(), "tei-mcp")
	if runtime.GOOS == "windows" {
		adapter += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", adapter, ".").CombinedOutput(); err != nil {
		t.Fatalf("build adapter: %v\n%s", err, out)
	}
	var stdout bytes.Buffer
	cmd := exec.Command(adapter, "-workspace", root)
	cmd.Stdin = strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_inspect","arguments":{"subjectKind":"semantic-data"}}}` + "\n")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("adapter: %v", err)
	}
	var response struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("adapter reply is not JSON: %v (%q)", err, stdout.String())
	}
	if response.Result != nil {
		t.Fatalf("adapter ran the call without the required id: %s", stdout.String())
	}
	if response.Error == nil || response.Error.Code != -32602 {
		t.Fatalf("adapter missing id: %s", stdout.String())
	}
}
