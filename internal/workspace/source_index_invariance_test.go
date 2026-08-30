package workspace_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestWorkspaceSourceMoveChangesOnlySourceIndexLocator(t *testing.T) {
	root := t.TempDir()
	manifest := []byte("workspace: source_traceability\nsources:\n  - semantic\n")
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	oldPath := "semantic/sales/order.tei"
	newPath := "semantic/domain/order.tei"
	source := []byte(`namespace sales

type OrderId from std.identifier

data Order {
    attribute orderId type sales.order_id
}
`)
	writeSource(t, root, oldPath, source)

	before, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load before move: %v", err)
	}
	beforeIndex, err := workspace.BuildSourceIndex(root, before)
	if err != nil {
		t.Fatalf("build source index before move: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(newPath))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(
		filepath.Join(root, filepath.FromSlash(oldPath)),
		filepath.Join(root, filepath.FromSlash(newPath)),
	); err != nil {
		t.Fatal(err)
	}

	after, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load after move: %v", err)
	}
	afterIndex, err := workspace.BuildSourceIndex(root, after)
	if err != nil {
		t.Fatalf("build source index after move: %v", err)
	}

	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatalf("normalized World changed after source-only move\nbefore=%#v\nafter=%#v", before.World, after.World)
	}

	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	beforeRefs := beforeIndex.ReferencesFor(subject)
	afterRefs := afterIndex.ReferencesFor(subject)
	if len(beforeRefs) != 1 || len(afterRefs) != 1 {
		t.Fatalf("unexpected references before=%#v after=%#v", beforeRefs, afterRefs)
	}
	if beforeRefs[0].Subject != afterRefs[0].Subject {
		t.Fatalf("semantic subject changed after move: before=%+v after=%+v", beforeRefs[0].Subject, afterRefs[0].Subject)
	}
	if beforeRefs[0].ArtifactPath != oldPath || afterRefs[0].ArtifactPath != newPath {
		t.Fatalf("source locator did not follow move: before=%q after=%q", beforeRefs[0].ArtifactPath, afterRefs[0].ArtifactPath)
	}

	beforeArtifact, ok := artifactByPath(beforeIndex, oldPath)
	if !ok {
		t.Fatalf("missing artifact %s before move", oldPath)
	}
	afterArtifact, ok := artifactByPath(afterIndex, newPath)
	if !ok {
		t.Fatalf("missing artifact %s after move", newPath)
	}
	if beforeArtifact.Fingerprint != afterArtifact.Fingerprint {
		t.Fatalf("same source content changed fingerprint: before=%s after=%s", beforeArtifact.Fingerprint, afterArtifact.Fingerprint)
	}

	beforeText := source[beforeRefs[0].Range.Start.Offset:beforeRefs[0].Range.End.Offset]
	afterText := source[afterRefs[0].Range.Start.Offset:afterRefs[0].Range.End.Offset]
	if string(beforeText) != "Order" || string(afterText) != "Order" {
		t.Fatalf("semantic-data anchor changed unexpectedly: before=%q after=%q", beforeText, afterText)
	}
}

func artifactByPath(index sourceindex.Index, path string) (sourceindex.Artifact, bool) {
	for _, artifact := range index.Artifacts {
		if artifact.Path == path {
			return artifact, true
		}
	}
	return sourceindex.Artifact{}, false
}
