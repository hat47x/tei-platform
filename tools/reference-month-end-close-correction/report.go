package monthendclose

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Report bases.
const (
	// BasisClose: the slips posted to the month as recorded at the close's
	// cut (database clock), with the close's tax. Reissuing it reproduces
	// the bytes the close stored.
	BasisClose = "close"
	// BasisCurrent: the balance of the month's business date as recorded
	// now, corrections posted to later months included, with the
	// difference from the close explained slip by slip.
	BasisCurrent = "current"
)

// ErrReportMismatch is a reissued close report that differs from the
// stored one.
var ErrReportMismatch = errors.New("reissued report differs from the close")

// ReportHeader is what a report carries from steps 1 and 3.
type ReportHeader struct {
	Month           string
	Basis           string
	DecisionDigest  string
	CalendarVersion int
	CloseDate       string
	CloseToken      int64
	CutAt           string
	TaxRule         string
}

// ReportLine is one slip of a report.
type ReportLine struct {
	SlipID        int64
	Kind          string
	OrderNo       string
	Customer      string
	ItemCode      string
	Rate          int
	ShipDate      string
	PostingMonth  string
	Qty           int64
	UnitPriceSen  int64
	AmountSen     int64
	Reverses      int64
	Corrects      int64
	PriceKnownAt  string
	PostingReason string
	Cause         string
}

// RenderReport writes a report as UTF-8 text with LF line ends. Every value
// comes from the arguments; nothing depends on the host or the session.
func RenderReport(h ReportHeader, lines []ReportLine, inv []InvoiceTax) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "report,month-end-close-correction,v1\n")
	fmt.Fprintf(&b, "month,%s\nbasis,%s\ndecision,%s\ncalendarVersion,%d\ncloseDate,%s\ncloseToken,%d\ncutAt,%s\ntaxRule,%s\n",
		h.Month, h.Basis, h.DecisionDigest, h.CalendarVersion, h.CloseDate, h.CloseToken, h.CutAt, h.TaxRule)
	fmt.Fprintf(&b, "lines,slip,kind,order,customer,item,rate,shipDate,postingMonth,qty,unitPrice,amount,reverses,corrects\n")
	var base int64
	for _, l := range lines {
		fmt.Fprintf(&b, "line,%d,%s,%s,%s,%s,%d,%s,%s,%d,%s,%s,%d,%d\n", l.SlipID, l.Kind, l.OrderNo, l.Customer, l.ItemCode, l.Rate, l.ShipDate, l.PostingMonth,
			l.Qty, FormatSen(l.UnitPriceSen), FormatSen(l.AmountSen), l.Reverses, l.Corrects)
		base += l.AmountSen
	}
	if inv != nil {
		fmt.Fprintf(&b, "invoices,customer,postingMonth,rate,lines,base,tax\n")
		var tax int64
		for _, v := range inv {
			fmt.Fprintf(&b, "invoice,%s,%s,%d,%d,%s,%d\n", v.Customer, v.PostingMonth, v.Rate, v.Lines, FormatSen(v.BaseSen), v.TaxYen)
			tax += v.TaxYen
		}
		fmt.Fprintf(&b, "total,%d,%s,%d\n", len(lines), FormatSen(base), tax)
	} else {
		fmt.Fprintf(&b, "total,%d,%s\n", len(lines), FormatSen(base))
	}
	return b.Bytes()
}

// ReportSHA256 is the digest stored with a close report.
func ReportSHA256(b []byte) string {
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:])
}

