package command

import (
	"context"

	"tei/internal/semantic"
)

// PersistenceState separates three questions a Command result can answer:
//
//   - Is there a durable store at all? A reference execution has none.
//   - Did a durable Commit happen? That is what UnitOfWork.Commit reports.
//   - Was the declared effect seen again by an independent read after Commit?
//     Only that last answer is persistence observation.
//
// None of the three is implied by another. In particular, a successful Commit
// does not prove an independent read saw the effect, and an independent read is
// not a substitute for authorization or for the transaction itself.
type PersistenceState string

const (
	// PersistenceNotApplicable means the execution has no durable store to
	// observe (for example a reference run with MemoryUnitOfWork).
	PersistenceNotApplicable PersistenceState = "not-applicable"
	// PersistenceNotObserved means a durable effect was committed but no
	// independent read confirmed it. It is the default when an observer is not
	// configured. It must not be read as "not persisted": the Commit may have
	// succeeded.
	PersistenceNotObserved PersistenceState = "not-observed"
	// PersistenceObserved means an independent observer read the declared
	// effect back after Commit and reported success.
	PersistenceObserved PersistenceState = "observed"
)

// PersistenceObserver performs an independent read after a successful Commit
// and reports whether the declared effect is visible at the storage boundary.
//
// The observer receives only the request and the declared effect. It does not
// receive the executor's Result, so it cannot simply echo the executor's claim
// back as an observation. A nil error means the effect was observed. Any error
// means the observation did not establish persistence; it is recorded as
// not-observed, not as an execution failure, because the Commit already
// happened.
type PersistenceObserver interface {
	Observe(ctx context.Context, request Request, declared semantic.DataEffect) error
}

// observePersistence returns the persistence state for one successful
// execution. The caller must call it only after Commit has succeeded (or after
// an executor with no UnitOfWork completed).
func (i Invoker) observePersistence(ctx context.Context, request Request, declared *semantic.DataEffect) PersistenceState {
	if declared == nil || !i.DurableStore {
		return PersistenceNotApplicable
	}
	if i.PersistenceObserver == nil {
		return PersistenceNotObserved
	}
	if err := i.PersistenceObserver.Observe(ctx, request, *declared); err != nil {
		return PersistenceNotObserved
	}
	return PersistenceObserved
}
