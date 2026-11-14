package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestArgumentErrorCarriesTheCLIFailureCode pins #3420. A usage mistake over
// MCP reaches the caller as a JSON-RPC -32602, and the CLI reports the same
// mistake as INVALID_ARGUMENT. A caller that switches on failure.code therefore
// reached two different answers for the same mistake and could not compare
// them: it had to parse the message text.
//
// The protocol error now carries the envelope code, so the two transports are
// comparable without string matching.
func TestArgumentErrorCarriesTheCLIFailureCode(t *testing.T) {
	cases := []struct {
		name string
		tool string
		args string
		want string
	}{
		{"a missing required member", "tei_inspect", `{"subjectKind":"application"}`, "subjectId is required"},
		{"an undeclared member", "tei_inspect", `{"subjectKind":"application","subjectId":"x","bogus":1}`, `unknown argument "bogus"`},
		{"an empty required member", "tei_inspect", `{"subjectKind":"application","subjectId":""}`, "must not be an empty string"},
		{"a non-string member", "tei_inspect", `{"subjectKind":1,"subjectId":"x"}`, "must be a string"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var arguments map[string]json.RawMessage
			if err := json.Unmarshal([]byte(c.args), &arguments); err != nil {
				t.Fatalf("unmarshal arguments: %v", err)
			}
			_, err := callTool(json.RawMessage(`{"name":"`+c.tool+`","arguments":`+c.args+`}`), t.TempDir())
			if err == nil {
				t.Fatalf("%q was accepted", c.args)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("message = %q, want it to mention %q", err.Error(), c.want)
			}
			// The message alone is not the contract: the code must travel with it.
			response := handle(rpcRequest{
				JSONRPC: "2.0", ID: json.RawMessage("1"),
				Method: "tools/call",
				Params: json.RawMessage(`{"name":"` + c.tool + `","arguments":` + c.args + `}`),
			}, t.TempDir())
			if response.Error == nil {
				t.Fatal("no protocol error was returned")
			}
			if response.Error.Code != -32602 {
				t.Fatalf("code = %d, want -32602", response.Error.Code)
			}
			code, _ := response.Error.Data["failureCode"].(string)
			if code != "INVALID_ARGUMENT" {
				t.Fatalf("data.failureCode = %q, want INVALID_ARGUMENT so the CLI's "+
					"answer is comparable", code)
			}
		})
	}
}
