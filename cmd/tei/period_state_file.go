package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/operations"
)

// periodStateFile reads the runtime state of periods from a JSON object such as
// {"sales.month": "closed"}. The file is an operator input: TEI does not store
// the state as accepted meaning, and a period absent from the file stays
// unguarded.
type periodStateFile struct {
	path string
}

func (s periodStateFile) PeriodStates(_ context.Context, _ string) (map[string]operations.PeriodState, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read period state file %s: %w", s.path, err)
	}
	var declared map[string]string
	if err := json.Unmarshal(raw, &declared); err != nil {
		return nil, fmt.Errorf("parse period state file %s: %w", s.path, err)
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
