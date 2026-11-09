package command

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/unicode/norm"

	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// MemoryStoreは、Commandの効果を適用する参照用の記憶域である。
// 実データベースではなく、参照実装の配線とRollbackを確認するためだけに使う。
type MemoryStore struct {
	mu         sync.RWMutex
	data       map[core.CanonicalID]map[string]map[string]string
	attributes map[core.CanonicalID]map[string]map[core.CanonicalID]string
}

// NewMemoryStoreは、空の参照記憶域を作る。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:       map[core.CanonicalID]map[string]map[string]string{},
		attributes: map[core.CanonicalID]map[string]map[core.CanonicalID]string{},
	}
}

// Applyは、入力全体からキーを作り、宣言された効果を記憶域へ適用する。
func (s *MemoryStore) Apply(effect *semantic.DataEffect, inputs map[string]string) error {
	return s.ApplyWithKey(effect, recordKey(inputs), inputs)
}

// ApplyWithKeyは、呼出し側が決めたキーで効果を適用する。
func (s *MemoryStore) ApplyWithKey(effect *semantic.DataEffect, key string, inputs map[string]string) error {
	return s.ApplyRecord(effect, key, inputs, nil)
}

// ApplyRecordは、レコード値と属性値の対応を一緒に記憶域へ適用する。
func (s *MemoryStore) ApplyRecord(effect *semantic.DataEffect, key string, inputs map[string]string, attributes map[core.CanonicalID]string) error {
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
	if s.attributes[effect.Data] == nil {
		s.attributes[effect.Data] = map[string]map[core.CanonicalID]string{}
	}
	switch effect.Kind {
	case semantic.DataEffectCreate:
		if _, exists := s.data[effect.Data][key]; exists {
			return referenceFailure(ReferenceRecordExists, "create", "record %s already exists", key)
		}
		s.data[effect.Data][key] = copyValues(inputs)
		if attributes != nil {
			s.attributes[effect.Data][key] = copyAttributes(attributes)
		}
	case semantic.DataEffectUpdate:
		if _, ok := s.data[effect.Data][key]; !ok {
			return referenceFailure(ReferenceRecordMissing, "update", "record %s is not present", key)
		}
		s.data[effect.Data][key] = copyValues(inputs)
		if attributes != nil {
			s.attributes[effect.Data][key] = copyAttributes(attributes)
		}
	case semantic.DataEffectDelete:
		if _, ok := s.data[effect.Data][key]; !ok {
			return referenceFailure(ReferenceRecordMissing, "delete", "record %s is not present", key)
		}
		delete(s.data[effect.Data], key)
		delete(s.attributes[effect.Data], key)
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
	delete(s.attributes, data)
}

// DeleteKeyは、キーに対応するレコードを削除する。
func (s *MemoryStore) DeleteKey(data core.CanonicalID, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data[data], key)
	delete(s.attributes[data], key)
}

// RecordSnapshotは、キー・レコード値・属性値をまとめた参照用の写しである。
type RecordSnapshot struct {
	Key        string
	Values     map[string]string
	Attributes map[core.CanonicalID]string
}

// RecordsWithAttributesは、Dataのレコードをキー順に返す。
func (s *MemoryStore) RecordsWithAttributes(data core.CanonicalID) []RecordSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data[data]))
	for key := range s.data[data] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]RecordSnapshot, 0, len(keys))
	for _, key := range keys {
		out = append(out, RecordSnapshot{
			Key:        key,
			Values:     copyValues(s.data[data][key]),
			Attributes: copyAttributes(s.attributes[data][key]),
		})
	}
	return out
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

// StoreSnapshot is a consistent copy of values and attribute values.
type StoreSnapshot struct {
	data       map[core.CanonicalID]map[string]map[string]string
	attributes map[core.CanonicalID]map[string]map[core.CanonicalID]string
}

// Snapshot returns a consistent copy of the store.
func (s *MemoryStore) Snapshot() StoreSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := StoreSnapshot{
		data:       map[core.CanonicalID]map[string]map[string]string{},
		attributes: map[core.CanonicalID]map[string]map[core.CanonicalID]string{},
	}
	for data, records := range s.data {
		out.data[data] = map[string]map[string]string{}
		for key, record := range records {
			out.data[data][key] = copyValues(record)
		}
	}
	for data, records := range s.attributes {
		out.attributes[data] = map[string]map[core.CanonicalID]string{}
		for key, attributes := range records {
			out.attributes[data][key] = copyAttributes(attributes)
		}
	}
	return out
}

