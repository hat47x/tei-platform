package ordermigrationdualdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// ReverseResult reports one reverse-sync pass (primary to legacy).
type ReverseResult struct {
	PlanDigest     string `json:"planDigest"`
	GateDigest     string `json:"gateDigest,omitempty"`
	Entries        int    `json:"entries"`
	Groups         int    `json:"groups"`
	AppliedGroups  int    `json:"appliedGroups"`
	AlreadyApplied int    `json:"alreadyAppliedGroups"`
}

type reverseGroup struct {
	entity, key string
	seqs        []int64
	fields      map[string]bool
}

// ReverseSync delivers pending reverse-sync entries to the legacy database.
// Entries of one entity key are delivered together in one legacy
// transaction that also writes a marker (so the change feed does not send
// the change back) and a receipt per outbox sequence carrying the plan
// digest and the gate digest. The legacy state is written from the primary's
// current state; a group whose receipts already exist is not written again.
// The primary outbox is marked delivered after the legacy commit.
func ReverseSync(ctx context.Context, legacy, primary *pgx.Conn, plan Plan) (ReverseResult, error) {
	res := ReverseResult{PlanDigest: plan.Digest}
	if _, err := CheckBound(ctx, primary, plan); err != nil {
		return res, err
	}
	{
		tx, err := primary.Begin(ctx)
		if err != nil {
			return res, err
		}
		_, res.GateDigest, err = orderAuthority(ctx, tx)
		tx.Rollback(context.Background())
		if err != nil {
			return res, err
		}
	}
	rows, err := primary.Query(ctx, `SELECT seq, entity, entity_key, fields FROM omdd.reverse_outbox WHERE state='pending' AND plan_digest=$1 ORDER BY seq`, plan.Digest)
	if err != nil {
		return res, err
	}
	var groups []*reverseGroup
	byKey := map[string]*reverseGroup{}
	for rows.Next() {
		var seq int64
		var entity, key string
		var fields []string
		if err := rows.Scan(&seq, &entity, &key, &fields); err != nil {
			rows.Close()
			return res, err
		}
		res.Entries++
		g := byKey[entity+"\x00"+key]
		if g == nil {
			g = &reverseGroup{entity: entity, key: key, fields: map[string]bool{}}
			byKey[entity+"\x00"+key] = g
			groups = append(groups, g)
		}
		g.seqs = append(g.seqs, seq)
		for _, f := range fields {
			g.fields[f] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	res.Groups = len(groups)
	for _, g := range groups {
		applied, err := applyReverseGroup(ctx, legacy, primary, plan, res.GateDigest, g)
		if err != nil {
			return res, fmt.Errorf("%s %s: %w", g.entity, g.key, err)
		}
		if applied {
			res.AppliedGroups++
		} else {
			res.AlreadyApplied++
		}
		crashPoint(CrashAfterReverseLegacyCommit)
		if _, err := primary.Exec(ctx, `UPDATE omdd.reverse_outbox SET state='delivered', delivered_at=now() WHERE seq = ANY($1)`, g.seqs); err != nil {
			return res, err
		}
	}
	return res, nil
}

func applyReverseGroup(ctx context.Context, legacy, primary *pgx.Conn, plan Plan, gate string, g *reverseGroup) (bool, error) {
	ltx, err := legacy.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer ltx.Rollback(context.Background())
	var have int
	if err := ltx.QueryRow(ctx, `SELECT count(*) FROM omdd.legacy_reverse_applied WHERE seq = ANY($1)`, g.seqs).Scan(&have); err != nil {
		return false, err
	}
	if have == len(g.seqs) {
		return false, nil
	}
	maxSeq := g.seqs[len(g.seqs)-1]
	if _, err := ltx.Exec(ctx, `INSERT INTO omdd.legacy_sync_marker(origin, ref) VALUES ('primary-reverse-sync', $1)`, "seq:"+strconv.FormatInt(maxSeq, 10)); err != nil {
		return false, err
	}
	switch g.entity {
	case "order":
		cur, err := loadOrders(ctx, primary, primaryTables, "", keyRange{exact: g.key})
		if err != nil {
			return false, err
		}
		if len(cur) != 1 {
			return false, errors.New("order deleted on the primary; reverse delete is not supported")
		}
		o := cur[0]
		if _, err := ltx.Exec(ctx, `INSERT INTO omdd.legacy_order(order_no, customer_code, ordered_on, status) VALUES ($1,$2,$3::text::date,$4)
			ON CONFLICT (order_no) DO UPDATE SET customer_code=EXCLUDED.customer_code, ordered_on=EXCLUDED.ordered_on, status=EXCLUDED.status`,
			o.OrderNo, o.CustomerCode, o.OrderedOn, o.Status); err != nil {
			return false, err
		}
		keep := make([]int32, 0, len(o.Lines))
		for _, l := range o.Lines {
			keep = append(keep, int32(l.LineNo))
		}
		if _, err := ltx.Exec(ctx, `DELETE FROM omdd.legacy_order_line WHERE order_no=$1 AND NOT (line_no = ANY($2))`, o.OrderNo, keep); err != nil {
			return false, err
		}
		for _, l := range o.Lines {
			if _, err := ltx.Exec(ctx, `INSERT INTO omdd.legacy_order_line(order_no, line_no, item_code, qty, unit_price, amount) VALUES ($1,$2,$3,$4,$5::text::numeric,$6::text::numeric)
				ON CONFLICT (order_no, line_no) DO UPDATE SET item_code=EXCLUDED.item_code, qty=EXCLUDED.qty, unit_price=EXCLUDED.unit_price, amount=EXCLUDED.amount`,
				o.OrderNo, l.LineNo, plan.LegacyCode(l.ItemCode), l.Qty, l.UnitPrice, l.Amount); err != nil {
				return false, err
			}
		}
	case "customer":
		cur, err := scanCustomers(ctx, primary, `SELECT customer_code, full_name, address, credit_limit::text FROM omdd.customer WHERE customer_code=$1`, g.key)
		if err != nil {
			return false, err
		}
		if len(cur) != 1 {
			return false, errors.New("customer missing on the primary")
		}
		for _, f := range CustomerFields() {
			if !g.fields[f] {
				continue
			}
			cast := ""
			if f == FieldCreditLimit {
				cast = "::text::numeric"
			}
			tag, err := ltx.Exec(ctx, fmt.Sprintf(`UPDATE omdd.legacy_customer SET %s = $2%s WHERE customer_code=$1`, legacyColumn(f), cast), g.key, cur[0].field(f))
			if err != nil {
				return false, err
			}
			if tag.RowsAffected() != 1 {
				return false, errors.New("customer missing on the legacy side")
			}
		}
	default:
		return false, fmt.Errorf("unknown entity %q", g.entity)
	}
	for _, seq := range g.seqs {
		if _, err := ltx.Exec(ctx, `INSERT INTO omdd.legacy_reverse_applied(seq, plan_digest, gate_digest, entity, entity_key) VALUES ($1,$2,NULLIF($3,''),$4,$5) ON CONFLICT (seq) DO NOTHING`,
			seq, plan.Digest, gate, g.entity, g.key); err != nil {
			return false, err
		}
	}
	return true, ltx.Commit(ctx)
}

// ErrNotAuthority is returned for new-side order writes before the cutover.
var ErrNotAuthority = errors.New("primary is not the authority for orders")

func requireOrderAuthority(ctx context.Context, tx pgx.Tx) error {
	auth, _, err := orderAuthority(ctx, tx)
	if err != nil {
		return err
	}
	switch auth {
	case "primary":
		return nil
	case "rolling-back":
		return ErrFrozen
	}
	return fmt.Errorf("%w (authority %q)", ErrNotAuthority, auth)
}

// PlaceOrder is the new application's order entry after the cutover. The
// order gets its reservation effect like a migrated one.
func PlaceOrder(ctx context.Context, primary *pgx.Conn, plan Plan, o Order) error {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return err
	}
	if err := requireOrderAuthority(ctx, tx); err != nil {
		return err
	}
	written, err := writeOrder(ctx, tx, plan, plan.ChunkFor(o.OrderNo), o)
	if err != nil {
		return err
	}
	if !written {
		return fmt.Errorf("order %s already exists", o.OrderNo)
	}
	return tx.Commit(ctx)
}

// UpdateOrderStatus is the new application's status change after the cutover.
func UpdateOrderStatus(ctx context.Context, primary *pgx.Conn, orderNo, status string) error {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err := requireOrderAuthority(ctx, tx); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE omdd.order_backlog SET status=$2 WHERE order_no=$1`, orderNo, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("order %s not found", orderNo)
	}
	return tx.Commit(ctx)
}

// RollbackSchema identifies the rollback drill record format.
const RollbackSchema = "tei.reference.order-migration-rollback-drill/v0"

// RollbackResult is the record of the rollback drill (step 6).
type RollbackResult struct {
	Schema       string         `json:"schema"`
	PlanDigest   string         `json:"planDigest"`
	GateDigest   string         `json:"gateDigest"`
	Reverse      ReverseResult  `json:"reverseSync"`
	ChangeFeed   ApplyResult    `json:"changeFeedDrain"`
	Reconcile    CrossReconcile `json:"reconcile"`
	Queues       Queues         `json:"queues"`
	ReceiptsGate int            `json:"legacyReceiptsWithGateDigest"`
	Blockers     []string       `json:"blockers"`
	Passed       bool           `json:"passed"`
	Digest       string         `json:"digest,omitempty"`
}

// RollbackDrill switches orders back to the legacy side (step 6). It freezes
// new-side writes, delivers every pending reverse-sync entry, drains the
// change feed (reverse-sync echoes only refresh the base; a legacy write
// made while the legacy side was not the authority is held and blocks), and
// reconciles both databases. Only a drill without differences, missing rows
// or open queues hands the authority and every customer field back to the
// legacy side. The drill is recorded with the plan and gate digests.
func RollbackDrill(ctx context.Context, legacy, primary *pgx.Conn, plan Plan) (RollbackResult, error) {
	res := RollbackResult{Schema: RollbackSchema, PlanDigest: plan.Digest, Blockers: []string{}}
	if _, err := CheckBound(ctx, primary, plan); err != nil {
		return res, err
	}
	gate, ok, err := passedGate(ctx, primary, plan)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, ErrNoPassedGate
	}
	res.GateDigest = gate.Digest
	if _, err := primary.Exec(ctx, `UPDATE omdd.mig_authority SET authority='rolling-back', changed_at=now() WHERE entity='order' AND authority='primary'`); err != nil {
		return res, err
	}
	if res.Reverse, err = ReverseSync(ctx, legacy, primary, plan); err != nil {
		return res, fmt.Errorf("reverse sync: %w", err)
	}
	if res.ChangeFeed, err = ApplyChanges(ctx, legacy, primary, plan); err != nil {
		return res, fmt.Errorf("drain change feed: %w", err)
	}
	err = readBoth(ctx, legacy, primary, plan, func(ltx, ptx pgx.Tx) error {
		var err error
		if res.Queues, err = readQueues(ctx, ptx); err != nil {
			return err
		}
		if res.Reconcile, err = compareLegacyPrimary(ctx, ltx, ptx, plan); err != nil {
			return err
		}
		return ltx.QueryRow(ctx, `SELECT count(*) FROM omdd.legacy_reverse_applied WHERE plan_digest=$1 AND gate_digest=$2`, plan.Digest, gate.Digest).Scan(&res.ReceiptsGate)
	})
	if err != nil {
		return res, err
	}
	res.Blockers = append(res.Blockers, res.Queues.blockers()...)
	for _, d := range res.Reconcile.OrderDifferences {
		res.Blockers = append(res.Blockers, "order-difference:"+d.Rule)
	}
	if n := len(res.Reconcile.MissingInLegacy); n > 0 {
		res.Blockers = append(res.Blockers, fmt.Sprintf("missing-in-legacy:%d", n))
	}
	if n := len(res.Reconcile.CustomerDifferences); n > 0 {
		res.Blockers = append(res.Blockers, fmt.Sprintf("customer-differences:%d", n))
	}
	res.Passed = len(res.Blockers) == 0
	if res.Digest, err = digestOf(res); err != nil {
		return res, err
	}
	tx, err := primary.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(context.Background())
	raw, err := json.Marshal(res)
	if err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.rollback_drill(drill_digest, plan_digest, gate_digest, passed, result) VALUES ($1,$2,$3,$4,$5)`,
		res.Digest, plan.Digest, gate.Digest, res.Passed, raw); err != nil {
		return res, err
	}
	if res.Passed {
		if _, err := tx.Exec(ctx, `UPDATE omdd.mig_authority SET authority='legacy', changed_at=now() WHERE entity='order' AND authority='rolling-back'`); err != nil {
			return res, err
		}
		for _, f := range CustomerFields() {
			if err := setOwner(ctx, tx, f, "legacy", gate.Digest, "rollback-drill:"+res.Digest); err != nil {
				return res, err
			}
		}
	}
	return res, tx.Commit(ctx)
}
