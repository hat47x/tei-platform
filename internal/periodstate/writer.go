package periodstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/operations"
)

// Writer records the runtime state of periods in the same JSON object File reads,
// so a closing run leaves a file the next read sees.
//
// The write is atomic: the new content goes to a temporary file in the same
// directory and is renamed over the old one, so a crash during the write leaves
// the previous state rather than a half-written file.
type Writer struct {
	Path string
}

// Write replaces the recorded state of the given periods and keeps the rest.
//
// A state the operations package does not define is refused before anything is
// written, so a misspelt state does not reach the file.
func (w Writer) Write(ctx context.Context, states map[string]operations.PeriodState) error {
	if w.Path == "" {
		return fmt.Errorf("period state path is not set")
	}
	merged := map[string]operations.PeriodState{}
	if existing, err := (File{Path: w.Path}).PeriodStates(ctx, ""); err == nil {
		for period, state := range existing {
			merged[period] = state
		}
	} else if !os.IsNotExist(err) {
		// A file that exists but cannot be read is reported: overwriting it would
		// drop states the caller did not touch.
		if _, statErr := os.Stat(w.Path); statErr == nil {
			return err
		}
	}
	for period, state := range states {
		switch state {
		case operations.PeriodOpen, operations.PeriodClosing, operations.PeriodClosed:
		default:
			return fmt.Errorf("period %s has unknown state %q (want open, closing, or closed)", period, state)
		}
		merged[period] = state
	}
	encoded := map[string]string{}
	for period, state := range merged {
		encoded[period] = string(state)
	}
	raw, err := json.MarshalIndent(encoded, "", "  ")
	if err != nil {
		return fmt.Errorf("encode period state: %w", err)
	}
	directory := filepath.Dir(w.Path)
	temporary, err := os.CreateTemp(directory, ".period-state-*.json")
	if err != nil {
		return fmt.Errorf("create temporary period state: %w", err)
	}
	temporaryName := temporary.Name()
	if _, err := temporary.Write(append(raw, '\n')); err != nil {
		temporary.Close()
		os.Remove(temporaryName)
		return fmt.Errorf("write temporary period state: %w", err)
	}
	if err := temporary.Close(); err != nil {
		os.Remove(temporaryName)
		return fmt.Errorf("close temporary period state: %w", err)
	}
	if err := os.Rename(temporaryName, w.Path); err != nil {
		os.Remove(temporaryName)
		return fmt.Errorf("replace period state %s: %w", w.Path, err)
	}
	return nil
}
