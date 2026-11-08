package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func referenceWorkspace(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("reference workspace: %v", err)
	}
	return root
}

// The adapter must expose exactly the four read-only operations and no other
// tool, so a client cannot reach a write, acceptance, deployment, or grant path
// through it (#3400).
func TestAdapterAdvertisesOnlyReadOnlyOperations(t *testing.T) {
	want := []string{"tei_model_catalog", "tei_inspect", "tei_trace", "tei_validate"}
	tools := toolList()
	if len(tools) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(tools), len(want))
	}
	seen := map[string]bool{}
	for _, tool := range tools {
		seen[tool.Name] = true
		if tool.Description == "" || tool.InputSchema == nil {
			t.Fatalf("%s has no description or schema", tool.Name)
		}
	}
	for _, name := range want {
		if !seen[name] {
			t.Fatalf("missing tool %s", name)
		}
	}
	for name := range seen {
		forbidden := false
		for _, part := range []string{"write", "accept", "deploy", "grant", "execute"} {
			if strings.Contains(name, part) {
				forbidden = true
			}
		}
		if forbidden {
			t.Fatalf("%s looks like a non-read-only tool", name)
		}
	}
}

// A tool call must return the same envelope the CLI writes for the same
// operation. The adapter is not allowed to reshape the answer, so this test
// compares the values that identify the read.
func TestAdapterMatchesTheSameOperationResults(t *testing.T) {
	root := referenceWorkspace(t)

	catalog, err := operations.CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := operations.InspectSubject(root, sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	if err != nil {
		t.Fatal(err)
	}
	validation, err := operations.ValidateWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := operations.TraceWorkspace(root,
		modelref.Key{Scope: "mcp-test", Subject: subject.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}},
		modelref.TraceOptions{Direction: "both", Mode: "summary"})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		request []byte
		check   func(t *testing.T, body map[string]any)
	}{
		{
			name:    "catalog",
			request: []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_model_catalog","arguments":{}}}`),
			check: func(t *testing.T, body map[string]any) {
				result := body["result"].(map[string]any)
				if result["worldFingerprint"] != catalog.WorldFingerprint {
					t.Fatalf("catalog worldFingerprint = %v", result["worldFingerprint"])
				}
				if result["sourceIndexFingerprint"] != catalog.SourceIndexFingerprint {
					t.Fatalf("catalog sourceIndexFingerprint = %v", result["sourceIndexFingerprint"])
				}
				if body["schemaVersion"] != "tei.cli.model/v0" {
					t.Fatalf("catalog schemaVersion = %v", body["schemaVersion"])
				}
			},
		},
		{
			name:    "inspect",
			request: []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"tei_inspect","arguments":{"subjectKind":"semantic-data","subjectId":"sales.order"}}}`),
			check: func(t *testing.T, body map[string]any) {
				result := body["result"].(map[string]any)
				if result["worldFingerprint"] != inspection.WorldFingerprint {
					t.Fatalf("inspect worldFingerprint = %v", result["worldFingerprint"])
				}
				if result["sourceStatus"] != inspection.SourceStatus {
					t.Fatalf("inspect sourceStatus = %v", result["sourceStatus"])
				}
				if body["schemaVersion"] != "tei.cli.inspect/v0" {
					t.Fatalf("inspect schemaVersion = %v", body["schemaVersion"])
				}
			},
		},
		{
			name:    "trace",
			request: []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"tei_trace","arguments":{"scope":"mcp-test","subjectKind":"semantic-relation","subjectId":"sales.order.customer","direction":"both","mode":"summary"}}}`),
			check: func(t *testing.T, body map[string]any) {
				result := body["result"].(map[string]any)
				if result["snapshotFingerprint"] != trace.SnapshotFingerprint {
					t.Fatalf("trace snapshotFingerprint = %v", result["snapshotFingerprint"])
				}
				if result["worldFingerprint"] != trace.WorldFingerprint {
					t.Fatalf("trace worldFingerprint = %v", result["worldFingerprint"])
				}
				if body["schemaVersion"] != "tei.cli.trace/v0" {
					t.Fatalf("trace schemaVersion = %v", body["schemaVersion"])
				}
			},
		},
		{
			name:    "validate",
			request: []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}`),
			check: func(t *testing.T, body map[string]any) {
				// validate is not wrapped in result: the CLI writes its
				// fingerprints, diagnostics and declarations at the top level,
				// and the adapter now writes the same document (#3420).
				if body["worldFingerprint"] != validation.WorldFingerprint {
					t.Fatalf("validate worldFingerprint = %v", body["worldFingerprint"])
				}
				if body["sourceIndexFingerprint"] != validation.SourceIndexFingerprint {
					t.Fatalf("validate sourceIndexFingerprint = %v", body["sourceIndexFingerprint"])
				}
				if body["schemaVersion"] != "tei.cli.validate/v0" {
					t.Fatalf("validate schemaVersion = %v", body["schemaVersion"])
				}
				if _, ok := body["diagnostics"].([]any); !ok {
					t.Fatalf("validate diagnostics is not an array: %T", body["diagnostics"])
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := serve(bytes.NewReader(append(tc.request, '\n')), &out, root); err != nil {
				t.Fatal(err)
			}
			var response struct {
				Result struct {
					Content []textContent `json:"content"`
					IsError bool          `json:"isError"`
				} `json:"result"`
			}
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v (%q)", err, out.String())
			}
			if len(response.Result.Content) != 1 || response.Result.Content[0].Type != "text" {
				t.Fatalf("unexpected tool content: %+v", response.Result.Content)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &body); err != nil {
				t.Fatalf("tool text is not the envelope: %v", err)
			}
			tc.check(t, body)
		})
	}
}

// A rejected read must be reported as an error content, not as a successful
// empty result, so a client cannot mistake an unsupported subject kind for
// "nothing found".
func TestAdapterReportsReadFailures(t *testing.T) {
	root := referenceWorkspace(t)
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_inspect","arguments":{"subjectKind":"not-a-kind","subjectId":"sales.order"}}}` + "\n")
	var out bytes.Buffer
	if err := serve(bytes.NewReader(request), &out, root); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Content []textContent `json:"content"`
			IsError bool          `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Result.IsError {
		t.Fatal("unsupported subject kind was not reported as an error")
	}
	if !strings.Contains(response.Result.Content[0].Text, "UNSUPPORTED_SUBJECT_KIND") {
		t.Fatalf("error text = %q", response.Result.Content[0].Text)
	}
}

// The initialize handshake must name the protocol revision this adapter
// implements, and a notification must not produce a reply.
func TestAdapterHandshakeAndNotifications(t *testing.T) {
	root := referenceWorkspace(t)
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := serve(strings.NewReader(requests), &out, root); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two replies (notification ignored), got %d: %q", len(lines), out.String())
	}
	var init struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &init); err != nil {
		t.Fatal(err)
	}
	if init.Result.ProtocolVersion != protocolVersion {
		t.Fatalf("protocolVersion = %q", init.Result.ProtocolVersion)
	}
	if init.Result.ServerInfo.Name != serverName {
		t.Fatalf("serverInfo.name = %q", init.Result.ServerInfo.Name)
	}
}

// The Workspace is the only read root the adapter can reach: every tool call
// resolves under the path fixed at startup, and no argument names a path.
func TestAdapterFixedsTheWorkspaceRoot(t *testing.T) {
	for _, tool := range toolList() {
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(schema))
		for _, forbidden := range []string{"workspace", "path", "root", "file"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s exposes %q in its input schema: %s", tool.Name, forbidden, schema)
			}
		}
	}
}
