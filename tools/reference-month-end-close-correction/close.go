package monthendclose

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tei/internal/resource"
)

// Errors of step 3.
var (
	ErrAlreadyRunning   = errors.New("close of the month is already running")
	ErrAlreadyCompleted = errors.New("close of the month is already completed")
	ErrNotYetScheduled  = errors.New("close job is not yet scheduled at the as-of instant")
	ErrNoImport         = errors.New("no import unit for the month")
	ErrCloseMismatch    = errors.New("close totals do not match the cut")
)

// EntryInput is one online order or correction.
type EntryInput struct {
	OrderNo    string
	Kind       string // "order" or "correction"
	AcceptedAt time.Time
	Amount     int64
	Awaiting   bool // left pending allocation
}

// Entry is a stored sales entry.
type Entry struct {
	ID            int64  `json:"entryId"`
	OrderNo       string `json:"orderNo"`
	BusinessDate  string `json:"businessDate"`
	BusinessMonth string `json:"businessMonth"`
	PostingMonth  string `json:"postingMonth"`
	PostingReason string `json:"postingReason"`
}

// Posting reasons.
const (
	PostingBusinessMonth = "business-month"
	PostingRedirected    = "redirected:period-not-open"
)

// PlaceEntry stores one online entry. Its business date is the local date of
// acceptance in the decision's timezone. It posts to its business month while
// that period is open; once the period is closing or closed it posts to the
// next open period and keeps its business date. The period row is read with
// a share lock, so the close's state change waits for in-flight entries and
// later entries see the new state.
func PlaceEntry(ctx context.Context, conn *pgx.Conn, d Decision, in EntryInput) (Entry, error) {
	bd, bm, err := d.BusinessDate(in.AcceptedAt)
	if err != nil {
		return Entry{}, err
	}
	if in.Kind == "" {
		in.Kind = "order"
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return Entry{}, err
	}
	defer tx.Rollback(ctx)
	e := Entry{OrderNo: in.OrderNo, BusinessDate: bd, BusinessMonth: bm, PostingMonth: bm, PostingReason: PostingBusinessMonth}
	for i := 0; ; i++ {
		if i > 24 {
			return Entry{}, fmt.Errorf("no open period after %s", bm)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.period(month,state) VALUES ($1,'open') ON CONFLICT DO NOTHING`, e.PostingMonth); err != nil {
			return Entry{}, err
		}
		var state string
		if err := tx.QueryRow(ctx, `SELECT state FROM mecc.period WHERE month=$1 FOR SHARE`, e.PostingMonth).Scan(&state); err != nil {
			return Entry{}, err
		}
		if state == "open" {
			break
		}
		e.PostingMonth, e.PostingReason = NextMonth(e.PostingMonth), PostingRedirected
	}
	alloc := "allocated"
	if in.Awaiting {
		alloc = "awaiting"
	}
	if err := tx.QueryRow(ctx, `INSERT INTO mecc.sales_entry(order_no,kind,accepted_at,business_date,business_month,posting_month,posting_reason,amount,alloc_status)
		VALUES ($1,$2,$3,$4::text::date,$5,$6,$7,$8,$9) RETURNING entry_id`,
		in.OrderNo, in.Kind, in.AcceptedAt, bd, bm, e.PostingMonth, e.PostingReason, in.Amount, alloc).Scan(&e.ID); err != nil {
		return Entry{}, err
	}
	return e, tx.Commit(ctx)
}

// AllocateEntry completes the allocation of a pending entry. It changes
// neither the amount nor the posting period.
func AllocateEntry(ctx context.Context, conn *pgx.Conn, id int64) error {
	_, err := conn.Exec(ctx, `UPDATE mecc.sales_entry SET alloc_status='allocated' WHERE entry_id=$1`, id)
	return err
}

// CloseOptions configures one close process.
type CloseOptions struct {
	OwnerID   string
	TTL       time.Duration
	ChunkSize int
	AsOf      time.Time
	HostNow   func() time.Time
	// ChunkPause is waited after each committed chunk (test pacing only).
	ChunkPause time.Duration
}

// CloseResult is step 3's record.
type CloseResult struct {
	Month           string  `json:"month"`
	Status          string  `json:"status"`
	DecisionDigest  string  `json:"decisionDigest"`
	CalendarVersion int     `json:"calendarVersion"`
	CloseDate       string  `json:"closeDate"`
	ImportDigest    string  `json:"importDigest"`
	ImportIDs       []int64 `json:"importIds"`
	OwnerID         string  `json:"ownerId"`
	FencingToken    uint64  `json:"fencingToken"`
	Takeover        bool    `json:"takeover"`
	Starts          int     `json:"starts"`
	CutAt           string  `json:"cutAt"`
	CutSnapshot     string  `json:"cutSnapshot"`
	Entries         int     `json:"entries"`
	Amount          int64   `json:"amount"`
	Awaiting        int     `json:"awaiting"`
	Shipments       int     `json:"shipments"`
	ShipmentAmount  int64   `json:"shipmentAmount"`
	Chunks          int     `json:"chunks"`
	ChunksThisRun   int     `json:"chunksThisRun"`
}

func recordAttempt(ctx context.Context, conn *pgx.Conn, month, owner, outcome string, token uint64, msg string) {
	_, _ = conn.Exec(ctx, `INSERT INTO mecc.close_attempt(month,owner_id,outcome,fencing_token,message) VALUES ($1,$2,$3,$4,$5)`, month, owner, outcome, int64(token), msg)
}

func isFence(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == FenceSQLState
}

// CloseJobKey is the business key of the close job of a month.
func CloseJobKey(month string) string { return "close:" + month }

// RunClose runs (or resumes) the close job of the decision's month. The job
// is identified by the month: a completed month is refused, a running month
// whose lease is alive is refused, and a running month whose lease expired
// is resumed by the caller with a new fencing token. Every write carries the
// token and the database refuses a stale one.
func RunClose(ctx context.Context, conn *pgx.Conn, d Decision, opt CloseOptions) (CloseResult, error) {
	month := d.TargetMonth
	res := CloseResult{Month: month, DecisionDigest: d.Digest, CalendarVersion: d.CalendarVersion, CloseDate: d.CloseDate, OwnerID: opt.OwnerID}
	refuse := func(outcome string, err error) (CloseResult, error) {
		recordAttempt(ctx, conn, month, opt.OwnerID, outcome, res.FencingToken, err.Error())
		return res, err
	}
	if opt.OwnerID == "" || opt.TTL <= 0 || opt.ChunkSize < 1 {
		return res, fmt.Errorf("owner, positive ttl and chunk size are required")
	}
	if err := RequireActive(ctx, conn, d); err != nil {
		return refuse("refused-decision", err)
	}
	jobAt, err := d.JobAt(JobClose)
	if err != nil {
		return res, err
	}
	if opt.AsOf.Before(jobAt) {
		return refuse("refused-not-scheduled", fmt.Errorf("%w: as of %s, scheduled %s (calendar v%d)", ErrNotYetScheduled, opt.AsOf.Format(time.RFC3339), jobAt.Format(time.RFC3339), d.CalendarVersion))
	}
	rows, err := conn.Query(ctx, `SELECT import_id, file_sha256, decision_digest, calendar_version FROM mecc.import_unit WHERE target_month=$1 ORDER BY import_id`, month)
	if err != nil {
		return res, err
	}
	var idents []string
	var mismatch error
	for rows.Next() {
		var id int64
		var sha, dig string
		var ver int
		if err := rows.Scan(&id, &sha, &dig, &ver); err != nil {
			rows.Close()
			return res, err
		}
		if dig != d.Digest && mismatch == nil {
			mismatch = fmt.Errorf("%w: import %d was made under calendar v%d decision %s, close received calendar v%d decision %s", ErrDecisionMismatch, id, ver, short(dig), d.CalendarVersion, short(d.Digest))
		}
		res.ImportIDs = append(res.ImportIDs, id)
		idents = append(idents, fmt.Sprintf("%d=%s", id, sha))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	if mismatch != nil {
		return refuse("refused-decision", mismatch)
	}
	if len(idents) == 0 {
		return refuse("refused-no-import", fmt.Errorf("%w: %s", ErrNoImport, month))
	}
	res.ImportDigest = digestOf(idents)
	var status string
	err = conn.QueryRow(ctx, `SELECT status FROM mecc.close_run WHERE month=$1`, month).Scan(&status)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return res, err
	}
	if status == "completed" {
		return refuse("refused-completed", fmt.Errorf("%w: %s", ErrAlreadyCompleted, month))
	}
	store := &PgOwnershipStore{Conn: conn, HostNow: opt.HostNow}
	own, takeover, err := store.Claim(ctx, CloseJobKey(month), opt.OwnerID, opt.TTL)
	if err != nil {
		var held *LeaseHeldError
		if errors.As(err, &held) {
			return refuse("refused-running", fmt.Errorf("%w: %v (host-clock design would take over: %v)", ErrAlreadyRunning, err, held.NaiveWouldTakeOver))
		}
		return res, err
	}
	res.FencingToken, res.Takeover = own.FencingToken, takeover
	if err := startRun(ctx, conn, d, &res, own); err != nil {
		if errors.Is(err, ErrAlreadyCompleted) || errors.Is(err, ErrDecisionMismatch) {
			_ = store.Release(ctx, own)
			outcome := "refused-completed"
			if errors.Is(err, ErrDecisionMismatch) {
				outcome = "refused-decision"
			}
			return refuse(outcome, err)
		}
		if isFence(err) {
			return refuse("fenced", fmt.Errorf("%w: %v", ErrFenced, err))
		}
		return res, err
	}
	outcome := "started"
	if res.Starts > 1 {
		outcome = "resumed"
	}
	recordAttempt(ctx, conn, month, opt.OwnerID, outcome, own.FencingToken, "")
	for {
		n, err := closeChunk(ctx, conn, &res, opt)
		if err != nil {
			if isFence(err) {
				return refuse("fenced", fmt.Errorf("%w: %v", ErrFenced, err))
			}
			return res, err
		}
		if n == 0 {
			break
		}
		res.ChunksThisRun++
		if opt.ChunkPause > 0 {
			time.Sleep(opt.ChunkPause)
		}
		if own, err = store.Heartbeat(ctx, own, opt.TTL); err != nil {
			return refuse("lost-lease", err)
		}
	}
	if err := finishRun(ctx, conn, d, &res); err != nil {
		if isFence(err) {
			return refuse("fenced", fmt.Errorf("%w: %v", ErrFenced, err))
		}
		return res, err
	}
	res.Status = "completed"
	recordAttempt(ctx, conn, month, opt.OwnerID, "completed", own.FencingToken, "")
	if err := store.Release(ctx, own); err != nil {
		return res, err
	}
	return res, nil
}

func startRun(ctx context.Context, conn *pgx.Conn, d Decision, res *CloseResult, own resource.SupervisorOwnership) error {
	month := d.TargetMonth
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var active string
	if err := tx.QueryRow(ctx, `SELECT active_digest FROM mecc.close_month WHERE target_month=$1 FOR SHARE`, month).Scan(&active); err != nil {
		return err
	}
	if active != d.Digest {
		return fmt.Errorf("%w: active decision changed to %s", ErrDecisionMismatch, short(active))
	}
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM mecc.close_run WHERE month=$1 FOR UPDATE`, month).Scan(&status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// First start: change the period to closing, which waits for every
		// entry transaction holding the period row, then cut.
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.period(month,state) VALUES ($1,'open') ON CONFLICT DO NOTHING`, month); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE mecc.period SET state='closing', changed_at=clock_timestamp() WHERE month=$1`, month); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.close_run(month,decision_digest,calendar_version,import_digest,import_ids,status,fencing_token,owner_id,as_of,
			cut_at,cut_snapshot,cut_entries,cut_amount,cut_max_entry,cut_shipments,cut_shipment_amount)
			SELECT $1,$2,$3,$4,$5,'running',$6,$7,$8,clock_timestamp(),pg_current_snapshot()::text,
			  (SELECT count(*) FROM mecc.sales_entry WHERE posting_month=$1),
			  (SELECT coalesce(sum(amount),0) FROM mecc.sales_entry WHERE posting_month=$1),
			  (SELECT coalesce(max(entry_id),0) FROM mecc.sales_entry WHERE posting_month=$1),
			  (SELECT count(*) FROM mecc.shipment_event WHERE decision_digest=$2 AND to_char(ship_date,'YYYY-MM')=$1),
			  (SELECT coalesce(sum(amount),0) FROM mecc.shipment_event WHERE decision_digest=$2 AND to_char(ship_date,'YYYY-MM')=$1)`,
			month, d.Digest, d.CalendarVersion, res.ImportDigest, res.ImportIDs, int64(own.FencingToken), own.OwnerID, time.Now()); err != nil {
			return err
		}
	case err != nil:
		return err
	case status == "completed":
		return fmt.Errorf("%w: %s", ErrAlreadyCompleted, month)
	default:
		var dig, imp string
		if err := tx.QueryRow(ctx, `SELECT decision_digest, import_digest FROM mecc.close_run WHERE month=$1`, month).Scan(&dig, &imp); err != nil {
			return err
		}
		if dig != d.Digest || imp != res.ImportDigest {
			return fmt.Errorf("%w: running close was started with decision %s imports %s", ErrDecisionMismatch, short(dig), short(imp))
		}
		if _, err := tx.Exec(ctx, `UPDATE mecc.close_run SET fencing_token=$2, owner_id=$3, starts=starts+1 WHERE month=$1`, month, int64(own.FencingToken), own.OwnerID); err != nil {
			return err
		}
	}
	var cutAt time.Time
	var cutAmount, shipAmount int64
	if err := tx.QueryRow(ctx, `SELECT starts, cut_at, cut_snapshot, cut_entries, cut_amount::bigint, cut_shipments, cut_shipment_amount::bigint FROM mecc.close_run WHERE month=$1`, month).
		Scan(&res.Starts, &cutAt, &res.CutSnapshot, &res.Entries, &cutAmount, &res.Shipments, &shipAmount); err != nil {
		return err
	}
	res.CutAt, res.Amount, res.ShipmentAmount = cutAt.UTC().Format(time.RFC3339Nano), cutAmount, shipAmount
	return tx.Commit(ctx)
}

func closeChunk(ctx context.Context, conn *pgx.Conn, res *CloseResult, opt CloseOptions) (int, error) {
	month := res.Month
	var last, cutMax int64
	var chunkNo int
	if err := conn.QueryRow(ctx, `SELECT coalesce((SELECT max(entry_id) FROM mecc.close_detail WHERE month=$1),0), (SELECT count(*) FROM mecc.close_chunk WHERE month=$1)+1,
		(SELECT cut_max_entry FROM mecc.close_run WHERE month=$1)`, month).Scan(&last, &chunkNo, &cutMax); err != nil {
		return 0, err
	}
	type row struct {
		id     int64
		amount int64
		bd     string
		alloc  string
	}
	rows, err := conn.Query(ctx, `SELECT entry_id, amount::bigint, business_date::text, alloc_status FROM mecc.sales_entry
		WHERE posting_month=$1 AND entry_id > $2 AND entry_id <= $3 ORDER BY entry_id LIMIT $4`, month, last, cutMax, opt.ChunkSize)
	if err != nil {
		return 0, err
	}
	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.amount, &r.bd, &r.alloc); err != nil {
			rows.Close()
			return 0, err
		}
		batch = append(batch, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(batch) == 0 {
		return 0, err
	}
	// A stopped owner resumes here with the batch it selected before the
	// stop; its write must be refused by the database, not by itself.
	stopPoint(PointBeforeChunkWrite)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var sum int64
	for i, r := range batch {
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.close_detail VALUES ($1,$2,$3,$4,$5::text::date,$6,$7)`, month, r.id, chunkNo, r.amount, r.bd, r.alloc, int64(res.FencingToken)); err != nil {
			return 0, err
		}
		sum += r.amount
		if i == len(batch)/2 {
			crashPoint(CrashCloseChunkMid)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.close_chunk(month,chunk_no,first_entry,last_entry,entries,amount,owner_id,fencing_token) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		month, chunkNo, batch[0].id, batch[len(batch)-1].id, len(batch), sum, res.OwnerID, int64(res.FencingToken)); err != nil {
		return 0, err
	}
	return len(batch), tx.Commit(ctx)
}

