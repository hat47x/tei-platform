// Package postgresdb contains the PostgreSQL connection setup shared by the
// reference evidence runners.
package postgresdb

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// OpenSQL opens a pgx-backed database/sql connection.
//
// The reference runners use multi-statement fixture scripts with ExecContext.
// simple_protocol preserves that behavior when moving from lib/pq, whose
// default execution path accepts those scripts, to pgx's database/sql adapter.
func OpenSQL(dsn string) (*sql.DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		// Parse failures can contain connection-string fragments. Do not expose
		// them through runner panic output or evidence logs.
		return nil, errors.New("invalid PostgreSQL connection configuration")
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	return stdlib.OpenDB(*config), nil
}
