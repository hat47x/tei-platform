package designoutput

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	generatorselection "tei/tools/reference-generator-selection"
)

func generatorContract(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repositoryRoot(t), "tools/reference-generator-selection/testdata/line-amount/contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGeneratorBundleBindsBytesAndRemainsAReference(t *testing.T) {
	root, raw := referenceWorkspace(t), generatorContract(t)
	first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	manifest, err := WriteWithGenerator(root, first, "postgres", raw, "template", "7")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteWithGenerator(root, second, "postgres", raw, "template", "7"); err != nil {
		t.Fatal(err)
	}
	files := readTree(t, first)
	if !reflect.DeepEqual(files, readTree(t, second)) {
		t.Fatal("document/source bundle is not byte deterministic")
	}
	for _, artifact := range manifest.Artifacts {
		b, ok := files[artifact.Path]
		if !ok || artifact.Digest != digestOf(b) || artifact.Bytes != len(b) {
			t.Fatalf("artifact binding mismatch: %+v", artifact)
		}
	}
	var candidate generatorselection.CandidateManifest
	if err := json.Unmarshal(files["generated/candidate.json"], &candidate); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(files["generated/contract.json"], raw) || candidate.SourceDigest != digestOf(files["generated/main.go"]) ||
		candidate.Generator == nil || candidate.Generator.Version != "7" || candidate.Generator.ContractDigest != digestOf(raw) || candidate.Generator.TemplateDigest == "" {
		t.Fatalf("candidate provenance not bound to archived bytes: %+v", candidate)
	}
	if !strings.Contains(string(files["index.html"]), "generated/main.go") || !strings.Contains(strings.Join(manifest.Notes, " "), "reference-local") {
		t.Fatal("index or reference limitation missing")
	}
	businessUnsupported := false
	for _, u := range manifest.Unsupported {
		if u.Name == "business-cross-view" {
			businessUnsupported = true
		}
	}
	if !businessUnsupported {
		t.Fatal("generation hid unsupported Business source")
	}

	// Qualification is a separate operation, using the unchanged fixed suite.
	candidates := t.TempDir()
	if err := os.CopyFS(filepath.Join(candidates, "template"), os.DirFS(filepath.Join(first, "generated"))); err != nil {
		t.Fatal(err)
	}
	fixtures := filepath.Join(repositoryRoot(t), "tools/reference-generator-selection/testdata/line-amount/fixtures.json")
	receipt, err := generatorselection.Select(context.Background(), root, filepath.Join(first, "generated/contract.json"), fixtures, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || receipt.WorkspaceRevision != manifest.SourceRevision || receipt.ContractStatus != "reference-local" || receipt.GrantsRuntimeAuthority || receipt.ProvesRealProvider {
		t.Fatalf("separate fixture evidence lost revision or authority boundary: %+v", receipt)
	}
	if len(receipt.Candidates) != 1 || len(receipt.Candidates[0].Results) != 10 {
		t.Fatalf("fixed suite not fully checked: %+v", receipt)
	}
	// Neither successful publication nor selection may overwrite the bundle.
	if _, err := WriteWithGenerator(root, first, "postgres", raw, "template", "7"); err == nil {
		t.Fatal("existing bundle accepted")
	}
	if !reflect.DeepEqual(files, readTree(t, first)) {
		t.Fatal("existing bundle was changed")
	}
}

func TestGeneratorBundleRejectsUnboundOrInvalidContractBeforePublication(t *testing.T) {
	raw := generatorContract(t)
	tests := []struct {
		name              string
		raw               []byte
		strategy, version string
	}{
		{"missing attribute", bytes.ReplaceAll(raw, []byte("sales.order_line.quantity"), []byte("sales.missing.quantity")), "template", "1"},
		{"wrong type", bytes.ReplaceAll(raw, []byte("std.decimal"), []byte("std.money")), "template", "1"},
		{"malformed JSON", []byte("{"), "template", "1"},
		{"unsupported rounding", bytes.ReplaceAll(raw, []byte("half-up"), []byte("half-even")), "template", "1"},
		{"missing contract", nil, "template", "1"},
		{"unknown strategy", raw, "unknown", "1"},
		{"missing strategy", raw, "", "1"},
		{"missing version", raw, "template", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()
			out := filepath.Join(parent, "bundle")
			if _, err := WriteWithGenerator(referenceWorkspace(t), out, "postgres", test.raw, test.strategy, test.version); err == nil {
				t.Fatal("invalid input accepted")
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 0 {
				t.Fatalf("failed generation left published or staging files: %v %v", entries, err)
			}
		})
	}
}
