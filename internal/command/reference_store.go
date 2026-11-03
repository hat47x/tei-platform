package command

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// MemoryStoreは、Commandの効果を適用する参照用の記憶域である。
// 実データベースではなく、参照実装の配線とRollbackを確認するためだけに使う。
type MemoryStore struct {
	mu   sync.RWMutex
	data map[core.CanonicalID]map[string]map[string]string
}

// NewMemoryStoreは、空の参照記憶域を作る。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: map[core.CanonicalID]map[string]map[string]string{}}
}

// Applyは、宣言された効果を記憶域へ適用する。
func (s *MemoryStore) Apply(effect *semantic.DataEffect, inputs map[string]string) error {
	if effect == nil {
		return fmt.Errorf("data effect is not declared")
	}
	key := recordKey(inputs)
	if key == "" {
		return fmt.Errorf("record key is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[effect.Data] == nil {
		s.data[effect.Data] = map[string]map[string]string{}
	}
	switch effect.Kind {
	case semantic.DataEffectCreate:
		s.data[effect.Data][key] = copyValues(inputs)
	case semantic.DataEffectUpdate:
		if _, ok := s.data[effect.Data][key]; !ok {
			return fmt.Errorf("record %s is not present", key)
		}
		s.data[effect.Data][key] = copyValues(inputs)
	case semantic.DataEffectDelete:
		if _, ok := s.data[effect.Data][key]; !ok {
			return fmt.Errorf("record %s is not present", key)
		}
		delete(s.data[effect.Data], key)
	default:
		return fmt.Errorf("unsupported data effect %s", effect.Kind)
	}
	return nil
}

// DeleteAllは、Dataのレコードをすべて削除する。
func (s *MemoryStore) DeleteAll(data core.CanonicalID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, data)
}

// Recordsは、Dataの現在のレコードを返す。
func (s *MemoryStore) Records(data core.CanonicalID) []map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []map[string]string
	for _, record := range s.data[data] {
		out = append(out, copyValues(record))
	}
	return out
}

func (s *MemoryStore) snapshot() map[core.CanonicalID]map[string]map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[core.CanonicalID]map[string]map[string]string{}
	for data, records := range s.data {
		out[data] = map[string]map[string]string{}
		for key, record := range records {
			out[data][key] = copyValues(record)
		}
	}
	return out
}

func (s *MemoryStore) restore(snapshot map[core.CanonicalID]map[string]map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = snapshot
}

func recordKey(inputs map[string]string) string {
	if len(inputs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+inputs[key])
	}
	return strings.Join(parts, "\x00")
}

func copyValues(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

// ReferenceDataExecutorは、宣言された効果をMemoryStoreへ適用する参照実装である。
// 業務ロジックは持たない。
type ReferenceDataExecutor struct {
	World ir.World
	Store *MemoryStore
}

func (e ReferenceDataExecutor) Execute(_ context.Context, request Request) (Result, error) {
	if e.Store == nil {
		return Result{}, fmt.Errorf("reference store is not configured")
	}
	var capability *semantic.Capability
	for i := range e.World.Semantics {
		for j := range e.World.Semantics[i].Capabilities {
			if e.World.Semantics[i].Capabilities[j].ID == request.Capability {
				capability = &e.World.Semantics[i].Capabilities[j]
				break
			}
		}
		if capability != nil {
			break
		}
	}
	if capability == nil {
		return Result{}, fmt.Errorf("capability %s is not defined", request.Capability)
	}
	if capability.Effect != nil && capability.Effect.Kind == semantic.DataEffectDelete {
		if err := e.checkDeletePolicy(capability.Effect.Data); err != nil {
			return Result{}, err
		}
	}
	if err := e.Store.Apply(capability.Effect, request.Inputs); err != nil {
		return Result{}, err
	}
	result := Result{Effect: capability.Effect}
	if len(capability.Outcomes) > 0 {
		outcome := capability.Outcomes[0]
		result.Outcome = &outcome
	}
	return result, nil
}

func (e ReferenceDataExecutor) checkDeletePolicy(target core.CanonicalID) error {
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.Target != target {
					continue
				}
				if len(e.Store.Records(data.ID)) == 0 {
					continue
				}
				switch relation.OnDelete {
				case "cascade":
					e.Store.DeleteAll(data.ID)
				case "preserve":
				default:
					return fmt.Errorf("inbound relation %s restricts delete of %s", relation.ID, target)
				}
			}
		}
	}
	return nil
}

// StoreUnitOfWorkは、MemoryStoreのスナップショットでRollbackする参照実装である。
type StoreUnitOfWork struct {
	Store      *MemoryStore
	FailCommit bool
}

func (u *StoreUnitOfWork) Begin(context.Context) (Transaction, error) {
	if u.Store == nil {
		return nil, fmt.Errorf("reference store is not configured")
	}
	return &storeTransaction{unit: u, snapshot: u.Store.snapshot()}, nil
}

type storeTransaction struct {
	unit     *StoreUnitOfWork
	snapshot map[core.CanonicalID]map[string]map[string]string
	done     bool
}

func (t *storeTransaction) Commit(context.Context) error {
	if t.unit.FailCommit {
		return fmt.Errorf("reference unit of work cannot commit")
	}
	t.done = true
	return nil
}

func (t *storeTransaction) Rollback(context.Context) error {
	if t.done {
		return nil
	}
	t.unit.Store.restore(t.snapshot)
	t.done = true
	return nil
}
