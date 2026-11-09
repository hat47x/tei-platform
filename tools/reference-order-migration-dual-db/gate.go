package ordermigrationdualdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

// CrossSchema identifies the legacy-versus-primary reconciliation format.
const CrossSchema = "tei.reference.order-migration-cross-reconcile/v0"

// CrossReconcile compares the current legacy database with the primary
// under the plan's rules. Only the plan's declared known differences are
// applied before comparing.
type CrossReconcile struct {
	Schema                  string       `json:"schema"`
	PlanDigest              string       `json:"planDigest"`
	Rules                   []string     `json:"rules"`
	KnownDifferencesApplied []string     `json:"knownDifferencesApplied"`
	Legacy                  Summary      `json:"legacy"`
	Primary                 Summary      `json:"primary"`
	OrderDifferences        []Difference `json:"orderDifferences"`
	MissingInLegacy         []string     `json:"missingInLegacy"`
	Customers               int          `json:"customers"`
	CustomerDifferences     []string     `json:"customerDifferences"`
	Passed                  bool         `json:"passed"`
}

// compareLegacyPrimary reads both sides in the given transactions.
func compareLegacyPrimary(ctx context.Context, ltx, ptx pgx.Tx, plan Plan) (CrossReconcile, error) {
	res := CrossReconcile{Schema: CrossSchema, PlanDigest: plan.Digest, Rules: plan.Reconciliation.Rules,
		KnownDifferencesApplied: []string{}, OrderDifferences: []Difference{}, MissingInLegacy: []string{}, CustomerDifferences: []string{}}
	legacyOrders, err := loadOrders(ctx, ltx, legacyTables, "", keyRange{})
	if err != nil {
		return res, err
	}
	primaryOrders, err := loadOrders(ctx, ptx, primaryTables, "", keyRange{})
	if err != nil {
		return res, err
	}
	if plan.Declares(KnownDifferenceItemCode) {
		res.KnownDifferencesApplied = append(res.KnownDifferencesApplied, KnownDifferenceItemCode)
		for i := range primaryOrders {
			for j := range primaryOrders[i].Lines {
				primaryOrders[i].Lines[j].ItemCode = plan.LegacyCode(primaryOrders[i].Lines[j].ItemCode)
			}
		}
	}
	lobs, err := Observe(legacyOrders)
	if err != nil {
		return res, err
	}
	pobs, err := Observe(primaryOrders)
	if err != nil {
		return res, err
	}
	res.Legacy, res.Primary = lobs.Summary, pobs.Summary
	res.OrderDifferences = Compare(plan.Reconciliation.Rules, lobs, pobs)
	for k := range pobs.RowHashes {
		if _, ok := lobs.RowHashes[k]; !ok {
			res.MissingInLegacy = append(res.MissingInLegacy, k)
		}
	}
	sort.Strings(res.MissingInLegacy)
	lc, err := loadLegacyCustomers(ctx, ltx)
	if err != nil {
		return res, err
	}
	pc, err := loadPrimaryCustomers(ctx, ptx)
	if err != nil {
		return res, err
	}
	byCode := map[string]Customer{}
	for _, c := range pc {
		byCode[c.Code] = c
	}
	res.Customers = len(lc)
	for _, l := range lc {
		p, ok := byCode[l.Code]
		if !ok {
			res.CustomerDifferences = append(res.CustomerDifferences, l.Code+":missing-in-primary")
			continue
		}
		delete(byCode, l.Code)
		for _, f := range CustomerFields() {
			if l.field(f) != p.field(f) {
				res.CustomerDifferences = append(res.CustomerDifferences, l.Code+":"+f)
			}
		}
	}
	for code := range byCode {
		res.CustomerDifferences = append(res.CustomerDifferences, code+":missing-in-legacy")
	}
	sort.Strings(res.CustomerDifferences)
	res.Passed = len(res.OrderDifferences) == 0 && len(res.MissingInLegacy) == 0 && len(res.CustomerDifferences) == 0
	return res, nil
}

// readBoth opens read-only repeatable-read transactions on both sides.
func readBoth(ctx context.Context, legacy, primary *pgx.Conn, plan Plan, fn func(ltx, ptx pgx.Tx) error) error {
	ltx, err := legacy.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer ltx.Rollback(context.Background())
	ptx, err := primary.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer ptx.Rollback(context.Background())
	if _, err := requireBound(ctx, ptx, plan); err != nil {
		return err
	}
	return fn(ltx, ptx)
}

// Queues counts the work that must be empty before a switch.
type Queues struct {
	HeldChanges        int `json:"heldChanges"`
	QueuedConflicts    int `json:"queuedMasterConflicts"`
	OpenSchemaIssues   int `json:"openSchemaIssues"`
	BillingUnsettled   int `json:"billingUnsettled"`
	ReverseUndelivered int `json:"reverseUndelivered"`
}

func (q Queues) blockers() []string {
	var out []string
	for name, n := range map[string]int{"held-changes": q.HeldChanges, "queued-master-conflicts": q.QueuedConflicts,
		"open-schema-issues": q.OpenSchemaIssues, "billing-unsettled": q.BillingUnsettled, "reverse-undelivered": q.ReverseUndelivered} {
		if n > 0 {
			out = append(out, fmt.Sprintf("%s:%d", name, n))
		}
	}
	sort.Strings(out)
	return out
}

