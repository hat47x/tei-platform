package ordermigrationdualdb

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

// Classes of the cross-database state of one migrated order. Only the first
// two are settled. "undetermined:*" means one database has committed and the
// other has not (yet); it is never folded into success or failure.
const (
	ClassBothCommitted             = "both-committed"
	ClassCompensated               = "compensated"
	ClassPrimaryOnlyAwaiting       = "undetermined:primary-only-awaiting-allocation"
	ClassInventoryOnlyUnacked      = "undetermined:inventory-reserved-primary-unacknowledged"
	ClassRejectedUnacked           = "undetermined:inventory-rejected-primary-unacknowledged"
	ClassCompensationPending       = "undetermined:rejected-compensation-pending"
	ClassInventoryUnobserved       = "undetermined:inventory-unobserved"
	ClassInconsistent              = "inconsistent"
	maxListedUndeterminedPerReport = 50
)

// StatusReport observes both databases.
type StatusReport struct {
	PlanDigest         string              `json:"planDigest"`
	ChunksPlanned      int                 `json:"chunksPlanned"`
	ChunksCheckpointed int                 `json:"chunksCheckpointed"`
	Orders             int                 `json:"orders"`
	PrimaryStates      map[string]int      `json:"primaryStates"`
	OutboxStates       map[string]int      `json:"outboxStates"`
	AwaitingByOutcome  map[string]int      `json:"awaitingByLastOutcome"`
	InventoryObserved  bool                `json:"inventoryObserved"`
	InventoryError     string              `json:"inventoryError,omitempty"`
	Classes            map[string]int      `json:"classes"`
	Undetermined       map[string][]string `json:"undetermined"`
	// Inventory-side totals.
	AppliedReserved      int   `json:"appliedReserved"`
	AppliedRejected      int   `json:"appliedRejected"`
	AllocationEffects    int   `json:"allocationEffects"`
	AllocationRows       int   `json:"allocationRows"`
	AllocationQty        int64 `json:"allocationQty"`
	StockReserved        int64 `json:"stockReserved"`
	StockOnHand          int64 `json:"stockOnHand"`
	PrimaryAllocatedQty  int64 `json:"primaryAllocatedQty"`
	PrimaryAllocatedRows int   `json:"primaryAllocatedLines"`
	// Duplicates counts allocation rows whose effect is unknown to the
	// primary outbox or whose line set differs from the outbox payload.
	Duplicates        int `json:"duplicates"`
	Compensations     int `json:"compensations"`
	OrphanAllocations int `json:"orphanAllocations"`
}

// Settled reports whether every order reached both-committed or compensated.
func (s StatusReport) Settled() bool {
	return s.InventoryObserved && s.Classes[ClassBothCommitted]+s.Classes[ClassCompensated] == s.Orders
}

