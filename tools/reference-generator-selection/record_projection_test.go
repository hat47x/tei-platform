package generatorselection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const recordContractJSON = `{
  "schema": "tei.reference.generator-contract/v0",
  "id": "sui.card.record",
  "version": "1",
  "status": "reference-local",
  "inputs": [
    {"attribute": "sui.card.text", "primitive": "std.text"},
    {"attribute": "sui.card.document_id", "primitive": "std.identifier"}
  ],
  "output": {"primitive": "std.text"},
  "projection": {
    "kind": "record",
    "fields": [
      {"name": "text", "input": "text", "attribute": "sui.card.text", "primitive": "std.text"},
      {"name": "documentId", "input": "documentId", "attribute": "sui.card.document_id", "primitive": "std.identifier"}
    ]
  }
}`

func writeRecordContract(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, []byte(recordContractJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateRecordContract(t *testing.T) {
	if _, err := os.Stat(writeRecordContract(t)); err != nil {
		t.Fatal(err)
	}
	// Generate validates the contract as part of rendering.
	if _, err := Generate(writeRecordContract(t), "text-record-template", filepath.Join(t.TempDir(), "candidate")); err != nil {
		t.Fatal(err)
	}
}

func TestRecordTemplateGeneratesFieldProjection(t *testing.T) {
	out := filepath.Join(t.TempDir(), "candidate")
	generated, err := Generate(writeRecordContract(t), "text-record-template", out)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Generator == nil || generated.Manifest.Generator.Strategy != "text-record-template" {
		t.Fatalf("manifest = %+v", generated.Manifest)
	}
	source, err := os.ReadFile(filepath.Join(out, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{`"text"`, `"documentId"`, `json.Marshal(record)`, `"missing"`, `"unresolved"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated source does not contain %s", want)
		}
	}
}

func TestRecordTemplateRequiresRecordContract(t *testing.T) {
	if _, err := Generate(writeTextContract(t), "text-record-template", filepath.Join(t.TempDir(), "candidate")); err == nil {
		t.Fatal("a single-value projection was accepted by text-record-template")
	}
}

func TestTextTemplateRejectsRecordContract(t *testing.T) {
	if _, err := Generate(writeRecordContract(t), "text-template", filepath.Join(t.TempDir(), "candidate")); err == nil {
		t.Fatal("a record projection was accepted by text-template")
	}
}

func TestRecordTemplateGeneratesArrayField(t *testing.T) {
	contract := `{
  "schema": "tei.reference.generator-contract/v0",
  "id": "sui.island.record",
  "version": "4",
  "status": "reference-local",
  "inputs": [{"attribute": "sui.island.card_id", "primitive": "std.identifier"}],
  "output": {"primitive": "std.text"},
  "projection": {
    "kind": "record",
    "fields": [
      {"name": "cardIds", "input": "cardIds", "attribute": "sui.island.card_id", "primitive": "std.identifier", "many": true}
    ]
  }
}`
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, []byte(contract), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "candidate")
	if _, err := Generate(path, "text-record-template", out); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(out, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "make([]string") {
		t.Fatal("generated source does not handle an array field")
	}
}
