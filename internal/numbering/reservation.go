package numbering

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrNumberingConflict reports that the counter changed between the read and the
// write. X-TEA wrote the counter back with an optimistic check on UPDCOUNTER and
// never inspected the result (D:Session.java L1995–2002), so a concurrent caller
// silently lost the issuance and two records could carry the same number. Here a
// conflict is reported, and the caller decides whether to retry.
var ErrNumberingConflict = errors.New("numbering conflict")

// CounterStore is where a counter lives and how a caller reserves it.
//
// The interface carries a version so a caller can detect a concurrent change. It
// does not itself provide a transaction: an implementation may join one.
type CounterStore interface {
	// Load returns the counter and its version.
	Load(ctx context.Context, ruleID string) (State, int, error)
	// Save writes the counter when the version still matches, and returns
	// ErrNumberingConflict otherwise.
	Save(ctx context.Context, ruleID string, expectedVersion int, state State) error
}

// MemoryCounterStore keeps counters in memory with versions, for tests and for a
// reference wiring. It is not durable.
type MemoryCounterStore struct {
	mu       sync.Mutex
	states   map[string]State
	versions map[string]int
}

// NewMemoryCounterStore creates an empty counter store.
func NewMemoryCounterStore() *MemoryCounterStore {
	return &MemoryCounterStore{states: map[string]State{}, versions: map[string]int{}}
}

// Load returns the counter and its version.
func (s *MemoryCounterStore) Load(_ context.Context, ruleID string) (State, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.states[ruleID], s.versions[ruleID], nil
}

// Save writes the counter when the version matches.
func (s *MemoryCounterStore) Save(_ context.Context, ruleID string, expectedVersion int, state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.versions[ruleID] != expectedVersion {
		return fmt.Errorf("%w: rule %s moved from version %d", ErrNumberingConflict, ruleID, expectedVersion)
	}
	s.states[ruleID] = state
	s.versions[ruleID]++
	return nil
}

// Reserve reads the counter, issues the next number, and writes the counter back
// inside the caller's transaction. It returns the number and a reservation the
// caller finishes with Commit or undoes with Release.
//
// X-TEA had no equivalent: a failure returned an empty string after the counter
// may already have moved, so a gap appeared with no record of why.
func (c Contract) Reserve(ctx context.Context, store CounterStore, ruleID string) (string, *Reservation, error) {
	if store == nil {
		return "", nil, fmt.Errorf("%w: no counter store is configured", ErrNumbering)
	}
	if err := c.validate(); err != nil {
		return "", nil, err
	}
	previous, version, err := store.Load(ctx, ruleID)
	if err != nil {
		return "", nil, fmt.Errorf("load counter for rule %s: %w", ruleID, err)
	}
	formatted, reserved, err := c.Next(previous)
	if err != nil {
		return "", nil, err
	}
	if err := store.Save(ctx, ruleID, version, reserved); err != nil {
		return "", nil, fmt.Errorf("reserve number for rule %s: %w", ruleID, err)
	}
	return formatted, &Reservation{
		store:           store,
		ruleID:          ruleID,
		previous:        previous,
		reserved:        reserved,
		reservedVersion: version + 1,
		number:          formatted,
	}, nil
}

// Reservation is a number issued inside a caller's transaction. The caller either
// commits it (the counter stays consumed) or releases it (the counter goes back),
// so a later failure does not consume a number silently.
type Reservation struct {
	store           CounterStore
	ruleID          string
	previous        State
	reserved        State
	reservedVersion int
	number          string
	finished        bool
	mu              sync.Mutex
}

// Number is the formatted number the reservation issued.
func (r *Reservation) Number() string { return r.number }

// Commit finishes the reservation and keeps the counter consumed.
func (r *Reservation) Commit() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finished {
		return fmt.Errorf("%w: the reservation is already finished", ErrNumbering)
	}
	r.finished = true
	return nil
}

// Release returns the counter to the state before the reservation, so a caller
// that cannot use the number does not leave a gap.
//
// The release is refused when the counter moved again after the reservation, so
// it never overwrites another caller's issuance.
func (r *Reservation) Release(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finished {
		return fmt.Errorf("%w: the reservation is already finished", ErrNumbering)
	}
	if err := r.store.Save(ctx, r.ruleID, r.reservedVersion, r.previous); err != nil {
		return fmt.Errorf("release numbering reservation: %w", err)
	}
	r.finished = true
	return nil
}
