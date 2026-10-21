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
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Sources of a shipment event.
const (
	SourceFile    = "file"
	SourceMessage = "message"
)

// Receipt outcomes.
const (
	OutcomeApplied     = "applied"
	OutcomeConfirmed   = "confirmed-same-event"
	OutcomeConflict    = "conflict-held"
	OutcomeRedelivered = "redelivered"
)

// ShipmentEvent is the business content of one shipment, whatever the
// source.
type ShipmentEvent struct {
	ShipmentID string `json:"shipmentId"`
	OrderNo    string `json:"orderNo"`
	ShipDate   string `json:"shipDate"` // YYYY-MM-DD
	Qty        int64  `json:"qty"`
	Amount     int64  `json:"amount"`
	ItemName   string `json:"itemName"`
}

// ContentDigest compares two events as one business event. Text is compared
// after the spec's declared equivalents are applied.
func (e ShipmentEvent) ContentDigest(spec *FileSpec) string {
	parts := []string{e.ShipmentID, e.OrderNo, e.ShipDate, strconv.FormatInt(e.Qty, 10), strconv.FormatInt(e.Amount, 10), spec.NormalizeForCompare(e.ItemName)}
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprintf(h, "%d:%s;", len(p), p)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ImportResult is step 2's record of one accepted file.
type ImportResult struct {
	ImportID        int64          `json:"importId"`
	FileSHA256      string         `json:"fileSha256"`
	Sender          string         `json:"sender"`
	FileSeq         int            `json:"fileSeq"`
	Records         int            `json:"records"`
	AmountSum       int64          `json:"amountSum"`
	Held            []HeldRecord   `json:"held"`
	Applied         int            `json:"applied"`
	Outcomes        map[string]int `json:"outcomes"`
	DecisionDigest  string         `json:"decisionDigest"`
	CalendarVersion int            `json:"calendarVersion"`
	CutoffAt        string         `json:"cutoffAt"`
	ReceivedAt      string         `json:"receivedAt"`
}

// HeldRecord is a data record kept as raw bytes and not applied.
type HeldRecord struct {
	Seq        int    `json:"seq"`
	ShipmentID string `json:"shipmentId"`
	Reason     string `json:"reason"`
	Field      string `json:"field"`
}

func recordRejection(ctx context.Context, conn *pgx.Conn, b []byte, reason, detail string, d Decision, receivedAt time.Time) {
	_, _ = conn.Exec(ctx, `INSERT INTO mecc.import_rejection(file_sha256,file_size,reason,detail,decision_digest,received_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		FileSHA256(b), len(b), reason, detail, d.Digest, receivedAt)
}

// ImportFile imports one partner file as one transaction. Every rejection is
// whole-file and recorded; nothing of a rejected file is applied.
func ImportFile(ctx context.Context, conn *pgx.Conn, d Decision, spec *FileSpec, b []byte, receivedAt time.Time) (ImportResult, error) {
	fail := func(err error) (ImportResult, error) {
		if reason, ok := IsReject(err); ok {
			recordRejection(ctx, conn, b, reason, err.Error(), d, receivedAt)
		}
		return ImportResult{}, err
	}
	if err := RequireActive(ctx, conn, d); err != nil {
		return fail(&RejectError{Reason: RejectDecision, Detail: err.Error()})
	}
	cutoff, _ := time.Parse(time.RFC3339, d.CutoffAt)
	if receivedAt.After(cutoff) {
		return fail(reject(RejectAfterCutoff, "received %s, cut-off %s", receivedAt.Format(time.RFC3339), d.CutoffAt))
	}
	pf, err := spec.ParseFile(b)
	if err != nil {
		return fail(err)
	}
	if m := pf.Header.Values[spec.MonthField]; m != strings.ReplaceAll(d.TargetMonth, "-", "") {
		return fail(reject(RejectMonth, "file month %s, decision month %s", m, d.TargetMonth))
	}
	sender := pf.Header.Values[spec.SenderField]
	seq, _ := strconv.Atoi(pf.Header.Values[spec.FileSeqField])
	tx, err := conn.Begin(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback(ctx)
	// The sender row serializes imports of one sender.
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.import_sender VALUES ($1) ON CONFLICT DO NOTHING`, sender); err != nil {
		return ImportResult{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT 1 FROM mecc.import_sender WHERE sender=$1 FOR UPDATE`, sender); err != nil {
		return ImportResult{}, err
	}
	if err := RequireActive(ctx, tx, d); err != nil {
		tx.Rollback(ctx)
		return fail(&RejectError{Reason: RejectDecision, Detail: err.Error()})
	}
	var prevID int64
	var prevSeq int
	err = tx.QueryRow(ctx, `SELECT import_id, file_seq FROM mecc.import_unit WHERE file_sha256=$1`, pf.SHA256).Scan(&prevID, &prevSeq)
	if err == nil {
		tx.Rollback(ctx)
		return fail(reject(RejectDuplicateFile, "same file already imported as import %d", prevID))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ImportResult{}, err
	}
	err = tx.QueryRow(ctx, `SELECT import_id FROM mecc.import_unit WHERE sender=$1 AND file_seq=$2`, sender, seq).Scan(&prevID)
	if err == nil {
		tx.Rollback(ctx)
		return fail(reject(RejectSeqReused, "sender %s file sequence %d already imported as import %d with other content", sender, seq, prevID))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ImportResult{}, err
	}
	sum, _ := strconv.ParseInt(pf.Trailer.Values[spec.TrailerSum], 10, 64)
	res := ImportResult{FileSHA256: pf.SHA256, Sender: sender, FileSeq: seq, Records: len(pf.Data), AmountSum: sum, Held: []HeldRecord{},
		Outcomes: map[string]int{}, DecisionDigest: d.Digest, CalendarVersion: d.CalendarVersion, CutoffAt: d.CutoffAt, ReceivedAt: receivedAt.Format(time.RFC3339)}
	for _, r := range pf.Data {
		if r.HeldReason != "" {
			res.Held = append(res.Held, HeldRecord{Seq: r.Seq, ShipmentID: r.Values[spec.EventMapping.ShipmentID], Reason: r.HeldReason, Field: r.HeldField})
		}
	}
	if err := tx.QueryRow(ctx, `INSERT INTO mecc.import_unit(sender,file_seq,file_sha256,file_size,spec_id,spec_digest,received_at,target_month,decision_digest,calendar_version,record_count,amount_sum,held_count,applied_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,0) RETURNING import_id`,
		sender, seq, pf.SHA256, pf.Size, spec.ID, SpecDigest(spec), receivedAt, d.TargetMonth, d.Digest, d.CalendarVersion, len(pf.Data), sum, len(res.Held)).Scan(&res.ImportID); err != nil {
		return ImportResult{}, err
	}
	all := append(append([]Record{pf.Header}, pf.Data...), pf.Trailer)
	for _, r := range all {
		vals, _ := json.Marshal(r.Values)
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.import_record VALUES ($1,$2,$3,$4,$5,nullif($6,''),nullif($7,''))`,
			res.ImportID, r.Seq, r.Kind, vals, r.Raw, r.HeldReason, r.HeldField); err != nil {
			return ImportResult{}, err
		}
	}
	for _, r := range pf.Data {
		if r.HeldReason != "" {
			continue
		}
		ev, err := spec.eventFromRecord(r)
		if err != nil {
			return ImportResult{}, err
		}
		outcome, err := applyShipment(ctx, tx, spec, d, SourceFile, fmt.Sprintf("%s#%d", pf.SHA256, r.Seq), ev)
		if err != nil {
			return ImportResult{}, err
		}
		res.Outcomes[outcome]++
		res.Applied++
	}
	if _, err := tx.Exec(ctx, `UPDATE mecc.import_unit SET applied_count=$2 WHERE import_id=$1`, res.ImportID, res.Applied); err != nil {
		return ImportResult{}, err
	}
	crashPoint(CrashImportBeforeCommit)
	return res, tx.Commit(ctx)
}

func (s *FileSpec) eventFromRecord(r Record) (ShipmentEvent, error) {
	m := s.EventMapping
	qty, err1 := strconv.ParseInt(r.Values[m.Qty], 10, 64)
	amt, err2 := strconv.ParseInt(r.Values[m.Amount], 10, 64)
	if err := errors.Join(err1, err2); err != nil {
		return ShipmentEvent{}, err
	}
	sd := r.Values[m.ShipDate]
	return ShipmentEvent{ShipmentID: r.Values[m.ShipmentID], OrderNo: r.Values[m.OrderNo], ShipDate: sd[:4] + "-" + sd[4:6] + "-" + sd[6:],
		Qty: qty, Amount: amt, ItemName: r.Values[m.ItemName]}, nil
}

// applyShipment records one delivery of a shipment event. The same source
// reference delivered again is a redelivery; the same shipment from another
// reference with the same content confirms the event; other content is held
// as a conflict and never overwrites the event.
func applyShipment(ctx context.Context, tx pgx.Tx, spec *FileSpec, d Decision, source, ref string, ev ShipmentEvent) (string, error) {
	digest := ev.ContentDigest(spec)
	var outcome string
	tag, err := tx.Exec(ctx, `UPDATE mecc.shipment_receipt SET deliveries=deliveries+1 WHERE source=$1 AND source_ref=$2`, source, ref)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 1 {
		outcome = OutcomeRedelivered
	} else {
		tag, err := tx.Exec(ctx, `INSERT INTO mecc.shipment_event(shipment_id,order_no,ship_date,qty,amount,item_name,content_digest,first_source,first_ref,decision_digest)
			VALUES ($1,$2,$3::text::date,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (shipment_id) DO NOTHING`,
			ev.ShipmentID, ev.OrderNo, ev.ShipDate, ev.Qty, ev.Amount, ev.ItemName, digest, source, ref, d.Digest)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 1 {
			outcome = OutcomeApplied
		} else {
			var existing string
			if err := tx.QueryRow(ctx, `SELECT content_digest FROM mecc.shipment_event WHERE shipment_id=$1`, ev.ShipmentID).Scan(&existing); err != nil {
				return "", err
			}
			outcome = OutcomeConfirmed
			if existing != digest {
				outcome = OutcomeConflict
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.shipment_receipt(source,source_ref,shipment_id,content_digest,outcome,decision_digest) VALUES ($1,$2,$3,$4,$5,$6)`,
			source, ref, ev.ShipmentID, digest, outcome, d.Digest); err != nil {
			return "", err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO mecc.delivery_log(source,source_ref,shipment_id,outcome) VALUES ($1,$2,$3,$4)`, source, ref, ev.ShipmentID, outcome)
	return outcome, err
}

// Message is one simulated message of the shipment topic. MessageID is the
// broker's identity; redelivery repeats it.
type Message struct {
	MessageID string        `json:"messageId"`
	Event     ShipmentEvent `json:"event"`
}

// ReceiveMessage applies one delivery of a message.
func ReceiveMessage(ctx context.Context, conn *pgx.Conn, d Decision, spec *FileSpec, m Message) (string, error) {
	if m.MessageID == "" || m.Event.ShipmentID == "" {
		return "", fmt.Errorf("message id and shipment id are required")
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := RequireActive(ctx, tx, d); err != nil {
		return "", err
	}
	outcome, err := applyShipment(ctx, tx, spec, d, SourceMessage, m.MessageID, m.Event)
	if err != nil {
		return "", err
	}
	return outcome, tx.Commit(ctx)
}

// Reconciliation classes.
const (
	ClassBoth        = "file+message"
	ClassFileOnly    = "file-only"
	ClassMessageOnly = "message-only"
	ClassConflict    = "conflict"
	ClassFileHeld    = "message-only+file-held"
	ClassHeldOnly    = "file-held-only"
)

// ShipmentReconciliation is the summary of one reconciliation run.
type ShipmentReconciliation struct {
	RunID           int64             `json:"runId"`
	DecisionDigest  string            `json:"decisionDigest"`
	CalendarVersion int               `json:"calendarVersion"`
	Events          int               `json:"events"`
	Classes         map[string]int    `json:"classes"`
	Receipts        int               `json:"receipts"`
	Deliveries      int               `json:"deliveries"`
	Qty             int64             `json:"qty"`
	Amount          int64             `json:"amount"`
	ByShipment      map[string]string `json:"byShipment"`
}

// ReconcileShipments classifies every shipment by the sources that
// delivered it and records the result.
func ReconcileShipments(ctx context.Context, conn *pgx.Conn, d Decision) (ShipmentReconciliation, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return ShipmentReconciliation{}, err
	}
	defer tx.Rollback(ctx)
	if err := RequireActive(ctx, tx, d); err != nil {
		return ShipmentReconciliation{}, err
	}
	res := ShipmentReconciliation{DecisionDigest: d.Digest, CalendarVersion: d.CalendarVersion, Classes: map[string]int{}, ByShipment: map[string]string{}}
	if err := tx.QueryRow(ctx, `SELECT coalesce(max(run_id),0)+1 FROM mecc.shipment_reconciliation`).Scan(&res.RunID); err != nil {
		return res, err
	}
	type acc struct {
		sources            map[string]bool
		receipts, delivery int
		conflict           bool
	}
	ship := map[string]*acc{}
	get := func(id string) *acc {
		if ship[id] == nil {
			ship[id] = &acc{sources: map[string]bool{}}
		}
		return ship[id]
	}
	rows, err := tx.Query(ctx, `SELECT shipment_id, source, outcome, deliveries FROM mecc.shipment_receipt WHERE decision_digest=$1`, d.Digest)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var id, source, outcome string
		var n int
		if err := rows.Scan(&id, &source, &outcome, &n); err != nil {
			return res, err
		}
		a := get(id)
		a.sources[source] = true
		a.receipts++
		a.delivery += n
		a.conflict = a.conflict || outcome == OutcomeConflict
	}
	if err := rows.Err(); err != nil {
		return res, err
	}
	held := map[string]bool{}
	rows, err = tx.Query(ctx, `SELECT r.vals->>'shipmentId' FROM mecc.import_record r JOIN mecc.import_unit u USING (import_id)
		WHERE u.decision_digest=$1 AND r.held_reason IS NOT NULL`, d.Digest)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return res, err
		}
		held[id] = true
		get(id)
	}
	if err := rows.Err(); err != nil {
		return res, err
	}
	ids := make([]string, 0, len(ship))
	for id := range ship {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		a := ship[id]
		var class string
		switch {
		case a.conflict:
			class = ClassConflict
		case a.sources[SourceFile] && a.sources[SourceMessage]:
			class = ClassBoth
		case a.sources[SourceFile]:
			class = ClassFileOnly
		case a.sources[SourceMessage] && held[id]:
			class = ClassFileHeld
		case a.sources[SourceMessage]:
			class = ClassMessageOnly
		default:
			class = ClassHeldOnly
		}
		srcs := []string{}
		for s := range a.sources {
			srcs = append(srcs, s)
		}
		sort.Strings(srcs)
		res.Classes[class]++
		res.ByShipment[id] = class
		res.Receipts += a.receipts
		res.Deliveries += a.delivery
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.shipment_reconciliation(run_id,shipment_id,class,sources,receipts,deliveries,decision_digest,calendar_version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			res.RunID, id, class, srcs, a.receipts, a.delivery, d.Digest, d.CalendarVersion); err != nil {
			return res, err
		}
	}
	if err := tx.QueryRow(ctx, `SELECT count(*), coalesce(sum(qty),0), coalesce(sum(amount),0)::bigint FROM mecc.shipment_event WHERE decision_digest=$1`, d.Digest).Scan(&res.Events, &res.Qty, &res.Amount); err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}

// Reemit rebuilds an imported file from the stored records: decoded values
// are encoded again by the spec, held records are written from their raw
// bytes.
func Reemit(ctx context.Context, conn *pgx.Conn, spec *FileSpec, importID int64) ([]byte, error) {
	rows, err := conn.Query(ctx, `SELECT seq, kind, vals, raw, coalesce(held_reason,'') FROM mecc.import_record WHERE import_id=$1 ORDER BY seq`, importID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []Record
	for rows.Next() {
		var r Record
		var vals []byte
		if err := rows.Scan(&r.Seq, &r.Kind, &vals, &r.Raw, &r.HeldReason); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(vals, &r.Values); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("import %d has no records", importID)
	}
	return spec.EmitFile(recs)
}
