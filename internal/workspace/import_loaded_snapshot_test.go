package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// A selected provider Namespace has a concrete parsed-byte witness from its
// ORIGINAL nested Load. The witness stays unchanged after the physical files
// move or change; a fresh navigation observation is a separate operation.
func TestImportedParsedLoadSnapshotRetainsSameLoadBytesWithoutReread(t *testing.T) {
	consumer, provider := importedEvidenceFixturePair(t)
	manifestPath := filepath.Join(provider, "tei.workspace.yaml")
	parsedPath := filepath.Join(provider, "semantic/os/process.tei")
	secondPath := filepath.Join(provider, "semantic/os/thread.tei")
	unrelatedPath := filepath.Join(provider, "semantic/unrelated.tei")
	writeImportedEvidenceFixture(t, secondPath, "namespace os\n\ntype ThreadId from std.identifier\n")
	writeImportedEvidenceFixture(t, unrelatedPath, "namespace unrelated\n\ntype SecretId from std.identifier\n")

	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	parsedBefore, err := os.ReadFile(parsedPath)
	if err != nil {
		t.Fatal(err)
	}
	secondBefore, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Imports) != 1 {
		t.Fatalf("expected one imported Namespace: %+v", loaded.Imports)
	}
	// A non-exported byte slice in a public Result can still be printed by
	// fmt's ordinary structural formatter. The NEW provider witness must not
	// appear in a generic Result log just because a Semantic Import succeeded.
	formatted := fmt.Sprintf("%+v", loaded)
	for _, raw := range [][]byte{manifestBefore, parsedBefore, secondBefore} {
		if strings.Contains(formatted, fmt.Sprint(raw)) {
			t.Fatal("formatted Result contains a raw Provider Source or Manifest byte sequence")
		}
	}
	evidence, read, err := importedParsedLoadSnapshot(loaded, "os")
	if err != nil {
		t.Fatal(err)
	}
	// Source/Manifest paths and fingerprints are potentially sensitive.
	// No new public accessor may disclose even their metadata merely because
	// a Semantic Namespace Import was permitted. The prior ResolvedImport
	// contract remains intact, but the private same-Load evidence is absent.
	encoded, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, privatePart := range []string{
		"semantic/os/process.tei",
		"semantic/os/thread.tei",
		"semantic/unrelated.tei",
		"workspace: provider",
		"FullInputClosure",
		"ParsedSources",
		evidence.Manifest.SHA256,
	} {
		if bytes.Contains(encoded, []byte(privatePart)) {
			t.Fatalf("public Result serialized private provider evidence %q", privatePart)
		}
	}
	for _, artifact := range evidence.ParsedSources {
		if bytes.Contains(encoded, []byte(artifact.Fingerprint)) {
			t.Fatalf("public Result serialized private source fingerprint for %q", artifact.Path)
		}
	}
	if loaded.Imports[0].Path != "../provider" {
		t.Fatal("existing public ResolvedImport projection unexpectedly changed")
	}
	if evidence.Alias != "os" || evidence.ProviderPath != "../provider" ||
		evidence.Namespace != "os" || evidence.SemanticDigest != loaded.Imports[0].SemanticDigest ||
		evidence.Manifest.Path != "tei.workspace.yaml" ||
		evidence.Manifest.SHA256 != sourceindex.Fingerprint(manifestBefore) ||
		evidence.FullInputClosure {
		t.Fatalf("same-Load witness has incorrect scope or falsely claims full closure: %+v", evidence)
	}
	expectedArtifacts := []sourceindex.Artifact{}
	for _, example := range []struct {
		path string
		raw  []byte
	}{
		{"semantic/os/process.tei", parsedBefore},
		{"semantic/os/thread.tei", secondBefore},
	} {
		artifact, err := sourceindex.NewArtifact(example.path, example.raw)
		if err != nil {
			t.Fatal(err)
		}
		expectedArtifacts = append(expectedArtifacts, artifact)
	}
	if !reflect.DeepEqual(evidence.ParsedSources, expectedArtifacts) {
		t.Fatalf("selected Namespace's source membership differs: got=%+v want=%+v", evidence.ParsedSources, expectedArtifacts)
	}
	assertRead := func(path string, want []byte) {
		t.Helper()
		got, err := read(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("captured Source %q differs from original Load: read=%q err=%v", path, got, err)
		}
		// A caller cannot modify the read-only evidence through returned bytes.
		got[0] ^= 0xff
		again, err := read(path)
		if err != nil || !bytes.Equal(again, want) {
			t.Fatalf("read-only evidence was modified by consumer at %q: %v", path, err)
		}
	}
	assertRead("tei.workspace.yaml", manifestBefore)
	assertRead("semantic/os/process.tei", parsedBefore)
	assertRead("semantic/os/thread.tei", secondBefore)
	for _, denied := range []string{"semantic/unrelated.tei", "../provider/semantic/os/process.tei", "semantic/os/missing.tei"} {
		if _, err := read(denied); err == nil {
			t.Fatalf("outside selected Namespace source %q was exposed", denied)
		}
	}
	if _, fn, err := importedParsedLoadSnapshot(loaded, "unrelated"); err == nil || fn != nil {
		t.Fatal("unimported Namespace must not expose provider Source")
	}
	if _, err := BuildSourceIndexFromLoad(consumer, loaded); !errors.Is(err, ErrImportsNotBound) {
		t.Fatalf("partial imported witness must not lift strict closure refusal: %v", err)
	}

	// Change both provider source and manifest AFTER the consumer Load. The
	// old read-only witness must stay tied to its original in-memory bytes.
	parsedAfter := append(append([]byte(nil), parsedBefore...), '\n')
	manifestAfter := append(append([]byte(nil), manifestBefore...), '\n')
	writeImportedEvidenceFixture(t, parsedPath, string(parsedAfter))
	writeImportedEvidenceFixture(t, manifestPath, string(manifestAfter))
	assertRead("tei.workspace.yaml", manifestBefore)
	assertRead("semantic/os/process.tei", parsedBefore)

	reloaded, err := Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	newEvidence, newRead, err := importedParsedLoadSnapshot(reloaded, "os")
	if err != nil {
		t.Fatal(err)
	}
	if newEvidence.SemanticDigest != evidence.SemanticDigest ||
		newEvidence.Manifest.SHA256 == evidence.Manifest.SHA256 ||
		newEvidence.FullInputClosure {
		t.Fatalf("raw changes incorrectly altered meaning or qualified complete closure: old=%+v new=%+v", evidence, newEvidence)
	}
	got, err := newRead("semantic/os/process.tei")
	if err != nil || !bytes.Equal(got, parsedAfter) {
		t.Fatalf("new Load did not capture changed provider raw Source: %v", err)
	}
	got, err = newRead("tei.workspace.yaml")
	if err != nil || !bytes.Equal(got, manifestAfter) {
		t.Fatalf("new Load did not capture changed provider Manifest: %v", err)
	}

	// A forged public ResolvedImport cannot reuse a private same-Load witness.
	forged := loaded
	forged.Imports = append([]ResolvedImport(nil), loaded.Imports...)
	forged.Imports[0].SemanticDigest = "sha256:" + string(bytes.Repeat([]byte{'0'}, 64))
	if _, _, err := importedParsedLoadSnapshot(forged, "os"); err == nil {
		t.Fatal("forged import digest must be rejected")
	}
}

