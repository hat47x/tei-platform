package domain2

import (
	"strings"
	"testing"
)

// TestUIAsyncMachineHasExpectedShape checks the independent expected result:
// 7 transitions, 5 states, IDLE initial.
func TestUIAsyncMachineHasExpectedShape(t *testing.T) {
	machine := DefaultUIAsyncMachine()
	if machine.Initial != UIAsyncStateIDLE {
		t.Fatalf("initial state is %s, want IDLE", machine.Initial)
	}
	if len(machine.Transitions) != 7 {
		t.Fatalf("transition count = %d, want 7", len(machine.Transitions))
	}
	reachable := machine.ReachableStates()
	if len(reachable) != 5 {
		t.Fatalf("reachable state count = %d, want 5", len(reachable))
	}
	expectedStates := []UIAsyncState{UIAsyncStateIDLE, UIAsyncStatePENDING, UIAsyncStateRESOLVED, UIAsyncStateREJECTED, UIAsyncStateCANCELLED}
	sorted := sortedUIAsyncStates(expectedStates)
	for i, s := range sorted {
		if reachable[i] != UIAsyncState(s) {
			t.Fatalf("reachable[%d] = %s, want %s", i, reachable[i], s)
		}
	}
}

// TestUIAsyncMachineRejectsUnknownState is an independent expected result:
// an unknown state name must be refused.
func TestUIAsyncMachineRejectsUnknownState(t *testing.T) {
	_, ok := stateFromStringUIAsync("UNKNOWN")
	if ok {
		t.Fatal("unknown state was accepted")
	}
}

// TestUIAsyncMachineRejectsDuplicateTransitionID is an independent expected
// result: two transitions with the same ID must be detectable.
func TestUIAsyncMachineRejectsDuplicateTransitionID(t *testing.T) {
	machine := DefaultUIAsyncMachine()
	seen := map[string]bool{}
	for _, tr := range machine.Transitions {
		if seen[tr.ID] {
			t.Fatalf("duplicate transition ID %q", tr.ID)
		}
		seen[tr.ID] = true
	}
}

// TestUIAsyncMachineRejectsUnboundSourceOrTarget is an independent expected
// result: every transition must name both a source and a target state.
func TestUIAsyncMachineRejectsUnboundSourceOrTarget(t *testing.T) {
	for _, tr := range DefaultUIAsyncMachine().Transitions {
		if tr.From == "" {
			t.Fatalf("transition %q has no source state", tr.ID)
		}
		if tr.To == "" {
			t.Fatalf("transition %q has no target state", tr.ID)
		}
	}
}

// TestUIAsyncMachineGuardsNameObservations is the analogue of the bear guard
// test: a guard that says only "time passed" is ambiguous. UI async guards
// must name the observable cause.
func TestUIAsyncMachineGuardsNameObservations(t *testing.T) {
	observationTerms := []string{
		"user", "system", "response", "error", "cancel", "timeout",
		"result", "resource", "operation",
	}
	for _, tr := range DefaultUIAsyncMachine().Transitions {
		named := false
		lower := strings.ToLower(tr.Guard)
		for _, term := range observationTerms {
			if strings.Contains(lower, term) {
				named = true
				break
			}
		}
		if !named {
			t.Fatalf("guard %q (%s) names no observation term", tr.Guard, tr.Reason)
		}
	}
}

// TestUIAsyncMachineRendersEveryReason is a readability check: the rendered
// table must show every reason.
func TestUIAsyncMachineRendersEveryReason(t *testing.T) {
	rendered := DefaultUIAsyncMachine().String()
	for _, reason := range UIAsyncTransitionReasons {
		if !strings.Contains(rendered, string(reason)) {
			t.Fatalf("rendered table is missing reason %q", reason)
		}
	}
}

