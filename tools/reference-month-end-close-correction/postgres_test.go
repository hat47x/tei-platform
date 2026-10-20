//go:build unix

package monthendclose

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// P-TIME-05: a temporary holiday in the next month, recorded the day before
// the close date, moves the close job and the due dates that fall on it; the
// difference is recorded, and the old decision stops later steps.
func TestPostgresPTIME05CalendarRevisionImpact(t *testing.T) {
	h := newHarness(t)
	d1 := h.decide(1, "2026-09-28T09:00:00+09:00").Decision
	v2 := h.revise(1, tempHoliday, "2026-09-29T10:00:00+09:00")
	if _, err := Decide(h.ctx, h.conn, month, "jp-business", 2, testRule(t), mustTime(t, "2026-09-28T09:00:00+09:00")); !errors.Is(err, ErrCalendarNotYetKnown) {
		t.Fatalf("v2 used before it was recorded: %v", err)
	}
	r2 := h.decide(2, "2026-09-29T12:00:00+09:00")
	if !r2.Activated || r2.Impact == nil || r2.PreviousDig != d1.Digest {
		t.Fatalf("decision v2 %+v", r2)
	}
	im := *r2.Impact
	if len(im.JobShifts) != 1 || im.JobShifts[0].Name != JobClose || im.JobShifts[0].From != "2026-10-01T00:00:00+09:00" || im.JobShifts[0].To != "2026-10-02T00:00:00+09:00" {
		t.Fatalf("job shifts %+v", im.JobShifts)
	}
	dueOnClose := []string{}
	for _, s := range im.DueShifts {
		if s.DueOnCloseDate {
			dueOnClose = append(dueOnClose, s.PayableID)
		}
	}
	if len(im.DueShifts) != 2 || len(dueOnClose) != 2 || dueOnClose[0] != "P-001" || dueOnClose[1] != "P-007" {
		t.Fatalf("due shifts %+v", im.DueShifts)
	}
	if n := h.int(`SELECT count(*) FROM mecc.calendar_impact WHERE from_version=1 AND to_version=2 AND target_month=$1`, month); n != 1 {
		t.Fatalf("impact records %d", n)
	}
	if h.str(`SELECT active_digest FROM mecc.close_month WHERE target_month=$1`, month) != r2.Decision.Digest {
		t.Fatal("v2 decision is not active")
	}
	if err := RequireActive(h.ctx, h.conn, d1); !errors.Is(err, ErrDecisionMismatch) {
		t.Fatalf("old decision still accepted: %v", err)
	}
	// A registered version cannot be replaced with other content.
	edited := testCalendar(t)
	edited.Holidays = append(edited.Holidays, Holiday{"2026-12-31", "x"})
	edited.Digest = CalendarDigest(edited)
	if _, err := LoadCalendar(h.ctx, h.conn, edited); err == nil {
		t.Fatal("calendar v1 was replaced")
	}
	// The scheduled close job moved: at 01:00 on 1 October it is refused.
	_, err := RunClose(h.ctx, h.conn, r2.Decision, CloseOptions{OwnerID: "srv-a", TTL: 3 * time.Second, ChunkSize: 5, AsOf: mustTime(t, "2026-10-01T01:00:00+09:00")})
	if !errors.Is(err, ErrNotYetScheduled) {
		t.Fatalf("close before the moved schedule: %v", err)
	}
	observed(t, "impact", map[string]any{"jobShifts": im.JobShifts, "dueShifts": im.DueShifts, "addedHolidays": im.AddedHolidays,
		"calendarDigests": []string{d1.CalendarDigest, v2.Digest}, "closeRefusedAt": "2026-10-01T01:00:00+09:00"})
}

