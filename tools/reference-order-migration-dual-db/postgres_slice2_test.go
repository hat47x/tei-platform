package ordermigrationdualdb

// Real PostgreSQL tests for steps 4-6. Same databases and skip rule as
// postgres_integration_test.go; the legacy instance needs wal_level=logical.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// legacyOrder writes one order with its lines in one legacy transaction.
func legacyOrder(ctx context.Context, conn *pgx.Conn, o Order) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `INSERT INTO omdd.legacy_order VALUES ($1,$2,$3::text::date,$4)`, o.OrderNo, o.CustomerCode, o.OrderedOn, o.Status); err != nil {
		return err
	}
	for _, l := range o.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO omdd.legacy_order_line VALUES ($1,$2,$3,$4,$5::text::numeric,$6::text::numeric)`,
			o.OrderNo, l.LineNo, l.ItemCode, l.Qty, l.UnitPrice, l.Amount); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func newLegacyOrder(key, customer, item string, qty int64) Order {
	return Order{OrderNo: key, CustomerCode: customer, OrderedOn: "2026-10-20", Status: "open",
		Lines: []Line{{LineNo: 1, ItemCode: item, Qty: qty, UnitPrice: "500", Amount: fmt.Sprint(500 * qty)}}}
}

func (h *harness) connect(role Role) *pgx.Conn {
	h.t.Helper()
	dsn := map[Role]string{RoleLegacy: h.legacyDSN, RolePrimary: h.primaryDSN, RoleInventory: h.inventoryDSN}[role]
	c, err := Connect(h.ctx, dsn, role, 10*time.Second)
	if err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() { c.Close(context.Background()) })
	return c
}

// legacyWriter keeps writing to the legacy database from its own connection
// until stopped: new orders (some with a one-to-one legacy item code) and
// status changes of migrated orders.
type legacyWriter struct {
	stop     chan struct{}
	done     chan struct{}
	inserted []string
	updates  int
	err      error
}

func (h *harness) startLegacyWriter(prefix string) *legacyWriter {
	conn := h.connect(RoleLegacy)
	w := &legacyWriter{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(w.done)
		for i := 1; ; i++ {
			select {
			case <-w.stop:
				return
			default:
			}
			item := "ITM-B"
			if i%3 == 0 {
				item = "100001"
			}
			key := fmt.Sprintf("%s%05d", prefix, i)
			if err := legacyOrder(h.ctx, conn, newLegacyOrder(key, fmt.Sprintf("C%03d", i%syntheticCustomers), item, int64(1+i%3))); err != nil {
				w.err = err
				return
			}
			w.inserted = append(w.inserted, key)
			if _, err := conn.Exec(h.ctx, `UPDATE omdd.legacy_order SET status = CASE status WHEN 'open' THEN 'hold' ELSE 'open' END WHERE order_no=$1`, fmt.Sprintf("SO-%06d", 1+i%100)); err != nil {
				w.err = err
				return
			}
			w.updates++
			time.Sleep(15 * time.Millisecond)
		}
	}()
	return w
}

func (w *legacyWriter) halt(t *testing.T) {
	t.Helper()
	close(w.stop)
	<-w.done
	if w.err != nil {
		t.Fatalf("legacy writer: %v", w.err)
	}
}

// bulkBypassingTriggers is the legacy batch that writes with triggers off.
// logged counts the legacy trigger log rows for the rows the batch touched
// (the concurrent writer never writes those lines).
func (h *harness) bulkBypassingTriggers() (rows int, logged int) {
	h.t.Helper()
	const bulkLog = `SELECT count(*) FROM omdd.legacy_change_log WHERE table_name='legacy_order_line' AND row_key LIKE 'SO-00001%'`
	before := h.count(h.legacy, bulkLog)
	tx, err := h.legacy.Begin(h.ctx)
	if err != nil {
		h.t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(h.ctx, `SET LOCAL session_replication_role = replica`); err != nil {
		h.t.Fatal(err)
	}
	tag, err := tx.Exec(h.ctx, `UPDATE omdd.legacy_order_line SET unit_price = unit_price + 1, amount = (unit_price + 1) * qty WHERE order_no LIKE 'SO-00001%'`)
	if err != nil {
		h.t.Fatal(err)
	}
	if err := tx.Commit(h.ctx); err != nil {
		h.t.Fatal(err)
	}
	return int(tag.RowsAffected()), h.count(h.legacy, bulkLog) - before
}

func (h *harness) crossReconcile(plan Plan) CrossReconcile {
	h.t.Helper()
	var res CrossReconcile
	err := readBoth(h.ctx, h.legacy, h.primary, plan, func(ltx, ptx pgx.Tx) error {
		var err error
		res, err = compareLegacyPrimary(h.ctx, ltx, ptx, plan)
		return err
	})
	if err != nil {
		h.t.Fatal(err)
	}
	return res
}

func (h *harness) applyChanges(plan Plan) ApplyResult {
	h.t.Helper()
	r, err := ApplyChanges(h.ctx, h.legacy, h.primary, plan)
	if err != nil {
		h.t.Fatal(err)
	}
	return r
}

func (h *harness) reverseSync(plan Plan) ReverseResult {
	h.t.Helper()
	r, err := ReverseSync(h.ctx, h.legacy, h.primary, plan)
	if err != nil {
		h.t.Fatal(err)
	}
	return r
}

func (h *harness) dispositions(plan Plan) map[string]int {
	out := map[string]int{}
	rows, err := h.primary.Query(h.ctx, `SELECT disposition, count(*) FROM omdd.cdc_tx WHERE plan_digest=$1 GROUP BY disposition`, plan.Digest)
	if err != nil {
		h.t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		var n int
		rows.Scan(&d, &n)
		out[d] = n
	}
	return out
}

// migratedPlan runs steps 1-3 in process and settles the allocations.
func (h *harness) migratedPlan(chunkSize int) (Plan, string) {
	h.t.Helper()
	plan, _, path := h.planAndSnapshot(chunkSize)
	if _, err := Run(h.ctx, h.primary, h.inventoryDSN, plan, h.opt); err != nil {
		h.t.Fatal(err)
	}
	return plan, path
}

// P-MIG-02: writes continue during the copy and the migration, including a
// batch that bypasses triggers; the change feed applies each legacy
// transaction exactly once across a kill, and nothing is lost.
func TestPostgresPMIG02ChangeFeedNoLossNoDuplicate(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, err := MakePlan(h.ctx, h.legacy, 20)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(h.dir, "plan.json")
	if err := WritePlan(path, plan); err != nil {
		t.Fatal(err)
	}
	writer := h.startLegacyWriter("SO-9")
	w1, w2 := h.connect(RoleLegacy), h.connect(RoleLegacy)
	var inProgress pgx.Tx
	afterSlotHook = func() {
		// Committed after the slot and before the snapshot: decoded, but
		// already in the copy.
		if err := legacyOrder(h.ctx, w1, newLegacyOrder("SO-800001", "C001", "ITM-A", 1)); err != nil {
			t.Error(err)
		}
		// In the copy with a one-to-many item code: step 3 must hold it.
		if err := legacyOrder(h.ctx, w1, newLegacyOrder("SO-800003", "C003", "100002", 1)); err != nil {
			t.Error(err)
		}
		// Running while the snapshot is taken: not in the copy.
		var err error
		if inProgress, err = w2.Begin(h.ctx); err == nil {
			_, err = inProgress.Exec(h.ctx, `INSERT INTO omdd.legacy_order VALUES ('SO-800002','C002','2026-10-20','open')`)
		}
		if err == nil {
			_, err = inProgress.Exec(h.ctx, `INSERT INTO omdd.legacy_order_line VALUES ('SO-800002',1,'ITM-C',2,700,1400)`)
		}
		if err != nil {
			t.Error(err)
		}
	}
	snap, _, err := TakeSnapshot(h.ctx, h.legacy, h.primary, plan)
	afterSlotHook = nil
	if err != nil || t.Failed() {
		t.Fatalf("snapshot: %v", err)
	}
	if err := inProgress.Commit(h.ctx); err != nil {
		t.Fatal(err)
	}
	if h.count(h.primary, `SELECT count(*) FROM omdd.stage_order WHERE order_no='SO-800001'`) != 1 ||
		h.count(h.primary, `SELECT count(*) FROM omdd.stage_order WHERE order_no='SO-800002'`) != 0 {
		t.Fatal("snapshot did not split the two transactions as arranged")
	}
	h.child(CrashChunkMid+":2", "run", "-plan", path)
	if r := h.child("", "run", "-plan", path); r.code != 0 {
		t.Fatalf("run: %s", r.stderr)
	}
	bulkRows, bulkLogged := h.bulkBypassingTriggers()
	writer.halt(t)
	if bulkRows == 0 || bulkLogged != 0 {
		t.Fatalf("bulk update rows %d, trigger log rows %d", bulkRows, bulkLogged)
	}
	h.child(CrashAfterChangeApplied+":3", "apply-changes", "-plan", path)
	afterKill := h.count(h.primary, `SELECT count(*) FROM omdd.cdc_tx`)
	r := h.child("", "apply-changes", "-plan", path)
	if r.code != 0 {
		t.Fatalf("apply-changes: %s", r.stderr)
	}
	var resumed ApplyResult
	if err := json.Unmarshal([]byte(r.stdout), &resumed); err != nil {
		t.Fatal(err)
	}
	again := h.applyChanges(plan)
	if again.Transactions != 0 {
		t.Fatalf("a third pass applied %d transactions", again.Transactions)
	}
	if afterKill != 3 || resumed.AlreadyRecorded == 0 {
		t.Fatalf("kill after the third transaction: recorded %d, resumed re-read %d recorded ones", afterKill, resumed.AlreadyRecorded)
	}
	disp := h.dispositions(plan)
	dupXids := h.count(h.primary, `SELECT count(*) - count(DISTINCT xid) FROM omdd.cdc_tx`)
	if disp[DispositionVisible] < 1 || dupXids != 0 || disp[DispositionHeld] != 0 {
		t.Fatalf("dispositions %v duplicate xids %d", disp, dupXids)
	}
	heldByStep3 := h.count(h.primary, `SELECT count(*) FROM omdd.cdc_held WHERE entity_key='SO-800003' AND commit_lsn IS NULL AND reason=$1 AND state='held'`, HeldOneToManyCode)
	if heldByStep3 != 1 || h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog WHERE order_no='SO-800003'`) != 0 {
		t.Fatal("step 3 did not hold the copied order with a one-to-many code")
	}
	if c := h.crossReconcile(plan); c.Passed {
		t.Fatal("the held order did not show as a difference")
	}
	if err := ResolveHeldOrder(h.ctx, h.legacy, h.primary, plan, "SO-800003", map[int]string{1: "1000020001"}, "test-operator"); err != nil {
		t.Fatal(err)
	}
	cross := h.crossReconcile(plan)
	if !cross.Passed {
		t.Fatalf("legacy and primary differ: %+v", cross)
	}
	for _, k := range append([]string{"SO-800001", "SO-800002", "SO-800003"}, writer.inserted...) {
		if h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog WHERE order_no=$1`, k) != 1 ||
			h.count(h.primary, `SELECT count(*) FROM omdd.effect_outbox WHERE order_no=$1`, k) != 1 {
			t.Fatalf("order %s not applied exactly once", k)
		}
	}
	mapped := h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog_line WHERE item_code='1000010001'`)
	st, _, err := WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 60*time.Second)
	if err != nil || !st.Settled() {
		t.Fatalf("allocations not settled: %v %+v", err, st.Classes)
	}
	requireNoDoubleAllocation(t, st)
	compensations := map[string]string{}
	rows, _ := h.primary.Query(h.ctx, `SELECT order_no, reason FROM omdd.compensation`)
	for rows.Next() {
		var k, reason string
		rows.Scan(&k, &reason)
		compensations[k] = reason
	}
	rows.Close()
	if len(compensations) != 0 {
		t.Fatalf("reservations rejected (an unconverted code?): %v", compensations)
	}
	observed(t, "changeFeed", map[string]any{"heldByStep3FromCopy": heldByStep3, "slotConsistentLsn": snap.Position.ChangeFeed.ConsistentLSN, "pgSnapshot": snap.Position.PGSnapshot,
		"approximateWalLsn": snap.Position.WALLSN, "exactness": snap.Position.ChangeFeed.Exactness, "dispositions": disp,
		"recordedBeforeKill": afterKill, "reReadAfterKill": resumed.AlreadyRecorded, "duplicateXids": dupXids,
		"writerInserted": len(writer.inserted), "writerUpdates": writer.updates, "bulkRowsWithoutTrigger": bulkRows,
		"triggerLogRowsForBulk": bulkLogged, "linesUpdatedByFeed": resumed.LinesUpdated, "mappedOneToOneLines": mapped,
		"legacyOrders": cross.Legacy.OrderCount, "primaryOrders": cross.Primary.OrderCount, "rowsetDigestEqual": cross.Legacy.RowsetDigest == cross.Primary.RowsetDigest,
		"allocationClasses": st.Classes})
}