func readQueues(ctx context.Context, q querier) (Queues, error) {
	var out Queues
	rows, err := q.Query(ctx, `SELECT
		(SELECT count(*) FROM omdd.cdc_held WHERE state='held'),
		(SELECT count(*) FROM omdd.master_conflict WHERE state='queued'),
		(SELECT count(*) FROM omdd.schema_issue WHERE state='open'),
		(SELECT count(*) FROM omdd.billing_outbox WHERE state<>'delivered'),
		(SELECT count(*) FROM omdd.reverse_outbox WHERE state='pending')`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&out.HeldChanges, &out.QueuedConflicts, &out.OpenSchemaIssues, &out.BillingUnsettled, &out.ReverseUndelivered); err != nil {
			return out, err
		}
	}
	return out, rows.Err()
}

// GateSchema identifies the cutover gate record format.
const GateSchema = "tei.reference.order-migration-cutover-gate/v0"

// GateResult is the record of one pass through the cutover gate (step 5).
type GateResult struct {
	Schema         string         `json:"schema"`
	PlanDigest     string         `json:"planDigest"`
	SnapshotDigest string         `json:"snapshotDigest"`
	LegacyWALLSN   string         `json:"legacyWalLsnAtGate"`
	ChangeFeed     ApplyResult    `json:"changeFeedDrain"`
	Reconcile      CrossReconcile `json:"reconcile"`
	Allocation     map[string]int `json:"allocationClasses"`
	Queues         Queues         `json:"queues"`
	Blockers       []string       `json:"blockers"`
	Passed         bool           `json:"passed"`
	Digest         string         `json:"digest,omitempty"`
	Existing       bool           `json:"existing,omitempty"`
}

func digestOf(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Gate is step 5. It drains the change feed, observes both databases of
// step 3 (every order must be both-committed or compensated), checks that
// no held change, queued conflict, open schema issue, unsettled billing
// entry or undelivered reverse-sync entry remains, and reconciles the
// legacy database with the primary under the plan's rules and declared
// known differences. Any blocker refuses the switch. A passing gate makes
// the primary the authority for orders and the owner of every customer
// field, in the same primary transaction that records the gate.
func Gate(ctx context.Context, legacy, primary *pgx.Conn, inventoryDSN string, plan Plan, opt DispatchOptions) (GateResult, error) {
	snap, err := CheckBound(ctx, primary, plan)
	if err != nil {
		return GateResult{}, err
	}
	if g, ok, err := passedGate(ctx, primary, plan); err != nil || ok {
		g.Existing = ok
		return g, err
	}
	res := GateResult{Schema: GateSchema, PlanDigest: plan.Digest, SnapshotDigest: snap.Digest, Blockers: []string{}}
	if err := legacy.QueryRow(ctx, `SELECT pg_current_wal_lsn()::text`).Scan(&res.LegacyWALLSN); err != nil {
		return res, err
	}
	if res.ChangeFeed, err = ApplyChanges(ctx, legacy, primary, plan); err != nil {
		return res, fmt.Errorf("drain change feed: %w", err)
	}
	st, err := Status(ctx, primary, inventoryDSN, plan, opt)
	if err != nil {
		return res, err
	}
	res.Allocation = st.Classes
	if !st.Settled() {
		res.Blockers = append(res.Blockers, fmt.Sprintf("allocation-unsettled:%d", st.Orders-st.Classes[ClassBothCommitted]-st.Classes[ClassCompensated]))
	}
	if st.Duplicates != 0 || st.OrphanAllocations != 0 {
		res.Blockers = append(res.Blockers, fmt.Sprintf("allocation-duplicates:%d", st.Duplicates+st.OrphanAllocations))
	}
	err = readBoth(ctx, legacy, primary, plan, func(ltx, ptx pgx.Tx) error {
		var err error
		if res.Queues, err = readQueues(ctx, ptx); err != nil {
			return err
		}
		res.Reconcile, err = compareLegacyPrimary(ctx, ltx, ptx, plan)
		return err
	})
	if err != nil {
		return res, err
	}
	res.Blockers = append(res.Blockers, res.Queues.blockers()...)
	for _, d := range res.Reconcile.OrderDifferences {
		res.Blockers = append(res.Blockers, "order-difference:"+d.Rule)
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
	if _, err := requireBound(ctx, tx, plan); err != nil {
		return res, err
	}
	raw, err := json.Marshal(res)
	if err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.cutover_gate(gate_digest, plan_digest, snapshot_digest, passed, result) VALUES ($1,$2,$3,$4,$5)`,
		res.Digest, plan.Digest, snap.Digest, res.Passed, raw); err != nil {
		return res, err
	}
	if res.Passed {
		if _, err := tx.Exec(ctx, `UPDATE omdd.mig_authority SET authority='primary', gate_digest=$1, changed_at=now() WHERE entity='order' AND authority='legacy'`, res.Digest); err != nil {
			return res, err
		}
		for _, f := range CustomerFields() {
			if err := setOwner(ctx, tx, f, "primary", res.Digest, "cutover:"+res.Digest); err != nil {
				return res, err
			}
		}
	}
	return res, tx.Commit(ctx)
}

// ErrNoPassedGate is returned by step 6 before a gate has passed.
var ErrNoPassedGate = errors.New("no passed cutover gate")

// passedGate returns the gate that made the primary the current authority.
func passedGate(ctx context.Context, primary *pgx.Conn, plan Plan) (GateResult, bool, error) {
	var raw []byte
	err := primary.QueryRow(ctx, `SELECT g.result FROM omdd.mig_authority a JOIN omdd.cutover_gate g ON g.gate_digest = a.gate_digest
		WHERE a.entity='order' AND a.authority IN ('primary','rolling-back') AND g.passed AND g.plan_digest=$1`, plan.Digest).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return GateResult{}, false, nil
	}
	if err != nil {
		return GateResult{}, false, err
	}
	var g GateResult
	if err := json.Unmarshal(raw, &g); err != nil {
		return GateResult{}, false, err
	}
	return g, true, nil
}
