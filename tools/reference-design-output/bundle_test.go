package designoutput

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func referenceWorkspace(t *testing.T) string {
	t.Helper()
	return filepath.Join(repositoryRoot(t), "reference", "workspace")
}

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = content
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestBundleIsDeterministicAndBindsEveryArtifactToTheSourceRevision(t *testing.T) {
	root := referenceWorkspace(t)
	firstDir := filepath.Join(t.TempDir(), "bundle-a")
	secondDir := filepath.Join(t.TempDir(), "bundle-b")

	first, err := Write(root, firstDir, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Write(root, secondDir, "postgres")
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	wantRevision := sourceindex.ModelRevision(index)
	if first.SourceRevision == "" {
		t.Fatal("manifest has no sourceRevision")
	}
	if first.SourceRevision != wantRevision {
		t.Fatalf("manifest sourceRevision %q does not match the loaded Source Index %q", first.SourceRevision, wantRevision)
	}
	if first.SourceRevision != second.SourceRevision {
		t.Fatalf("two runs disagree on sourceRevision: %q vs %q", first.SourceRevision, second.SourceRevision)
	}
	if len(index.Artifacts) == 0 {
		t.Fatal("reference Workspace has no Source Index artifacts")
	}

	firstFiles := readTree(t, firstDir)
	secondFiles := readTree(t, secondDir)
	if len(firstFiles) != len(secondFiles) {
		t.Fatalf("two runs produced different file sets: %d vs %d", len(firstFiles), len(secondFiles))
	}
	names := make([]string, 0, len(firstFiles))
	for name := range firstFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		other, ok := secondFiles[name]
		if !ok {
			t.Fatalf("second run is missing %s", name)
		}
		if !bytes.Equal(firstFiles[name], other) {
			t.Fatalf("artifact %s is not deterministic", name)
		}
	}

	if _, ok := firstFiles["manifest.json"]; !ok {
		t.Fatal("manifest.json was not written")
	}
	for _, artifact := range first.Artifacts {
		content, ok := firstFiles[artifact.Path]
		if !ok {
			t.Fatalf("artifact %s names missing file %s", artifact.Name, artifact.Path)
		}
		sum := sha256.Sum256(content)
		if want := "sha256:" + hex.EncodeToString(sum[:]); artifact.Digest != want {
			t.Fatalf("artifact %s digest does not match its bytes: %s vs %s", artifact.Name, artifact.Digest, want)
		}
		if artifact.Bytes != len(content) {
			t.Fatalf("artifact %s byte count does not match: %d vs %d", artifact.Name, artifact.Bytes, len(content))
		}
	}
	for i := 1; i < len(first.Artifacts); i++ {
		if first.Artifacts[i-1].Name >= first.Artifacts[i].Name {
			t.Fatalf("artifacts are not sorted by name: %q then %q", first.Artifacts[i-1].Name, first.Artifacts[i].Name)
		}
	}

	var sawDDL bool
	for _, artifact := range first.Artifacts {
		if artifact.Kind == "ddl-script" {
			sawDDL = true
			if !strings.Contains(string(firstFiles[artifact.Path]), "fresh-create DDL") {
				t.Fatalf("DDL artifact %s has no boundary statement", artifact.Path)
			}
		}
	}
	if !sawDDL {
		t.Fatal("reference Workspace produced no DDL artifact")
	}
	inverse := string(firstFiles["data-application-matrix.md"])
	if !strings.Contains(inverse, "`sales.order`") || !strings.Contains(inverse, "`order_entry`") {
		t.Fatalf("inverse matrix does not contain the declared Data and Application:\n%s", inverse)
	}
}

func TestBundleRefusesAnExistingOutputDirectoryAndLeavesNoStagingDirectory(t *testing.T) {
	root := referenceWorkspace(t)
	parent := t.TempDir()
	out := filepath.Join(parent, "existing")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(out, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("do not replace\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Write(root, out, "postgres"); err == nil {
		t.Fatal("existing output directory was accepted")
	}
	content, err := os.ReadFile(sentinel)
	if err != nil || string(content) != "do not replace\n" {
		t.Fatalf("existing output was modified: %q %v", content, err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tei-design-output-") {
			t.Fatalf("staging directory was left behind: %s", entry.Name())
		}
	}
}

func TestBundleDoesNotPromoteAnUnacceptedCandidate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.CopyFS(root, os.DirFS(referenceWorkspace(t))); err != nil {
		t.Fatalf("copy reference Workspace: %v", err)
	}
	candidateDir := filepath.Join(root, ".tei", "candidates")
	if err := os.MkdirAll(candidateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	candidate := []byte(`{
  "schema": "tei.reference.change-candidate/v0",
  "id": "unaccepted-new-data",
  "delta": [{"op": "add-data", "id": "sales.fake_candidate_data"}]
}
`)
	if err := os.WriteFile(filepath.Join(candidateDir, "unaccepted.json"), candidate, 0o600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "bundle")
	if _, err := Write(root, out, "postgres"); err != nil {
		t.Fatal(err)
	}
	files := readTree(t, out)
	for _, name := range []string{"table-definitions.md", "function-definitions.md", "application-data-matrix.md", "data-application-matrix.md"} {
		if strings.Contains(string(files[name]), "fake_candidate_data") {
			t.Fatalf("unaccepted candidate leaked into %s", name)
		}
	}
}

func TestBundleManifestIsSortedAndRecordsUnsupportedDimensions(t *testing.T) {
	root := referenceWorkspace(t)
	out := filepath.Join(t.TempDir(), "bundle")
	manifest, err := Write(root, out, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("manifest.json is not valid JSON: %v", err)
	}
	if decoded.SourceRevision != manifest.SourceRevision {
		t.Fatalf("returned and stored sourceRevision differ: %q vs %q", manifest.SourceRevision, decoded.SourceRevision)
	}
	found := false
	for _, unsupported := range decoded.Unsupported {
		if unsupported.Name == "business-cross-view" {
			found = true
			if !strings.Contains(unsupported.Reason, "#1125") {
				t.Fatalf("unsupported reason does not name the Owner: %q", unsupported.Reason)
			}
		}
	}
	if !found {
		t.Fatal("unaccepted Business cross-view was not recorded as unsupported")
	}
	if len(decoded.Notes) == 0 {
		t.Fatal("manifest has no boundary notes")
	}
}
