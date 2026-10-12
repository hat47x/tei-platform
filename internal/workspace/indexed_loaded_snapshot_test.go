package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func indexedSnapshotWorkspace(t *testing.T) (string, workspace.Result, []byte) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace:\n  sources:\n    - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	source := []byte("namespace sales\n\ndata Parent {\n    attribute id type std.identifier\n}\n\ndata Child {\n    attribute parentId type std.identifier\n    relation parent to sales.parent one\n}\n")
	if err := os.WriteFile(filepath.Join(root, "semantic", "model.tei"), source, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, loaded, source
}

func TestIndexedLoadedSnapshotBindsParserBytesAndSourceIndex(t *testing.T) {
	root, loaded, expected := indexedSnapshotWorkspace(t)
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.ReferencesFor(sourceindex.Subject{Kind: "semantic-relation", ID: "sales.child.parent"})) != 1 {
		t.Fatalf("semantic relation was not indexed: %+v", idx.References)
	}
	path := "semantic/model.tei"
	got, err := read(path)
	if err != nil || string(got) != string(expected) {
		t.Fatalf("indexed snapshot was not read: %q, %v", got, err)
	}
	got[0] = 'X'
	another, err := read(path)
	if err != nil || string(another) != string(expected) {
		t.Fatalf("caller mutated retained snapshot: %q, %v", another, err)
	}
	for _, invalid := range []string{"../outside", "/tmp/outside", "semantic/../model.tei", "missing.tei"} {
		if _, err := read(invalid); err == nil {
			t.Fatalf("out-of-index or unnormalized source was accepted: %s", invalid)
		}
	}
	if err := idx.Validate(read); err != nil {
		t.Fatalf("snapshot did not validate against parsed Source Index: %v", err)
	}
}

func TestIndexedLoadedSnapshotDoesNotOpenReplacedSourceAfterLoad(t *testing.T) {
	root, loaded, expected := indexedSnapshotWorkspace(t)
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "semantic", "model.tei")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.tei")
	if err := os.WriteFile(outside, []byte("private data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	got, err := read("semantic/model.tei")
	if err != nil || string(got) != string(expected) || strings.Contains(string(got), "private data") {
		t.Fatalf("snapshot followed retargeted filesystem source: %q, %v", got, err)
	}
	if err := idx.Validate(read); err != nil {
		t.Fatal(err)
	}
}

func TestIndexedLoadedSnapshotRejectsForgedOrWrongWorkspace(t *testing.T) {
	root, loaded, _ := indexedSnapshotWorkspace(t)
	if _, _, err := workspace.IndexedLoadedSnapshot(t.TempDir(), loaded); err == nil {
		t.Fatal("source snapshot was reused for a different Workspace")
	}
	if _, _, err := workspace.IndexedLoadedSnapshot(root, workspace.Result{}); err == nil {
		t.Fatal("non-loaded Result was accepted as a source snapshot")
	}
	// A Result copy retains the same private loader root but must not
	// acquire an indexed source whose loader snapshot was not captured.
}