// P-INT-01: a CP932 fixed-length file with mapping-sensitive codes,
// half-width kana, a field-filling double-byte text, a duplicate code, a
// user-defined character and a straddling character. Re-emission is byte
// for byte; a second copy, a sequence reuse, a count mismatch, a truncated
// file and a late file are rejected whole; a killed import leaves nothing.
func TestPostgresPINT01FixedLengthImportAndRejections(t *testing.T) {
	h := newHarness(t)
	d := h.decide(1, "2026-09-28T09:00:00+09:00").Decision
	rows := syntheticShipments(t)
	f1 := buildFile(t, rows, fileOpts{seq: 1})
	recv := mustTime(t, receivedBeforeCutoff)
	res, err := ImportFile(h.ctx, h.conn, d, h.spec, f1, recv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Records != 30 || res.Applied != 27 || len(res.Held) != 3 {
		t.Fatalf("import %+v", res)
	}
	out, err := Reemit(h.ctx, h.conn, h.spec, res.ImportID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, f1) {
		t.Fatalf("re-emitted %d bytes differ from the original %d bytes", len(out), len(f1))
	}
	stored := h.str(`SELECT item_name FROM mecc.shipment_event WHERE shipment_id='SH-0001'`)
	if stored != "波～ダッシュ品" {
		t.Fatalf("stored %q", stored)
	}
	before := h.int(`SELECT count(*) FROM mecc.shipment_event`)
	reject := func(name string, b []byte, at time.Time, want string) {
		t.Helper()
		_, err := ImportFile(h.ctx, h.conn, d, h.spec, b, at)
		if reason, ok := IsReject(err); !ok || reason != want {
			t.Fatalf("%s: %v, want %s", name, err, want)
		}
	}
	reject("same file again", f1, recv, RejectDuplicateFile)
	reject("trailer count off by one", buildFile(t, rows[:10], fileOpts{seq: 2, countDelta: 1}), recv, RejectTrailerCount)
	reject("truncated", buildFile(t, rows[:10], fileOpts{seq: 2, truncateTail: 50}), recv, RejectTruncated)
	reject("sequence reused with other content", buildFile(t, rows[:10], fileOpts{seq: 1}), recv, RejectSeqReused)
	reject("after cut-off", buildFile(t, rows[:10], fileOpts{seq: 2}), mustTime(t, "2026-09-30T17:00:01+09:00"), RejectAfterCutoff)
	reject("other month", buildFile(t, rows[:10], fileOpts{seq: 2, month: "202610"}), recv, RejectMonth)
	if n := h.int(`SELECT count(*) FROM mecc.import_unit`); n != 1 {
		t.Fatalf("%d import units after rejections", n)
	}
	if n := h.int(`SELECT count(*) FROM mecc.shipment_event`); n != before {
		t.Fatalf("rejected files changed shipments: %d -> %d", before, n)
	}
	if n := h.int(`SELECT count(*) FROM mecc.import_rejection`); n != 6 {
		t.Fatalf("%d rejection records", n)
	}
	// A second file of new shipments, killed before commit, then imported.
	var more []shipmentRow
	for i := 31; i <= 35; i++ {
		more = append(more, shipmentRow{ShipmentID: fmt.Sprintf("SH-%04d", i), OrderNo: fmt.Sprintf("SO-%04d", i), ShipDate: "20260929",
			Qty: 2, Amount: int64(500 * i), Item: sjis(t, "追加品"), Note: []byte("M")})
	}
	f2 := h.write("f2.dat", buildFile(t, more, fileOpts{seq: 2}))
	decisionPath := h.writeJSON("d.json", d)
	args := []string{"import", "-decision", decisionPath, "-spec", "testdata/file-spec.json", "-file", f2, "-received-at", receivedBeforeCutoff}
	h.killed(CrashImportBeforeCommit+":1", args...)
	if n := h.int(`SELECT count(*) FROM mecc.import_unit`); n != 1 {
		t.Fatalf("killed import left %d units", n)
	}
	if n := h.int(`SELECT count(*) FROM mecc.shipment_event`); n != before {
		t.Fatalf("killed import left shipments: %d", n)
	}
	h.run(0, nil, args...)
	h.run(4, nil, args...)
	if n := h.int(`SELECT count(*) FROM mecc.shipment_event`); n != before+5 {
		t.Fatalf("shipments %d, want %d", n, before+5)
	}
	reasons := map[string]int64{}
	rows2, _ := h.conn.Query(h.ctx, `SELECT reason, count(*) FROM mecc.import_rejection GROUP BY reason`)
	for rows2.Next() {
		var r string
		var n int64
		rows2.Scan(&r, &n)
		reasons[r] = n
	}
	rows2.Close()
	held := map[string]string{}
	for _, x := range res.Held {
		held[x.ShipmentID] = x.Reason
	}
	observed(t, "import", map[string]any{"records": res.Records, "applied": res.Applied, "held": held, "reemitBytes": len(out),
		"reemitEqual": bytes.Equal(out, f1), "fileSha256": res.FileSHA256, "rejections": reasons, "storedWaveDash": fmt.Sprintf("%U", []rune(stored)[1])})
}

// P-INT-02: the same shipments arrive by message (JIS-mapped text,
// redelivered) and by file (after a truncated copy is rejected). Each
// shipment is one event, conflicts are held, and the reconciliation is
// recorded.
func TestPostgresPINT02FileAndMessageOneEvent(t *testing.T) {
	h := newHarness(t)
	d := h.decide(1, "2026-09-28T09:00:00+09:00").Decision
	rows := syntheticShipments(t)
	msg := func(i int, id string) Message {
		r := rows[i]
		sd := r.ShipDate
		return Message{MessageID: id, Event: ShipmentEvent{ShipmentID: r.ShipmentID, OrderNo: r.OrderNo, ShipDate: sd[:4] + "-" + sd[4:6] + "-" + sd[6:],
			Qty: r.Qty, Amount: r.Amount, ItemName: r.ItemText}}
	}
	deliver := func(m Message, want string) {
		t.Helper()
		got, err := ReceiveMessage(h.ctx, h.conn, d, h.spec, m)
		if err != nil || got != want {
			t.Fatalf("%s: %s %v, want %s", m.MessageID, got, err, want)
		}
	}
	for i := 0; i < 10; i++ {
		deliver(msg(i, fmt.Sprintf("m-%d", i)), OutcomeApplied)
		if i%3 == 0 {
			deliver(msg(i, fmt.Sprintf("m-%d", i)), OutcomeRedelivered)
		}
	}
	recv := mustTime(t, receivedBeforeCutoff)
	if _, err := ImportFile(h.ctx, h.conn, d, h.spec, buildFile(t, rows, fileOpts{seq: 1, truncateTail: 82 * 12}), recv); err == nil {
		t.Fatal("truncated file accepted")
	}
	res, err := ImportFile(h.ctx, h.conn, d, h.spec, buildFile(t, rows, fileOpts{seq: 1}), recv)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcomes[OutcomeConfirmed] != 7 || res.Outcomes[OutcomeApplied] != 20 {
		t.Fatalf("file outcomes %v", res.Outcomes)
	}
	for i := 10; i < 15; i++ {
		deliver(msg(i, fmt.Sprintf("m-%d", i)), OutcomeConfirmed)
		deliver(msg(i, fmt.Sprintf("m-%d", i)), OutcomeRedelivered)
	}
	bad := msg(19, "m-19-conflict")
	bad.Event.Qty++
	deliver(bad, OutcomeConflict)
	rec, err := ReconcileShipments(h.ctx, h.conn, d)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{ClassBoth: 12, ClassFileOnly: 14, ClassFileHeld: 3, ClassConflict: 1}
	for k, v := range want {
		if rec.Classes[k] != v {
			t.Fatalf("classes %v, want %v", rec.Classes, want)
		}
	}
	if rec.Events != 30 || h.int(`SELECT count(DISTINCT shipment_id) FROM mecc.shipment_event`) != 30 {
		t.Fatalf("events %d", rec.Events)
	}
	var wantQty, wantAmount int64
	for _, r := range rows {
		wantQty += r.Qty
		wantAmount += r.Amount
	}
	if rec.Qty != wantQty || rec.Amount != wantAmount {
		t.Fatalf("final qty %d amount %d, want %d %d (the conflicting message must not overwrite)", rec.Qty, rec.Amount, wantQty, wantAmount)
	}
	if n := h.int(`SELECT count(*) FROM mecc.shipment_reconciliation WHERE run_id=$1 AND decision_digest=$2`, rec.RunID, d.Digest); n != 30 {
		t.Fatalf("reconciliation rows %d", n)
	}
	observed(t, "reconciliation", map[string]any{"classes": rec.Classes, "events": rec.Events, "receipts": rec.Receipts, "deliveries": rec.Deliveries,
		"deliveryLog": h.int(`SELECT count(*) FROM mecc.delivery_log`), "qty": rec.Qty, "amount": rec.Amount})
}

// closeFixture prepares a month ready to close: decision, one imported file
// and the entries.
func (h *harness) closeFixture() (Decision, string, map[string]Entry) {
	h.t.Helper()
	d := h.decide(1, "2026-09-28T09:00:00+09:00").Decision
	if _, err := ImportFile(h.ctx, h.conn, d, h.spec, buildFile(h.t, syntheticShipments(h.t), fileOpts{seq: 1}), mustTime(h.t, receivedBeforeCutoff)); err != nil {
		h.t.Fatal(err)
	}
	return d, h.writeJSON("decision.json", d), h.seedEntries(d)
}

func closeArgs(decision, owner, ttl, pause string) []string {
	args := []string{"close", "-decision", decision, "-owner", owner, "-ttl", ttl, "-chunk-size", "5", "-as-of", "2026-10-01T00:30:00+09:00"}
	if pause != "" {
		args = append(args, "-chunk-pause", pause)
	}
	return args
}

// P-JOB-01: the close is killed inside a chunk, a restart while the lease is
// alive is refused, the restart after expiry resumes, a concurrent second
// launch is refused, a launch after completion is refused, and the output is
// exactly one copy.
func TestPostgresPJOB01KillRestartDoubleAndAfterComplete(t *testing.T) {
	h := newHarness(t)
	_, dpath, _ := h.closeFixture()
	h.killed(CrashCloseChunkMid+":3", closeArgs(dpath, "srv-a", "3s", "")...)
	if n := h.int(`SELECT count(*) FROM mecc.close_chunk`); n != 2 {
		t.Fatalf("chunks after kill %d", n)
	}
	if n := h.int(`SELECT count(*) FROM mecc.close_detail`); n != 10 {
		t.Fatalf("details after kill %d: a killed chunk left rows", n)
	}
	refusedWith(t, h.run(4, nil, closeArgs(dpath, "srv-b", "3s", "")...), ErrAlreadyRunning.Error())
	time.Sleep(3500 * time.Millisecond)
	resumer := h.start(nil, closeArgs(dpath, "srv-b", "3s", "300ms")...)
	for h.int(`SELECT count(*) FROM mecc.close_attempt WHERE outcome='resumed'`) == 0 {
		time.Sleep(50 * time.Millisecond)
	}
	second := h.run(4, nil, closeArgs(dpath, "srv-c", "3s", "")...)
	refusedWith(t, second, ErrAlreadyRunning.Error())
	h.wait(resumer)
	if resumer.code != 0 {
		t.Fatalf("resume exit %d\n%s", resumer.code, resumer.stderr())
	}
	r := decodeOut[CloseResult](t, resumer)
	if !r.Takeover || r.FencingToken != 2 || r.Starts != 2 {
		t.Fatalf("resume %+v", r)
	}
	after := h.run(4, nil, closeArgs(dpath, "srv-d", "3s", "")...)
	refusedWith(t, after, ErrAlreadyCompleted.Error())
	inv := h.closeInvariants()
	outcomes := map[string]int64{}
	rows, _ := h.conn.Query(h.ctx, `SELECT outcome, count(*) FROM mecc.close_attempt GROUP BY outcome`)
	for rows.Next() {
		var o string
		var n int64
		rows.Scan(&o, &n)
		outcomes[o] = n
	}
	rows.Close()
	if outcomes["refused-running"] != 2 || outcomes["refused-completed"] != 1 || outcomes["completed"] != 1 {
		t.Fatalf("attempts %v", outcomes)
	}
	observed(t, "close", map[string]any{"invariants": inv, "attempts": outcomes, "resume": map[string]any{"token": r.FencingToken, "starts": r.Starts, "chunksThisRun": r.ChunksThisRun}})
}

// P-OPS-01: the owner (host clock 10 minutes slow) is stopped with a chunk
// selected; a server whose host clock is 10 minutes fast is refused while
// the lease is alive by the database clock, although a host-clock design
// would take over; after expiry it takes over with a larger token and
// finishes; the old owner resumes and its write is refused by the database.
func TestPostgresPOPS01FencingAfterTakeoverWithClockSkew(t *testing.T) {
	h := newHarness(t)
	_, dpath, _ := h.closeFixture()
	a := h.start(map[string]string{StopEnv: PointBeforeChunkWrite + ":2", ClockOffsetEnv: "-10m"}, closeArgs(dpath, "srv-a", "2s", "")...)
	h.waitMarker(a, StopMarker+" "+PointBeforeChunkWrite)
	fast := map[string]string{ClockOffsetEnv: "+10m"}
	refusedWith(t, h.run(4, fast, closeArgs(dpath, "srv-b", "2s", "")...), ErrAlreadyRunning.Error())
	var naive bool
	if err := h.conn.QueryRow(h.ctx, `SELECT naive_would_take_over FROM mecc.lease_event WHERE owner_id='srv-b' AND state='claim-refused'`).Scan(&naive); err != nil || !naive {
		t.Fatalf("host-clock judgment not recorded as a takeover: %v %v", naive, err)
	}
	time.Sleep(2500 * time.Millisecond)
	b := h.start(fast, closeArgs(dpath, "srv-b", "2s", "300ms")...)
	for h.int(`SELECT count(*) FROM mecc.lease_event WHERE owner_id='srv-b' AND state='takeover'`) == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	// The old owner resumes while the new owner holds the lease.
	if err := a.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		t.Fatal(err)
	}
	h.wait(a)
	if a.code != 4 {
		t.Fatalf("old owner exit %d\n%s", a.code, a.stderr())
	}
	refusedWith(t, a, "is not the current token 2")
	h.wait(b)
	if b.code != 0 {
		t.Fatalf("new owner exit %d\n%s", b.code, b.stderr())
	}
	rb := decodeOut[CloseResult](t, b)
	if !rb.Takeover || rb.FencingToken != 2 {
		t.Fatalf("takeover %+v", rb)
	}
	if n := h.int(`SELECT count(*) FROM mecc.close_attempt WHERE owner_id='srv-a' AND outcome='fenced'`); n != 1 {
		t.Fatalf("fenced attempts %d", n)
	}
	if n := h.int(`SELECT count(*) FROM mecc.close_detail WHERE fencing_token=1`); n != 5 {
		t.Fatalf("token-1 details %d, want the first chunk only", n)
	}
	// A direct write with the stale token is refused by the trigger even for
	// an entry that has no detail yet.
	_, err := h.conn.Exec(h.ctx, `INSERT INTO mecc.close_total VALUES ($1,0,0,0,0,0,'x',1,'x',1,'srv-a') ON CONFLICT DO NOTHING`, month)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != FenceSQLState {
		t.Fatalf("stale token write: %v", err)
	}
	// Reverse skew, in process: the holder's host clock is fast, the
	// claimant's slow. After the database-clock expiry the claimant takes
	// over although its host clock says the lease has 20 minutes left.
	key := "skew-probe"
	slow := &PgOwnershipStore{Conn: h.conn, HostNow: func() time.Time { return time.Now().Add(-10 * time.Minute) }}
	fastStore := &PgOwnershipStore{Conn: h.conn, HostNow: func() time.Time { return time.Now().Add(10 * time.Minute) }}
	o1, _, err := fastStore.Claim(h.ctx, key, "x", 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	var hostExp time.Time
	h.conn.QueryRow(h.ctx, `SELECT owner_host_expires_at FROM mecc.job_lease WHERE job_key=$1`, key).Scan(&hostExp)
	time.Sleep(700 * time.Millisecond)
	o2, takeover, err := slow.Claim(h.ctx, key, "y", time.Second)
	if err != nil || !takeover || o2.FencingToken != o1.FencingToken+1 {
		t.Fatalf("takeover after db expiry: %+v %v %v", o2, takeover, err)
	}
	naiveWait := slow.HostNow().Before(hostExp)
	if !naiveWait {
		t.Fatal("the reverse skew does not exercise a host-clock wait")
	}
	if _, err := fastStore.Heartbeat(h.ctx, o1, time.Second); !errors.Is(err, ErrFenced) {
		t.Fatalf("old heartbeat: %v", err)
	}
	inv := h.closeInvariants()
	observed(t, "fencing", map[string]any{"hostClockDesignWouldTakeOverWhileAlive": naive, "hostClockDesignWouldWaitAfterExpiry": naiveWait,
		"takeoverToken": rb.FencingToken, "oldOwnerExit": a.code, "oldOwnerStderr": lastLine(a.stderr()), "invariants": inv})
}

func lastLine(s string) string {
	b := bytes.TrimRight([]byte(s), "\n")
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		return string(b[i+1:])
	}
	return string(b)
}

