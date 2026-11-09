package ordermigrationdualdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// Dispositions of a decoded legacy transaction (omdd.cdc_tx).
const (
	DispositionApplied = "applied"
	DispositionVisible = "skipped-visible-in-snapshot"
	DispositionEcho    = "echo-of-reverse-sync"
	DispositionHeld    = "held"
)

// Reasons for holding a legacy change instead of applying it.
const (
	HeldOneToManyCode       = "one-to-many-item-code"
	HeldOrderAlreadyHeld    = "order-already-held"
	HeldOrderNotInPrimary   = "order-not-in-primary"
	HeldOrderDeleted        = "order-deleted"
	HeldLineAdded           = "line-added-to-migrated-order"
	HeldLineDeleted         = "line-deleted-from-migrated-order"
	HeldQtyOrItemChanged    = "quantity-or-item-changed-on-migrated-order"
	HeldCustomerDeleted     = "customer-deleted"
	HeldTruncate            = "truncate-not-applicable"
	HeldCodeMapChanged      = "item-code-map-changed-after-plan"
	HeldWriteAfterCutover   = "legacy-write-after-cutover"
	HeldUnknownLegacyTable  = "unknown-legacy-table"
	changeFeedPeekBatchSize = 2000
)

// ApplyResult reports one ApplyChanges call.
type ApplyResult struct {
	PlanDigest        string         `json:"planDigest"`
	SnapshotDigest    string         `json:"snapshotDigest"`
	Slot              string         `json:"slot"`
	FromLSN           string         `json:"fromLsn"`
	ThroughLSN        string         `json:"throughLsn"`
	Transactions      int            `json:"transactions"`
	AlreadyRecorded   int            `json:"alreadyRecorded"`
	Dispositions      map[string]int `json:"dispositions"`
	OrdersInserted    int            `json:"ordersInserted"`
	OrdersUpdated     int            `json:"ordersUpdated"`
	LinesUpdated      int            `json:"linesUpdated"`
	CustomersInserted int            `json:"customersInserted"`
	CustomersUpdated  int            `json:"customersUpdated"`
	BaseRefreshed     int            `json:"baseRefreshedByEcho"`
	Held              map[string]int `json:"held"`
	Conflicts         map[string]int `json:"conflicts"`
	Ignored           int            `json:"ignoredChanges"`
}

// ErrStep3Incomplete is returned when step 4 starts before every chunk of
// step 3 has a checkpoint. The legacy changes stay in the slot.
var ErrStep3Incomplete = errors.New("step 3 has not checkpointed every chunk")

