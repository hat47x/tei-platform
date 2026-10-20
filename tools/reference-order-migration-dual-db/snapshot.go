package ordermigrationdualdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrPlanMismatch is returned when a later step receives a plan whose digest
// differs from the one the primary database is bound to.
var ErrPlanMismatch = errors.New("plan digest mismatch")

// ObserveLegacy is step 1's observation: it reads the legacy order backlog in
// one repeatable-read transaction.
func ObserveLegacy(ctx context.Context, legacy *pgx.Conn) (Observation, error) {
	tx, err := legacy.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Observation{}, err
	}
	defer tx.Rollback(context.Background())
	orders, err := loadOrders(ctx, tx, legacyTables, "", keyRange{})
	if err != nil {
		return Observation{}, err
	}
	return Observe(orders)
}

// MakePlan runs step 1: observe the legacy database and build the plan.
func MakePlan(ctx context.Context, legacy *pgx.Conn, chunkSize int) (Plan, error) {
	obs, err := ObserveLegacy(ctx, legacy)
	if err != nil {
		return Plan{}, err
	}
	return BuildPlan(obs, chunkSize)
}

// SnapshotSchema identifies the snapshot record format.
const SnapshotSchema = "tei.reference.order-migration-snapshot/v0"

// SnapshotRecord is the output of step 2.
type SnapshotRecord struct {
	Schema     string           `json:"schema"`
	PlanDigest string           `json:"planDigest"`
	Position   SnapshotPosition `json:"position"`
	Copied     Summary          `json:"copied"`
	// MatchesPlanObservation is false when the legacy database changed
	// between step 1 and step 2. That is recorded, not refused: the copy is
	// the source for step 3 and the change feed belongs to step 4.
	MatchesPlanObservation bool   `json:"matchesPlanObservation"`
	Digest                 string `json:"digest,omitempty"`
}

// SnapshotPosition records where in the legacy database the copy was taken.
type SnapshotPosition struct {
	Isolation     string `json:"isolation"`
	PGSnapshot    string `json:"pgSnapshot"`
	WALLSN        string `json:"walLsn"`
	ServerVersion string `json:"serverVersion"`
	// Exactness states how WALLSN relates to PGSnapshot. They are read in the
	// same statement but WAL position is not transactional, so the pair is an
	// approximation. An exact pairing needs a logical replication slot's
	// consistent point (CREATE_REPLICATION_SLOT ... EXPORT_SNAPSHOT), which
	// belongs to step 4 and is not implemented.
	Exactness string `json:"exactness"`
}