func loadReportLines(ctx context.Context, tx pgx.Tx, where string, args ...any) ([]ReportLine, error) {
	rows, err := tx.Query(ctx, `SELECT slip_id, kind, order_no, customer, item_code, tax_rate, ship_date::text, posting_month, qty, (unit_price*100)::bigint, (amount*100)::bigint,
		coalesce(reverses,0), coalesce(corrects,0), price_known_at, posting_reason, cause FROM mecc.slip WHERE `+where+` ORDER BY slip_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReportLine
	for rows.Next() {
		var l ReportLine
		var known time.Time
		if err := rows.Scan(&l.SlipID, &l.Kind, &l.OrderNo, &l.Customer, &l.ItemCode, &l.Rate, &l.ShipDate, &l.PostingMonth, &l.Qty, &l.UnitPriceSen, &l.AmountSen,
			&l.Reverses, &l.Corrects, &known, &l.PostingReason, &l.Cause); err != nil {
			return nil, err
		}
		l.PriceKnownAt = known.UTC().Format(time.RFC3339)
		out = append(out, l)
	}
	return out, rows.Err()
}

// closeReport builds the close-basis report of the month inside tx, from
// the slips recorded at or before the cut.
func closeReport(ctx context.Context, tx pgx.Tx, h ReportHeader, cutAt time.Time) ([]byte, []ReportLine, []InvoiceTax, error) {
	lines, err := loadReportLines(ctx, tx, `posting_month=$1 AND inserted_at <= $2`, h.Month, cutAt)
	if err != nil {
		return nil, nil, nil, err
	}
	// The tax is computed over exactly these slips.
	var ids []int64
	for _, l := range lines {
		ids = append(ids, l.SlipID)
	}
	rows, err := tx.Query(ctx, `SELECT customer, posting_month, tax_rate, count(*), (sum(amount)*100)::bigint, div(sum(amount)*tax_rate, 100)::bigint
		FROM mecc.slip WHERE slip_id = ANY($1) GROUP BY customer, posting_month, tax_rate`, ids)
	if err != nil {
		return nil, nil, nil, err
	}
	inv := []InvoiceTax{}
	for rows.Next() {
		var v InvoiceTax
		if err := rows.Scan(&v.Customer, &v.PostingMonth, &v.Rate, &v.Lines, &v.BaseSen, &v.TaxYen); err != nil {
			rows.Close()
			return nil, nil, nil, err
		}
		inv = append(inv, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	sortInvoices(inv)
	return RenderReport(h, lines, inv), lines, inv, nil
}

// writeCloseReport is called by the close in its completing transaction.
// The report row is fenced like the other writes of the close.
func writeCloseReport(ctx context.Context, tx pgx.Tx, d Decision, res *CloseResult) error {
	var cutAt time.Time
	if err := tx.QueryRow(ctx, `SELECT cut_at FROM mecc.close_run WHERE month=$1`, d.TargetMonth).Scan(&cutAt); err != nil {
		return err
	}
	h := ReportHeader{Month: d.TargetMonth, Basis: BasisClose, DecisionDigest: d.Digest, CalendarVersion: d.CalendarVersion, CloseDate: d.CloseDate,
		CloseToken: int64(res.FencingToken), CutAt: cutAt.UTC().Format(time.RFC3339Nano), TaxRule: DefaultTaxRule.ID}
	b, lines, inv, err := closeReport(ctx, tx, h, cutAt)
	if err != nil {
		return err
	}
	var base, tax int64
	for _, l := range lines {
		base += l.AmountSen
	}
	for _, v := range inv {
		tax += v.TaxYen
	}
	if len(lines) != res.Slips || base != res.SlipAmountSen {
		return fmt.Errorf("%w: slips %d (%s), cut %d (%s)", ErrCloseMismatch, len(lines), FormatSen(base), res.Slips, FormatSen(res.SlipAmountSen))
	}
	res.ReportSHA256, res.TaxTotal = ReportSHA256(b), tax
	_, err = tx.Exec(ctx, `INSERT INTO mecc.close_report(month,slips,base_amount,tax_total,tax_rule,report_sha256,report,decision_digest,calendar_version,fencing_token,owner_id)
		VALUES ($1,$2,$3::text::numeric,$4,$5,$6,$7,$8,$9,$10,$11)`,
		d.TargetMonth, len(lines), FormatSen(base), tax, DefaultTaxRule.ID, res.ReportSHA256, b, d.Digest, d.CalendarVersion, int64(res.FencingToken), res.OwnerID)
	return err
}

// Correction is one slip that makes the current balance differ from the
// close.
type Correction struct {
	SlipID        int64  `json:"slipId"`
	Kind          string `json:"kind"`
	OrderNo       string `json:"orderNo"`
	PostingMonth  string `json:"postingMonth"`
	PostingReason string `json:"postingReason"`
	Reverses      int64  `json:"reverses,omitempty"`
	Corrects      int64  `json:"corrects,omitempty"`
	Amount        string `json:"amount"`
	Cause         string `json:"cause"`
}

// ReissueResult is step 6's record.
type ReissueResult struct {
	Month           string       `json:"month"`
	Basis           string       `json:"basis"`
	DecisionDigest  string       `json:"decisionDigest"`
	CalendarVersion int          `json:"calendarVersion"`
	CloseToken      int64        `json:"closeToken"`
	CutAt           string       `json:"cutAt"`
	Lines           int          `json:"lines"`
	Base            string       `json:"base"`
	TaxTotal        int64        `json:"taxTotal,omitempty"`
	ReportSHA256    string       `json:"reportSha256"`
	StoredSHA256    string       `json:"storedSha256,omitempty"`
	ByteEqual       bool         `json:"byteEqual,omitempty"`
	CloseBase       string       `json:"closeBase,omitempty"`
	Difference      string       `json:"difference,omitempty"`
	Corrections     []Correction `json:"corrections,omitempty"`
	Report          []byte       `json:"-"`
}

// Reissue renders the report of the decision's month on a basis. The
// decision must be the active one and the one the month was closed with.
// On the close basis the bytes must equal the stored report; on the
// current basis the difference from the close must equal the corrections.
func Reissue(ctx context.Context, conn *pgx.Conn, d Decision, basis string) (ReissueResult, error) {
	res := ReissueResult{Month: d.TargetMonth, Basis: basis}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)
	if err := RequireActive(ctx, tx, d); err != nil {
		return res, err
	}
	var cutAt time.Time
	var runDigest, status string
	var token int64
	err = tx.QueryRow(ctx, `SELECT decision_digest, status, fencing_token, cut_at FROM mecc.close_run WHERE month=$1`, d.TargetMonth).Scan(&runDigest, &status, &token, &cutAt)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && status != "completed") {
		return res, fmt.Errorf("%s is not closed", d.TargetMonth)
	}
	if err != nil {
		return res, err
	}
	if runDigest != d.Digest {
		return res, fmt.Errorf("%w: %s was closed under decision %s", ErrDecisionMismatch, d.TargetMonth, short(runDigest))
	}
	h := ReportHeader{Month: d.TargetMonth, Basis: BasisClose, DecisionDigest: d.Digest, CalendarVersion: d.CalendarVersion, CloseDate: d.CloseDate,
		CloseToken: token, CutAt: cutAt.UTC().Format(time.RFC3339Nano), TaxRule: DefaultTaxRule.ID}
	res.DecisionDigest, res.CalendarVersion, res.CloseToken, res.CutAt = h.DecisionDigest, h.CalendarVersion, h.CloseToken, h.CutAt
	var stored []byte
	var storedSHA string
	var storedToken int64
	if err := tx.QueryRow(ctx, `SELECT report, report_sha256, fencing_token FROM mecc.close_report WHERE month=$1`, d.TargetMonth).Scan(&stored, &storedSHA, &storedToken); err != nil {
		return res, fmt.Errorf("close report of %s: %w", d.TargetMonth, err)
	}
	closeBytes, closeLines, inv, err := closeReport(ctx, tx, h, cutAt)
	if err != nil {
		return res, err
	}
	var closeBase int64
	for _, l := range closeLines {
		closeBase += l.AmountSen
	}
	switch basis {
	case BasisClose:
		res.Report, res.Lines, res.Base = closeBytes, len(closeLines), FormatSen(closeBase)
		for _, v := range inv {
			res.TaxTotal += v.TaxYen
		}
		res.ReportSHA256, res.StoredSHA256 = ReportSHA256(closeBytes), storedSHA
		res.ByteEqual = bytes.Equal(closeBytes, stored) && storedToken == token
		if !res.ByteEqual {
			return res, fmt.Errorf("%w: %s, stored %s", ErrReportMismatch, res.ReportSHA256, storedSHA)
		}
	case BasisCurrent:
		h.Basis = BasisCurrent
		lines, err := loadReportLines(ctx, tx, `business_month=$1`, d.TargetMonth)
		if err != nil {
			return res, err
		}
		inClose := map[int64]bool{}
		for _, l := range closeLines {
			inClose[l.SlipID] = true
		}
		var base, diff int64
		for _, l := range lines {
			base += l.AmountSen
			if !inClose[l.SlipID] {
				diff += l.AmountSen
				res.Corrections = append(res.Corrections, Correction{SlipID: l.SlipID, Kind: l.Kind, OrderNo: l.OrderNo, PostingMonth: l.PostingMonth,
					PostingReason: l.PostingReason, Reverses: l.Reverses, Corrects: l.Corrects, Amount: FormatSen(l.AmountSen), Cause: l.Cause})
			}
		}
		if base-closeBase != diff {
			return res, fmt.Errorf("current %s minus close %s is not the corrections %s", FormatSen(base), FormatSen(closeBase), FormatSen(diff))
		}
		res.Report = RenderReport(h, lines, nil)
		res.Lines, res.Base, res.CloseBase, res.Difference = len(lines), FormatSen(base), FormatSen(closeBase), FormatSen(diff)
		res.ReportSHA256 = ReportSHA256(res.Report)
	default:
		return res, fmt.Errorf("basis %q: want %s or %s", basis, BasisClose, BasisCurrent)
	}
	return res, nil
}
