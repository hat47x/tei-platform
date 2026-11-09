package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const inspectBody = `{
  "ok": true,
  "result": {
    "subject": {"kind": "semantic-attribute", "id": "sales.order.total"},
    "canonical": {"id": "sales.order.total", "type": "std.money"},
    "worldFingerprint": "sha256:world",
    "sources": {"artifacts": [{"path": "semantic/sales/order.tei", "fingerprint": "sha256:f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33"}]}
  }
}`

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func setArgs(t *testing.T, args ...string) {
	t.Helper()
	old := os.Args
	os.Args = append([]string{"probe"}, args...)
	t.Cleanup(func() { os.Args = old })
}

func TestWritesPresenceReportAndKeepsOutputRemovable(t *testing.T) {
	dir := t.TempDir()
	answer := writeFile(t, dir, "answer.md", "total は std.money。定義元は semantic/sales/order.tei。")
	inspect := writeFile(t, dir, "inspect.json", inspectBody)
	out := filepath.Join(dir, "out.json")

	setArgs(t, "-answer", answer, "-inspect", inspect, "-out", out)
	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var doc struct {
		Note   string `json:"note"`
		Report struct {
			Found   int      `json:"found"`
			Missing int      `json:"missing"`
			Limits  []string `json:"limits"`
		} `json:"report"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if doc.Report.Found == 0 {
		t.Fatalf("expected some facts to be found: %+v", doc)
	}
	if len(doc.Report.Limits) == 0 {
		t.Fatal("limits must be present")
	}
}

func TestRefusesExistingOutput(t *testing.T) {
	dir := t.TempDir()
	answer := writeFile(t, dir, "answer.md", "x")
	inspect := writeFile(t, dir, "inspect.json", inspectBody)
	out := writeFile(t, dir, "out.json", "keep me")

	setArgs(t, "-answer", answer, "-inspect", inspect, "-out", out)
	if err := run(); err == nil {
		t.Fatal("an existing output file must be refused")
	}
	if body, _ := os.ReadFile(out); string(body) != "keep me" {
		t.Fatalf("existing file changed: %q", string(body))
	}
}

func TestRequiresInputs(t *testing.T) {
	dir := t.TempDir()
	answer := writeFile(t, dir, "answer.md", "x")
	inspect := writeFile(t, dir, "inspect.json", inspectBody)

	setArgs(t, "-inspect", inspect, "-out", filepath.Join(dir, "a.json"))
	if err := run(); err == nil {
		t.Fatal("missing -answer must be refused")
	}
	setArgs(t, "-answer", answer, "-out", filepath.Join(dir, "b.json"))
	if err := run(); err == nil {
		t.Fatal("missing -inspect must be refused")
	}
	setArgs(t, "-answer", answer, "-inspect", inspect)
	if err := run(); err == nil {
		t.Fatal("missing -out must be refused")
	}
}

func TestRejectsInspectedFailure(t *testing.T) {
	dir := t.TempDir()
	answer := writeFile(t, dir, "answer.md", "x")
	inspect := writeFile(t, dir, "inspect.json", `{"ok": false, "error": {"code": "SUBJECT_NOT_FOUND"}}`)

	setArgs(t, "-answer", answer, "-inspect", inspect, "-out", filepath.Join(dir, "out.json"))
	if err := run(); err == nil {
		t.Fatal("a failed inspect response must be refused")
	}
}