// P-INT-03: field owners decide concurrent customer updates or queue them;
// a one-to-many item code is held, not guessed, until a person decides.
func TestPostgresPINT03FieldOwnersAndCodeMap(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, _ := h.migratedPlan(30)
	h.applyChanges(plan)
	if err := SwitchFieldOwner(h.ctx, h.primary, plan, FieldCreditLimit, "primary", "test:credit-to-new"); err != nil {
		t.Fatal(err)
	}
	str := func(s string) *string { return &s }

	// Address (legacy-owned) on the legacy side and credit limit
	// (primary-owned) on the new side, on the same customer.
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET address='大阪府大阪市1-1' WHERE customer_code='C001'`)
	if err := UpdateCustomer(h.ctx, h.primary, "C001", CustomerChange{CreditLimit: str("2000000")}); err != nil {
		t.Fatal(err)
	}
	if err := UpdateCustomer(h.ctx, h.primary, "C001", CustomerChange{Address: str("x")}); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("non-owned write on the new side: %v", err)
	}
	// Both sides change the credit limit: the owner (primary) wins.
	if err := UpdateCustomer(h.ctx, h.primary, "C002", CustomerChange{CreditLimit: str("3000000")}); err != nil {
		t.Fatal(err)
	}
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET credit_limit=2500000 WHERE customer_code='C002'`)
	// Only the legacy side changes the credit limit it does not own: queued.
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET credit_limit=999 WHERE customer_code='C003'`)
	ap := h.applyChanges(plan)
	if ap.Conflicts["owner-wins:primary"] != 1 || ap.Conflicts["queued:non-owner-write-by-legacy"] != 1 || ap.CustomersUpdated != 1 {
		t.Fatalf("conflicts %+v updated %d", ap.Conflicts, ap.CustomersUpdated)
	}
	rv := h.reverseSync(plan)
	echo := h.applyChanges(plan)
	if echo.Dispositions[DispositionEcho] != rv.AppliedGroups || echo.BaseRefreshed == 0 {
		t.Fatalf("reverse %+v echo %+v", rv, echo)
	}
	cross := h.crossReconcile(plan)
	if len(cross.CustomerDifferences) != 1 || cross.CustomerDifferences[0] != "C003:credit_limit" {
		t.Fatalf("customer differences %v", cross.CustomerDifferences)
	}
	var queuedID int64
	h.primary.QueryRow(h.ctx, `SELECT id FROM omdd.master_conflict WHERE state='queued'`).Scan(&queuedID)
	if err := ResolveMasterConflict(h.ctx, h.primary, plan, queuedID, false); err != nil {
		t.Fatal(err)
	}
	h.reverseSync(plan)
	h.applyChanges(plan)
	if cross = h.crossReconcile(plan); len(cross.CustomerDifferences) != 0 {
		t.Fatalf("after resolution %v", cross.CustomerDifferences)
	}
	conflicts := map[string]int{}
	rows, _ := h.primary.Query(h.ctx, `SELECT resolution, count(*) FROM omdd.master_conflict GROUP BY resolution`)
	for rows.Next() {
		var r string
		var n int
		rows.Scan(&r, &n)
		conflicts[r] = n
	}
	rows.Close()

	// Item codes: one-to-one converts, one-to-many is held.
	if err := legacyOrder(h.ctx, h.legacy, newLegacyOrder("SO-700001", "C004", "100001", 2)); err != nil {
		t.Fatal(err)
	}
	if err := legacyOrder(h.ctx, h.legacy, newLegacyOrder("SO-700002", "C005", "100002", 3)); err != nil {
		t.Fatal(err)
	}
	h.exec(h.legacy, `UPDATE omdd.legacy_order SET status='hold' WHERE order_no='SO-700002'`)
	ap = h.applyChanges(plan)
	if ap.Held[HeldOneToManyCode] != 1 || ap.Held[HeldOrderAlreadyHeld] != 1 || ap.OrdersInserted != 1 {
		t.Fatalf("code map handling %+v", ap)
	}
	if h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog_line WHERE order_no='SO-700001' AND item_code='1000010001'`) != 1 ||
		h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog WHERE order_no='SO-700002'`) != 0 {
		t.Fatal("one-to-one not converted or one-to-many guessed")
	}
	cross = h.crossReconcile(plan)
	if cross.Passed || len(cross.OrderDifferences) == 0 || !strings.Contains(fmt.Sprint(cross.OrderDifferences), "missing:SO-700002") {
		t.Fatalf("held order not visible as a difference: %+v", cross.OrderDifferences)
	}
	heldDiff := cross.OrderDifferences
	if err := ResolveHeldOrder(h.ctx, h.legacy, h.primary, plan, "SO-700002", map[int]string{1: "1000099999"}, "test"); err == nil {
		t.Fatal("a code outside the map was accepted")
	}
	if err := ResolveHeldOrder(h.ctx, h.legacy, h.primary, plan, "SO-700002", map[int]string{1: "1000020002"}, "test-operator"); err != nil {
		t.Fatal(err)
	}
	if cross = h.crossReconcile(plan); !cross.Passed {
		t.Fatalf("after the decision: %+v", cross)
	}
	if h.count(h.primary, `SELECT count(*) FROM omdd.cdc_held WHERE state='held'`) != 0 {
		t.Fatal("held changes remain")
	}
	// Without the declared known difference, the converted codes differ.
	undeclared := plan
	undeclared.KnownDifferences = []string{}
	var raw CrossReconcile
	readBoth(h.ctx, h.legacy, h.primary, plan, func(ltx, ptx pgx.Tx) error {
		var err error
		raw, err = compareLegacyPrimary(h.ctx, ltx, ptx, undeclared)
		return err
	})
	if raw.Passed {
		t.Fatal("item code difference passed without being declared")
	}
	observed(t, "fieldOwners", map[string]any{"conflicts": conflicts, "reverseGroups": rv.AppliedGroups, "echoBaseRefreshed": echo.BaseRefreshed})
	observed(t, "codeMap", map[string]any{"held": ap.Held, "differencesWhileHeld": heldDiff, "passedAfterDecision": cross.Passed,
		"undeclaredDifferences": raw.OrderDifferences})
}

// P-MIG-03: old-version writers keep writing full names during the name
// split; both new columns are filled, and contraction is refused while an
// old-version writer remains.
func TestPostgresPMIG03ExpandWithOldWritersContractRefused(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, _ := h.migratedPlan(30)
	h.applyChanges(plan)
	ex, err := ExpandNameSplit(h.ctx, h.primary, plan)
	if err != nil {
		t.Fatal(err)
	}
	if ex.Backfilled != syntheticCustomers || len(ex.Unsplittable) != 0 {
		t.Fatalf("backfill %+v", ex)
	}
	// The legacy system is an old-version writer (through the change feed).
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET customer_name='新姓 新名' WHERE customer_code='C004'`)
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET customer_name='山田　花子' WHERE customer_code='C005'`)
	h.applyChanges(plan)
	// An old integration writes the primary directly, full name only.
	if err := RegisterSchemaClient(h.ctx, h.primary, "old-integration", SchemaVersionV1); err != nil {
		t.Fatal(err)
	}
	if err := SwitchFieldOwner(h.ctx, h.primary, plan, FieldName, "primary", "test:name-to-new"); err != nil {
		t.Fatal(err)
	}
	h.exec(h.primary, `UPDATE omdd.customer SET full_name='旧連携 太郎' WHERE customer_code='C006'`)
	h.exec(h.primary, `UPDATE omdd.customer SET full_name='旧連携一郎' WHERE customer_code='C007'`)
	names := map[string][2]*string{}
	rows, _ := h.primary.Query(h.ctx, `SELECT customer_code, family_name, given_name FROM omdd.customer WHERE customer_code IN ('C004','C005','C006','C007')`)
	for rows.Next() {
		var code string
		var f, g *string
		rows.Scan(&code, &f, &g)
		names[code] = [2]*string{f, g}
	}
	rows.Close()
	is := func(code, f, g string) bool {
		n := names[code]
		return n[0] != nil && n[1] != nil && *n[0] == f && *n[1] == g
	}
	if !is("C004", "新姓", "新名") || !is("C005", "山田", "花子") || !is("C006", "旧連携", "太郎") || names["C007"][0] != nil {
		t.Fatalf("old-format writes not split as expected: %v", names)
	}
	if h.count(h.primary, `SELECT count(*) FROM omdd.schema_issue WHERE state='open' AND entity_key='C007'`) != 1 {
		t.Fatal("unsplittable name not listed")
	}
	refused, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{Since: time.Now().Add(-time.Hour)})
	if !errors.Is(err, ErrContractRefused) || refused.Allowed {
		t.Fatalf("contract with old writers: %+v %v", refused, err)
	}
	if err := RetireSchemaClient(h.ctx, h.primary, "old-integration"); err != nil {
		t.Fatal(err)
	}
	// Observe from here on: the old integration is retired, and the next
	// write is the new-version UpdateCustomer.
	var observationStart time.Time
	if err := h.primary.QueryRow(h.ctx, `SELECT clock_timestamp()`).Scan(&observationStart); err != nil {
		t.Fatal(err)
	}
	str := func(s string) *string { return &s }
	if err := UpdateCustomer(h.ctx, h.primary, "C007", CustomerChange{FamilyName: str("旧連携"), GivenName: str("一郎")}); err != nil {
		t.Fatal(err)
	}
	stillRefused, err := CheckContractNameSplit(h.ctx, h.primary, plan, ContractOptions{Since: observationStart})
	if !errors.Is(err, ErrContractRefused) || len(stillRefused.Blockers) != 1 || stillRefused.Blockers[0] != "old-version-writer:"+ClientLegacyCDC {
		t.Fatalf("contract with only the legacy writer left: %+v %v", stillRefused, err)
	}
	// Only when no old-version writer is left would the contract be allowed.
	outer, err := h.primary.Begin(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	outer.Exec(h.ctx, `UPDATE omdd.schema_client SET active=false WHERE client_id=$1`, ClientLegacyCDC)
	allowed, err := CheckContractNameSplit(h.ctx, outer, plan, ContractOptions{Since: observationStart})
	outer.Rollback(h.ctx)
	if err != nil || !allowed.Allowed {
		t.Fatalf("contract without old writers: %+v %v", allowed, err)
	}
	h.reverseSync(plan)
	h.applyChanges(plan)
	cross := h.crossReconcile(plan)
	bad := h.count(h.primary, `SELECT count(*) FROM omdd.customer WHERE family_name IS NULL OR given_name IS NULL OR omdd.split_name(full_name) IS DISTINCT FROM ARRAY[family_name, given_name]`)
	if bad != 0 || !cross.Passed {
		t.Fatalf("names inconsistent %d, cross %+v", bad, cross.CustomerDifferences)
	}
	events := []string{}
	rows, _ = h.primary.Query(h.ctx, `SELECT event FROM omdd.schema_change_event ORDER BY id`)
	for rows.Next() {
		var e string
		rows.Scan(&e)
		events = append(events, e)
	}
	rows.Close()
	observed(t, "nameSplit", map[string]any{"backfilled": ex.Backfilled, "refusedBlockers": refused.Blockers,
		"blockersAfterRetiringIntegration": stillRefused.Blockers, "allowedWithoutOldWriters": allowed.Allowed, "events": events})
}

// P-TX-02: billing calls time out after or before the billing system
// applied them. Unknown results are settled by query, resends carry the
// same idempotency key, and every change is applied once.
func TestPostgresPTX02BillingTimeoutResendAppliedOnce(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, path := h.migratedPlan(30)
	h.applyChanges(plan)
	if err := SwitchFieldOwner(h.ctx, h.primary, plan, FieldCreditLimit, "primary", "test"); err != nil {
		t.Fatal(err)
	}
	str := func(s string) *string { return &s }
	for i := 1; i <= 4; i++ {
		if err := UpdateCustomer(h.ctx, h.primary, fmt.Sprintf("C%03d", i), CustomerChange{CreditLimit: str(fmt.Sprint(5000000 + i))}); err != nil {
			t.Fatal(err)
		}
	}
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET address='福岡県福岡市5-5' WHERE customer_code='C005'`)
	h.applyChanges(plan)
	entries := h.count(h.primary, `SELECT count(*) FROM omdd.billing_outbox`)
	if entries != 5 {
		t.Fatalf("billing entries %d", entries)
	}
	mock := NewBillingMock()
	srv := httptest.NewServer(mock)
	defer srv.Close()
	opt := BillingOptions{URL: srv.URL, Timeout: 400 * time.Millisecond}

	mock.HangAfterApply(1, 3*time.Second)
	r1, err := SendBilling(h.ctx, h.primary, opt)
	if err != nil || r1.Unknown != 1 || r1.Delivered != 0 {
		t.Fatalf("first pass %+v %v", r1, err)
	}
	before := mock.Stats().Requests
	r2, _ := SendBilling(h.ctx, h.primary, opt)
	if !r2.BlockedByUnknown || mock.Stats().Requests != before {
		t.Fatalf("an unknown result was passed over: %+v", r2)
	}
	c1, err := ReconcileBilling(h.ctx, h.primary, opt)
	if err != nil || c1.Confirmed != 1 {
		t.Fatalf("reconcile after apply %+v %v", c1, err)
	}
	mock.HangBeforeApply(1, 3*time.Second)
	r3, _ := SendBilling(h.ctx, h.primary, opt)
	if r3.Unknown != 1 || r3.Delivered != 0 {
		t.Fatalf("second timeout %+v", r3)
	}
	c2, _ := ReconcileBilling(h.ctx, h.primary, opt)
	if c2.NotReceived != 1 {
		t.Fatalf("reconcile before apply %+v", c2)
	}
	r4 := h.child("", "send-billing", "-plan", path, "-billing-url", srv.URL, "-billing-timeout", "2s")
	if r4.code != 0 {
		t.Fatalf("send-billing: %s", r4.stderr)
	}
	// A lost acknowledgement on our side: the same key is sent again.
	h.exec(h.primary, `UPDATE omdd.billing_outbox SET state='pending' WHERE seq=(SELECT min(seq) FROM omdd.billing_outbox)`)
	r5, _ := SendBilling(h.ctx, h.primary, opt)
	st := mock.Stats()
	if r5.Replayed != 1 || st.Applies != entries || st.DistinctKeys != entries || st.MaxPerKey != 1 ||
		h.count(h.primary, `SELECT count(*) FROM omdd.billing_outbox WHERE state<>'delivered'`) != 0 {
		t.Fatalf("billing applied more or less than once: %+v %+v", r5, st)
	}
	var credit string
	h.primary.QueryRow(h.ctx, `SELECT credit_limit::text FROM omdd.customer WHERE customer_code='C004'`).Scan(&credit)
	if p, _ := st.Profiles["C004"].(map[string]any); p == nil || p["creditLimit"] != credit {
		t.Fatalf("billing profile for C004 %v, primary %s", st.Profiles["C004"], credit)
	}
	outcomes := map[string]int{}
	rows, _ := h.primary.Query(h.ctx, `SELECT coalesce(confirmed_by,''), count(*) FROM omdd.billing_outbox GROUP BY 1`)
	for rows.Next() {
		var c string
		var n int
		rows.Scan(&c, &n)
		outcomes[c] = n
	}
	rows.Close()
	observed(t, "billing", map[string]any{"entries": entries, "requests": st.Requests, "applies": st.Applies, "replays": st.Replays,
		"maxAppliesPerKey": st.MaxPerKey, "confirmedBy": outcomes, "unknownSettledByQuery": c1.Confirmed, "notReceivedByQuery": c2.NotReceived})
}

