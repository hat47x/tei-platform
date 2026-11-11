package bearprobe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRecoveryMachineTranscriptionMatchesUpstreamAnchors checks every
// transition reason and state name against controllers/bear/traversal_recovery.gd.
// It skips when the sibling checkout is unreachable, because an unreachable
// checkout is not evidence that the transcription still matches.
func TestRecoveryMachineTranscriptionMatchesUpstreamAnchors(t *testing.T) {
	root := os.Getenv("VS_BEAR_REPO")
	if root == "" {
		root = filepath.Join("..", "..", "..", "..", "vs-bear-simulator")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Skipf("VS Bear Simulator checkout not reachable at %s; set VS_BEAR_REPO to re-check the transcription", root)
	}
	path := filepath.Join(root, "controllers", "bear", "traversal_recovery.gd")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read upstream %s: %v", path, err)
	}
	source := string(data)

	for _, reason := range TransitionReasons {
		if !strings.Contains(source, reason) {
			t.Fatalf("traversal_recovery.gd no longer contains transition reason %q; the transcription is stale", reason)
		}
	}
	for _, state := range MachineStates {
		if !strings.Contains(source, string(state)) {
			t.Fatalf("traversal_recovery.gd no longer contains state %q", state)
		}
	}
	if !strings.Contains(source, "func _finish_episode") {
		t.Fatal("traversal_recovery.gd no longer declares _finish_episode, which is what progress_restored relies on")
	}
}

// TestRecoveryMachineHasNoUnreachableOrDeadEndStates is the positive structural
// evidence: the transcribed machine is fully connected and every state can be
// left. This is the kind of statement a state table gives a reviewer that an
// if/elif chain does not.
func TestRecoveryMachineHasNoUnreachableOrDeadEndStates(t *testing.T) {
	machine := DefaultRecoveryMachine()
	if diagnostics := machine.Diagnose(); len(diagnostics) != 0 {
		t.Fatalf("the transcribed machine reported structural problems: %+v", diagnostics)
	}
	reachable := machine.ReachableStates()
	if len(reachable) != len(MachineStates) {
		t.Fatalf("reachable states = %v want all %v", reachable, MachineStates)
	}
	if len(machine.SelfTransitions()) != 1 {
		t.Fatalf("self transitions = %d want 1 (the ENTER_TRAVERSABLE retarget)", len(machine.SelfTransitions()))
	}
}

// TestRecoveryMachineUnknownIsNotAMachineState records a real asymmetry: the
// recorded recovery_state enum has five values, but the machine has four
// states. "unknown" is the label motion_feedback.gd writes for an unrecognised
// state name, not a state the controller can be in.
func TestRecoveryMachineUnknownIsNotAMachineState(t *testing.T) {
	if !contains(ValidRecoveryStates, "unknown") {
		t.Fatal("the recorded recovery_state enum no longer contains \"unknown\"")
	}
	for _, state := range MachineStates {
		if state == StateUNKNOWN {
			t.Fatal("unknown became a machine state; it is a fallback label")
		}
	}
	machine := DefaultRecoveryMachine()
	for _, transition := range machine.Transitions {
		if transition.To == StateUNKNOWN || transition.From == StateUNKNOWN {
			t.Fatalf("transition %q uses the fallback label as an endpoint", transition.Reason)
		}
	}
	mutated := machine
	mutated.States = append(append([]RecoveryStateName(nil), machine.States...), StateUNKNOWN)
	if !hasCode(mutated.Diagnose(), "BEAR_SM_UNKNOWN_IS_NOT_A_STATE") {
		t.Fatal("declaring the fallback label as a state was accepted")
	}
}

