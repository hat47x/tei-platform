package saveunit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// SemanticWorld is the minimal semantic context needed for save ordering.
type SemanticWorld interface {
	Semantics() []semantic.Namespace
}

// ChangeKind names the business intent for one entity in a save unit.
type ChangeKind string

const (
	ChangeKindCreate ChangeKind = "create"
	ChangeKindUpdate ChangeKind = "update"
	ChangeKindDelete ChangeKind = "delete"
)

// FieldChange distinguishes four meanings for one field in a partial update.
type FieldChange struct {
	Value     *string // present value; nil means no meaning assigned
	Delete    bool    // explicit delete marker, distinct from missing or NULL
	SetNull   bool    // explicit NULL assignment
	Unchanged bool    // field is included but must not be changed
}

// PatchSet is the explicit field-level change set for one entity.
// A nil or empty map means the caller did not specify any field changes.
// A key with FieldChange{} means the field was intentionally left as-is.
type PatchSet map[string]FieldChange

// EntityChange is one node in a hierarchical save unit.
type EntityChange struct {
	Data      core.CanonicalID
	Kind      ChangeKind
	Key       string
	Fields    PatchSet
	Relations []RelationChange
	Children  []EntityChange
}

// RelationChange describes an edge that must be applied with the parent.
type RelationChange struct {
	Relation core.CanonicalID
	Target   core.CanonicalID
	Kind     ChangeKind
	Key      string
	Fields   PatchSet
	Children []EntityChange
}

// SaveUnit is the root of a model-driven save request.
type SaveUnit struct {
	Application core.CanonicalID
	Pattern     core.CanonicalID
	World       SemanticWorld
	Owner       string
	Generation  int
	Changes     []EntityChange
	Hooks       HookPlan
}

// HookPlan declares pre/post/validation hooks around a save unit.
type HookPlan struct {
	BeforeSave []Hook
	AfterSave  []Hook
	OnFailure  []Hook
	Validation []Hook
	RowScoped  bool
	Reads      []core.CanonicalID
}

// Hook is a pluggable save boundary hook.
type Hook interface {
	ID() string
	Run(ctx context.Context, input HookInput) HookResult
}

// HookInput is the read-only context given to hooks.
type HookInput struct {
	Application core.CanonicalID
	Pattern     core.CanonicalID
	World       SemanticWorld
	Unit        *SaveUnit
	Snapshot    StoreSnapshot
}

// HookResult can allow, deny, or warn the save unit.
type HookResult struct {
	Allow    bool
	Warnings []string
	Message  string
}

// HookFailure is returned when a hook denies the save unit.
type HookFailure struct {
	Code    string
	Message string
	Cause   error
}

func (e *HookFailure) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + " (" + e.Cause.Error() + ")"
	}
	return e.Code + ": " + e.Message
}

// Store is the provider-neutral persistence boundary for a save unit.
type Store interface {
	ReadSnapshot(keys []EntityKey) (StoreSnapshot, error)
	ApplyChanges(ctx context.Context, unit *SaveUnit, tx Transaction) (StoreSnapshot, []AppliedChange, error)
	Release()
}

// Transaction is a provider-neutral transaction boundary.
type Transaction interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// UnitOfWorkTx adapts command.UnitOfWork to saveunit.Transaction.
type UnitOfWorkTx struct {
	Start func(ctx context.Context) (Transaction, error)
}

// Begin starts a transaction from the underlying command unit of work.
func (u UnitOfWorkTx) Begin(ctx context.Context) (Transaction, error) {
	if u.Start == nil {
		return nil, errors.New("saveunit: unit of work begin is nil")
	}
	return u.Start(ctx)
}

// EntityKey identifies one stored entity.
type EntityKey struct {
	Data core.CanonicalID
	Key  string
}

// StoreSnapshot is a point-in-time copy of the affected rows.
type StoreSnapshot map[EntityKey]map[string]*string

