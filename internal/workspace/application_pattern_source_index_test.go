package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestApplicationPatternSourceIndexUsesCanonicalIdentifierRangeAndSurvivesFileRename(t *testing.T) {
	root := t.TempDir()
	writePatternWorkspaceFile(t, root, "tei.workspace.yaml", `workspace: pattern_source_index_test
sources:
  - applications
`)
	writePatternWorkspaceFile(t, root, "applications/sample/application.tei", `application sample_app {
}`)
	oldPath := "applications/sample/patterns/recent-orders.tei"
	patternSource := `pattern sample_app.recent_orders browse {
    data sales.order
    query sample_app.recent_orders_query
}`
	writePatternWorkspaceFile(t, root, oldPath, patternSource)

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatalf("build Source Index: %v", err)
	}

	subject := sourceindex.Subject{Kind: "application-pattern", ID: "sample_app.recent_orders"}
	refs := idx.ReferencesFor(subject)
	if len(refs) != 1 {
		t.Fatalf("application pattern references=%#v", refs)
	}
	if refs[0].ArtifactPath != oldPath {
		t.Fatalf("artifact path=%q want=%q", refs[0].ArtifactPath, oldPath)
	}
	content := []byte(patternSource)
	rng := refs[0].Range
	if rng.Start.Offset == rng.End.Offset {
		t.Fatalf("application pattern anchor remained a point: %+v", rng)
	}
	if got := string(content[rng.Start.Offset:rng.End.Offset]); got != "sample_app.recent_orders" {
		t.Fatalf("application pattern anchor=%q range=%+v", got, rng)
	}

	newPath := "applications/sample/patterns/orders-browse.tei"
	if err := os.Rename(
		filepath.Join(root, filepath.FromSlash(oldPath)),
		filepath.Join(root, filepath.FromSlash(newPath)),
	); err != nil {
		t.Fatal(err)
	}
	movedLoaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load renamed workspace: %v", err)
	}
	moved, err := workspace.BuildSourceIndex(root, movedLoaded)
	if err != nil {
		t.Fatalf("build renamed Source Index: %v", err)
	}
	movedRefs := moved.ReferencesFor(subject)
	if len(movedRefs) != 1 {
		t.Fatalf("renamed application pattern references=%#v", movedRefs)
	}
	if movedRefs[0].ArtifactPath != newPath {
		t.Fatalf("renamed artifact path=%q want=%q", movedRefs[0].ArtifactPath, newPath)
	}
	if movedRefs[0].Subject != refs[0].Subject {
		t.Fatalf("Pattern Subject changed after file rename: before=%+v after=%+v", refs[0].Subject, movedRefs[0].Subject)
	}
}
