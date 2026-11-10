package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// These cases follow JSON-RPC 2.0 and the MCP 2024-11-05 base protocol, not
// the adapter's current output: a request id is a string or an integer (MCP
// forbids null), a malformed line is answered once and the loop keeps going,
// and a notification is never answered (#3420 item 4).

type wireResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func serveLines(t *testing.T, lines ...string) []wireResponse {
	t.Helper()
	var out bytes.Buffer
	input := strings.Join(lines, "\n") + "\n"
	if err := serve(strings.NewReader(input), &out, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var responses []wireResponse
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var response wireResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("response line is not JSON: %v: %s", err, line)
		}
		responses = append(responses, response)
	}
	return responses
}

const followingPing = `{"jsonrpc":"2.0","id":99,"method":"ping"}`

func TestMalformedEnvelopeIsRefusedAndLoopContinues(t *testing.T) {
	cases := []struct {
		name string
		line string
		code int
	}{
		{"invalid JSON", `{"jsonrpc":"2.0","id":1,"method":`, -32700},
		{"top-level array", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, -32600},
		{"top-level number", `42`, -32600},
		{"top-level string", `"ping"`, -32600},
		{"missing jsonrpc", `{"id":1,"method":"ping"}`, -32600},
		{"jsonrpc 1.0", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, -32600},
		{"jsonrpc not a string", `{"jsonrpc":2,"id":1,"method":"ping"}`, -32600},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, -32600},
		{"object id", `{"jsonrpc":"2.0","id":{"a":1},"method":"ping"}`, -32600},
		{"array id", `{"jsonrpc":"2.0","id":[1],"method":"ping"}`, -32600},
		{"boolean id", `{"jsonrpc":"2.0","id":true,"method":"ping"}`, -32600},
		{"missing method", `{"jsonrpc":"2.0","id":1}`, -32600},
		{"method not a string", `{"jsonrpc":"2.0","id":1,"method":7}`, -32600},
		{"empty method", `{"jsonrpc":"2.0","id":1,"method":""}`, -32600},
		{"params is a string", `{"jsonrpc":"2.0","id":1,"method":"ping","params":"x"}`, -32600},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			responses := serveLines(t, tc.line, followingPing)
			if len(responses) != 2 {
				t.Fatalf("want one refusal and one ping answer, got %d responses", len(responses))
			}
			if responses[0].Error == nil || responses[0].Error.Code != tc.code {
				t.Fatalf("first response: want error %d, got %+v", tc.code, responses[0])
			}
			if responses[1].Error != nil || string(responses[1].ID) != "99" {
				t.Fatalf("the loop must keep serving: %+v", responses[1])
			}
		})
	}
}

func TestInvalidIDIsAnsweredWithNullID(t *testing.T) {
	// An id that is not a string or integer cannot be echoed back.
	for _, line := range []string{
		`{"jsonrpc":"2.0","id":{"a":1},"method":"ping"}`,
		`{"jsonrpc":"2.0","id":null,"method":"ping"}`,
	} {
		responses := serveLines(t, line)
		if len(responses) != 1 || string(responses[0].ID) != "null" {
			t.Fatalf("%s: want one response with id null, got %+v", line, responses)
		}
	}
}

func TestStringAndIntegerIDsAreEchoed(t *testing.T) {
	responses := serveLines(t,
		`{"jsonrpc":"2.0","id":"abc","method":"ping"}`,
		`{"jsonrpc":"2.0","id":7,"method":"ping"}`,
	)
	if len(responses) != 2 || string(responses[0].ID) != `"abc"` || string(responses[1].ID) != "7" {
		t.Fatalf("ids must be echoed unchanged: %+v", responses)
	}
	for _, response := range responses {
		if response.Error != nil {
			t.Fatalf("a valid ping must succeed: %+v", response)
		}
	}
}

func TestNotificationsAreNeverAnswered(t *testing.T) {
	responses := serveLines(t,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","method":"no/such/method"}`,
		`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"tei_validate","arguments":{}}}`,
		followingPing,
	)
	if len(responses) != 1 || string(responses[0].ID) != "99" {
		t.Fatalf("only the ping may be answered: %+v", responses)
	}
}

func TestToolsCallWithoutUsableParamsIsInvalidParams(t *testing.T) {
	for name, line := range map[string]string{
		"params missing":      `{"jsonrpc":"2.0","id":1,"method":"tools/call"}`,
		"params is an array":  `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":[]}`,
		"name missing":        `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{}}}`,
		"name is not string":  `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":5}}`,
		"unknown tool":        `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope","arguments":{}}}`,
		"arguments is a list": `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":[]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			responses := serveLines(t, line)
			if len(responses) != 1 || responses[0].Error == nil || responses[0].Error.Code != -32602 {
				t.Fatalf("want one -32602 response, got %+v", responses)
			}
		})
	}
}
