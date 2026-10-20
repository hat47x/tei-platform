package monthendclose

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DatabaseName is the only database the sample connects to. Any other name is
// refused so the sample cannot write into a shared database.
const DatabaseName = "tei_mecc"

// DSNEnv carries the DSN of the dedicated database.
const DSNEnv = "TEI_MECC_DSN"

// Connect opens a connection and checks that it reached the dedicated
// database.
func Connect(ctx context.Context, dsn string, timeout time.Duration) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}
	if timeout > 0 {
		cfg.ConnectTimeout = timeout
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	var name string
	if err := conn.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		conn.Close(context.Background())
		return nil, err
	}
	if name != DatabaseName {
		conn.Close(context.Background())
		return nil, fmt.Errorf("connected to database %q, want dedicated %q", name, DatabaseName)
	}
	return conn, nil
}

// FenceSQLState is raised by the storage-side fencing trigger.
const FenceSQLState = "TF001"

const ddl = `
CREATE TABLE mecc.calendar_version (
  calendar_id text NOT NULL,
  version integer NOT NULL,
  body jsonb NOT NULL,
  digest text NOT NULL,
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY (calendar_id, version)
);
CREATE TABLE mecc.payable (
  payable_id text PRIMARY KEY,
  nominal_due date NOT NULL
);
CREATE TABLE mecc.close_decision (
  digest text PRIMARY KEY,
  target_month text NOT NULL,
  calendar_id text NOT NULL,
  calendar_version integer NOT NULL,
  body jsonb NOT NULL,
  decided_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.close_month (
  target_month text PRIMARY KEY,
  active_digest text NOT NULL REFERENCES mecc.close_decision(digest),
  calendar_version integer NOT NULL,
  activated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.decision_activation (
  id bigserial PRIMARY KEY,
  target_month text NOT NULL,
  digest text NOT NULL,
  calendar_version integer NOT NULL,
  previous_digest text,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.calendar_impact (
  id bigserial PRIMARY KEY,
  target_month text NOT NULL,
  from_digest text NOT NULL,
  to_digest text NOT NULL,
  from_version integer NOT NULL,
  to_version integer NOT NULL,
  body jsonb NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Step 2: import units, records, holds, rejections.
CREATE TABLE mecc.import_sender (sender text PRIMARY KEY);
CREATE TABLE mecc.import_unit (
  import_id bigserial PRIMARY KEY,
  sender text NOT NULL,
  file_seq integer NOT NULL,
  file_sha256 text NOT NULL UNIQUE,
  file_size integer NOT NULL,
  spec_id text NOT NULL,
  spec_digest text NOT NULL,
  received_at timestamptz NOT NULL,
  target_month text NOT NULL,
  decision_digest text NOT NULL REFERENCES mecc.close_decision(digest),
  calendar_version integer NOT NULL,
  record_count integer NOT NULL,
  amount_sum bigint NOT NULL,
  held_count integer NOT NULL,
  applied_count integer NOT NULL,
  imported_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  UNIQUE (sender, file_seq)
);
CREATE TABLE mecc.import_record (
  import_id bigint NOT NULL REFERENCES mecc.import_unit(import_id),
  seq integer NOT NULL,
  kind text NOT NULL,
  vals jsonb NOT NULL,
  raw bytea NOT NULL,
  held_reason text,
  held_field text,
  PRIMARY KEY (import_id, seq)
);
CREATE TABLE mecc.import_rejection (
  id bigserial PRIMARY KEY,
  file_sha256 text NOT NULL,
  file_size integer NOT NULL,
  reason text NOT NULL,
  detail text NOT NULL,
  decision_digest text,
  received_at timestamptz,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Shipment events received by file and by message.
CREATE TABLE mecc.shipment_event (
  shipment_id text PRIMARY KEY,
  order_no text NOT NULL,
  ship_date date NOT NULL,
  qty bigint NOT NULL,
  amount numeric(18,0) NOT NULL,
  item_name text NOT NULL,
  content_digest text NOT NULL,
  first_source text NOT NULL,
  first_ref text NOT NULL,
  decision_digest text NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.shipment_receipt (
  source text NOT NULL,
  source_ref text NOT NULL,
  shipment_id text NOT NULL,
  content_digest text NOT NULL,
  outcome text NOT NULL,
  deliveries integer NOT NULL DEFAULT 1,
  decision_digest text NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (source, source_ref)
);
CREATE TABLE mecc.delivery_log (
  id bigserial PRIMARY KEY,
  source text NOT NULL,
  source_ref text NOT NULL,
  shipment_id text NOT NULL,
  outcome text NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.shipment_reconciliation (
  run_id bigint NOT NULL,
  shipment_id text NOT NULL,
  class text NOT NULL,
  sources text[] NOT NULL,
  receipts integer NOT NULL,
  deliveries integer NOT NULL,
  decision_digest text NOT NULL,
  calendar_version integer NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (run_id, shipment_id)
);

-- Step 3: periods, sales entries, the close run and its outputs.
CREATE TABLE mecc.period (
  month text PRIMARY KEY,
  state text NOT NULL CHECK (state IN ('open','closing','closed')),
  changed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.sales_entry (
  entry_id bigserial PRIMARY KEY,
  order_no text NOT NULL,
  kind text NOT NULL CHECK (kind IN ('order','correction')),
  accepted_at timestamptz NOT NULL,
  business_date date NOT NULL,
  business_month text NOT NULL,
  posting_month text NOT NULL,
  posting_reason text NOT NULL,
  amount numeric(18,0) NOT NULL,
  alloc_status text NOT NULL CHECK (alloc_status IN ('allocated','awaiting')),
  inserted_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX ON mecc.sales_entry (posting_month, entry_id);

CREATE TABLE mecc.job_lease_seq (job_key text PRIMARY KEY, last_token bigint NOT NULL);
CREATE TABLE mecc.job_lease (
  job_key text PRIMARY KEY,
  owner_id text NOT NULL,
  fencing_token bigint NOT NULL,
  claimed_at timestamptz NOT NULL,
  heartbeat_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  owner_host_expires_at timestamptz NOT NULL
);
CREATE TABLE mecc.lease_event (
  id bigserial PRIMARY KEY,
  job_key text NOT NULL,
  owner_id text NOT NULL,
  fencing_token bigint,
  state text NOT NULL,
  db_time timestamptz NOT NULL DEFAULT clock_timestamp(),
  host_time timestamptz NOT NULL,
  naive_would_take_over boolean,
  message text NOT NULL DEFAULT ''
);

CREATE TABLE mecc.close_run (
  month text PRIMARY KEY,
  decision_digest text NOT NULL REFERENCES mecc.close_decision(digest),
  calendar_version integer NOT NULL,
  import_digest text NOT NULL,
  import_ids bigint[] NOT NULL,
  status text NOT NULL CHECK (status IN ('running','completed')),
  fencing_token bigint NOT NULL,
  owner_id text NOT NULL,
  as_of timestamptz NOT NULL,
  cut_at timestamptz NOT NULL,
  cut_snapshot text NOT NULL,
  cut_entries integer NOT NULL,
  cut_amount numeric(18,0) NOT NULL,
  cut_max_entry bigint NOT NULL,
  cut_shipments integer NOT NULL,
  cut_shipment_amount numeric(18,0) NOT NULL,
  cut_slips integer NOT NULL DEFAULT 0,
  cut_slip_amount numeric(18,2) NOT NULL DEFAULT 0,
  starts integer NOT NULL DEFAULT 1,
  started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  completed_at timestamptz
);
CREATE TABLE mecc.close_attempt (
  id bigserial PRIMARY KEY,
  month text NOT NULL,
  owner_id text NOT NULL,
  outcome text NOT NULL,
  fencing_token bigint,
  message text NOT NULL DEFAULT '',
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.close_chunk (
  month text NOT NULL,
  chunk_no integer NOT NULL,
  first_entry bigint NOT NULL,
  last_entry bigint NOT NULL,
  entries integer NOT NULL,
  amount numeric(18,0) NOT NULL,
  owner_id text NOT NULL,
  fencing_token bigint NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp(),
  PRIMARY KEY (month, chunk_no)
);
CREATE TABLE mecc.close_detail (
  month text NOT NULL,
  entry_id bigint NOT NULL,
  chunk_no integer NOT NULL,
  amount numeric(18,0) NOT NULL,
  business_date date NOT NULL,
  alloc_status text NOT NULL,
  fencing_token bigint NOT NULL,
  PRIMARY KEY (month, entry_id)
);
CREATE TABLE mecc.close_total (
  month text PRIMARY KEY,
  entries integer NOT NULL,
  amount numeric(18,0) NOT NULL,
  awaiting integer NOT NULL,
  shipments integer NOT NULL,
  shipment_amount numeric(18,0) NOT NULL,
  decision_digest text NOT NULL,
  calendar_version integer NOT NULL,
  import_digest text NOT NULL,
  fencing_token bigint NOT NULL,
  owner_id text NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Storage-side fencing: every write of the close carries the writer's
-- fencing token; the database refuses it unless it is the current token of
-- the close's lease and the lease has not expired by the database clock.
CREATE FUNCTION mecc.check_fence() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE cur bigint; exp timestamptz;
BEGIN
  SELECT fencing_token, expires_at INTO cur, exp FROM mecc.job_lease WHERE job_key = 'close:' || NEW.month FOR SHARE;
  IF cur IS NULL OR cur <> NEW.fencing_token THEN
    RAISE EXCEPTION 'fenced: % token % is not the current token %', TG_TABLE_NAME, NEW.fencing_token, cur USING ERRCODE = 'TF001';
  END IF;
  IF exp <= clock_timestamp() THEN
    RAISE EXCEPTION 'fenced: % token % lease expired at %', TG_TABLE_NAME, NEW.fencing_token, exp USING ERRCODE = 'TF001';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER fence BEFORE INSERT OR UPDATE ON mecc.close_run FOR EACH ROW EXECUTE FUNCTION mecc.check_fence();
CREATE TRIGGER fence BEFORE INSERT OR UPDATE ON mecc.close_chunk FOR EACH ROW EXECUTE FUNCTION mecc.check_fence();
CREATE TRIGGER fence BEFORE INSERT OR UPDATE ON mecc.close_detail FOR EACH ROW EXECUTE FUNCTION mecc.check_fence();
CREATE TRIGGER fence BEFORE INSERT OR UPDATE ON mecc.close_total FOR EACH ROW EXECUTE FUNCTION mecc.check_fence();

-- A sales entry can be posted to, or change its amount in, an open period
-- only. The application redirects to the next open period; this trigger is
-- the storage-side guard for any other writer.
CREATE FUNCTION mecc.guard_period() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE st text;
BEGIN
  IF TG_OP = 'UPDATE' THEN
    SELECT state INTO st FROM mecc.period WHERE month = OLD.posting_month FOR SHARE;
    IF st IS DISTINCT FROM 'open' THEN
      RAISE EXCEPTION 'period % is %: entry % cannot change', OLD.posting_month, st, OLD.entry_id USING ERRCODE = 'TP001';
    END IF;
  END IF;
  SELECT state INTO st FROM mecc.period WHERE month = NEW.posting_month FOR SHARE;
  IF st IS DISTINCT FROM 'open' THEN
    RAISE EXCEPTION 'period % is %: cannot post', NEW.posting_month, coalesce(st, 'missing') USING ERRCODE = 'TP001';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER guard BEFORE INSERT OR UPDATE OF amount, posting_month ON mecc.sales_entry FOR EACH ROW EXECUTE FUNCTION mecc.guard_period();
`

// ResetSchema drops and recreates the sample schema. Connect has already
// refused any database other than the dedicated one.
func ResetSchema(ctx context.Context, conn *pgx.Conn) error {
	var name string
	if err := conn.QueryRow(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return err
	}
	if name != DatabaseName {
		return fmt.Errorf("refuse to reset schema in %q", name)
	}
	_, err := conn.Exec(ctx, `DROP SCHEMA IF EXISTS mecc CASCADE; CREATE SCHEMA mecc;`+ddl+ddlSlice2)
	return err
}
