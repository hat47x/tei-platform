// Package periodstate provides sources for the runtime state of periods. The
// state is not accepted meaning: it changes as operations open and close a
// period, so it lives outside the semantic model.
package periodstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/operations"
)

// File reads the state of periods from a JSON object such as
// {"sales.month": "closed"}. A period absent from the file stays unguarded; an
// unknown state or an unreadable file is an error, not a refusal.
type File struct {
	Path string
}

// PeriodStates implements operations.PeriodStateSource.
func (f File) PeriodStates(context.Context, string) (map[string]operations.PeriodState, error) {
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, fmt.Errorf("read period state file %s: %w", f.Path, err)
	}
	var declared map[string]string
	if err := json.Unmarshal(raw, &declared); err != nil {
		return nil, fmt.Errorf("parse period state file %s: %w", f.Path, err)
	}
	states := make(map[string]operations.PeriodState, len(declared))
	for period, state := range declared {
		switch operations.PeriodState(state) {
		case operations.PeriodOpen, operations.PeriodClosing, operations.PeriodClosed:
			states[period] = operations.PeriodState(state)
		default:
			return nil, fmt.Errorf("period %s has unknown state %q (want open, closing, or closed)", period, state)
		}
	}
	return states, nil
}
