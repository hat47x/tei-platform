package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestCatalogNamespaceFailureMatchesTheCLI pins the case where the two
// transports disagreed: `tei_model_catalog` answered `ok:true` with
// namespaceMatched=false while the CLI returned exit 1 with
// NAMESPACE_NOT_FOUND. A caller comparing only `ok` therefore saw a success on
// one transport and a failure on the other (#3420).
func TestCatalogNamespaceFailureMatchesTheCLI(t *testing.T) {
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
	cli := exec.Command(cliPath, "model", "catalog", "-workspace", root, "-namespace", "nope")
	cli.Stdout, cli.Stderr = &cliOut, &cliErr
	runErr := cli.Run()
	if exit, ok := runErr.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		t.Fatalf("CLI did not report the namespace as a finding: %v (%s)", runErr, cliErr.String())
	}
	var cliDoc struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		Error         *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(cliOut.Bytes(), &cliDoc); err != nil {
		t.Fatalf("CLI catalog is not JSON: %v (%q)", err, cliOut.String())
	}
	if cliDoc.OK || cliDoc.Error == nil || cliDoc.Error.Code != "NAMESPACE_NOT_FOUND" {
		t.Fatalf("CLI catalog failure = %+v", cliDoc)
	}

	var mcpOut bytes.Buffer
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_model_catalog","arguments":{"namespace":"nope"}}}` + "\n")
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
		t.Fatal("the adapter reported a missing namespace as a success")
	}
	var mcpDoc struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		Error         *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &mcpDoc); err != nil {
		t.Fatalf("adapter text: %v", err)
	}
	if mcpDoc.SchemaVersion != cliDoc.SchemaVersion || mcpDoc.OK ||
		mcpDoc.Error == nil || mcpDoc.Error.Code != cliDoc.Error.Code ||
		mcpDoc.Error.Message != cliDoc.Error.Message {
		t.Fatalf("answers disagree:\nCLI: %+v\nMCP: %+v", cliDoc, mcpDoc)
	}
}

// A declared namespace without entries stays a success on both transports: the
// projection distinguishes "nothing declared" from "that namespace is not here".
func TestCatalogEmptyNamespaceStaysASuccessOnBothTransports(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	var mcpOut bytes.Buffer
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_model_catalog","arguments":{"namespace":"std"}}}` + "\n")
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
		t.Fatal(err)
	}
	if response.Result.IsError {
		t.Fatalf("a declared namespace was reported as a failure: %q", response.Result.Content[0].Text)
	}
	var doc struct {
		OK     bool `json:"ok"`
		Result struct {
			NamespaceMatched bool `json:"namespaceMatched"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &doc); err != nil {
		t.Fatal(err)
	}
	if !doc.OK || !doc.Result.NamespaceMatched {
		t.Fatalf("declared namespace answered %+v", doc)
	}
}

// inspect and trace already agree on an unknown subject; this keeps that from
// regressing while the catalog case is fixed next to it.
func TestInspectAndTraceSubjectFailureMatchesTheCLI(t *testing.T) {
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

	cases := []struct {
		tool      string
		arguments string
		cliArgs   []string
	}{
		{
			tool:      "tei_inspect",
			arguments: `{"subjectKind":"semantic-data","subjectId":"sales.nope"}`,
			cliArgs:   []string{"inspect", "-workspace", root, "-subject-kind", "semantic-data", "-subject-id", "sales.nope"},
		},
		{
			tool:      "tei_trace",
			arguments: `{"scope":"s","subjectKind":"semantic-data","subjectId":"sales.nope"}`,
			cliArgs:   []string{"trace", "-workspace", root, "-scope", "s", "-subject-kind", "semantic-data", "-subject-id", "sales.nope", "-mode", "summary"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			var cliOut bytes.Buffer
			cli := exec.Command(cliPath, tc.cliArgs...)
			cli.Stdout = &cliOut
			if exit, ok := cli.Run().(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("CLI did not report the subject as a finding: %q", cliOut.String())
			}
			var cliDoc struct {
				OK    bool `json:"ok"`
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(cliOut.Bytes(), &cliDoc); err != nil {
				t.Fatalf("CLI answer is not JSON: %v (%q)", err, cliOut.String())
			}
			if cliDoc.OK || cliDoc.Error == nil || cliDoc.Error.Code != "SUBJECT_NOT_FOUND" {
				t.Fatalf("CLI failure = %+v", cliDoc)
			}

			var mcpOut bytes.Buffer
			request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + tc.tool + `","arguments":` + tc.arguments + `}}` + "\n")
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
				t.Fatal(err)
			}
			if !response.Result.IsError {
				t.Fatal("the adapter reported a missing subject as a success")
			}
			var mcpDoc struct {
				OK    bool `json:"ok"`
				Error *struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &mcpDoc); err != nil {
				t.Fatal(err)
			}
			if mcpDoc.OK || mcpDoc.Error == nil || mcpDoc.Error.Code != cliDoc.Error.Code {
				t.Fatalf("answers disagree:\nCLI: %+v\nMCP: %+v", cliDoc, mcpDoc)
			}
		})
	}
}
