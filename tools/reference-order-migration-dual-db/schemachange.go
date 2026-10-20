package ordermigrationdualdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// The name split of the customer master (P-MIG-03): full_name is split into
// family_name and given_name with the expand, migrate and contract phases.
const (
	NameSplitChange   = "customer-name-split"
	SchemaVersionV1   = "v1-full-name"
	SchemaVersionV2   = "v2-split-name"
	ClientLegacyCDC   = "legacy-cdc"
	ClientNewApp      = "new-app"
	PhaseExpand       = "expand"
	PhaseMigrated     = "migrate-complete"
	PhaseContracted   = "contract"
	nameSyncSplitExpr = `^([^ 　]+)[ 　]+([^ 　].*)$`
)

// nameSplitDDL adds the new columns and the trigger that keeps both forms
// correct while old-version writers remain. An old-version write changes
// full_name only: the trigger splits it. A new-version write changes the
// two names only: the trigger composes full_name. A full name that cannot
// be split is not guessed: both new columns become NULL and the row is
// listed in omdd.schema_issue.
const nameSplitDDL = `
ALTER TABLE omdd.customer ADD COLUMN IF NOT EXISTS family_name text, ADD COLUMN IF NOT EXISTS given_name text;
CREATE OR REPLACE FUNCTION omdd.split_name(n text) RETURNS text[] LANGUAGE sql IMMUTABLE AS
  $$ SELECT regexp_match(n, '` + nameSyncSplitExpr + `') $$;
CREATE OR REPLACE FUNCTION omdd.customer_name_sync() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  m text[];
  do_split boolean;
  do_compose boolean;
BEGIN
  IF TG_OP = 'INSERT' THEN
    do_split := NEW.family_name IS NULL AND NEW.given_name IS NULL;
    do_compose := NOT do_split;
  ELSE
    do_split := NEW.full_name IS DISTINCT FROM OLD.full_name
      AND NEW.family_name IS NOT DISTINCT FROM OLD.family_name AND NEW.given_name IS NOT DISTINCT FROM OLD.given_name;
    do_compose := NEW.full_name IS NOT DISTINCT FROM OLD.full_name
      AND (NEW.family_name IS DISTINCT FROM OLD.family_name OR NEW.given_name IS DISTINCT FROM OLD.given_name);
  END IF;
  IF do_split THEN
    m := omdd.split_name(NEW.full_name);
    IF m IS NULL THEN
      NEW.family_name := NULL;
      NEW.given_name := NULL;
      INSERT INTO omdd.schema_issue(change_id, entity_key, reason, value)
        VALUES ('customer-name-split', NEW.customer_code, 'unsplittable-full-name', NEW.full_name);
      RETURN NEW;
    END IF;
    NEW.family_name := m[1];
    NEW.given_name := m[2];
  ELSIF do_compose THEN
    IF NEW.family_name IS NULL OR NEW.given_name IS NULL THEN
      RAISE EXCEPTION 'customer %: family_name and given_name must be written together', NEW.customer_code;
    END IF;
    m := omdd.split_name(NEW.full_name);
    IF m IS NULL OR m[1] IS DISTINCT FROM NEW.family_name OR m[2] IS DISTINCT FROM NEW.given_name THEN
      NEW.full_name := NEW.family_name || ' ' || NEW.given_name;
    END IF;
  ELSIF TG_OP = 'UPDATE' AND NEW.full_name IS DISTINCT FROM OLD.full_name THEN
    m := omdd.split_name(NEW.full_name);
    IF m IS NULL OR m[1] IS DISTINCT FROM NEW.family_name OR m[2] IS DISTINCT FROM NEW.given_name THEN
      RAISE EXCEPTION 'customer %: full_name and family/given names disagree', NEW.customer_code;
    END IF;
  END IF;
  IF NEW.family_name IS NOT NULL AND NEW.given_name IS NOT NULL THEN
    UPDATE omdd.schema_issue SET state = 'resolved', resolved_at = now()
      WHERE change_id = 'customer-name-split' AND entity_key = NEW.customer_code AND state = 'open';
  END IF;
  RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS customer_name_sync ON omdd.customer;
CREATE TRIGGER customer_name_sync BEFORE INSERT OR UPDATE ON omdd.customer FOR EACH ROW EXECUTE FUNCTION omdd.customer_name_sync();`

// ExpandResult reports the expand and migrate phases.
type ExpandResult struct {
	Phase        string   `json:"phase"`
	Backfilled   int      `json:"backfilled"`
	Unsplittable []string `json:"unsplittable"`
}

