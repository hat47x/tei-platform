//go:build linux

package workspace_test

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"tei/internal/workspace"
)

func TestReadSourceRejectsFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "fifo.xead")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := workspace.ReadSource(root, "fifo.xead", 1024)
		result <- err
	}()
	timedOut := false
	t.Cleanup(func() {
		if !timedOut {
			return
		}
		if err := releaseBlockedFIFOReader(fifo, done); err != nil {
			t.Errorf("release blocked FIFO reader: %v", err)
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("FIFO reader goroutine did not exit after cleanup")
		}
	})

	select {
	case err := <-result:
		<-done
		if err == nil {
			t.Fatal("ReadSource accepted a FIFO")
		}
	case <-time.After(2 * time.Second):
		timedOut = true
		t.Error("ReadSource blocked while opening a FIFO")
	}
}

func releaseBlockedFIFOReader(path string, done <-chan struct{}) error {
	deadline := time.Now().Add(2 * time.Second)
	for {
		select {
		case <-done:
			return nil
		default:
		}
		fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			return syscall.Close(fd)
		}
		if !errors.Is(err, syscall.ENXIO) && !errors.Is(err, syscall.EINTR) {
			return err
		}
		if time.Now().After(deadline) {
			return errors.New("no reader appeared before release deadline")
		}
		time.Sleep(time.Millisecond)
	}
}
