package saveunit

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
)

type simpleWorld struct {
	semantics []semantic.Namespace
}

func (s *simpleWorld) Semantics() []semantic.Namespace { return s.semantics }

func orderLineWorld() SemanticWorld {
	return &simpleWorld{semantics: []semantic.Namespace{{
		Data: []semantic.Data{{
			ID: "sales.order",
			Attributes: []semantic.Attribute{
				{ID: "sales.order.order_id", TypeRef: "std.identifier", Required: true},
				{ID: "sales.order.customer", TypeRef: "std.identifier"},
				{ID: "sales.order.status", TypeRef: "std.identifier"},
			},
			Relations: []semantic.Relation{{
				ID:          "sales.order.lines",
				Target:      "sales.line",
				Cardinality: semantic.CardinalityMany,
				Keys:        []semantic.RelationKey{{SourceAttribute: "sales.order.order_id", TargetAttribute: "sales.line.order_id"}},
				OnDelete:    "cascade",
			}},
			CandidateKeys: []semantic.CandidateKey{{ID: "order_id", Attributes: []core.CanonicalID{"sales.order.order_id"}}},
		}, {
			ID: "sales.line",
			Attributes: []semantic.Attribute{
				{ID: "sales.line.order_id", TypeRef: "std.identifier", Required: true},
				{ID: "sales.line.line_no", TypeRef: "std.integer", Required: true},
				{ID: "sales.line.product", TypeRef: "std.identifier", Required: true},
				{ID: "sales.line.qty", TypeRef: "std.integer", Required: true},
			},
			Relations: []semantic.Relation{{
				ID:          "sales.line.order",
				Target:      "sales.order",
				Cardinality: semantic.CardinalityOne,
				Keys:        []semantic.RelationKey{{SourceAttribute: "sales.line.order_id", TargetAttribute: "sales.order.order_id"}},
				OnDelete:    "restrict",
			}},
			CandidateKeys: []semantic.CandidateKey{{ID: "line_key", Attributes: []core.CanonicalID{"sales.line.order_id", "sales.line.line_no"}}},
		}},
	}}}
}

func TestPlannerOrdersParentBeforeChildren(t *testing.T) {
	world := orderLineWorld()
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       world,
		Changes: []EntityChange{
			{Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1", Fields: PatchSet{"customer": {Value: strp("C1")}}, Relations: []RelationChange{{
				Relation: "sales.order.lines", Target: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=1",
				Children: []EntityChange{{Data: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=1", Fields: PatchSet{"product": {Value: strp("A")}}}},
			}}},
		},
	}

	plan, err := Planner{World: world}.Plan(unit)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	if len(plan.Cycles) > 0 {
		t.Fatalf("unexpected cycles: %v", plan.Cycles)
	}
	if len(plan.Order) != 2 || plan.Order[0].Key != "order=ORD-1" || plan.Order[1].Key != "order=ORD-1/line=1" {
		t.Fatalf("order=%+v", plan.Order)
	}
}

func TestPlannerRejectsDependencyCycles(t *testing.T) {
	world := orderLineWorld()
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       world,
		Changes: []EntityChange{
			{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1", Relations: []RelationChange{{
				Relation: "sales.order.lines", Target: "sales.line", Kind: ChangeKindUpdate, Key: "order=ORD-1/line=1",
				Children: []EntityChange{{Data: "sales.line", Kind: ChangeKindUpdate, Key: "order=ORD-1/line=1", Relations: []RelationChange{{
					Relation: "sales.line.order", Target: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1",
					Children: []EntityChange{{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1"}},
				}}}},
			}}},
		},
	}

	plan, err := Planner{World: world}.Plan(unit)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	if len(plan.Cycles) == 0 {
		t.Fatalf("expected cycle")
	}
}