func writeImportedEvidenceFixture(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func importedEvidenceFixturePair(t *testing.T) (consumer, provider string) {
	t.Helper()
	base := t.TempDir()
	consumer, provider = filepath.Join(base, "consumer"), filepath.Join(base, "provider")
	writeImportedEvidenceFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	writeImportedEvidenceFixture(t, filepath.Join(provider, "semantic/os/process.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	writeImportedEvidenceFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeImportedEvidenceFixture(t, filepath.Join(consumer, "semantic/watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	return consumer, provider
}

func TestCaptureImportedParsedLoadRedactsPrivatePaths(t *testing.T) {
	const canary = "secret-private-provider-source-canary"
	rawManifest := []byte("workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "missing-parser-byte", path: "semantic/" + canary + ".tei"},
		{name: "invalid-relative-path", path: "../" + canary + ".tei"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Construct a private malformed provider result without opening the
			// provider's filesystem; external diagnostics must not echo its path.
			provider := Result{
				semanticFilesByNamespace: map[string][]string{"os": {tc.path}},
				sourceSnapshot:           make(map[string][]byte),
				sourceManifest: ManifestIdentity{
					Path:   "tei.workspace.yaml",
					SHA256: sourceindex.Fingerprint(rawManifest),
				},
				sourceManifestBytes: &manifestReadBytes{content: append([]byte(nil), rawManifest...)},
			}
			_, err := captureImportedParsedLoad(
				WorkspaceImport{Alias: "os", Namespace: "os", Path: "../provider"},
				"sha256:independent-test-only", provider,
			)
			if err == nil {
				t.Fatal("invalid provider read evidence was accepted")
			}
			if strings.Contains(err.Error(), canary) {
				t.Fatalf("provider-private Source path leaked in rejection: %v", err)
			}
		})
	}
}