func finishRun(ctx context.Context, conn *pgx.Conn, d Decision, res *CloseResult) error {
	month := d.TargetMonth
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var entries, awaiting int
	var amount int64
	if err := tx.QueryRow(ctx, `SELECT count(*), coalesce(sum(amount),0)::bigint, count(*) FILTER (WHERE alloc_status='awaiting') FROM mecc.close_detail WHERE month=$1`, month).
		Scan(&entries, &amount, &awaiting); err != nil {
		return err
	}
	if entries != res.Entries || amount != res.Amount {
		return fmt.Errorf("%w: details %d entries %d yen, cut %d entries %d yen", ErrCloseMismatch, entries, amount, res.Entries, res.Amount)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM mecc.close_chunk WHERE month=$1`, month).Scan(&res.Chunks); err != nil {
		return err
	}
	res.Awaiting = awaiting
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.close_total VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		month, entries, amount, awaiting, res.Shipments, res.ShipmentAmount, d.Digest, d.CalendarVersion, res.ImportDigest, int64(res.FencingToken), res.OwnerID); err != nil {
		return err
	}
	crashPoint(CrashCloseBeforeComplete)
	if _, err := tx.Exec(ctx, `UPDATE mecc.close_run SET status='completed', completed_at=clock_timestamp(), fencing_token=$2 WHERE month=$1`, month, int64(res.FencingToken)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE mecc.period SET state='closed', changed_at=clock_timestamp() WHERE month=$1`, month); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
