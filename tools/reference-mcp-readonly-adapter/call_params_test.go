package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestAdapterRejectsMalformedToolCallParams pins the params/name contract of
// tools/call. Before this, a missing params object reported the decoder's
// "unexpected end of JSON input", and a missing or null name reported
// `unknown tool ""`, which names the wrong thing and hides a client bug
// (#3420).
func TestAdapterRejectsMalformedToolCallParams(t *testing.T) {
	root := referenceWorkspace(t)
	cases := []struct {
		name    string
		request string
		want    string
	}{
		{
			name:    "no params",
			request: `{"jsonrpc":"2.0","id":1,"method":"tools/call"}`,
			want:    "tools/call requires params with a name",
		},
		{
			name:    "params null",
			request: `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":null}`,
			want:    "tools/call requires params with a name, not null",
		},
		{
			name:    "no name",
			request: `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{}}`,
			want:    "tools/call requires a name",
		},
		{
			name:    "name null",
			request: `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":null}}`,
			want:    "tools/call requires a name",
		},
		{
			name:    "name is a number",
			request: `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":7}}`,
			want:    "tools/call params must be an object with a string name",
		},
		{
			name:    "name is empty",
			request: `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":""}}`,
			want:    "tools/call name must not be an empty string",
		},
		{
			name:    "parameters is not an object",
			request: `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":[1,2]}`,
			want:    "tools/call params must be an object with a string name",
		},
		{
			name:    "arguments is an array",
			request: `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"tei_validate","arguments":[]}}`,
			want:    "tools/call arguments must be an object",
		},
		{
			name:    "arguments is a string",
			request: `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"tei_validate","arguments":"x"}}`,
			want:    "tools/call arguments must be an object",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := serve(strings.NewReader(tc.request+"\n"), &out, root); err != nil {
				t.Fatalf("serve: %v", err)
			}
			var response struct {
				Result any `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatalf("response is not JSON: %v (%q)", err, out.String())
			}
			if response.Result != nil {
				t.Fatalf("a malformed call produced a tool result: %q", out.String())
			}
			if response.Error == nil || response.Error.Code != -32602 {
				t.Fatalf("malformed call is not invalid params: %q", out.String())
			}
			if !strings.Contains(response.Error.Message, tc.want) {
				t.Fatalf("message %q does not contain %q", response.Error.Message, tc.want)
			}
			// The old messages must not come back.
			for _, stale := range []string{"unknown tool \"\"", "unexpected end of JSON input", "cannot unmarshal"} {
				if strings.Contains(response.Error.Message, stale) {
					t.Fatalf("message still reports the decoder or an empty tool name: %q", response.Error.Message)
				}
			}
		})
	}
}

// An omitted arguments member and an explicit null are treated the same way: the
// tool runs with no arguments, because both mean "nothing supplied". A
// tool that takes no arguments must not fail for either spelling.
func TestAdapterAcceptsOmittedOrNullArguments(t *testing.T) {
	root := referenceWorkspace(t)
	for _, tc := range []struct{ name, request string }{
		{"omitted", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate"}}`},
		{"null", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":null}}`},
		{"empty object", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":{}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			if err := serve(strings.NewReader(tc.request+"\n"), &out, root); err != nil {
				t.Fatalf("serve: %v", err)
			}
			var response struct {
				Result *struct {
					IsError bool `json:"isError"`
				} `json:"result"`
				Error any `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Error != nil || response.Result == nil {
				t.Fatalf("arguments spelling was rejected: %q", out.String())
			}
			if response.Result.IsError {
				t.Fatalf("a valid validate call failed: %q", out.String())
			}
		})
	}
}