func (s SnapshotRecord) computeDigest() (string, error) {
	s.Digest = ""
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// boundPlanDigest returns the plan digest the primary database is bound to,
// or "" when no plan has been recorded yet.
func boundPlanDigest(ctx context.Context, q pgx.Tx) (string, error) {
	var d string
	err := q.QueryRow(ctx, `SELECT plan_digest FROM omdd.mig_plan`).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return d, err
}

func requireBound(ctx context.Context, tx pgx.Tx, plan Plan) (SnapshotRecord, error) {
	bound, err := boundPlanDigest(ctx, tx)
	if err != nil {
		return SnapshotRecord{}, err
	}
	if bound == "" {
		return SnapshotRecord{}, errors.New("primary database has no plan; run step 2 first")
	}
	if bound != plan.Digest {
		return SnapshotRecord{}, fmt.Errorf("%w: primary is bound to %s, step received %s", ErrPlanMismatch, bound, plan.Digest)
	}
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT record_json FROM omdd.mig_snapshot WHERE plan_digest=$1`, plan.Digest).Scan(&raw); err != nil {
		return SnapshotRecord{}, fmt.Errorf("snapshot record for plan %s: %w", plan.Digest, err)
	}
	var rec SnapshotRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return SnapshotRecord{}, err
	}
	want, err := rec.computeDigest()
	if err != nil {
		return SnapshotRecord{}, err
	}
	if rec.Digest != want || rec.PlanDigest != plan.Digest {
		return SnapshotRecord{}, fmt.Errorf("%w: snapshot record does not carry plan %s (digest %s, recomputed %s)", ErrPlanMismatch, plan.Digest, rec.Digest, want)
	}
	return rec, nil
}

// TakeSnapshot runs step 2. It copies every legacy row inside one
// repeatable-read transaction into the primary database's staging tables and
// binds the copy to the plan digest. A second call for the same plan returns
// the existing record so the position stays fixed.
func TakeSnapshot(ctx context.Context, legacy, primary *pgx.Conn, plan Plan) (SnapshotRecord, bool, error) {
	if err := plan.Verify(); err != nil {
		return SnapshotRecord{}, false, err
	}
	{
		tx, err := primary.Begin(ctx)
		if err != nil {
			return SnapshotRecord{}, false, err
		}
		bound, err := boundPlanDigest(ctx, tx)
		if err == nil && bound != "" {
			rec, rerr := requireBound(ctx, tx, plan)
			tx.Rollback(context.Background())
			return rec, true, rerr
		}
		tx.Rollback(context.Background())
		if err != nil {
			return SnapshotRecord{}, false, err
		}
	}

	ltx, err := legacy.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return SnapshotRecord{}, false, err
	}
	defer ltx.Rollback(context.Background())
	var pos SnapshotPosition
	if err := ltx.QueryRow(ctx, `SELECT pg_current_snapshot()::text, pg_current_wal_lsn()::text, current_setting('server_version')`).
		Scan(&pos.PGSnapshot, &pos.WALLSN, &pos.ServerVersion); err != nil {
		return SnapshotRecord{}, false, err
	}
	pos.Isolation = "repeatable-read-read-only"
	pos.Exactness = "approximate-wal-position-read-in-snapshot-statement"
	orders, err := loadOrders(ctx, ltx, legacyTables, "", keyRange{})
	if err != nil {
		return SnapshotRecord{}, false, err
	}
	if err := ltx.Commit(ctx); err != nil {
		return SnapshotRecord{}, false, err
	}
	obs, err := Observe(orders)
	if err != nil {
		return SnapshotRecord{}, false, err
	}
	rec := SnapshotRecord{
		Schema:                 SnapshotSchema,
		PlanDigest:             plan.Digest,
		Position:               pos,
		Copied:                 obs.Summary,
		MatchesPlanObservation: obs.RowsetDigest == plan.Source.Observed.RowsetDigest,
	}
	if rec.Digest, err = rec.computeDigest(); err != nil {
		return SnapshotRecord{}, false, err
	}
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return SnapshotRecord{}, false, err
	}
	recJSON, err := json.Marshal(rec)
	if err != nil {
		return SnapshotRecord{}, false, err
	}

	tx, err := primary.Begin(ctx)
	if err != nil {
		return SnapshotRecord{}, false, err
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.mig_plan(plan_digest, plan_json) VALUES ($1,$2)`, plan.Digest, planJSON); err != nil {
		return SnapshotRecord{}, false, fmt.Errorf("record plan (a primary database hosts one plan): %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.mig_snapshot(plan_digest, snapshot_digest, record_json) VALUES ($1,$2,$3)`, plan.Digest, rec.Digest, recJSON); err != nil {
		return SnapshotRecord{}, false, err
	}
	for _, o := range orders {
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.stage_order(plan_digest, order_no, customer_code, ordered_on, status) VALUES ($1,$2,$3,$4::text::date,$5)`,
			plan.Digest, o.OrderNo, o.CustomerCode, o.OrderedOn, o.Status); err != nil {
			return SnapshotRecord{}, false, err
		}
		for _, l := range o.Lines {
			if _, err := tx.Exec(ctx, `INSERT INTO omdd.stage_order_line(plan_digest, order_no, line_no, item_code, qty, unit_price, amount) VALUES ($1,$2,$3,$4,$5,$6::text::numeric,$7::text::numeric)`,
				plan.Digest, o.OrderNo, l.LineNo, l.ItemCode, l.Qty, l.UnitPrice, l.Amount); err != nil {
				return SnapshotRecord{}, false, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SnapshotRecord{}, false, err
	}
	return rec, false, nil
}

// CheckBound verifies that the primary database is bound to plan and that
// its snapshot record still carries the plan digest.
func CheckBound(ctx context.Context, primary *pgx.Conn, plan Plan) (SnapshotRecord, error) {
	if err := plan.Verify(); err != nil {
		return SnapshotRecord{}, err
	}
	tx, err := primary.Begin(ctx)
	if err != nil {
		return SnapshotRecord{}, err
	}
	defer tx.Rollback(context.Background())
	return requireBound(ctx, tx, plan)
}
