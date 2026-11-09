package monthendclose

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Event kinds of an order line. The ordering key is (OrderNo, Version):
// the registration is version 1, every later change or cancellation has a
// higher version. Versions decide; arrival order does not.
const (
	EventRegistered = "shipment-registered"
	EventChanged    = "shipment-changed"
	EventCancelled  = "shipment-cancelled"
)

// Event outcomes.
const (
	EventApplied     = "applied"
	EventPendingBase = "pending-base"  // accepted; the registration has not arrived
	EventStale       = "stale-version" // accepted; a higher version is already known
	EventRedelivered = "redelivered"
	EventConfirmed   = "confirmed-same-event"
	EventConflict    = "conflict-held"
	EventHeldLate    = "held-late"
)

// Late-arrival policies, applied when the business month of an event is not
// open when it arrives.
const (
	PolicyAdjust = "adjust-open-period"
	PolicyHold   = "hold"
	// PolicyKeyPrice is the policy key of a retroactive price change that
	// reprices slips of a closed month.
	PolicyKeyPrice = "price-revision"
)

// Slip kinds.
const (
	SlipOriginal = "original"
	SlipRed      = "red"
	SlipBlack    = "black"
)

// ErrEventMonth refuses an event whose business month is not the month of
// the decision it was given with.
var ErrEventMonth = errors.New("event business month is not the decision month")

// ErrClosedPeriod is a write refused by the period guard.
var ErrClosedPeriod = errors.New("period is not open")

// ErrImmutableSlip is a change of a slip refused by the slip guard.
var ErrImmutableSlip = errors.New("slip is immutable")

// LatePolicy declares how an event or a price change is handled when the
// business month is no longer open.
type LatePolicy struct {
	ID             string            `json:"id"`
	OnClosedPeriod map[string]string `json:"onClosedPeriod"`
}

// Validate checks that every kind has a supported policy and that a price
// change adjusts in the open period.
func (p LatePolicy) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("late policy needs an id")
	}
	for _, k := range []string{EventRegistered, EventChanged, EventCancelled, PolicyKeyPrice} {
		v := p.OnClosedPeriod[k]
		if v != PolicyAdjust && v != PolicyHold {
			return fmt.Errorf("late policy %s: %s has %q, want %s or %s", p.ID, k, v, PolicyAdjust, PolicyHold)
		}
	}
	if p.OnClosedPeriod[PolicyKeyPrice] != PolicyAdjust {
		return fmt.Errorf("late policy %s: %s must be %s", p.ID, PolicyKeyPrice, PolicyAdjust)
	}
	return nil
}

