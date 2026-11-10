package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// callRaw sends one tools/call with the given raw arguments through the same
// serve loop a client uses and returns the decoded response.
func callRaw(t *testing.T, root, name, arguments string) (isError bool, rpcCode int, message string) {
	t.Helper()
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + arguments + `}}` + "\n")
	var out bytes.Buffer
	if err := serve(bytes.NewReader(request), &out, root); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result *struct {
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v: %s", err, out.String())
	}
	if response.Error != nil {
		return false, response.Error.Code, response.Error.Message
	}
	if response.Result == nil {
		t.Fatalf("response has neither result nor error: %s", out.String())
	}
	return response.Result.IsError, 0, ""
}

// An invalid argument must be rejected as invalid params. It must not be
// replaced by a default, and it must not reach the operation (#3420 CAPI-2b).
func TestAdapterRejectsInvalidArgumentsInsteadOfDefaulting(t *testing.T) {
	root := referenceWorkspace(t)
	cases := []struct {
		name, tool, arguments, want string
	}{
		{"subjectKind is a number", "tei_inspect", `{"subjectKind":3,"subjectId":"sales.order"}`, "subjectKind must be a string"},
		{"subjectId is null", "tei_inspect", `{"subjectKind":"entity","subjectId":null}`, "subjectId must be a string"},
		{"subjectKind is empty", "tei_inspect", `{"subjectKind":"","subjectId":"sales.order"}`, "subjectKind must not be an empty string"},
		{"unknown member", "tei_inspect", `{"subjectKind":"entity","subjectId":"sales.order","path":"/etc"}`, `unknown argument "path"`},
		{"workspace override", "tei_model_catalog", `{"workspace":"/tmp/other"}`, `unknown argument "workspace"`},
		{"namespace is a number", "tei_model_catalog", `{"namespace":7}`, "namespace must be a string"},
		{"namespace is null", "tei_model_catalog", `{"namespace":null}`, "namespace must be a string"},
		{"namespace is empty", "tei_model_catalog", `{"namespace":""}`, "namespace must not be an empty string"},
		{"validate takes no arguments", "tei_validate", `{"root":"x"}`, `unknown argument "root"`},
		{"direction is a number", "tei_trace", `{"scope":"reference/workspace","subjectKind":"entity","subjectId":"sales.order","direction":1}`, "direction must be a string"},
		{"mode is empty", "tei_trace", `{"scope":"reference/workspace","subjectKind":"entity","subjectId":"sales.order","mode":""}`, "mode must not be an empty string"},
		{"expectedFingerprint is null", "tei_trace", `{"scope":"reference/workspace","subjectKind":"entity","subjectId":"sales.order","expectedFingerprint":null}`, "expectedFingerprint must be a string"},
		{"limit is a string", "tei_trace", `{"scope":"reference/workspace","subjectKind":"entity","subjectId":"sales.order","limit":"10"}`, "limit must be an integer"},
		{"offset is null", "tei_trace", `{"scope":"reference/workspace","subjectKind":"entity","subjectId":"sales.order","offset":null}`, "offset must be an integer"},
		{"overview limit zero", "tei_model_overview", `{"limit":0}`, "limit must be between 1 and"},
		{"overview unknown member", "tei_model_overview", `{"namespace":"sales"}`, `unknown argument "namespace"`},
		{"locate limit zero", "tei_model_locate", `{"subjectKind":"semantic-data","subjectId":"sales.order","limit":0}`, "limit must be between 1 and"},
		{"locate limit is a string", "tei_model_locate", `{"subjectKind":"semantic-data","subjectId":"sales.order","limit":"5"}`, "limit must be an integer"},
		{"trace unknown member", "tei_trace", `{"scope":"reference/workspace","subjectKind":"entity","subjectId":"sales.order","depth":2}`, `unknown argument "depth"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isError, code, message := callRaw(t, root, tc.tool, tc.arguments)
			if code != -32602 {
				t.Fatalf("rpc code = %d (isError=%v), want -32602", code, isError)
			}
			if !strings.Contains(message, tc.want) {
				t.Fatalf("message = %q, want it to contain %q", message, tc.want)
			}
		})
	}
}

// Leaving an optional member out must keep working; only an explicit invalid
// value is rejected.
func TestAdapterAcceptsOmittedOptionalArguments(t *testing.T) {
	root := referenceWorkspace(t)
	for _, tc := range []struct{ tool, arguments string }{
		{"tei_model_catalog", `{}`},
		{"tei_model_locate", `{"subjectKind":"semantic-data","subjectId":"sales.order"}`},
		{"tei_model_overview", `{}`},
		{"tei_validate", `{}`},
	} {
		isError, code, message := callRaw(t, root, tc.tool, tc.arguments)
		if code != 0 || isError {
			t.Fatalf("%s: code=%d isError=%v message=%q", tc.tool, code, isError, message)
		}
	}
}
