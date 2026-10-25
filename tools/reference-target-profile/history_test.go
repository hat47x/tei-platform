package targetprofile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeHistoryArtifact(t *testing.T, root, content string, modified time.Time) string {
	t.Helper()
	dir := filepath.Join(root, "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	digest := strings.TrimPrefix(digestBytes([]byte(content)), "sha256:")
	path := filepath.Join(dir, digest)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHistoryListsAndVerifiesArtifacts(t *testing.T) {
	root := t.TempDir()
	first := writeHistoryArtifact(t, root, "first", time.Unix(100, 0))
	writeHistoryArtifact(t, root, "second", time.Unix(200, 0))
	receipt, err := History(root)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Entries) != 2 {
		t.Fatalf("history did not verify: %+v", receipt)
	}
	for _, entry := range receipt.Entries {
		if !entry.Verified || entry.Bytes == 0 {
			t.Fatalf("history entry is incomplete: %+v", entry)
		}
	}
	if err := os.WriteFile(first, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	receipt, err = History(root)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || receipt.Reason == "" {
		t.Fatalf("tampered history was accepted: %+v", receipt)
	}
}

func TestPruneHistoryKeepsNewestAndDryRun(t *testing.T) {
	root := t.TempDir()
	writeHistoryArtifact(t, root, "old", time.Unix(100, 0))
	writeHistoryArtifact(t, root, "middle", time.Unix(200, 0))
	newest := writeHistoryArtifact(t, root, "newest", time.Unix(300, 0))

	dryRun, err := PruneHistory(root, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if !dryRun.Passed || len(dryRun.Kept) != 1 || len(dryRun.WouldDelete) != 2 || len(dryRun.Deleted) != 0 {
		t.Fatalf("unexpected dry-run receipt: %+v", dryRun)
	}
	if _, err := os.Stat(newest); err != nil {
		t.Fatalf("dry-run deleted the newest artifact: %v", err)
	}
	if entries, err := os.ReadDir(filepath.Join(root, "history")); err != nil || len(entries) != 3 {
		t.Fatalf("dry-run changed history: %v %d", err, len(entries))
	}

	pruned, err := PruneHistory(root, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if !pruned.Passed || len(pruned.Kept) != 1 || len(pruned.Deleted) != 2 {
		t.Fatalf("unexpected prune receipt: %+v", pruned)
	}
	entries, err := os.ReadDir(filepath.Join(root, "history"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(newest) {
		t.Fatalf("prune kept the wrong artifacts: %+v", entries)
	}
}

func TestPruneHistoryRefusesUnverifiedArtifact(t *testing.T) {
	root := t.TempDir()
	path := writeHistoryArtifact(t, root, "broken", time.Unix(100, 0))
	writeHistoryArtifact(t, root, "newer", time.Unix(200, 0))
	if err := os.WriteFile(path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	receipt, err := PruneHistory(root, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || receipt.Reason == "" || len(receipt.Deleted) != 0 {
		t.Fatalf("unverified history was pruned: %+v", receipt)
	}
}

func TestPruneHistoryRejectsNonPositiveKeep(t *testing.T) {
	receipt, err := PruneHistory(t.TempDir(), 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || receipt.Reason == "" {
		t.Fatalf("invalid keepLast was accepted: %+v", receipt)
	}
}

func TestHistoryRejectsSymbolicLinks(t *testing.T) {
	for _, directoryLink := range []bool{false, true} {
		t.Run(map[bool]string{false: "artifact-link", true: "history-directory-link"}[directoryLink], func(t *testing.T) {
			root, external := t.TempDir(), t.TempDir()
			target := writeHistoryArtifact(t, external, "external", time.Unix(100, 0))
			if directoryLink {
				if err := os.Symlink(filepath.Join(external, "history"), filepath.Join(root, "history")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			} else {
				if err := os.Mkdir(filepath.Join(root, "history"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(root, "history", filepath.Base(target))); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			}
			receipt, err := History(root)
			if err == nil && receipt.Passed {
				t.Fatal("history followed a symlink and verified external bytes")
			}
			pruned, err := PruneHistory(root, 1, false)
			if err == nil && pruned.Passed {
				t.Fatal("prune accepted a symbolic link")
			}
			raw, err := os.ReadFile(target)
			if err != nil || string(raw) != "external" {
				t.Fatalf("external artifact changed: %q %v", raw, err)
			}
			if _, err := os.Lstat(filepath.Join(root, ".update-lock")); !os.IsNotExist(err) {
				t.Fatalf("failed prune left lock: %v", err)
			}
		})
	}
}

func TestPruneHistoryReportsOnlyCompletedDeletions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX directory permissions")
	}
	root := t.TempDir()
	writeHistoryArtifact(t, root, "old", time.Unix(100, 0))
	writeHistoryArtifact(t, root, "new", time.Unix(200, 0))
	dir := filepath.Join(root, "history")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	// Privileged users can bypass directory permissions; do not count that as proof.
	probe := filepath.Join(dir, "permission-probe")
	if err := os.WriteFile(probe, nil, 0o600); err == nil {
		os.Remove(probe)
		t.Skip("directory permissions are bypassed by this user/filesystem")
	}
	receipt, err := PruneHistory(root, 1, false)
	if err != nil || receipt.Passed || receipt.Reason == "" || len(receipt.Deleted) != 0 {
		t.Fatalf("failed deletion reported success or a deleted file: %+v %v", receipt, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("failed deletion changed history: %v %v", entries, err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".update-lock")); !os.IsNotExist(err) {
		t.Fatalf("failed deletion left lock: %v", err)
	}
}
