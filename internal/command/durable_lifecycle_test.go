package command

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDurableLifecyclePersistsUnknownAndObservedStages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.json")
	first, err := OpenDurableLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := first.MarkUnknown("cmd-1", "owner-a", 1, "process exited"); err != nil {
		t.Fatal(err)
	}

	second, err := OpenDurableLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok := second.Attempt("cmd-1")
	if !ok || attempt.Stage != StageUnknown || attempt.Reason != "process exited" {
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
	if err := second.Observe("cmd-1", "owner-b", 2, OutcomeCommitted); err != nil {
		t.Fatal(err)
	}

	third, err := OpenDurableLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok = third.Attempt("cmd-1")
	if !ok || attempt.Stage != StageObserved || attempt.Outcome != OutcomeCommitted || attempt.TakeoverFrom != 1 {
		t.Fatalf("reopened observed attempt=%+v ok=%v", attempt, ok)
	}
	if _, err := third.Dispatch("cmd-1", "owner-c", 3); err == nil {
		t.Fatal("an observed attempt was re-dispatched after reopen")
	}
}

func TestDurableLifecycleRejectsCorruptOrUnknownState(t *testing.T) {
	directory := t.TempDir()
	cases := map[string]string{
		"not json":        "{not json",
		"unknown schema":  `{"schema":"tei.command-lifecycle/v999","attempts":[]}`,
		"missing command": `{"schema":"tei.command-lifecycle/v0","attempts":[{"stage":"unknown"}]}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(directory, name+".json")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenDurableLifecycle(path); err == nil {
				t.Fatalf("corrupt state %q was accepted", content)
			}
		})
	}
}

func TestDurableLifecycleStartsEmptyWhenTheFileIsMissing(t *testing.T) {
	store, err := OpenDurableLifecycle(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !store.MayReplay("cmd-1") {
		t.Fatal("a missing state file did not start empty")
	}
	if _, ok := store.Attempt("cmd-1"); ok {
		t.Fatal("a missing state file produced an attempt")
	}
}
