package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeRequest(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func readDocument(t *testing.T, path string) document {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read summary: %v", err)
	}
	var doc document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse summary: %v", err)
	}
	return doc
}

// The tool must report the model-input and session-log locations separately,
// because a marker that only appears in one of them is a different finding.
func TestSummariesKeepModelInputAndSessionLogApart(t *testing.T) {
	dir := t.TempDir()
	reqDir := filepath.Join(dir, "requests")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeRequest(t, reqDir, "001-request.json", `{
	  "model": "m", "stream": true,
	  "messages": [{"role": "user", "content": "MODEL_MARKER"}],
	  "tools": [],
	  "dsh_session_log": {"afterSeq": 17, "throughSeq": 25,
	    "events": [{"type": "tool/result", "data": "SESSION_MARKER"}]}
	}`)
	writeRequest(t, reqDir, "002-request.json", `{"model":"m","messages":[],"tools":[]}`)

	out := filepath.Join(dir, "summary.json")
	os.Args = []string{"probe", "-requests", reqDir, "-markers", "MODEL_MARKER,SESSION_MARKER", "-out", out}
	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	doc := readDocument(t, out)
	if doc.Totals["requests"] != 2 {
		t.Fatalf("request count: %+v", doc.Totals)
	}
	if doc.Totals["withSessionLog"] != 1 {
		t.Fatalf("session-log count: %+v", doc.Totals)
	}
	if doc.Totals["withMarkerInModelInput"] != 1 || doc.Totals["withMarkerInSessionLog"] != 1 {
		t.Fatalf("marker counts: %+v", doc.Totals)
	}

	first := doc.Requests[0]
	if first.File != "001-request.json" {
		t.Fatalf("files must be ordered by name: %q", first.File)
	}
	if first.Summary.ModelInputHits["MODEL_MARKER"] != 1 || first.Summary.ModelInputHits["SESSION_MARKER"] != 0 {
		t.Fatalf("model-input hits: %+v", first.Summary.ModelInputHits)
	}
	if first.Summary.SessionLog == nil || first.Summary.SessionLog.MarkerHits["SESSION_MARKER"] != 1 {
		t.Fatalf("session-log hits: %+v", first.Summary.SessionLog)
	}
	if doc.Requests[1].Summary.SessionLog != nil {
		t.Fatalf("second request has no extension: %+v", doc.Requests[1].Summary.SessionLog)
	}
	if doc.Limits["notProven"] == nil {
		t.Fatal("the summary must state what it does not prove")
	}
}

func TestRefusesToOverwriteAnExistingSummary(t *testing.T) {
	dir := t.TempDir()
	reqDir := filepath.Join(dir, "requests")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeRequest(t, reqDir, "001-request.json", `{"model":"m","messages":[],"tools":[]}`)
	out := filepath.Join(dir, "summary.json")
	if err := os.WriteFile(out, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("seed output: %v", err)
	}

	os.Args = []string{"probe", "-requests", reqDir, "-markers", "ANY", "-out", out}
	if err := run(); err == nil {
		t.Fatal("an existing output file must be refused")
	}
	raw, err := os.ReadFile(out)
	if err != nil || string(raw) != "keep me" {
		t.Fatalf("existing file must stay untouched: %q %v", string(raw), err)
	}
}

func TestRequiresMarkersAndRequests(t *testing.T) {
	dir := t.TempDir()
	reqDir := filepath.Join(dir, "requests")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	os.Args = []string{"probe", "-requests", reqDir, "-out", filepath.Join(dir, "s.json")}
	if err := run(); err == nil {
		t.Fatal("missing markers must be refused")
	}
	os.Args = []string{"probe", "-markers", "ANY", "-out", filepath.Join(dir, "s.json")}
	if err := run(); err == nil {
		t.Fatal("missing requests directory must be refused")
	}
}

func TestUnparseableRequestIsReportedNotDropped(t *testing.T) {
	dir := t.TempDir()
	reqDir := filepath.Join(dir, "requests")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeRequest(t, reqDir, "001-request.json", "this is not json")
	out := filepath.Join(dir, "summary.json")
	os.Args = []string{"probe", "-requests", reqDir, "-markers", "ANY", "-out", out}
	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}

	doc := readDocument(t, out)
	if len(doc.Requests) != 1 {
		t.Fatalf("the file must still be listed: %+v", doc.Requests)
	}
	if doc.Requests[0].Problem == "" {
		t.Fatal("the problem must be reported alongside the file")
	}
	if doc.Requests[0].Summary.Bytes == 0 || doc.Requests[0].Summary.SHA256 == "" {
		t.Fatalf("size and digest must survive: %+v", doc.Requests[0].Summary)
	}
}
