package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The admission rules are tested at the package (internal/threeelement). What
// this pins is that the real command surfaces them as an input failure instead
// of reading the document as v0: the CLI is what a Consumer actually calls, and
// before #6113 it answered ok=true for an unknown version and for a field the
// contract does not define.
//
// The expectations are written from the response contract, not from the
// implementation: exit 2 with MODEL_LOAD_FAILED, because a document that cannot
// be admitted is an argument the command was given, not a structural finding
// inside a model it understood.
func TestThreeElementValidateRefusesAModelItCannotAdmit(t *testing.T) {
	cases := map[string]string{
		"unknown model version": `{
  "schemaVersion": "tei.three-element/v0",
  "effect": "read-only",
  "modelId": "v0-without-model-suffix",
  "name": "Probe",
  "dataElements": [{"id": "order", "name": "Order"}]
}`,
		"field outside the contract": `{
  "schemaVersion": "tei.three-element.model/v0",
  "effect": "read-only",
  "modelId": "extra-field",
  "name": "Probe",
  "presence": "not-fetched",
  "dataElements": [{"id": "order", "name": "Order"}]
}`,
		"no model version": `{
  "effect": "read-only",
  "modelId": "missing-version",
  "name": "Probe",
  "dataElements": [{"id": "order", "name": "Order"}]
}`,
	}
	for name, document := range cases {
		path := writeModelDocument(t, document)
		var out strings.Builder
		code, err := runThreeElementCommand([]string{"validate", "-model", path}, &out)
		if err != nil {
			t.Fatalf("%s: command error: %v", name, err)
		}
		if code != 2 {
			t.Fatalf("%s: exit=%d, want 2 (input the command cannot admit): %s", name, code, out.String())
		}
		body := out.String()
		if !strings.Contains(body, "MODEL_LOAD_FAILED") {
			t.Fatalf("%s: a refused model must be MODEL_LOAD_FAILED, got %s", name, body)
		}
		if strings.Contains(body, `"ok":true`) {
			t.Fatalf("%s: a refused model was admitted: %s", name, body)
		}
	}
}

// The canonical version is still admitted: the rule refuses unknown documents,
// not model files as such.
func TestThreeElementValidateAdmitsTheModelVersionItDeclares(t *testing.T) {
	document := `{
  "schemaVersion": "tei.three-element.model/v0",
  "effect": "read-only",
  "modelId": "canonical",
  "name": "Canonical",
  "dataElements": [{"id": "order", "name": "Order"}]
}`
	path := writeModelDocument(t, document)
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"validate", "-model", path}, &out)
	if err != nil || code != 0 {
		t.Fatalf("exit=%d err=%v body=%s", code, err, out.String())
	}
	if !strings.Contains(out.String(), `"ok":true`) {
		t.Fatalf("a canonical model was not admitted: %s", out.String())
	}
}

// writeModelDocument writes one raw model document and returns its path. The
// document is written byte for byte, because the admission rule is about the
// document a caller actually passes, not about a decoded model.
func writeModelDocument(t *testing.T, document string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model.json")
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