// Restore replaces the store with the snapshot.
func (s *MemoryStore) Restore(snapshot StoreSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = snapshot.data
	s.attributes = snapshot.attributes
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

func copyAttributes(values map[core.CanonicalID]string) map[core.CanonicalID]string {
	out := make(map[core.CanonicalID]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
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
		if err := e.checkCandidateKeys(request); err != nil {
			return Result{}, err
		}
		if capability.Effect.Kind == semantic.DataEffectUpdate {
			if err := e.checkImmutableInputs(request); err != nil {
				return Result{}, err
			}
		}
	}
	plan, _ := applicationview.BuildCommandPlan(e.World, request.Application, request.Pattern, request.Capability, request.Inputs)
	targetAttributes := attributeValues(plan)
	var applied []AppliedOperation
	if capability.Effect != nil && capability.Effect.Kind == semantic.DataEffectDelete {
		snapshot := e.Store.Snapshot()
		cascaded, err := e.checkDeletePolicy(capability.Effect.Data, targetAttributes)
		if err != nil {
			e.Store.Restore(snapshot)
			return Result{}, err
		}
		applied = append(applied, cascaded...)
	}
	key := recordKeyFields(request.Inputs, e.KeyFields)
	if err := e.Store.ApplyRecord(capability.Effect, key, request.Inputs, targetAttributes); err != nil {
		return Result{}, err
	}
	applied = append(applied, AppliedOperation{Kind: capability.Effect.Kind, Data: capability.Effect.Data, Key: key})
	result := Result{Effect: capability.Effect, Applied: applied}
	switch len(capability.Outcomes) {
	case 0:
	case 1:
		outcome := capability.Outcomes[0]
		result.Outcome = &outcome
	default:
		result.OutcomeUndetermined = fmt.Sprintf(
			"capability %s declares %d outcomes and no declared rule selects one",
			capability.ID, len(capability.Outcomes),
		)
	}
	return result, nil
}

func (e ReferenceDataExecutor) checkCandidateKeys(request Request) error {
	plan, err := applicationview.BuildCommandPlan(e.World, request.Application, request.Pattern, request.Capability, request.Inputs)
	if err != nil {
		return nil
	}
	attributes := attributeValues(plan)
	var dataID core.CanonicalID
	var candidateKeys []semantic.CandidateKey
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
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID == dataID {
				candidateKeys = data.CandidateKeys
			}
		}
	}
	if len(candidateKeys) == 0 {
		return nil
	}
	types := map[core.CanonicalID]semantic.Type{}
	for _, namespace := range e.World.Semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
	}
	attributeTypes := map[core.CanonicalID]core.CanonicalID{}
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			for _, attribute := range data.Attributes {
				attributeTypes[attribute.ID] = attribute.TypeRef
			}
		}
	}
	currentKey := recordKeyFields(request.Inputs, e.KeyFields)
	for _, candidate := range candidateKeys {
		values := map[core.CanonicalID]string{}
		complete := true
		for _, attribute := range candidate.Attributes {
			value, ok := attributes[attribute]
			if !ok {
				complete = false
				break
			}
			values[attribute] = value
		}
		if !complete {
			continue
		}
		normalized := map[core.CanonicalID]string{}
		for attribute, value := range values {
			normalized[attribute] = normalizeValue(attributeTypes[attribute], value, types)
		}
		for _, record := range e.Store.RecordsWithAttributes(dataID) {
			if record.Key == currentKey {
				continue
			}
			matched := true
			for attribute, value := range normalized {
				if normalizeValue(attributeTypes[attribute], record.Attributes[attribute], types) != value {
					matched = false
					break
				}
			}
			if matched {
				return referenceFailure(ReferenceValueRejected, "candidate-key", "candidate key %s is duplicated", candidate.ID)
			}
		}
	}
	return nil
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
			return referenceFailure(ReferenceValueRejected, "immutable", "immutable attribute %s cannot change", input.Attribute)
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
			return referenceFailure(ReferenceValueRejected, "value-domain", "value %s is outside the domain of %s", input.Value, input.Attribute)
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
					return referenceFailure(ReferenceValueRejected, "required", "required attribute %s is not provided", attribute.ID)
				}
			}
		}
	}
	return nil
}