// P-TIME-01: during the close, back-dated corrections and new orders keep
// arriving, and the 23:59:59 order is allocated mid-close. The close equals
// its details and the entries posted to the month; the boundary orders land
// in the month their local business date says, not the UTC date.
func TestPostgresPTIME01OnlineWritesDuringCloseAndBoundary(t *testing.T) {
	h := newHarness(t)
	d, dpath, seeded := h.closeFixture()
	c := h.start(nil, closeArgs(dpath, "srv-a", "3s", "120ms")...)
	for h.int(`SELECT count(*) FROM mecc.close_run`) == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	var during []Entry
	for i := 0; ; i++ {
		select {
		case <-c.done:
		default:
			corr, err := PlaceEntry(h.ctx, h.conn, d, EntryInput{OrderNo: fmt.Sprintf("SO-9%03d", 1+i%53), Kind: "correction", AcceptedAt: mustTime(t, "2026-09-15T10:00:00+09:00"), Amount: -100})
			if err != nil {
				t.Fatal(err)
			}
			neu, err := PlaceEntry(h.ctx, h.conn, d, EntryInput{OrderNo: fmt.Sprintf("SO-A%03d", i), AcceptedAt: mustTime(t, "2026-10-01T09:00:00+09:00").Add(time.Duration(i) * time.Second), Amount: 300})
			if err != nil {
				t.Fatal(err)
			}
			during = append(during, corr, neu)
			time.Sleep(30 * time.Millisecond)
			continue
		}
		break
	}
	h.wait(c)
	if c.code != 0 {
		t.Fatalf("close exit %d\n%s", c.code, c.stderr())
	}
	if len(during) < 4 {
		t.Fatalf("only %d entries arrived during the close", len(during))
	}
	for _, e := range during {
		if e.PostingMonth != "2026-10" {
			t.Fatalf("entry during close posted to %s: %+v", e.PostingMonth, e)
		}
		if e.BusinessMonth == month && e.PostingReason != PostingRedirected {
			t.Fatalf("correction not redirected: %+v", e)
		}
	}
	inv := h.closeInvariants()
	in := func(no string) bool {
		return h.int(`SELECT count(*) FROM mecc.close_detail WHERE entry_id=$1`, seeded[no].ID) == 1
	}
	if !in("SO-SEP-LAST") || !in("SO-SEP-FIRST") || in("SO-OCT-FIRST") || in("SO-AUG-LAST") {
		t.Fatal("boundary orders in the wrong month")
	}
	// The same rows read by the UTC date of the database session.
	h.exec(`SET TimeZone='UTC'`)
	utcMonth := func(no string) string {
		return h.str(`SELECT to_char(accepted_at::date,'YYYY-MM') FROM mecc.sales_entry WHERE entry_id=$1`, seeded[no].ID)
	}
	naive := map[string]string{"SO-OCT-FIRST": utcMonth("SO-OCT-FIRST"), "SO-SEP-FIRST": utcMonth("SO-SEP-FIRST")}
	if naive["SO-OCT-FIRST"] != "2026-09" || naive["SO-SEP-FIRST"] != "2026-08" {
		t.Fatalf("the boundary does not exercise the zone: %v", naive)
	}
	// Writers that bypass the online path are refused by the period guard.
	_, err := h.conn.Exec(h.ctx, `UPDATE mecc.sales_entry SET amount=amount+1 WHERE entry_id=$1`, seeded["SO-9001"].ID)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "TP001" {
		t.Fatalf("update of a closed period: %v", err)
	}
	// The 23:59:59 order was still pending allocation at the close: it is in
	// September as pending, and allocating it afterwards changes neither its
	// month nor the closed totals.
	awaitingAtClose := h.str(`SELECT alloc_status FROM mecc.close_detail WHERE entry_id=$1`, seeded["SO-SEP-LAST"].ID)
	if awaitingAtClose != "awaiting" || h.int(`SELECT awaiting FROM mecc.close_total WHERE month=$1`, month) != 1 {
		t.Fatalf("pending order at close: %s", awaitingAtClose)
	}
	if err := AllocateEntry(h.ctx, h.conn, seeded["SO-SEP-LAST"].ID); err != nil {
		t.Fatal(err)
	}
	if h.str(`SELECT posting_month FROM mecc.sales_entry WHERE entry_id=$1`, seeded["SO-SEP-LAST"].ID) != month || h.closeInvariants()["totalAmount"] != inv["totalAmount"] {
		t.Fatal("allocation after the close moved the order or the totals")
	}
	observed(t, "boundary", map[string]any{"invariants": inv, "entriesDuringClose": len(during), "sepLastAllocAtClose": awaitingAtClose,
		"utcDateDesignMonth": naive, "periodState": h.str(`SELECT state FROM mecc.period WHERE month=$1`, month)})
}

