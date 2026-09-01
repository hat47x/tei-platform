package workspace_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceBrowsePatternIdentitySurvivesFileRename(t *testing.T) {
	repo := repoRoot(t)
	originalRoot := filepath.Join(repo, "reference", "workspace")
	movedRoot := filepath.Join(t.TempDir(), "workspace")
	copyWorkspaceTree(t, originalRoot, movedRoot)

	const patternID = "order_entry.recent_orders"
	const oldPath = "applications/order-entry/patterns/recent-orders.tei"
	const newPath = "applications/order-entry/patterns/orders-browse.tei"
	subject := sourceindex.Subject{Kind: "application-pattern", ID: patternID}

	before, err := workspace.Load(originalRoot)
	if err != nil {
		t.Fatalf("元のReference Workspaceを読み込めませんでした: %v", err)
	}
	beforeIndex, err := workspace.BuildSourceIndex(originalRoot, before)
	if err != nil {
		t.Fatalf("元のSource Indexを構築できませんでした: %v", err)
	}
	beforeRefs := beforeIndex.ReferencesFor(subject)
	if len(beforeRefs) != 1 || beforeRefs[0].ArtifactPath != oldPath {
		t.Fatalf("変更前のPattern sourceが期待値と異なります: %#v", beforeRefs)
	}

	if err := os.Rename(
		filepath.Join(movedRoot, filepath.FromSlash(oldPath)),
		filepath.Join(movedRoot, filepath.FromSlash(newPath)),
	); err != nil {
		t.Fatalf("Patternファイル名を変更できませんでした: %v", err)
	}

	after, err := workspace.Load(movedRoot)
	if err != nil {
		t.Fatalf("ファイル名変更後のReference Workspaceを読み込めませんでした: %v", err)
	}
	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatalf("Patternファイル名の変更でCanonical Worldが変化しました\nbefore=%#v\nafter=%#v", before.World, after.World)
	}

	afterIndex, err := workspace.BuildSourceIndex(movedRoot, after)
	if err != nil {
		t.Fatalf("ファイル名変更後のSource Indexを構築できませんでした: %v", err)
	}
	afterRefs := afterIndex.ReferencesFor(subject)
	if len(afterRefs) != 1 {
		t.Fatalf("変更後のPattern source references=%#v", afterRefs)
	}
	if afterRefs[0].Subject != beforeRefs[0].Subject {
		t.Fatalf("Pattern Subject identityが変化しました: before=%+v after=%+v", beforeRefs[0].Subject, afterRefs[0].Subject)
	}
	if afterRefs[0].ArtifactPath != newPath {
		t.Fatalf("変更後のArtifact path=%q want=%q", afterRefs[0].ArtifactPath, newPath)
	}
	if afterRefs[0].ArtifactPath == oldPath {
		t.Fatalf("変更後のSource Indexに古いArtifact pathが残っています: %#v", afterRefs)
	}
}

func copyWorkspaceTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	}); err != nil {
		t.Fatalf("Reference Workspaceを一時領域へコピーできませんでした: %v", err)
	}
}
