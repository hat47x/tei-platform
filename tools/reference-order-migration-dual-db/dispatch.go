package ordermigrationdualdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// DispatchOptions bounds every call into the inventory database. The bounds
// are the sample's stop policy for the inventory reference: keep the effect
// pending and record why, never treat "could not ask" as "no stock".
type DispatchOptions struct {
	ConnectTimeout time.Duration
	OpTimeout      time.Duration
}

// DefaultDispatchOptions are the bounds used by the CLI and the tests.
func DefaultDispatchOptions() DispatchOptions {
	return DispatchOptions{ConnectTimeout: 3 * time.Second, OpTimeout: 3 * time.Second}
}

// Outcomes recorded in effect_outbox.last_outcome.
const (
	OutcomeReserved    = "reserved"
	OutcomeRejected    = "rejected"
	OutcomeUnavailable = "inventory-unavailable" // not sent: connection failed
	OutcomeUnknown     = "unknown"               // sent, result lost; redelivery is deduplicated
	OutcomeNotAttempt  = "not-attempted-circuit-open"
)

// DispatchResult reports one dispatch pass.
type DispatchResult struct {
	Pending            int     `json:"pending"`
	InventoryReachable bool    `json:"inventoryReachable"`
	Reserved           int     `json:"reserved"`
	Rejected           int     `json:"rejected"`
	Replayed           int     `json:"replayed"`
	Unknown            int     `json:"unknown"`
	LeftPending        int     `json:"leftPending"`
	Compensated        int     `json:"compensated"`
	ElapsedSeconds     float64 `json:"elapsedSeconds"`
	Error              string  `json:"error,omitempty"`
}

// beforeSendHook is an in-process test hook (nil in the CLI). The P-DB-02
// test uses it to stop the inventory database in the middle of a pass.
var beforeSendHook func(index int)

type outboxEntry struct {
	effectID, orderNo string
	payload           ReservationPayload
}

// Dispatch delivers pending outbox effects to the inventory database and
// then runs compensation for definitively rejected ones. Inventory failures
// are recorded on the outbox and returned in the result, not as an error.
func Dispatch(ctx context.Context, primary *pgx.Conn, inventoryDSN string, opt DispatchOptions) (DispatchResult, error) {
	start := time.Now()
	res, err := dispatch(ctx, primary, inventoryDSN, opt)
	if err == nil {
		var n int
		n, err = Compensate(ctx, primary)
		res.Compensated = n
	}
	res.ElapsedSeconds = time.Since(start).Seconds()
	return res, err
}

func dispatch(ctx context.Context, primary *pgx.Conn, inventoryDSN string, opt DispatchOptions) (DispatchResult, error) {
	var res DispatchResult
	entries, err := pendingEntries(ctx, primary)
	if err != nil {
		return res, err
	}
	res.Pending = len(entries)
	if len(entries) == 0 {
		return res, nil
	}
	cctx, cancel := context.WithTimeout(ctx, opt.ConnectTimeout)
	inv, err := Connect(cctx, inventoryDSN, RoleInventory, opt.ConnectTimeout)
	cancel()
	if err != nil {
		res.Error = err.Error()
		res.LeftPending = len(entries)
		return res, markPending(ctx, primary, entries, OutcomeUnavailable, err)
	}
	defer inv.Close(context.Background())
	res.InventoryReachable = true
	for i, e := range entries {
		crashPoint(CrashBeforeSend)
		if beforeSendHook != nil {
			beforeSendHook(i)
		}
		octx, cancel := context.WithTimeout(ctx, opt.OpTimeout)
		outcome, reason, replayed, err := applyReservation(octx, inv, e)
		cancel()
		if err != nil {
			// The request may or may not have committed. Keep it pending as
			// unknown; the receiver deduplicates the redelivery. Stop this
			// pass (circuit open) so the primary side stays bounded.
			res.Error = err.Error()
			res.Unknown++
			res.LeftPending = len(entries) - i
			if merr := markPending(ctx, primary, entries[i:i+1], OutcomeUnknown, err); merr != nil {
				return res, merr
			}
			return res, markPending(ctx, primary, entries[i+1:], OutcomeNotAttempt, err)
		}
		if replayed {
			res.Replayed++
		}
		crashPoint(CrashAfterInventoryCommit)
		if err := acknowledge(ctx, primary, e, outcome, reason); err != nil {
			return res, err
		}
		switch outcome {
		case OutcomeReserved:
			res.Reserved++
		case OutcomeRejected:
			res.Rejected++
			crashPoint(CrashBeforeCompensation)
		}
	}
	return res, nil
}

