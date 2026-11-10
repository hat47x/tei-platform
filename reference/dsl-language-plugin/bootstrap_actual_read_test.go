package languagereference

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// Actual ReadByLoader is a narrower claim than managed inventory or Source Index
// presence. It does not prove semantic acceptance, selected Language identity,
// Operator lowering, or Build consumption.
func TestBootstrapK1LoaderReadIsDistinctFromManagedInventory(t *testing.T) {
	root := t.TempDir()
	write := func(relative, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: bootstrap-read\nformatVersion: 1\nsources:\n  - semantic\n")
	write("semantic/model.tei", "namespace organization\n\ntype OrgUnitId from std.identifier\n\ndata OrgUnit {\n    attribute orgUnitId type organization.org_unit_id\n}\n")
	write("semantic/README.md", "# Draft documentation, not TEI meaning\n")
	write("drafts/unmanaged.tei.md", "# Not a declared TEI Source\n")

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	readByFile := make(map[string]bool)
	for _, managed := range loaded.ManagedFiles {
		readByFile[filepath.ToSlash(managed.Path)] = managed.ReadByLoader
	}
	nativeRead, nativeManaged := readByFile["semantic/model.tei"]
	if !nativeManaged || !nativeRead {
		t.Fatalf("actual native TEI Source was not read: %+v", readByFile)
	}
	notesRead, notesManaged := readByFile["semantic/README.md"]
	if !notesManaged || notesRead {
		t.Fatalf("ordinary Markdown should be inventoried but not parsed: %+v", readByFile)
	}
	if _, found := readByFile["drafts/unmanaged.tei.md"]; found {
		t.Fatalf("unmanaged draft entered Source inventory: %+v", readByFile)
	}

	idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := make(map[string]bool)
	for _, item := range idx.Artifacts {
		artifacts[filepath.ToSlash(item.Path)] = true
	}
	if !artifacts["semantic/model.tei"] || !artifacts["semantic/README.md"] {
		t.Fatalf("Source Index should keep both read and unread managed artifacts: %+v", artifacts)
	}
	if artifacts["drafts/unmanaged.tei.md"] {
		t.Fatalf("Source Index pulled unmanaged draft into the Workspace: %+v", artifacts)
	}
	nativeRefs := 0
	for _, ref := range idx.References {
		switch filepath.ToSlash(ref.ArtifactPath) {
		case "semantic/model.tei":
			nativeRefs++
		case "semantic/README.md", "drafts/unmanaged.tei.md":
			t.Fatalf("unparsed Markdown or unmanaged draft acquired a semantic Source reference: %+v", ref)
		}
	}
	if nativeRefs == 0 {
		t.Fatal("valid native TEI Source must produce a Source reference; otherwise the negative case is vacuous")
	}

	// Editing managed but unread Markdown changes the Source revision without
	// changing the semantic World or its qualified Source references. This
	// separates content change detection from actual semantic consumption.
	write("semantic/README.md", "# Updated documentation, still not TEI meaning\n")
	reloaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	reloadedIndex, err := workspace.BuildSourceIndexFromLoad(root, reloaded)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(idx) == sourceindex.ModelRevision(reloadedIndex) {
		t.Fatal("changed unread Markdown was lost from Source revision")
	}
	if !reflect.DeepEqual(loaded.World, reloaded.World) {
		t.Fatal("unread Markdown mutated the semantic World")
	}
	if !reflect.DeepEqual(idx.References, reloadedIndex.References) {
		t.Fatal("unread Markdown changed semantic Source references")
	}
	for _, managed := range reloaded.ManagedFiles {
		if filepath.ToSlash(managed.Path) == "semantic/README.md" && managed.ReadByLoader {
			t.Fatal("edited Markdown was incorrectly promoted to Loader read")
		}
	}

	// The same unread bytes at a different managed path produce a different
	// Source revision, but do not change the parsed World or semantic refs.
	if err := os.Rename(filepath.Join(root, "semantic", "README.md"),
		filepath.Join(root, "semantic", "notes.md")); err != nil {
		t.Fatal(err)
	}
	moved, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	movedIndex, err := workspace.BuildSourceIndexFromLoad(root, moved)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(reloadedIndex) == sourceindex.ModelRevision(movedIndex) {
		t.Fatal("managed Source path change was lost from the revision")
	}
	if !reflect.DeepEqual(reloaded.World, moved.World) ||
		!reflect.DeepEqual(reloadedIndex.References, movedIndex.References) {
		t.Fatal("moving unread Markdown changed parsed meaning or semantic references")
	}

	// A -> changed bytes -> relocated path -> A can return to the same
	// content-addressed revision. That equality is NOT a record of Owner
	// reacceptance, intermediate observations, or runtime authorization.
	if err := os.Rename(filepath.Join(root, "semantic", "notes.md"),
		filepath.Join(root, "semantic", "README.md")); err != nil {
		t.Fatal(err)
	}
	write("semantic/README.md", "# Draft documentation, not TEI meaning\n")
	restored, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	restoredIndex, err := workspace.BuildSourceIndexFromLoad(root, restored)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(idx) != sourceindex.ModelRevision(restoredIndex) {
		t.Fatal("exact restored Source path/bytes did not restore content-addressed revision")
	}
	if !reflect.DeepEqual(loaded.World, restored.World) ||
		!reflect.DeepEqual(idx.References, restoredIndex.References) {
		t.Fatal("restored unread document altered parsed meaning or semantic references")
	}
}
