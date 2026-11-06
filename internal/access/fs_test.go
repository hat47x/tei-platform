package access_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

func TestFileSystemWritesWhenTheGrantCoversIt(t *testing.T) {
	root := t.TempDir()
	journal := &memoryJournal{}
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: root}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}, Journal: journal}}

	if err := fs.WriteFile("reports/close.txt", []byte("done"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	content, err := fs.ReadFile("reports/close.txt")
	if err != nil || string(content) != "done" {
		t.Fatalf("read: %q err=%v", content, err)
	}
	if len(journal.records) != 2 {
		t.Fatalf("both accesses must be recorded: %+v", journal.records)
	}
}

func TestFileSystemRefusesAnUngrantedWriteAndLeavesNoFile(t *testing.T) {
	root := t.TempDir()
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: root}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}}}

	err := fs.WriteFile("reports/close.txt", []byte("done"), 0o644)
	if err == nil {
		t.Fatal("an ungranted write must be refused")
	}
	if !strings.Contains(err.Error(), "do not cover") {
		t.Fatalf("the refusal must say why: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "reports/close.txt")); !os.IsNotExist(statErr) {
		t.Fatal("a refused write must leave no file behind")
	}
}

func TestFileSystemRefusesAPathOutsideTheRoot(t *testing.T) {
	root := t.TempDir()
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "/"}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}}}

	if err := fs.WriteFile("../escape.txt", []byte("x"), 0o644); err == nil {
		t.Fatal("a path outside the root must be refused even when the permission allows it")
	}
}

func TestFileSystemWithoutARootAllowsNothing(t *testing.T) {
	fs := access.FileSystem{Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{}}
	if _, err := fs.ReadFile("anything"); err == nil {
		t.Fatal("no root means no path is allowed")
	}
}
