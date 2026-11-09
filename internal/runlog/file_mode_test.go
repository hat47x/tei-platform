package runlog_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/runlog"
)

// TestFileIsOwnerOnly checks the run journal's permissions. A process that can
// rewrite the run journal can make a completed run look unfinished (or the
// reverse), so the file cannot be shared.
func TestFileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not observable on Windows; the Linux CI run checks it")
	}
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	if err := (runlog.File{Path: path}).Append(runlog.Record{
		RunKey: "run-1", PlanID: "sales.close", Attempt: 1, FenceEpoch: 1, Status: runlog.Running,
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("the run journal is readable or writable by others: %v", info.Mode().Perm())
	}
}
