package ordermigrationdualdb

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// ReconcileSchema identifies the reconciliation result format.
const ReconcileSchema = "tei.reference.order-migration-reconcile/v0"

// ReconcileResult compares the staged copy (the legacy rows at the snapshot)
// with the primary order backlog under the plan's declared rules.
type ReconcileResult struct {
	Schema         string       `json:"schema"`
	PlanDigest     string       `json:"planDigest"`
	SnapshotDigest string       `json:"snapshotDigest"`
	Rules          []string     `json:"rules"`
	Source         Summary      `json:"source"`
	Target         Summary      `json:"target"`
	Passed         bool         `json:"passed"`
	Differences    []Difference `json:"differences"`
}

// Reconcile is step 3's reconciliation. It refuses a plan the primary is not
// bound to, and reports (rather than hides) a staged copy that no longer
// matches its snapshot record or chunks that have no checkpoint yet.
func Reconcile(ctx context.Context, primary *pgx.Conn, plan Plan) (ReconcileResult, error) {
	if err := plan.Verify(); err != nil {
		return ReconcileResult{}, err
	}
	tx, err := primary.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ReconcileResult{}, err
	}
	defer tx.Rollback(context.Background())
	snap, err := requireBound(ctx, tx, plan)
	if err != nil {
		return ReconcileResult{}, err
	}
	res := ReconcileResult{Schema: ReconcileSchema, PlanDigest: plan.Digest, SnapshotDigest: snap.Digest, Rules: plan.Reconciliation.Rules, Differences: []Difference{}}
	staged, err := loadOrders(ctx, tx, stageTables, plan.Digest, keyRange{})
	if err != nil {
		return res, err
	}
	source, err := Observe(staged)
	if err != nil {
		return res, err
	}
	migrated, err := loadOrders(ctx, tx, primaryTables, "", keyRange{})
	if err != nil {
		return res, err
	}
	if plan.Declares(KnownDifferenceItemCode) {
		// The only declared known difference: compare converted item codes
		// as legacy codes.
		for i := range migrated {
			for j := range migrated[i].Lines {
				migrated[i].Lines[j].ItemCode = plan.LegacyCode(migrated[i].Lines[j].ItemCode)
			}
		}
	}
	target, err := Observe(migrated)
	if err != nil {
		return res, err
	}
	res.Source, res.Target = source.Summary, target.Summary
	if source.RowsetDigest != snap.Copied.RowsetDigest {
		res.Differences = append(res.Differences, Difference{Rule: "staged-copy-matches-snapshot", Source: snap.Copied.RowsetDigest, Target: source.RowsetDigest})
	}
	var checkpoints int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM omdd.mig_checkpoint WHERE plan_digest=$1 AND snapshot_digest=$2`, plan.Digest, snap.Digest).Scan(&checkpoints); err != nil {
		return res, err
	}
	if checkpoints != len(plan.Chunks) {
		res.Differences = append(res.Differences, Difference{Rule: "all-chunks-checkpointed", Source: strconv.Itoa(len(plan.Chunks)), Target: strconv.Itoa(checkpoints)})
	}
	res.Differences = append(res.Differences, Compare(plan.Reconciliation.Rules, source, target)...)
	res.Passed = len(res.Differences) == 0
	return res, nil
}
