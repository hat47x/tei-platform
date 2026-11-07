package effectlog_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/effectlog"
)

// TestFileIsOwnerOnly checks the effect journal's permissions. A process that can
// rewrite it can erase an effect that happened, so the file cannot be shared.
func TestFileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not observable on Windows; the Linux CI run checks it")
	}
	path := filepath.Join(t.TempDir(), "effects.jsonl")
	journal := effectlog.File{Path: path}
	if err := journal.Append(effectlog.Record{
		RuleID: "sales.order.stock", Stage: "after-commit", Capability: "sales.stock.decrement", Succeeded: true, Observed: true,
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("the effect journal is readable or writable by others: %v", info.Mode().Perm())
	}
}
