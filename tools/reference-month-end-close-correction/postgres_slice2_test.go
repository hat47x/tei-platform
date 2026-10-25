//go:build unix

package monthendclose

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const nextMonth = "2026-10"

func testPolicy(t *testing.T) LatePolicy {
	t.Helper()
	p, err := ReadLatePolicy("testdata/late-arrival-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// slice2Fixture prepares a month ready to close with items: decision v1,
// one imported file, the online entries and the items with their prices.
func (h *harness) slice2Fixture() (Decision, string) {
	h.t.Helper()
	d, dpath, _ := h.closeFixture()
	if err := LoadItems(h.ctx, h.conn, syntheticItems()); err != nil {
		h.t.Fatal(err)
	}
	return d, dpath
}

func (h *harness) apply(d Decision, pol LatePolicy, ev OrderEvent) EventResult {
	h.t.Helper()
	r, err := ApplyEvent(h.ctx, h.conn, d, pol, ev)
	if err != nil {
		h.t.Fatalf("%s: %v", ev.EventID, err)
	}
	return r
}

func (h *harness) closeMonth(d Decision) CloseResult {
	h.t.Helper()
	r, err := RunClose(h.ctx, h.conn, d, CloseOptions{OwnerID: "srv-a", TTL: 10 * time.Second, ChunkSize: 10, AsOf: mustTime(h.t, "2026-10-01T00:30:00+09:00")})
	if err != nil {
		h.t.Fatal(err)
	}
	return r
}

// closedSnapshot digests everything the close of the month fixed.
func (h *harness) closedSnapshot() string {
	h.t.Helper()
	return h.str(`SELECT md5(
		coalesce((SELECT string_agg(concat_ws('|',slip_id,kind,order_no,qty,unit_price,amount,posting_month,inserted_at), ',' ORDER BY slip_id) FROM mecc.slip WHERE posting_month=$1),'') ||
		(SELECT concat_ws('|',entries,amount,awaiting,shipments,fencing_token) FROM mecc.close_total WHERE month=$1) ||
		(SELECT concat_ws('|',slips,base_amount,tax_total,report_sha256,fencing_token) FROM mecc.close_report WHERE month=$1) ||
		(SELECT concat_ws('|',status,cut_slips,cut_slip_amount,fencing_token) FROM mecc.close_run WHERE month=$1))`, month)
}

// orderOutcome is the final state of an order and what its slips net to.
type orderOutcome struct {
	State     OrderState
	NetQty    int64
	NetAmount int64
	Live      int64 // live slips: 1 for an active order, 0 for a cancelled one
}

func (h *harness) outcomes() map[string]orderOutcome {
	h.t.Helper()
	out := map[string]orderOutcome{}
	rows, err := h.conn.Query(h.ctx, `SELECT o.order_no, o.status, coalesce(o.customer,''), coalesce(o.item_code,''), coalesce(o.ship_date::text,''), coalesce(o.qty,0), o.version,
		coalesce((SELECT sum(qty) FROM mecc.slip s WHERE s.order_no=o.order_no),0),
		coalesce((SELECT (sum(amount)*100)::bigint FROM mecc.slip s WHERE s.order_no=o.order_no),0),
		(SELECT count(*) FROM mecc.slip s WHERE s.order_no=o.order_no AND s.kind IN ('original','black') AND NOT EXISTS (SELECT 1 FROM mecc.slip r WHERE r.reverses=s.slip_id))
		FROM mecc.order_state o`)
	if err != nil {
		h.t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var o orderOutcome
		if err := rows.Scan(&o.State.OrderNo, &o.State.Status, &o.State.Customer, &o.State.ItemCode, &o.State.ShipDate, &o.State.Qty, &o.State.Version, &o.NetQty, &o.NetAmount, &o.Live); err != nil {
			h.t.Fatal(err)
		}
		out[o.State.OrderNo] = o
	}
	return out
}

// P-TIME-04 (with P-INT-02): run A applies every event in order before the
// close. Run B shuffles the events, delivers changes before their
// registrations, redelivers some, sends the same event by another route,
// sends a conflicting one, and delivers part of them after the close. The
// final states and the net slips equal run A; nothing the close fixed
// changes; late events follow the declared policy (adjust in the open
// month; a cancellation is held until a person releases it).
func TestPostgresPTIME04LateAndReorderedEvents(t *testing.T) {
	h := newHarness(t)
	pol := testPolicy(t)
	events := syntheticOrderEvents(100)

	// Run A: in order, on time.
	dA, _ := h.slice2Fixture()
	for _, ev := range events {
		h.apply(dA, pol, ev)
	}
	h.closeMonth(dA)
	inOrder := h.outcomes()
	inOrderSept := h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month=$1`, month)
	if n := h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month<>$1`, month); n != 0 {
		t.Fatalf("run A posted %d slips outside September", n)
	}

	// Run B.
	h.reset()
	dB, _ := h.slice2Fixture()
	orderIndex := func(no string) int { n, _ := strconv.Atoi(strings.TrimPrefix(no, "SO-L")); return n }
	var early, late []OrderEvent
	for _, ev := range events {
		i := orderIndex(ev.OrderNo)
		if (i%5 == 0 && ev.Version > 1) || i%10 == 0 {
			late = append(late, ev)
		} else {
			early = append(early, ev)
		}
	}
	rng := rand.New(rand.NewSource(20261020))
	rng.Shuffle(len(early), func(i, j int) { early[i], early[j] = early[j], early[i] })
	// Changes ahead of their registrations, for certain: every twelfth
	// order's registration goes after its last change.
	pos := map[string][]int{}
	for k, ev := range early {
		pos[ev.OrderNo] = append(pos[ev.OrderNo], k)
	}
	swapped := 0
	for no, ks := range pos {
		if orderIndex(no)%12 != 0 || len(ks) < 2 {
			continue
		}
		reg, last := -1, -1
		for _, k := range ks {
			if early[k].Version == 1 {
				reg = k
			}
			if last < 0 || k > last {
				last = k
			}
		}
		if reg >= 0 && reg != last {
			early[reg], early[last] = early[last], early[reg]
			swapped++
		}
	}
	if swapped == 0 {
		t.Fatal("no registration was moved behind its change")
	}
	for _, i := range []int{3, 17, 41} {
		early = append(early, early[i]) // redelivery of the same event id
	}
	viaFile := early[5]
	viaFile.EventID, viaFile.Source = viaFile.EventID+"-file", SourceFile
	early = append(early, viaFile)
	var conflict OrderEvent
	for _, ev := range early {
		if ev.Kind == EventChanged {
			conflict = ev
			break
		}
	}
	conflict.EventID, conflict.Qty = conflict.EventID+"-conflict", conflict.Qty+100
	early = append(early, conflict)
	outcomesB := map[string]int{}
	for _, ev := range early {
		outcomesB[h.apply(dB, pol, ev).Outcome]++
	}
	if outcomesB[EventPendingBase] == 0 || outcomesB[EventRedelivered] != 3 || outcomesB[EventConfirmed] != 1 || outcomesB[EventConflict] != 1 {
		t.Fatalf("early outcomes %v", outcomesB)
	}
	h.closeMonth(dB)
	closed := h.closedSnapshot()
	rng.Shuffle(len(late), func(i, j int) { late[i], late[j] = late[j], late[i] })
	late = append(late, late[0])
	lateOutcomes := map[string]int{}
	var held []string
	for k, ev := range late {
		ev.ReceivedAt = fmt.Sprintf("2026-10-02T18:%02d:00+09:00", k%60)
		r := h.apply(dB, pol, ev)
		lateOutcomes[r.Outcome]++
		if r.Outcome == EventHeldLate {
			held = append(held, ev.EventID)
		}
		if r.Outcome != EventRedelivered && (r.PeriodState != "closed" || (r.Outcome != EventHeldLate && r.Policy != PolicyAdjust)) {
			t.Fatalf("late %s: %+v", ev.EventID, r)
		}
	}
	if h.closedSnapshot() != closed {
		t.Fatal("late events changed what the close fixed")
	}
	if len(held) != 4 || lateOutcomes[EventRedelivered] != 1 {
		t.Fatalf("late outcomes %v held %v", lateOutcomes, held)
	}
	// Before the release, every order but the held cancellations equals run A.
	before := h.outcomes()
	heldOrders := map[string]bool{}
	for _, id := range held {
		heldOrders[strings.Split(id, "-v")[0][3:]] = true
	}
	for no, a := range inOrder {
		b := before[no]
		if heldOrders[no] {
			if b.State.Status != "active" || b.Live != 1 {
				t.Fatalf("held cancellation applied to %s: %+v", no, b)
			}
			continue
		}
		if a != b {
			t.Fatalf("%s: shuffled %+v, in order %+v", no, b, a)
		}
	}
	for _, id := range held {
		if _, err := ReleaseHeld(h.ctx, h.conn, dB, pol, id, mustTime(t, "2026-10-03T09:00:00+09:00")); err != nil {
			t.Fatal(err)
		}
	}
	final := h.outcomes()
	if len(final) != len(inOrder) {
		t.Fatalf("%d orders, in order %d", len(final), len(inOrder))
	}
	for no, a := range inOrder {
		if final[no] != a {
			t.Fatalf("%s after release: %+v, in order %+v", no, final[no], a)
		}
	}
	if h.closedSnapshot() != closed {
		t.Fatal("the release changed what the close fixed")
	}
	// Every slip after the close went to October, carries the close it
	// corrects, and says why.
	token := h.int(`SELECT fencing_token FROM mecc.close_run WHERE month=$1`, month)
	bad := h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month=$1 AND (business_month<>$2 OR posting_reason NOT LIKE 'late:%' OR closed_month<>$2 OR close_token<>$3 OR close_decision_digest<>$4)`,
		nextMonth, month, token, dB.Digest)
	lateSlips := h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month=$1`, nextMonth)
	if bad != 0 || lateSlips == 0 {
		t.Fatalf("late slips %d, %d without the declared handling", lateSlips, bad)
	}
	keys := h.int(`SELECT count(*) FROM mecc.order_event WHERE received_at IS NULL OR business_date IS NULL OR version < 1`)
	reasons := map[string]int64{}
	rows, _ := h.conn.Query(h.ctx, `SELECT posting_reason, count(*) FROM mecc.slip WHERE posting_month=$1 GROUP BY 1`, nextMonth)
	for rows.Next() {
		var r string
		var n int64
		rows.Scan(&r, &n)
		reasons[r] = n
	}
	rows.Close()
	observed(t, "late", map[string]any{"events": len(events), "early": len(early), "late": len(late), "earlyOutcomes": outcomesB, "lateOutcomes": lateOutcomes,
		"held": held, "inOrderSeptSlips": inOrderSept, "shuffledSeptSlips": h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month=$1`, month),
		"lateSlips": lateSlips, "lateReasons": reasons, "ordersCompared": len(inOrder), "eventsWithoutKeys": keys})
}

func pgCode(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// P-TIME-03 and P-TIME-02: prices on two axes with a retroactive revision,
// a reservation and a withdrawn revision; overlapping and backwards
// registrations are refused. Slips of the closed month cannot be updated;
// a retroactive revision and a late change after the close become red and
// black slips in the open month, linked to the originals.
func TestPostgresPTIME02PTIME03RedBlackAndBitemporalPrice(t *testing.T) {
	h := newHarness(t)
	pol := testPolicy(t)
	d, _ := h.slice2Fixture()
	reg := func(no, item, ship string, qty int64, recv string) EventResult {
		return h.apply(d, pol, OrderEvent{EventID: "ev-" + no + "-v1", Kind: EventRegistered, OrderNo: no, Version: 1, BusinessDate: ship, Customer: "C01", ItemCode: item, Qty: qty, ReceivedAt: recv})
	}
	reg("SO-A1", "IT-06", "2026-09-03", 3, "2026-09-03T10:00:00+09:00")
	reg("SO-B1", "IT-02", "2026-09-05", 4, "2026-09-05T10:00:00+09:00")
	reg("SO-C1", "IT-09", "2026-09-08", 1, "2026-09-08T10:00:00+09:00")
	reg("SO-C2", "IT-09", "2026-09-20", 3, "2026-09-20T10:00:00+09:00")
	revise := func(item, from, price, at string) ReviseResult {
		t.Helper()
		r, err := RevisePrice(h.ctx, h.conn, &d, pol, item, from, price, mustTime(t, at))
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	r1 := revise("IT-06", "2026-09-01", "104.11", "2026-09-10T09:00:00+09:00")
	if r1.Change.Op != PriceRetroactive || len(r1.Slips) != 2 || r1.Slips[0].Kind != SlipRed || r1.Slips[0].Amount != "-304.11" || r1.Slips[1].Amount != "312.33" || r1.Slips[1].PostingMonth != month {
		t.Fatalf("retroactive revision %+v", r1)
	}
	revise("IT-02", "2026-09-01", "34.56", "2026-09-10T09:00:00+09:00")
	r3 := revise("IT-06", "2026-11-01", "110.00", "2026-09-12T09:00:00+09:00")
	if r3.Change.Op != PriceReserve || len(r3.Slips) != 0 {
		t.Fatalf("reservation %+v", r3)
	}
	a2 := reg("SO-A2", "IT-06", "2026-09-12", 2, "2026-09-12T10:00:00+09:00")
	if a2.Slips[0].UnitPrice != "104.11" {
		t.Fatalf("SO-A2 priced %s", a2.Slips[0].UnitPrice)
	}
	c4, err := CancelRevision(h.ctx, h.conn, &d, pol, "IT-02", "2026-09-01", mustTime(t, "2026-09-15T09:00:00+09:00"))
	if err != nil || c4.Change.Op != PriceCancel || len(c4.Slips) != 2 || c4.Slips[1].UnitPrice != "33.33" || c4.Slips[1].Kind != SlipBlack {
		t.Fatalf("cancel %+v %v", c4, err)
	}
	// The two axes.
	at := func(item, on, known string) string {
		t.Helper()
		p, err := PriceAt(h.ctx, h.conn, item, on, mustTime(t, known))
		if err != nil {
			t.Fatalf("%s on %s known %s: %v", item, on, known, err)
		}
		return p.UnitPrice
	}
	asked := map[string]string{
		"IT-06 2026-09-05 @09-07": at("IT-06", "2026-09-05", "2026-09-07T12:00:00+09:00"),
		"IT-06 2026-09-05 @09-12": at("IT-06", "2026-09-05", "2026-09-12T12:00:00+09:00"),
		"IT-06 2026-08-31 @09-12": at("IT-06", "2026-08-31", "2026-09-12T12:00:00+09:00"),
		"IT-06 2026-11-05 @09-11": at("IT-06", "2026-11-05", "2026-09-11T12:00:00+09:00"),
		"IT-06 2026-11-05 @09-12": at("IT-06", "2026-11-05", "2026-09-12T12:00:00+09:00"),
		"IT-02 2026-09-05 @09-07": at("IT-02", "2026-09-05", "2026-09-07T12:00:00+09:00"),
		"IT-02 2026-09-05 @09-12": at("IT-02", "2026-09-05", "2026-09-12T12:00:00+09:00"),
		"IT-02 2026-09-05 @09-20": at("IT-02", "2026-09-05", "2026-09-20T12:00:00+09:00"),
	}
	want := map[string]string{"IT-06 2026-09-05 @09-07": "101.37", "IT-06 2026-09-05 @09-12": "104.11", "IT-06 2026-08-31 @09-12": "101.37",
		"IT-06 2026-11-05 @09-11": "104.11", "IT-06 2026-11-05 @09-12": "110.00",
		"IT-02 2026-09-05 @09-07": "33.33", "IT-02 2026-09-05 @09-12": "34.56", "IT-02 2026-09-05 @09-20": "33.33"}
	for k, v := range want {
		if asked[k] != v {
			t.Fatalf("%s: %s, want %s (all %v)", k, asked[k], v, asked)
		}
	}
	if _, err := PriceAt(h.ctx, h.conn, "IT-06", "2025-12-31", mustTime(t, "2026-09-12T12:00:00+09:00")); !errors.Is(err, ErrNoPrice) {
		t.Fatalf("price before the first validity: %v", err)
	}
	// Refusals.
	refusals := map[string]string{}
	if err := RegisterPriceRaw(h.ctx, h.conn, "IT-06", "2026-10-01", "infinity", "99.99", mustTime(t, "2026-09-20T09:00:00+09:00")); !errors.Is(err, ErrPriceOverlap) {
		t.Fatalf("overlapping registration: %v", err)
	}
	refusals["raw-overlap"] = "ErrPriceOverlap"
	_, err = h.conn.Exec(h.ctx, `INSERT INTO mecc.item_price(item_code,unit_price,valid_from,recorded_from,op,change_id) VALUES ('IT-06',99.99,'2026-10-01','2026-09-20T09:00:00+09:00','sql',0)`)
	if refusals["sql-overlap"] = pgCode(err); refusals["sql-overlap"] != PriceOverlapSQLState {
		t.Fatalf("SQL overlap: %v", err)
	}
	if _, err := RevisePrice(h.ctx, h.conn, &d, pol, "IT-06", "2026-09-01", "105.00", mustTime(t, "2026-09-11T09:00:00+09:00")); !errors.Is(err, ErrPriceRecordedBackwards) {
		t.Fatalf("backwards revision: %v", err)
	}
	refusals["backwards"] = "ErrPriceRecordedBackwards"
	_, err = h.conn.Exec(h.ctx, `UPDATE mecc.item_price SET unit_price=1 WHERE item_code='IT-06'`)
	if refusals["sql-update"] = pgCode(err); refusals["sql-update"] != PriceImmutableSQLState {
		t.Fatalf("price update: %v", err)
	}
	if _, err := CancelRevision(h.ctx, h.conn, &d, pol, "IT-02", "2026-09-01", mustTime(t, "2026-09-16T09:00:00+09:00")); !errors.Is(err, ErrNothingToCancel) {
		t.Fatalf("second cancel: %v", err)
	}
	priceRefused := h.int(`SELECT count(*) FROM mecc.price_change WHERE outcome LIKE 'refused%'`)
	if priceRefused != 3 {
		t.Fatalf("%d refused price changes recorded", priceRefused)
	}

	cr := h.closeMonth(d)
	closed := h.closedSnapshot()
	// P-TIME-02: the closed month's slips cannot be updated, deleted or added to.
	septSlip := h.int(`SELECT min(slip_id) FROM mecc.slip WHERE posting_month=$1`, month)
	if err := AmendSlipInPlace(h.ctx, h.conn, septSlip, 99); !errors.Is(err, ErrClosedPeriod) {
		t.Fatalf("in-place update of a closed slip: %v", err)
	}
	_, err = h.conn.Exec(h.ctx, `DELETE FROM mecc.slip WHERE slip_id=$1`, septSlip)
	delCode := pgCode(err)
	_, err = h.conn.Exec(h.ctx, `INSERT INTO mecc.slip(kind,order_no,customer,item_code,tax_rate,ship_date,business_month,posting_month,posting_reason,qty,unit_price,amount,price_id,price_valid_on,price_known_at,known_at,cause,decision_digest,calendar_version)
		SELECT 'original','SO-X','C01',item_code,tax_rate,ship_date,business_month,posting_month,'x',1,unit_price,unit_price,price_id,price_valid_on,price_known_at,known_at,'sql',decision_digest,calendar_version FROM mecc.slip WHERE slip_id=$1`, septSlip)
	insCode := pgCode(err)
	if delCode != PeriodSQLState || insCode != PeriodSQLState {
		t.Fatalf("delete %q insert %q", delCode, insCode)
	}
	// After the close: a retroactive revision and a late change.
	rc := revise("IT-09", "2026-09-01", "125.00", "2026-10-02T18:00:00+09:00")
	late := h.apply(d, pol, OrderEvent{EventID: "ev-SO-A1-v2", Kind: EventChanged, OrderNo: "SO-A1", Version: 2, BusinessDate: "2026-09-03", Qty: 5, ReceivedAt: "2026-10-02T18:30:00+09:00"})
	if h.closedSnapshot() != closed {
		t.Fatal("corrections changed the closed month")
	}
	corr := append(append([]SlipRef(nil), rc.Slips...), late.Slips...)
	if len(rc.Slips) != 4 || len(late.Slips) != 2 {
		t.Fatalf("corrections %+v / %+v", rc.Slips, late.Slips)
	}
	for _, s := range corr {
		if s.PostingMonth != nextMonth || s.BusinessMonth != month || s.PostingReason != "late:"+PolicyAdjust || s.CloseToken != int64(cr.FencingToken) {
			t.Fatalf("correction %+v", s)
		}
	}
	// Each red is the exact negation of the slip it reverses; each black
	// corrects that same slip and is priced on the ship date as known at the
	// correction.
	pairs := h.int(`SELECT count(*) FROM mecc.slip r JOIN mecc.slip o ON o.slip_id=r.reverses JOIN mecc.slip b ON b.corrects=o.slip_id
		WHERE r.posting_month=$1 AND r.amount=-o.amount AND r.qty=-o.qty AND r.unit_price=o.unit_price AND b.price_valid_on=b.ship_date AND b.price_known_at=b.known_at`, nextMonth)
	if pairs != 3 {
		t.Fatalf("%d red-black pairs in October", pairs)
	}
	black := rc.Slips[1]
	if black.UnitPrice != "125.00" || black.PriceValidOn != "2026-09-08" || black.PriceKnownAt != "2026-10-02T09:00:00Z" {
		t.Fatalf("black %+v", black)
	}
	if late.Slips[0].Amount != "-312.33" || late.Slips[1].Amount != "520.55" {
		t.Fatalf("late change %+v", late.Slips)
	}
	// In the open month a slip is immutable too.
	if err := AmendSlipInPlace(h.ctx, h.conn, late.Slips[1].SlipID, 1); !errors.Is(err, ErrImmutableSlip) {
		t.Fatalf("in-place update of an open slip: %v", err)
	}
	observed(t, "bitemporal", map[string]any{"prices": asked, "refusals": refusals, "priceRefusalsRecorded": priceRefused,
		"closedSlipUpdate": "ErrClosedPeriod", "closedSlipDelete": delCode, "closedSlipInsert": insCode, "redBlackPairs": pairs, "corrections": corr})
}

// P-CALC-01: 100 lines at 8% and 10% with prices that leave fractions, and
// red and black slips after the close. PostgreSQL, Go and SQLite give the
// same invoice taxes and allocations. The close's report is reissued byte
// for byte as of the close; repricing with today's knowledge would not
// reproduce it; the current balance is the corrected value.
func TestPostgresPCALC01TaxPathsAndReissue(t *testing.T) {
	h := newHarness(t)
	pol := testPolicy(t)
	d, _ := h.slice2Fixture()
	for _, ev := range syntheticOrderEvents(100) {
		h.apply(d, pol, ev)
	}
	cr := h.closeMonth(d)
	if cr.Slips < 100 || cr.ReportSHA256 == "" {
		t.Fatalf("close %+v", cr)
	}
	// Late changes: orders 3, 13, ... (IT-02 at 8%) to 7 units, orders 8,
	// 18, ... (IT-07 at 10%) down to 1 unit, so that an October invoice is
	// negative.
	for i := 3; i <= 98; i += 5 {
		no := fmt.Sprintf("SO-L%03d", i)
		qty := int64(7)
		if i%10 == 8 {
			qty = 1
		}
		v := h.int(`SELECT version FROM mecc.order_state WHERE order_no=$1`, no)
		ship := h.str(`SELECT ship_date::text FROM mecc.order_state WHERE order_no=$1`, no)
		h.apply(d, pol, OrderEvent{EventID: fmt.Sprintf("late-%s", no), Kind: EventChanged, OrderNo: no, Version: int(v) + 1, BusinessDate: ship, Qty: qty, ReceivedAt: "2026-10-02T18:00:00+09:00"})
	}
	if _, err := RevisePrice(h.ctx, h.conn, &d, pol, "IT-03", "2026-09-01", "1.03", mustTime(t, "2026-10-03T09:00:00+09:00")); err != nil {
		t.Fatal(err)
	}
	rule, err := ReadTaxRule("testdata/tax-rule.json")
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]TaxCheck{}
	for _, ms := range [][]string{{month}, {nextMonth}, {month, nextMonth}} {
		c, err := CheckTax(h.ctx, h.conn, rule, ms)
		if err != nil {
			t.Fatal(err)
		}
		checks[strings.Join(ms, "+")] = c
	}
	sept, oct := checks[month], checks[nextMonth]
	negative := 0
	for _, v := range oct.Invoices {
		if v.BaseSen < 0 {
			negative++
		}
	}
	// The fixture must exercise the declared rounding: per-line rounding
	// and rounding toward negative infinity would give other taxes.
	if sept.Lines < 100 || !sept.Agree || !oct.Agree || sept.PerLineRoundingDiffers == 0 || negative == 0 || oct.FloorDiffers == 0 || sept.SQLiteVersion == "" {
		t.Fatalf("tax checks %+v", checks)
	}
	if sept.TaxTotal != cr.TaxTotal {
		t.Fatalf("September tax %d, close %d", sept.TaxTotal, cr.TaxTotal)
	}
	fractional := h.int(`SELECT count(*) FROM mecc.slip r JOIN mecc.slip b ON b.corrects=r.reverses WHERE r.kind='red' AND r.posting_month=$1 AND (r.amount+b.amount) <> trunc(r.amount+b.amount)`, nextMonth)
	if fractional == 0 {
		t.Fatal("no red-black difference has a fraction of a yen")
	}
	// Reissue as of the close.
	re, err := Reissue(h.ctx, h.conn, d, BasisClose)
	if err != nil {
		t.Fatal(err)
	}
	if !re.ByteEqual || re.ReportSHA256 != cr.ReportSHA256 || re.TaxTotal != cr.TaxTotal {
		t.Fatalf("reissue %+v, close %s", re, cr.ReportSHA256)
	}
	// Recomputing every line from the price history at the line's own
	// record time reproduces the report; repricing with today's knowledge
	// does not.
	tx, err := h.conn.Begin(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	cutAt := mustTime(t, re.CutAt)
	lines, err := loadReportLines(h.ctx, tx, `posting_month=$1 AND inserted_at <= $2`, month, cutAt)
	tx.Rollback(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	reprice := func(known func(ReportLine) time.Time) []ReportLine {
		out := make([]ReportLine, len(lines))
		for i, l := range lines {
			if l.Kind == SlipRed {
				o := out[indexOf(lines, l.Reverses)]
				l.UnitPriceSen, l.AmountSen = o.UnitPriceSen, l.Qty*o.UnitPriceSen
			} else {
				p, err := PriceAt(h.ctx, h.conn, l.ItemCode, l.ShipDate, known(l))
				if err != nil {
					t.Fatal(err)
				}
				l.UnitPriceSen, l.AmountSen = p.UnitPriceSen, l.Qty*p.UnitPriceSen
			}
			out[i] = l
		}
		return out
	}
	header := ReportHeader{Month: month, Basis: BasisClose, DecisionDigest: re.DecisionDigest, CalendarVersion: re.CalendarVersion, CloseDate: d.CloseDate, CloseToken: re.CloseToken, CutAt: re.CutAt, TaxRule: DefaultTaxRule.ID}
	render := func(ls []ReportLine) []byte {
		var tl []TaxLine
		for _, l := range ls {
			tl = append(tl, TaxLine{l.SlipID, l.Customer, l.PostingMonth, l.Rate, l.AmountSen})
		}
		return RenderReport(header, ls, GoTax(tl).Invoices)
	}
	bitemporal := render(reprice(func(l ReportLine) time.Time { return mustTime(t, l.PriceKnownAt) }))
	today := render(reprice(func(ReportLine) time.Time { return mustTime(t, "2026-10-03T10:00:00+09:00") }))
	if !bytes.Equal(bitemporal, re.Report) || bytes.Equal(today, re.Report) {
		t.Fatalf("bitemporal recomputation equal %v, today's repricing equal %v", bytes.Equal(bitemporal, re.Report), bytes.Equal(today, re.Report))
	}
	// The current balance is the corrected value: each live order line at
	// today's price, and the difference from the close is the corrections.
	cur, err := Reissue(h.ctx, h.conn, d, BasisCurrent)
	if err != nil {
		t.Fatal(err)
	}
	var want int64
	for _, o := range h.outcomes() {
		if o.State.Status == "active" {
			p, err := PriceAt(h.ctx, h.conn, o.State.ItemCode, o.State.ShipDate, mustTime(t, "2026-10-03T10:00:00+09:00"))
			if err != nil {
				t.Fatal(err)
			}
			want += o.State.Qty * p.UnitPriceSen
		}
	}
	if cur.Base != FormatSen(want) || len(cur.Corrections) == 0 {
		t.Fatalf("current balance %s, want %s (%d corrections)", cur.Base, FormatSen(want), len(cur.Corrections))
	}
	observed(t, "tax", map[string]any{"september": sept, "october": oct, "both": checks[month+"+"+nextMonth].Agree, "closeTax": cr.TaxTotal, "closeSlips": cr.Slips,
		"fractionalRedBlack": fractional, "negativeOctoberInvoices": negative, "reportSha256": re.ReportSHA256, "reportBytes": len(re.Report), "byteEqual": re.ByteEqual,
		"bitemporalRecomputationEqual": bytes.Equal(bitemporal, re.Report), "todayRepricingEqual": bytes.Equal(today, re.Report),
		"closeBase": cur.CloseBase, "currentBase": cur.Base, "difference": cur.Difference, "corrections": len(cur.Corrections)})
}

func indexOf(lines []ReportLine, id int64) int {
	for i, l := range lines {
		if l.SlipID == id {
			return i
		}
	}
	return -1
}

// Through steps 1-6 with the CLI in separate processes: the calendar
// version and the decision digest of step 1 and the close run of step 3
// are carried by the late events, the red and black slips, and the
// reissued report. A decision of another calendar version stops step 4.
func TestPostgresThroughSlice2(t *testing.T) {
	h := newHarness(t)
	h.exec(`DELETE FROM mecc.calendar_version`)
	h.exec(`DELETE FROM mecc.payable`)
	rule, cal, spec, policy, taxRule := "testdata/close-rule.json", "testdata/calendar-v1.json", "testdata/file-spec.json", "testdata/late-arrival-policy.json", "testdata/tax-rule.json"
	h.run(0, nil, "calendar-load", "-file", cal)
	h.run(0, nil, "payable-load", "-file", h.writeJSON("payables.json", syntheticPayables()))
	d1p, d2p := h.dir+"/d1.json", h.dir+"/d2.json"
	h.run(0, nil, "decide", "-month", month, "-calendar", "jp-business", "-version", "1", "-rule", rule, "-evaluated-at", "2026-09-28T09:00:00+09:00", "-out", d1p)
	h.run(0, nil, "calendar-revise", "-calendar", "jp-business", "-from", "1", "-holiday", tempHoliday+"=臨時休業", "-recorded-at", "2026-09-29T10:00:00+09:00")
	d2 := decodeOut[DecideResult](t, h.run(0, nil, "decide", "-month", month, "-calendar", "jp-business", "-version", "2", "-rule", rule, "-evaluated-at", "2026-09-29T12:00:00+09:00", "-out", d2p)).Decision
	f1 := h.write("f1.dat", buildFile(t, syntheticShipments(t), fileOpts{seq: 1}))
	h.run(0, nil, "import", "-decision", d2p, "-spec", spec, "-file", f1, "-received-at", receivedBeforeCutoff)
	h.run(0, nil, "item-load", "-file", h.writeJSON("items.json", syntheticItems()))
	all := syntheticOrderEvents(40)
	var onTime, late []OrderEvent
	for _, ev := range all {
		if (ev.Version > 1 && ev.OrderNo >= "SO-L020") || ev.OrderNo >= "SO-L039" {
			late = append(late, ev)
		} else {
			onTime = append(onTime, ev)
		}
	}
	h.run(0, nil, "events", "-decision", d2p, "-policy", policy, "-file", h.writeJSON("ev1.json", onTime))
	cr := decodeOut[CloseResult](t, h.run(0, nil, "close", "-decision", d2p, "-owner", "srv-a", "-ttl", "5s", "-chunk-size", "10", "-as-of", "2026-10-02T00:30:00+09:00"))
	// Step 4: a decision of calendar v1 stops; the step-1 decision applies.
	for i := range late {
		late[i].ReceivedAt = "2026-10-02T18:00:00+09:00"
	}
	lateFile := h.writeJSON("ev2.json", late)
	refusedWith(t, h.run(4, nil, "events", "-decision", d1p, "-policy", policy, "-file", lateFile), "decision mismatch")
	stepFour := decodeOut[[]EventResult](t, h.run(0, nil, "events", "-decision", d2p, "-policy", policy, "-file", lateFile))
	// Step 5.
	septSlip := strconv.FormatInt(h.int(`SELECT min(slip_id) FROM mecc.slip WHERE posting_month=$1`, month), 10)
	refusedWith(t, h.run(4, nil, "slip-amend", "-slip", septSlip, "-qty", "9"), ErrClosedPeriod.Error())
	rev := decodeOut[ReviseResult](t, h.run(0, nil, "price-revise", "-item", "IT-09", "-valid-from", "2026-09-01", "-price", "125.00", "-recorded-at", "2026-10-02T19:00:00+09:00", "-decision", d2p, "-policy", policy))
	pa := decodeOut[Price](t, h.run(0, nil, "price-at", "-item", "IT-09", "-on", "2026-09-05", "-known-at", "2026-10-01T00:00:00+09:00"))
	if pa.UnitPrice != "123.45" || len(rev.Slips) == 0 {
		t.Fatalf("price at close knowledge %s, repriced %d", pa.UnitPrice, len(rev.Slips))
	}
	// Step 6.
	tax := decodeOut[TaxCheck](t, h.run(0, nil, "tax", "-tax-rule", taxRule, "-months", month+","+nextMonth))
	repClose := decodeOut[ReissueResult](t, h.run(0, nil, "report", "-decision", d2p, "-basis", BasisClose, "-out", h.dir+"/close.txt"))
	repCur := decodeOut[ReissueResult](t, h.run(0, nil, "report", "-decision", d2p, "-basis", BasisCurrent, "-out", h.dir+"/current.txt"))
	refusedWith(t, h.run(4, nil, "report", "-decision", d1p, "-basis", BasisClose, "-out", h.dir+"/x.txt"), "decision mismatch")
	closeBytes := mustRead(t, h.dir+"/close.txt")
	stored := h.str(`SELECT report_sha256 FROM mecc.close_report WHERE month=$1`, month)
	if !tax.Agree || !repClose.ByteEqual || ReportSHA256(closeBytes) != stored || stored != cr.ReportSHA256 || repCur.Difference == "0.00" {
		t.Fatalf("tax %v reissue %+v stored %s close %s", tax.Agree, repClose, stored, cr.ReportSHA256)
	}
	token := int64(cr.FencingToken)
	for _, want := range []string{"decision," + d2.Digest + "\n", "calendarVersion,2\n", "closeDate,2026-09-30\n", fmt.Sprintf("closeToken,%d\n", token)} {
		if !bytes.Contains(closeBytes, []byte(want)) || !bytes.Contains(mustRead(t, h.dir+"/current.txt"), []byte(want)) {
			t.Fatalf("reports do not carry %q", want)
		}
	}
	checks := map[string]int64{
		"slipsOtherDecision":   h.int(`SELECT count(*) FROM mecc.slip WHERE decision_digest<>$1 OR calendar_version<>2`, d2.Digest),
		"eventsOtherDecision":  h.int(`SELECT count(*) FROM mecc.order_event WHERE decision_digest<>$1 OR calendar_version<>2`, d2.Digest),
		"lateSlips":            h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month=$1`, nextMonth),
		"lateSlipsOtherClose":  h.int(`SELECT count(*) FROM mecc.slip WHERE posting_month=$1 AND (closed_month IS DISTINCT FROM $2 OR close_token IS DISTINCT FROM $3 OR close_decision_digest IS DISTINCT FROM $4)`, nextMonth, month, token, d2.Digest),
		"redWithoutOriginal":   h.int(`SELECT count(*) FROM mecc.slip r WHERE r.kind='red' AND NOT EXISTS (SELECT 1 FROM mecc.slip o WHERE o.slip_id=r.reverses AND o.order_no=r.order_no)`),
		"closeReportOther":     h.int(`SELECT count(*) FROM mecc.close_report r JOIN mecc.close_run c USING (month) WHERE r.decision_digest<>c.decision_digest OR r.calendar_version<>c.calendar_version OR r.fencing_token<>c.fencing_token`),
		"closeRunCarriesStep1": h.int(`SELECT count(*) FROM mecc.close_run WHERE decision_digest=$1 AND calendar_version=2 AND status='completed'`, d2.Digest),
	}
	if checks["slipsOtherDecision"] != 0 || checks["eventsOtherDecision"] != 0 || checks["lateSlips"] == 0 || checks["lateSlipsOtherClose"] != 0 ||
		checks["redWithoutOriginal"] != 0 || checks["closeReportOther"] != 0 || checks["closeRunCarriesStep1"] != 1 {
		t.Fatalf("carried values %v", checks)
	}
	if repClose.DecisionDigest != d2.Digest || repClose.CalendarVersion != 2 || repClose.CloseToken != token || repCur.CloseToken != token {
		t.Fatalf("reissue carries %+v", repClose)
	}
	observed(t, "through2", map[string]any{"decisionDigest": d2.Digest, "calendarVersion": 2, "closeToken": token, "closeSlips": cr.Slips, "lateEvents": len(stepFour),
		"repriced": len(rev.Slips), "taxAgree": tax.Agree, "taxTotal": tax.TaxTotal, "sqliteVersion": tax.SQLiteVersion, "reportSha256": stored,
		"closeBase": repCur.CloseBase, "currentBase": repCur.Base, "difference": repCur.Difference, "checks": checks})
}