func pendingEntries(ctx context.Context, primary *pgx.Conn) ([]outboxEntry, error) {
	rows, err := primary.Query(ctx, `SELECT effect_id, order_no, payload FROM omdd.effect_outbox WHERE state='pending' ORDER BY order_no COLLATE "C"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []outboxEntry
	for rows.Next() {
		var e outboxEntry
		var raw []byte
		if err := rows.Scan(&e.effectID, &e.orderNo, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &e.payload); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func markPending(ctx context.Context, primary *pgx.Conn, entries []outboxEntry, outcome string, cause error) error {
	if len(entries) == 0 {
		return nil
	}
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.effectID
	}
	_, err := primary.Exec(ctx, `UPDATE omdd.effect_outbox SET attempts = attempts + CASE WHEN $2 = $4 THEN 0 ELSE 1 END,
		last_outcome=$2, last_error=$3, last_attempt_at=now() WHERE effect_id = ANY($1) AND state='pending'`,
		ids, outcome, truncate(cause.Error(), 300), OutcomeNotAttempt)
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// applyReservation is the inventory side. One order's lines are reserved in
// one inventory transaction, deduplicated by effect identifier. A shortage is
// recorded durably as a rejection so a late redelivery cannot reserve later.
func applyReservation(ctx context.Context, inv *pgx.Conn, e outboxEntry) (string, string, bool, error) {
	if outcome, reason, ok, err := appliedOutcome(ctx, inv, e.effectID); err != nil || ok {
		return outcome, reason, ok, err
	}
	tx, err := inv.Begin(ctx)
	if err != nil {
		return "", "", false, err
	}
	defer tx.Rollback(context.Background())
	shortage := ""
	for _, l := range e.payload.Lines {
		tag, err := tx.Exec(ctx, `UPDATE omdd.stock SET reserved = reserved + $2 WHERE item_code=$1 AND on_hand - reserved >= $2`, l.ItemCode, l.Qty)
		if err != nil {
			return "", "", false, err
		}
		if tag.RowsAffected() == 0 {
			var known bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM omdd.stock WHERE item_code=$1)`, l.ItemCode).Scan(&known); err != nil {
				return "", "", false, err
			}
			shortage = fmt.Sprintf("insufficient-stock:%s:line-%d", l.ItemCode, l.LineNo)
			if !known {
				shortage = fmt.Sprintf("unknown-item:%s:line-%d", l.ItemCode, l.LineNo)
			}
			break
		}
	}
	if shortage != "" {
		if err := tx.Rollback(ctx); err != nil {
			return "", "", false, err
		}
		if _, err := inv.Exec(ctx, `INSERT INTO omdd.applied_effect(effect_id, order_no, outcome, reason) VALUES ($1,$2,'rejected',$3) ON CONFLICT (effect_id) DO NOTHING`,
			e.effectID, e.orderNo, shortage); err != nil {
			return "", "", false, err
		}
		outcome, reason, _, err := appliedOutcome(ctx, inv, e.effectID)
		return outcome, reason, false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.applied_effect(effect_id, order_no, outcome) VALUES ($1,$2,'reserved')`, e.effectID, e.orderNo); err != nil {
		return "", "", false, err
	}
	for _, l := range e.payload.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.allocation(effect_id, line_no, order_no, item_code, qty) VALUES ($1,$2,$3,$4,$5)`,
			e.effectID, l.LineNo, e.orderNo, l.ItemCode, l.Qty); err != nil {
			return "", "", false, err
		}
	}
	return OutcomeReserved, "", false, tx.Commit(ctx)
}

