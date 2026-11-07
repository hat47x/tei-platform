package access_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// journalModeIsOwnerOnly is the check every journal test repeats: a file that
// records decisions must not be writable by other users, because a process that
// can rewrite it can rewrite its own history.
func journalModeIsOwnerOnly(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not observable on Windows; the Linux CI run checks it")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("%s is readable or writable by others: %v", path, info.Mode().Perm())
	}
}

func TestAccessJournalIsOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.jsonl")
	guard := access.Guard{Journal: access.File{Path: path}}
	if _, err := guard.Check(core.MustCanonicalID("sales.close"), nil); err != nil {
		t.Fatal(err)
	}
	journalModeIsOwnerOnly(t, path)
}

func TestIssueLogIsOwnerOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issued.jsonl")
	log := access.IssueFile{Path: path}
	capability := core.MustCanonicalID("tei.demo")
	spec := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/tmp/demo.txt"}
	if _, err := access.Issue(log, "run-1", "environment:development", capability, spec, "because"); err != nil {
		t.Fatal(err)
	}
	journalModeIsOwnerOnly(t, path)
}
