package domain2

import (
	"sort"
	"strings"
)

// uiAsyncNote records the provenance of this synthetic model.
//
// Domain 2 is UI async lifecycle. Unlike Domain 1 (VS Bear recovery), this
// domain has no single upstream file to transcribe. The states and transitions
// below are a **synthetic read-only projection** built from common UI async
// patterns (promises, futures, async/await cancellation, timeout, error
// handling, cleanup). They are not copied from a specific upstream repository.
//
// The point of keeping Domain 2 synthetic is to answer #4193 acceptance
// criterion 1 ("at least two heterogeneous domains") without depending on a
// second external repository. Real UI frameworks (React, SwiftUI, Jetpack
// Compose, Flutter) have equivalent lifecycles; this model captures the
// intersection.
//
// SM-01 lessons carried forward:
//   - guards name observations/affordances, not only timers
//   - transitions that keep state internally are not "no-op"
//   - global transitions must be explicit
//   - fallback labels must be named
const uiAsyncNote = "synthetic UI async lifecycle projection; not transcribed from a single upstream"

// UIAsyncState is an async UI operation lifecycle state.
type UIAsyncState string

const (
	UIAsyncStateIDLE      UIAsyncState = "IDLE"
	UIAsyncStatePENDING   UIAsyncState = "PENDING"
	UIAsyncStateRESOLVED  UIAsyncState = "RESOLVED"
	UIAsyncStateREJECTED  UIAsyncState = "REJECTED"
	UIAsyncStateCANCELLED UIAsyncState = "CANCELLED"
)

// UIAsyncStates lists every valid state.
var UIAsyncStates = []UIAsyncState{
	UIAsyncStateIDLE,
	UIAsyncStatePENDING,
	UIAsyncStateRESOLVED,
	UIAsyncStateREJECTED,
	UIAsyncStateCANCELLED,
}

// UIAsyncTransitionReason names a transition cause.
type UIAsyncTransitionReason string

// UIAsyncTransitionReasons lists every reason this projection writes.
// Real systems may use different wording, but the causes are equivalent.
var UIAsyncTransitionReasons = []UIAsyncTransitionReason{
	"async_request_started", // user action or system-initiated async work begins
	"response_received",     // successful response arrives before cancel/timeout
	"error_received",        // error response or runtime failure
	"request_cancelled",     // user abort or higher-level timeout
	"result_processed",      // success path consumes the result and clears pending UI
	"error_handled",         // error path surfaces the failure and clears pending UI
	"cleanup_done",          // temporary resources are released after cancel
}

// UIAsyncTransition describes one edge in the UI async lifecycle.
type UIAsyncTransition struct {
	ID     string
	From   UIAsyncState
	To     UIAsyncState
	Reason UIAsyncTransitionReason
	Guard  string
}

// UIAsyncMachine is a read-only projection of the UI async lifecycle.
type UIAsyncMachine struct {
	Initial     UIAsyncState
	States      []UIAsyncState
	Transitions []UIAsyncTransition
}

// DefaultUIAsyncMachine returns the canonical projection.
//
// Priority is source order within each state (if/elif chain equivalent).
// A global "observation_invalid" style transition is omitted on purpose: UI
// async invalidity is usually a local timeout/error, not a global escape.
func DefaultUIAsyncMachine() UIAsyncMachine {
	return UIAsyncMachine{
		Initial: UIAsyncStateIDLE,
		States:  append([]UIAsyncState(nil), UIAsyncStates...),
		Transitions: []UIAsyncTransition{
			// IDLE -> PENDING
			{ID: "t-ui-1", From: UIAsyncStateIDLE, To: UIAsyncStatePENDING, Reason: "async_request_started",
				Guard: "an async operation is started by the user or system"},

			// PENDING -> terminal states
			{ID: "t-ui-2", From: UIAsyncStatePENDING, To: UIAsyncStateRESOLVED, Reason: "response_received",
				Guard: "a successful response is received before cancellation or timeout"},
			{ID: "t-ui-3", From: UIAsyncStatePENDING, To: UIAsyncStateREJECTED, Reason: "error_received",
				Guard: "an error response is received or the operation fails"},
			{ID: "t-ui-4", From: UIAsyncStatePENDING, To: UIAsyncStateCANCELLED, Reason: "request_cancelled",
				Guard: "the user cancels the operation or a higher-level timeout fires"},

			// terminal -> IDLE
			{ID: "t-ui-5", From: UIAsyncStateRESOLVED, To: UIAsyncStateIDLE, Reason: "result_processed",
				Guard: "the UI consumes the successful result"},
			{ID: "t-ui-6", From: UIAsyncStateREJECTED, To: UIAsyncStateIDLE, Reason: "error_handled",
				Guard: "the UI surfaces the error and clears the pending state"},
			{ID: "t-ui-7", From: UIAsyncStateCANCELLED, To: UIAsyncStateIDLE, Reason: "cleanup_done",
				Guard: "any temporary resources are released"},
		},
	}
}

// ReachableStates returns the states reachable from the initial state.
func (m UIAsyncMachine) ReachableStates() []UIAsyncState {
	seen := map[UIAsyncState]bool{m.Initial: true}
	var queue []UIAsyncState
	queue = append(queue, m.Initial)
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		for _, t := range m.Transitions {
			if t.From == current && !seen[t.To] {
				seen[t.To] = true
				queue = append(queue, t.To)
			}
		}
	}
	out := make([]UIAsyncState, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// String renders a readable table of transitions.
func (m UIAsyncMachine) String() string {
	var sb strings.Builder
	sb.WriteString("UIAsyncMachine\n")
	sb.WriteString("Initial: " + string(m.Initial) + "\n")
	sb.WriteString("States:  " + strings.Join(sortedUIAsyncStates(m.States), ", ") + "\n")
	sb.WriteString("Transitions:\n")
	for _, t := range m.Transitions {
		marker := ""
		if t.From == "" {
			marker = " *"
		}
		sb.WriteString("  " + t.ID + ": " + string(t.From) + " -> " + string(t.To) +
			" (" + string(t.Reason) + ")" + marker + " guard=" + t.Guard + "\n")
	}
	return sb.String()
}

func sortedUIAsyncStates(states []UIAsyncState) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = string(s)
	}
	sort.Strings(out)
	return out
}