func (e ReferenceDataExecutor) checkDeletePolicy(target core.CanonicalID, targetAttributes map[core.CanonicalID]string) ([]AppliedOperation, error) {
	var applied []AppliedOperation
	if err := e.cascadeDelete(target, targetAttributes, map[string]bool{}, &applied); err != nil {
		return nil, err
	}
	return applied, nil
}

func (e ReferenceDataExecutor) cascadeDelete(target core.CanonicalID, targetAttributes map[core.CanonicalID]string, visited map[string]bool, applied *[]AppliedOperation) error {
	for _, namespace := range e.World.Semantics {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.Target != target {
					continue
				}
				if len(relation.Keys) > 0 {
					for _, record := range e.Store.RecordsWithAttributes(data.ID) {
						matched := true
						for _, key := range relation.Keys {
							source, sourceOK := record.Attributes[key.SourceAttribute]
							targetValue, targetOK := targetAttributes[key.TargetAttribute]
							if !sourceOK || !targetOK || source != targetValue {
								matched = false
								break
							}
						}
						if !matched {
							continue
						}
						switch relation.OnDelete {
						case "cascade":
							visitKey := string(data.ID) + "\\x00" + record.Key
							if visited[visitKey] {
								continue
							}
							visited[visitKey] = true
							e.Store.DeleteKey(data.ID, record.Key)
							*applied = append(*applied, AppliedOperation{Kind: semantic.DataEffectDelete, Data: data.ID, Key: record.Key})
							if err := e.cascadeDelete(data.ID, record.Attributes, visited, applied); err != nil {
								return err
							}
						case "preserve":
						default:
							return referenceFailure(ReferenceRuleRejected, "delete-policy", "inbound relation %s restricts delete of %s", relation.ID, target)
						}
					}
					continue
				}
				if len(e.Store.Records(data.ID)) == 0 {
					continue
				}
				switch relation.OnDelete {
				case "cascade":
					for _, record := range e.Store.RecordsWithAttributes(data.ID) {
						visitKey := string(data.ID) + "\\x00" + record.Key
						if visited[visitKey] {
							continue
						}
						visited[visitKey] = true
						e.Store.DeleteKey(data.ID, record.Key)
						*applied = append(*applied, AppliedOperation{Kind: semantic.DataEffectDelete, Data: data.ID, Key: record.Key})
						if err := e.cascadeDelete(data.ID, record.Attributes, visited, applied); err != nil {
							return err
						}
					}
				case "preserve":
				default:
					return referenceFailure(ReferenceRuleRejected, "delete-policy", "inbound relation %s restricts delete of %s", relation.ID, target)
				}
			}
		}
	}
	return nil
}

func normalizeValue(typeRef core.CanonicalID, value string, types map[core.CanonicalID]semantic.Type) string {
	primitive := typeRef
	for depth := 0; depth < 16; depth++ {
		typ, ok := types[primitive]
		if !ok || typ.Kind != semantic.TypeAlias || typ.Underlying == "" {
			break
		}
		primitive = typ.Underlying
	}
	switch primitive {
	case "std.integer":
		if number, err := strconv.ParseInt(value, 10, 64); err == nil {
			return strconv.FormatInt(number, 10)
		}
	case "std.decimal", "std.money":
		if number, err := strconv.ParseFloat(value, 64); err == nil {
			return strconv.FormatFloat(number, 'f', -1, 64)
		}
	case "std.boolean":
		if strings.EqualFold(value, "true") {
			return "true"
		}
		if strings.EqualFold(value, "false") {
			return "false"
		}
	case "std.date":
		if date, err := time.Parse("2006-01-02", value); err == nil {
			return date.Format("2006-01-02")
		}
	case "std.datetime":
		if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
			return timestamp.UTC().Format(time.RFC3339Nano)
		}
	case "std.time":
		if clock, err := time.Parse("15:04:05", value); err == nil {
			return clock.Format("15:04:05")
		}
	case "std.identifier", "std.text", "std.name":
		return norm.NFC.String(value)
	}
	return value
}

func attributeValues(plan applicationview.CommandPlan) map[core.CanonicalID]string {
	out := map[core.CanonicalID]string{}
	for _, input := range plan.Inputs {
		if input.Provided && input.Attribute != "" {
			out[input.Attribute] = input.Value
		}
	}
	return out
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
	return &storeTransaction{unit: u, snapshot: u.Store.Snapshot()}, nil
}

type storeTransaction struct {
	unit     *StoreUnitOfWork
	snapshot StoreSnapshot
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
	t.unit.Store.Restore(t.snapshot)
	t.done = true
	return nil
}
