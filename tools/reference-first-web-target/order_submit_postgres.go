//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"tei/tools/internal/postgresdb"
)

// orderSubmitStore is the opt-in seam through which the accepted Web Command
// changes real business state. When it is not injected the Command stays
// synthetic (an in-memory map) and its Observation is labelled synthetic.
//
// It is not an Authority: permission is decided before the Adapter calls it
// (business authorization and the Host effect Grant), and Source acceptance is
// not a write permission.
type orderSubmitStore interface {
	// Submit moves the order from draft to submitted. It fails, without
	// changing state, when no draft order with that key exists.
	Submit(ctx context.Context, orderID int64) error
	// ReadStatus reads the order status over a connection that did not perform
	// the write.
	ReadStatus(ctx context.Context, orderID int64) (string, error)
}

var errOrderNotSubmittable = errors.New("ORDER_NOT_SUBMITTABLE")

// postgresOrderSubmitStore writes public.orders (the physical table declared by
// databases/sales-core/persistence/order.tei) over one pool and reads over a
// second pool, so the read does not reuse the writer's session.
type postgresOrderSubmitStore struct {
	write *sql.DB
	read  *sql.DB
}

func newPostgresOrderSubmitStore(writeDSN, readDSN string) (*postgresOrderSubmitStore, error) {
	write, err := postgresdb.OpenSQL(writeDSN)
	if err != nil {
		return nil, err
	}
	read, err := postgresdb.OpenSQL(readDSN)
	if err != nil {
		_ = write.Close()
		return nil, err
	}
	return &postgresOrderSubmitStore{write: write, read: read}, nil
}

func (s *postgresOrderSubmitStore) Close() error {
	return errors.Join(s.write.Close(), s.read.Close())
}

func (s *postgresOrderSubmitStore) Submit(ctx context.Context, orderID int64) error {
	transaction, err := s.write.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin order submit: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	result, err := transaction.ExecContext(
		ctx,
		`UPDATE public.orders SET status = 'submitted' WHERE order_id = $1 AND status = 'draft'`,
		orderID,
	)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return errOrderNotSubmittable
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit order submit: %w", err)
	}
	return nil
}

func (s *postgresOrderSubmitStore) ReadStatus(ctx context.Context, orderID int64) (string, error) {
	var status string
	err := s.read.QueryRowContext(
		ctx,
		`SELECT status FROM public.orders WHERE order_id = $1`,
		orderID,
	).Scan(&status)
	if err != nil {
		// The Observation error reaches the HTTP client; keep the driver's
		// connection detail (user, host, port, database) out of it, as Submit does.
		return "", errors.New("read order status failed")
	}
	return status, nil
}
