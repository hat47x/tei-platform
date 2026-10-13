package semanticdistribution

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func writeFixture(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func preparePair(t *testing.T) (consumer, provider string) {
	t.Helper()
	root := t.TempDir()
	consumer, provider = filepath.Join(root, "consumer"), filepath.Join(root, "provider")
	writeFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\n")
	writeFixture(t, filepath.Join(provider, "semantic", "os", "process.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute processId type os.process_id\n}\n")
	writeFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute processId type os.process_id\n}\n")
	return consumer, provider
}

func findNamespace(t *testing.T, ns []semantic.Namespace, name string) semantic.Namespace {
	t.Helper()
	for _, n := range ns {
		if string(n.ID) == name {
			return n
		}
	}
	t.Fatalf("namespace %s is missing", name)
	return semantic.Namespace{}
}

func TestPathAndDataOnlyPackagePreserveCanonicalMeaning(t *testing.T) {
	consumer, providerRoot := preparePair(t)
	pathView, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := workspace.Load(providerRoot)
	if err != nil {
		t.Fatal(err)
	}
	ns := findNamespace(t, provider.World.Semantics, "os")
	digest := pathView.Imports[0].SemanticDigest
	id := Identity{PackageID: "community.os_models", Version: "1.0.0"}
	payload, err := MaterializeReference(id, ns)
	if err != nil {
		t.Fatal(err)
	}
	packageView, err := InspectSelectedReference(payload, Expectation{
		Package: id, NamespaceID: "os", ArtifactDigest: sourceindex.Fingerprint(payload),
		SemanticDigest: digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ns, packageView) {
		t.Fatal("distribution changed the domain model")
	}
	// Remove the path import so that only the explicitly inspected package
	// value supplies the external namespace in this isolated projection.
	writeFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\n")
	result, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	result.World.Semantics = append(result.World.Semantics, packageView)
	workspace.Normalize(&result.World)
	if !reflect.DeepEqual(pathView.World.Semantics, result.World.Semantics) {
		t.Fatal("path and data-only distribution did not produce identical normalized semantic views")
	}
	for _, item := range []struct {
		name string
		view workspace.Result
	}{{"path", pathView}, {"data-only", result}} {
		if diags := validation.Validate(item.view.World); validation.HasErrors(diags) {
			t.Fatalf("%s model failed validation: %+v", item.name, diags)
		}
	}
	if len(result.Imports) != 0 {
		t.Fatal("package projection must not forge a Workspace import or Source Index ownership")
	}
	consumerIndex, err := workspace.BuildSourceIndex(consumer, result)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range consumerIndex.Artifacts {
		if strings.Contains(a.Path, "../") || strings.Contains(a.Path, "provider") {
			t.Fatalf("external source was wrongly imported into the consumer Source Index: %+v", a)
		}
	}
}

func TestDataOnlyReferenceRejectsWrongIdentityOrAlteredBytes(t *testing.T) {
	consumer, provider := preparePair(t)
	loaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	provided, err := workspace.Load(provider)
	if err != nil {
		t.Fatal(err)
	}
	ns := findNamespace(t, provided.World.Semantics, "os")
	id := Identity{PackageID: "community.os_models", Version: "1.0.0"}
	payload, err := MaterializeReference(id, ns)
	if err != nil {
		t.Fatal(err)
	}
	want := Expectation{Package: id, NamespaceID: "os", ArtifactDigest: sourceindex.Fingerprint(payload), SemanticDigest: loaded.Imports[0].SemanticDigest}
	cases := []struct {
		name   string
		change func(*Expectation, *[]byte)
		reason string
	}{
		{"artifact-changed", func(_ *Expectation, data *[]byte) { *data = append(append([]byte(nil), *data...), 'x') }, "artifact digest mismatch"},
		{"wrong-package", func(e *Expectation, _ *[]byte) { e.Package.PackageID = "other.os_models" }, "distribution identity mismatch"},
		{"wrong-version", func(e *Expectation, _ *[]byte) { e.Package.Version = "2.0.0" }, "distribution identity mismatch"},
		{"wrong-namespace", func(e *Expectation, _ *[]byte) { e.NamespaceID = "other" }, "exported namespace mismatch"},
		{"wrong-semantic-digest", func(e *Expectation, _ *[]byte) { e.SemanticDigest = "sha256:" + strings.Repeat("0", 64) }, "semantic digest mismatch"},
		{"missing-selection", func(e *Expectation, _ *[]byte) { e.ArtifactDigest = "" }, "both digests are required"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			expected := want
			data := append([]byte(nil), payload...)
			item.change(&expected, &data)
			if _, err := InspectSelectedReference(data, expected); err == nil || !strings.Contains(err.Error(), item.reason) {
				t.Fatalf("expected %q failure, got: %v", item.reason, err)
			}
		})
	}
	t.Run("same-canonical-id-changed-meaning", func(t *testing.T) {
		other := ns
		other.Types = append([]semantic.Type(nil), ns.Types...)
		other.Types[0].Underlying = "std.text"
		newPayload, err := MaterializeReference(id, other)
		if err != nil {
			t.Fatal(err)
		}
		expected := want
		expected.ArtifactDigest = sourceindex.Fingerprint(newPayload)
		if _, err := InspectSelectedReference(newPayload, expected); err == nil || !strings.Contains(err.Error(), "semantic digest mismatch") {
			t.Fatalf("changed meaning under same ID must fail: %v", err)
		}
	})
	t.Run("unrecognized-runtime-field", func(t *testing.T) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(payload, &raw); err != nil {
			t.Fatal(err)
		}
		raw["runtime"] = json.RawMessage(`{"command":"untrusted"}`)
		altered, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		expected := want
		expected.ArtifactDigest = sourceindex.Fingerprint(altered)
		if _, err := InspectSelectedReference(altered, expected); err == nil || !strings.Contains(err.Error(), "unknown field") {
			t.Fatalf("unexpected executable field must fail: %v", err)
		}
	})
	t.Run("trailing-payload", func(t *testing.T) {
		altered := bytes.Join([][]byte{payload, []byte("{}")}, nil)
		expected := want
		expected.ArtifactDigest = sourceindex.Fingerprint(altered)
		if _, err := InspectSelectedReference(altered, expected); err == nil || !strings.Contains(err.Error(), "trailing content") {
			t.Fatalf("multiple payloads must fail: %v", err)
		}
	})
}
