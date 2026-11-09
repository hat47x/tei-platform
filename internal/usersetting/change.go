package usersetting

import (
	"context"
	"fmt"
	"sync"
)

// Change is a requested update of one setting.
//
// X-TEA's script API wrote a value straight into the session's variable map
// (setVariant, setUserValueOf: D:Session.java L2112, D:XFScriptable.java L56–60),
// so the write was not checked against any declaration and nothing recorded who
// changed what. Here a change names a declared key, and a key that is not
// declared is refused instead of creating a new one by accident.
type Change struct {
	Key   string
	User  string
	Value string
}

// Table is a setting Data together with the settings it holds, so a change can be
// checked against the declaration and the current rows at once.
type Table struct {
	Declaration Declaration
	Settings    Settings
	// Values is the bound setting list the table was built from. A change
	// rewrites this list, so a caller sees the whole table after the update.
	Values []Setting
}

// NewTable indexes the settings and keeps the declaration beside them.
func NewTable(declaration Declaration, settings []Setting) (Table, error) {
	indexed, err := NewSettings(declaration.User != "", settings)
	if err != nil {
		return Table{}, err
	}
	return Table{Declaration: declaration, Settings: indexed, Values: append([]Setting(nil), settings...)}, nil
}

// Change checks a requested update and returns the full setting list after it.
//
// The rules are the ones the declaration makes checkable:
//   - the key must already be declared somewhere in the table, so a typo does not
//     create a second setting;
//   - a Data with a user role needs the user named for a user's own row, because a
//     row without a user is the shared default and is not the same setting;
//   - a Data without a user role refuses a user, because there is no row to hold it.
//
// The change does not write: the caller passes the returned list to a store.
func (t Table) Change(change Change) ([]Setting, error) {
	if t.Declaration.Key == "" {
		return nil, fmt.Errorf("setting data %s is not declared", t.Declaration.DataID)
	}
	if change.Key == "" {
		return nil, fmt.Errorf("the setting key is not named")
	}
	if t.Declaration.User == "" && change.User != "" {
		return nil, fmt.Errorf("setting data %s does not declare %s, so a change cannot name a user", t.Declaration.DataID, RoleUser)
	}
	if !t.declares(change.Key) {
		return nil, fmt.Errorf("setting %s is not declared, so it cannot be changed", change.Key)
	}

	out := make([]Setting, 0, len(t.Values)+1)
	replaced := false
	for _, setting := range t.Values {
		if setting.Key == change.Key && setting.User == change.User {
			out = append(out, Setting{Key: change.Key, User: change.User, Value: change.Value})
			replaced = true
			continue
		}
		out = append(out, setting)
	}
	if !replaced {
		out = append(out, Setting{Key: change.Key, User: change.User, Value: change.Value})
	}
	// Rebuilding the index proves the result is still readable: the change must
	// not produce two rows for one key and user.
	if _, err := NewSettings(t.Declaration.User != "", out); err != nil {
		return nil, err
	}
	return out, nil
}

// declares reports whether the key appears anywhere in the table. A key declared
// for one user is declared, so a change may add it for another user.
func (t Table) declares(key string) bool {
	for _, setting := range t.Values {
		if setting.Key == key {
			return true
		}
	}
	return false
}

// Store records a whole setting list. A caller that changed one setting passes the
// list Change returned, so the store does not have to merge partial updates.
type Store interface {
	Put(ctx context.Context, dataID string, settings []Setting) error
}

// MemoryStore keeps the recorded lists in memory, for tests and for a reference
// wiring. It is not durable.
type MemoryStore struct {
	mu     sync.Mutex
	stored map[string][]Setting
}

// NewMemoryStore creates an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{stored: map[string][]Setting{}}
}

// Put records the list for one Data.
func (s *MemoryStore) Put(_ context.Context, dataID string, settings []Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stored[dataID] = append([]Setting(nil), settings...)
	return nil
}

// Get returns the recorded list. A Data nothing was written for has no rows, which
// is different from a write that emptied it.
func (s *MemoryStore) Get(dataID string) ([]Setting, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, ok := s.stored[dataID]
	if !ok {
		return nil, false
	}
	return append([]Setting(nil), settings...), true
}

// Apply checks the change and records the result in one call, so a caller cannot
// check with one table and write another's list.
func (t Table) Apply(ctx context.Context, store Store, change Change) error {
	if store == nil {
		return fmt.Errorf("no setting store is configured")
	}
	updated, err := t.Change(change)
	if err != nil {
		return err
	}
	return store.Put(ctx, string(t.Declaration.DataID), updated)
}
