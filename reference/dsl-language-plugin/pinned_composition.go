package languagereference

import "fmt"

// PinnedCall is an operator call whose meaning is bound to an explicit,
// already-accepted contract identity. This reference does not issue pins.
type PinnedCall struct {
	Contract  OperatorContract
	InputType string
}

// ComposePinned resolves every contract by ID, version and fingerprint before
// checking type continuity. Catalog order never selects an implementation.
func ComposePinned(input string, calls []PinnedCall, catalog []OperatorContract) (string, error) {
	if input == "" || len(calls) == 0 {
		return "", ErrOperatorInvalid
	}
	current := input
	for i, call := range calls {
		if call.InputType != current {
			return "", fmt.Errorf("step %d: %w: expected %s, got %s", i, ErrOperatorInvalid, current, call.InputType)
		}
		resolved, err := ResolveOperator(call.Contract, catalog)
		if err != nil {
			return "", fmt.Errorf("step %d: %w", i, err)
		}
		if resolved.InputType != current {
			return "", fmt.Errorf("step %d: %w: contract input mismatch", i, ErrOperatorInvalid)
		}
		current = resolved.OutputType
	}
	return current, nil
}
