// Package numberingcommand adapts the Command boundary's UnitOfWork to the
// numbering package's Transaction contract, so one transaction covers the record
// write and the counter.
//
// The two boundaries describe the same shape (Begin, then Commit or Rollback) but
// they are separate packages on purpose: the meaning boundary does not depend on
// the numbering provider, and the numbering package does not depend on Command.
// This package is the adapter, and it is the only place the two meet.
package numberingcommand

import (
	"context"

	"tei/internal/command"
	"tei/internal/numbering"
)

// UnitOfWork adapts a Command UnitOfWork to the numbering UnitOfWork.
type UnitOfWork struct {
	Inner command.UnitOfWork
}

// Begin starts the Command transaction and returns it as a numbering Transaction.
func (u UnitOfWork) Begin(ctx context.Context) (numbering.Transaction, error) {
	transaction, err := u.Inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return Transaction{Inner: transaction}, nil
}

// Transaction adapts a Command Transaction to the numbering Transaction.
type Transaction struct {
	Inner command.Transaction
}

// Commit commits the Command transaction.
func (t Transaction) Commit(ctx context.Context) error { return t.Inner.Commit(ctx) }

// Rollback rolls the Command transaction back.
func (t Transaction) Rollback(ctx context.Context) error { return t.Inner.Rollback(ctx) }
