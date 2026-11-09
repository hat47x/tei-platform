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

// TestFileSystemRecordsAnEscapeAttempt checks that a path outside the root is
// recorded as a refusal. The permission check never runs for it, so without this
// the most interesting refusal (an attempt to leave the root) would be the one
// with no trace.
func TestFileSystemRecordsAnEscapeAttempt(t *testing.T) {
	root := t.TempDir()
	journal := &memoryJournal{}
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: root}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}, Journal: journal}}

	if _, err := fs.ReadFile("../escape.txt"); err == nil {
		t.Fatal("a path outside the root must be refused")
	}
	if len(journal.records) != 1 {
		t.Fatalf("the escape attempt must be recorded: %+v", journal.records)
	}
	record := journal.records[0]
	if record.Allowed {
		t.Fatalf("an escape attempt must be recorded as refused: %+v", record)
	}
	if strings.Join(record.Missing, "") == "" {
		t.Fatalf("the record must say what was refused: %+v", record)
	}
}

// TestFileSystemDoesNotFailASecondTimeWhenTheJournalIsBroken pins the tradeoff:
// the path is already refused, so a journal failure must not change the answer
// the caller gets. Reporting a journal error instead would hide the refusal.
func TestFileSystemDoesNotFailASecondTimeWhenTheJournalIsBroken(t *testing.T) {
	root := t.TempDir()
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: root}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}, Journal: &memoryJournal{err: os.ErrPermission}}}

	_, err := fs.ReadFile("../escape.txt")
	if err == nil {
		t.Fatal("a path outside the root must be refused")
	}
	if strings.Contains(err.Error(), "escape.txt is outside the root") == false {
		t.Fatalf("the caller must see the refusal, not the journal problem: %v", err)
	}
}
