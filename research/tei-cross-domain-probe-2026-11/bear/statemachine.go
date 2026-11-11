package bearprobe

import (
	"fmt"
	"sort"
	"strings"
)

// RecoveryMachine is a read-only transcription of the traversal-recovery state
// machine in controllers/bear/traversal_recovery.gd. It exists so that the
// structural properties #4193 asks about (reachability, dead ends, guard
// priority, unknown endpoints) can be checked without running Godot and
// without adding state-machine support to the TEI meaning DSL.
//
// The transcription is anchored to the upstream file by
// TestRecoveryMachineTranscriptionMatchesUpstreamAnchors. The machine is NOT a
// second implementation: it does not step, does not decide, and grants no
// authority over the runtime controller.

// RecoveryStateName is a state of the traversal-recovery machine. The values
// match RecoveryState enum entries and BearTraversalRecovery.state_name().
type RecoveryStateName string

const (
	StateIDLE              RecoveryStateName = "IDLE"
	StateWALL_FOLLOW       RecoveryStateName = "WALL_FOLLOW"
	StateENTER_TRAVERSABLE RecoveryStateName = "ENTER_TRAVERSABLE"
	StateCLEARING          RecoveryStateName = "CLEARING"

	// StateUNKNOWN is the label motion_feedback.gd writes when the recorded
	// state name is not one of the four enum entries. It is a fallback label,
	// not a state of the machine, and must never appear as a transition
	// endpoint.
	StateUNKNOWN RecoveryStateName = "unknown"
)

// MachineStates are the four states of the machine. StateUNKNOWN is
// deliberately excluded: it is a recorded label, not a state.
var MachineStates = []RecoveryStateName{
	StateIDLE, StateWALL_FOLLOW, StateENTER_TRAVERSABLE, StateCLEARING,
}

// RecoveryTransition is one transition as written in the if/elif chain of
// BearTraversalRecovery.step(). Priority is the position of the branch inside
// one source state: the runtime resolves competing guards by source order, so
// the order is part of the meaning and is transcribed rather than inferred.
type RecoveryTransition struct {
	ID       string            `json:"id"`
	From     RecoveryStateName `json:"from"`
	To       RecoveryStateName `json:"to"`
	Reason   string            `json:"reason"`
	Guard    string            `json:"guard"`
	Priority int               `json:"priority"`

	// Global marks a transition that applies from every non-initial state.
	// observation_invalid is such a transition: it is written once, before the
	// per-state chain, and returns the machine to IDLE.
	Global bool `json:"global,omitempty"`
}

// RecoveryMachine is the transcribed machine.
type RecoveryMachine struct {
	Initial     RecoveryStateName    `json:"initial"`
	States      []RecoveryStateName  `json:"states"`
	Transitions []RecoveryTransition `json:"transitions"`
}

// DefaultRecoveryMachine returns the machine transcribed from
// controllers/bear/traversal_recovery.gd at the checked upstream commit. Every
// Guard string quotes the condition as the source writes it; the numeric
// constants are named, not inlined, so a retuned constant does not look like a
// changed guard.
func DefaultRecoveryMachine() RecoveryMachine {
	return RecoveryMachine{
		Initial: StateIDLE,
		States:  append([]RecoveryStateName(nil), MachineStates...),
		Transitions: []RecoveryTransition{
			// IDLE
			{ID: "t-idle-1", From: StateIDLE, To: StateENTER_TRAVERSABLE, Reason: "direct_traversable",
				Guard: "provider_speed >= meaningful_drive_mps and a direct affordance exists", Priority: 1},
			{ID: "t-idle-2", From: StateIDLE, To: StateWALL_FOLLOW, Reason: "blocked_drive_stall",
				// The stall is a conjunction inside _update_stall_age(): a
				// meaningful drive speed, a valid lagged sample below
				// stall_progress_mps, and forward blocking. "Time passed" alone
				// is not the guard; writing only stall_age_s >= stall_activation_s
				// would let a reader treat any pause as a stall.
				Guard: "provider_speed >= meaningful_drive_mps and a valid lagged progress sample < stall_progress_mps and forward blocking accumulated for stall_activation_s", Priority: 2},

			// WALL_FOLLOW
			{ID: "t-wall-1", From: StateWALL_FOLLOW, To: StateCLEARING, Reason: "provider_drive_zero",
				Guard: "provider_speed <= 0", Priority: 1},
			{ID: "t-wall-2", From: StateWALL_FOLLOW, To: StateENTER_TRAVERSABLE, Reason: "recovery_traversable",
				Guard: "a recovery affordance exists", Priority: 2},
			{ID: "t-wall-3", From: StateWALL_FOLLOW, To: StateCLEARING, Reason: "wall_clear_progress",
				Guard: "wall-follow clear progress observed and clear_age_s >= clear_hysteresis_s", Priority: 3},

			// ENTER_TRAVERSABLE
			{ID: "t-trav-1", From: StateENTER_TRAVERSABLE, To: StateCLEARING, Reason: "provider_drive_zero",
				Guard: "provider_speed <= 0", Priority: 1},
			{ID: "t-trav-2", From: StateENTER_TRAVERSABLE, To: StateENTER_TRAVERSABLE, Reason: "traversable_retargeted",
				Guard: "a replacement affordance exists and the selected affordance changed", Priority: 2},
			{ID: "t-trav-3", From: StateENTER_TRAVERSABLE, To: StateCLEARING, Reason: "traversable_cleared",
				Guard: "healthy clear progress, the selected affordance is no longer traversable, and clear_age_s >= clear_hysteresis_s", Priority: 3},

			// CLEARING
			{ID: "t-clear-1", From: StateCLEARING, To: StateENTER_TRAVERSABLE, Reason: "traversable_returned",
				Guard: "provider_speed > 0 and a clearing affordance exists", Priority: 1},
			{ID: "t-clear-2", From: StateCLEARING, To: StateWALL_FOLLOW, Reason: "blocking_returned",
				Guard: "provider_speed > 0 and forward or followed-boundary blocking", Priority: 2},
			{ID: "t-clear-3", From: StateCLEARING, To: StateIDLE, Reason: "progress_restored",
				Guard: "healthy clear progress and clear_age_s >= clear_hysteresis_s", Priority: 3},

			// Written before the per-state chain; returns to IDLE from any
			// non-initial state. Transcribed as Global so that a reader does not
			// have to infer it from the absence of a From value.
			{ID: "t-global-1", From: "", To: StateIDLE, Reason: "observation_invalid",
				Guard: "the observation is invalid", Priority: 0, Global: true},
		},
	}
}