// AppliedChange records one successful persistence effect.
type AppliedChange struct {
	Data core.CanonicalID
	Key  string
	Kind ChangeKind
}

// SaveResult reports the outcome of one save attempt.
type SaveResult struct {
	Committed     bool
	Applied       []AppliedChange
	Warnings      []string
	Persistence   PersistenceState
	TransactionID string
}

// PersistenceState is reused from command but redeclared here to keep the save
// boundary independent of CLI/Command implementations.
type PersistenceState string

const (
	PersistenceNotApplicable    PersistenceState = "not-applicable"
	PersistenceNotObserved      PersistenceState = "not-observed"
	PersistenceReceiptObserved  PersistenceState = "receipt-observed"
	PersistenceResourceObserved PersistenceState = "resource-observed"
)

// Planner builds an ordered execution plan from semantic relations.
type Planner struct {
	World SemanticWorld
}

// PlanResult contains the ordered operations and any detected semantic problems.
type PlanResult struct {
	Order       []*EntityChange
	Unreachable []*EntityChange
	Cycles      []string
}

// Plan returns a save order derived from declared relations and detects cycles.
func (p Planner) Plan(unit *SaveUnit) (*PlanResult, error) {
	if p.World == nil {
		return nil, errors.New("saveunit: planner world is nil")
	}
	flat := map[string]*EntityChange{}
	var all []*EntityChange
	collect(unit.Changes, flat, &all)

	graph := map[string][]string{}
	indegree := map[string]int{}
	for _, c := range all {
		if _, ok := indegree[c.Key]; !ok {
			indegree[c.Key] = 0
		}
		for _, rel := range entityRelations(p.World, *c) {
			childKey := childKeyFor(rel)
			if childKey == "" {
				continue
			}
			graph[c.Key] = append(graph[c.Key], childKey)
			indegree[childKey]++
		}
	}

	var order []*EntityChange
	queue := make([]*EntityChange, 0, len(all))
	for _, c := range all {
		if indegree[c.Key] == 0 {
			queue = append(queue, c)
		}
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)
		for _, parent := range graph[current.Key] {
			indegree[parent]--
			if indegree[parent] == 0 {
				queue = append(queue, flat[parent])
			}
		}
	}

	result := &PlanResult{Order: order}
	if len(order) != len(all) {
		remain := make([]string, 0, len(all)-len(order))
		for _, c := range all {
			found := false
			for _, o := range order {
				if o.Key == c.Key {
					found = true
					break
				}
			}
			if !found {
				remain = append(remain, c.Key)
			}
		}
		sort.Strings(remain)
		result.Cycles = remain
		result.Unreachable = make([]*EntityChange, 0, len(remain))
		for _, c := range all {
			for _, k := range remain {
				if c.Key == k {
					result.Unreachable = append(result.Unreachable, c)
				}
			}
		}
	}
	return result, nil
}

func collect(changes []EntityChange, flat map[string]*EntityChange, all *[]*EntityChange) {
	for i := range changes {
		c := &changes[i]
		if _, exists := flat[c.Key]; exists {
			continue
		}
		flat[c.Key] = c
		*all = append(*all, c)
		collectRelations(c.Relations, flat, all)
		collect(c.Children, flat, all)
	}
}

func collectRelations(rels []RelationChange, flat map[string]*EntityChange, all *[]*EntityChange) {
	for i := range rels {
		r := &rels[i]
		if len(r.Children) == 0 {
			continue
		}
		child := &r.Children[0]
		if _, exists := flat[r.Key]; exists {
			continue
		}
		flat[r.Key] = child
		*all = append(*all, child)
		collect(child.Children, flat, all)
	}
}

func entityRelations(world SemanticWorld, change EntityChange) []RelationChange {
	var out []RelationChange
	for _, ns := range world.Semantics() {
		for _, d := range ns.Data {
			if d.ID != change.Data {
				continue
			}
			for _, r := range d.Relations {
				for _, rel := range change.Relations {
					if rel.Relation != r.ID {
						continue
					}
					out = append(out, rel)
				}
			}
		}
	}
	return out
}

