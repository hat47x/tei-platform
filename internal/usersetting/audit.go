package usersetting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tei/internal/core"
)

// AuditEntry records one accepted setting change: which setting, by whom, what it
// became, and when.
//
// X-TEA's script API wrote values straight into the session's variable map
// (setVariant, setUserValueOf: D:Session.java L2112, D:XFScriptable.java L56–60), so
// nothing recorded who changed a setting or what it was before. A question about a
// screen that behaved differently last week had no answer. The entry keeps the
// actor and the previous value so that question can be asked of the record.
type AuditEntry struct {
	// At is the time the change was accepted.
	At time.Time `json:"at"`
	// Data is the setting Data the change belongs to.
	Data core.CanonicalID `json:"data"`
	// Key and User identify the setting, as Change does.
	Key  string `json:"key"`
	User string `json:"user,omitempty"`
	// Before is the value the setting had, and Had reports whether it existed. A
	// first write and a change from empty are different facts.
	Before string `json:"before,omitempty"`
	Had    bool   `json:"had,omitempty"`
	// After is the value the setting now has.
	After string `json:"after"`
	// Actor is who asked for the change. It is required: an unattributed change
	// is not an audit record.
	Actor string `json:"actor"`
}

// AuditedStore is a Store that also keeps the change log.
//
// The log is append-only in this type: Put still replaces the list, but the history
// of accepted changes grows. A caller that needs to prove a change happened reads
// Entries, not the current list.
type AuditedStore struct {
	mu      sync.Mutex
	inner   Store
	entries []AuditEntry
	// clock returns the time of a change. It is a field so a test can fix it.
	clock func() time.Time
}

// NewAuditedStore wraps a store and keeps the change log beside it.
func NewAuditedStore(inner Store) *AuditedStore {
	if inner == nil {
		inner = NewMemoryStore()
	}
	return &AuditedStore{inner: inner, clock: time.Now}
}

// Put records the list and nothing else. It exists so AuditedStore satisfies Store;
// a caller that wants an audit entry uses ApplyAudited, which carries the actor.
func (s *AuditedStore) Put(ctx context.Context, dataID string, settings []Setting) error {
	return s.inner.Put(ctx, dataID, settings)
}

// Get returns the recorded list, so a caller can read the store through the
// audited wrapper.
func (s *AuditedStore) Get(dataID string) ([]Setting, bool) {
	type getter interface {
		Get(string) ([]Setting, bool)
	}
	if g, ok := s.inner.(getter); ok {
		return g.Get(dataID)
	}
	return nil, false
}

// Entries returns the recorded changes in the order they were accepted.
func (s *AuditedStore) Entries() []AuditEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]AuditEntry(nil), s.entries...)
}

// SetClock replaces the clock, so a test can make the timestamps stable.
func (s *AuditedStore) SetClock(clock func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clock = clock
}

// ApplyAudited checks the change, records the new list, and appends one audit
// entry.
//
// The entry is written only when the change was accepted: a refused change leaves
// no trace in the log, because nothing happened to the settings. The previous value
// is read from the table the change was checked against, so the entry cannot
// disagree with what was replaced.
func (t Table) ApplyAudited(ctx context.Context, store *AuditedStore, change Change, actor string) error {
	if store == nil {
		return fmt.Errorf("no setting store is configured")
	}
	if actor == "" {
		// An unattributed change cannot be audited, so it is refused rather than
		// recorded with an empty actor.
		return fmt.Errorf("the setting change names no actor, so it cannot be audited")
	}
	updated, err := t.Change(change)
	if err != nil {
		return err
	}
	if err := store.Put(ctx, string(t.Declaration.DataID), updated); err != nil {
		return err
	}
	previous, had := t.value(change.Key, change.User)
	store.mu.Lock()
	defer store.mu.Unlock()
	store.entries = append(store.entries, AuditEntry{
		At:     store.clock(),
		Data:   t.Declaration.DataID,
		Key:    change.Key,
		User:   change.User,
		Before: previous,
		Had:    had,
		After:  change.Value,
		Actor:  actor,
	})
	return nil
}

// value returns the setting's current value and whether it existed, which is what
// the audit entry records as "before".
func (t Table) value(key, user string) (string, bool) {
	for _, setting := range t.Values {
		if setting.Key == key && setting.User == user {
			return setting.Value, true
		}
	}
	return "", false
}
