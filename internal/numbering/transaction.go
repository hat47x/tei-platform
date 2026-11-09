package numbering

import (
	"context"
	"fmt"
)

// Transaction is the boundary a reservation joins. It is deliberately smaller
// than the Command boundary's UnitOfWork: the numbering package does not need to
// know what else the transaction covers, only that the counter change and the
// caller's change finish together.
type Transaction interface {
	// Commit finishes the transaction.
	Commit(ctx context.Context) error
	// Rollback undoes the transaction.
	Rollback(ctx context.Context) error
}

// UnitOfWork starts a transaction. An implementation may be shared with the
// Command boundary, so one transaction covers the record write and the counter.
type UnitOfWork interface {
	Begin(ctx context.Context) (Transaction, error)
}

// IssuedReservation is a reservation bound to a transaction. The caller finishes
// it with Commit or Rollback, so the counter and the caller's change are decided
// together.
//
// X-TEA read, counted up, and wrote the counter outside a caller-visible
// transaction (D:Session.java L1980–2009). A later failure could leave a gap, and
// a concurrent caller could lose an issuance without a word. Here the caller
// either commits both or returns the counter.
type IssuedReservation struct {
	reservation *Reservation
	transaction Transaction
}

// Number is the formatted number the reservation issued.
func (i *IssuedReservation) Number() string { return i.reservation.Number() }

// Commit commits the transaction, then marks the reservation finished. If the
// transaction commit fails, the counter is released and the error is returned, so
// a failed commit does not consume a number.
func (i *IssuedReservation) Commit(ctx context.Context) error {
	if err := i.transaction.Commit(ctx); err != nil {
		// The transaction did not commit, so the number is not used. Return the
		// counter and report the commit failure; the caller sees one error.
		if releaseErr := i.reservation.Release(ctx); releaseErr != nil {
			return fmt.Errorf("commit numbering transaction: %w (release also failed: %v)", err, releaseErr)
		}
		return fmt.Errorf("commit numbering transaction: %w", err)
	}
	return i.reservation.Commit()
}

// Rollback rolls the transaction back and returns the counter, in that order so
// a rollback failure leaves the number consumed rather than double-issuing it.
func (i *IssuedReservation) Rollback(ctx context.Context) error {
	if err := i.transaction.Rollback(ctx); err != nil {
		return fmt.Errorf("rollback numbering transaction: %w", err)
	}
	if err := i.reservation.Release(ctx); err != nil {
		return fmt.Errorf("release numbering reservation after rollback: %w", err)
	}
	return nil
}

// ReserveInTransaction issues a number and begins the transaction it belongs to.
//
// The returned reservation must be finished with Commit or Rollback. A begin that
// fails issues nothing; a reserve that fails rolls the transaction back, so a
// failed pair does not leak an open transaction.
func (c Contract) ReserveInTransaction(ctx context.Context, store CounterStore, unitOfWork UnitOfWork, ruleID string) (*IssuedReservation, error) {
	if unitOfWork == nil {
		return nil, fmt.Errorf("%w: no unit of work is configured", ErrNumbering)
	}
	transaction, err := unitOfWork.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin numbering transaction: %w", err)
	}
	_, reservation, err := c.Reserve(ctx, store, ruleID)
	if err != nil {
		_ = transaction.Rollback(ctx)
		return nil, err
	}
	return &IssuedReservation{reservation: reservation, transaction: transaction}, nil
}