func relationsFromChange(rels []RelationChange) []RelationChange {
	var out []RelationChange
	for _, r := range rels {
		out = append(out, r)
	}
	return out
}

func entityRelationsFromChildren(children []EntityChange) []RelationChange {
	var out []RelationChange
	for _, child := range children {
		out = append(out, relationsFromChange(child.Relations)...)
		out = append(out, entityRelationsFromChildren(child.Children)...)
	}
	return out
}

func childKeyFor(rel RelationChange) string {
	if len(rel.Children) == 0 {
		return ""
	}
	return rel.Children[0].Key
}

// Saver executes a planned save unit with hooks, transaction, rollback, and
// optional retry on transient failures.
type Saver struct {
	world        SemanticWorld
	Store        Store
	UnitOfWork   UnitOfWorkTx
	MaxAttempts  int
	BeforeHooks  []Hook
	AfterHooks   []Hook
	FailureHooks []Hook
	Validation   []Hook
	Observer     PersistenceObserver
	Durable      bool
}

// PersistenceObserver performs an independent read after a successful save.
type PersistenceObserver interface {
	Observe(ctx context.Context, unit *SaveUnit, result SaveResult) (PersistenceObservation, error)
}

// PersistenceObservation says which persistence fact was observed.
type PersistenceObservation struct {
	Method string
}

// SaveResultBuilder builds a SaveResult.
type SaveResultBuilder struct {
	result SaveResult
}

// NewSaveResultBuilder starts a fresh builder.
func NewSaveResultBuilder() *SaveResultBuilder {
	return &SaveResultBuilder{result: SaveResult{Persistence: PersistenceNotObserved}}
}

// WithCommitted marks the save as committed.
func (b *SaveResultBuilder) WithCommitted() *SaveResultBuilder {
	b.result.Committed = true
	return b
}

// WithApplied adds applied changes.
func (b *SaveResultBuilder) WithApplied(changes []AppliedChange) *SaveResultBuilder {
	b.result.Applied = append(b.result.Applied, changes...)
	return b
}

// WithWarnings adds non-fatal warnings.
func (b *SaveResultBuilder) WithWarnings(warnings ...string) *SaveResultBuilder {
	b.result.Warnings = append(b.result.Warnings, warnings...)
	return b
}

// WithPersistence sets persistence state.
func (b *SaveResultBuilder) WithPersistence(state PersistenceState) *SaveResultBuilder {
	b.result.Persistence = state
	return b
}

// WithTransactionID sets the transaction id.
func (b *SaveResultBuilder) WithTransactionID(id string) *SaveResultBuilder {
	b.result.TransactionID = id
	return b
}

// Build returns the result.
func (b *SaveResultBuilder) Build() SaveResult {
	return b.result
}

// BuildPtr returns a pointer to the result for convenience.
func (b *SaveResultBuilder) BuildPtr() *SaveResult {
	return &b.result
}

