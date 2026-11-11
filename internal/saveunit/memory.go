package saveunit

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"tei/internal/core"
)

// MemoryStore is a minimal in-memory store interface for tests and adapters.
type MemoryStore interface {
	Get(data core.CanonicalID, key string) (map[string]string, bool)
	Apply(effect memoryEffect, inputs map[string]string) error
}

// commitableStore is implemented by test stores to allow transaction commit.
type commitableStore interface {
	CommitPending(pending map[core.CanonicalID]map[string]map[string]string)
}

// memoryEffect is the minimal effect declaration needed by the in-memory adapter.
type memoryEffect struct {
	Kind ChangeKind
	Data core.CanonicalID
}

// MemoryStoreAdapter adapts a minimal MemoryStore to saveunit.Store.
type MemoryStoreAdapter struct {
	Store   MemoryStore
	pending map[core.CanonicalID]map[string]map[string]string
}

// ReadSnapshot copies current values for the requested keys, including pending changes.
func (a *MemoryStoreAdapter) ReadSnapshot(keys []EntityKey) (StoreSnapshot, error) {
	if a.Store == nil {
		return nil, errors.New("saveunit: memory store is nil")
	}
	out := StoreSnapshot{}
	for _, key := range keys {
		values, ok := a.Store.Get(key.Data, key.Key)
		if !ok {
			values = nil
		}
		if pending, ok := a.pending[key.Data]; ok {
			if row, ok := pending[key.Key]; ok {
				merged := make(map[string]string, len(values))
				for k, v := range values {
					merged[k] = v
				}
				for k, v := range row {
					merged[k] = v
				}
				values = merged
			}
		}
		if values == nil {
			continue
		}
		ptrValues := make(map[string]*string, len(values))
		for k, v := range values {
			clone := v
			ptrValues[k] = &clone
		}
		out[key] = ptrValues
	}
	return out, nil
}

// ApplyChanges applies the planned save unit to the underlying memory store in
// dependency order, using an in-memory transaction boundary.
func (a *MemoryStoreAdapter) ApplyChanges(ctx context.Context, unit *SaveUnit, tx Transaction) (StoreSnapshot, []AppliedChange, error) {
	if a.Store == nil {
		return StoreSnapshot{}, nil, errors.New("saveunit: memory store is nil")
	}
	if tx == nil {
		return StoreSnapshot{}, nil, errors.New("saveunit: transaction is nil")
	}

	a.pending = map[core.CanonicalID]map[string]map[string]string{}
	applied := make([]AppliedChange, 0, len(unit.Changes))
	for _, op := range unit.Changes {
		if err := applyMemoryEntity(ctx, unit.Application, unit.Pattern, a, op, &applied); err != nil {
			_ = tx.Rollback(ctx)
			a.pending = nil
			return StoreSnapshot{}, nil, err
		}
	}

	snapshot, err := a.ReadSnapshot(entityKeysFromApplied(applied))
	if err != nil {
		_ = tx.Rollback(ctx)
		a.pending = nil
		return StoreSnapshot{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		_ = tx.Rollback(ctx)
		a.pending = nil
		return StoreSnapshot{}, nil, &SaveError{Code: "SAVE_TRANSACTION_COMMIT_FAILED", Message: err.Error(), Cause: err, EffectUnknown: true}
	}
	a.pending = nil
	return snapshot, applied, nil
}

// Release is a no-op for the memory adapter.
func (a *MemoryStoreAdapter) Release() {}

func applyMemoryEntity(_ context.Context, app, pattern core.CanonicalID, adapter *MemoryStoreAdapter, change EntityChange, applied *[]AppliedChange) error {
	if adapter.Store == nil {
		return errors.New("saveunit: memory store is nil")
	}
	switch change.Kind {
	case ChangeKindCreate:
		if _, ok := adapter.Store.Get(change.Data, change.Key); ok {
			return wrapApplyError(&ReferenceFailure{Code: ReferenceRecordExists, Message: "record exists"}, change)
		}
		if adapter.pending[change.Data] == nil {
			adapter.pending[change.Data] = map[string]map[string]string{}
		}
		adapter.pending[change.Data][change.Key] = stringMap(change.Fields, change.Key)
	case ChangeKindUpdate:
		current, _ := adapter.Store.Get(change.Data, change.Key)
		if pending, ok := adapter.pending[change.Data]; ok {
			if row, ok := pending[change.Key]; ok {
				current = row
			}
		}
		merged := mergeStringMap(current, change.Fields, change.Key)
		if adapter.pending[change.Data] == nil {
			adapter.pending[change.Data] = map[string]map[string]string{}
		}
		adapter.pending[change.Data][change.Key] = merged
	case ChangeKindDelete:
		if adapter.pending[change.Data] == nil {
			adapter.pending[change.Data] = map[string]map[string]string{}
		}
		adapter.pending[change.Data][change.Key] = nil
	default:
		return &SaveError{Code: "SAVE_UNKNOWN_KIND", Message: fmt.Sprintf("unsupported change kind: %s", change.Kind)}
	}
	*applied = append(*applied, AppliedChange{Data: change.Data, Key: change.Key, Kind: change.Kind})
	return nil
}

func mergeStringMap(current map[string]string, patch PatchSet, key string) map[string]string {
	out := make(map[string]string, len(current))
	for k, v := range current {
		out[k] = v
	}
	for field, change := range patch {
		switch {
		case change.Unchanged:
			// preserve current value
		case change.Delete:
			delete(out, field)
		case change.SetNull:
			out[field] = ""
		case change.Value != nil:
			out[field] = *change.Value
		default:
			delete(out, field)
		}
	}
	out["_key"] = key
	return out
}

func wrapApplyError(err error, change EntityChange) error {
	return &SaveError{Code: "SAVE_APPLY_FAILED", Message: fmt.Sprintf("apply %s %s failed: %s", change.Kind, change.Key, err.Error()), Cause: err}
}

func stringMap(patch PatchSet, key string) map[string]string {
	out := make(map[string]string, len(patch)+1)
	for field, change := range patch {
		if change.Delete {
			continue
		}
		if change.SetNull || change.Value == nil {
			out[field] = ""
			continue
		}
		out[field] = *change.Value
	}
	out["_key"] = key
	return out
}

func entityKeysFromApplied(applied []AppliedChange) []EntityKey {
	keys := make([]EntityKey, 0, len(applied))
	for _, a := range applied {
		keys = append(keys, EntityKey{Data: a.Data, Key: a.Key})
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Data != keys[j].Data {
			return keys[i].Data < keys[j].Data
		}
		return keys[i].Key < keys[j].Key
	})
	return keys
}

