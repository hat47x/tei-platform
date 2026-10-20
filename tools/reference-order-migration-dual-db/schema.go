package ordermigrationdualdb

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Role is one of the three databases of the sample.
type Role string

const (
	RoleLegacy    Role = "legacy"
	RolePrimary   Role = "primary"
	RoleInventory Role = "inventory"
)

// DatabaseName is the exact database name each role must connect to. The
// sample refuses any other database so it cannot write into a shared one.
func (r Role) DatabaseName() string { return "tei_omdd_" + string(r) }

// DSNEnv is the environment variable that carries the DSN for a role.
func (r Role) DSNEnv() string {
	switch r {
	case RoleLegacy:
		return "TEI_OMDD_LEGACY_DSN"
	case RolePrimary:
		return "TEI_OMDD_PRIMARY_DSN"
	default:
		return "TEI_OMDD_INVENTORY_DSN"
	}
}

// Connect opens a connection and checks that it reached the dedicated
// database for the role.
func Connect(ctx context.Context, dsn string, role Role, timeout time.Duration) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: parse DSN: %w", role, err)
	}
	if timeout > 0 {
		cfg.ConnectTimeout = timeout
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: connect: %w", role, err)
	}
	var name string
	if err := conn.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		conn.Close(context.Background())
		return nil, fmt.Errorf("%s: current_database: %w", role, err)
	}
	if name != role.DatabaseName() {
		conn.Close(context.Background())
		return nil, fmt.Errorf("%s: connected to database %q, want dedicated %q", role, name, role.DatabaseName())
	}
	return conn, nil
}

const legacyDDL = `
CREATE TABLE omdd.legacy_order (
  order_no text PRIMARY KEY,
  customer_code text NOT NULL,
  ordered_on date NOT NULL,
  status text NOT NULL
);
CREATE TABLE omdd.legacy_order_line (
  order_no text NOT NULL REFERENCES omdd.legacy_order(order_no),
  line_no integer NOT NULL,
  item_code text NOT NULL,
  qty bigint NOT NULL CHECK (qty > 0),
  unit_price numeric(18,0) NOT NULL,
  amount numeric(18,0) NOT NULL,
  PRIMARY KEY (order_no, line_no)
);`

const primaryDDL = `
CREATE TABLE omdd.mig_plan (
  plan_digest text PRIMARY KEY,
  plan_json jsonb NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);
-- One primary database hosts one migration: a second plan is refused.
CREATE UNIQUE INDEX mig_plan_single ON omdd.mig_plan ((true));
CREATE TABLE omdd.mig_snapshot (
  plan_digest text PRIMARY KEY REFERENCES omdd.mig_plan(plan_digest),
  snapshot_digest text NOT NULL UNIQUE,
  record_json jsonb NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.stage_order (
  plan_digest text NOT NULL REFERENCES omdd.mig_snapshot(plan_digest),
  order_no text NOT NULL,
  customer_code text NOT NULL,
  ordered_on date NOT NULL,
  status text NOT NULL,
  PRIMARY KEY (plan_digest, order_no)
);
CREATE TABLE omdd.stage_order_line (
  plan_digest text NOT NULL,
  order_no text NOT NULL,
  line_no integer NOT NULL,
  item_code text NOT NULL,
  qty bigint NOT NULL,
  unit_price numeric(18,0) NOT NULL,
  amount numeric(18,0) NOT NULL,
  PRIMARY KEY (plan_digest, order_no, line_no),
  FOREIGN KEY (plan_digest, order_no) REFERENCES omdd.stage_order(plan_digest, order_no)
);
CREATE TABLE omdd.order_backlog (
  order_no text PRIMARY KEY,
  customer_code text NOT NULL,
  ordered_on date NOT NULL,
  status text NOT NULL,
  plan_digest text NOT NULL REFERENCES omdd.mig_plan(plan_digest),
  chunk_id text NOT NULL,
  allocation_state text NOT NULL CHECK (allocation_state IN ('awaiting','allocated','rejected','compensated'))
);
CREATE TABLE omdd.order_backlog_line (
  order_no text NOT NULL REFERENCES omdd.order_backlog(order_no),
  line_no integer NOT NULL,
  item_code text NOT NULL,
  qty bigint NOT NULL,
  unit_price numeric(18,0) NOT NULL,
  amount numeric(18,0) NOT NULL,
  PRIMARY KEY (order_no, line_no)
);
CREATE TABLE omdd.mig_checkpoint (
  plan_digest text NOT NULL REFERENCES omdd.mig_plan(plan_digest),
  chunk_id text NOT NULL,
  snapshot_digest text NOT NULL,
  order_count integer NOT NULL,
  line_count integer NOT NULL,
  amount_sum numeric NOT NULL,
  committed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (plan_digest, chunk_id)
);
CREATE TABLE omdd.effect_outbox (
  effect_id text PRIMARY KEY,
  operation_id text NOT NULL,
  plan_digest text NOT NULL,
  order_no text NOT NULL UNIQUE REFERENCES omdd.order_backlog(order_no),
  kind text NOT NULL,
  destination text NOT NULL,
  payload jsonb NOT NULL,
  state text NOT NULL CHECK (state IN ('pending','acknowledged','rejected')),
  attempts integer NOT NULL DEFAULT 0,
  last_outcome text,
  last_error text,
  last_attempt_at timestamptz,
  acknowledged_at timestamptz
);
CREATE TABLE omdd.compensation (
  compensation_id text PRIMARY KEY,
  compensates_operation_id text NOT NULL UNIQUE,
  effect_id text NOT NULL UNIQUE REFERENCES omdd.effect_outbox(effect_id),
  order_no text NOT NULL,
  reason text NOT NULL,
  committed_at timestamptz NOT NULL DEFAULT now()
);`

const inventoryDDL = `
CREATE TABLE omdd.stock (
  item_code text PRIMARY KEY,
  on_hand bigint NOT NULL CHECK (on_hand >= 0),
  reserved bigint NOT NULL DEFAULT 0 CHECK (reserved >= 0 AND reserved <= on_hand)
);
CREATE TABLE omdd.applied_effect (
  effect_id text PRIMARY KEY,
  order_no text NOT NULL,
  outcome text NOT NULL CHECK (outcome IN ('reserved','rejected')),
  reason text,
  applied_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.allocation (
  effect_id text NOT NULL REFERENCES omdd.applied_effect(effect_id),
  line_no integer NOT NULL,
  order_no text NOT NULL,
  item_code text NOT NULL REFERENCES omdd.stock(item_code),
  qty bigint NOT NULL CHECK (qty > 0),
  PRIMARY KEY (effect_id, line_no)
);`

// ResetSchema drops and recreates the sample schema in the dedicated
// database of the role. Connect has already refused any other database.
func ResetSchema(ctx context.Context, conn *pgx.Conn, role Role) error {
	var name string
	if err := conn.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if name != role.DatabaseName() {
		return fmt.Errorf("refuse to reset schema in %q", name)
	}
	ddl := map[Role]string{RoleLegacy: legacyDDL, RolePrimary: primaryDDL, RoleInventory: inventoryDDL}[role]
	_, err := conn.Exec(ctx, `DROP SCHEMA IF EXISTS omdd CASCADE; CREATE SCHEMA omdd;`+ddl)
	return err
}
