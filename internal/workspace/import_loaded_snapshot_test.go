package workspace_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// A selected provider Namespace has a concrete parsed-byte witness from its
// ORIGINAL nested Load. The witness stays unchanged after the physical files
// move or change; a fresh navigation observation is a separate operation.
func TestImportedParsedLoadSnapshotRetainsSameLoadBytesWithoutReread(t *testing.T) {
	consumer, provider := externalPair(t)
	manifestPath := filepath.Join(provider, "tei.workspace.yaml")
	parsedPath := filepath.Join(provider, "semantic/os/process.tei")
	secondPath := filepath.Join(provider, "semantic/os/thread.tei")
	unrelatedPath := filepath.Join(provider, "semantic/unrelated.tei")
	writeExternalFixture(t, secondPath, "namespace os\n\ntype ThreadId from std.identifier\n")
	writeExternalFixture(t, unrelatedPath, "namespace unrelated\n\ntype SecretId from std.identifier\n")

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
	loaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Imports) != 1 {
		t.Fatalf("expected one imported Namespace: %+v", loaded.Imports)
	}
	evidence, read, err := workspace.ImportedParsedLoadSnapshot(loaded, "os")
	if err != nil {
		t.Fatal(err)
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
	if _, fn, err := workspace.ImportedParsedLoadSnapshot(loaded, "unrelated"); err == nil || fn != nil {
		t.Fatal("unimported Namespace must not expose provider Source")
	}
	if _, err := workspace.BuildSourceIndexFromLoad(consumer, loaded); !errors.Is(err, workspace.ErrImportsNotBound) {
		t.Fatalf("partial imported witness must not lift strict closure refusal: %v", err)
	}

	// Change both provider source and manifest AFTER the consumer Load. The
	// old read-only witness must stay tied to its original in-memory bytes.
	parsedAfter := append(append([]byte(nil), parsedBefore...), '\n')
	manifestAfter := append(append([]byte(nil), manifestBefore...), '\n')
	writeExternalFixture(t, parsedPath, string(parsedAfter))
	writeExternalFixture(t, manifestPath, string(manifestAfter))
	assertRead("tei.workspace.yaml", manifestBefore)
	assertRead("semantic/os/process.tei", parsedBefore)

	reloaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	newEvidence, newRead, err := workspace.ImportedParsedLoadSnapshot(reloaded, "os")
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
	forged.Imports = append([]workspace.ResolvedImport(nil), loaded.Imports...)
	forged.Imports[0].SemanticDigest = "sha256:" + string(bytes.Repeat([]byte{'0'}, 64))
	if _, _, err := workspace.ImportedParsedLoadSnapshot(forged, "os"); err == nil {
		t.Fatal("forged import digest must be rejected")
	}
}
