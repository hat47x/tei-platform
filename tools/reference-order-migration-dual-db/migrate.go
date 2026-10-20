package ordermigrationdualdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5"
)

// MigrateResult reports one call of Migrate.
type MigrateResult struct {
	PlanDigest     string   `json:"planDigest"`
	SnapshotDigest string   `json:"snapshotDigest"`
	ChunksPlanned  int      `json:"chunksPlanned"`
	ChunksResumed  []string `json:"chunksResumed"`
	ChunksWritten  []string `json:"chunksWritten"`
	OrdersWritten  int      `json:"ordersWritten"`
	OrdersExisting int      `json:"ordersExisting"`
}

// ReservationPayload is the outbox payload delivered to the inventory DB.
type ReservationPayload struct {
	OrderNo string            `json:"orderNo"`
	Lines   []ReservationLine `json:"lines"`
}

// ReservationLine is one line to reserve.
type ReservationLine struct {
	LineNo   int    `json:"lineNo"`
	ItemCode string `json:"itemCode"`
	Qty      int64  `json:"qty"`
}

// OperationID derives the business operation identifier for allocating one
// migrated order. It depends on the plan digest and the order key only.
func OperationID(planDigest, orderNo string) string {
	short := planDigest
	if len(short) > len("sha256:")+16 {
		short = short[len("sha256:") : len("sha256:")+16]
	}
	return "migrate-allocate:" + short + ":" + orderNo
}

// EffectID derives the effect identifier: operation identifier plus ordinal.
func EffectID(planDigest, orderNo string) string { return OperationID(planDigest, orderNo) + "#1" }

// Migrate runs step 3's primary side. Each chunk is one primary transaction
// holding the order rows, their outbox effects and the chunk checkpoint.
// Chunks with a checkpoint are skipped, which is the resume point.
func Migrate(ctx context.Context, primary *pgx.Conn, plan Plan) (MigrateResult, error) {
	if err := plan.Verify(); err != nil {
		return MigrateResult{}, err
	}
	res := MigrateResult{PlanDigest: plan.Digest, ChunksPlanned: len(plan.Chunks), ChunksResumed: []string{}, ChunksWritten: []string{}}
	for _, chunk := range plan.Chunks {
		done, err := migrateChunk(ctx, primary, plan, chunk, &res)
		if err != nil {
			return res, fmt.Errorf("chunk %s: %w", chunk.ID, err)
		}
		if done {
			res.ChunksResumed = append(res.ChunksResumed, chunk.ID)
			continue
		}
		res.ChunksWritten = append(res.ChunksWritten, chunk.ID)
		crashPoint(CrashAfterPrimaryCommit)
	}
	return res, nil
}

func migrateChunk(ctx context.Context, primary *pgx.Conn, plan Plan, chunk Chunk, res *MigrateResult) (bool, error) {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(context.Background())
	snap, err := requireBound(ctx, tx, plan)
	if err != nil {
		return false, err
	}
	res.SnapshotDigest = snap.Digest
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM omdd.mig_checkpoint WHERE plan_digest=$1 AND chunk_id=$2)`, plan.Digest, chunk.ID).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	orders, err := loadOrders(ctx, tx, stageTables, plan.Digest, chunkRange(chunk))
	if err != nil {
		return false, err
	}
	lines := 0
	sum := new(big.Int)
	for i, o := range orders {
		if i == len(orders)/2 {
			crashPoint(CrashChunkMid)
		}
		written, err := writeOrder(ctx, tx, plan, chunk, o)
		if err != nil {
			return false, fmt.Errorf("order %s: %w", o.OrderNo, err)
		}
		if written {
			res.OrdersWritten++
		} else {
			res.OrdersExisting++
		}
		for _, l := range o.Lines {
			a, _ := new(big.Int).SetString(l.Amount, 10)
			sum.Add(sum, a)
			lines++
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.mig_checkpoint(plan_digest, chunk_id, snapshot_digest, order_count, line_count, amount_sum) VALUES ($1,$2,$3,$4,$5,$6::text::numeric)`,
		plan.Digest, chunk.ID, snap.Digest, len(orders), lines, sum.String()); err != nil {
		return false, err
	}
	return false, tx.Commit(ctx)
}

// writeOrder writes one order idempotently: an existing row with the same
// content is accepted, an existing row with different content is refused.
func writeOrder(ctx context.Context, tx pgx.Tx, plan Plan, chunk Chunk, o Order) (bool, error) {
	tag, err := tx.Exec(ctx, `INSERT INTO omdd.order_backlog(order_no, customer_code, ordered_on, status, plan_digest, chunk_id, allocation_state)
		VALUES ($1,$2,$3::text::date,$4,$5,$6,'awaiting') ON CONFLICT (order_no) DO NOTHING`,
		o.OrderNo, o.CustomerCode, o.OrderedOn, o.Status, plan.Digest, chunk.ID)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		existing, err := loadOrders(ctx, tx, primaryTables, "", keyRange{exact: o.OrderNo})
		if err != nil {
			return false, err
		}
		if len(existing) != 1 || RowHash(existing[0]) != RowHash(o) {
			return false, errors.New("primary already holds a different row for this key")
		}
		return false, nil
	}
	payload := ReservationPayload{OrderNo: o.OrderNo}
	for _, l := range o.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.order_backlog_line(order_no, line_no, item_code, qty, unit_price, amount) VALUES ($1,$2,$3,$4,$5::text::numeric,$6::text::numeric)`,
			o.OrderNo, l.LineNo, l.ItemCode, l.Qty, l.UnitPrice, l.Amount); err != nil {
			return false, err
		}
		payload.Lines = append(payload.Lines, ReservationLine{LineNo: l.LineNo, ItemCode: l.ItemCode, Qty: l.Qty})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO omdd.effect_outbox(effect_id, operation_id, plan_digest, order_no, kind, destination, payload, state)
		VALUES ($1,$2,$3,$4,'reserve-stock','inventory',$5,'pending') ON CONFLICT (effect_id) DO NOTHING`,
		EffectID(plan.Digest, o.OrderNo), OperationID(plan.Digest, o.OrderNo), plan.Digest, o.OrderNo, raw)
	return true, err
}
