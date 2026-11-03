package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// firstParsedPath returns a managed file whose bytes a domain loader parsed.
func firstParsedPath(t *testing.T, r Result) string {
	t.Helper()
	for _, m := range r.ManagedFiles {
		if _, ok := r.sourceSnapshot[m.Path]; ok && strings.HasSuffix(m.Path, ".tei") {
			return m.Path
		}
	}
	t.Fatal("no parsed .tei managed file")
	return ""
}

// B4: a PARSED file (sourceSnapshot) rewritten between Load and index build.
func TestBuildSourceIndexFromLoadIgnoresLaterChangeToParsedFile(t *testing.T) {
	root := copyReferenceWorkspace(t)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	rel := firstParsedPath(t, result)
	path := filepath.Join(root, filepath.FromSlash(rel))
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := append(append([]byte(nil), original...), []byte("\n// tampered after Load\n")...)
	if err := os.WriteFile(path, tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := BuildSourceIndexFromLoad(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := artifactFor(t, idx, rel), mustArtifact(t, rel, original); !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed file Artifact is not bound to Load-time bytes: got %+v want %+v", got, want)
	}
}

// B4: the "changed while loading" branch of captureManagedExtra. A parsed
// file is rewritten after the loaders read it and before the capture.
func TestLoadRefusesParsedFileChangedBeforeManagedCapture(t *testing.T) {
	root := copyReferenceWorkspace(t)
	probe, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	rel := firstParsedPath(t, probe)
	path := filepath.Join(root, filepath.FromSlash(rel))
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hookRan := false
	_, err = loadWithStackHook(root, map[string]bool{}, func() {
		hookRan = true
		if werr := os.WriteFile(path, append(append([]byte(nil), original...), '\n'), 0o644); werr != nil {
			t.Error(werr)
		}
	})
	if !hookRan {
		t.Fatal("test seam did not run")
	}
	if err == nil || !strings.Contains(err.Error(), "changed while loading Workspace") || !strings.Contains(err.Error(), rel) {
		t.Fatalf("parsed file changed before capture must fail Load naming the path: %v", err)
	}
}

// B4 companion: an unparsed file is captured by the capture step itself, so
// the hook (which runs after the loaders) shows which bytes win.
func TestLoadCapturesUnparsedFileAtCaptureTime(t *testing.T) {
	root := copyReferenceWorkspace(t)
	path := filepath.Join(root, filepath.FromSlash(loadBoundReadme))
	result, err := loadWithStackHook(root, map[string]bool{}, func() {
		if werr := os.WriteFile(path, []byte("changed before capture\n"), 0o644); werr != nil {
			t.Error(werr)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(result.sourceManagedExtra[loadBoundReadme]); got != "changed before capture\n" {
		t.Fatalf("unparsed file capture = %q", got)
	}
}

func loadBoundImportPair(t *testing.T) (consumer, provider string) {
	t.Helper()
	base := t.TempDir()
	consumer = filepath.Join(base, "consumer")
	provider = filepath.Join(base, "provider")
	write := func(file, content string) {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(provider, "tei.workspace.yaml"), "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	write(filepath.Join(provider, "semantic", "os", "process.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	write(filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	write(filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	return consumer, provider
}

// B1: a Workspace with imports cannot be given a Load-bound Source Index,
// because the provider's bytes are not part of it.
func TestBuildSourceIndexFromLoadRefusesWorkspaceWithImports(t *testing.T) {
	consumer, provider := loadBoundImportPair(t)
	result, err := Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imports) != 1 {
		t.Fatalf("fixture must have one import: %#v", result.Imports)
	}
	if _, err := BuildSourceIndexFromLoad(consumer, result); !errors.Is(err, ErrImportsNotBound) {
		t.Fatalf("Workspace with imports must be refused with ErrImportsNotBound: %v", err)
	}
	// The provider changing afterwards cannot make it acceptable either.
	file := filepath.Join(provider, "semantic", "os", "process.tei")
	changed := "namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n    attribute extra type os.process_id\n}\n"
	if err := os.WriteFile(file, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildSourceIndexFromLoad(consumer, result); !errors.Is(err, ErrImportsNotBound) {
		t.Fatalf("provider change must not lift the refusal: %v", err)
	}
	// Clearing the exported import list does not hide a declared import.
	forged := result
	forged.Imports = nil
	if _, err := BuildSourceIndexFromLoad(consumer, forged); !errors.Is(err, ErrImportsNotBound) {
		t.Fatalf("cleared Imports must not hide a declared import: %v", err)
	}
	// The legacy builder is unchanged and still works for imports.
	if _, err := BuildSourceIndex(consumer, result); err != nil {
		t.Fatalf("legacy builder regressed: %v", err)
	}
}

// B2: the manifest decides source roots and imports but is not in the Source
// Index. Its Load-time identity is exposed, and a later change is not adopted.
func TestManifestChangeAfterLoadIsNotAdoptedByStrictBuilder(t *testing.T) {
	root := copyReferenceWorkspace(t)
	manifestPath := filepath.Join(root, "tei.workspace.yaml")
	original, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := ManifestIdentity{Path: "tei.workspace.yaml", SHA256: "sha256:" + sha256Hex(original)}
	if got := result.ManifestIdentity(); got != want {
		t.Fatalf("manifest identity = %+v, want %+v", got, want)
	}
	before, err := BuildSourceIndexFromLoad(root, result)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range before.Artifacts {
		if a.Path == "tei.workspace.yaml" {
			t.Fatal("manifest became a Source Index Artifact; that would change every accepted revision")
		}
	}
	changed := append(append([]byte(nil), original...), []byte("  - extra-root\n")...)
	if err := os.WriteFile(manifestPath, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := BuildSourceIndexFromLoad(root, result)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("strict builder reacted to a manifest change after Load")
	}
	if got := result.ManifestIdentity(); got != want {
		t.Fatalf("manifest identity moved after Load: %+v", got)
	}
	// A fresh Load reports the changed manifest under a different identity.
	// (The extra root has no directory, which Load may reject; either way the
	// identity of the first Load stays as recorded above.)
	if fresh, err := Load(root); err == nil && fresh.ManifestIdentity() == want {
		t.Fatal("fresh Load did not report the changed manifest")
	}
}
