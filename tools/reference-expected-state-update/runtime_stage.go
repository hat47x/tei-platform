package expectedstateupdate

import "fmt"

// Stage is the runtime stage of one CAS command (#2564 -> #1129). A command is
// dispatched, then either observed or unknown. An unknown outcome must never be
// replayed by the same owner and generation: a new owner must take over with a
// higher generation, and the takeover alone does not claim success.
type Stage string

const (
	// StageDispatched means the request was sent but no outcome was observed.
	StageDispatched Stage = "dispatched"
	// StageObserved means a business outcome was observed for this attempt.
	StageObserved Stage = "observed"
	// StageUnknown means the dispatch completed without a reliable outcome.
	StageUnknown Stage = "unknown"
)

// Attempt is one dispatch of a command.
type Attempt struct {
	CommandID    string `json:"commandId"`
	Owner        string `json:"owner"`
	Generation   int    `json:"generation"`
	Stage        Stage  `json:"stage"`
	Outcome      string `json:"outcome,omitempty"`
	Reason       string `json:"reason,omitempty"`
	TakeoverFrom int    `json:"takeoverFrom,omitempty"`
	TakeoverBy   string `json:"takeoverBy,omitempty"`
}

// RuntimeStage tracks the latest attempt per command. It is a verification
// reference for unknown/takeover handling, not a product runtime store.
type RuntimeStage struct {
	attempts map[string]Attempt
}

// NewRuntimeStage returns an empty stage.
func NewRuntimeStage() *RuntimeStage {
	return &RuntimeStage{attempts: map[string]Attempt{}}
}

// Attempt returns the latest attempt for a command.
func (s *RuntimeStage) Attempt(commandID string) (Attempt, bool) {
	attempt, ok := s.attempts[commandID]
	return attempt, ok
}

// Dispatch records a new attempt. Re-dispatching after an unknown outcome is
// refused unless the caller first takes over with a higher generation.
func (s *RuntimeStage) Dispatch(commandID, owner string, generation int) (Attempt, error) {
	if commandID == "" || owner == "" {
		return Attempt{}, fmt.Errorf("runtime stage: command and owner are required")
	}
	if generation < 1 {
		return Attempt{}, fmt.Errorf("runtime stage: generation must be positive")
	}
	if existing, ok := s.attempts[commandID]; ok {
		switch existing.Stage {
		case StageObserved:
			return Attempt{}, fmt.Errorf("runtime stage: command %q is already observed; do not re-dispatch", commandID)
		case StageDispatched:
			return Attempt{}, fmt.Errorf("runtime stage: command %q is already in flight (%s/%d)", commandID, existing.Owner, existing.Generation)
		case StageUnknown:
			if existing.Owner == owner || generation <= existing.Generation {
				return Attempt{}, fmt.Errorf("runtime stage: command %q has an unknown outcome; replay requires a takeover with a higher generation", commandID)
			}
		}
	}
	attempt := Attempt{CommandID: commandID, Owner: owner, Generation: generation, Stage: StageDispatched}
	if existing, ok := s.attempts[commandID]; ok && existing.Stage == StageUnknown {
		attempt.TakeoverFrom = existing.Generation
	}
	s.attempts[commandID] = attempt
	return attempt, nil
}

// Takeover starts a new attempt for an unknown command under a new owner and a
// higher generation. It records the takeover and leaves the command dispatched;
// success is only claimed after Observe.
func (s *RuntimeStage) Takeover(commandID, owner string, generation int) (Attempt, error) {
	existing, ok := s.attempts[commandID]
	if !ok {
		return Attempt{}, fmt.Errorf("runtime stage: command %q was not dispatched", commandID)
	}
	if existing.Stage != StageUnknown {
		return Attempt{}, fmt.Errorf("runtime stage: takeover requires an unknown outcome, command %q is %s", commandID, existing.Stage)
	}
	if generation <= existing.Generation {
		return Attempt{}, fmt.Errorf("runtime stage: takeover generation %d must exceed the current generation %d", generation, existing.Generation)
	}
	if owner == "" {
		return Attempt{}, fmt.Errorf("runtime stage: takeover owner is required")
	}
	attempt := Attempt{
		CommandID: commandID, Owner: owner, Generation: generation, Stage: StageDispatched,
		TakeoverFrom: existing.Generation,
	}
	s.attempts[commandID] = attempt
	return attempt, nil
}

// Observe records the business outcome of the current attempt. The owner and
// generation must match the dispatched attempt.
func (s *RuntimeStage) Observe(commandID, owner string, generation int, outcome string) error {
	existing, ok := s.attempts[commandID]
	if !ok {
		return fmt.Errorf("runtime stage: command %q was not dispatched", commandID)
	}
	if existing.Stage != StageDispatched {
		return fmt.Errorf("runtime stage: command %q is %s, not dispatched", commandID, existing.Stage)
	}
	if existing.Owner != owner || existing.Generation != generation {
		return fmt.Errorf("runtime stage: observation owner/generation does not match the dispatched attempt")
	}
	if outcome == "" {
		return fmt.Errorf("runtime stage: outcome is required")
	}
	existing.Stage = StageObserved
	existing.Outcome = outcome
	s.attempts[commandID] = existing
	return nil
}

// MarkUnknown records that the dispatch completed without a reliable outcome.
func (s *RuntimeStage) MarkUnknown(commandID, owner string, generation int, reason string) error {
	existing, ok := s.attempts[commandID]
	if !ok {
		return fmt.Errorf("runtime stage: command %q was not dispatched", commandID)
	}
	if existing.Stage != StageDispatched {
		return fmt.Errorf("runtime stage: command %q is %s, not dispatched", commandID, existing.Stage)
	}
	if existing.Owner != owner || existing.Generation != generation {
		return fmt.Errorf("runtime stage: unknown marker owner/generation does not match the dispatched attempt")
	}
	existing.Stage = StageUnknown
	existing.Reason = reason
	s.attempts[commandID] = existing
	return nil
}

// MayReplay reports whether the same owner/generation may re-dispatch. Unknown
// and observed commands must not be replayed.
func (s *RuntimeStage) MayReplay(commandID string) bool {
	existing, ok := s.attempts[commandID]
	if !ok {
		return true
	}
	return existing.Stage != StageUnknown && existing.Stage != StageObserved
}
