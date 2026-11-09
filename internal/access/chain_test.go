package access_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// chain writes two decisions and returns the journal path.
func chain(t *testing.T) (string, access.ChainFile) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "access.jsonl")
	journal := access.ChainFile{Path: path}
	guard := access.Guard{
		Granted: nil,
		Journal: journal,
	}
	for _, capability := range []string{"sales.close", "sales.reopen"} {
		// Require something nothing is granted, so the journal holds refused
		// decisions. A journal of allowed decisions cannot show that a flip is
		// detected.
		if _, err := guard.Check(core.MustCanonicalID(capability), []permission.Spec{
			{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"},
		}); err != nil {
			t.Fatalf("check %s: %v", capability, err)
		}
	}
	return path, journal
}

func TestChainFileRecordsEveryDecision(t *testing.T) {
	path, journal := chain(t)
	records, err := journal.Records()
	if err != nil {
		t.Fatalf("records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("both decisions must be kept: %+v", records)
	}
	// The plain reader must read a chained journal too: the decision fields are
	// inline, so a reader that does not know about the chain still works.
	plain, err := (access.File{Path: path}).Records()
	if err != nil || len(plain) != 2 || plain[0].Capability != "sales.close" {
		t.Fatalf("a chained journal must stay readable as decisions: %+v err=%v", plain, err)
	}
	if err := journal.Verify(); err != nil {
		t.Fatalf("an untouched journal must verify: %v", err)
	}
}

func TestChainFileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows does not report the POSIX permission bits this check is about;
		// the file mode passed to OpenFile is honoured by the CI (Linux) run.
		t.Skip("file permission bits are not observable on Windows")
	}
	path, _ := chain(t)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("the journal must not be readable or writable by others: %v", info.Mode().Perm())
	}
}

func TestChainFileDetectsAChangedDecision(t *testing.T) {
	path, journal := chain(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Flip the recorded decision of the first record. The chain must notice: a
	// journal that records "refused" and later reads "allowed" is worse than no
	// journal at all.
	changed := strings.Replace(string(raw), `"allowed":false`, `"allowed":true`, 1)
	if changed == string(raw) {
		t.Fatalf("the fixture must contain a refused decision to change: %s", raw)
	}
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := journal.Verify(); err == nil {
		t.Fatal("a changed decision must be reported")
	}
}

func TestChainFileDetectsARemovedDecision(t *testing.T) {
	path, journal := chain(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected fixture: %q", lines)
	}
	// Removing the *first* record breaks the link of the second one; removing the
	// last one is what a pinned head catches, tested separately.
	if err := os.WriteFile(path, []byte(lines[1]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := journal.Verify(); err == nil {
		t.Fatal("a removed record must be reported")
	}
}

func TestChainFileNeedsAPinnedHeadToDetectATruncatedSuffix(t *testing.T) {
	path, journal := chain(t)
	head, err := journal.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if err := os.WriteFile(path, []byte(lines[0]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Without a pinned head a shortened chain is still a valid chain. This is the
	// limit, stated as a test so nobody reads the chain as complete protection.
	if err := journal.Verify(); err != nil {
		t.Fatalf("a truncated suffix is invisible without a pinned head: %v", err)
	}
	if err := journal.VerifyHead(head); err == nil {
		t.Fatal("a pinned head must catch the truncation")
	}
}

func TestChainFileRefusesToAppendToAChangedJournal(t *testing.T) {
	path, journal := chain(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), `"capability":"sales.close"`, `"capability":"sales.other"`, 1)
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	// Appending to a changed journal would extend a chain the reader can no longer
	// trust, and would put a valid hash after an invalid link.
	if err := journal.Append(access.Record{Capability: "sales.after", Allowed: true}); err == nil {
		t.Fatal("appending to a changed journal must be refused")
	}
}

func TestChainFileDetectsAForgedPreviousHash(t *testing.T) {
	path, journal := chain(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var first map[string]any
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	first["prevHash"] = "sha256:" + strings.Repeat("a", 64)
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(string(encoded)+"\n"+lines[1]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := journal.Verify(); err == nil {
		t.Fatal("a forged previous hash must be reported")
	}
}

func TestChainFileOfAnEmptyJournalVerifies(t *testing.T) {
	journal := access.ChainFile{Path: filepath.Join(t.TempDir(), "none.jsonl")}
	if err := journal.Verify(); err != nil {
		t.Fatalf("a missing journal has no records to contradict: %v", err)
	}
	head, err := journal.Head()
	if err != nil || head == "" {
		t.Fatalf("an empty journal has a genesis head: %q err=%v", head, err)
	}
}