// ReadLatePolicy reads and validates a policy file.
func ReadLatePolicy(path string) (LatePolicy, error) {
	var p LatePolicy
	if err := readJSON(path, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}

// OrderEvent is one event of an order line. Registration carries every
// field; a change carries the new quantity; a cancellation carries none.
type OrderEvent struct {
	EventID      string `json:"eventId"`
	Source       string `json:"source"` // file or message; the same key may arrive by both
	Kind         string `json:"kind"`
	OrderNo      string `json:"orderNo"`
	Version      int    `json:"version"`
	BusinessDate string `json:"businessDate"`
	Customer     string `json:"customer,omitempty"`
	ItemCode     string `json:"itemCode,omitempty"`
	Qty          int64  `json:"qty,omitempty"`
	ReceivedAt   string `json:"receivedAt"`
}

// Validate checks the shape of an event.
func (e OrderEvent) Validate() error {
	if e.EventID == "" || e.OrderNo == "" || e.Version < 1 {
		return fmt.Errorf("event %q: id, order and version >= 1 are required", e.EventID)
	}
	if _, err := parseCivil(e.BusinessDate); err != nil {
		return fmt.Errorf("event %s: business date: %w", e.EventID, err)
	}
	if _, err := time.Parse(time.RFC3339, e.ReceivedAt); err != nil {
		return fmt.Errorf("event %s: receivedAt: %w", e.EventID, err)
	}
	switch e.Kind {
	case EventRegistered:
		if e.Version != 1 || e.Customer == "" || e.ItemCode == "" || e.Qty <= 0 {
			return fmt.Errorf("event %s: a registration is version 1 with customer, item and qty > 0", e.EventID)
		}
	case EventChanged:
		if e.Version < 2 || e.Qty <= 0 {
			return fmt.Errorf("event %s: a change is version >= 2 with qty > 0", e.EventID)
		}
	case EventCancelled:
		if e.Version < 2 {
			return fmt.Errorf("event %s: a cancellation is version >= 2", e.EventID)
		}
	default:
		return fmt.Errorf("event %s: unknown kind %q", e.EventID, e.Kind)
	}
	return nil
}

// ContentDigest identifies the business content of an event, independent
// of its id, source and reception.
func (e OrderEvent) ContentDigest() string {
	parts := []string{e.Kind, e.OrderNo, strconv.Itoa(e.Version), e.BusinessDate, e.Customer, e.ItemCode, strconv.FormatInt(e.Qty, 10)}
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprintf(h, "%d:%s;", len(p), p)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// OrderState is the projection of an order line.
type OrderState struct {
	OrderNo  string `json:"orderNo"`
	Status   string `json:"status"` // pending-base, active, cancelled
	Customer string `json:"customer,omitempty"`
	ItemCode string `json:"itemCode,omitempty"`
	ShipDate string `json:"shipDate,omitempty"`
	Qty      int64  `json:"qty"`
	Version  int    `json:"version"`
}

// FoldOrder computes the state of one order from its accepted events in
// version order. The arrival order of the events does not matter. Without
// the registration the order is pending and keeps its events.
func FoldOrder(orderNo string, events []OrderEvent) OrderState {
	evs := append([]OrderEvent(nil), events...)
	sort.Slice(evs, func(i, j int) bool { return evs[i].Version < evs[j].Version })
	st := OrderState{OrderNo: orderNo, Status: "pending-base"}
	for _, e := range evs {
		if e.Version > st.Version {
			st.Version = e.Version
		}
	}
	var reg *OrderEvent
	for i := range evs {
		if evs[i].Kind == EventRegistered {
			reg = &evs[i]
		}
	}
	if reg == nil {
		return st
	}
	st.Status, st.Customer, st.ItemCode, st.ShipDate, st.Qty = "active", reg.Customer, reg.ItemCode, reg.BusinessDate, reg.Qty
	for _, e := range evs {
		switch e.Kind {
		case EventChanged:
			st.Qty, st.Status = e.Qty, "active"
		case EventCancelled:
			st.Qty, st.Status = 0, "cancelled"
		}
	}
	return st
}

// SlipRef summarizes one posted slip.
type SlipRef struct {
	SlipID        int64  `json:"slipId"`
	Kind          string `json:"kind"`
	OrderNo       string `json:"orderNo"`
	PostingMonth  string `json:"postingMonth"`
	BusinessMonth string `json:"businessMonth"`
	PostingReason string `json:"postingReason"`
	Qty           int64  `json:"qty"`
	UnitPrice     string `json:"unitPrice"`
	Amount        string `json:"amount"`
	Reverses      int64  `json:"reverses,omitempty"`
	Corrects      int64  `json:"corrects,omitempty"`
	PriceID       int64  `json:"priceId"`
	PriceValidOn  string `json:"priceValidOn"`
	PriceKnownAt  string `json:"priceKnownAt"`
	CloseToken    int64  `json:"closeToken,omitempty"`
}

// EventResult is the record of one delivered event.
type EventResult struct {
	EventID     string     `json:"eventId"`
	OrderNo     string     `json:"orderNo"`
	Version     int        `json:"version"`
	Outcome     string     `json:"outcome"`
	PeriodState string     `json:"periodState"`
	Policy      string     `json:"policy,omitempty"`
	State       OrderState `json:"state"`
	Slips       []SlipRef  `json:"slips,omitempty"`
}

// ledgerScope carries what every slip of a step-4/5 call records: the
// decision of the business month and, once that month is closed, its close.
type ledgerScope struct {
	d      Decision
	policy LatePolicy
	loc    *time.Location
}

func newScope(d Decision, p LatePolicy) (ledgerScope, error) {
	if err := p.Validate(); err != nil {
		return ledgerScope{}, err
	}
	loc, err := d.Location()
	if err != nil {
		return ledgerScope{}, err
	}
	return ledgerScope{d: d, policy: p, loc: loc}, nil
}

func periodState(ctx context.Context, tx pgx.Tx, month string) (string, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.period(month,state) VALUES ($1,'open') ON CONFLICT DO NOTHING`, month); err != nil {
		return "", err
	}
	var st string
	err := tx.QueryRow(ctx, `SELECT state FROM mecc.period WHERE month=$1 FOR SHARE`, month).Scan(&st)
	return st, err
}

// ApplyEvent records one delivery of an order event and books its effect.
// One transaction: the order row is locked, so events of one order apply
// one at a time in any arrival order.
func ApplyEvent(ctx context.Context, conn *pgx.Conn, d Decision, pol LatePolicy, ev OrderEvent) (EventResult, error) {
	res := EventResult{EventID: ev.EventID, OrderNo: ev.OrderNo, Version: ev.Version}
	sc, err := newScope(d, pol)
	if err != nil {
		return res, err
	}
	if err := ev.Validate(); err != nil {
		return res, err
	}
	if ev.Source == "" {
		ev.Source = SourceMessage
	}
	if m := ev.BusinessDate[:7]; m != d.TargetMonth {
		return res, fmt.Errorf("%w: event %s is dated %s, decision month %s", ErrEventMonth, ev.EventID, ev.BusinessDate, d.TargetMonth)
	}
	receivedAt, _ := time.Parse(time.RFC3339, ev.ReceivedAt)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)
	if err := RequireActive(ctx, tx, d); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.order_state(order_no,status) VALUES ($1,'pending-base') ON CONFLICT DO NOTHING`, ev.OrderNo); err != nil {
		return res, err
	}
	var curVersion int
	if err := tx.QueryRow(ctx, `SELECT version FROM mecc.order_state WHERE order_no=$1 FOR UPDATE`, ev.OrderNo).Scan(&curVersion); err != nil {
		return res, err
	}
	digest := ev.ContentDigest()
	body, _ := json.Marshal(ev)
	logDelivery := func(outcome string) error {
		_, err := tx.Exec(ctx, `INSERT INTO mecc.order_event_delivery(event_id,outcome) VALUES ($1,$2)`, ev.EventID, outcome)
		return err
	}
	// The same event id again is a redelivery.
	var prevDigest string
	err = tx.QueryRow(ctx, `SELECT content_digest FROM mecc.order_event WHERE event_id=$1`, ev.EventID).Scan(&prevDigest)
	if err == nil {
		if prevDigest != digest {
			return res, fmt.Errorf("event %s redelivered with other content", ev.EventID)
		}
		if _, err := tx.Exec(ctx, `UPDATE mecc.order_event SET deliveries=deliveries+1 WHERE event_id=$1`, ev.EventID); err != nil {
			return res, err
		}
		res.Outcome = EventRedelivered
		if err := logDelivery(res.Outcome); err != nil {
			return res, err
		}
		res.State, err = loadState(ctx, tx, ev.OrderNo)
		if err != nil {
			return res, err
		}
		return res, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return res, err
	}
	state, err := periodState(ctx, tx, ev.BusinessDate[:7])
	if err != nil {
		return res, err
	}
	res.PeriodState = state
	insert := func(status, policy string) error {
		var pol any
		if policy != "" {
			pol = policy
		}
		_, err := tx.Exec(ctx, `INSERT INTO mecc.order_event(event_id,kind,order_no,version,business_date,received_at,source,body,content_digest,status,period_state,policy,arrived_after_version,decision_digest,calendar_version)
			VALUES ($1,$2,$3,$4,$5::text::date,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			ev.EventID, ev.Kind, ev.OrderNo, ev.Version, ev.BusinessDate, receivedAt, ev.Source, body, digest, status, state, pol, curVersion, d.Digest, d.CalendarVersion)
		return err
	}
	// The same ordering key from another id or route is the same event if
	// the content matches, and a held conflict otherwise.
	var keyDigest string
	err = tx.QueryRow(ctx, `SELECT content_digest FROM mecc.order_event WHERE order_no=$1 AND version=$2 AND status IN ('accepted','held','released')`, ev.OrderNo, ev.Version).Scan(&keyDigest)
	if err == nil {
		status, outcome := "confirmed", EventConfirmed
		if keyDigest != digest {
			status, outcome = "conflict", EventConflict
		}
		if err := insert(status, ""); err != nil {
			return res, err
		}
		res.Outcome = outcome
		if err := logDelivery(outcome); err != nil {
			return res, err
		}
		if res.State, err = loadState(ctx, tx, ev.OrderNo); err != nil {
			return res, err
		}
		return res, tx.Commit(ctx)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return res, err
	}
	policy := ""
	if state != "open" {
		policy = sc.policy.OnClosedPeriod[ev.Kind]
		res.Policy = policy
		if policy == PolicyHold {
			if err := insert("held", policy); err != nil {
				return res, err
			}
			res.Outcome = EventHeldLate
			if err := logDelivery(res.Outcome); err != nil {
				return res, err
			}
			if res.State, err = loadState(ctx, tx, ev.OrderNo); err != nil {
				return res, err
			}
			return res, tx.Commit(ctx)
		}
	}
	if err := insert("accepted", policy); err != nil {
		return res, err
	}
	st, err := reproject(ctx, tx, ev.OrderNo)
	if err != nil {
		return res, err
	}
	res.State = st
	switch {
	case st.Status == "pending-base":
		res.Outcome = EventPendingBase
	case ev.Version < st.Version:
		res.Outcome = EventStale
	default:
		res.Outcome = EventApplied
	}
	if err := logDelivery(res.Outcome); err != nil {
		return res, err
	}
	reason := "late:" + policy
	if res.Slips, err = resync(ctx, tx, sc, st, receivedAt, "event:"+ev.EventID, reason); err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}

// ReleaseHeld lets a person apply an event that the hold policy kept out.
// Its effect is booked like a late event, at the release instant.
func ReleaseHeld(ctx context.Context, conn *pgx.Conn, d Decision, pol LatePolicy, eventID string, at time.Time) (EventResult, error) {
	res := EventResult{EventID: eventID}
	sc, err := newScope(d, pol)
	if err != nil {
		return res, err
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)
	if err := RequireActive(ctx, tx, d); err != nil {
		return res, err
	}
	var status string
	if err := tx.QueryRow(ctx, `SELECT order_no, version, status FROM mecc.order_event WHERE event_id=$1`, eventID).Scan(&res.OrderNo, &res.Version, &status); err != nil {
		return res, fmt.Errorf("event %s: %w", eventID, err)
	}
	if _, err := tx.Exec(ctx, `SELECT 1 FROM mecc.order_state WHERE order_no=$1 FOR UPDATE`, res.OrderNo); err != nil {
		return res, err
	}
	if status != "held" {
		return res, fmt.Errorf("event %s is %s, not held", eventID, status)
	}
	if _, err := tx.Exec(ctx, `UPDATE mecc.order_event SET status='released' WHERE event_id=$1`, eventID); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.order_event_delivery(event_id,outcome) VALUES ($1,'released')`, eventID); err != nil {
		return res, err
	}
	st, err := reproject(ctx, tx, res.OrderNo)
	if err != nil {
		return res, err
	}
	res.State, res.Outcome = st, "released"
	if res.Slips, err = resync(ctx, tx, sc, st, at, "release:"+eventID, "late:hold-released"); err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}

func loadState(ctx context.Context, tx pgx.Tx, orderNo string) (OrderState, error) {
	st := OrderState{OrderNo: orderNo}
	var customer, item, ship *string
	var qty *int64
	err := tx.QueryRow(ctx, `SELECT status, customer, item_code, ship_date::text, qty, version FROM mecc.order_state WHERE order_no=$1`, orderNo).
		Scan(&st.Status, &customer, &item, &ship, &qty, &st.Version)
	if customer != nil {
		st.Customer, st.ItemCode, st.ShipDate = *customer, *item, *ship
	}
	if qty != nil {
		st.Qty = *qty
	}
	return st, err
}

// reproject folds the accepted and released events of an order and stores
// the projection.
func reproject(ctx context.Context, tx pgx.Tx, orderNo string) (OrderState, error) {
	rows, err := tx.Query(ctx, `SELECT body FROM mecc.order_event WHERE order_no=$1 AND status IN ('accepted','released')`, orderNo)
	if err != nil {
		return OrderState{}, err
	}
	var evs []OrderEvent
	for rows.Next() {
		var b []byte
		var e OrderEvent
		if err := rows.Scan(&b); err != nil {
			rows.Close()
			return OrderState{}, err
		}
		if err := json.Unmarshal(b, &e); err != nil {
			rows.Close()
			return OrderState{}, err
		}
		evs = append(evs, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return OrderState{}, err
	}
	st := FoldOrder(orderNo, evs)
	var customer, item, ship any
	var qty any
	if st.Status != "pending-base" {
		customer, item, ship, qty = st.Customer, st.ItemCode, st.ShipDate, st.Qty
	}
	_, err = tx.Exec(ctx, `UPDATE mecc.order_state SET status=$2, customer=$3, item_code=$4, ship_date=$5::text::date, qty=$6, version=$7, updated_at=clock_timestamp() WHERE order_no=$1`,
		orderNo, st.Status, customer, item, ship, qty, st.Version)
	return st, err
}

type liveSlip struct {
	id                     int64
	item, shipDate         string
	qty, priceSen          int64
	priceID                int64
	priceValidOn, customer string
	priceKnownAt           time.Time
	taxRate                int
}

func liveSlipOf(ctx context.Context, tx pgx.Tx, orderNo string) (*liveSlip, error) {
	var s liveSlip
	err := tx.QueryRow(ctx, `SELECT s.slip_id, s.item_code, s.ship_date::text, s.qty, (s.unit_price*100)::bigint, s.price_id, s.price_valid_on::text, s.price_known_at, s.customer, s.tax_rate
		FROM mecc.slip s WHERE s.order_no=$1 AND s.kind IN ('original','black') AND NOT EXISTS (SELECT 1 FROM mecc.slip r WHERE r.reverses=s.slip_id)`, orderNo).
		Scan(&s.id, &s.item, &s.shipDate, &s.qty, &s.priceSen, &s.priceID, &s.priceValidOn, &s.priceKnownAt, &s.customer, &s.taxRate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

// resync books the difference between the order's projection and its live
// slip: nothing if they agree, otherwise a red slip for the live slip and a
// black slip for the projection, or an original if nothing was booked. The
// price is the one valid on the ship date as known at knownAt. A slip goes
// to its business month while that month is open, otherwise to the next
// open month with lateReason.
func resync(ctx context.Context, tx pgx.Tx, sc ledgerScope, st OrderState, knownAt time.Time, cause, lateReason string) ([]SlipRef, error) {
	live, err := liveSlipOf(ctx, tx, st.OrderNo)
	if err != nil {
		return nil, err
	}
	var want *Price
	if st.Status == "active" {
		p, err := PriceAt(ctx, tx, st.ItemCode, st.ShipDate, knownAt)
		if err != nil {
			return nil, err
		}
		want = &p
	}
	if live == nil && want == nil {
		return nil, nil
	}
	if live != nil && want != nil && live.item == st.ItemCode && live.shipDate == st.ShipDate && live.qty == st.Qty && live.priceSen == want.UnitPriceSen {
		return nil, nil
	}
	var out []SlipRef
	if live != nil {
		red := slipInput{kind: SlipRed, orderNo: st.OrderNo, customer: live.customer, item: live.item, taxRate: live.taxRate, shipDate: live.shipDate,
			qty: -live.qty, priceSen: live.priceSen, priceID: live.priceID, priceValidOn: live.priceValidOn, priceKnownAt: live.priceKnownAt, reverses: live.id}
		ref, err := postSlip(ctx, tx, sc, red, knownAt, cause, lateReason)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	if want != nil {
		in := slipInput{kind: SlipOriginal, orderNo: st.OrderNo, customer: st.Customer, item: st.ItemCode, taxRate: want.TaxRate, shipDate: st.ShipDate,
			qty: st.Qty, priceSen: want.UnitPriceSen, priceID: want.PriceID, priceValidOn: st.ShipDate, priceKnownAt: knownAt}
		if live != nil {
			in.kind, in.corrects = SlipBlack, live.id
		} else {
			// A re-booking after a cancellation corrects the last reversed slip.
			var last *int64
			if err := tx.QueryRow(ctx, `SELECT max(s.slip_id) FROM mecc.slip s WHERE s.order_no=$1 AND s.kind IN ('original','black') AND NOT EXISTS (SELECT 1 FROM mecc.slip b WHERE b.corrects=s.slip_id)`, st.OrderNo).Scan(&last); err != nil {
				return nil, err
			}
			if last != nil {
				in.kind, in.corrects = SlipBlack, *last
			}
		}
		ref, err := postSlip(ctx, tx, sc, in, knownAt, cause, lateReason)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

type slipInput struct {
	kind, orderNo, customer, item, shipDate, priceValidOn string
	taxRate                                               int
	qty, priceSen, priceID                                int64
	priceKnownAt                                          time.Time
	reverses, corrects                                    int64
}

func nullID(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func postSlip(ctx context.Context, tx pgx.Tx, sc ledgerScope, in slipInput, knownAt time.Time, cause, lateReason string) (SlipRef, error) {
	bm := in.shipDate[:7]
	if bm != sc.d.TargetMonth {
		return SlipRef{}, fmt.Errorf("%w: slip of %s dated %s under the decision of %s", ErrEventMonth, in.orderNo, in.shipDate, sc.d.TargetMonth)
	}
	ref := SlipRef{Kind: in.kind, OrderNo: in.orderNo, BusinessMonth: bm, PostingMonth: bm, PostingReason: PostingBusinessMonth, Qty: in.qty,
		Reverses: in.reverses, Corrects: in.corrects, PriceID: in.priceID, PriceValidOn: in.priceValidOn, PriceKnownAt: in.priceKnownAt.UTC().Format(time.RFC3339)}
	for i := 0; ; i++ {
		if i > 24 {
			return ref, fmt.Errorf("no open period after %s", bm)
		}
		st, err := periodState(ctx, tx, ref.PostingMonth)
		if err != nil {
			return ref, err
		}
		if st == "open" {
			break
		}
		ref.PostingMonth, ref.PostingReason = NextMonth(ref.PostingMonth), lateReason
	}
	// A slip of a closed month carries the close it corrects.
	var closeToken *int64
	var closeDigest *string
	var closedMonth any
	err := tx.QueryRow(ctx, `SELECT fencing_token, decision_digest FROM mecc.close_run WHERE month=$1 AND status='completed'`, bm).Scan(&closeToken, &closeDigest)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ref, err
	}
	if closeToken != nil {
		closedMonth, ref.CloseToken = bm, *closeToken
		if *closeDigest != sc.d.Digest {
			return ref, fmt.Errorf("%w: %s was closed under decision %s, received %s", ErrDecisionMismatch, bm, short(*closeDigest), short(sc.d.Digest))
		}
	}
	var amountSen int64
	if err := tx.QueryRow(ctx, `INSERT INTO mecc.slip(kind,order_no,customer,item_code,tax_rate,ship_date,business_month,posting_month,posting_reason,qty,unit_price,amount,
		price_id,price_valid_on,price_known_at,known_at,reverses,corrects,cause,decision_digest,calendar_version,closed_month,close_token,close_decision_digest)
		VALUES ($1,$2,$3,$4,$5,$6::text::date,$7,$8,$9,$10::bigint,$11::text::numeric,$10::bigint*$11::text::numeric,$12,$13::text::date,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		RETURNING slip_id, (amount*100)::bigint`,
		in.kind, in.orderNo, in.customer, in.item, in.taxRate, in.shipDate, bm, ref.PostingMonth, ref.PostingReason, in.qty, FormatSen(in.priceSen),
		in.priceID, in.priceValidOn, in.priceKnownAt, knownAt, nullID(in.reverses), nullID(in.corrects), cause, sc.d.Digest, sc.d.CalendarVersion,
		closedMonth, closeToken, closeDigest).Scan(&ref.SlipID, &amountSen); err != nil {
		return ref, err
	}
	ref.UnitPrice, ref.Amount = FormatSen(in.priceSen), FormatSen(amountSen)
	return ref, nil
}

// ReviseResult is the record of a price change and the slips it booked.
type ReviseResult struct {
	Change PriceChange `json:"change"`
	Slips  []SlipRef   `json:"slips"`
}

// RevisePrice records a revision of an item's price at the explicit record
// instant (retroactive, current or reserved by its date). With reprice, the
// live slips of the decision's month for that item are repriced at the
// same instant in the same transaction.
func RevisePrice(ctx context.Context, conn *pgx.Conn, d *Decision, pol LatePolicy, item, validFrom, price string, at time.Time) (ReviseResult, error) {
	return priceTx(ctx, conn, d, pol, item, at, func(tx pgx.Tx, loc *time.Location) (PriceChange, error) {
		return revisePriceTx(ctx, tx, item, validFrom, price, at, loc)
	})
}

// CancelRevision withdraws a revision at the explicit record instant, and
// reprices like RevisePrice.
func CancelRevision(ctx context.Context, conn *pgx.Conn, d *Decision, pol LatePolicy, item, validFrom string, at time.Time) (ReviseResult, error) {
	return priceTx(ctx, conn, d, pol, item, at, func(tx pgx.Tx, _ *time.Location) (PriceChange, error) {
		return cancelRevisionTx(ctx, tx, item, validFrom, at)
	})
}

func priceTx(ctx context.Context, conn *pgx.Conn, d *Decision, pol LatePolicy, item string, at time.Time, change func(pgx.Tx, *time.Location) (PriceChange, error)) (ReviseResult, error) {
	var res ReviseResult
	loc := time.UTC
	var sc ledgerScope
	if d != nil {
		var err error
		if sc, err = newScope(*d, pol); err != nil {
			return res, err
		}
		loc = sc.loc
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)
	if d != nil {
		if err := RequireActive(ctx, tx, *d); err != nil {
			return res, err
		}
	}
	c, err := change(tx, loc)
	res.Change = c
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == PriceOverlapSQLState {
			err = fmt.Errorf("%w: %v", ErrPriceOverlap, err)
		}
		if errors.Is(err, ErrPriceRecordedBackwards) || errors.Is(err, ErrNothingToCancel) || errors.Is(err, ErrPriceOverlap) {
			tx.Rollback(ctx)
			_ = recordPriceChange(ctx, conn, &c, "refused", err.Error())
		}
		return res, err
	}
	if d != nil {
		rows, err := tx.Query(ctx, `SELECT s.order_no FROM mecc.slip s WHERE s.item_code=$1 AND s.business_month=$2 AND s.kind IN ('original','black')
			AND NOT EXISTS (SELECT 1 FROM mecc.slip r WHERE r.reverses=s.slip_id) ORDER BY s.order_no`, item, d.TargetMonth)
		if err != nil {
			return res, err
		}
		var orders []string
		for rows.Next() {
			var o string
			if err := rows.Scan(&o); err != nil {
				rows.Close()
				return res, err
			}
			orders = append(orders, o)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return res, err
		}
		for _, o := range orders {
			if _, err := tx.Exec(ctx, `SELECT 1 FROM mecc.order_state WHERE order_no=$1 FOR UPDATE`, o); err != nil {
				return res, err
			}
			st, err := loadState(ctx, tx, o)
			if err != nil {
				return res, err
			}
			refs, err := resync(ctx, tx, sc, st, at, fmt.Sprintf("price:%d", c.ChangeID), "late:"+pol.OnClosedPeriod[PolicyKeyPrice])
			if err != nil {
				return res, err
			}
			res.Slips = append(res.Slips, refs...)
		}
		if _, err := tx.Exec(ctx, `UPDATE mecc.price_change SET repriced=$2 WHERE change_id=$1`, c.ChangeID, len(res.Slips)); err != nil {
			return res, err
		}
	}
	return res, tx.Commit(ctx)
}

// AmendSlipInPlace is the naive correction: it updates the slip's quantity
// and amount directly. The slip guard refuses it; a closed period is
// refused as such.
func AmendSlipInPlace(ctx context.Context, conn *pgx.Conn, slipID, qty int64) error {
	_, err := conn.Exec(ctx, `UPDATE mecc.slip SET qty=$2::bigint, amount=$2::bigint*unit_price WHERE slip_id=$1`, slipID, qty)
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case PeriodSQLState:
			return fmt.Errorf("%w: %s", ErrClosedPeriod, pe.Message)
		case ImmutableSQLState:
			return fmt.Errorf("%w: %s", ErrImmutableSlip, pe.Message)
		}
	}
	return err
}