// Save executes the save unit.
func (s *Saver) Save(ctx context.Context, unit *SaveUnit) (*SaveResult, error) {
	if s.Store == nil {
		return nil, &SaveError{Code: "SAVE_STORE_UNAVAILABLE", Message: "no save store is configured"}
	}
	if unit == nil || len(unit.Changes) == 0 {
		return nil, &SaveError{Code: "SAVE_UNIT_EMPTY", Message: "save unit has no changes"}
	}

	world := s.world
	if world == nil {
		world = unit.World
	}
	if world == nil {
		return nil, errors.New("saveunit: saver world is nil")
	}

	plan, err := Planner{World: world}.Plan(unit)
	if err != nil {
		return nil, err
	}
	if len(plan.Cycles) > 0 {
		return nil, &SaveError{Code: "SAVE_CYCLE", Message: fmt.Sprintf("save changes contain a dependency cycle: %s", strings.Join(plan.Cycles, ", "))}
	}

	validation := s.Validation
	if len(validation) == 0 {
		validation = unit.Hooks.Validation
	}
	if err := runHooks(ctx, validation, HookInput{Application: unit.Application, Pattern: unit.Pattern, World: world, Unit: unit}); err != nil {
		return nil, err
	}

	if err := ValidateEmptySets(unit, world); err != nil {
		return nil, err
	}

	if s.MaxAttempts <= 0 {
		s.MaxAttempts = 1
	}

	var last *SaveResult
	var lastErr error
	for attempt := 0; attempt < s.MaxAttempts; attempt++ {
		last, lastErr = s.attemptSave(ctx, unit, plan, world)
		if lastErr == nil {
			return last, nil
		}
		if !isTransientSaveError(lastErr) {
			return last, lastErr
		}
		failure := s.FailureHooks
		if len(failure) == 0 {
			failure = unit.Hooks.OnFailure
		}
		if attempt+1 < s.MaxAttempts {
			_ = runHooks(ctx, failure, HookInput{Application: unit.Application, Pattern: unit.Pattern, World: world, Unit: unit})
		}
	}
	return last, lastErr
}

func (s *Saver) attemptSave(ctx context.Context, unit *SaveUnit, plan *PlanResult, world SemanticWorld) (*SaveResult, error) {
	snapshot, err := snapshotKeys(plan.Order, s.Store)
	if err != nil {
		return nil, err
	}

	tx, err := s.UnitOfWork.Begin(ctx)
	if err != nil {
		return nil, &SaveError{Code: "SAVE_TRANSACTION_BEGIN_FAILED", Message: err.Error(), Cause: err}
	}

	before := s.BeforeHooks
	if len(before) == 0 {
		before = unit.Hooks.BeforeSave
	}
	if err := runHooks(ctx, before, HookInput{Application: unit.Application, Pattern: unit.Pattern, World: world, Unit: unit, Snapshot: snapshot}); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}

	snapshotOut, applied, applyErr := s.Store.ApplyChanges(ctx, unit, tx)
	if applyErr != nil {
		_ = tx.Rollback(ctx)
		failure := s.FailureHooks
		if len(failure) == 0 {
			failure = unit.Hooks.OnFailure
		}
		_ = runHooks(ctx, failure, HookInput{Application: unit.Application, Pattern: unit.Pattern, World: world, Unit: unit, Snapshot: snapshot})
		var hookErr *HookFailure
		if errors.As(applyErr, &hookErr) {
			return NewSaveResultBuilder().BuildPtr(), applyErr
		}
		var saveErr *SaveError
		if errors.As(applyErr, &saveErr) {
			return NewSaveResultBuilder().BuildPtr(), saveErr
		}
		return NewSaveResultBuilder().BuildPtr(), &SaveError{Code: "SAVE_APPLY_FAILED", Message: applyErr.Error(), Cause: applyErr}
	}
	if snapshotOut != nil {
		snapshot = snapshotOut
	}

	if err := tx.Commit(ctx); err != nil {
		_ = tx.Rollback(ctx)
		return NewSaveResultBuilder().BuildPtr(), &SaveError{Code: "SAVE_TRANSACTION_COMMIT_FAILED", Message: err.Error(), Cause: err, EffectUnknown: true}
	}

	after := s.AfterHooks
	if len(after) == 0 {
		after = unit.Hooks.AfterSave
	}
	_ = runHooks(ctx, after, HookInput{Application: unit.Application, Pattern: unit.Pattern, World: world, Unit: unit, Snapshot: snapshot})

	observation := PersistenceNotObserved
	if s.Durable && s.Observer != nil {
		obs, err := s.Observer.Observe(ctx, unit, NewSaveResultBuilder().WithApplied(applied).WithCommitted().Build())
		if err == nil && obs.Method != "" {
			observation = toPersistenceState(obs.Method)
		}
	}

	return NewSaveResultBuilder().WithApplied(applied).WithCommitted().WithPersistence(observation).BuildPtr(), nil
}

