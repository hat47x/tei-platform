package lifecyclesqlite

import (
	"path/filepath"
	"sync"
	"testing"

	"tei/internal/command"
)

func TestStorePersistsStagesAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.db")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := first.MarkUnknown("cmd-1", "owner-a", 1, "response lost"); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	attempt, ok := second.Attempt("cmd-1")
	if !ok || attempt.Stage != command.StageUnknown || attempt.Reason != "response lost" {
		t.Fatalf("reopened attempt=%+v ok=%v", attempt, ok)
	}
	if second.MayReplay("cmd-1") {
		t.Fatal("an unknown attempt was reported as replayable after reopen")
	}
	if _, err := second.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("a same-attempt replay was accepted after reopen")
	}
	if _, err := second.Takeover("cmd-1", "owner-b", 2); err != nil {
		t.Fatal(err)
	}
	if err := second.Observe("cmd-1", "owner-b", 2, command.OutcomeCommitted); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Dispatch("cmd-1", "owner-c", 3); err == nil {
		t.Fatal("an observed attempt was re-dispatched")
	}
}

func TestStoreSerializesConcurrentDispatches(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "lifecycle.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	const racers = 4
	var wg sync.WaitGroup
	var mu sync.Mutex
	succeeded := 0
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.Dispatch("cmd-1", "owner-a", 1); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if succeeded != 1 {
		t.Fatalf("dispatches succeeded=%d, want exactly 1", succeeded)
	}
	if attempt, _ := store.Attempt("cmd-1"); attempt.Stage != command.StageDispatched {
		t.Fatalf("final attempt=%+v", attempt)
	}
}
