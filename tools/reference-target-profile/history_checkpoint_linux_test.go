//go:build linux

package targetprofile

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestHistoryAuditCheckpointUsesOneRead(t *testing.T) {
	if root := os.Getenv("TEI_CHECKPOINT_TEST_ROOT"); root != "" {
		checkpoint, err := BuildHistoryAuditCheckpoint(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(checkpoint); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	record := historyAuditRecord{Schema: HistoryAuditSchema, Operation: "prune-history", RecordedAt: "2026-10-29T00:00:00Z", Storage: StorageFilesystem, KeepLast: 1, Deleted: []string{digestBytes([]byte("one"))}}
	var err error
	record.RecordDigest, err = historyAuditDigest(record)
	if err != nil {
		t.Fatal(err)
	}
	first, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	first = append(first, '\n')
	firstHead := record.RecordDigest
	record.PreviousDigest = record.RecordDigest
	record.Deleted = []string{digestBytes([]byte("two"))}
	record.RecordDigest, err = historyAuditDigest(record)
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	second = append(append(append([]byte(nil), first...), second...), '\n')
	root := t.TempDir()
	fifo := historyAuditPath(root)
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	watch, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(watch)
	if _, err := syscall.InotifyAddWatch(watch, fifo, syscall.IN_CLOSE_NOWRITE); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestHistoryAuditCheckpointUsesOneRead$")
	cmd.Env = append(os.Environ(), "TEI_CHECKPOINT_TEST_ROOT="+root)
	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var waitErr error
	go func() { waitErr = cmd.Wait(); close(done) }()
	// The stream supplies different generations if the function opens it twice.
	send := func(payload []byte) bool {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-done:
				return false
			default:
			}
			fd, err := syscall.Open(fifo, syscall.O_WRONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
			if err == nil {
				f := os.NewFile(uintptr(fd), filepath.Base(fifo))
				n, e := f.Write(payload)
				closeErr := f.Close()
				if e != nil || closeErr != nil || n != len(payload) {
					t.Fatalf("stream write: %d %v %v", n, e, closeErr)
				}
				return true
			}
			if err != syscall.ENXIO {
				t.Fatal(err)
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("checkpoint did not open the audit stream")
		return false
	}
	if !send(first) {
		t.Fatal("checkpoint exited before its first read")
	}
	// Wait for the first reader to close, so a second generation cannot be
	// appended into the first ReadFile. This avoids a timing-based race test.
	event := make([]byte, 4096)
	deadline := time.Now().Add(10 * time.Second)
	for {
		n, err := syscall.Read(watch, event)
		if n > 0 {
			break
		}
		if err != nil && err != syscall.EAGAIN {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("first audit reader did not close")
		}
		time.Sleep(time.Millisecond)
	}
	send(second)
	<-done
	if waitErr != nil {
		t.Fatalf("checkpoint process: %v %s", waitErr, stderr.String())
	}
	var got HistoryAuditCheckpoint
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("checkpoint JSON: %v %s", err, output.String())
	}
	if !got.Passed || got.Count != 1 || got.HeadDigest != firstHead || got.AuditDigest != digestBytes(first) {
		t.Fatalf("checkpoint mixed audit generations: %+v", got)
	}
}
