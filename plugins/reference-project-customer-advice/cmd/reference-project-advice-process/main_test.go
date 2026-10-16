package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const testDispatch = "runtime_dispatch.reference_project_advice"

func projectProcessRequest(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(request{
		ProtocolVersion: 1,
		RequestID: "bounded-request",
		Plugin: json.RawMessage(`{"id":"tei.reference.project_customer_advice","version":"1.0.0"}`),
		CapabilityKind: "generic-runtime-dispatch",
		CapabilityID: testDispatch,
		Operation: "evaluate-advice",
		Payload: json.RawMessage(`{"customerCode":"VIP-123","note":""}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProjectProcessEnforcesWholeInputBound(t *testing.T) {
	valid := projectProcessRequest(t)
	// Before the fix the decoder saw only the first 64KiB: a complete valid
	// JSON object followed by a huge whitespace suffix was accepted.
	oversize := append(append([]byte(nil), valid...),
		bytes.Repeat([]byte(" "), (64<<10)+1-len(valid))...)
	for _, tc := range []struct {
		name string
		input []byte
		ok bool
	}{
		{"valid", valid, true},
		{"valid newline", append(append([]byte(nil), valid...), '\n'), true},
		{"exact limit with whitespace", append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), (64<<10)-len(valid))...), true},
		{"oversize whitespace suffix", oversize, false},
		{"oversize second document", append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), (64<<10)-len(valid))...), false},
		{"short extra document", append(append([]byte(nil), valid...), []byte("{}")...), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "oversize second document" {
				tc.input = append(tc.input, []byte("{}")...)
			}
			var out bytes.Buffer
			err := serve(bytes.NewReader(tc.input), &out, []string{"--tei-expected-dispatch=" + testDispatch})
			if !tc.ok {
				if err == nil {
					t.Fatal("malformed or oversized request was accepted")
				}
				if out.Len() != 0 {
					t.Fatal("invalid framing produced a success/error response")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var response response
			if err := json.Unmarshal(out.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Status != "ok" || !strings.Contains(string(response.Payload), "review_suggested") {
				t.Fatalf("valid bounded advisory response = %+v", response)
			}
		})
	}
}