func snapshotKeys(ops []*EntityChange, store Store) (StoreSnapshot, error) {
	keys := make([]EntityKey, 0, len(ops))
	for _, op := range ops {
		keys = append(keys, EntityKey{Data: op.Data, Key: op.Key})
		for _, rel := range op.Relations {
			if len(rel.Children) > 0 {
				keys = append(keys, EntityKey{Data: rel.Target, Key: rel.Children[0].Key})
				keys = append(keys, collectChildKeys(rel.Children)...)
			}
		}
		keys = append(keys, collectChildKeys(op.Children)...)
	}
	snap, err := store.ReadSnapshot(keys)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		snap = StoreSnapshot{}
	}
	return snap, nil
}

func collectChildKeys(children []EntityChange) []EntityKey {
	var out []EntityKey
	for i := range children {
		c := &children[i]
		out = append(out, EntityKey{Data: c.Data, Key: c.Key})
		out = append(out, collectChildKeys(c.Children)...)
		for _, rel := range c.Relations {
			if len(rel.Children) > 0 {
				out = append(out, EntityKey{Data: rel.Target, Key: rel.Children[0].Key})
				out = append(out, collectChildKeys(rel.Children)...)
			}
		}
	}
	return out
}

func runHooks(ctx context.Context, hooks []Hook, input HookInput) error {
	for _, h := range hooks {
		res := h.Run(ctx, input)
		if !res.Allow {
			return &HookFailure{Code: hookCode(h.ID()), Message: res.Message}
		}
	}
	return nil
}

func hookCode(id string) string {
	if id == "" {
		return "SAVE_HOOK_DENIED"
	}
	upper := strings.ToUpper(strings.ReplaceAll(id, "-", "_"))
	if !strings.HasPrefix(upper, "SAVE_HOOK_") {
		upper = "SAVE_HOOK_" + upper
	}
	return upper
}

func isTransientSaveError(err error) bool {
	var saveErr *SaveError
	if errors.As(err, &saveErr) {
		switch saveErr.Code {
		case "SAVE_TRANSACTION_COMMIT_FAILED", "SAVE_PROVIDER_TRANSIENT":
			return true
		}
	}
	return false
}

func toPersistenceState(method string) PersistenceState {
	switch strings.ToLower(method) {
	case "receipt":
		return PersistenceReceiptObserved
	case "resource":
		return PersistenceResourceObserved
	default:
		return PersistenceNotObserved
	}
}

// SaveError is a structured save boundary failure.
type SaveError struct {
	Code          string
	Message       string
	Cause         error
	EffectUnknown bool
}

func (e *SaveError) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + " (" + e.Cause.Error() + ")"
	}
	return e.Code + ": " + e.Message
}

func (e *SaveError) Unwrap() error { return e.Cause }

// RequiredFieldMissing records a required field that was not supplied in an
// update, distinguishing that from an explicit NULL assignment.
type RequiredFieldMissing struct {
	Data  core.CanonicalID
	Field core.CanonicalID
	Key   string
}

func (e *RequiredFieldMissing) Error() string {
	return fmt.Sprintf("saveunit: required field missing for %s key=%s field=%s", e.Data, e.Key, e.Field)
}

// ConstraintViolation records a model-derived constraint failure.
type ConstraintViolation struct {
	Data       core.CanonicalID
	Constraint string
	Key        string
	Cause      error
}

func (e *ConstraintViolation) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("saveunit: constraint violated for %s key=%s constraint=%s (%s)", e.Data, e.Key, e.Constraint, e.Cause.Error())
	}
	return fmt.Sprintf("saveunit: constraint violated for %s key=%s constraint=%s", e.Data, e.Key, e.Constraint)
}

// UnchangedField records an attempt to change an immutable field.
type UnchangedField struct {
	Data  core.CanonicalID
	Field core.CanonicalID
	Key   string
}

func (e *UnchangedField) Error() string {
	return fmt.Sprintf("saveunit: immutable field changed for %s key=%s field=%s", e.Data, e.Key, e.Field)
}