// TransitionReasons lists every transition reason the upstream writes. The
// anchor test checks each string against the cited file.
var TransitionReasons = []string{
	"direct_traversable", "blocked_drive_stall",
	"provider_drive_zero", "recovery_traversable", "wall_clear_progress",
	"traversable_retargeted", "traversable_cleared",
	"traversable_returned", "blocking_returned", "progress_restored",
	"observation_invalid",
}

// Diagnose checks the transcribed machine for the structural properties #4193
// §5 lists: unknown endpoints, duplicate identity, guard priority, unreachable
// states and dead ends. It reports what a reader of the table would need to be
// told; it does not simulate the machine.
func (m RecoveryMachine) Diagnose() []Diagnostic {
	var out []Diagnostic
	add := func(code, format string, args ...any) {
		out = append(out, Diagnostic{Code: code, Severity: SeverityError, Message: fmt.Sprintf(format, args...)})
	}

	known := map[RecoveryStateName]bool{}
	for _, state := range m.States {
		if state == StateUNKNOWN {
			add("BEAR_SM_UNKNOWN_IS_NOT_A_STATE", "%q is a recorded fallback label, not a machine state", state)
			continue
		}
		if state == "" {
			add("BEAR_SM_STATE_NAME_EMPTY", "a state has an empty name")
			continue
		}
		if known[state] {
			add("BEAR_SM_STATE_DUPLICATE", "state %q is declared more than once", state)
			continue
		}
		known[state] = true
	}
	if m.Initial == "" {
		add("BEAR_SM_INITIAL_MISSING", "no initial state is declared")
	} else if !known[m.Initial] {
		add("BEAR_SM_INITIAL_UNKNOWN", "initial state %q is not a declared state", m.Initial)
	}

	seenIDs := map[string]bool{}
	seenFromReason := map[string]bool{}
	priority := map[RecoveryStateName]map[int]string{}
	outgoing := map[RecoveryStateName]int{}
	for _, transition := range m.Transitions {
		at := transition.ID
		if transition.ID == "" {
			at = transition.Reason
			add("BEAR_SM_TRANSITION_ID_MISSING", "transition %q has no id", transition.Reason)
		}
		if seenIDs[transition.ID] {
			add("BEAR_SM_TRANSITION_ID_DUPLICATE", "transition id %q is used more than once", transition.ID)
		}
		seenIDs[transition.ID] = true

		if transition.Global {
			if transition.From != "" {
				add("BEAR_SM_GLOBAL_TRANSITION_SCOPED", "global transition %q also names a source state %q; a global transition must not be scoped", transition.Reason, transition.From)
			}
		} else {
			if transition.From == "" {
				add("BEAR_SM_TRANSITION_SOURCE_MISSING", "transition %q has no source state and is not marked global", transition.Reason)
			} else if !known[transition.From] {
				add("BEAR_SM_TRANSITION_SOURCE_UNKNOWN", "transition %q names unknown source state %q", transition.Reason, transition.From)
			}
		}
		if !known[transition.To] {
			add("BEAR_SM_TRANSITION_TARGET_UNKNOWN", "transition %q names unknown target state %q", transition.Reason, transition.To)
		}
		if transition.Guard == "" {
			add("BEAR_SM_TRANSITION_GUARD_MISSING", "transition %q has no guard; a transition without a condition cannot be reviewed", transition.Reason)
		}
		if transition.Reason == "" {
			add("BEAR_SM_TRANSITION_REASON_MISSING", "transition %s has no reason", at)
		}

		if !transition.Global {
			key := string(transition.From) + "/" + transition.Reason
			if seenFromReason[key] {
				add("BEAR_SM_TRANSITION_REASON_DUPLICATE", "state %q declares reason %q more than once", transition.From, transition.Reason)
			}
			seenFromReason[key] = true

			if transition.Priority <= 0 {
				add("BEAR_SM_TRANSITION_PRIORITY_MISSING", "transition %q from %q has no priority; source order is part of the meaning", transition.Reason, transition.From)
			}
			if priority[transition.From] == nil {
				priority[transition.From] = map[int]string{}
			}
			if other, dup := priority[transition.From][transition.Priority]; dup {
				add("BEAR_SM_TRANSITION_PRIORITY_CONFLICT", "state %q has two transitions at priority %d: %q and %q; the runtime would resolve them by source order alone", transition.From, transition.Priority, other, transition.Reason)
			}
			priority[transition.From][transition.Priority] = transition.Reason
			outgoing[transition.From]++
		}
	}

	// Reachability from the initial state. A global transition counts as an
	// outgoing edge from every state it applies to.
	reachable := map[RecoveryStateName]bool{m.Initial: true}
	for changed := true; changed; {
		changed = false
		for _, transition := range m.Transitions {
			if !known[transition.To] {
				continue
			}
			if transition.Global {
				for state := range known {
					if state != m.Initial && reachable[state] && !reachable[transition.To] {
						reachable[transition.To] = true
						changed = true
					}
				}
				continue
			}
			if reachable[transition.From] && !reachable[transition.To] {
				reachable[transition.To] = true
				changed = true
			}
		}
	}
	for _, state := range m.States {
		if state == StateUNKNOWN || !known[state] {
			continue
		}
		if !reachable[state] {
			add("BEAR_SM_STATE_UNREACHABLE", "state %q is not reachable from %q", state, m.Initial)
		}
		if outgoing[state] == 0 && !hasGlobalTransition(m) {
			add("BEAR_SM_STATE_DEAD_END", "state %q has no outgoing transition and no global transition applies", state)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func hasGlobalTransition(m RecoveryMachine) bool {
	for _, transition := range m.Transitions {
		if transition.Global {
			return true
		}
	}
	return false
}

// ReachableStates returns the states reachable from the initial state. It is
// exposed so a test can assert a specific set instead of only the absence of
// diagnostics.
func (m RecoveryMachine) ReachableStates() []RecoveryStateName {
	out := map[RecoveryStateName]bool{m.Initial: true}
	for changed := true; changed; {
		changed = false
		for _, transition := range m.Transitions {
			if transition.Global {
				for state := range stateSet(m.States) {
					if state != m.Initial && out[state] && !out[transition.To] {
						out[transition.To] = true
						changed = true
					}
				}
				continue
			}
			if out[transition.From] && !out[transition.To] {
				out[transition.To] = true
				changed = true
			}
		}
	}
	names := make([]RecoveryStateName, 0, len(out))
	for state := range out {
		names = append(names, state)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// PrioritiesFor returns the transitions of one source state ordered by the
// transcribed priority. A reader can compare this order with the if/elif chain
// in the source.
func (m RecoveryMachine) PrioritiesFor(from RecoveryStateName) []RecoveryTransition {
	var out []RecoveryTransition
	for _, transition := range m.Transitions {
		if !transition.Global && transition.From == from {
			out = append(out, transition)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Priority < out[j].Priority })
	return out
}

func stateSet(states []RecoveryStateName) map[RecoveryStateName]bool {
	out := map[RecoveryStateName]bool{}
	for _, state := range states {
		out[state] = true
	}
	return out
}

// SelfTransitions lists transitions whose source and target are the same
// state. ENTER_TRAVERSABLE retargets in place; a reader of a flat transition
// table can mistake that for a missing transition.
func (m RecoveryMachine) SelfTransitions() []RecoveryTransition {
	var out []RecoveryTransition
	for _, transition := range m.Transitions {
		if transition.From == transition.To {
			out = append(out, transition)
		}
	}
	return out
}

// Reasons returns every transition reason, sorted, for stable reporting.
func (m RecoveryMachine) Reasons() []string {
	out := append([]string(nil), TransitionReasons...)
	sort.Strings(out)
	return out
}

// String renders the machine as a compact table for a review comment.
func (m RecoveryMachine) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "initial=%s states=%d transitions=%d\n", m.Initial, len(m.States), len(m.Transitions))
	for _, transition := range m.Transitions {
		from := string(transition.From)
		if transition.Global {
			from = "*"
		}
		fmt.Fprintf(&b, "  %-22s %s -> %-18s p=%d %s\n", transition.Reason, from, transition.To, transition.Priority, transition.Guard)
	}
	return b.String()
}