// Status observes the primary and, if reachable within the bounds, the
// inventory database.
func Status(ctx context.Context, primary *pgx.Conn, inventoryDSN string, plan Plan, opt DispatchOptions) (StatusReport, error) {
	r := StatusReport{
		PlanDigest: plan.Digest, ChunksPlanned: len(plan.Chunks),
		PrimaryStates: map[string]int{}, OutboxStates: map[string]int{}, AwaitingByOutcome: map[string]int{},
		Classes: map[string]int{}, Undetermined: map[string][]string{},
	}
	if err := primary.QueryRow(ctx, `SELECT count(*) FROM omdd.mig_checkpoint WHERE plan_digest=$1`, plan.Digest).Scan(&r.ChunksCheckpointed); err != nil {
		return r, err
	}
	if err := primary.QueryRow(ctx, `SELECT count(*) FROM omdd.compensation`).Scan(&r.Compensations); err != nil {
		return r, err
	}
	if err := primary.QueryRow(ctx, `SELECT count(*), coalesce(sum(l.qty),0) FROM omdd.order_backlog_line l JOIN omdd.order_backlog b USING (order_no) WHERE b.allocation_state='allocated'`).
		Scan(&r.PrimaryAllocatedRows, &r.PrimaryAllocatedQty); err != nil {
		return r, err
	}
	type prim struct {
		state, effectID, outboxState, lastOutcome string
		lines                                     int
	}
	byOrder := map[string]prim{}
	effects := map[string]string{}
	rows, err := primary.Query(ctx, `SELECT b.order_no, b.allocation_state, o.effect_id, o.state, coalesce(o.last_outcome,''),
		(SELECT count(*) FROM omdd.order_backlog_line l WHERE l.order_no=b.order_no)
		FROM omdd.order_backlog b JOIN omdd.effect_outbox o USING (order_no)`)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var key string
		var p prim
		if err := rows.Scan(&key, &p.state, &p.effectID, &p.outboxState, &p.lastOutcome, &p.lines); err != nil {
			rows.Close()
			return r, err
		}
		byOrder[key] = p
		effects[p.effectID] = key
		r.PrimaryStates[p.state]++
		r.OutboxStates[p.outboxState]++
		if p.state == "awaiting" {
			r.AwaitingByOutcome[p.lastOutcome]++
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return r, err
	}
	var backlog int
	if err := primary.QueryRow(ctx, `SELECT count(*) FROM omdd.order_backlog`).Scan(&backlog); err != nil {
		return r, err
	}
	r.Orders = backlog
	if backlog != len(byOrder) {
		// A backlog row without an outbox effect would be an allocation that
		// can never be delivered.
		r.Classes[ClassInconsistent] += backlog - len(byOrder)
	}

	cctx, cancel := context.WithTimeout(ctx, opt.ConnectTimeout)
	inv, err := Connect(cctx, inventoryDSN, RoleInventory, opt.ConnectTimeout)
	cancel()
	inventory := map[string]string{}
	allocLines := map[string]int{}
	if err == nil {
		defer inv.Close(context.Background())
		octx, cancel := context.WithTimeout(ctx, opt.OpTimeout)
		err = observeInventory(octx, inv, &r, inventory, allocLines)
		cancel()
	}
	if err != nil {
		r.InventoryError = err.Error()
		for key := range byOrder {
			r.Classes[ClassInventoryUnobserved]++
			r.addUndetermined(ClassInventoryUnobserved, key)
		}
		return r, nil
	}
	r.InventoryObserved = true
	for eff, n := range allocLines {
		key, ok := effects[eff]
		if !ok {
			r.OrphanAllocations++
			r.Duplicates += n
			continue
		}
		if n != byOrder[key].lines {
			r.Duplicates++
		}
	}
	for key, p := range byOrder {
		inv := inventory[p.effectID]
		class := ClassInconsistent
		switch {
		case p.state == "allocated" && inv == OutcomeReserved:
			class = ClassBothCommitted
		case p.state == "compensated" && inv == OutcomeRejected:
			class = ClassCompensated
		case p.state == "awaiting" && inv == "":
			class = ClassPrimaryOnlyAwaiting
		case p.state == "awaiting" && inv == OutcomeReserved:
			class = ClassInventoryOnlyUnacked
		case p.state == "awaiting" && inv == OutcomeRejected:
			class = ClassRejectedUnacked
		case p.state == "rejected" && inv == OutcomeRejected:
			class = ClassCompensationPending
		}
		r.Classes[class]++
		if class != ClassBothCommitted && class != ClassCompensated {
			r.addUndetermined(class, key)
		}
	}
	for _, keys := range r.Undetermined {
		sort.Strings(keys)
	}
	return r, nil
}

func (r *StatusReport) addUndetermined(class, key string) {
	if len(r.Undetermined[class]) < maxListedUndeterminedPerReport {
		r.Undetermined[class] = append(r.Undetermined[class], key)
	}
}

func observeInventory(ctx context.Context, inv *pgx.Conn, r *StatusReport, outcomes map[string]string, allocLines map[string]int) error {
	rows, err := inv.Query(ctx, `SELECT effect_id, outcome FROM omdd.applied_effect`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, outcome string
		if err := rows.Scan(&id, &outcome); err != nil {
			rows.Close()
			return err
		}
		outcomes[id] = outcome
		if outcome == OutcomeReserved {
			r.AppliedReserved++
		} else {
			r.AppliedRejected++
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	rows, err = inv.Query(ctx, `SELECT effect_id, count(*), sum(qty) FROM omdd.allocation GROUP BY effect_id`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id string
		var n int
		var qty int64
		if err := rows.Scan(&id, &n, &qty); err != nil {
			rows.Close()
			return err
		}
		allocLines[id] = n
		r.AllocationEffects++
		r.AllocationRows += n
		r.AllocationQty += qty
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	return inv.QueryRow(ctx, `SELECT coalesce(sum(reserved),0), coalesce(sum(on_hand),0) FROM omdd.stock`).Scan(&r.StockReserved, &r.StockOnHand)
}

// WaitSettled repeats dispatch passes until the status is settled or the
// deadline passes. It returns the last status.
func WaitSettled(ctx context.Context, primary *pgx.Conn, inventoryDSN string, plan Plan, opt DispatchOptions, deadline time.Duration) (StatusReport, int, error) {
	end := time.Now().Add(deadline)
	passes := 0
	for {
		if _, err := Dispatch(ctx, primary, inventoryDSN, opt); err != nil {
			return StatusReport{}, passes, err
		}
		passes++
		st, err := Status(ctx, primary, inventoryDSN, plan, opt)
		if err != nil || st.Settled() || time.Now().After(end) {
			return st, passes, err
		}
		time.Sleep(500 * time.Millisecond)
	}
}
