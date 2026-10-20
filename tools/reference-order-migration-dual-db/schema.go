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
);CREATE TABLE omdd.legacy_customer (
  customer_code text PRIMARY KEY,
  customer_name text NOT NULL,
  address text NOT NULL,
  credit_limit numeric(18,0) NOT NULL
);
-- Item code conversion prepared for the migration: legacy 6-digit codes to
-- new 10-digit codes. A legacy code with two rows is one-to-many.
CREATE TABLE omdd.legacy_item_code_map (
  legacy_code text NOT NULL,
  new_code text NOT NULL,
  PRIMARY KEY (legacy_code, new_code)
);
-- The legacy system's own trigger-based change log. The sample does not use
-- it as the change feed; it shows what a trigger-based capture would miss.
CREATE TABLE omdd.legacy_change_log (
  id bigserial PRIMARY KEY,
  table_name text NOT NULL,
  op text NOT NULL,
  row_key text NOT NULL,
  logged_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION omdd.legacy_log_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE r jsonb;
BEGIN
  IF TG_OP = 'DELETE' THEN r := to_jsonb(OLD); ELSE r := to_jsonb(NEW); END IF;
  INSERT INTO omdd.legacy_change_log(table_name, op, row_key)
    VALUES (TG_TABLE_NAME, TG_OP, coalesce(r->>'order_no', r->>'customer_code'));
  RETURN NULL;
END $$;
CREATE TRIGGER legacy_order_log AFTER INSERT OR UPDATE OR DELETE ON omdd.legacy_order FOR EACH ROW EXECUTE FUNCTION omdd.legacy_log_change();
CREATE TRIGGER legacy_order_line_log AFTER INSERT OR UPDATE OR DELETE ON omdd.legacy_order_line FOR EACH ROW EXECUTE FUNCTION omdd.legacy_log_change();
CREATE TRIGGER legacy_customer_log AFTER INSERT OR UPDATE OR DELETE ON omdd.legacy_customer FOR EACH ROW EXECUTE FUNCTION omdd.legacy_log_change();
-- Every reverse-sync transaction writes one marker row. The change feed
-- recognises the transaction by it and does not send it back (no echo).
CREATE TABLE omdd.legacy_sync_marker (
  id bigserial PRIMARY KEY,
  origin text NOT NULL,
  ref text NOT NULL,
  marked_at timestamptz NOT NULL DEFAULT now()
);
-- Reverse-sync receipt on the legacy side, deduplicated by outbox sequence.
CREATE TABLE omdd.legacy_reverse_applied (
  seq bigint PRIMARY KEY,
  plan_digest text NOT NULL,
  gate_digest text,
  entity text NOT NULL,
  entity_key text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now()
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
);
CREATE TABLE omdd.stage_customer (
  plan_digest text NOT NULL REFERENCES omdd.mig_snapshot(plan_digest),
  customer_code text NOT NULL,
  customer_name text NOT NULL,
  address text NOT NULL,
  credit_limit numeric(18,0) NOT NULL,
  PRIMARY KEY (plan_digest, customer_code)
);
-- The new customer master. family_name and given_name are added by the
-- expand phase of the name split (ExpandNameSplit), not here.
CREATE TABLE omdd.customer (
  customer_code text PRIMARY KEY,
  full_name text NOT NULL,
  address text NOT NULL,
  credit_limit numeric(18,0) NOT NULL
);
-- Last legacy values seen through the change feed or written by reverse
-- sync; a legacy change is a change against this base.
CREATE TABLE omdd.customer_legacy_base (
  customer_code text PRIMARY KEY,
  customer_name text NOT NULL,
  address text NOT NULL,
  credit_limit numeric(18,0) NOT NULL
);
CREATE TABLE omdd.master_field_owner (
  entity text NOT NULL,
  field text NOT NULL,
  owner text NOT NULL CHECK (owner IN ('legacy','primary')),
  gate_digest text,
  changed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (entity, field)
);
CREATE TABLE omdd.master_field_owner_history (
  id bigserial PRIMARY KEY,
  entity text NOT NULL,
  field text NOT NULL,
  owner text NOT NULL,
  reason text NOT NULL,
  changed_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.master_conflict (
  id bigserial PRIMARY KEY,
  plan_digest text NOT NULL,
  customer_code text NOT NULL,
  field text NOT NULL,
  owner text,
  legacy_value text,
  primary_value text,
  base_value text,
  commit_lsn pg_lsn,
  state text NOT NULL CHECK (state IN ('resolved','queued')),
  resolution text NOT NULL,
  detected_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz
);
CREATE TABLE omdd.mig_authority (
  entity text PRIMARY KEY,
  authority text NOT NULL CHECK (authority IN ('legacy','primary','rolling-back')),
  gate_digest text,
  changed_at timestamptz NOT NULL DEFAULT now()
);
-- Every decoded legacy transaction is recorded once, whatever was done
-- with it; the primary key refuses a second application.
CREATE TABLE omdd.cdc_tx (
  commit_lsn pg_lsn PRIMARY KEY,
  xid bigint NOT NULL,
  plan_digest text NOT NULL REFERENCES omdd.mig_plan(plan_digest),
  disposition text NOT NULL CHECK (disposition IN ('applied','skipped-visible-in-snapshot','echo-of-reverse-sync','held')),
  changes integer NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.cdc_held (
  id bigserial PRIMARY KEY,
  plan_digest text NOT NULL,
  commit_lsn pg_lsn, -- NULL when held by step 3 from the copy
  entity text NOT NULL,
  entity_key text NOT NULL,
  reason text NOT NULL,
  change jsonb NOT NULL,
  state text NOT NULL DEFAULT 'held' CHECK (state IN ('held','resolved')),
  resolution text,
  resolved_at timestamptz
);
CREATE TABLE omdd.code_decision (
  order_no text NOT NULL,
  line_no integer NOT NULL,
  legacy_code text NOT NULL,
  new_code text NOT NULL,
  decided_by text NOT NULL,
  decided_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (order_no, line_no)
);
CREATE TABLE omdd.reverse_outbox (
  seq bigserial PRIMARY KEY,
  plan_digest text NOT NULL,
  entity text NOT NULL CHECK (entity IN ('order','customer')),
  entity_key text NOT NULL,
  fields text[] NOT NULL DEFAULT '{}',
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','delivered')),
  created_at timestamptz NOT NULL DEFAULT now(),
  delivered_at timestamptz
);
CREATE TABLE omdd.billing_outbox (
  seq bigserial PRIMARY KEY,
  idempotency_key text NOT NULL UNIQUE,
  plan_digest text NOT NULL,
  customer_code text NOT NULL,
  payload jsonb NOT NULL,
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','unknown','delivered')),
  attempts integer NOT NULL DEFAULT 0,
  last_outcome text,
  last_error text,
  apply_id text,
  confirmed_by text,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.schema_change (
  change_id text PRIMARY KEY,
  phase text NOT NULL,
  plan_digest text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.schema_change_event (
  id bigserial PRIMARY KEY,
  change_id text NOT NULL,
  event text NOT NULL,
  detail jsonb NOT NULL,
  at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.schema_client (
  client_id text PRIMARY KEY,
  schema_version text NOT NULL,
  active boolean NOT NULL DEFAULT true,
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  retired_at timestamptz
);
CREATE TABLE omdd.schema_issue (
  id bigserial PRIMARY KEY,
  change_id text NOT NULL,
  entity_key text NOT NULL,
  reason text NOT NULL,
  value text,
  state text NOT NULL DEFAULT 'open' CHECK (state IN ('open','resolved')),
  at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz
);
CREATE TABLE omdd.cutover_gate (
  gate_digest text PRIMARY KEY,
  plan_digest text NOT NULL REFERENCES omdd.mig_plan(plan_digest),
  snapshot_digest text NOT NULL,
  passed boolean NOT NULL,
  result jsonb NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE omdd.rollback_drill (
  drill_digest text PRIMARY KEY,
  plan_digest text NOT NULL REFERENCES omdd.mig_plan(plan_digest),
  gate_digest text NOT NULL REFERENCES omdd.cutover_gate(gate_digest),
  passed boolean NOT NULL,
  result jsonb NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION omdd.field_owner(f text) RETURNS text LANGUAGE sql STABLE AS
  $$ SELECT owner FROM omdd.master_field_owner WHERE entity = 'customer' AND field = f $$;
-- Customer changes: every change of a billed field is queued for the
-- billing system (P-TX-02), whatever its origin. Changes of fields the
-- primary owns are queued for reverse sync, except those the change feed
-- itself applied (origin legacy-cdc), which would echo.
CREATE FUNCTION omdd.capture_customer() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  origin text := coalesce(current_setting('omdd.origin', true), '');
  pd text;
  f text[] := '{}';
  s bigint;
BEGIN
  SELECT plan_digest INTO pd FROM omdd.mig_plan;
  IF (NEW.full_name, NEW.address, NEW.credit_limit) IS DISTINCT FROM (OLD.full_name, OLD.address, OLD.credit_limit) THEN
    s := nextval(pg_get_serial_sequence('omdd.billing_outbox', 'seq'));
    INSERT INTO omdd.billing_outbox(seq, idempotency_key, plan_digest, customer_code, payload)
    VALUES (s, 'billing:' || NEW.customer_code || ':' || s, pd, NEW.customer_code,
      jsonb_build_object('customerCode', NEW.customer_code, 'billingName', NEW.full_name,
        'address', NEW.address, 'creditLimit', NEW.credit_limit::text));
  END IF;
  IF origin <> 'legacy-cdc' THEN
    IF NEW.full_name IS DISTINCT FROM OLD.full_name AND omdd.field_owner('name') = 'primary' THEN f := f || 'name'::text; END IF;
    IF NEW.address IS DISTINCT FROM OLD.address AND omdd.field_owner('address') = 'primary' THEN f := f || 'address'::text; END IF;
    IF NEW.credit_limit IS DISTINCT FROM OLD.credit_limit AND omdd.field_owner('credit_limit') = 'primary' THEN f := f || 'credit_limit'::text; END IF;
    IF cardinality(f) > 0 THEN
      INSERT INTO omdd.reverse_outbox(plan_digest, entity, entity_key, fields) VALUES (pd, 'customer', NEW.customer_code, f);
    END IF;
  END IF;
  RETURN NULL;
END $$;
CREATE TRIGGER customer_capture AFTER UPDATE ON omdd.customer FOR EACH ROW EXECUTE FUNCTION omdd.capture_customer();
-- Order changes are queued for reverse sync only while the primary is the
-- authority for orders (after the cutover gate).
CREATE FUNCTION omdd.capture_order() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  origin text := coalesce(current_setting('omdd.origin', true), '');
  k text;
BEGIN
  IF origin = 'legacy-cdc' THEN RETURN NULL; END IF;
  IF (SELECT authority FROM omdd.mig_authority WHERE entity = 'order') IS DISTINCT FROM 'primary' THEN RETURN NULL; END IF;
  IF TG_OP = 'DELETE' THEN k := OLD.order_no; ELSE k := NEW.order_no; END IF;
  -- Nested: PL/pgSQL does not short-circuit, and line rows have no
  -- customer_code.
  IF TG_TABLE_NAME = 'order_backlog' AND TG_OP = 'UPDATE' THEN
    IF (NEW.customer_code, NEW.ordered_on, NEW.status) IS NOT DISTINCT FROM (OLD.customer_code, OLD.ordered_on, OLD.status) THEN
      RETURN NULL;
    END IF;
  END IF;
  INSERT INTO omdd.reverse_outbox(plan_digest, entity, entity_key) SELECT plan_digest, 'order', k FROM omdd.mig_plan;
  RETURN NULL;
END $$;
CREATE TRIGGER order_capture AFTER INSERT OR UPDATE OR DELETE ON omdd.order_backlog FOR EACH ROW EXECUTE FUNCTION omdd.capture_order();
CREATE TRIGGER order_line_capture AFTER INSERT OR UPDATE OR DELETE ON omdd.order_backlog_line FOR EACH ROW EXECUTE FUNCTION omdd.capture_order();`

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
	if role == RoleLegacy {
		// A replication slot outlives the schema; drop the sample's slot so
		// the next copy pairs with a fresh one.
		if _, err := conn.Exec(ctx, `SELECT pg_drop_replication_slot(slot_name) FROM pg_replication_slots WHERE slot_name = $1 AND database = current_database()`, ChangeFeedSlot); err != nil {
			return fmt.Errorf("drop change feed slot: %w", err)
		}
	}
	ddl := map[Role]string{RoleLegacy: legacyDDL, RolePrimary: primaryDDL, RoleInventory: inventoryDDL}[role]
	_, err := conn.Exec(ctx, `DROP SCHEMA IF EXISTS omdd CASCADE; CREATE SCHEMA omdd;`+ddl)
	return err
}