// The through run of steps 1-6, mostly through the CLI in separate
// processes, carrying the plan digest from the plan to the reverse-sync
// receipts and the rollback drill.
func TestPostgresThroughSteps1To6(t *testing.T) {
	h := newHarness(t, ampleStock())
	planPath := filepath.Join(h.dir, "plan.json")
	ok := func(r childResult, what string) childResult {
		t.Helper()
		if r.code != 0 {
			t.Fatalf("%s exit %d: %s", what, r.code, r.stderr)
		}
		return r
	}
	str := func(s string) *string { return &s }
	ok(h.child("", "plan", "-chunk-size", "15", "-out", planPath), "plan")
	plan, err := ReadPlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	// Step 2 and 3 with the legacy side still in business.
	writer := h.startLegacyWriter("SO-6")
	var snapOut struct {
		Snapshot SnapshotRecord `json:"snapshot"`
	}
	json.Unmarshal([]byte(ok(h.child("", "snapshot", "-plan", planPath), "snapshot").stdout), &snapOut)
	snap := snapOut.Snapshot
	h.child(CrashChunkMid+":2", "run", "-plan", planPath)
	ok(h.child("", "run", "-plan", planPath), "run")
	bulkRows, bulkLogged := h.bulkBypassingTriggers()
	writer.halt(t)

	// Step 4.
	h.child(CrashAfterChangeApplied+":2", "apply-changes", "-plan", planPath)
	ok(h.child("", "apply-changes", "-plan", planPath), "apply-changes")
	ok(h.child("", "switch-owner", "-plan", planPath, "-field", FieldCreditLimit, "-owner", "primary"), "switch-owner")
	ok(h.child("", "expand-name", "-plan", planPath), "expand-name")
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET address='名古屋市中区3-3', customer_name='得意先01 新担当' WHERE customer_code='C001'`)
	h.exec(h.legacy, `UPDATE omdd.legacy_customer SET credit_limit=1 WHERE customer_code='C002'`)
	if err := UpdateCustomer(h.ctx, h.primary, "C003", CustomerChange{CreditLimit: str("7777777")}); err != nil {
		t.Fatal(err)
	}
	if err := legacyOrder(h.ctx, h.legacy, newLegacyOrder("SO-500001", "C006", "100002", 1)); err != nil {
		t.Fatal(err)
	}
	ok(h.child("", "apply-changes", "-plan", planPath), "apply-changes")
	if r := h.child("", "contract-name", "-plan", planPath); r.code == 0 || !strings.Contains(r.stderr, "contract refused") {
		t.Fatalf("contract-name: exit %d %s", r.code, r.stderr)
	}
	mock := NewBillingMock()
	srv := httptest.NewServer(mock)
	defer srv.Close()
	mock.HangAfterApply(1, 3*time.Second)
	billing := func() {
		t.Helper()
		ok(h.child("", "send-billing", "-plan", planPath, "-billing-url", srv.URL, "-billing-timeout", "500ms"), "send-billing")
		ok(h.child("", "reconcile-billing", "-plan", planPath, "-billing-url", srv.URL), "reconcile-billing")
		ok(h.child("", "send-billing", "-plan", planPath, "-billing-url", srv.URL, "-billing-timeout", "2s"), "send-billing")
	}
	billing()
	ok(h.child("", "reverse-sync", "-plan", planPath), "reverse-sync")
	ok(h.child("", "run", "-plan", planPath), "dispatch new orders")
	if st, _, err := WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 60*time.Second); err != nil || !st.Settled() {
		t.Fatalf("not settled before the gate: %v", err)
	}

	// Step 5: the gate refuses while differences and queues remain.
	r := h.child("", "gate", "-plan", planPath)
	var refused GateResult
	json.Unmarshal([]byte(r.stdout), &refused)
	if r.code != 3 || refused.Passed || refused.PlanDigest != plan.Digest {
		t.Fatalf("gate with open work: exit %d %+v %s", r.code, refused.Blockers, r.stderr)
	}
	refusedBlockers := refused.Blockers
	var authority string
	h.primary.QueryRow(h.ctx, `SELECT authority FROM omdd.mig_authority WHERE entity='order'`).Scan(&authority)
	if authority != "legacy" {
		t.Fatalf("a refused gate switched the authority to %s", authority)
	}
	if err := ResolveHeldOrder(h.ctx, h.legacy, h.primary, plan, "SO-500001", map[int]string{1: "1000020001"}, "test-operator"); err != nil {
		t.Fatal(err)
	}
	var queued int64
	h.primary.QueryRow(h.ctx, `SELECT id FROM omdd.master_conflict WHERE state='queued'`).Scan(&queued)
	if err := ResolveMasterConflict(h.ctx, h.primary, plan, queued, false); err != nil {
		t.Fatal(err)
	}
	ok(h.child("", "reverse-sync", "-plan", planPath), "reverse-sync")
	ok(h.child("", "run", "-plan", planPath), "dispatch resolved order")
	WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 60*time.Second)
	billing()
	var gate GateResult
	r = ok(h.child("", "gate", "-plan", planPath), "gate")
	json.Unmarshal([]byte(r.stdout), &gate)
	if !gate.Passed || gate.PlanDigest != plan.Digest || gate.SnapshotDigest != snap.Digest || gate.Digest == "" {
		t.Fatalf("gate %+v", gate)
	}
	// Negative: a re-made plan is refused by the gate.
	remade := filepath.Join(h.dir, "plan-remade.json")
	ok(h.child("", "plan", "-chunk-size", "7", "-out", remade), "re-plan")
	if r := h.child("", "gate", "-plan", remade); r.code == 0 || !strings.Contains(r.stderr, "plan digest mismatch") {
		t.Fatalf("gate with re-made plan: exit %d %s", r.code, r.stderr)
	}

	// Step 6: new-side updates after the cutover, reverse sync with a kill,
	// then the rollback drill.
	if err := PlaceOrder(h.ctx, h.primary, plan, Order{OrderNo: "SO-400001", CustomerCode: "C007", OrderedOn: "2026-10-21", Status: "open",
		Lines: []Line{{LineNo: 1, ItemCode: "ITM-A", Qty: 2, UnitPrice: "120", Amount: "240"}, {LineNo: 2, ItemCode: "1000020002", Qty: 1, UnitPrice: "900", Amount: "900"}}}); err != nil {
		t.Fatal(err)
	}
	if err := UpdateOrderStatus(h.ctx, h.primary, "SO-000005", "shipped"); err != nil {
		t.Fatal(err)
	}
	if err := UpdateCustomer(h.ctx, h.primary, "C008", CustomerChange{Address: str("札幌市北区8-8"), FamilyName: str("新姓"), GivenName: str("新名")}); err != nil {
		t.Fatal(err)
	}
	h.child(CrashAfterReverseLegacyCommit+":1", "reverse-sync", "-plan", planPath)
	var rs ReverseResult
	json.Unmarshal([]byte(ok(h.child("", "reverse-sync", "-plan", planPath), "reverse-sync").stdout), &rs)
	if rs.AlreadyApplied != 1 {
		t.Fatalf("redelivery after the kill was not recognised: %+v", rs)
	}
	ok(h.child("", "run", "-plan", planPath), "dispatch new-side order")
	WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 60*time.Second)
	billing()
	var drill RollbackResult
	r = ok(h.child("", "rollback", "-plan", planPath), "rollback")
	json.Unmarshal([]byte(r.stdout), &drill)
	if !drill.Passed || drill.GateDigest != gate.Digest || drill.PlanDigest != plan.Digest || len(drill.Reconcile.MissingInLegacy) != 0 ||
		len(drill.Reconcile.OrderDifferences) != 0 || len(drill.Reconcile.CustomerDifferences) != 0 || drill.ReceiptsGate == 0 {
		t.Fatalf("rollback %+v", drill)
	}
	var legacyStatus, legacyItem string
	h.legacy.QueryRow(h.ctx, `SELECT status FROM omdd.legacy_order WHERE order_no='SO-000005'`).Scan(&legacyStatus)
	h.legacy.QueryRow(h.ctx, `SELECT item_code FROM omdd.legacy_order_line WHERE order_no='SO-400001' AND line_no=2`).Scan(&legacyItem)
	if legacyStatus != "shipped" || legacyItem != "100002" {
		t.Fatalf("new-side updates not in legacy: %s %s", legacyStatus, legacyItem)
	}
	if err := UpdateOrderStatus(h.ctx, h.primary, "SO-000005", "open"); !errors.Is(err, ErrNotAuthority) {
		t.Fatalf("new-side write after rollback: %v", err)
	}
	// After the rollback the legacy side is in business again.
	h.exec(h.legacy, `UPDATE omdd.legacy_order SET status='closed' WHERE order_no='SO-000006'`)
	after := h.applyChanges(plan)
	if after.Dispositions[DispositionApplied] != 1 || !h.crossReconcile(plan).Passed {
		t.Fatalf("legacy write after rollback %+v", after)
	}

	carried := map[string]int{
		"gatesWithOtherPlan":         h.count(h.primary, `SELECT count(*) FROM omdd.cutover_gate WHERE plan_digest<>$1 OR snapshot_digest<>$2`, plan.Digest, snap.Digest),
		"drillsWithOtherDigest":      h.count(h.primary, `SELECT count(*) FROM omdd.rollback_drill WHERE plan_digest<>$1 OR gate_digest<>$2`, plan.Digest, gate.Digest),
		"cdcWithOtherPlan":           h.count(h.primary, `SELECT count(*) FROM omdd.cdc_tx WHERE plan_digest<>$1`, plan.Digest),
		"reverseOutboxOtherPlan":     h.count(h.primary, `SELECT count(*) FROM omdd.reverse_outbox WHERE plan_digest<>$1`, plan.Digest),
		"legacyReceiptsOtherPlan":    h.count(h.legacy, `SELECT count(*) FROM omdd.legacy_reverse_applied WHERE plan_digest<>$1`, plan.Digest),
		"postCutoverReceiptsNoGate":  h.count(h.legacy, `SELECT count(*) FROM omdd.legacy_reverse_applied WHERE entity='order' AND gate_digest IS DISTINCT FROM $1`, gate.Digest),
		"checkpointsWithOtherDigest": h.count(h.primary, `SELECT count(*) FROM omdd.mig_checkpoint WHERE plan_digest<>$1 OR snapshot_digest<>$2`, plan.Digest, snap.Digest),
	}
	for k, n := range carried {
		if n != 0 {
			t.Fatalf("%s: %d", k, n)
		}
	}
	receipts := h.count(h.legacy, `SELECT count(*) FROM omdd.legacy_reverse_applied WHERE plan_digest=$1 AND gate_digest=$2`, plan.Digest, gate.Digest)
	observed(t, "through", map[string]any{"planDigest": plan.Digest, "snapshotDigest": snap.Digest, "gateDigest": gate.Digest, "drillDigest": drill.Digest,
		"slotConsistentLsn": snap.Position.ChangeFeed.ConsistentLSN, "writerInserted": len(writer.inserted), "bulkRowsWithoutTrigger": bulkRows,
		"triggerLogRowsForBulk": bulkLogged, "refusedGateBlockers": refusedBlockers, "gateReconcile": map[string]any{"legacyOrders": gate.Reconcile.Legacy.OrderCount,
			"primaryOrders": gate.Reconcile.Primary.OrderCount, "knownDifferencesApplied": gate.Reconcile.KnownDifferencesApplied, "customers": gate.Reconcile.Customers},
		"reverseAfterKill": rs, "receiptsWithGateDigest": receipts, "drillReconcile": map[string]any{"orderDifferences": len(drill.Reconcile.OrderDifferences),
			"missingInLegacy": len(drill.Reconcile.MissingInLegacy), "customerDifferences": len(drill.Reconcile.CustomerDifferences), "legacyOrders": drill.Reconcile.Legacy.OrderCount},
		"billing": mock.Stats().Applies, "billingMaxPerKey": mock.Stats().MaxPerKey, "carried": carried})
}
