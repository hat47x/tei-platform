package ordermigrationdualdb

// Real PostgreSQL tests. They need three dedicated, disposable databases
// (tei_omdd_legacy, tei_omdd_primary, tei_omdd_inventory) on three separate
// instances, normally started by tools/verify-order-migration-dual-db.py.
// Without the DSNs they skip; a skip is not evidence that anything passed.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	childArgsEnv      = "TEI_OMDD_CHILD_ARGS"
	confirmEnv        = "TEI_OMDD_CONFIRM_ISOLATED"
	containerEnv      = "TEI_OMDD_INVENTORY_CONTAINER"
	outDirEnv         = "TEI_OMDD_OUT_DIR"
	containerPrefix   = "tei-omdd-"
	confirmValue      = "tei_omdd"
	syntheticOrders   = 123
	observedLogPrefix = "OMDD-OBSERVED"
)

// TestMain lets the test binary act as the CLI so a test can start, and see
// killed, a separate process running exactly the CLI code path.
func TestMain(m *testing.M) {
	if raw := os.Getenv(childArgsEnv); raw != "" {
		var args []string
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		os.Exit(Main(args, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

type harness struct {
	t                                   *testing.T
	ctx                                 context.Context
	legacyDSN, primaryDSN, inventoryDSN string
	legacy, primary, inventory          *pgx.Conn
	dir                                 string
	opt                                 DispatchOptions
}

func newHarness(t *testing.T, stock map[string]int64) *harness {
	t.Helper()
	h := &harness{t: t, dir: t.TempDir(), opt: DispatchOptions{ConnectTimeout: 2 * time.Second, OpTimeout: 2 * time.Second}}
	h.legacyDSN, h.primaryDSN, h.inventoryDSN = os.Getenv(RoleLegacy.DSNEnv()), os.Getenv(RolePrimary.DSNEnv()), os.Getenv(RoleInventory.DSNEnv())
	if h.legacyDSN == "" || h.primaryDSN == "" || h.inventoryDSN == "" {
		t.Skip("requires three dedicated PostgreSQL databases (TEI_OMDD_*_DSN); skip is not a pass")
	}
	if os.Getenv(confirmEnv) != confirmValue {
		t.Fatalf("%s must be %q to confirm the databases are disposable", confirmEnv, confirmValue)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	t.Cleanup(cancel)
	h.ctx = ctx
	var err error
	for _, c := range []struct {
		conn **pgx.Conn
		dsn  string
		role Role
	}{{&h.legacy, h.legacyDSN, RoleLegacy}, {&h.primary, h.primaryDSN, RolePrimary}, {&h.inventory, h.inventoryDSN, RoleInventory}} {
		if *c.conn, err = Connect(ctx, c.dsn, c.role, 10*time.Second); err != nil {
			t.Fatal(err)
		}
		conn := *c.conn
		t.Cleanup(func() { conn.Close(context.Background()) })
		if err := ResetSchema(ctx, conn, c.role); err != nil {
			t.Fatalf("%s schema: %v", c.role, err)
		}
	}
	h.requireSeparateInstances()
	h.seedLegacy()
	h.seedStock(stock)
	return h
}

func (h *harness) requireSeparateInstances() {
	ids := map[string]Role{}
	for role, conn := range map[Role]*pgx.Conn{RoleLegacy: h.legacy, RolePrimary: h.primary, RoleInventory: h.inventory} {
		var id string
		if err := conn.QueryRow(h.ctx, `SELECT system_identifier::text FROM pg_control_system()`).Scan(&id); err != nil {
			h.t.Fatal(err)
		}
		if other, dup := ids[id]; dup {
			h.t.Fatalf("%s and %s are the same PostgreSQL instance", role, other)
		}
		ids[id] = role
	}
}

func syntheticLegacy() []Order {
	items := []string{"ITM-A", "ITM-B", "ITM-C", "ITM-D", "ITM-E", "ITM-F"}
	prices := []string{"120", "3800", "45000", "999999", "123456789012", "7"}
	var orders []Order
	for i := 1; i <= syntheticOrders; i++ {
		key := fmt.Sprintf("SO-%06d", i)
		if i > syntheticOrders-3 {
			// Lower-case keys sort after upper-case keys only in byte order.
			key = fmt.Sprintf("so-legacy-%02d", i)
		}
		o := Order{OrderNo: key, CustomerCode: fmt.Sprintf("C%03d", i%17), OrderedOn: fmt.Sprintf("2026-%02d-%02d", 7+i%3, 1+i%28), Status: "open"}
		for l := 1; l <= 1+i%3; l++ {
			item := (i + l) % len(items)
			qty := int64(1 + (i*l)%4)
			price := prices[item]
			amount := mulDecimal(price, qty)
			o.Lines = append(o.Lines, Line{LineNo: l, ItemCode: items[item], Qty: qty, UnitPrice: price, Amount: amount})
		}
		orders = append(orders, o)
	}
	return orders
}

func mulDecimal(price string, qty int64) string {
	var p int64
	fmt.Sscan(price, &p)
	return fmt.Sprint(p * qty)
}

func (h *harness) seedLegacy() {
	for _, o := range syntheticLegacy() {
		h.exec(h.legacy, `INSERT INTO omdd.legacy_order VALUES ($1,$2,$3::text::date,$4)`, o.OrderNo, o.CustomerCode, o.OrderedOn, o.Status)
		for _, l := range o.Lines {
			h.exec(h.legacy, `INSERT INTO omdd.legacy_order_line VALUES ($1,$2,$3,$4,$5::text::numeric,$6::text::numeric)`, o.OrderNo, l.LineNo, l.ItemCode, l.Qty, l.UnitPrice, l.Amount)
		}
	}
}

func ampleStock() map[string]int64 {
	return map[string]int64{"ITM-A": 10000, "ITM-B": 10000, "ITM-C": 10000, "ITM-D": 10000, "ITM-E": 10000, "ITM-F": 10000}
}

func (h *harness) seedStock(stock map[string]int64) {
	for item, n := range stock {
		h.exec(h.inventory, `INSERT INTO omdd.stock(item_code, on_hand) VALUES ($1,$2)`, item, n)
	}
}

func (h *harness) exec(conn *pgx.Conn, sql string, args ...any) {
	h.t.Helper()
	if _, err := conn.Exec(h.ctx, sql, args...); err != nil {
		h.t.Fatalf("%v: %s", err, sql)
	}
}

func (h *harness) count(conn *pgx.Conn, sql string, args ...any) int {
	h.t.Helper()
	var n int
	if err := conn.QueryRow(h.ctx, sql, args...).Scan(&n); err != nil {
		h.t.Fatalf("%v: %s", err, sql)
	}
	return n
}

// planAndSnapshot runs steps 1 and 2 in process.
func (h *harness) planAndSnapshot(chunkSize int) (Plan, SnapshotRecord, string) {
	h.t.Helper()
	plan, err := MakePlan(h.ctx, h.legacy, chunkSize)
	if err != nil {
		h.t.Fatal(err)
	}
	path := filepath.Join(h.dir, "plan.json")
	if err := WritePlan(path, plan); err != nil {
		h.t.Fatal(err)
	}
	rec, existing, err := TakeSnapshot(h.ctx, h.legacy, h.primary, plan)
	if err != nil || existing {
		h.t.Fatalf("snapshot: %v existing=%v", err, existing)
	}
	return plan, rec, path
}

type childResult struct {
	code    int
	crashed string
	stdout  string
	stderr  string
}

// child runs the CLI in a separate process. With crashAt set the process
// kills itself at that point; the test checks the marker and that it died.
func (h *harness) child(crashAt string, args ...string) childResult {
	h.t.Helper()
	raw, _ := json.Marshal(args)
	cmd := exec.CommandContext(h.ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), childArgsEnv+"="+string(raw), CrashEnv+"="+crashAt)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	r := childResult{stdout: out.String(), stderr: errb.String()}
	if cmd.ProcessState != nil {
		r.code = cmd.ProcessState.ExitCode()
	}
	for _, line := range strings.Split(r.stderr, "\n") {
		if strings.HasPrefix(line, CrashMarker+" ") {
			r.crashed = strings.TrimPrefix(line, CrashMarker+" ")
		}
	}
	if crashAt != "" {
		point := strings.SplitN(crashAt, ":", 2)[0]
		if r.crashed == "" || !strings.HasPrefix(r.crashed, point) || err == nil {
			h.t.Fatalf("child %v with %s did not die at the point (exit %d, err %v)\nstderr:\n%s", args, crashAt, r.code, err, r.stderr)
		}
	}
	return r
}

func (h *harness) status(plan Plan) StatusReport {
	h.t.Helper()
	st, err := Status(h.ctx, h.primary, h.inventoryDSN, plan, h.opt)
	if err != nil {
		h.t.Fatal(err)
	}
	return st
}

func observed(t *testing.T, name string, v any) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"test": t.Name(), "name": name, "value": v})
	t.Logf("%s %s", observedLogPrefix, raw)
	if dir := os.Getenv(outDirEnv); dir != "" {
		f, err := os.OpenFile(filepath.Join(dir, "observed.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			f.Write(append(raw, '\n'))
			f.Close()
		}
	}
}

// requireNoDoubleAllocation checks the delivery invariants of P-TX-01.
func requireNoDoubleAllocation(t *testing.T, st StatusReport) {
	t.Helper()
	if st.Duplicates != 0 || st.OrphanAllocations != 0 {
		t.Fatalf("duplicates %d orphans %d", st.Duplicates, st.OrphanAllocations)
	}
	if st.AppliedReserved != st.Classes[ClassBothCommitted] || st.AllocationEffects != st.AppliedReserved {
		t.Fatalf("reserved %d allocation effects %d both-committed %d", st.AppliedReserved, st.AllocationEffects, st.Classes[ClassBothCommitted])
	}
	if st.StockReserved != st.AllocationQty || st.AllocationQty != st.PrimaryAllocatedQty || st.AllocationRows != st.PrimaryAllocatedRows {
		t.Fatalf("stock reserved %d allocation qty %d primary allocated qty %d rows %d/%d",
			st.StockReserved, st.AllocationQty, st.PrimaryAllocatedQty, st.AllocationRows, st.PrimaryAllocatedRows)
	}
}

// P-MIG-01: ten real process kills inside chunks, resume, reconcile, and a
// one-yen error on one row is detected.
func TestPostgresPMIG01ResumeAfterKillsAndDetectOneYen(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, _, path := h.planAndSnapshot(10)
	observedOrders := map[string]int{}
	for _, c := range plan.Chunks {
		observedOrders[c.ID] = c.ObservedOrders
	}
	prev := 0
	for cycle := 1; cycle <= 10; cycle++ {
		k := 1 + cycle%2 // 1: die in the first unfinished chunk, 2: finish one, die in the next
		h.child(fmt.Sprintf("%s:%d", CrashChunkMid, k), "run", "-plan", path)
		cps := h.count(h.primary, `SELECT count(*) FROM omdd.mig_checkpoint`)
		var inChunks int
		h.primary.QueryRow(h.ctx, `SELECT coalesce(sum(order_count),0) FROM omdd.mig_checkpoint`).Scan(&inChunks)
		backlog := h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog`)
		if backlog != inChunks {
			t.Fatalf("cycle %d: %d backlog rows but checkpoints cover %d: a killed chunk left rows", cycle, backlog, inChunks)
		}
		if cps != prev+k-1 {
			t.Fatalf("cycle %d: checkpoints %d, want %d", cycle, cps, prev+k-1)
		}
		prev = cps
	}
	observed(t, "checkpointsAfterTenKills", prev)
	if r := h.child("", "run", "-plan", path); r.code != 0 {
		t.Fatalf("final run exit %d: %s", r.code, r.stderr)
	}
	res, err := Reconcile(h.ctx, h.primary, plan)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("reconcile after kills: %+v", res.Differences)
	}
	if res.Source.OrderCount != syntheticOrders || res.Target.AmountSum != plan.Source.Observed.AmountSum || res.Target.RowsetDigest != plan.Source.Observed.RowsetDigest {
		t.Fatalf("target %+v plan %+v", res.Target, plan.Source.Observed)
	}
	observed(t, "reconcileAfterKills", map[string]any{"orders": res.Target.OrderCount, "lines": res.Target.LineCount, "amountSum": res.Target.AmountSum, "rowsetDigest": res.Target.RowsetDigest, "passed": res.Passed})

	h.exec(h.primary, `UPDATE omdd.order_backlog_line SET amount = amount + 1 WHERE order_no='SO-000042' AND line_no=1`)
	bad, err := Reconcile(h.ctx, h.primary, plan)
	if err != nil {
		t.Fatal(err)
	}
	rules := map[string]Difference{}
	for _, d := range bad.Differences {
		rules[d.Rule] = d
	}
	if bad.Passed || rules[RuleAmountSum].Source == "" || len(rules[RuleRowHash].Keys) != 1 || rules[RuleRowHash].Keys[0] != "SO-000042" {
		t.Fatalf("one yen not detected: %+v", bad.Differences)
	}
	if _, ok := rules[RuleOrderCount]; ok {
		t.Fatal("count rule reported a value difference")
	}
	observed(t, "oneYenDetection", bad.Differences)
	h.exec(h.primary, `UPDATE omdd.order_backlog_line SET amount = amount - 1 WHERE order_no='SO-000042' AND line_no=1`)
	if again, _ := Reconcile(h.ctx, h.primary, plan); !again.Passed {
		t.Fatalf("restored row still differs: %+v", again.Differences)
	}
}

// P-TX-01: kill right after the primary commit, before the send and after
// the inventory commit. Allocations equal backlog rows, none twice.
func TestPostgresPTX01KillAroundDelivery(t *testing.T) {
	h := newHarness(t, ampleStock())
	plan, _, path := h.planAndSnapshot(20)

	h.child(CrashAfterPrimaryCommit+":1", "run", "-plan", path)
	st := h.status(plan)
	if st.Orders != plan.Chunks[0].ObservedOrders || st.Classes[ClassPrimaryOnlyAwaiting] != st.Orders || st.AppliedReserved != 0 {
		t.Fatalf("after primary commit: %+v", st)
	}
	observed(t, "afterPrimaryCommit", st.Classes)

	h.child(CrashBeforeSend+":3", "run", "-plan", path)
	st = h.status(plan)
	if st.Classes[ClassBothCommitted] != 2 || st.Classes[ClassPrimaryOnlyAwaiting] != st.Orders-2 {
		t.Fatalf("before send: %+v", st.Classes)
	}
	observed(t, "beforeSend", st.Classes)

	h.child(CrashAfterInventoryCommit+":2", "run", "-plan", path)
	st = h.status(plan)
	if st.Classes[ClassInventoryOnlyUnacked] != 1 || st.Classes[ClassBothCommitted] != 3 {
		t.Fatalf("after inventory commit: %+v", st.Classes)
	}
	observed(t, "afterInventoryCommit", map[string]any{"classes": st.Classes, "undetermined": st.Undetermined})

	r := h.child("", "run", "-plan", path)
	if r.code != 0 {
		t.Fatalf("final run: %s", r.stderr)
	}
	var run RunResult
	if err := json.Unmarshal([]byte(r.stdout), &run); err != nil {
		t.Fatal(err)
	}
	if run.Dispatch.Replayed != 1 {
		t.Fatalf("the redelivered effect was not deduplicated by the receiver: %+v", run.Dispatch)
	}
	st, passes, err := WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Settled() || st.Orders != syntheticOrders || st.Classes[ClassBothCommitted] != syntheticOrders {
		t.Fatalf("not settled after %d passes: %+v", passes, st)
	}
	requireNoDoubleAllocation(t, st)
	observed(t, "settled", map[string]any{"orders": st.Orders, "appliedReserved": st.AppliedReserved, "allocationRows": st.AllocationRows,
		"allocationQty": st.AllocationQty, "stockReserved": st.StockReserved, "duplicates": st.Duplicates, "replayedOnResume": run.Dispatch.Replayed})
}

// P-TX-03: stock runs out part way. Every order ends both-committed or
// compensated; an order rejected but not yet compensated is undetermined.
func TestPostgresPTX03ShortageConvergesBothOrCompensated(t *testing.T) {
	stock := ampleStock()
	stock["ITM-F"] = 12
	h := newHarness(t, stock)
	plan, _, path := h.planAndSnapshot(15)

	h.child(CrashBeforeCompensation+":1", "run", "-plan", path)
	st := h.status(plan)
	if st.Classes[ClassCompensationPending] != 1 || st.Classes[ClassCompensated] != 0 {
		t.Fatalf("after first rejection: %+v", st.Classes)
	}
	observed(t, "rejectedBeforeCompensation", map[string]any{"classes": st.Classes, "undetermined": st.Undetermined})

	if r := h.child("", "run", "-plan", path); r.code != 0 {
		t.Fatalf("final run: %s", r.stderr)
	}
	st, _, err := WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Settled() || st.Classes[ClassCompensated] == 0 || st.Classes[ClassBothCommitted] == 0 || st.Compensations != st.Classes[ClassCompensated] {
		t.Fatalf("not converged: %+v", st)
	}
	requireNoDoubleAllocation(t, st)
	var fReserved, fOnHand int64
	h.inventory.QueryRow(h.ctx, `SELECT reserved, on_hand FROM omdd.stock WHERE item_code='ITM-F'`).Scan(&fReserved, &fOnHand)
	if fReserved > fOnHand {
		t.Fatalf("over-reserved ITM-F %d/%d", fReserved, fOnHand)
	}
	observed(t, "converged", map[string]any{"classes": st.Classes, "compensations": st.Compensations, "itmFReserved": fReserved, "itmFOnHand": fOnHand})

	// A stale relay redelivers a compensated effect after stock arrives.
	var key, effect string
	h.primary.QueryRow(h.ctx, `SELECT order_no, effect_id FROM omdd.compensation ORDER BY order_no LIMIT 1`).Scan(&key, &effect)
	h.exec(h.inventory, `UPDATE omdd.stock SET on_hand = on_hand + 1000 WHERE item_code='ITM-F'`)
	h.exec(h.primary, `UPDATE omdd.effect_outbox SET state='pending' WHERE effect_id=$1`, effect)
	d, err := Dispatch(h.ctx, h.primary, h.inventoryDSN, h.opt)
	if err != nil {
		t.Fatal(err)
	}
	after := h.status(plan)
	if d.Replayed != 1 || d.Reserved != 0 || h.count(h.inventory, `SELECT count(*) FROM omdd.allocation WHERE effect_id=$1`, effect) != 0 || !after.Settled() {
		t.Fatalf("late redelivery after compensation changed state: %+v %+v", d, after.Classes)
	}
	observed(t, "lateRedeliveryAfterCompensation", map[string]any{"order": key, "replayed": d.Replayed, "reserved": d.Reserved})
}

func docker(t *testing.T, args ...string) {
	t.Helper()
	name := args[len(args)-1]
	if !strings.HasPrefix(name, containerPrefix) {
		t.Fatalf("refuse to touch container %q (prefix %q required)", name, containerPrefix)
	}
	if out, err := exec.Command("docker", args...).CombinedOutput(); err != nil {
		t.Fatalf("docker %v: %v %s", args, err, out)
	}
}

func inventoryContainer(t *testing.T) string {
	t.Helper()
	name := os.Getenv(containerEnv)
	if name == "" {
		t.Skip("requires " + containerEnv + " (a tei-omdd-* container); skip is not a pass")
	}
	if !strings.HasPrefix(name, containerPrefix) {
		t.Fatalf("%s=%q does not start with %q", containerEnv, name, containerPrefix)
	}
	return name
}

// P-DB-02: with the inventory DB hanging, mid-pass hanging, or stopped, the
// primary side answers within a bound, every waiting allocation is recorded,
// and all are resolved after recovery.
func TestPostgresPDB02InventoryStopBoundedAndResolved(t *testing.T) {
	container := inventoryContainer(t)
	h := newHarness(t, ampleStock())
	plan, _, _ := h.planAndSnapshot(20)
	bound := h.opt.ConnectTimeout + h.opt.OpTimeout + 5*time.Second
	paused := false
	t.Cleanup(func() {
		if paused {
			exec.Command("docker", "unpause", container).Run()
		}
		exec.Command("docker", "start", container).Run()
	})

	// 1. Inventory frozen (accepts TCP, never answers) for the whole run.
	docker(t, "pause", container)
	paused = true
	start := time.Now()
	run, err := Run(h.ctx, h.primary, h.inventoryDSN, plan, h.opt)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > bound || run.Dispatch.InventoryReachable || len(run.Migrate.ChunksWritten) != len(plan.Chunks) {
		t.Fatalf("paused inventory: elapsed %v (bound %v) %+v", elapsed, bound, run)
	}
	recorded := h.count(h.primary, `SELECT count(*) FROM omdd.effect_outbox WHERE state='pending' AND last_outcome=$1 AND attempts=1`, OutcomeUnavailable)
	awaiting := h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog WHERE allocation_state='awaiting'`)
	if recorded != syntheticOrders || awaiting != syntheticOrders {
		t.Fatalf("awaiting allocations not all recorded: %d recorded, %d awaiting", recorded, awaiting)
	}
	observed(t, "pausedWholeRun", map[string]any{"elapsedSeconds": elapsed.Seconds(), "boundSeconds": bound.Seconds(), "recordedAwaiting": recorded, "chunksWritten": len(run.Migrate.ChunksWritten)})
	docker(t, "unpause", container)
	paused = false

	// 2. Inventory freezes in the middle of a dispatch pass.
	beforeSendHook = func(i int) {
		if i == 5 {
			docker(t, "pause", container)
			paused = true
		}
	}
	start = time.Now()
	d, err := Dispatch(h.ctx, h.primary, h.inventoryDSN, h.opt)
	elapsed = time.Since(start)
	beforeSendHook = nil
	if err != nil {
		t.Fatal(err)
	}
	if elapsed > bound || d.Reserved != 5 || d.Unknown != 1 || d.LeftPending != syntheticOrders-5 {
		t.Fatalf("mid-pass freeze: elapsed %v %+v", elapsed, d)
	}
	byOutcome := map[string]int{}
	rows, _ := h.primary.Query(h.ctx, `SELECT last_outcome, count(*) FROM omdd.effect_outbox WHERE state='pending' GROUP BY last_outcome`)
	for rows.Next() {
		var o string
		var n int
		rows.Scan(&o, &n)
		byOutcome[o] = n
	}
	rows.Close()
	if byOutcome[OutcomeUnknown] != 1 || byOutcome[OutcomeNotAttempt] != syntheticOrders-6 {
		t.Fatalf("pending outcomes %v", byOutcome)
	}
	observed(t, "midPassFreeze", map[string]any{"elapsedSeconds": elapsed.Seconds(), "reserved": d.Reserved, "unknown": d.Unknown, "pendingByOutcome": byOutcome})
	docker(t, "unpause", container)
	paused = false

	// 3. Inventory stopped (connection refused).
	docker(t, "stop", container)
	start = time.Now()
	d, err = Dispatch(h.ctx, h.primary, h.inventoryDSN, h.opt)
	elapsed = time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	st := h.status(plan)
	if elapsed > bound || d.InventoryReachable || st.InventoryObserved || st.Classes[ClassInventoryUnobserved] != syntheticOrders {
		t.Fatalf("stopped inventory: elapsed %v %+v %+v", elapsed, d, st.Classes)
	}
	observed(t, "stopped", map[string]any{"elapsedSeconds": elapsed.Seconds(), "statusClasses": st.Classes})

	docker(t, "start", container)
	st, passes, err := WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 90*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Settled() || st.Classes[ClassBothCommitted] != syntheticOrders || st.PrimaryStates["awaiting"] != 0 {
		t.Fatalf("not resolved after recovery (%d passes): %+v", passes, st)
	}
	requireNoDoubleAllocation(t, st)
	observed(t, "recovered", map[string]any{"passes": passes, "bothCommitted": st.Classes[ClassBothCommitted], "awaiting": st.PrimaryStates["awaiting"]})
}

// The through run: steps 1-3 via the CLI in separate processes, carrying the
// plan digest and resume points, with kills, a shortage and an inventory
// stop, and negative cases for a re-made or edited plan.
func TestPostgresThroughSlice1(t *testing.T) {
	stock := ampleStock()
	stock["ITM-F"] = 20
	h := newHarness(t, stock)
	container := os.Getenv(containerEnv)
	planPath := filepath.Join(h.dir, "plan.json")
	ok := func(r childResult, what string) childResult {
		t.Helper()
		if r.code != 0 {
			t.Fatalf("%s exit %d: %s", what, r.code, r.stderr)
		}
		return r
	}

	var planOut struct {
		PlanDigest string `json:"planDigest"`
	}
	json.Unmarshal([]byte(ok(h.child("", "plan", "-chunk-size", "12", "-out", planPath), "plan").stdout), &planOut)
	plan, err := ReadPlan(planPath)
	if err != nil || plan.Digest != planOut.PlanDigest {
		t.Fatalf("plan %v %s/%s", err, plan.Digest, planOut.PlanDigest)
	}
	var snapOut struct {
		Snapshot SnapshotRecord `json:"snapshot"`
	}
	json.Unmarshal([]byte(ok(h.child("", "snapshot", "-plan", planPath), "snapshot").stdout), &snapOut)
	snap := snapOut.Snapshot
	if snap.PlanDigest != plan.Digest || !snap.MatchesPlanObservation || snap.Position.WALLSN == "" || snap.Position.PGSnapshot == "" {
		t.Fatalf("snapshot %+v", snap)
	}
	var again struct {
		Snapshot SnapshotRecord `json:"snapshot"`
		Existing bool           `json:"existing"`
	}
	json.Unmarshal([]byte(ok(h.child("", "snapshot", "-plan", planPath), "snapshot again").stdout), &again)
	if !again.Existing || again.Snapshot.Digest != snap.Digest {
		t.Fatal("second snapshot moved the position")
	}

	h.child(CrashChunkMid+":2", "run", "-plan", planPath)
	h.child(CrashAfterPrimaryCommit+":2", "run", "-plan", planPath)
	h.child(CrashAfterInventoryCommit+":4", "run", "-plan", planPath)
	h.child(CrashBeforeCompensation+":1", "run", "-plan", planPath)
	mid := h.status(plan)
	undetermined := 0
	for c, n := range mid.Classes {
		if strings.HasPrefix(c, "undetermined:") {
			undetermined += n
		}
	}
	if undetermined == 0 {
		t.Fatalf("no undetermined orders recorded mid-way: %+v", mid.Classes)
	}
	observed(t, "midwayClasses", mid.Classes)
	if container != "" {
		docker(t, "stop", container)
		start := time.Now()
		ok(h.child("", "run", "-plan", planPath, "-connect-timeout", "2s", "-op-timeout", "2s"), "run with inventory stopped")
		observed(t, "runWithInventoryStoppedSeconds", time.Since(start).Seconds())
		docker(t, "start", container)
	}
	ok(h.child("", "run", "-plan", planPath), "final run")
	st, _, err := WaitSettled(h.ctx, h.primary, h.inventoryDSN, plan, h.opt, 90*time.Second)
	if err != nil || !st.Settled() {
		t.Fatalf("not settled: %v %+v", err, st)
	}
	requireNoDoubleAllocation(t, st)

	var rec ReconcileResult
	json.Unmarshal([]byte(ok(h.child("", "reconcile", "-plan", planPath), "reconcile").stdout), &rec)
	if !rec.Passed || rec.PlanDigest != plan.Digest || rec.SnapshotDigest != snap.Digest {
		t.Fatalf("reconcile %+v", rec)
	}
	carried := map[string]int{
		"checkpointsWithOtherDigest": h.count(h.primary, `SELECT count(*) FROM omdd.mig_checkpoint WHERE plan_digest<>$1 OR snapshot_digest<>$2`, plan.Digest, snap.Digest),
		"outboxWithOtherDigest":      h.count(h.primary, `SELECT count(*) FROM omdd.effect_outbox WHERE plan_digest<>$1`, plan.Digest),
		"backlogWithOtherDigest":     h.count(h.primary, `SELECT count(*) FROM omdd.order_backlog WHERE plan_digest<>$1`, plan.Digest),
	}
	for k, n := range carried {
		if n != 0 {
			t.Fatalf("%s: %d", k, n)
		}
	}

	// Negative: a plan re-made with another chunking has another digest.
	remade := filepath.Join(h.dir, "plan-remade.json")
	ok(h.child("", "plan", "-chunk-size", "7", "-out", remade), "re-plan")
	negatives := map[string]string{}
	for _, cmd := range []string{"migrate", "reconcile", "status"} {
		r := h.child("", cmd, "-plan", remade)
		if r.code == 0 || !strings.Contains(r.stderr, "plan digest mismatch") {
			t.Fatalf("%s with re-made plan: exit %d %s", cmd, r.code, r.stderr)
		}
		negatives["remade:"+cmd] = strings.TrimSpace(r.stderr)
	}
	// Negative: the plan file edited without resealing.
	raw, _ := os.ReadFile(planPath)
	edited := filepath.Join(h.dir, "plan-edited.json")
	os.WriteFile(edited, bytes.Replace(raw, []byte(`"chunkSize": 12`), []byte(`"chunkSize": 13`), 1), 0o644)
	if r := h.child("", "reconcile", "-plan", edited); r.code == 0 || !strings.Contains(r.stderr, "plan digest mismatch") {
		t.Fatalf("edited plan accepted: %s", r.stderr)
	} else {
		negatives["edited:reconcile"] = strings.TrimSpace(r.stderr)
	}
	observed(t, "through", map[string]any{"planDigest": plan.Digest, "snapshotDigest": snap.Digest, "walLsn": snap.Position.WALLSN,
		"pgSnapshot": snap.Position.PGSnapshot, "reconcilePassed": rec.Passed, "orders": rec.Target.OrderCount, "amountSum": rec.Target.AmountSum,
		"classes": st.Classes, "carried": carried, "negatives": negatives})
}
