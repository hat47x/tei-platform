package postgresquery

import (
	"context"
	"fmt"

	"tei/internal/functioncontext"
)

// ObserveConnection reads what the connection handed to the Executor looks like, using the
// same Queryer (normally the same transaction) as the read. The result is an observation
// of the connection, not a verification that it is the declared Database instance.
//
// pg_current_snapshot() is only a stable identifier inside a REPEATABLE READ or stricter
// transaction, which is why the host must pass a transaction.
func ObserveConnection(ctx context.Context, db Queryer) (functioncontext.ConnectionObservation, error) {
	if db == nil {
		return functioncontext.ConnectionObservation{}, fmt.Errorf("PostgreSQL connection observation needs a connection")
	}
	rows, err := db.QueryContext(ctx, `SELECT current_database(), current_user, version(),
		current_setting('transaction_read_only'), pg_current_snapshot()::text, now()`)
	if err != nil {
		return functioncontext.ConnectionObservation{}, fmt.Errorf("cannot observe the PostgreSQL connection: %w", err)
	}
	defer rows.Close()
	var (
		o        functioncontext.ConnectionObservation
		readOnly string
	)
	if !rows.Next() {
		return o, fmt.Errorf("PostgreSQL connection observation returned no row")
	}
	if err := rows.Scan(&o.Database, &o.User, &o.ServerVersion, &readOnly, &o.Snapshot, &o.ObservedAt); err != nil {
		return functioncontext.ConnectionObservation{}, fmt.Errorf("cannot read the PostgreSQL connection observation: %w", err)
	}
	if err := rows.Err(); err != nil {
		return functioncontext.ConnectionObservation{}, err
	}
	o.ReadOnly = readOnly == "on"
	return o, nil
}