func TestSaveUnitPreservesMissingVersusNullVersusDelete(t *testing.T) {
	store := newTestStore()
	_ = store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.order"}, map[string]string{"sales.order.order_id": "ORD-1", "sales.order.customer": "C1", "sales.order.note": "keep", "sales.order.ignored": "keep", "_key": "order=ORD-1"})
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1", Fields: PatchSet{
			"sales.order.status":   {Value: strp("closed")},
			"sales.order.customer": {SetNull: true},
			"sales.order.note":     {Delete: true},
		}}},
	}

	if _, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	post, _ := adapter.ReadSnapshot([]EntityKey{{Data: "sales.order", Key: "order=ORD-1"}})
	postRow := post[EntityKey{Data: "sales.order", Key: "order=ORD-1"}]
	if postRow == nil {
		t.Fatal("order row missing after save")
	}
	if *postRow["sales.order.status"] != "closed" {
		t.Fatalf("status=%s", *postRow["sales.order.status"])
	}
	if *postRow["sales.order.customer"] != "" {
		t.Fatalf("customer not null")
	}
	if _, ok := postRow["sales.order.note"]; ok {
		t.Fatalf("deleted field still present")
	}
}

func TestSaveUnitDoesNotLoseUnspecifiedFields(t *testing.T) {
	store := newTestStore()
	_ = store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.order"}, map[string]string{"sales.order.order_id": "ORD-1", "sales.order.customer": "C1", "_key": "order=ORD-1"})
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1", Fields: PatchSet{
			"sales.order.status": {Value: strp("open")},
		}}},
	}

	if _, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	post, _ := adapter.ReadSnapshot([]EntityKey{{Data: "sales.order", Key: "order=ORD-1"}})
	postRow := post[EntityKey{Data: "sales.order", Key: "order=ORD-1"}]
	if postRow == nil {
		t.Fatal("order row missing after save")
	}
	if *postRow["sales.order.customer"] != "C1" {
		t.Fatalf("unspecified customer field was lost")
	}
}

func TestSaveUnitRejectsEmptySetReplacement(t *testing.T) {
	store := newTestStore()
	_ = store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.line"}, map[string]string{"sales.line.order_id": "ORD-1", "sales.line.line_no": "1", "_key": "order=ORD-1/line=1"})
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.line_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{{Data: "sales.line", Kind: ChangeKindUpdate, Key: "order=ORD-1/line=1", Relations: []RelationChange{{
			Relation: "sales.line.order", Target: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1", Children: []EntityChange{},
		}}}},
	}

	_, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit)
	var empty *EmptySetReplacement
	if !errors.As(err, &empty) || empty.Relation != "sales.line.order" {
		t.Fatalf("expected empty set rejection, got %v", err)
	}
}

func TestSaveUnitRollsBackOnMidSaveFailure(t *testing.T) {
	store := newTestStore()
	_ = store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.order"}, map[string]string{"sales.order.order_id": "ORD-1", "_key": "order=ORD-1"})
	memAdapter := &MemoryStoreAdapter{Store: store}
	adapter := &FailingStoreAdapter{Store: memAdapter, FailOn: 1}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{
			{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1", Fields: PatchSet{"sales.order.status": {Value: strp("open")}}},
			{Data: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=1", Fields: PatchSet{"sales.line.product": {Value: strp("A")}}},
		},
	}

	_, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(memAdapter))).Save(context.Background(), unit)
	if err == nil {
		t.Fatal("expected failure")
	}
	post, _ := memAdapter.ReadSnapshot([]EntityKey{{Data: "sales.order", Key: "order=ORD-1"}})
	postRow := post[EntityKey{Data: "sales.order", Key: "order=ORD-1"}]
	if postRow == nil {
		t.Fatal("order row missing after save")
	}
	if postRow["sales.order.status"] != nil {
		t.Fatalf("partial update was not rolled back")
	}
}