// ApplyChanges is step 4's change feed (P-MIG-02). It reads the legacy
// changes from the slot created in step 2 and applies each committed legacy
// transaction to the primary in one primary transaction that also records
// the transaction in omdd.cdc_tx. A transaction already in the copy (its
// xid is visible in the step-2 pg_snapshot) is recorded and skipped; a
// reverse-sync transaction (it wrote omdd.legacy_sync_marker) only refreshes
// the customer base. The slot is advanced after the primary commit; a crash
// in between re-reads transactions that cdc_tx already holds, and they are
// not applied again.
func ApplyChanges(ctx context.Context, legacy, primary *pgx.Conn, plan Plan) (ApplyResult, error) {
	snap, err := CheckBound(ctx, primary, plan)
	if err != nil {
		return ApplyResult{}, err
	}
	feed := snap.Position.ChangeFeed
	res := ApplyResult{PlanDigest: plan.Digest, SnapshotDigest: snap.Digest, Slot: feed.Slot,
		Dispositions: map[string]int{}, Held: map[string]int{}, Conflicts: map[string]int{}}
	if feed.Slot == "" {
		return res, errors.New("snapshot record has no change feed slot")
	}
	pgSnap, err := ParsePGSnapshot(snap.Position.PGSnapshot)
	if err != nil {
		return res, err
	}
	if err := startStep4(ctx, primary, plan); err != nil {
		return res, err
	}
	last, err := lastRecordedLSN(ctx, primary, plan)
	if err != nil {
		return res, err
	}
	res.FromLSN = last.String()
	res.ThroughLSN = last.String()
	for {
		if err := requireSlot(ctx, legacy, feed.Slot); err != nil {
			return res, err
		}
		rows, err := peekChanges(ctx, legacy, feed.Slot)
		if err != nil {
			return res, err
		}
		txs, err := GroupTransactions(rows)
		if err != nil {
			return res, err
		}
		fresh := 0
		for _, t := range txs {
			if t.CommitLSN <= last {
				res.AlreadyRecorded++
				continue
			}
			fresh++
			disp, err := processLegacyTx(ctx, primary, plan, pgSnap, t, &res)
			if err != nil {
				return res, fmt.Errorf("legacy transaction %d at %s: %w", t.Xid, t.CommitLSN, err)
			}
			res.Transactions++
			res.Dispositions[disp]++
			last = t.CommitLSN
			res.ThroughLSN = last.String()
			crashPoint(CrashAfterChangeApplied)
		}
		if len(txs) == 0 {
			return res, nil
		}
		moved, err := advanceSlot(ctx, legacy, feed.Slot, txs[len(txs)-1].CommitLSN)
		if err != nil {
			return res, err
		}
		// A batch of only recorded transactions still moves the slot past
		// them; stop when neither new transactions nor the slot moved.
		if fresh == 0 && !moved {
			return res, nil
		}
	}
}

// startStep4 checks that step 3 finished and loads the customer master.
func startStep4(ctx context.Context, primary *pgx.Conn, plan Plan) error {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM omdd.mig_checkpoint WHERE plan_digest=$1`, plan.Digest).Scan(&n); err != nil {
		return err
	}
	if n != len(plan.Chunks) {
		return fmt.Errorf("%w: %d of %d", ErrStep3Incomplete, n, len(plan.Chunks))
	}
	if err := loadCustomerMaster(ctx, tx, plan); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func lastRecordedLSN(ctx context.Context, primary *pgx.Conn, plan Plan) (LSN, error) {
	var s *string
	if err := primary.QueryRow(ctx, `SELECT max(commit_lsn)::text FROM omdd.cdc_tx WHERE plan_digest=$1`, plan.Digest).Scan(&s); err != nil {
		return 0, err
	}
	if s == nil {
		return 0, nil
	}
	return ParseLSN(*s)
}

// ErrChangeFeedLost is returned when the slot is gone. Changes after its
// last confirmed position can no longer be captured; the gate's
// reconciliation shows what is missing and the affected keys must be copied
// again (not implemented).
var ErrChangeFeedLost = errors.New("change feed slot is missing")

func requireSlot(ctx context.Context, legacy *pgx.Conn, slot string) error {
	var ok bool
	if err := legacy.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1 AND database=current_database())`, slot).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s", ErrChangeFeedLost, slot)
	}
	return nil
}

