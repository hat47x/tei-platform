// Package tablelifecycle is an executable, synthetic reference for the
// table-side update/transaction/consequence boundary of issue #1129.
// It is not a TEI Canonical model or an implementation of legacy X-TEA scripts.
package tablelifecycle

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Phase is a trace stage, not a script API or a business-model declaration.
type Phase string

const (
	PhaseValidate    Phase = "validate"
	PhaseAuthorize   Phase = "authorize"
	PhaseRead        Phase = "read-current"
	PhaseBefore      Phase = "before-write"
	PhaseConsequence Phase = "transaction-consequence"
	PhaseCommit      Phase = "commit"
	PhasePostCommit  Phase = "post-commit-outbox"
)

const (
	Submitted    = "submitted"
	NotSubmitted = "not_submitted"
	Deleted      = "deleted"
	NotDeleted   = "not_deleted"
	Denied       = "denied"
	Unknown      = "unknown"
)

// Submit requests an atomic conditional update. OperationID is a durable
// idempotency identity within this synthetic store; it is not a user session.
type Submit struct {
	OperationID     string
	SubjectID       string
	OrderID         string
	ExpectedVersion uint64
	Units           int
}

// Cancel requests a distinct version-checked business cancellation. It is not
// the same as cancellation of the Go request context.
type Cancel struct {
	OperationID     string
	SubjectID       string
	OrderID         string
	ExpectedVersion uint64
}

// DeletePendingRequest identifies a version-checked deletion of a pending
// order. It never represents the legacy X-TEA delete(checkOnly) behavior.
type DeletePendingRequest struct {
	OperationID     string
	SubjectID       string
	OrderID         string
	ExpectedVersion uint64
}

// Authorizer represents an existing authorization capability at the boundary;
// this reference never reimplements its policy or asserts a commit-time fence.
type Authorizer interface {
	Allow(context.Context, string, string) (bool, error)
}

type AuthorizeFunc func(context.Context, string, string) (bool, error)

func (f AuthorizeFunc) Allow(ctx context.Context, subject, order string) (bool, error) {
	return f(ctx, subject, order)
}

// Result distinguishes a known business outcome, a refusal, and a lost
// acknowledgement; Unknown does not imply that the write was rolled back.
type Result struct {
	Outcome   string
	Committed bool
	Trace     []Phase
}

type Order struct {
	ID      string
	State   string
	Version uint64
	Units   int
}

type Notice struct {
	OperationID string
	OrderID     string
	Kind        string
}

type Snapshot struct {
	Order     Order
	Available int
	Reserved  int
	History   []string
	Notices   []Notice
}

type receipt struct {
	Request any // typed Submit or Cancel; both are comparable structs
	Result  Result
}

type state struct {
	orders       map[string]Order
	available    int
	reservations map[string]int
	history      map[string][]string
	receipts     map[string]receipt
	pending      map[string]Notice
	delivered    map[string]bool
}

func (s state) copy() state {
	c := state{
		orders: make(map[string]Order, len(s.orders)), available: s.available,
		reservations: make(map[string]int, len(s.reservations)),
		history:      make(map[string][]string, len(s.history)),
		receipts:     make(map[string]receipt, len(s.receipts)),
		pending:      make(map[string]Notice, len(s.pending)),
		delivered:    make(map[string]bool, len(s.delivered)),
	}
	for k, v := range s.orders {
		c.orders[k] = v
	}
	for k, v := range s.reservations {
		c.reservations[k] = v
	}
	for k, v := range s.history {
		c.history[k] = append([]string(nil), v...)
	}
	for k, v := range s.receipts {
		c.receipts[k] = v
	}
	for k, v := range s.pending {
		c.pending[k] = v
	}
	for k, v := range s.delivered {
		c.delivered[k] = v
	}
	return c
}

// Store is a single-process deterministic transactional fixture. Its mutex
// makes the staged changes, resource check and outbox insertion one boundary.
// Cross-process persistence, provider authority and distributed atomicity are
// explicitly outside this reference.
type Store struct {
	mu                  sync.Mutex
	state               state
	failBeforeCommit    bool            // synthetic fault injection; tests in this package only
	loseAcknowledgement bool            // synthetic transport loss after a successful commit
	delivering          map[string]bool // single-process delivery lease
}

