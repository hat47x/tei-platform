package domain2

import (
	"sort"
	"strings"
)

// CommSessionState is a communication session lifecycle state.
type CommSessionState string

const (
	CommSessionStateIDLE          CommSessionState = "IDLE"
	CommSessionStateCONNECTING    CommSessionState = "CONNECTING"
	CommSessionStateCONNECTED     CommSessionState = "CONNECTED"
	CommSessionStateDISCONNECTING CommSessionState = "DISCONNECTING"
	CommSessionStateFAILED        CommSessionState = "FAILED"
)

// CommSessionStates lists every valid state.
var CommSessionStates = []CommSessionState{
	CommSessionStateIDLE,
	CommSessionStateCONNECTING,
	CommSessionStateCONNECTED,
	CommSessionStateDISCONNECTING,
	CommSessionStateFAILED,
}

// CommSessionTransitionReason names a transition cause.
type CommSessionTransitionReason string

// CommSessionTransitionReasons lists every reason this projection writes.
var CommSessionTransitionReasons = []CommSessionTransitionReason{
	"connect_initiated",
	"handshake_completed",
	"disconnect_requested",
	"session_ended",
	"connection_lost",
	"reconnect_attempted",
}

// CommSessionTransition describes one edge in the session lifecycle.
type CommSessionTransition struct {
	ID     string
	From   CommSessionState
	To     CommSessionState
	Reason CommSessionTransitionReason
	Guard  string
}

// CommSessionMachine is a read-only projection of the communication session lifecycle.
type CommSessionMachine struct {
	Initial     CommSessionState
	States      []CommSessionState
	Transitions []CommSessionTransition
}

// DefaultCommSessionMachine returns the canonical projection.
//
// Priority is source order within each state. A global observation-invalid
// transition is omitted because invalidity here is usually local (timeout,
// transport error) rather than a global escape.
func DefaultCommSessionMachine() CommSessionMachine {
	return CommSessionMachine{
		Initial: CommSessionStateIDLE,
		States:  append([]CommSessionState(nil), CommSessionStates...),
		Transitions: []CommSessionTransition{
			// IDLE -> CONNECTING
			{ID: "t-comm-1", From: CommSessionStateIDLE, To: CommSessionStateCONNECTING, Reason: "connect_initiated",
				Guard: "the client or server initiates a new connection"},

			// CONNECTING -> CONNECTED / FAILED
			{ID: "t-comm-2", From: CommSessionStateCONNECTING, To: CommSessionStateCONNECTED, Reason: "handshake_completed",
				Guard: "the handshake or authentication succeeds"},
			{ID: "t-comm-3", From: CommSessionStateCONNECTING, To: CommSessionStateFAILED, Reason: "connection_lost",
				Guard: "the handshake fails or the transport is unavailable"},

			// CONNECTED -> DISCONNECTING / FAILED
			{ID: "t-comm-4", From: CommSessionStateCONNECTED, To: CommSessionStateDISCONNECTING, Reason: "disconnect_requested",
				Guard: "a graceful close is requested"},
			{ID: "t-comm-5", From: CommSessionStateCONNECTED, To: CommSessionStateFAILED, Reason: "connection_lost",
				Guard: "the transport drops unexpectedly"},

			// DISCONNECTING -> terminal
			{ID: "t-comm-6", From: CommSessionStateDISCONNECTING, To: CommSessionStateIDLE, Reason: "session_ended",
				Guard: "the close handshake completes"},
			{ID: "t-comm-7", From: CommSessionStateDISCONNECTING, To: CommSessionStateFAILED, Reason: "connection_lost",
				Guard: "the transport drops during close"},

			// FAILED -> reconnect
			{ID: "t-comm-8", From: CommSessionStateFAILED, To: CommSessionStateCONNECTING, Reason: "reconnect_attempted",
				Guard: "the client retries after a backoff"},
		},
	}
}

// ReachableStates returns the states reachable from the initial state.
func (m CommSessionMachine) ReachableStates() []CommSessionState {
	seen := map[CommSessionState]bool{m.Initial: true}
	var queue []CommSessionState
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
	out := make([]CommSessionState, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// String renders a readable table of transitions.
func (m CommSessionMachine) String() string {
	var sb strings.Builder
	sb.WriteString("CommSessionMachine\n")
	sb.WriteString("Initial: " + string(m.Initial) + "\n")
	sb.WriteString("States:  " + strings.Join(sortedCommSessionStates(m.States), ", ") + "\n")
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

func sortedCommSessionStates(states []CommSessionState) []string {
	out := make([]string, len(states))
	for i, s := range states {
		out[i] = string(s)
	}
	sort.Strings(out)
	return out
}
