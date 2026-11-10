package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestOversizedRequestDoesNotStopTheLoop pins the read path. bufio.Scanner
// returned ErrTooLong, which ended serve and exited the process, so one
// oversized call stopped the adapter for every later request (#3420). The
// request is now refused as invalid (-32600) and the loop continues.
func TestOversizedRequestDoesNotStopTheLoop(t *testing.T) {
	root := referenceWorkspace(t)
	padding := strings.Repeat("x", maxRequestBytes+1024)
	oversized := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":{"pad":"` + padding + `"}}}` + "\n"
	// A well-formed request after the oversized one must still be answered.
	following := `{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n"

	var out bytes.Buffer
	if err := serve(strings.NewReader(oversized+following), &out, root); err != nil {
		t.Fatalf("serve stopped on an oversized request: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected a refusal and a ping reply, got %d: %q", len(lines), truncate(out.String()))
	}
	var refusal struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &refusal); err != nil {
		t.Fatal(err)
	}
	if refusal.Error == nil || refusal.Error.Code != -32600 {
		t.Fatalf("oversized request was not refused as invalid: %q", lines[0])
	}
	if !strings.Contains(refusal.Error.Message, "4 MiB") {
		t.Fatalf("refusal does not state the limit: %q", lines[0])
	}
	var ping struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &ping); err != nil {
		t.Fatal(err)
	}
	if ping.Result == nil {
		t.Fatalf("the request after the oversized one was not answered: %q", lines[1])
	}
}

// A line just under the limit must still be read whole, so the bound is a limit
// on one line, not on the request stream.
func TestRequestAtTheLimitIsRead(t *testing.T) {
	root := referenceWorkspace(t)
	payload := strings.Repeat("y", maxRequestBytes-200)
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tei_validate","arguments":{"pad":"` + payload + `"}}}` + "\n"
	if len(request) > maxRequestBytes+1 {
		t.Fatalf("fixture is larger than the limit: %d", len(request))
	}
	var out bytes.Buffer
	if err := serve(strings.NewReader(request), &out, root); err != nil {
		t.Fatalf("serve: %v", err)
	}
	var response struct {
		Result *struct {
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil {
		t.Fatalf("response is not JSON: %v (%q)", err, truncate(out.String()))
	}
	// The padding member is unknown, so the tool refuses it; the point is that
	// the line was read and produced a tool answer, not a -32600 line refusal.
	if response.Error != nil && response.Error.Code == -32600 {
		t.Fatalf("a line under the limit was refused as oversized: %q", truncate(out.String()))
	}
	if response.Result == nil {
		// The unknown member is refused inside the tool, which means the line
		// was read and dispatched: a -32600 line refusal would be the failure.
		var refusal struct {
			Error *struct {
				Code int `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(out.Bytes(), &refusal); err != nil || refusal.Error == nil {
			t.Fatalf("no tool answer and no error: %q", truncate(out.String()))
		}
	}
}

// An empty request stream is a clean end, not an error.
func TestEmptyStreamEndsCleanly(t *testing.T) {
	var out bytes.Buffer
	if err := serve(strings.NewReader(""), &out, referenceWorkspace(t)); err != nil {
		t.Fatalf("an empty stream produced an error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("an empty stream wrote a reply: %q", out.String())
	}
}

func truncate(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "…"
}
