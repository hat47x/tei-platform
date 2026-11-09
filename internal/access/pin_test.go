package access_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// pinnedJournal builds a chained journal with its head pinned next to it.
func pinnedJournal(t *testing.T) (access.PinnedJournal, string) {
	t.Helper()
	dir := t.TempDir()
	journal := access.PinnedJournal{
		Journal: access.ChainFile{Path: filepath.Join(dir, "access.jsonl")},
		Pin:     access.HeadPin{Path: filepath.Join(dir, "access.head")},
	}
	guard := access.Guard{Journal: journal}
	for _, capability := range []string{"sales.close", "sales.reopen"} {
		if _, err := guard.Check(core.MustCanonicalID(capability), []permission.Spec{
			{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"},
		}); err != nil {
			t.Fatalf("check %s: %v", capability, err)
		}
	}
	return journal, journal.Journal.Path
}

func TestPinnedJournalRecordsAndPinsTheHead(t *testing.T) {
	journal, path := pinnedJournal(t)
	records, err := journal.Journal.Records()
	if err != nil || len(records) != 2 {
		t.Fatalf("both decisions must be kept: %+v err=%v", records, err)
	}
	head, err := journal.Journal.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	pinned, err := journal.Pin.Read()
	if err != nil || pinned != head {
		t.Fatalf("the pin must match the journal: pinned=%q head=%q err=%v", pinned, head, err)
	}
	if err := journal.Verify(); err != nil {
		t.Fatalf("an untouched journal must verify against its pin: %v", err)
	}
	_ = path
}

// TestPinnedJournalDetectsATruncatedSuffix is the case the chain alone cannot
// catch: removing the last record leaves a valid chain, and only the pin
// notices.
func TestPinnedJournalDetectsATruncatedSuffix(t *testing.T) {
	journal, path := pinnedJournal(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := 0
	for _, line := range []byte(raw) {
		if line == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Fatalf("unexpected fixture: %d lines", lines)
	}
	// Drop the last record.
	text := string(raw)
	lastNewline := len(text) - 1
	for lastNewline >= 0 && text[lastNewline] != '\n' {
		lastNewline--
	}
	previousNewline := lastNewline - 1
	for previousNewline >= 0 && text[previousNewline] != '\n' {
		previousNewline--
	}
	if err := os.WriteFile(path, []byte(text[:previousNewline+1]), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := journal.Journal.Verify(); err != nil {
		t.Fatalf("the shortened chain is still internally consistent: %v", err)
	}
	if err := journal.Verify(); err == nil {
		t.Fatal("the pin must catch the removed record")
	}
}

func TestPinnedJournalRefusesToAppendAfterTruncation(t *testing.T) {
	journal, path := pinnedJournal(t)
	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := journal.Append(access.Record{Capability: "sales.after", Allowed: true}); err == nil {
		t.Fatal("appending to a truncated journal must be refused: it would extend a chain the pin does not cover")
	}
}

func TestPinnedJournalNeedsAPin(t *testing.T) {
	journal := access.PinnedJournal{Journal: access.ChainFile{Path: filepath.Join(t.TempDir(), "access.jsonl")}}
	if err := journal.Append(access.Record{Capability: "sales.close", Allowed: true}); err == nil {
		t.Fatal("a journal without a pin must not claim the suffix is checked")
	}
}

func TestHeadPinRefusesTwoHeads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.head")
	if err := os.WriteFile(path, []byte("sha256:aaa\nsha256:bbb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (access.HeadPin{Path: path}).Read(); err == nil {
		t.Fatal("a pin that holds two heads pins neither")
	}
}

func TestHeadPinIgnoresCommentsAndBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.head")
	content := "# the last hash of the access journal\n\nsha256:" + "0123456789abcdef" + "0123456789abcdef" + "0123456789abcdef" + "0123456789abcdef\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	head, err := (access.HeadPin{Path: path}).Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if head != "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected head: %q", head)
	}
}

func TestHeadPinOfAMissingFileIsEmpty(t *testing.T) {
	head, err := (access.HeadPin{Path: filepath.Join(t.TempDir(), "none.head")}).Read()
	if err != nil || head != "" {
		t.Fatalf("a missing pin must read as empty, not as an error: %q err=%v", head, err)
	}
}