func NewStore(available int, orders ...Order) (*Store, error) {
	if available < 0 {
		return nil, errors.New("negative inventory")
	}
	s := state{orders: map[string]Order{}, available: available, reservations: map[string]int{}, history: map[string][]string{}, receipts: map[string]receipt{}, pending: map[string]Notice{}, delivered: map[string]bool{}}
	for _, o := range orders {
		if o.ID == "" || o.Units <= 0 || o.State == "" {
			return nil, errors.New("invalid initial order")
		}
		if _, exists := s.orders[o.ID]; exists {
			return nil, errors.New("duplicate initial order")
		}
		s.orders[o.ID] = o
	}
	return &Store{state: s, delivering: map[string]bool{}}, nil
}

func validate(c Submit) error {
	if c.OperationID == "" || c.SubjectID == "" || c.OrderID == "" || c.Units <= 0 {
		return errors.New("invalid submit identity or units")
	}
	return nil
}

func cloneResult(r Result) Result {
	r.Trace = append([]Phase(nil), r.Trace...)
	return r
}

// Execute first validates the request and consults the provided authorization
// capability. It then performs the expected-version/state check *under the
// same mutex as the staged transaction commit*. Related reservation and history
// updates are invoked through the owner-side action below, not through a raw
// externally supplied script callback.
func (s *Store) Execute(ctx context.Context, c Submit, auth Authorizer) (Result, error) {
	trace := []Phase{PhaseValidate}
	if err := validate(c); err != nil {
		return Result{}, err
	}
	if auth == nil {
		return Result{}, errors.New("authorization capability missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseAuthorize)
	allowed, err := auth.Allow(ctx, c.SubjectID, c.OrderID)
	if err != nil {
		return Result{}, fmt.Errorf("authorization unavailable: %w", err)
	}
	if !allowed {
		return Result{Outcome: Denied, Trace: trace}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseRead)
	if r, ok := s.state.receipts[c.OperationID]; ok {
		if r.Request != c {
			return Result{}, errors.New("idempotency identity conflicts with a different request")
		}
		return cloneResult(r.Result), nil
	}
	current, ok := s.state.orders[c.OrderID]
	if !ok || current.Version != c.ExpectedVersion || current.State != "pending" || current.Units != c.Units || s.state.available < c.Units {
		return Result{Outcome: NotSubmitted, Trace: trace}, nil
	}
	trace = append(trace, PhaseBefore)
	// All mutations apply to a staged copy. Failure/cancel discards the copy.
	staged := s.state.copy()
	order := staged.orders[c.OrderID]
	order.State = Submitted
	order.Version++
	staged.orders[c.OrderID] = order
	trace = append(trace, PhaseConsequence)
	// The inventory owner action updates reservation and history together.
	if err := applyReservation(&staged, c); err != nil {
		return Result{}, err
	}
	if s.failBeforeCommit {
		return Result{}, errors.New("injected pre-commit transaction failure")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseCommit, PhasePostCommit)
	confirmed := Result{Outcome: Submitted, Committed: true, Trace: append([]Phase(nil), trace...)}
	staged.receipts[c.OperationID] = receipt{Request: c, Result: confirmed}
	staged.pending[c.OperationID] = Notice{OperationID: c.OperationID, OrderID: c.OrderID, Kind: "order.submitted"}
	s.state = staged // atomic linearization point for synthetic provider
	if s.loseAcknowledgement {
		return Result{Outcome: Unknown, Committed: false, Trace: append([]Phase(nil), trace...)}, nil
	}
	return confirmed, nil
}

// applyReservation is deliberately owner-side; external rules may not mutate
// other entities by writing arbitrary fields into the original order.
func applyReservation(staged *state, c Submit) error {
	if staged.reservations[c.OrderID] != 0 {
		return errors.New("reservation already exists")
	}
	if staged.available < c.Units {
		return errors.New("insufficient inventory at consequence")
	}
	staged.available -= c.Units
	staged.reservations[c.OrderID] = c.Units
	staged.history[c.OrderID] = append(staged.history[c.OrderID], "submitted:"+c.OperationID)
	return nil
}

// Lookup reauthorizes *before* disclosing a commit receipt. Not found is not a
// proof that an external side effect has not happened.
func (s *Store) Lookup(ctx context.Context, c Submit, auth Authorizer) (Result, bool, error) {
	if err := validate(c); err != nil {
		return Result{}, false, err
	}
	if auth == nil {
		return Result{}, false, errors.New("authorization capability missing")
	}
	allowed, err := auth.Allow(ctx, c.SubjectID, c.OrderID)
	if err != nil {
		return Result{}, false, err
	}
	if !allowed {
		return Result{Outcome: Denied}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.state.receipts[c.OperationID]
	if !ok {
		return Result{}, false, nil
	}
	if r.Request != c {
		return Result{}, false, errors.New("receipt identity conflicts with request")
	}
	return cloneResult(r.Result), true, nil
}

// LookupCancel reauthorizes before revealing a cancellation receipt. It is
// the recovery path after ExecuteCancel returned Unknown (lost acknowledgement).
// A missing receipt does not establish that an external side effect is absent.
// As with Lookup, this is only the synthetic Store's authorization boundary,
// not an atomic commit-time authorization fence.
func (s *Store) LookupCancel(ctx context.Context, c Cancel, auth Authorizer) (Result, bool, error) {
	if c.OperationID == "" || c.SubjectID == "" || c.OrderID == "" {
		return Result{}, false, errors.New("invalid cancel identity")
	}
	if auth == nil {
		return Result{}, false, errors.New("authorization capability missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, false, err
	}
	allowed, err := auth.Allow(ctx, c.SubjectID, c.OrderID)
	if err != nil {
		return Result{}, false, fmt.Errorf("authorization unavailable: %w", err)
	}
	if !allowed {
		return Result{Outcome: Denied}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, false, err
	}
	r, found := s.state.receipts[c.OperationID]
	if !found {
		return Result{}, false, nil
	}
	if r.Request != c {
		return Result{}, false, errors.New("receipt identity conflicts with request")
	}
	return cloneResult(r.Result), true, nil
}

// SendPending uses the same idempotency identity as the commit receipt. The
// sender must implement receiver-side deduplication. A returned error may mean
// the remote service received the notice; the outbox therefore retains it.
// Only one sender per operation runs at a time (not a distributed lease).
func (s *Store) SendPending(ctx context.Context, operationID string, sender func(context.Context, Notice) error) (bool, error) {
	if operationID == "" || sender == nil {
		return false, errors.New("invalid delivery request")
	}
	s.mu.Lock()
	n, ok := s.state.pending[operationID]
	if !ok || s.state.delivered[operationID] {
		s.mu.Unlock()
		return false, nil
	}
	if s.delivering[operationID] {
		s.mu.Unlock()
		return false, errors.New("delivery already in progress")
	}
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		return false, err
	}
	s.delivering[operationID] = true
	s.mu.Unlock()
	// No arbitrary external callback runs under the transaction mutex.
	err := sender(ctx, n)
	s.mu.Lock()
	delete(s.delivering, operationID)
	if err == nil {
		s.state.delivered[operationID] = true
	}
	s.mu.Unlock()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Cancel is a separate, version-checked business action. Cancellation of
// context.Context is *not* the same as this compensating business transition.
// ExecuteCancel changes a submitted order to cancelled, deletes its reservation
// and refunds inventory in the same staged transaction.
func (s *Store) ExecuteCancel(ctx context.Context, c Cancel, auth Authorizer) (Result, error) {
	trace := []Phase{PhaseValidate}
	if c.OperationID == "" || c.SubjectID == "" || c.OrderID == "" {
		return Result{}, errors.New("invalid cancel identity")
	}
	if auth == nil {
		return Result{}, errors.New("authorization capability missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseAuthorize)
	allowed, err := auth.Allow(ctx, c.SubjectID, c.OrderID)
	if err != nil {
		return Result{}, fmt.Errorf("authorization unavailable: %w", err)
	}
	if !allowed {
		return Result{Outcome: Denied, Trace: trace}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseRead)
	if r, ok := s.state.receipts[c.OperationID]; ok {
		if r.Request != c {
			return Result{}, errors.New("idempotency identity conflicts with a different request")
		}
		return cloneResult(r.Result), nil
	}
	current, ok := s.state.orders[c.OrderID]
	units := s.state.reservations[c.OrderID]
	if !ok || current.State != Submitted || current.Version != c.ExpectedVersion || units <= 0 || units != current.Units {
		return Result{Outcome: NotSubmitted, Trace: trace}, nil
	}
	trace = append(trace, PhaseBefore)
	staged := s.state.copy()
	order := staged.orders[c.OrderID]
	order.State = "cancelled"
	order.Version++
	staged.orders[c.OrderID] = order
	trace = append(trace, PhaseConsequence)
	staged.available += units
	delete(staged.reservations, c.OrderID)
	staged.history[c.OrderID] = append(staged.history[c.OrderID], "cancelled:"+c.OperationID)
	if s.failBeforeCommit {
		return Result{}, errors.New("injected pre-commit transaction failure")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseCommit, PhasePostCommit)
	result := Result{Outcome: "cancelled", Committed: true, Trace: append([]Phase(nil), trace...)}
	staged.receipts[c.OperationID] = receipt{Request: c, Result: result}
	staged.pending[c.OperationID] = Notice{OperationID: c.OperationID, OrderID: c.OrderID, Kind: "order.cancelled"}
	s.state = staged
	if s.loseAcknowledgement {
		return Result{Outcome: Unknown, Trace: trace}, nil
	}
	return result, nil
}

// DeletePending removes only an unsubmitted order with no existing reservation.
// The old-version/owner check, tombstone history, receipt, and outbox are
// atomically staged in the same synthetic Store boundary. This method does not
// implement or infer legacy X-TEA's checkOnly semantics.
func (s *Store) DeletePending(ctx context.Context, c DeletePendingRequest, auth Authorizer) (Result, error) {
	trace := []Phase{PhaseValidate}
	if c.OperationID == "" || c.SubjectID == "" || c.OrderID == "" {
		return Result{}, errors.New("invalid delete identity")
	}
	if auth == nil {
		return Result{}, errors.New("authorization capability missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseAuthorize)
	allowed, err := auth.Allow(ctx, c.SubjectID, c.OrderID)
	if err != nil {
		return Result{}, fmt.Errorf("authorization unavailable: %w", err)
	}
	if !allowed {
		return Result{Outcome: Denied, Trace: trace}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseRead)
	if r, found := s.state.receipts[c.OperationID]; found {
		if r.Request != c {
			return Result{}, errors.New("idempotency identity conflicts with a different request")
		}
		return cloneResult(r.Result), nil
	}
	current, present := s.state.orders[c.OrderID]
	_, hasReservation := s.state.reservations[c.OrderID]
	if !present || current.State != "pending" || current.Version != c.ExpectedVersion || hasReservation {
		return Result{Outcome: NotDeleted, Trace: trace}, nil
	}

	trace = append(trace, PhaseBefore)
	staged := s.state.copy()
	delete(staged.orders, c.OrderID)
	trace = append(trace, PhaseConsequence)
	// This history is an owner-side tombstone, not an externally triggered
	// arbitrary after-delete Script; neither inventory nor committed fact
	// changes when a pending, unreserved order is deleted.
	staged.history[c.OrderID] = append(staged.history[c.OrderID], "deleted:"+c.OperationID)
	if s.failBeforeCommit {
		return Result{}, errors.New("injected pre-commit transaction failure")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	trace = append(trace, PhaseCommit, PhasePostCommit)
	committed := Result{Outcome: Deleted, Committed: true, Trace: append([]Phase(nil), trace...)}
	staged.receipts[c.OperationID] = receipt{Request: c, Result: committed}
	staged.pending[c.OperationID] = Notice{OperationID: c.OperationID, OrderID: c.OrderID, Kind: "order.deleted"}
	s.state = staged
	if s.loseAcknowledgement {
		return Result{Outcome: Unknown, Trace: append([]Phase(nil), trace...)}, nil
	}
	return committed, nil
}

// LookupDelete reauthorizes before disclosing a tombstone receipt after an
// acknowledgement loss. A missing receipt is not evidence about remote effects.
func (s *Store) LookupDelete(ctx context.Context, c DeletePendingRequest, auth Authorizer) (Result, bool, error) {
	if c.OperationID == "" || c.SubjectID == "" || c.OrderID == "" {
		return Result{}, false, errors.New("invalid delete identity")
	}
	if auth == nil {
		return Result{}, false, errors.New("authorization capability missing")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, false, err
	}
	allowed, err := auth.Allow(ctx, c.SubjectID, c.OrderID)
	if err != nil {
		return Result{}, false, fmt.Errorf("authorization unavailable: %w", err)
	}
	if !allowed {
		return Result{Outcome: Denied}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Result{}, false, err
	}
	r, found := s.state.receipts[c.OperationID]
	if !found {
		return Result{}, false, nil
	}
	if r.Request != c {
		return Result{}, false, errors.New("receipt identity conflicts with request")
	}
	return cloneResult(r.Result), true, nil
}

// Inspect is a synthetic test-only observer, not an authorized public API.
func (s *Store) Inspect(orderID string) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.state.orders[orderID]
	if !ok {
		return Snapshot{}, false
	}
	snap := Snapshot{Order: o, Available: s.state.available, Reserved: s.state.reservations[orderID], History: append([]string(nil), s.state.history[orderID]...)}
	for _, n := range s.state.pending {
		if n.OrderID == orderID {
			snap.Notices = append(snap.Notices, n)
		}
	}
	return snap, true
}
