package expectedstateupdate

import (
	"strings"
	"testing"
)

// #2564 -> #1129: an unknown CAS outcome must not be replayed by the same
// owner/generation. A takeover needs a higher generation and an observation
// before success can be claimed.
func TestRuntimeStageStopsReplayAfterUnknown(t *testing.T) {
	stage := NewRuntimeStage()
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := stage.MarkUnknown("cmd-1", "owner-a", 1, "response lost"); err != nil {
		t.Fatal(err)
	}
	if stage.MayReplay("cmd-1") {
		t.Fatal("unknown outcome was reported as replayable")
	}
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err == nil {
		t.Fatal("the same owner/generation replayed an unknown command")
	}
	if _, err := stage.Takeover("cmd-1", "owner-b", 1); err == nil {
		t.Fatal("takeover with an equal generation was accepted")
	}
}

func TestRuntimeStageTakeoverRequiresObservationBeforeSuccess(t *testing.T) {
	stage := NewRuntimeStage()
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := stage.MarkUnknown("cmd-1", "owner-a", 1, "timeout"); err != nil {
		t.Fatal(err)
	}
	attempt, err := stage.Takeover("cmd-1", "owner-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.TakeoverFrom != 1 || attempt.Owner != "owner-b" || attempt.Stage != StageDispatched {
		t.Fatalf("unexpected takeover attempt: %+v", attempt)
	}
	// The takeover itself is not success.
	if current, _ := stage.Attempt("cmd-1"); current.Stage != StageDispatched || current.Outcome != "" {
		t.Fatalf("takeover claimed an outcome: %+v", current)
	}
	if err := stage.Observe("cmd-1", "owner-b", 2, OutcomeApplied); err != nil {
		t.Fatal(err)
	}
	current, _ := stage.Attempt("cmd-1")
	if current.Stage != StageObserved || current.Outcome != OutcomeApplied {
		t.Fatalf("observation was not recorded: %+v", current)
	}
	if _, err := stage.Dispatch("cmd-1", "owner-c", 3); err == nil {
		t.Fatal("an observed command was re-dispatched")
	}
}

func TestRuntimeStageObserveRequiresMatchingAttempt(t *testing.T) {
	stage := NewRuntimeStage()
	if err := stage.Observe("cmd-1", "owner-a", 1, OutcomeApplied); err == nil {
		t.Fatal("observation without a dispatch was accepted")
	}
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := stage.Dispatch("cmd-1", "owner-a", 2); err == nil {
		t.Fatal("a second dispatch while in flight was accepted")
	}
	if err := stage.Observe("cmd-1", "owner-b", 1, OutcomeApplied); err == nil {
		t.Fatal("observation from a different owner was accepted")
	}
	if err := stage.Observe("cmd-1", "owner-a", 2, OutcomeApplied); err == nil {
		t.Fatal("observation from a different generation was accepted")
	}
	if err := stage.Observe("cmd-1", "owner-a", 1, ""); err == nil {
		t.Fatal("observation without an outcome was accepted")
	}
	if err := stage.Observe("cmd-1", "owner-a", 1, OutcomeConflict); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeStageRejectsTakeoverWithoutUnknownOutcome(t *testing.T) {
	stage := NewRuntimeStage()
	if _, err := stage.Dispatch("cmd-1", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := stage.Takeover("cmd-1", "owner-b", 2); err == nil {
		t.Fatal("takeover without an unknown outcome was accepted")
	}
	if err := stage.Observe("cmd-1", "owner-a", 1, OutcomeApplied); err != nil {
		t.Fatal(err)
	}
	if _, err := stage.Takeover("cmd-1", "owner-b", 2); err == nil {
		t.Fatal("takeover of an observed command was accepted")
	}
}

// The stage uses the same outcome codes as the CAS reference, so an APPLIED or
// CONFLICT observation is recorded without translating it into success text.
func TestRuntimeStageRecordsCASCodes(t *testing.T) {
	stage := NewRuntimeStage()
	for _, outcome := range []string{OutcomeApplied, OutcomeConflict, OutcomeMissing, OutcomeDenied} {
		command := "cmd-" + strings.ToLower(outcome)
		if _, err := stage.Dispatch(command, "owner-a", 1); err != nil {
			t.Fatal(err)
		}
		if err := stage.Observe(command, "owner-a", 1, outcome); err != nil {
			t.Fatal(err)
		}
		current, _ := stage.Attempt(command)
		if current.Outcome != outcome {
			t.Fatalf("outcome=%q want %q", current.Outcome, outcome)
		}
	}
}