// EmptySetReplacement records replacement of a populated relation with an empty
// set, when the relation did not declare an empty policy that allows it.
type EmptySetReplacement struct {
	Data        core.CanonicalID
	Relation    core.CanonicalID
	Key         string
	EmptyPolicy string
}

func (e *EmptySetReplacement) Error() string {
	return fmt.Sprintf("saveunit: empty set replacement not allowed for %s key=%s relation=%s emptyPolicy=%s", e.Data, e.Key, e.Relation, e.EmptyPolicy)
}

// MergePatchSet applies a sparse patch onto current values, preserving
// unchanged fields and distinguishing explicit deletes, NULLs, missing, and
// unchanged sentinels.
func MergePatchSet(current map[string]*string, patch PatchSet) map[string]*string {
	out := make(map[string]*string, len(current))
	for k, v := range current {
		out[k] = copyPtr(v)
	}
	if patch == nil {
		return out
	}
	for field, change := range patch {
		if change.Unchanged {
			continue
		}
		if change.Delete {
			delete(out, field)
			continue
		}
		if change.SetNull {
			out[field] = nil
			continue
		}
		if change.Value == nil {
			delete(out, field)
			continue
		}
		out[field] = copyPtr(change.Value)
	}
	return out
}

func copyPtr(v *string) *string {
	if v == nil {
		return nil
	}
	clone := *v
	return &clone
}

// DiffPatchSet returns true when the patch set contains a meaningful change
// against current values. Missing fields, unchanged sentinels, and identical
// values are not meaningful changes.
func DiffPatchSet(current map[string]*string, patch PatchSet) bool {
	if len(patch) == 0 {
		return false
	}
	for field, change := range patch {
		if change.Unchanged {
			continue
		}
		if change.Delete {
			if _, ok := current[field]; ok {
				return true
			}
			continue
		}
		if change.SetNull {
			if _, ok := current[field]; ok {
				return true
			}
			continue
		}
		if change.Value == nil {
			continue
		}
		if existing, ok := current[field]; !ok || existing == nil || *existing != *change.Value {
			return true
		}
	}
	return false
}

// ValidateEmptySets rejects replacement of a relation with an empty set when
// the relation is effectively required.
func ValidateEmptySets(unit *SaveUnit, world SemanticWorld) error {
	if world == nil {
		return nil
	}
	return validateEmpty(unit.Changes, world)
}

func validateEmpty(changes []EntityChange, world SemanticWorld) error {
	for _, change := range changes {
		for _, rel := range change.Relations {
			if rel.Kind != ChangeKindDelete && len(rel.Children) == 0 {
				data := lookupData(world, change.Data)
				relation := lookupRelation(data, rel.Relation)
				if relation == nil {
					continue
				}
				if relation.Cardinality == semantic.CardinalityOne && rel.Kind != ChangeKindDelete {
					return &EmptySetReplacement{Data: change.Data, Relation: rel.Relation, Key: change.Key, EmptyPolicy: string(semantic.EmptyPolicyDeny)}
				}
			}
			if err := validateEmpty(rel.Children, world); err != nil {
				return err
			}
		}
		if err := validateEmpty(change.Children, world); err != nil {
			return err
		}
	}
	return nil
}

func lookupData(world SemanticWorld, id core.CanonicalID) semantic.Data {
	for _, ns := range world.Semantics() {
		for _, d := range ns.Data {
			if d.ID == id {
				return d
			}
		}
	}
	return semantic.Data{}
}

func lookupRelation(data semantic.Data, id core.CanonicalID) *semantic.Relation {
	for i := range data.Relations {
		if data.Relations[i].ID == id {
			return &data.Relations[i]
		}
	}
	return nil
}