// Through steps 1-3 with the CLI in separate processes: the calendar
// version and the decision digest chosen in step 1 are carried by the
// import units, the reconciliation, the close run and the close totals. A
// decision made under another calendar version stops the import and the
// close.
func TestPostgresThroughSlice1(t *testing.T) {
	h := newHarness(t)
	h.exec(`DELETE FROM mecc.calendar_version`)
	h.exec(`DELETE FROM mecc.payable`)
	rule, cal := "testdata/close-rule.json", "testdata/calendar-v1.json"
	h.run(0, nil, "calendar-load", "-file", cal)
	h.run(0, nil, "payable-load", "-file", h.writeJSON("payables.json", syntheticPayables()))
	decide := func(version int, at, out string) DecideResult {
		p := h.run(0, nil, "decide", "-month", month, "-calendar", "jp-business", "-version", strconv.Itoa(version), "-rule", rule, "-evaluated-at", at, "-out", h.dir+"/"+out)
		return decodeOut[DecideResult](t, p)
	}
	d1 := decide(1, "2026-09-28T09:00:00+09:00", "d1.json")
	h.run(0, nil, "calendar-revise", "-calendar", "jp-business", "-from", "1", "-holiday", tempHoliday+"=臨時休業", "-recorded-at", "2026-09-29T10:00:00+09:00")
	d2 := decide(2, "2026-09-29T12:00:00+09:00", "d2.json")
	if d2.Impact == nil || len(d2.Impact.JobShifts) != 1 || len(d2.Impact.DueShifts) != 2 {
		t.Fatalf("impact %+v", d2.Impact)
	}
	d1p, d2p := h.dir+"/d1.json", h.dir+"/d2.json"
	spec := "testdata/file-spec.json"
	rows := syntheticShipments(t)
	f1 := h.write("f1.dat", buildFile(t, rows, fileOpts{seq: 1}))
	refusedWith(t, h.run(4, nil, "import", "-decision", d1p, "-spec", spec, "-file", f1, "-received-at", receivedBeforeCutoff), RejectDecision)
	imp := decodeOut[ImportResult](t, h.run(0, nil, "import", "-decision", d2p, "-spec", spec, "-file", f1, "-received-at", receivedBeforeCutoff))
	refusedWith(t, h.run(4, nil, "import", "-decision", d2p, "-spec", spec, "-file", f1, "-received-at", receivedBeforeCutoff), RejectDuplicateFile)
	re := h.dir + "/f1.reemit"
	h.run(0, nil, "reemit", "-spec", spec, "-import-id", strconv.FormatInt(imp.ImportID, 10), "-out", re)
	if FileSHA256(mustRead(t, re)) != imp.FileSHA256 {
		t.Fatal("re-emitted file differs")
	}
	var msgs []Message
	for i := 0; i < 15; i++ {
		r := rows[i]
		m := Message{MessageID: fmt.Sprintf("m-%d", i), Event: ShipmentEvent{ShipmentID: r.ShipmentID, OrderNo: r.OrderNo,
			ShipDate: r.ShipDate[:4] + "-" + r.ShipDate[4:6] + "-" + r.ShipDate[6:], Qty: r.Qty, Amount: r.Amount, ItemName: r.ItemText}}
		msgs = append(msgs, m, m)
	}
	h.run(0, nil, "message", "-decision", d2p, "-spec", spec, "-file", h.writeJSON("msgs.json", msgs))
	rec := decodeOut[ShipmentReconciliation](t, h.run(0, nil, "reconcile-shipments", "-decision", d2p))
	if rec.Events != 30 || rec.Classes[ClassBoth] != 12 || rec.Classes[ClassFileHeld] != 3 || rec.Classes[ClassFileOnly] != 15 {
		t.Fatalf("reconciliation %+v", rec.Classes)
	}
	dec2, err := ReadDecision(d2p)
	if err != nil {
		t.Fatal(err)
	}
	h.seedEntries(dec2)
	asOf := func(at string, args ...string) []string { return append(args, "-as-of", at) }
	base := func(d, owner string) []string {
		return []string{"close", "-decision", d, "-owner", owner, "-ttl", "3s", "-chunk-size", "7"}
	}
	// The close job moved to 2 October with calendar v2.
	refusedWith(t, h.run(4, nil, asOf("2026-10-01T00:30:00+09:00", base(d2p, "srv-a")...)...), ErrNotYetScheduled.Error())
	// A calendar revision in the middle re-decides the month: the step-2
	// decision no longer matches, and the new decision does not match the
	// import units.
	h.run(0, nil, "calendar-revise", "-calendar", "jp-business", "-from", "2", "-holiday", "2026-11-24=臨時休業", "-recorded-at", "2026-10-01T09:00:00+09:00")
	d3 := decide(3, "2026-10-01T10:00:00+09:00", "d3.json")
	d3p := h.dir + "/d3.json"
	at := "2026-10-02T00:30:00+09:00"
	refusedWith(t, h.run(4, nil, asOf(at, base(d2p, "srv-a")...)...), "decision mismatch")
	refusedWith(t, h.run(4, nil, asOf(at, base(d3p, "srv-a")...)...), "was made under calendar v2")
	// The operator pins the month back to the step-2 decision: the same
	// calendar version and evaluation instant give the same digest, and the
	// re-activation is recorded with its impact.
	again := decide(2, "2026-09-29T12:00:00+09:00", "d2c.json")
	if again.Decision.Digest != d2.Decision.Digest || !again.Activated {
		t.Fatalf("re-activation %+v", again)
	}
	h.killed(CrashCloseChunkMid+":2", asOf(at, base(d2p, "srv-a")...)...)
	time.Sleep(3500 * time.Millisecond)
	cr := decodeOut[CloseResult](t, h.run(0, nil, asOf(at, base(d2p, "srv-b")...)...))
	refusedWith(t, h.run(4, nil, asOf(at, base(d2p, "srv-c")...)...), ErrAlreadyCompleted.Error())
	inv := h.closeInvariants()
	// Carried values.
	digest, version := d2.Decision.Digest, int64(2)
	checks := map[string]int64{
		"importUnitsOther":    h.int(`SELECT count(*) FROM mecc.import_unit WHERE decision_digest<>$1 OR calendar_version<>$2`, digest, version),
		"importUnits":         h.int(`SELECT count(*) FROM mecc.import_unit WHERE decision_digest=$1 AND calendar_version=$2`, digest, version),
		"reconciliationOther": h.int(`SELECT count(*) FROM mecc.shipment_reconciliation WHERE decision_digest<>$1 OR calendar_version<>$2`, digest, version),
		"closeRun":            h.int(`SELECT count(*) FROM mecc.close_run WHERE decision_digest=$1 AND calendar_version=$2 AND status='completed'`, digest, version),
		"closeTotal":          h.int(`SELECT count(*) FROM mecc.close_total WHERE decision_digest=$1 AND calendar_version=$2`, digest, version),
		"activeIsStep1":       h.int(`SELECT count(*) FROM mecc.close_month WHERE active_digest=$1`, digest),
	}
	if checks["importUnitsOther"] != 0 || checks["importUnits"] != 1 || checks["reconciliationOther"] != 0 || checks["closeRun"] != 1 || checks["closeTotal"] != 1 || checks["activeIsStep1"] != 1 {
		t.Fatalf("carried values %v", checks)
	}
	if cr.DecisionDigest != digest || cr.CalendarVersion != 2 || cr.CloseDate != "2026-09-30" || imp.DecisionDigest != digest || rec.DecisionDigest != digest {
		t.Fatalf("outputs carry %s/%s/%s", cr.DecisionDigest, imp.DecisionDigest, rec.DecisionDigest)
	}
	if cr.Shipments != 30 || h.int(`SELECT shipments FROM mecc.close_total WHERE month=$1`, month) != 30 {
		t.Fatalf("close shipments %d", cr.Shipments)
	}
	if _, err := Decide(h.ctx, h.conn, month, "jp-business", 3, testRule(t), mustTime(t, "2026-10-01T10:00:00+09:00")); !errors.Is(err, ErrMonthClosing) {
		t.Fatalf("decision changed after the close: %v", err)
	}
	observed(t, "through", map[string]any{"decisionDigest": digest, "calendarVersion": version, "d1": d1.Decision.Digest, "d3": d3.Decision.Digest,
		"checks": checks, "invariants": inv, "closeTakeover": cr.Takeover, "closeToken": cr.FencingToken, "importId": imp.ImportID, "reconciliation": rec.Classes})
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
