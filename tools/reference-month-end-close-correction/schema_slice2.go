package monthendclose

// SQLSTATEs raised by the storage-side guards of steps 4-6.
const (
	// PeriodSQLState: a write into, or a change of a row in, a period that
	// is not open.
	PeriodSQLState = "TP001"
	// ImmutableSQLState: a slip of an open period was updated or deleted.
	// Slips are never changed; a correction is a red and a black slip.
	ImmutableSQLState = "TP002"
	// PriceOverlapSQLState: two price rows of an item overlap in both the
	// valid time and the record time.
	PriceOverlapSQLState = "TQ001"
	// PriceImmutableSQLState: a price row was changed other than by closing
	// its record time.
	PriceImmutableSQLState = "TQ002"
)

const ddlSlice2 = `
-- Step 5: items and unit prices on two axes. valid_from/valid_to is the
-- business validity (half-open dates), recorded_from/recorded_to is when the
-- system knew it (half-open instants, explicit business inputs). Only plain
-- comparisons are used, no range type or extension.
CREATE TABLE mecc.item (
  item_code text PRIMARY KEY,
  name text NOT NULL,
  tax_rate integer NOT NULL CHECK (tax_rate IN (8,10))
);
CREATE TABLE mecc.item_price (
  price_id bigserial PRIMARY KEY,
  item_code text NOT NULL REFERENCES mecc.item(item_code),
  unit_price numeric(12,2) NOT NULL CHECK (unit_price >= 0),
  valid_from date NOT NULL,
  valid_to date NOT NULL DEFAULT 'infinity',
  recorded_from timestamptz NOT NULL,
  recorded_to timestamptz NOT NULL DEFAULT 'infinity',
  op text NOT NULL,
  change_id bigint NOT NULL,
  CHECK (valid_from < valid_to),
  CHECK (recorded_from < recorded_to)
);
CREATE INDEX ON mecc.item_price (item_code, recorded_to);
CREATE TABLE mecc.price_change (
  change_id bigserial PRIMARY KEY,
  item_code text NOT NULL,
  op text NOT NULL,
  valid_from date NOT NULL,
  valid_to date,
  unit_price numeric(12,2),
  recorded_at timestamptz NOT NULL,
  outcome text NOT NULL,
  message text NOT NULL DEFAULT '',
  repriced integer NOT NULL DEFAULT 0,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Storage-side guard of the price history: rows of one item never overlap
-- in both axes, and a row only changes by closing its record time.
CREATE FUNCTION mecc.guard_price() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    RAISE EXCEPTION 'price % cannot be deleted', OLD.price_id USING ERRCODE = 'TQ002';
  END IF;
  IF TG_OP = 'UPDATE' THEN
    IF NEW.item_code <> OLD.item_code OR NEW.unit_price <> OLD.unit_price OR NEW.valid_from <> OLD.valid_from OR NEW.valid_to <> OLD.valid_to
       OR NEW.recorded_from <> OLD.recorded_from OR OLD.recorded_to <> 'infinity' OR NEW.recorded_to > OLD.recorded_to THEN
      RAISE EXCEPTION 'price % can only close its record time', OLD.price_id USING ERRCODE = 'TQ002';
    END IF;
    RETURN NEW;
  END IF;
  PERFORM 1 FROM mecc.item WHERE item_code = NEW.item_code FOR UPDATE;
  IF EXISTS (SELECT 1 FROM mecc.item_price p WHERE p.item_code = NEW.item_code AND p.price_id <> NEW.price_id
             AND p.valid_from < NEW.valid_to AND NEW.valid_from < p.valid_to
             AND p.recorded_from < NEW.recorded_to AND NEW.recorded_from < p.recorded_to) THEN
    RAISE EXCEPTION 'price of % for [%,%) overlaps a row known at the same time', NEW.item_code, NEW.valid_from, NEW.valid_to USING ERRCODE = 'TQ001';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER guard BEFORE INSERT OR UPDATE OR DELETE ON mecc.item_price FOR EACH ROW EXECUTE FUNCTION mecc.guard_price();

-- Step 4: order events with their ordering key (order_no, version), business
-- date and reception instant, and the projection of each order.
CREATE TABLE mecc.order_event (
  event_id text PRIMARY KEY,
  kind text NOT NULL CHECK (kind IN ('shipment-registered','shipment-changed','shipment-cancelled')),
  order_no text NOT NULL,
  version integer NOT NULL CHECK (version >= 1),
  business_date date NOT NULL,
  received_at timestamptz NOT NULL,
  source text NOT NULL,
  body jsonb NOT NULL,
  content_digest text NOT NULL,
  status text NOT NULL CHECK (status IN ('accepted','confirmed','conflict','held','released')),
  period_state text NOT NULL,
  policy text,
  arrived_after_version integer NOT NULL,
  deliveries integer NOT NULL DEFAULT 1,
  decision_digest text NOT NULL,
  calendar_version integer NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX order_event_key ON mecc.order_event (order_no, version) WHERE status IN ('accepted','held','released');
CREATE TABLE mecc.order_event_delivery (
  id bigserial PRIMARY KEY,
  event_id text NOT NULL,
  outcome text NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE mecc.order_state (
  order_no text PRIMARY KEY,
  status text NOT NULL CHECK (status IN ('pending-base','active','cancelled')),
  customer text,
  item_code text,
  ship_date date,
  qty bigint,
  version integer NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

-- Step 5: the slip ledger. A slip is one line. A slip is never changed: a
-- correction is a red slip (the exact negation of the live slip, linked by
-- reverses) and a black slip (the new values, linked by corrects).
CREATE TABLE mecc.slip (
  slip_id bigserial PRIMARY KEY,
  kind text NOT NULL CHECK (kind IN ('original','red','black')),
  order_no text NOT NULL,
  customer text NOT NULL,
  item_code text NOT NULL REFERENCES mecc.item(item_code),
  tax_rate integer NOT NULL,
  ship_date date NOT NULL,
  business_month text NOT NULL,
  posting_month text NOT NULL,
  posting_reason text NOT NULL,
  qty bigint NOT NULL,
  unit_price numeric(12,2) NOT NULL,
  amount numeric(14,2) NOT NULL,
  price_id bigint NOT NULL REFERENCES mecc.item_price(price_id),
  price_valid_on date NOT NULL,
  price_known_at timestamptz NOT NULL,
  known_at timestamptz NOT NULL,
  reverses bigint UNIQUE REFERENCES mecc.slip(slip_id),
  corrects bigint UNIQUE REFERENCES mecc.slip(slip_id),
  cause text NOT NULL,
  decision_digest text NOT NULL,
  calendar_version integer NOT NULL,
  closed_month text,
  close_token bigint,
  close_decision_digest text,
  inserted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
  CHECK (amount = qty * unit_price),
  CHECK ((kind = 'red') = (reverses IS NOT NULL)),
  CHECK ((kind = 'black') = (corrects IS NOT NULL))
);
CREATE INDEX ON mecc.slip (order_no);
CREATE INDEX ON mecc.slip (posting_month, slip_id);

CREATE FUNCTION mecc.guard_slip() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE st text;
BEGIN
  IF TG_OP = 'INSERT' THEN
    SELECT state INTO st FROM mecc.period WHERE month = NEW.posting_month FOR SHARE;
    IF st IS DISTINCT FROM 'open' THEN
      RAISE EXCEPTION 'period % is %: cannot post a slip', NEW.posting_month, coalesce(st, 'missing') USING ERRCODE = 'TP001';
    END IF;
    RETURN NEW;
  END IF;
  SELECT state INTO st FROM mecc.period WHERE month = OLD.posting_month FOR SHARE;
  IF st IS DISTINCT FROM 'open' THEN
    RAISE EXCEPTION 'period % is %: slip % cannot change', OLD.posting_month, st, OLD.slip_id USING ERRCODE = 'TP001';
  END IF;
  RAISE EXCEPTION 'slip % is immutable: post a red and a black slip', OLD.slip_id USING ERRCODE = 'TP002';
END $$;
CREATE TRIGGER guard BEFORE INSERT OR UPDATE OR DELETE ON mecc.slip FOR EACH ROW EXECUTE FUNCTION mecc.guard_slip();

-- Step 6: the report the close wrote, with its digest. Fenced like every
-- other write of the close.
CREATE TABLE mecc.close_report (
  month text PRIMARY KEY,
  slips integer NOT NULL,
  base_amount numeric(18,2) NOT NULL,
  tax_total bigint NOT NULL,
  tax_rule text NOT NULL,
  report_sha256 text NOT NULL,
  report bytea NOT NULL,
  decision_digest text NOT NULL,
  calendar_version integer NOT NULL,
  fencing_token bigint NOT NULL,
  owner_id text NOT NULL,
  at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER fence BEFORE INSERT OR UPDATE ON mecc.close_report FOR EACH ROW EXECUTE FUNCTION mecc.check_fence();
`
