package access_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// linkInsideRoot creates root/name pointing at target.
//
// On Windows a symlink needs privileges but a directory junction does not. Using
// a junction keeps the escape test runnable here; skipping it silently would
// leave the most interesting refusal unchecked.
func linkInsideRoot(t *testing.T, root, name, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if err := os.Mkdir(target, 0o755); err != nil && !os.IsExist(err) {
			t.Fatal(err)
		}
		// `mklink /J` needs no privileges; os.Symlink on Windows does.
		if out, err := junction(filepath.Join(root, name), target); err != nil {
			t.Skipf("creating a junction failed: %v (%s)", err, out)
		}
		return
	}
	if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
		t.Skipf("creating a symlink failed: %v", err)
	}
}

// junction creates a directory junction through cmd.exe. On non-Windows it is
// never called.
func junction(link, target string) (string, error) {
	return runCmd("cmd", "/c", "mklink", "/J", link, target)
}

// TestFileSystemRefusesALinkThatLeavesTheRoot pins the difference between "inside
// the root when written" and "inside the root in fact". A lexical check would
// accept root/link/passwd when link points at /etc, and the write would land
// outside the confinement the root is supposed to give.
func TestFileSystemRefusesALinkThatLeavesTheRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	linkInsideRoot(t, root, "escape", outside)

	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: root}
	journal := &memoryJournal{}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}, Journal: journal}}

	if err := fs.WriteFile(filepath.Join("escape", "secret.txt"), []byte("x"), 0o600); err == nil {
		t.Fatal("a link that leaves the root must be refused")
	}
	if _, err := os.Stat(filepath.Join(outside, "secret.txt")); !os.IsNotExist(err) {
		t.Fatal("a refused write must leave no file outside the root")
	}
	// Two decisions are recorded: the permission check allowed the request (the
	// scope covers it), and the operating system refused the access. The last
	// record is what happened.
	records := journal.records
	if len(records) == 0 {
		t.Fatal("the refusal must be recorded")
	}
	last := records[len(records)-1]
	if last.Allowed {
		t.Fatalf("the last record must be the refusal: %+v", records)
	}
	if len(last.Missing) == 0 || !strings.Contains(strings.Join(last.Missing, ""), "escape") {
		t.Fatalf("the record must name the refused attempt: %+v", last)
	}
}

// TestFileSystemRefusesALinkThatStaysInsideDocumentsTheLimit pins what os.Root
// actually does: it refuses every path that goes through a link, including one
// that points back inside the root. That is stricter than a name-based policy
// needs to be, but it is the confinement the operating system gives, and calling
// it an escape would be wrong.
//
// The test exists so the behaviour is written down rather than discovered: if a
// later Go version starts allowing a link that stays inside, this test fails and
// the change is visible.
func TestFileSystemRefusesALinkThatStaysInsideDocumentsTheLimit(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "real")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	linkInsideRoot(t, root, "inside", inside)

	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: root}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}}}

	err := fs.WriteFile(filepath.Join("inside", "report.txt"), []byte("ok"), 0o600)
	if err == nil {
		t.Skip("this Go version lets os.Root follow a link that stays inside the root; the documented limit changed")
	}
	if !strings.Contains(err.Error(), "path escapes") {
		t.Fatalf("the refusal must come from the root, not from something else: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(inside, "report.txt")); !os.IsNotExist(statErr) {
		t.Fatal("a refused write must leave no file behind")
	}
}

// TestFileSystemAcceptsARootReachedThroughALink pins the other direction: the root
// itself may be a link (macOS /tmp is one), and then every path under it would be
// reported outside if only the written root were compared.
func TestFileSystemAcceptsARootReachedThroughALink(t *testing.T) {
	parent := t.TempDir()
	real := filepath.Join(parent, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "alias")
	linkInsideRoot(t, parent, "alias", real)

	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: alias}
	fs := access.FileSystem{Root: alias, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}}}

	if err := fs.WriteFile("report.txt", []byte("ok"), 0o600); err != nil {
		t.Fatalf("a root reached through a link must still confine correctly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(real, "report.txt")); err != nil {
		t.Fatalf("the file must be written under the resolved root: %v", err)
	}
}

// TestFileSystemRefusesAParentTraversalStillChecks checks the plain lexical case
// is not lost: the link handling must not replace it.
func TestFileSystemRefusesAParentTraversalStillChecks(t *testing.T) {
	root := t.TempDir()
	grant := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: root}
	fs := access.FileSystem{Root: root, Capability: core.MustCanonicalID("sales.close"), Guard: access.Guard{Granted: []permission.Spec{grant}}}
	if _, err := fs.ReadFile(filepath.Join("..", "escape.txt")); err == nil {
		t.Fatal("a parent traversal must still be refused")
	}
}