func peekChanges(ctx context.Context, legacy *pgx.Conn, slot string) ([]DecodedRow, error) {
	rows, err := legacy.Query(ctx, `SELECT lsn::text, data FROM pg_logical_slot_peek_changes($1, NULL, $2, 'include-xids', '1', 'skip-empty-xacts', '1')`, slot, changeFeedPeekBatchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DecodedRow
	for rows.Next() {
		var r DecodedRow
		if err := rows.Scan(&r.LSN, &r.Data); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func advanceSlot(ctx context.Context, legacy *pgx.Conn, slot string, to LSN) (bool, error) {
	var confirmed string
	if err := legacy.QueryRow(ctx, `SELECT confirmed_flush_lsn::text FROM pg_replication_slots WHERE slot_name=$1`, slot).Scan(&confirmed); err != nil {
		return false, err
	}
	c, err := ParseLSN(confirmed)
	if err != nil {
		return false, err
	}
	if to <= c {
		return false, nil
	}
	_, err = legacy.Exec(ctx, `SELECT pg_replication_slot_advance($1, $2::pg_lsn)`, slot, to.String())
	return err == nil, err
}

// orderChanges collects the decoded changes of one order key in one legacy
// transaction, in decoded order.
type orderChanges struct {
	key     string
	changes []RowChange
}

func processLegacyTx(ctx context.Context, primary *pgx.Conn, plan Plan, pgSnap PGSnapshot, t DecodedTx, res *ApplyResult) (string, error) {
	tx, err := primary.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SET LOCAL omdd.origin = 'legacy-cdc'`); err != nil {
		return "", err
	}
	xid64 := pgSnap.Extend(t.Xid)
	disp := DispositionApplied
	record := func() error {
		_, err := tx.Exec(ctx, `INSERT INTO omdd.cdc_tx(commit_lsn, xid, plan_digest, disposition, changes) VALUES ($1::pg_lsn,$2,$3,$4,$5)`,
			t.CommitLSN.String(), int64(xid64), plan.Digest, disp, len(t.Changes))
		return err
	}
	if pgSnap.Visible(xid64) {
		disp = DispositionVisible
		if err := record(); err != nil {
			return "", err
		}
		return disp, tx.Commit(ctx)
	}
	echo := false
	for _, ch := range t.Changes {
		echo = echo || ch.Table == "omdd.legacy_sync_marker"
	}
	if echo {
		disp = DispositionEcho
		for _, ch := range t.Changes {
			if ch.Table != "omdd.legacy_customer" || ch.Op == "DELETE" || ch.Op == "TRUNCATE" {
				continue
			}
			c, err := customerFromChange(ch)
			if err != nil {
				return "", err
			}
			if _, err := tx.Exec(ctx, `UPDATE omdd.customer_legacy_base SET customer_name=$2, address=$3, credit_limit=$4::text::numeric WHERE customer_code=$1`,
				c.Code, c.Name, c.Address, c.CreditLimit); err != nil {
				return "", err
			}
			res.BaseRefreshed++
		}
		if err := record(); err != nil {
			return "", err
		}
		return disp, tx.Commit(ctx)
	}

	auth, _, err := orderAuthority(ctx, tx)
	if err != nil {
		return "", err
	}
	owners, err := fieldOwners(ctx, tx)
	if err != nil {
		return "", err
	}
	held, applied := 0, 0
	hold := func(entity, key, reason string, change any) error {
		raw, err := json.Marshal(change)
		if err != nil {
			return err
		}
		res.Held[reason]++
		held++
		_, err = tx.Exec(ctx, `INSERT INTO omdd.cdc_held(plan_digest, commit_lsn, entity, entity_key, reason, change) VALUES ($1,$2::pg_lsn,$3,$4,$5,$6)`,
			plan.Digest, t.CommitLSN.String(), entity, key, reason, raw)
		return err
	}
	var orders []*orderChanges
	byKey := map[string]*orderChanges{}
	for _, ch := range t.Changes {
		switch ch.Table {
		case "omdd.legacy_change_log", "omdd.legacy_reverse_applied":
			res.Ignored++
			continue
		case "omdd.legacy_item_code_map":
			if err := hold("code-map", "", HeldCodeMapChanged, ch); err != nil {
				return "", err
			}
			continue
		case "omdd.legacy_order", "omdd.legacy_order_line", "omdd.legacy_customer":
		default:
			if err := hold("table", ch.Table, HeldUnknownLegacyTable, ch); err != nil {
				return "", err
			}
			continue
		}
		if ch.Op == "TRUNCATE" {
			if err := hold("table", ch.Table, HeldTruncate, ch); err != nil {
				return "", err
			}
			continue
		}
		if auth != "legacy" {
			key := ""
			if c, ok := ch.Get("order_no"); ok {
				key = c.Value
			} else if c, ok := ch.Get("customer_code"); ok {
				key = c.Value
			}
			if err := hold("write-after-cutover", key, HeldWriteAfterCutover, ch); err != nil {
				return "", err
			}
			continue
		}
		if ch.Table == "omdd.legacy_customer" {
			if ch.Op == "DELETE" {
				c, _ := ch.Get("customer_code")
				if err := hold("customer", c.Value, HeldCustomerDeleted, ch); err != nil {
					return "", err
				}
				continue
			}
			c, err := customerFromChange(ch)
			if err != nil {
				return "", err
			}
			if err := applyLegacyCustomer(ctx, tx, plan, t.CommitLSN, c, owners, res); err != nil {
				return "", err
			}
			if _, err := tx.Exec(ctx, `UPDATE omdd.schema_client SET last_seen_at=now() WHERE client_id='legacy-cdc'`); err != nil {
				return "", err
			}
			applied++
			continue
		}
		k, ok := ch.Get("order_no")
		if !ok {
			return "", fmt.Errorf("%s change without order_no", ch.Table)
		}
		oc := byKey[k.Value]
		if oc == nil {
			oc = &orderChanges{key: k.Value}
			byKey[k.Value] = oc
			orders = append(orders, oc)
		}
		oc.changes = append(oc.changes, ch)
	}
	for _, oc := range orders {
		a, err := applyOrderChanges(ctx, tx, plan, oc, res, hold)
		if err != nil {
			return "", fmt.Errorf("order %s: %w", oc.key, err)
		}
		applied += a
	}
	if applied == 0 && held > 0 {
		disp = DispositionHeld
	}
	if err := record(); err != nil {
		return "", err
	}
	return disp, tx.Commit(ctx)
}

type holdFunc func(entity, key, reason string, change any) error

// applyOrderChanges applies the changes of one order key and returns how
// many changes were applied.
func applyOrderChanges(ctx context.Context, tx pgx.Tx, plan Plan, oc *orderChanges, res *ApplyResult, hold holdFunc) (int, error) {
	var heldBefore bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM omdd.cdc_held WHERE entity='order' AND entity_key=$1 AND state='held')`, oc.key).Scan(&heldBefore); err != nil {
		return 0, err
	}
	if heldBefore {
		return 0, hold("order", oc.key, HeldOrderAlreadyHeld, oc.changes)
	}
	existing, err := loadOrders(ctx, tx, primaryTables, "", keyRange{exact: oc.key})
	if err != nil {
		return 0, err
	}
	if len(existing) == 0 {
		return applyNewOrder(ctx, tx, plan, oc, res, hold)
	}
	cur := existing[0]
	lines := map[int]Line{}
	for _, l := range cur.Lines {
		lines[l.LineNo] = l
	}
	applied := 0
	for _, ch := range oc.changes {
		if ch.Table == "omdd.legacy_order" {
			if ch.Op == "DELETE" {
				if err := hold("order", oc.key, HeldOrderDeleted, ch); err != nil {
					return applied, err
				}
				continue
			}
			o, err := orderHeaderFromChange(ch)
			if err != nil {
				return applied, err
			}
			if o.CustomerCode != cur.CustomerCode || o.OrderedOn != cur.OrderedOn || o.Status != cur.Status {
				if _, err := tx.Exec(ctx, `UPDATE omdd.order_backlog SET customer_code=$2, ordered_on=$3::text::date, status=$4 WHERE order_no=$1`,
					oc.key, o.CustomerCode, o.OrderedOn, o.Status); err != nil {
					return applied, err
				}
				cur.CustomerCode, cur.OrderedOn, cur.Status = o.CustomerCode, o.OrderedOn, o.Status
				res.OrdersUpdated++
			}
			applied++
			continue
		}
		if ch.Op == "DELETE" {
			if err := hold("order", oc.key, HeldLineDeleted, ch); err != nil {
				return applied, err
			}
			continue
		}
		l, err := lineFromChange(ch)
		if err != nil {
			return applied, err
		}
		code, ambiguous, err := newCodeFor(ctx, tx, plan, oc.key, l.LineNo, l.ItemCode)
		if err != nil {
			return applied, err
		}
		if ambiguous {
			if err := hold("order", oc.key, HeldOneToManyCode, ch); err != nil {
				return applied, err
			}
			continue
		}
		have, ok := lines[l.LineNo]
		if !ok {
			if err := hold("order", oc.key, HeldLineAdded, ch); err != nil {
				return applied, err
			}
			continue
		}
		if have.ItemCode != code || have.Qty != l.Qty {
			// The reservation was made for the old quantity and item; a
			// change needs a new reservation, which this sample does not do.
			if err := hold("order", oc.key, HeldQtyOrItemChanged, ch); err != nil {
				return applied, err
			}
			continue
		}
		if have.UnitPrice != l.UnitPrice || have.Amount != l.Amount {
			if _, err := tx.Exec(ctx, `UPDATE omdd.order_backlog_line SET unit_price=$3::text::numeric, amount=$4::text::numeric WHERE order_no=$1 AND line_no=$2`,
				oc.key, l.LineNo, l.UnitPrice, l.Amount); err != nil {
				return applied, err
			}
			have.UnitPrice, have.Amount = l.UnitPrice, l.Amount
			lines[l.LineNo] = have
			res.LinesUpdated++
		}
		applied++
	}
	return applied, nil
}

// applyNewOrder inserts an order created on the legacy side after the copy.
// The whole order is held when a line's item code is one-to-many.
func applyNewOrder(ctx context.Context, tx pgx.Tx, plan Plan, oc *orderChanges, res *ApplyResult, hold holdFunc) (int, error) {
	var header *Order
	lines := map[int]Line{}
	for _, ch := range oc.changes {
		switch {
		case ch.Table == "omdd.legacy_order" && ch.Op == "DELETE":
			header = nil
		case ch.Table == "omdd.legacy_order":
			o, err := orderHeaderFromChange(ch)
			if err != nil {
				return 0, err
			}
			header = &o
		case ch.Op == "DELETE":
			l, _ := ch.Get("line_no")
			n, _ := strconv.Atoi(l.Value)
			delete(lines, n)
		default:
			l, err := lineFromChange(ch)
			if err != nil {
				return 0, err
			}
			lines[l.LineNo] = l
		}
	}
	if header == nil {
		return 0, hold("order", oc.key, HeldOrderNotInPrimary, oc.changes)
	}
	o := *header
	nos := make([]int, 0, len(lines))
	for n := range lines {
		nos = append(nos, n)
	}
	sort.Ints(nos)
	for _, n := range nos {
		l := lines[n]
		code, ambiguous, err := newCodeFor(ctx, tx, plan, o.OrderNo, l.LineNo, l.ItemCode)
		if err != nil {
			return 0, err
		}
		if ambiguous {
			return 0, hold("order", oc.key, HeldOneToManyCode, oc.changes)
		}
		l.ItemCode = code
		o.Lines = append(o.Lines, l)
	}
	if _, err := writeOrder(ctx, tx, plan, plan.ChunkFor(o.OrderNo), o); err != nil {
		return 0, err
	}
	res.OrdersInserted++
	return len(oc.changes), nil
}

// newCodeFor converts a legacy item code with the plan's map. A one-to-many
// code converts only with a recorded decision for that order line.
func newCodeFor(ctx context.Context, tx pgx.Tx, plan Plan, orderNo string, lineNo int, legacyCode string) (string, bool, error) {
	fw := plan.ForwardCodes(legacyCode)
	switch len(fw) {
	case 0:
		return legacyCode, false, nil
	case 1:
		return fw[0], false, nil
	}
	var decided string
	err := tx.QueryRow(ctx, `SELECT new_code FROM omdd.code_decision WHERE order_no=$1 AND line_no=$2 AND legacy_code=$3`, orderNo, lineNo, legacyCode).Scan(&decided)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", true, nil
	}
	return decided, false, err
}

func colValue(ch RowChange, name string) (string, error) {
	c, ok := ch.Get(name)
	if !ok {
		return "", fmt.Errorf("%s %s: column %s missing", ch.Table, ch.Op, name)
	}
	if c.Null {
		return "", fmt.Errorf("%s %s: column %s is null", ch.Table, ch.Op, name)
	}
	return c.Value, nil
}

func orderHeaderFromChange(ch RowChange) (Order, error) {
	var o Order
	var err error
	for _, f := range []struct {
		dst  *string
		name string
	}{{&o.OrderNo, "order_no"}, {&o.CustomerCode, "customer_code"}, {&o.OrderedOn, "ordered_on"}, {&o.Status, "status"}} {
		if *f.dst, err = colValue(ch, f.name); err != nil {
			return Order{}, err
		}
	}
	return o, nil
}

func lineFromChange(ch RowChange) (Line, error) {
	var l Line
	vals := map[string]string{}
	for _, n := range []string{"line_no", "item_code", "qty", "unit_price", "amount"} {
		v, err := colValue(ch, n)
		if err != nil {
			return Line{}, err
		}
		vals[n] = v
	}
	var err error
	if l.LineNo, err = strconv.Atoi(vals["line_no"]); err != nil {
		return Line{}, err
	}
	if l.Qty, err = strconv.ParseInt(vals["qty"], 10, 64); err != nil {
		return Line{}, err
	}
	l.ItemCode, l.UnitPrice, l.Amount = vals["item_code"], vals["unit_price"], vals["amount"]
	return l, nil
}

func customerFromChange(ch RowChange) (Customer, error) {
	var c Customer
	var err error
	for _, f := range []struct {
		dst  *string
		name string
	}{{&c.Code, "customer_code"}, {&c.Name, "customer_name"}, {&c.Address, "address"}, {&c.CreditLimit, "credit_limit"}} {
		if *f.dst, err = colValue(ch, f.name); err != nil {
			return Customer{}, err
		}
	}
	return c, nil
}

// ResolveHeldOrder is a person's decision on an order held for a one-to-many
// item code. choices gives the new code for each ambiguous line. The order
// is read again from the legacy database (its current state), converted
// with the map and the decisions, and inserted with its reservation effect.
func ResolveHeldOrder(ctx context.Context, legacy, primary *pgx.Conn, plan Plan, orderNo string, choices map[int]string, decidedBy string) error {
	cur, err := loadOrders(ctx, legacy, legacyTables, "", keyRange{exact: orderNo})
	if err != nil {
		return err
	}
	if len(cur) != 1 {
		return fmt.Errorf("order %s is not in the legacy database", orderNo)
	}
	tx, err := primary.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL omdd.origin = 'legacy-cdc'`); err != nil {
		return err
	}
	o := cur[0]
	for i, l := range o.Lines {
		fw := plan.ForwardCodes(l.ItemCode)
		if len(fw) <= 1 {
			if len(fw) == 1 {
				o.Lines[i].ItemCode = fw[0]
			}
			continue
		}
		choice, ok := choices[l.LineNo]
		valid := false
		for _, c := range fw {
			valid = valid || c == choice
		}
		if !ok || !valid {
			return fmt.Errorf("line %d: legacy code %s needs one of %v, got %q", l.LineNo, l.ItemCode, fw, choice)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.code_decision(order_no, line_no, legacy_code, new_code, decided_by) VALUES ($1,$2,$3,$4,$5)`,
			orderNo, l.LineNo, l.ItemCode, choice, decidedBy); err != nil {
			return err
		}
		o.Lines[i].ItemCode = choice
	}
	written, err := writeOrder(ctx, tx, plan, plan.ChunkFor(orderNo), o)
	if err != nil {
		return err
	}
	if !written {
		return fmt.Errorf("order %s is already in the primary database", orderNo)
	}
	if _, err := tx.Exec(ctx, `UPDATE omdd.cdc_held SET state='resolved', resolution=$2, resolved_at=now() WHERE entity='order' AND entity_key=$1 AND state='held'`,
		orderNo, "inserted-with-code-decision-by:"+decidedBy); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
