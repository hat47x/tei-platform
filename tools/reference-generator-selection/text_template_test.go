package generatorselection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const textContractJSON = `{
  "schema": "tei.reference.generator-contract/v0",
  "id": "sui.card.text",
  "version": "1",
  "status": "reference-local",
  "inputs": [{"attribute": "sui.card.text", "primitive": "std.text"}],
  "output": {"primitive": "std.text"},
  "projection": {"kind": "normalize"}
}`

func writeTextContract(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(path, []byte(textContractJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTextTemplateGeneratesProjectionCandidate(t *testing.T) {
	out := filepath.Join(t.TempDir(), "candidate")
	generated, err := Generate(writeTextContract(t), "text-template", out)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Manifest.Generator == nil || generated.Manifest.Generator.Strategy != "text-template" {
		t.Fatalf("manifest = %+v", generated.Manifest)
	}
	source, err := os.ReadFile(filepath.Join(out, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{`"missing"`, `"unresolved"`, `trimSpace(text)`} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated source does not contain %s", want)
		}
	}
}

func TestTextTemplateRejectsMoneyContract(t *testing.T) {
	if _, err := Generate(filepath.Join("testdata", "line-amount", "contract.json"), "text-template", filepath.Join(t.TempDir(), "candidate")); err == nil {
		t.Fatal("a money contract was accepted by text-template")
	}
}

func TestMoneyTemplateRejectsProjectionContract(t *testing.T) {
	if _, err := Generate(writeTextContract(t), "template", filepath.Join(t.TempDir(), "candidate")); err == nil {
		t.Fatal("a projection contract was accepted by the money template")
	}
}
