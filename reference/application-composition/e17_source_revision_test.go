// E17 checks two concrete source byte states without claiming semantic change.
// This is not an accepted revision, provider decision, or runtime authority.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func e17CopyReferenceWorkspace(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "reference", "workspace")
	destination := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		out := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(out, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("not a regular Workspace file: %s", relative)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, bytes, 0600)
	})
	if err != nil {
		t.Fatalf("prepare isolated reference Workspace: %v", err)
	}
	return destination
}

func e17ArtifactFingerprint(idx sourceindex.Index, path string) (string, bool) {
	var found string
	count := 0
	for _, artifact := range idx.Artifacts {
		if artifact.Path == path {
			found = artifact.Fingerprint
			count++
		}
	}
	return found, count == 1 && found != ""
}

func e17RelationSource(report e5Report) (string, bool) {
	seen := 0
	var fingerprint string
	for _, dep := range report.Dependencies {
		if dep.Subject.Kind == "semantic-relation" &&
			dep.Subject.ID == "sales.order.customer" {
			seen++
			fingerprint = dep.Source.Fingerprint
		}
	}
	return fingerprint, seen == 1 && fingerprint != ""
}

func TestE17ParsedSourceSnapshotRetainsOriginalBytesAfterFileEdit(t *testing.T) {
	root := e17CopyReferenceWorkspace(t)
	loadedBefore, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load copied Workspace: %v", err)
	}
	before, err := workspace.BuildSourceIndex(root, loadedBefore)
	if err != nil {
		t.Fatalf("build original Source Index: %v", err)
	}
	const changedPath = "semantic/sales/order.tei"
	original, ok := e17ArtifactFingerprint(before, changedPath)
	if !ok {
		t.Fatal("original semantic source not uniquely indexed")
	}
	beforeReport := inspectE5(loadedBefore.World, before, "order_entry")
	if beforeReport.Status != "observed" {
		t.Fatalf("original E5 Source report unresolved: %d issues", len(beforeReport.Issues))
	}
	if relation, ok := e17RelationSource(beforeReport); !ok || relation != original {
		t.Fatal("original Relation Source does not match its Artifact")
	}

	// Create a second concrete file state with changed bytes but identical
	// semantic declarations. This is not a separate Git revision.
	file := filepath.Join(root, filepath.FromSlash(changedPath))
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read isolated semantic source: %v", err)
	}
	if err := os.WriteFile(file, append(content, '\n'), 0600); err != nil {
		t.Fatalf("modify isolated semantic source: %v", err)
	}
	// Re-index the *old* loaded World after disk changes. The merged #1017
	// loader must use the bytes it parsed, not silently read the new content.
	pinned, err := workspace.BuildSourceIndex(root, loadedBefore)
	if err != nil {
		t.Fatalf("re-index original parsed snapshot: %v", err)
	}
	if evidence, ok := e17ArtifactFingerprint(pinned, changedPath); !ok || evidence != original {
		t.Fatal("parsed-byte Source snapshot was overwritten by later file bytes")
	}
	loadedAfter, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load changed Workspace: %v", err)
	}
	after, err := workspace.BuildSourceIndex(root, loadedAfter)
	if err != nil {
		t.Fatalf("build changed Source Index: %v", err)
	}
	changed, ok := e17ArtifactFingerprint(after, changedPath)
	if !ok || changed == original {
		t.Fatal("real file byte change was not observed in Source Index")
	}
	afterReport := inspectE5(loadedAfter.World, after, "order_entry")
	if afterReport.Status != "observed" {
		t.Fatalf("new E5 Source report unresolved: %d issues", len(afterReport.Issues))
	}
	if relation, ok := e17RelationSource(afterReport); !ok || relation != changed {
		t.Fatal("new Relation Source did not observe the changed Artifact fingerprint")
	}

	// Source freshness and typed semantic change are separate claims.
	// Whitespace-only source changes must requalify Source evidence, but may
	// legitimately leave the Derived model and modelref Graph unchanged.
	const scope = "order-entry-reference"
	a := derivedbridge.FromIndex(index.Build(loadedBefore.World), scope)
	b := derivedbridge.FromIndex(index.Build(loadedAfter.World), scope)
	if changes := modelref.Diff(a.Graph, b.Graph); len(changes) != 0 {
		t.Fatalf("whitespace-only Source drift became %d semantic Graph changes", len(changes))
	}
}
