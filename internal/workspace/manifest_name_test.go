package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// TestLoadReportsTheManifestNameAndFormat pins #3181: the `workspace:` label
// and `formatVersion` are read so the read-only projections can say which
// Workspace produced a result. The label is descriptive 窶・it does not select
// source roots and is not an identity 窶・so an empty label stays accepted,
// exactly as before.
func TestLoadReportsTheManifestNameAndFormat(t *testing.T) {
	root := t.TempDir()
	writeRootFixture(t, root, "workspace: labelled\nformatVersion: 1\nsources:\n  - semantic\n", map[string]string{
		"semantic/marker.tei": "namespace labelled\n",
	})
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.WorkspaceName(); got != "labelled" {
		t.Fatalf("workspace name=%q", got)
	}
	if got := loaded.ManifestFormatVersion(); got != 1 {
		t.Fatalf("formatVersion=%d", got)
	}
	if got := loaded.ManifestIdentity().Path; got != "tei.workspace.yaml" {
		t.Fatalf("manifest path=%q", got)
	}
	if got := loaded.ManifestIdentity().SHA256; got == "" {
		t.Fatal("manifest sha256 is empty")
	}
}

func TestLoadAcceptsAnEmptyManifestName(t *testing.T) {
	root := t.TempDir()
	writeRootFixture(t, root, "workspace:\nformatVersion: 1\nsources:\n  - semantic\n", map[string]string{
		"semantic/marker.tei": "namespace unlabelled\n",
	})
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("an empty workspace label must stay accepted: %v", err)
	}
	if got := loaded.WorkspaceName(); got != "" {
		t.Fatalf("workspace name=%q", got)
	}
}

// TestManifestNameDoesNotChangeTheModelRevision keeps the label out of the
// managed-Source revision: renaming a Workspace must not look like a change of
// its declarations, and the manifest is not a Source Index Artifact.
func TestManifestNameDoesNotChangeTheModelRevision(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	for dir, name := range map[string]string{first: "before", second: "after"} {
		writeRootFixture(t, dir, "workspace: "+name+"\nformatVersion: 1\nsources:\n  - semantic\n", map[string]string{
			"semantic/marker.tei": "namespace shared\n",
		})
	}
	var revisions []string
	for _, dir := range []string{first, second} {
		loaded, err := workspace.Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		idx, err := workspace.BuildSourceIndex(dir, loaded)
		if err != nil {
			t.Fatal(err)
		}
		revisions = append(revisions, sourceindex.ModelRevision(idx))
	}
	if revisions[0] != revisions[1] {
		t.Fatalf("renaming the Workspace changed the managed Source revision: %v", revisions)
	}
	if _, err := os.Stat(filepath.Join(first, "tei.workspace.yaml")); err != nil {
		t.Fatal(err)
	}
}
