package command

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLifecycleHelperProcess is the child side of the restart test. It marks an
// attempt unknown and exits, so the parent can verify that the state survives a
// real process boundary rather than only a new in-memory instance.
func TestLifecycleHelperProcess(t *testing.T) {
	if os.Getenv("TEI_LIFECYCLE_HELPER") != "1" {
		t.Skip("helper process only")
	}
	store, err := OpenDurableLifecycle(os.Getenv("TEI_LIFECYCLE_STATE"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Dispatch("cmd-restart", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkUnknown("cmd-restart", "owner-a", 1, "helper process exited before observing"); err != nil {
		t.Fatal(err)
	}
}

// #2564 RCAS-02 with a real process restart: the helper marks the attempt
// unknown and exits; a fresh process must still refuse the same attempt and
// recover only through a takeover followed by an observation.
func TestDurableLifecycleSurvivesARealProcessRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lifecycle.json")
	helper := exec.Command(os.Args[0], "-test.run=^TestLifecycleHelperProcess$")
	helper.Env = append(os.Environ(),
		"TEI_LIFECYCLE_HELPER=1",
		"TEI_LIFECYCLE_STATE="+path,
	)
	output, err := helper.CombinedOutput()
	if err != nil {
		t.Fatalf("helper process failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS") {
		t.Fatalf("helper process did not report PASS:\n%s", output)
	}

	// This process never saw the helper's in-memory state.
	restarted, err := OpenDurableLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok := restarted.Attempt("cmd-restart")
	if !ok || attempt.Stage != StageUnknown || attempt.Owner != "owner-a" || attempt.Generation != 1 {
		t.Fatalf("restarted attempt=%+v ok=%v", attempt, ok)
	}
	if restarted.MayReplay("cmd-restart") {
		t.Fatal("the restarted process reported an unknown attempt as replayable")
	}
	if _, err := restarted.Dispatch("cmd-restart", "owner-a", 1); err == nil {
		t.Fatal("the restarted process replayed the same attempt")
	}

	// Recovery requires a takeover and an observation.
	if _, err := restarted.Takeover("cmd-restart", "owner-b", 2); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Observe("cmd-restart", "owner-b", 2, OutcomeCommitted); err != nil {
		t.Fatal(err)
	}

	// A third process sees the observed outcome and still refuses a replay.
	again, err := OpenDurableLifecycle(path)
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok = again.Attempt("cmd-restart")
	if !ok || attempt.Stage != StageObserved || attempt.Outcome != OutcomeCommitted ||
		attempt.Owner != "owner-b" || attempt.TakeoverFrom != 1 {
		t.Fatalf("third process attempt=%+v ok=%v", attempt, ok)
	}
	if _, err := again.Dispatch("cmd-restart", "owner-c", 3); err == nil {
		t.Fatal("the third process re-dispatched an observed attempt")
	}
	if again.MayReplay("cmd-restart") {
		t.Fatal("the third process reported an observed attempt as replayable")
	}
}