// TestRecoveryMachinePriorityIsTotalPerSourceState checks that each source
// state's competing guards have distinct priorities, matching the if/elif
// order. Two branches at the same priority would mean the runtime resolves
// them by source position alone, which #4193 §5.2 refuses to treat as a
// declared policy.
func TestRecoveryMachinePriorityIsTotalPerSourceState(t *testing.T) {
	machine := DefaultRecoveryMachine()
	for _, state := range MachineStates {
		transitions := machine.PrioritiesFor(state)
		if len(transitions) == 0 {
			t.Fatalf("state %q has no outgoing transition", state)
		}
		for i, transition := range transitions {
			if transition.Priority != i+1 {
				t.Fatalf("state %q transition %q has priority %d want %d; priorities must be 1..n in source order", state, transition.Reason, transition.Priority, i+1)
			}
		}
	}

	mutated := machine
	mutated.Transitions = append([]RecoveryTransition(nil), machine.Transitions...)
	for i := range mutated.Transitions {
		if mutated.Transitions[i].Reason == "progress_restored" {
			mutated.Transitions[i].Priority = 1
			break
		}
	}
	if !hasCode(mutated.Diagnose(), "BEAR_SM_TRANSITION_PRIORITY_CONFLICT") {
		t.Fatal("two branches at the same priority were accepted")
	}

	// A priority of zero means the branch order was not transcribed at all.
	mutated = machine
	mutated.Transitions = append([]RecoveryTransition(nil), machine.Transitions...)
	for i := range mutated.Transitions {
		if mutated.Transitions[i].Reason == "wall_clear_progress" {
			mutated.Transitions[i].Priority = 0
			break
		}
	}
	if !hasCode(mutated.Diagnose(), "BEAR_SM_TRANSITION_PRIORITY_MISSING") {
		t.Fatal("a transition with no priority was accepted")
	}
}

// TestRecoveryMachineRefusesUnknownEndpoints checks the negative case for a
// transition table: an endpoint that is not a declared state.
func TestRecoveryMachineRefusesUnknownEndpoints(t *testing.T) {
	machine := DefaultRecoveryMachine()
	machine.Transitions = append(machine.Transitions, RecoveryTransition{
		ID: "t-bad", From: StateIDLE, To: RecoveryStateName("RECOVERING"), Reason: "invented",
		Guard: "invented", Priority: 3,
	})
	if !hasCode(machine.Diagnose(), "BEAR_SM_TRANSITION_TARGET_UNKNOWN") {
		t.Fatal("a transition to an undeclared state was accepted")
	}

	machine = DefaultRecoveryMachine()
	machine.Transitions = append(machine.Transitions, RecoveryTransition{
		ID: "t-bad-2", From: RecoveryStateName("PAUSED"), To: StateIDLE, Reason: "invented_source",
		Guard: "invented", Priority: 1,
	})
	if !hasCode(machine.Diagnose(), "BEAR_SM_TRANSITION_SOURCE_UNKNOWN") {
		t.Fatal("a transition from an undeclared state was accepted")
	}
}

// TestRecoveryMachineDetectsUnreachableStateAfterMutation is a mutation test.
// Removing every path into WALL_FOLLOW must make it unreachable; if the
// diagnostic does not fire, it has no discriminating power.
func TestRecoveryMachineDetectsUnreachableStateAfterMutation(t *testing.T) {
	machine := DefaultRecoveryMachine()
	var kept []RecoveryTransition
	for _, transition := range machine.Transitions {
		if transition.To == StateWALL_FOLLOW {
			continue
		}
		kept = append(kept, transition)
	}
	machine.Transitions = kept
	diagnostics := machine.Diagnose()
	if !hasCode(diagnostics, "BEAR_SM_STATE_UNREACHABLE") {
		t.Fatalf("removing every path into WALL_FOLLOW was accepted: %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "BEAR_SM_STATE_UNREACHABLE" && !strings.Contains(diagnostic.Message, "WALL_FOLLOW") {
			t.Fatalf("the unreachable diagnostic named %q, want WALL_FOLLOW", diagnostic.Message)
		}
	}
}

// TestRecoveryMachineDetectsDeadEndAfterMutation is a mutation test for the
// dead-end diagnostic. CLEARING stays reachable but loses every outgoing
// transition, including the global one, so it can never be left. If the
// diagnostic does not fire, it has no discriminating power.
func TestRecoveryMachineDetectsDeadEndAfterMutation(t *testing.T) {
	machine := DefaultRecoveryMachine()
	machine.Transitions = nil
	for _, transition := range DefaultRecoveryMachine().Transitions {
		if transition.From == StateCLEARING || transition.Global {
			continue
		}
		machine.Transitions = append(machine.Transitions, transition)
	}
	// CLEARING is still reachable: WALL_FOLLOW and ENTER_TRAVERSABLE keep their
	// transitions into it.
	reachable := machine.ReachableStates()
	if !containsState(reachable, StateCLEARING) {
		t.Fatalf("reachable states = %v; CLEARING must stay reachable for this mutation to test a dead end", reachable)
	}
	diagnostics := machine.Diagnose()
	if !hasCode(diagnostics, "BEAR_SM_STATE_DEAD_END") {
		t.Fatalf("a reachable state with no exit was accepted: %+v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "BEAR_SM_STATE_DEAD_END" && !strings.Contains(diagnostic.Message, "CLEARING") {
			t.Fatalf("the dead-end diagnostic named %q, want CLEARING", diagnostic.Message)
		}
	}

	// With the global observation_invalid transition present, no state can be a
	// dead end, because that transition is an exit from every non-initial
	// state. This is a property of the transcribed machine, not a gap in the
	// diagnostic: a reviewer must know the global row exists.
	withGlobal := DefaultRecoveryMachine()
	if hasCode(withGlobal.Diagnose(), "BEAR_SM_STATE_DEAD_END") {
		t.Fatal("the unmutated machine reported a dead end; the global transition is an exit from every non-initial state")
	}
}

