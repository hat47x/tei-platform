//go:build !tei_repair_only

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatchJournalExclusiveStorageAndFailureIsSticky(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if _, err := openBatchJournal(root, filepath.Join(root, "log")); err == nil {
		t.Fatal("model workspace overwritten")
	}
	path := filepath.Join(outside, "progress.jsonl")
	j, err := openBatchJournal(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Append(batchJournalEvent{Kind: "planned", Planned: []int64{1001}}); err != nil {
		t.Fatal(err)
	}
	if _, err := openBatchJournal(root, path); err == nil {
		t.Fatal("existing journal replaced")
	}
	if err := j.file.Close(); err != nil {
		t.Fatal(err)
	}
	if j.Append(batchJournalEvent{Kind: "starting", OrderID: 1001}) == nil || !j.failed {
		t.Fatal("closed-file failure ignored")
	}
	if j.Append(batchJournalEvent{Kind: "ended"}) == nil || j.sequence != 1 {
		t.Fatal("failed journal resumed")
	}
	raw, err := os.ReadFile(path)
	if err != nil || strings.Count(string(raw), "\n") != 1 {
		t.Fatal("saved prefix lost")
	}
}

func TestBatchJournalKilledProcessRetainsSyncedPrefix(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	path := filepath.Join(outside, "progress.jsonl")
	cmd := exec.Command(os.Args[0], "-test.run=^TestBatchJournalChild$")
	cmd.Env = append(os.Environ(), "TEI_BATCH_JOURNAL_CHILD_ROOT="+root, "TEI_BATCH_JOURNAL_CHILD_PATH="+path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	ready, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || ready != "synced\n" {
		t.Fatalf("child not ready: %q %v", ready, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("saved prefix: %s", raw)
	}
	for i, line := range lines {
		var e batchJournalEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil || e.Sequence != uint64(i+1) || e.Schema != "tei.reference.batch-progress/v1" {
			t.Fatal("journal corrupted")
		}
	}
}

func TestBatchJournalChild(t *testing.T) {
	path := os.Getenv("TEI_BATCH_JOURNAL_CHILD_PATH")
	if path == "" {
		return
	}
	j, err := openBatchJournal(os.Getenv("TEI_BATCH_JOURNAL_CHILD_ROOT"), path)
	if err != nil {
		t.Fatal(err)
	}
	defer j.file.Close()
	for _, e := range []batchJournalEvent{{Kind: "planned", Planned: []int64{1001}}, {Kind: "starting", OrderID: 1001}} {
		if err := j.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Println("synced")
	_, _ = bufio.NewReader(os.Stdin).ReadByte()
}