// InMemoryTransaction is a simple in-memory transaction wrapper.
type InMemoryTransaction struct {
	adapter *MemoryStoreAdapter
}

// Commit merges pending changes into the underlying store.
func (t *InMemoryTransaction) Commit(_ context.Context) error {
	if t.adapter == nil || t.adapter.Store == nil {
		return nil
	}
	if cs, ok := t.adapter.Store.(commitableStore); ok {
		cs.CommitPending(t.adapter.pending)
	}
	return nil
}

// Rollback discards pending changes.
func (t *InMemoryTransaction) Rollback(_ context.Context) error {
	t.adapter.pending = nil
	return nil
}

// InMemoryUnitOfWork returns a UnitOfWorkTx that creates a new transaction each call.
func InMemoryUnitOfWork(adapter *MemoryStoreAdapter) UnitOfWorkTx {
	return UnitOfWorkTx{Start: func(ctx context.Context) (Transaction, error) {
		return &InMemoryTransaction{adapter: adapter}, nil
	}}
}

// FailingStoreAdapter fails the first N apply calls.
type FailingStoreAdapter struct {
	Store    Store
	FailOn   int
	attempts int
}

// ReadSnapshot delegates.
func (a *FailingStoreAdapter) ReadSnapshot(keys []EntityKey) (StoreSnapshot, error) {
	return a.Store.ReadSnapshot(keys)
}

// ApplyChanges fails the first N calls.
func (a *FailingStoreAdapter) ApplyChanges(ctx context.Context, unit *SaveUnit, tx Transaction) (StoreSnapshot, []AppliedChange, error) {
	if a.FailOn > 0 && a.attempts < a.FailOn {
		a.attempts++
		return StoreSnapshot{}, nil, &SaveError{Code: "SAVE_PROVIDER_TRANSIENT", Message: "transient provider failure"}
	}
	return a.Store.ApplyChanges(ctx, unit, tx)
}

// Release delegates.
func (a *FailingStoreAdapter) Release() { a.Store.Release() }