func appliedOutcome(ctx context.Context, inv *pgx.Conn, effectID string) (string, string, bool, error) {
	var outcome string
	var reason *string
	err := inv.QueryRow(ctx, `SELECT outcome, reason FROM omdd.applied_effect WHERE effect_id=$1`, effectID).Scan(&outcome, &reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	r := ""
	if reason != nil {
		r = *reason
	}
	return outcome, r, true, nil
}

// acknowledge records the inventory outcome on the primary side. It refuses
// an acknowledgement that contradicts the current business state, e.g. a
// reservation arriving after the order was compensated.
func acknowledge(ctx context.Context, primary *pgx.Conn, e outboxEntry, outcome, reason string) error {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	state, target := "acknowledged", "allocated"
	if outcome == OutcomeRejected {
		state, target = "rejected", "rejected"
	} else if outcome != OutcomeReserved {
		return fmt.Errorf("effect %s: unexpected inventory outcome %q", e.effectID, outcome)
	}
	if _, err := tx.Exec(ctx, `UPDATE omdd.effect_outbox SET state=$2, last_outcome=$3, last_error=NULLIF($4,''), attempts=attempts+1, last_attempt_at=now(), acknowledged_at=now()
		WHERE effect_id=$1 AND state='pending'`, e.effectID, state, outcome, reason); err != nil {
		return err
	}
	var current string
	if err := tx.QueryRow(ctx, `SELECT allocation_state FROM omdd.order_backlog WHERE order_no=$1 FOR UPDATE`, e.orderNo).Scan(&current); err != nil {
		return err
	}
	switch {
	case current == target:
	case current == "compensated" && target == "rejected":
		// A redelivered rejection after compensation changes nothing.
	case current == "awaiting":
		if _, err := tx.Exec(ctx, `UPDATE omdd.order_backlog SET allocation_state=$2 WHERE order_no=$1`, e.orderNo, target); err != nil {
			return err
		}
	default:
		return fmt.Errorf("effect %s: inventory outcome %q contradicts primary state %q", e.effectID, outcome, current)
	}
	return tx.Commit(ctx)
}

// Compensate closes every definitively rejected allocation. The compensation
// is its own operation with its own identifier, recorded once; the migrated
// backlog row stays (it is migrated data) and is marked compensated, i.e.
// held without stock. "Unknown" outcomes are never compensated.
func Compensate(ctx context.Context, primary *pgx.Conn) (int, error) {
	rows, err := primary.Query(ctx, `SELECT o.effect_id, o.operation_id, o.order_no, coalesce(o.last_error,'')
		FROM omdd.effect_outbox o JOIN omdd.order_backlog b USING (order_no)
		WHERE o.state='rejected' AND b.allocation_state='rejected' ORDER BY o.order_no COLLATE "C"`)
	if err != nil {
		return 0, err
	}
	type item struct{ effectID, opID, orderNo, reason string }
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.effectID, &it.opID, &it.orderNo, &it.reason); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	n := 0
	for _, it := range items {
		tx, err := primary.Begin(ctx)
		if err != nil {
			return n, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.compensation(compensation_id, compensates_operation_id, effect_id, order_no, reason)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (effect_id) DO NOTHING`, "compensate:"+it.opID, it.opID, it.effectID, it.orderNo, it.reason); err != nil {
			tx.Rollback(context.Background())
			return n, err
		}
		if _, err := tx.Exec(ctx, `UPDATE omdd.order_backlog SET allocation_state='compensated' WHERE order_no=$1 AND allocation_state='rejected'`, it.orderNo); err != nil {
			tx.Rollback(context.Background())
			return n, err
		}
		if err := tx.Commit(ctx); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// RunResult is one call of Run: step 3's migration followed by dispatch.
type RunResult struct {
	Migrate        MigrateResult  `json:"migrate"`
	Dispatch       DispatchResult `json:"dispatch"`
	ElapsedSeconds float64        `json:"elapsedSeconds"`
}

// Run migrates every chunk and then makes one dispatch pass.
func Run(ctx context.Context, primary *pgx.Conn, inventoryDSN string, plan Plan, opt DispatchOptions) (RunResult, error) {
	start := time.Now()
	var res RunResult
	var err error
	res.Migrate, err = Migrate(ctx, primary, plan)
	if err == nil {
		res.Dispatch, err = Dispatch(ctx, primary, inventoryDSN, opt)
	}
	res.ElapsedSeconds = time.Since(start).Seconds()
	return res, err
}