// StableOperationID returns a deterministic operation id for idempotency.
func StableOperationID(change *EntityChange) string {
	parts := []string{string(change.Kind), string(change.Data), change.Key}
	for _, rel := range change.Relations {
		parts = append(parts, stableRelationID(rel)...)
	}
	parts = append(parts, stableChildIDs(change.Children)...)
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func stableRelationID(rel RelationChange) []string {
	parts := []string{string(rel.Kind), string(rel.Relation), rel.Key}
	if len(rel.Children) > 0 {
		parts = append(parts, stableChildIDs(rel.Children)...)
	}
	sort.Strings(parts)
	return parts
}

func stableChildIDs(children []EntityChange) []string {
	var out []string
	for i := range children {
		out = append(out, StableOperationID(&children[i]))
	}
	sort.Strings(out)
	return out
}

// HookInputBuilder builds a HookInput for tests and adapters.
type HookInputBuilder struct {
	Application core.CanonicalID
	Pattern     core.CanonicalID
	World       SemanticWorld
	Unit        *SaveUnit
	Snapshot    StoreSnapshot
}

// NewHookInputBuilder creates a builder.
func NewHookInputBuilder() *HookInputBuilder {
	return &HookInputBuilder{}
}

// WithApplication sets application.
func (b *HookInputBuilder) WithApplication(app core.CanonicalID) *HookInputBuilder {
	b.Application = app
	return b
}

// WithPattern sets pattern.
func (b *HookInputBuilder) WithPattern(pattern core.CanonicalID) *HookInputBuilder {
	b.Pattern = pattern
	return b
}

// WithWorld sets world.
func (b *HookInputBuilder) WithWorld(world SemanticWorld) *HookInputBuilder {
	b.World = world
	return b
}

// WithUnit sets save unit.
func (b *HookInputBuilder) WithUnit(unit *SaveUnit) *HookInputBuilder {
	b.Unit = unit
	return b
}

// WithSnapshot sets snapshot.
func (b *HookInputBuilder) WithSnapshot(snapshot StoreSnapshot) *HookInputBuilder {
	b.Snapshot = snapshot
	return b
}

// Build returns HookInput.
func (b *HookInputBuilder) Build() HookInput {
	return HookInput{Application: b.Application, Pattern: b.Pattern, World: b.World, Unit: b.Unit, Snapshot: b.Snapshot}
}

// SaverOption customizes a Saver.
type SaverOption func(*Saver)

// WithStore sets the save store.
func WithStore(store Store) SaverOption {
	return func(s *Saver) { s.Store = store }
}

// WithUnitOfWork sets the transaction factory.
func WithUnitOfWork(uow UnitOfWorkTx) SaverOption {
	return func(s *Saver) { s.UnitOfWork = uow }
}

// WithMaxAttempts sets retry attempts.
func WithMaxAttempts(n int) SaverOption {
	return func(s *Saver) { s.MaxAttempts = n }
}

// WithBeforeHooks sets before-save hooks.
func WithBeforeHooks(hooks []Hook) SaverOption {
	return func(s *Saver) { s.BeforeHooks = hooks }
}

// WithAfterHooks sets after-save hooks.
func WithAfterHooks(hooks []Hook) SaverOption {
	return func(s *Saver) { s.AfterHooks = hooks }
}

// WithFailureHooks sets on-failure hooks.
func WithFailureHooks(hooks []Hook) SaverOption {
	return func(s *Saver) { s.FailureHooks = hooks }
}

// WithValidationHooks sets validation hooks.
func WithValidationHooks(hooks ...Hook) SaverOption {
	return func(s *Saver) { s.Validation = hooks }
}

// WithObserver sets the persistence observer.
func WithObserver(observer PersistenceObserver) SaverOption {
	return func(s *Saver) { s.Observer = observer }
}

// WithDurable sets durable observation.
func WithDurable(durable bool) SaverOption {
	return func(s *Saver) { s.Durable = durable }
}

// WithWorld sets the semantic world for ordering.
func WithWorld(world SemanticWorld) SaverOption {
	return func(s *Saver) { s.world = world }
}

// NewSaver creates a Saver from options.
func NewSaver(opts ...SaverOption) *Saver {
	s := &Saver{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
