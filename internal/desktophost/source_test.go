package desktophost

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func fixtureWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace")
	folder := filepath.Join(root, "custom")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: demo\nsources:\n  - custom\n"), 0600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(folder, "asset.txt")
	if err := os.WriteFile(file, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	return root, file
}

func TestObserveSourceFindsExternalEditWithoutWatcher(t *testing.T) {
	root, file := fixtureWorkspace(t)
	before, err := ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Artifacts) != 1 || before.Artifacts[0].Path != "custom/asset.txt" {
		t.Fatalf("source snapshot has unexpected artifacts: %+v", before)
	}
	if _, err := CheckSourceFresh(before); err != nil {
		t.Fatalf("unchanged Source reported as changed: %v", err)
	}
	if err := os.WriteFile(file, []byte("external editor change"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckSourceFresh(before); !errors.Is(err, ErrSourceChanged) {
		t.Fatalf("external Source change was missed: %v", err)
	}
}

func TestObserveSourceDetectsAddedAndRemovedFiles(t *testing.T) {
	root, file := fixtureWorkspace(t)
	before, err := ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(root, "custom", "new.txt")
	if err := os.WriteFile(added, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckSourceFresh(before); !errors.Is(err, ErrSourceChanged) {
		t.Fatalf("added file was missed: %v", err)
	}
	if err := os.Remove(added); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckSourceFresh(before); !errors.Is(err, ErrSourceChanged) {
		t.Fatalf("removed file was missed: %v", err)
	}
}

func TestSourceOutsideWorkspaceIsNotObserved(t *testing.T) {
	root, file := fixtureWorkspace(t)
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, file); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ObserveSource(root); err == nil {
		t.Fatal("source outside Workspace was observed without an error")
	}
}
