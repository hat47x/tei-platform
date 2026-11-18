package driver

import (
	"context"
	"database/sql"
	"strings"

	"tei/internal/ir"
)

// PostgresPhysicalReadAllowsComplete checks the PostgreSQL physical table in
// the same read-only transaction that ran a selected Contract Query. It is a
// conservative qualification shared by the product CLI and the first Web
// target: RLS, inheritance, views, ambiguous names and catalog uncertainty
// prevent a claim that the entire relation was read.
func PostgresPhysicalReadAllowsComplete(
	ctx context.Context,
	tx *sql.Tx,
	world ir.World,
	resolution ResolvedHeaderDetailContractQuery,
) (bool, string) {
	var physicalTable string
	matches := 0
	for _, mapping := range world.Persistence {
		if mapping.Database != resolution.Binding.Target.Database ||
			mapping.SemanticData != resolution.Relation.Target.ID {
			continue
		}
		matches++
		if matches != 1 {
			return false, "the PostgreSQL persistence table is ambiguous"
		}
		physicalTable = mapping.Table
	}
	if matches != 1 || physicalTable == "" {
		return false, "the PostgreSQL persistence table was not identified"
	}
	// A name resolved through search_path can refer to a different table
	// than the one the SQL execution actually read. Require an explicit
	// schema and compare names via pg_namespace instead of to_regclass.
	schema, name, qualified := strings.Cut(physicalTable, ".")
	if !qualified || schema == "" || name == "" || strings.Contains(name, ".") {
		return false, "the PostgreSQL persistence table needs an unambiguous schema-qualified name"
	}
	var kind string
	var rlsEnabled, rlsForced, hasChildren bool
	err := tx.QueryRowContext(ctx, `SELECT c.relkind::text, c.relrowsecurity,
		c.relforcerowsecurity, c.relhassubclass
		FROM pg_catalog.pg_class AS c
		JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2`, schema, name).
		Scan(&kind, &rlsEnabled, &rlsForced, &hasChildren)
	if err != nil {
		return false, "the physical PostgreSQL table could not be verified"
	}
	if kind != "r" || hasChildren {
		return false, "the physical PostgreSQL source is not a plain table without inherited rows"
	}
	if rlsEnabled || rlsForced {
		return false, "the PostgreSQL table enables row-level security and may hide relation rows"
	}
	return true, ""
}