func TestSaveUnitRetriesTransientFailure(t *testing.T) {
	memAdapter := &MemoryStoreAdapter{Store: newTestStore()}
	adapter := &FailingStoreAdapter{Store: memAdapter, FailOn: 1}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes:     []EntityChange{{Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1", Fields: PatchSet{"sales.order.customer": {Value: strp("C1")}}}},
	}

	result, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(memAdapter)), WithMaxAttempts(2)).Save(context.Background(), unit)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if !result.Committed || len(result.Applied) != 1 || result.Applied[0].Key != "order=ORD-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSaveUnitOptimisticConflictViaHook(t *testing.T) {
	store := newTestStore()
	_ = store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.order"}, map[string]string{"sales.order.order_id": "ORD-1", "sales.order.status": "open", "_key": "order=ORD-1"})
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{{Data: "sales.order", Kind: ChangeKindUpdate, Key: "order=ORD-1", Fields: PatchSet{
			"sales.order.status": {Value: strp("closed")},
		}}},
	}

	hook := HookFunc(func(ctx context.Context, input HookInput) HookResult {
		snap, _ := adapter.ReadSnapshot([]EntityKey{{Data: "sales.order", Key: "order=ORD-1"}})
		if snap == nil {
			return HookResult{Allow: true}
		}
		if row, ok := snap[EntityKey{Data: "sales.order", Key: "order=ORD-1"}]; ok && row["sales.order.status"] != nil && *row["sales.order.status"] == "open" {
			return HookResult{Allow: false, Message: "concurrent modification detected"}
		}
		return HookResult{Allow: true}
	})

	_, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter)), WithValidationHooks(hook)).Save(context.Background(), unit)
	if err == nil {
		t.Fatal("expected optimistic conflict")
	}
	var hookErr *HookFailure
	if !errors.As(err, &hookErr) || hookErr.Code != "SAVE_HOOK_TEST_HOOK" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlannerOrdersCompositeKeyChildren(t *testing.T) {
	world := orderLineWorld()
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       world,
		Changes: []EntityChange{
			{Data: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=1", Fields: PatchSet{"sales.line.product": {Value: strp("A")}}},
			{Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1", Fields: PatchSet{"sales.order.customer": {Value: strp("C1")}}, Relations: []RelationChange{{
				Relation: "sales.order.lines", Target: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=1",
				Children: []EntityChange{{Data: "sales.line", Kind: ChangeKindCreate, Key: "order=ORD-1/line=1", Fields: PatchSet{"sales.line.product": {Value: strp("A")}}}},
			}}},
		},
	}

	plan, err := Planner{World: world}.Plan(unit)
	if err != nil {
		t.Fatalf("plan error: %v", err)
	}
	if len(plan.Cycles) > 0 {
		t.Fatalf("unexpected cycles: %v", plan.Cycles)
	}
	if len(plan.Order) != 2 || plan.Order[0].Key != "order=ORD-1" || plan.Order[1].Key != "order=ORD-1/line=1" {
		t.Fatalf("order=%+v", plan.Order)
	}
}

func TestSaveUnitCompositeKeyUpdatePreservesUnspecifiedFields(t *testing.T) {
	store := newTestStore()
	_ = store.Apply(memoryEffect{Kind: ChangeKindCreate, Data: "sales.line"}, map[string]string{"sales.line.order_id": "ORD-1", "sales.line.line_no": "1", "sales.line.product": "A", "sales.line.qty": "10", "_key": "order=ORD-1/line=1"})
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.line_record",
		World:       orderLineWorld(),
		Changes: []EntityChange{{Data: "sales.line", Kind: ChangeKindUpdate, Key: "order=ORD-1/line=1", Fields: PatchSet{
			"sales.line.qty": {Value: strp("20")},
		}}},
	}

	if _, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	post, _ := adapter.ReadSnapshot([]EntityKey{{Data: "sales.line", Key: "order=ORD-1/line=1"}})
	postRow := post[EntityKey{Data: "sales.line", Key: "order=ORD-1/line=1"}]
	if postRow == nil {
		t.Fatal("line row missing after save")
	}
	if *postRow["sales.line.qty"] != "20" {
		t.Fatalf("qty=%s", *postRow["sales.line.qty"])
	}
	if *postRow["sales.line.product"] != "A" {
		t.Fatalf("unspecified product field was lost")
	}
}

