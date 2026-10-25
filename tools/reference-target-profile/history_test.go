package targetprofile

import (
	"os"
	"path/filepath"
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
