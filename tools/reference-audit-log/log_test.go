package auditlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newLog(t *testing.T) (*Log, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	log, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return log, path
}

func TestAppendAndVerifyChain(t *testing.T) {
	log, _ := newLog(t)
	first, err := log.Append("tenant-a", "user-1", "read", "order/1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := log.Append("tenant-a", "user-1", "write", "order/1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Seq != 1 || first.PrevHash != genesisHash || first.Hash == "" {
		t.Fatalf("unexpected first entry: %+v", first)
	}
	if second.Seq != 2 || second.PrevHash != first.Hash || second.Hash == "" {
		t.Fatalf("unexpected second entry: %+v", second)
	}
	if err := log.Verify(); err != nil {
		t.Fatal(err)
	}
	entries, err := log.Entries()
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
}

func TestTamperedEntryIsDetected(t *testing.T) {
	log, path := newLog(t)
	if _, err := log.Append("tenant-a", "user-1", "read", "order/1"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), `"action":"read"`, `"action":"write"`, 1)
	if tampered == string(raw) {
		t.Fatal("test did not find the action field")
	}
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := log.Verify(); err == nil {
		t.Fatal("tampered entry was verified")
	}
	if _, err := log.Append("tenant-a", "user-1", "read", "order/2"); err == nil {
		t.Fatal("append to a tampered log was accepted")
	}
}

func TestEmptyTenantOrActionIsRejected(t *testing.T) {
	log, _ := newLog(t)
	if _, err := log.Append("", "user-1", "read", "order/1"); err == nil {
		t.Fatal("empty tenant was accepted")
	}
	if _, err := log.Append("tenant-a", "user-1", "", "order/1"); err == nil {
		t.Fatal("empty action was accepted")
	}
}

func TestReopenContinuesTheChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := first.Append("tenant-a", "user-1", "read", "order/1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	next, err := second.Append("tenant-a", "user-2", "read", "order/2")
	if err != nil {
		t.Fatal(err)
	}
	if next.Seq != 2 || next.PrevHash != entry.Hash {
		t.Fatalf("reopened log broke the chain: %+v", next)
	}
	if err := second.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestHeadPinningDetectsSuffixDeletion(t *testing.T) {
	log, path := newLog(t)
	if _, err := log.Append("tenant-a", "user-1", "read", "order/1"); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Append("tenant-a", "user-1", "write", "order/1"); err != nil {
		t.Fatal(err)
	}
	head, err := log.Head()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines=%d", len(lines))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines[:1], "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := log.Verify(); err != nil {
		t.Fatalf("without a pinned head, the remaining chain is valid: %v", err)
	}
	if err := log.VerifyHead(head); err == nil {
		t.Fatal("pinned head did not detect suffix deletion")
	}
}

func TestHeadOfEmptyLogIsGenesis(t *testing.T) {
	log, _ := newLog(t)
	head, err := log.Head()
	if err != nil {
		t.Fatal(err)
	}
	if head != genesisHash {
		t.Fatalf("head=%s", head)
	}
}