func TestSaveUnitBeforeAfterHooksRunAroundCommit(t *testing.T) {
	var before, after bool
	store := newTestStore()
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes:     []EntityChange{{Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1", Fields: PatchSet{"sales.order.customer": {Value: strp("C1")}}}},
		Hooks: HookPlan{
			BeforeSave: []Hook{HookFunc(func(ctx context.Context, input HookInput) HookResult { before = true; return HookResult{Allow: true} })},
			AfterSave:  []Hook{HookFunc(func(ctx context.Context, input HookInput) HookResult { after = true; return HookResult{Allow: true} })},
		},
	}

	if _, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if !before || !after {
		t.Fatalf("hooks did not run: before=%v after=%v", before, after)
	}
}

func TestSaveUnitRollsBackOnHookFailure(t *testing.T) {
	store := newTestStore()
	adapter := &MemoryStoreAdapter{Store: store}
	unit := &SaveUnit{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		World:       orderLineWorld(),
		Changes:     []EntityChange{{Data: "sales.order", Kind: ChangeKindCreate, Key: "order=ORD-1", Fields: PatchSet{"sales.order.customer": {Value: strp("C1")}}}},
		Hooks: HookPlan{
			BeforeSave: []Hook{HookFunc(func(ctx context.Context, input HookInput) HookResult {
				return HookResult{Allow: false, Message: "deny"}
			})},
		},
	}

	_, err := NewSaver(WithStore(adapter), WithUnitOfWork(InMemoryUnitOfWork(adapter))).Save(context.Background(), unit)
	var hookErr *HookFailure
	if !errors.As(err, &hookErr) || hookErr.Code != "SAVE_HOOK_TEST_HOOK" {
		t.Fatalf("expected hook deny, got %v", err)
	}
	post, _ := adapter.ReadSnapshot([]EntityKey{{Data: "sales.order", Key: "order=ORD-1"}})
	if len(post) != 0 {
		t.Fatalf("hook denial did not roll back: %+v", post)
	}
}

func strp(v string) *string { return &v }

// HookFunc is a test helper implementing Hook.
type HookFunc func(context.Context, HookInput) HookResult

func (f HookFunc) ID() string                                          { return "test-hook" }
func (f HookFunc) Run(ctx context.Context, input HookInput) HookResult { return f(ctx, input) }

type testStore struct {
	data      map[core.CanonicalID]map[string]map[string]string
	snapshots map[EntityKey]map[string]*string
}

func newTestStore() *testStore {
	return &testStore{data: map[core.CanonicalID]map[string]map[string]string{}, snapshots: map[EntityKey]map[string]*string{}}
}

func (t *testStore) Get(data core.CanonicalID, key string) (map[string]string, bool) {
	row, ok := t.data[data][key]
	if !ok {
		return nil, false
	}
	out := make(map[string]string, len(row))
	for k, v := range row {
		out[k] = v
	}
	return out, true
}

func (t *testStore) Apply(effect memoryEffect, inputs map[string]string) error {
	if t.data[effect.Data] == nil {
		t.data[effect.Data] = map[string]map[string]string{}
	}
	switch effect.Kind {
	case ChangeKindCreate:
		if _, ok := t.data[effect.Data][inputs["_key"]]; ok {
			return &ReferenceFailure{Code: ReferenceRecordExists, Message: "record exists"}
		}
	case ChangeKindUpdate:
		if _, ok := t.data[effect.Data][inputs["_key"]]; !ok {
			return &ReferenceFailure{Code: ReferenceRecordMissing, Message: "record missing"}
		}
	case ChangeKindDelete:
		if _, ok := t.data[effect.Data][inputs["_key"]]; !ok {
			return &ReferenceFailure{Code: ReferenceRecordMissing, Message: "record missing"}
		}
		delete(t.data[effect.Data], inputs["_key"])
		return nil
	default:
		return errors.New("unknown kind")
	}
	row := make(map[string]string, len(inputs))
	for k, v := range inputs {
		row[k] = v
	}
	t.data[effect.Data][inputs["_key"]] = row
	return nil
}

func (t *testStore) CommitPending(pending map[core.CanonicalID]map[string]map[string]string) {
	for data, rows := range pending {
		if t.data[data] == nil {
			t.data[data] = map[string]map[string]string{}
		}
		for key, row := range rows {
			if row == nil {
				delete(t.data[data], key)
				continue
			}
			t.data[data][key] = row
		}
	}
}
