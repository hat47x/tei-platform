package orderrollouthost

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ControlSchema is the control database of the supervising hosts: ownership
// of the lease supervisor, the resource lease authority, durable invocation
// handles, the call records, the application runtime registry and the run
// records. One sequence (event_seq) orders every event across the tables; it
// is the logical clock used for "after the command" decisions.
const ControlSchema = `
CREATE SCHEMA IF NOT EXISTS orhc;
CREATE SEQUENCE IF NOT EXISTS orhc.event_seq;
CREATE TABLE IF NOT EXISTS orhc.ownership_seq (lease_id text PRIMARY KEY, last_token bigint NOT NULL);
CREATE TABLE IF NOT EXISTS orhc.supervisor_ownership (
  lease_id text PRIMARY KEY, owner_id text NOT NULL, fencing_token bigint NOT NULL,
  heartbeat_seq bigint NOT NULL DEFAULT 0, claimed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  heartbeat_at timestamptz NOT NULL DEFAULT clock_timestamp(), owner_host_expires_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS orhc.ownership_event (
  seq bigint PRIMARY KEY DEFAULT nextval('orhc.event_seq'), lease_id text NOT NULL, actor text NOT NULL,
  op text NOT NULL, accepted boolean NOT NULL, fencing_token bigint, current_owner text, current_token bigint,
  host_time timestamptz, naive_host_clock_takeover boolean, message text, db_time timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE IF NOT EXISTS orhc.resource_lease (
  lease_id text PRIMARY KEY, revision int NOT NULL, fencing_token bigint NOT NULL DEFAULT 0, state text NOT NULL,
  request_fingerprint text NOT NULL, runtime_policy_fingerprint text NOT NULL, records jsonb NOT NULL,
  issued_at text NOT NULL, expires_at text NOT NULL, acquired_by text NOT NULL, released_by text);
CREATE TABLE IF NOT EXISTS orhc.lease_event (
  seq bigint PRIMARY KEY DEFAULT nextval('orhc.event_seq'), lease_id text NOT NULL, actor text NOT NULL,
  op text NOT NULL, accepted boolean NOT NULL, revision int, fencing_token bigint, current_token bigint,
  message text, db_time timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE IF NOT EXISTS orhc.invocation_handle (
  invocation_id text PRIMARY KEY, revision int NOT NULL, handle jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS orhc.handle_event (
  seq bigint PRIMARY KEY DEFAULT nextval('orhc.event_seq'), invocation_id text NOT NULL, revision int NOT NULL,
  handle jsonb NOT NULL, db_time timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE IF NOT EXISTS orhc.call_event (
  seq bigint PRIMARY KEY DEFAULT nextval('orhc.event_seq'), invocation_id text NOT NULL, actor text NOT NULL,
  kind text NOT NULL, detail jsonb NOT NULL DEFAULT '{}', db_time timestamptz NOT NULL DEFAULT clock_timestamp());
CREATE TABLE IF NOT EXISTS orhc.barrier (name text PRIMARY KEY, seq bigint NOT NULL DEFAULT nextval('orhc.event_seq'));
CREATE TABLE IF NOT EXISTS orhc.app_instance (
  instance_id text PRIMARY KEY, application text NOT NULL, version text NOT NULL, active boolean NOT NULL);
CREATE TABLE IF NOT EXISTS orhc.run_record (
  seq bigint PRIMARY KEY DEFAULT nextval('orhc.event_seq'), run_id text NOT NULL, revision int NOT NULL,
  status text NOT NULL, run jsonb NOT NULL, db_time timestamptz NOT NULL DEFAULT clock_timestamp());
`

// SalesSchema is the 4.0.0 state of the sales_core instance (the legacy
// order table of persistence sales_core.order, status still nullable) and
// the execution registry of the migration tool (tei_atlas), which plays the
// external runtime that host B observes and reattaches to.
const SalesSchema = `
CREATE TABLE IF NOT EXISTS public.orders (
  order_id bigint PRIMARY KEY, customer_id bigint NOT NULL, ordered_at timestamptz NOT NULL,
  status varchar(32), total_amount numeric(14,2) NOT NULL, external_id varchar(64));
CREATE SCHEMA IF NOT EXISTS tei_atlas;
CREATE TABLE IF NOT EXISTS tei_atlas.job (
  runtime_handle text PRIMARY KEY, action_id text NOT NULL, idempotency_key text NOT NULL UNIQUE,
  lifecycle text NOT NULL, runtime_token bigint NOT NULL, heartbeat_seq bigint NOT NULL DEFAULT 0,
  gate_open boolean NOT NULL DEFAULT false, os_pid int, accepted_at timestamptz, finished_at timestamptz,
  max_order_at_apply_start bigint, max_order_at_apply_end bigint, message text);
CREATE TABLE IF NOT EXISTS tei_atlas.revision (
  release text PRIMARY KEY, migration_id text NOT NULL, schema_fingerprint text NOT NULL,
  runtime_handle text NOT NULL, applied_at timestamptz NOT NULL DEFAULT clock_timestamp());
`

// ApplyControlSchema creates the control tables.
func ApplyControlSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, ControlSchema); err != nil {
		return fmt.Errorf("control schema: %w", err)
	}
	return nil
}

// ApplySalesSchema creates the 4.0.0 sales_core state and the tool registry.
func ApplySalesSchema(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, SalesSchema); err != nil {
		return fmt.Errorf("sales schema: %w", err)
	}
	return nil
}
