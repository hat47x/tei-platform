package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const lifecycleFileSchema = "tei.command-lifecycle/v0"

type lifecycleFile struct {
	Schema   string    `json:"schema"`
	Attempts []Attempt `json:"attempts"`
}

// DurableLifecycle is a file-backed Command stage store. It applies the same
// fail-closed rules as Lifecycle and persists the latest attempt per command
// after every mutation, so a restarted process still sees an unknown attempt
// and refuses a same-attempt replay (#2564 RCAS-02).
//
// The file is written to a temporary file in the same directory and renamed,
// so a crash during a write does not leave a partially written state file. A
// corrupt or unknown-schema file is rejected instead of being treated as empty
// state.
type DurableLifecycle struct {
	path  string
	mu    sync.Mutex
	inner *Lifecycle
}

// OpenDurableLifecycle loads the stage store from path. A missing file is an
// empty store; an unreadable, corrupt or unknown-schema file is an error.
func OpenDurableLifecycle(path string) (*DurableLifecycle, error) {
	if path == "" {
		return nil, fmt.Errorf("command lifecycle: state path is required")
	}
	store := &DurableLifecycle{path: path, inner: NewLifecycle()}
	encoded, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("command lifecycle: read state file: %w", err)
	}
	var file lifecycleFile
	if err := json.Unmarshal(encoded, &file); err != nil {
		return nil, fmt.Errorf("command lifecycle: state file is not valid JSON: %w", err)
	}
	if file.Schema != lifecycleFileSchema {
		return nil, fmt.Errorf("command lifecycle: unexpected state schema %q", file.Schema)
	}
	for _, attempt := range file.Attempts {
		if attempt.CommandID == "" {
			return nil, fmt.Errorf("command lifecycle: state file has an attempt without a command id")
		}
		store.inner.attempts[attempt.CommandID] = attempt
	}
	return store, nil
}

// Path is the state file location.
func (d *DurableLifecycle) Path() string { return d.path }

// Attempt returns the persisted attempt for a command.
func (d *DurableLifecycle) Attempt(commandID string) (Attempt, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.inner.Attempt(commandID)
}

// MayReplay reports whether the command may be dispatched without a takeover.
func (d *DurableLifecycle) MayReplay(commandID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.inner.MayReplay(commandID)
}

// Dispatch records a new attempt and persists it.
func (d *DurableLifecycle) Dispatch(commandID, owner string, generation int) (Attempt, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	attempt, err := d.inner.Dispatch(commandID, owner, generation)
	if err != nil {
		return attempt, err
	}
	return attempt, d.persistLocked()
}

// Takeover starts a new attempt for an unknown command and persists it.
func (d *DurableLifecycle) Takeover(commandID, owner string, generation int) (Attempt, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	attempt, err := d.inner.Takeover(commandID, owner, generation)
	if err != nil {
		return attempt, err
	}
	return attempt, d.persistLocked()
}

// Observe records the outcome of the current attempt and persists it.
func (d *DurableLifecycle) Observe(commandID, owner string, generation int, outcome string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.inner.Observe(commandID, owner, generation, outcome); err != nil {
		return err
	}
	return d.persistLocked()
}

// MarkUnknown records that the attempt has no reliable outcome and persists it.
func (d *DurableLifecycle) MarkUnknown(commandID, owner string, generation int, reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.inner.MarkUnknown(commandID, owner, generation, reason); err != nil {
		return err
	}
	return d.persistLocked()
}

func (d *DurableLifecycle) persistLocked() error {
	attempts := d.inner.Attempts()
	sort.Slice(attempts, func(i, j int) bool { return attempts[i].CommandID < attempts[j].CommandID })
	encoded, err := json.MarshalIndent(lifecycleFile{Schema: lifecycleFileSchema, Attempts: attempts}, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	directory := filepath.Dir(d.path)
	temporary, err := os.CreateTemp(directory, ".tei-lifecycle-*")
	if err != nil {
		return fmt.Errorf("command lifecycle: create temporary state file: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() { _ = os.Remove(temporaryPath) }
	if _, err := temporary.Write(encoded); err != nil {
		_ = temporary.Close()
		cleanup()
		return fmt.Errorf("command lifecycle: write state file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		cleanup()
		return fmt.Errorf("command lifecycle: sync state file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		cleanup()
		return fmt.Errorf("command lifecycle: close state file: %w", err)
	}
	if err := os.Rename(temporaryPath, d.path); err != nil {
		cleanup()
		return fmt.Errorf("command lifecycle: replace state file: %w", err)
	}
	return nil
}