// TestCommSessionMachineHasExpectedShape checks the independent expected
// result: 8 transitions, 5 states, IDLE initial.
func TestCommSessionMachineHasExpectedShape(t *testing.T) {
	machine := DefaultCommSessionMachine()
	if machine.Initial != CommSessionStateIDLE {
		t.Fatalf("initial state is %s, want IDLE", machine.Initial)
	}
	if len(machine.Transitions) != 8 {
		t.Fatalf("transition count = %d, want 8", len(machine.Transitions))
	}
	reachable := machine.ReachableStates()
	if len(reachable) != 5 {
		t.Fatalf("reachable state count = %d, want 5", len(reachable))
	}
	expectedStates := []CommSessionState{CommSessionStateIDLE, CommSessionStateCONNECTING, CommSessionStateCONNECTED, CommSessionStateDISCONNECTING, CommSessionStateFAILED}
	sorted := sortedCommSessionStates(expectedStates)
	for i, s := range sorted {
		if reachable[i] != CommSessionState(s) {
			t.Fatalf("reachable[%d] = %s, want %s", i, reachable[i], s)
		}
	}
}

// TestCommSessionMachineRejectsUnknownState is an independent expected
// result: an unknown state name must be refused.
func TestCommSessionMachineRejectsUnknownState(t *testing.T) {
	_, ok := stateFromStringCommSession("UNKNOWN")
	if ok {
		t.Fatal("unknown state was accepted")
	}
}

// TestCommSessionMachineRejectsDuplicateTransitionID is an independent
// expected result: two transitions with the same ID must be detectable.
func TestCommSessionMachineRejectsDuplicateTransitionID(t *testing.T) {
	machine := DefaultCommSessionMachine()
	seen := map[string]bool{}
	for _, tr := range machine.Transitions {
		if seen[tr.ID] {
			t.Fatalf("duplicate transition ID %q", tr.ID)
		}
		seen[tr.ID] = true
	}
}

// TestCommSessionMachineRejectsUnboundSourceOrTarget is an independent
// expected result: every transition must name both a source and a target.
func TestCommSessionMachineRejectsUnboundSourceOrTarget(t *testing.T) {
	for _, tr := range DefaultCommSessionMachine().Transitions {
		if tr.From == "" {
			t.Fatalf("transition %q has no source state", tr.ID)
		}
		if tr.To == "" {
			t.Fatalf("transition %q has no target state", tr.ID)
		}
	}
}

// TestCommSessionMachineGuardsNameObservations is the analogue of the bear
// guard test and the UI async guard test.
func TestCommSessionMachineGuardsNameObservations(t *testing.T) {
	observationTerms := []string{
		"client", "server", "handshake", "transport", "close", "retry",
		"connection", "timeout", "backoff", "authentication",
	}
	for _, tr := range DefaultCommSessionMachine().Transitions {
		named := false
		lower := strings.ToLower(tr.Guard)
		for _, term := range observationTerms {
			if strings.Contains(lower, term) {
				named = true
				break
			}
		}
		if !named {
			t.Fatalf("guard %q (%s) names no observation term", tr.Guard, tr.Reason)
		}
	}
}

// TestCommSessionMachineRendersEveryReason is a readability check.
func TestCommSessionMachineRendersEveryReason(t *testing.T) {
	rendered := DefaultCommSessionMachine().String()
	for _, reason := range CommSessionTransitionReasons {
		if !strings.Contains(rendered, string(reason)) {
			t.Fatalf("rendered table is missing reason %q", reason)
		}
	}
}

// TestDomain2MachinesHaveDistinctReasonSpaces ensures the two domains do not
// accidentally share transition reason names.
func TestDomain2MachinesHaveDistinctReasonSpaces(t *testing.T) {
	uiReasons := map[string]bool{}
	for _, r := range UIAsyncTransitionReasons {
		uiReasons[string(r)] = true
	}
	commReasons := map[string]bool{}
	for _, r := range CommSessionTransitionReasons {
		commReasons[string(r)] = true
	}
	for r := range uiReasons {
		if commReasons[r] {
			t.Fatalf("transition reason %q appears in both UIAsync and CommSession", r)
		}
	}
}

func stateFromStringUIAsync(s string) (UIAsyncState, bool) {
	for _, state := range UIAsyncStates {
		if string(state) == s {
			return state, true
		}
	}
	return "", false
}

func stateFromStringCommSession(s string) (CommSessionState, bool) {
	for _, state := range CommSessionStates {
		if string(state) == s {
			return state, true
		}
	}
	return "", false
}
