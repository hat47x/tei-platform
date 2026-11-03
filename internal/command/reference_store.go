package command

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"tei/internal/applicationview"
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

// Applyは、入力全体からキーを作り、宣言された効果を記憶域へ適用する。
func (s *MemoryStore) Apply(effect *semantic.DataEffect, inputs map[string]string) error {
	return s.ApplyWithKey(effect, recordKey(inputs), inputs)
}

// ApplyWithKeyは、呼出し側が決めたキーで効果を適用する。
func (s *MemoryStore) ApplyWithKey(effect *semantic.DataEffect, key string, inputs map[string]string) error {
	if effect == nil {
		return fmt.Errorf("data effect is not declared")
	}
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
		if _, exists := s.data[effect.Data][key]; exists {
			return fmt.Errorf("record %s already exists", key)
		}
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

// Getは、キーに対応するレコードを返す。
func (s *MemoryStore) Get(data core.CanonicalID, key string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.data[data][key]
	if !ok {
		return nil, false
	}
	return copyValues(record), true
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
	keys := make([]string, 0, len(s.data[data]))
	for key := range s.data[data] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, copyValues(s.data[data][key]))
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
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keyText(inputs, keys)
}

func recordKeyFields(inputs map[string]string, fields []string) string {
	if len(fields) == 0 {
		return recordKey(inputs)
	}
	keys := append([]string(nil), fields...)
	sort.Strings(keys)
	return keyText(inputs, keys)
}

func keyText(inputs map[string]string, keys []string) string {
	if len(keys) == 0 {
		return ""
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
	World     ir.World
	Store     *MemoryStore
	KeyFields []string
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
	if capability.Effect != nil && (capability.Effect.Kind == semantic.DataEffectCreate || capability.Effect.Kind == semantic.DataEffectUpdate) {
		if capability.Effect.Kind == semantic.DataEffectCreate {
			if err := e.checkRequiredInputs(request); err != nil {
				return Result{}, err
			}
		}
		if err := e.checkValueDomains(request); err != nil {
			return Result{}, err
		}
		if capability.Effect.Kind == semantic.DataEffectUpdate {
			if err := e.checkImmutableInputs(request); err != nil {
				return Result{}, err
			}
		}
	}
	if capability.Effect != nil && capability.Effect.Kind == semantic.DataEffectDelete {
		if err := e.checkDeletePolicy(capability.Effect.Data); err != nil {
			return Result{}, err
		}
	}
	key := recordKeyFields(request.Inputs, e.KeyFields)
	if err := e.Store.ApplyWithKey(capability.Effect, key, request.Inputs); err != nil {
		return Result{}, err
	}
	result := Result{Effect: capability.Effect}
	if len(capability.Outcomes) > 0 {
		outcome := capability.Outcomes[0]
		result.Outcome = &outcome
	}
	return result, nil
}

func (e ReferenceDataExecutor) checkImmutableInputs(request Request) error {
	plan, err := applicationview.BuildCommandPlan(e.World, request.Application, request.Pattern, request.Capability, request.Inputs)
	if err != nil {
		return nil
	}
	var dataID core.CanonicalID
	for i := range e.World.Applications {
		if e.World.Applications[i].ID != request.Application {
			continue
		}
		for j := range e.World.Applications[i].Patterns {
			if e.World.Applications[i].Patterns[j].ID == request.Pattern {
				dataID = e.World.Applications[i].Patterns[j].Data
			}
		}
	}
	if dataID == "" {
		return nil
	}
	record, ok := e.Store.Get(dataID, recordKeyFields(request.Inputs, e.KeyFields))
	if !ok {
		return nil
	}
	immutable := map[core.CanonicalID]bool{}
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			for _, attribute := range data.Attributes {
				if attribute.Immutable {
					immutable[attribute.ID] = true
				}
			}
		}
	}
	for _, input := range plan.Inputs {
		if !input.Provided || !immutable[input.Attribute] {
			continue
		}
		if old, exists := record[input.Name]; exists && old != input.Value {
			return fmt.Errorf("immutable attribute %s cannot change", input.Attribute)
		}
	}
	return nil
}

func (e ReferenceDataExecutor) checkValueDomains(request Request) error {
	plan, err := applicationview.BuildCommandPlan(e.World, request.Application, request.Pattern, request.Capability, request.Inputs)
	if err != nil {
		return nil
	}
	attributes := map[core.CanonicalID]semantic.Attribute{}
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = attribute
			}
		}
	}
	types := map[core.CanonicalID]semantic.Type{}
	for _, namespace := range e.World.Semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
	}
	for _, input := range plan.Inputs {
		if !input.Provided || input.Attribute == "" {
			continue
		}
		attribute, ok := attributes[input.Attribute]
		if !ok {
			continue
		}
		typ, ok := types[attribute.TypeRef]
		if !ok {
			continue
		}
		for depth := 0; depth < 16 && len(typ.Values) == 0; depth++ {
			if typ.Kind != semantic.TypeAlias || typ.Underlying == "" {
				break
			}
			next, exists := types[typ.Underlying]
			if !exists {
				break
			}
			typ = next
		}
		if len(typ.Values) == 0 {
			continue
		}
		allowed := false
		for _, value := range typ.Values {
			if value == input.Value {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("value %s is outside the domain of %s", input.Value, input.Attribute)
		}
	}
	return nil
}

func (e ReferenceDataExecutor) checkRequiredInputs(request Request) error {
	var dataID core.CanonicalID
	for i := range e.World.Applications {
		if e.World.Applications[i].ID != request.Application {
			continue
		}
		for j := range e.World.Applications[i].Patterns {
			if e.World.Applications[i].Patterns[j].ID == request.Pattern {
				dataID = e.World.Applications[i].Patterns[j].Data
			}
		}
	}
	if dataID == "" {
		return nil
	}
	plan, err := applicationview.BuildCommandPlan(e.World, request.Application, request.Pattern, request.Capability, request.Inputs)
	if err != nil {
		return nil
	}
	provided := map[core.CanonicalID]bool{}
	bound := false
	for _, input := range plan.Inputs {
		if input.Attribute != "" {
			bound = true
		}
		if input.Provided && input.Attribute != "" {
			provided[input.Attribute] = true
		}
	}
	if !bound {
		// command Patternはattributeへの束縛を持たないため、必須属性を判定できない。
		return nil
	}
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			for _, attribute := range data.Attributes {
				if attribute.Required && !provided[attribute.ID] {
					return fmt.Errorf("required attribute %s is not provided", attribute.ID)
				}
			}
		}
	}
	return nil
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