// ExpandNameSplit runs the expand phase (new columns, trigger) and the
// migrate phase (backfill) of the name split. It registers the change feed
// from the legacy side as an old-version (full name) writer and the new
// application as a new-version writer. It is idempotent.
func ExpandNameSplit(ctx context.Context, primary *pgx.Conn, plan Plan) (ExpandResult, error) {
	res := ExpandResult{Unsplittable: []string{}}
	tx, err := primary.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(context.Background())
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return res, err
	}
	if err := loadCustomerMaster(ctx, tx, plan); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, nameSplitDDL); err != nil {
		return res, fmt.Errorf("expand: %w", err)
	}
	if err := setPhase(ctx, tx, plan, PhaseExpand, map[string]any{"columns": []string{"family_name", "given_name"}, "trigger": "customer_name_sync"}); err != nil {
		return res, err
	}
	for id, v := range map[string]string{ClientLegacyCDC: SchemaVersionV1, ClientNewApp: SchemaVersionV2} {
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.schema_client(client_id, schema_version) VALUES ($1,$2) ON CONFLICT DO NOTHING`, id, v); err != nil {
			return res, err
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE omdd.customer SET family_name = (omdd.split_name(full_name))[1], given_name = (omdd.split_name(full_name))[2]
		WHERE family_name IS NULL AND given_name IS NULL AND omdd.split_name(full_name) IS NOT NULL`)
	if err != nil {
		return res, fmt.Errorf("backfill: %w", err)
	}
	res.Backfilled = int(tag.RowsAffected())
	rows, err := tx.Query(ctx, `SELECT customer_code FROM omdd.customer WHERE family_name IS NULL AND given_name IS NULL ORDER BY customer_code COLLATE "C"`)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return res, err
		}
		res.Unsplittable = append(res.Unsplittable, code)
	}
	rows.Close()
	for _, code := range res.Unsplittable {
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.schema_issue(change_id, entity_key, reason, value)
			SELECT $1, customer_code, 'unsplittable-full-name', full_name FROM omdd.customer c WHERE customer_code=$2
			AND NOT EXISTS (SELECT 1 FROM omdd.schema_issue i WHERE i.change_id=$1 AND i.entity_key=$2 AND i.state='open')`, NameSplitChange, code); err != nil {
			return res, err
		}
	}
	if err := setPhase(ctx, tx, plan, PhaseMigrated, map[string]any{"backfilled": res.Backfilled, "unsplittable": res.Unsplittable}); err != nil {
		return res, err
	}
	res.Phase = PhaseMigrated
	return res, tx.Commit(ctx)
}

func setPhase(ctx context.Context, tx pgx.Tx, plan Plan, phase string, detail map[string]any) error {
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.schema_change(change_id, phase, plan_digest) VALUES ($1,$2,$3)
		ON CONFLICT (change_id) DO UPDATE SET phase=EXCLUDED.phase, updated_at=now()`, NameSplitChange, phase, plan.Digest); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO omdd.schema_change_event(change_id, event, detail) VALUES ($1,$2,$3)`, NameSplitChange, phase, detail)
	return err
}

// RegisterSchemaClient records a writer of the customer master and the
// schema version it writes.
func RegisterSchemaClient(ctx context.Context, primary *pgx.Conn, id, version string) error {
	_, err := primary.Exec(ctx, `INSERT INTO omdd.schema_client(client_id, schema_version) VALUES ($1,$2)
		ON CONFLICT (client_id) DO UPDATE SET schema_version=EXCLUDED.schema_version, active=true, retired_at=NULL, last_seen_at=now()`, id, version)
	return err
}

// RetireSchemaClient records that a writer is gone.
func RetireSchemaClient(ctx context.Context, primary *pgx.Conn, id string) error {
	tag, err := primary.Exec(ctx, `UPDATE omdd.schema_client SET active=false, retired_at=now() WHERE client_id=$1`, id)
	if err == nil && tag.RowsAffected() != 1 {
		err = fmt.Errorf("schema client %s is not registered", id)
	}
	return err
}

// ErrContractRefused is returned while the contract phase is not allowed.
var ErrContractRefused = errors.New("contract refused")

// ContractCheck is the decision on the contract phase.
type ContractCheck struct {
	Phase    string   `json:"phase"`
	Allowed  bool     `json:"allowed"`
	Blockers []string `json:"blockers"`
}

// CheckContractNameSplit decides whether full_name may be dropped. It is
// refused while any active writer still writes the old version, while names
// remain that could not be split, and before the migrate phase. A refusal is
// recorded as an event. Dropping the column itself is outside this sample:
// the legacy system stays an old-version writer until it is retired, which
// the sample never reaches.
func CheckContractNameSplit(ctx context.Context, q interface {
	Begin(context.Context) (pgx.Tx, error)
}, plan Plan) (ContractCheck, error) {
	res := ContractCheck{Blockers: []string{}}
	tx, err := q.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(context.Background())
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return res, err
	}
	err = tx.QueryRow(ctx, `SELECT phase FROM omdd.schema_change WHERE change_id=$1`, NameSplitChange).Scan(&res.Phase)
	if errors.Is(err, pgx.ErrNoRows) {
		res.Blockers = append(res.Blockers, "phase:not-expanded")
	} else if err != nil {
		return res, err
	} else if res.Phase != PhaseMigrated {
		res.Blockers = append(res.Blockers, "phase:"+res.Phase)
	}
	rows, err := tx.Query(ctx, `SELECT client_id FROM omdd.schema_client WHERE active AND schema_version=$1 ORDER BY client_id`, SchemaVersionV1)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return res, err
		}
		res.Blockers = append(res.Blockers, "old-version-writer:"+id)
	}
	rows.Close()
	var open int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM omdd.schema_issue WHERE change_id=$1 AND state='open'`, NameSplitChange).Scan(&open); err != nil {
		return res, err
	}
	if open > 0 {
		res.Blockers = append(res.Blockers, fmt.Sprintf("open-issues:%d", open))
	}
	res.Allowed = len(res.Blockers) == 0
	event := "contract-allowed"
	if !res.Allowed {
		event = "contract-refused"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.schema_change_event(change_id, event, detail) VALUES ($1,$2,$3)`, NameSplitChange, event, map[string]any{"blockers": res.Blockers}); err != nil {
		return res, err
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	if !res.Allowed {
		return res, fmt.Errorf("%w: %v", ErrContractRefused, res.Blockers)
	}
	return res, nil
}