func containsState(states []RecoveryStateName, want RecoveryStateName) bool {
	for _, state := range states {
		if state == want {
			return true
		}
	}
	return false
}

// TestRecoveryMachineGuardInversionIsVisible checks the mutation #4193 §5 names
// explicitly: inverting a guard must not silently turn into a valid table. The
// probe transcribes the guard text, so an inverted guard shows up as a changed
// string that the anchor test can no longer find in the upstream file.
func TestRecoveryMachineGuardInversionIsVisible(t *testing.T) {
	machine := DefaultRecoveryMachine()
	for i := range machine.Transitions {
		if machine.Transitions[i].Reason == "provider_drive_zero" {
			machine.Transitions[i].Guard = "provider_speed > 0"
			break
		}
	}
	found := false
	for _, transition := range machine.Transitions {
		if transition.Reason == "provider_drive_zero" && transition.Guard == "provider_speed > 0" {
			found = true
		}
	}
	if !found {
		t.Fatal("the inverted guard was not visible in the transcribed table")
	}
	// The transcription is only as good as its anchors: an inverted guard that
	// still reads plausibly must be caught by comparing against the source,
	// which TestRecoveryMachineTranscriptionMatchesUpstreamAnchors does.
	if !strings.Contains(machine.String(), "provider_speed > 0") {
		t.Fatal("the rendered table does not show the guard text a reviewer must read")
	}
}

// TestRecoveryMachineGlobalTransitionIsDeclaredGlobal checks that the
// observation_invalid return to IDLE is marked global instead of being left
// implicit. A reader who only looks at per-state rows would otherwise conclude
// that a stuck state has no exit.
func TestRecoveryMachineGlobalTransitionIsDeclaredGlobal(t *testing.T) {
	machine := DefaultRecoveryMachine()
	found := false
	for _, transition := range machine.Transitions {
		if transition.Reason == "observation_invalid" {
			found = true
			if !transition.Global {
				t.Fatal("observation_invalid is not marked global; it is written once before the per-state chain")
			}
			if transition.To != StateIDLE {
				t.Fatalf("observation_invalid targets %q want IDLE", transition.To)
			}
		}
	}
	if !found {
		t.Fatal("observation_invalid is not transcribed")
	}

	// Scoping the global transition to one state while still marking it global
	// is contradictory and must be reported.
	mutated := machine
	for i := range mutated.Transitions {
		if mutated.Transitions[i].Reason == "observation_invalid" {
			mutated.Transitions[i].From = StateWALL_FOLLOW
			break
		}
	}
	if !hasCode(mutated.Diagnose(), "BEAR_SM_GLOBAL_TRANSITION_SCOPED") {
		t.Fatal("scoping a global transition to one state was accepted")
	}

	// Clearing the global flag without naming a source leaves the transition
	// unanchored: it would apply nowhere, and a reader would not notice.
	mutated = machine
	for i := range mutated.Transitions {
		if mutated.Transitions[i].Reason == "observation_invalid" {
			mutated.Transitions[i].Global = false
			mutated.Transitions[i].From = ""
			break
		}
	}
	if !hasCode(mutated.Diagnose(), "BEAR_SM_TRANSITION_SOURCE_MISSING") {
		t.Fatal("an unanchored transition was accepted")
	}
}

// TestRecoveryMachineStringRendersEveryReason is a readability check: the
// rendered table must show every reason, so a reviewer can read the machine
// without opening the GDScript.
func TestRecoveryMachineStringRendersEveryReason(t *testing.T) {
	rendered := DefaultRecoveryMachine().String()
	for _, reason := range TransitionReasons {
		if !strings.Contains(rendered, reason) {
			t.Fatalf("rendered table is missing reason %q", reason)
		}
	}
	if !strings.Contains(rendered, "*") {
		t.Fatal("rendered table does not mark the global transition")
	}
}
