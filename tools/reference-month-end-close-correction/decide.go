package monthendclose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrDecisionMismatch stops a later step that received a decision other than
// the month's active decision, or whose inputs were made under another one.
var ErrDecisionMismatch = errors.New("decision mismatch")

// ErrMonthClosing refuses to change the decision of a month whose close has
// started.
var ErrMonthClosing = errors.New("close of the month has started; its decision is fixed")

// LoadCalendar registers a calendar version. The same version with another
// digest is refused: versions are immutable.
func LoadCalendar(ctx context.Context, conn *pgx.Conn, c Calendar) (bool, error) {
	if err := ValidateCalendar(c); err != nil {
		return false, err
	}
	if c.Digest == "" {
		c.Digest = CalendarDigest(c)
	}
	body, _ := json.Marshal(c)
	tag, err := conn.Exec(ctx, `INSERT INTO mecc.calendar_version VALUES ($1,$2,$3,$4,$5::text::timestamptz) ON CONFLICT DO NOTHING`,
		c.ID, c.Version, body, c.Digest, c.RecordedAt)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	var digest string
	if err := conn.QueryRow(ctx, `SELECT digest FROM mecc.calendar_version WHERE calendar_id=$1 AND version=$2`, c.ID, c.Version).Scan(&digest); err != nil {
		return false, err
	}
	if digest != c.Digest {
		return false, fmt.Errorf("calendar %s v%d is already registered with another content (%s)", c.ID, c.Version, digest)
	}
	return false, nil
}

// GetCalendar reads one calendar version and checks its digest.
func GetCalendar(ctx context.Context, q pgx.Tx, id string, version int) (Calendar, error) {
	var body []byte
	if err := q.QueryRow(ctx, `SELECT body FROM mecc.calendar_version WHERE calendar_id=$1 AND version=$2`, id, version).Scan(&body); err != nil {
		return Calendar{}, fmt.Errorf("calendar %s v%d: %w", id, version, err)
	}
	var c Calendar
	if err := json.Unmarshal(body, &c); err != nil {
		return Calendar{}, err
	}
	return c, ValidateCalendar(c)
}

// ReviseCalendar registers the next version of a calendar with added
// holidays. It refuses to fork: the base must be the latest version.
func ReviseCalendar(ctx context.Context, conn *pgx.Conn, id string, from int, add []Holiday, recordedAt string) (Calendar, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return Calendar{}, err
	}
	defer tx.Rollback(ctx)
	var latest int
	if err := tx.QueryRow(ctx, `SELECT coalesce(max(version),0) FROM mecc.calendar_version WHERE calendar_id=$1`, id).Scan(&latest); err != nil {
		return Calendar{}, err
	}
	if latest != from {
		return Calendar{}, fmt.Errorf("calendar %s: revise from v%d, latest is v%d", id, from, latest)
	}
	base, err := GetCalendar(ctx, tx, id, from)
	if err != nil {
		return Calendar{}, err
	}
	next, err := base.Revise(add, recordedAt)
	if err != nil {
		return Calendar{}, err
	}
	body, _ := json.Marshal(next)
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.calendar_version VALUES ($1,$2,$3,$4,$5::text::timestamptz)`, next.ID, next.Version, body, next.Digest, next.RecordedAt); err != nil {
		return Calendar{}, err
	}
	return next, tx.Commit(ctx)
}

// DecideResult is step 1's output.
type DecideResult struct {
	Decision    Decision `json:"decision"`
	Activated   bool     `json:"activated"`
	PreviousDig string   `json:"previousDecisionDigest,omitempty"`
	Impact      *Impact  `json:"impact,omitempty"`
}

// Decide computes the close decision of a month from one calendar version,
// records it, makes it the month's active decision and, when another
// decision was active, records the recomputation difference.
func Decide(ctx context.Context, conn *pgx.Conn, month, calendarID string, version int, rule CloseRule, evaluatedAt time.Time) (DecideResult, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return DecideResult{}, err
	}
	defer tx.Rollback(ctx)
	cal, err := GetCalendar(ctx, tx, calendarID, version)
	if err != nil {
		return DecideResult{}, err
	}
	var payables []Payable
	rows, err := tx.Query(ctx, `SELECT payable_id, nominal_due::text FROM mecc.payable ORDER BY payable_id`)
	if err != nil {
		return DecideResult{}, err
	}
	for rows.Next() {
		var p Payable
		if err := rows.Scan(&p.ID, &p.NominalDue); err != nil {
			return DecideResult{}, err
		}
		payables = append(payables, p)
	}
	if err := rows.Err(); err != nil {
		return DecideResult{}, err
	}
	d, err := ComputeDecision(cal, rule, month, evaluatedAt, payables)
	if err != nil {
		return DecideResult{}, err
	}
	body, _ := json.Marshal(d)
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.close_decision(digest,target_month,calendar_id,calendar_version,body) VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`,
		d.Digest, month, cal.ID, cal.Version, body); err != nil {
		return DecideResult{}, err
	}
	res := DecideResult{Decision: d}
	var prevDigest string
	var prevBody []byte
	err = tx.QueryRow(ctx, `SELECT m.active_digest, d.body FROM mecc.close_month m JOIN mecc.close_decision d ON d.digest=m.active_digest WHERE m.target_month=$1 FOR UPDATE OF m`, month).Scan(&prevDigest, &prevBody)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return DecideResult{}, err
	default:
		res.PreviousDig = prevDigest
	}
	// The month row (locked above) serializes deciders and the close start,
	// which locks the same row before it creates the close run.
	var running int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM mecc.close_run WHERE month=$1`, month).Scan(&running); err != nil {
		return DecideResult{}, err
	}
	if running > 0 {
		return DecideResult{}, fmt.Errorf("%s: %w", month, ErrMonthClosing)
	}
	if prevDigest == d.Digest {
		return res, tx.Commit(ctx)
	}
	if prevDigest != "" {
		var prev Decision
		if err := json.Unmarshal(prevBody, &prev); err != nil {
			return DecideResult{}, err
		}
		prevCal, err := GetCalendar(ctx, tx, prev.CalendarID, prev.CalendarVersion)
		if err != nil {
			return DecideResult{}, err
		}
		im := DiffDecisions(prev, d, prevCal, cal)
		res.Impact = &im
		ib, _ := json.Marshal(im)
		if _, err := tx.Exec(ctx, `INSERT INTO mecc.calendar_impact(target_month,from_digest,to_digest,from_version,to_version,body) VALUES ($1,$2,$3,$4,$5,$6)`,
			month, prev.Digest, d.Digest, prev.CalendarVersion, d.CalendarVersion, ib); err != nil {
			return DecideResult{}, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.close_month(target_month,active_digest,calendar_version) VALUES ($1,$2,$3)
		ON CONFLICT (target_month) DO UPDATE SET active_digest=EXCLUDED.active_digest, calendar_version=EXCLUDED.calendar_version, activated_at=clock_timestamp()`,
		month, d.Digest, d.CalendarVersion); err != nil {
		return DecideResult{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.decision_activation(target_month,digest,calendar_version,previous_digest) VALUES ($1,$2,$3,nullif($4,''))`,
		month, d.Digest, d.CalendarVersion, prevDigest); err != nil {
		return DecideResult{}, err
	}
	res.Activated = true
	return res, tx.Commit(ctx)
}

// queryer is satisfied by *pgx.Conn and pgx.Tx.
type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RequireActive checks that a received decision is intact, recorded, and the
// month's active decision.
func RequireActive(ctx context.Context, q queryer, d Decision) error {
	if err := d.Verify(); err != nil {
		return err
	}
	var active string
	var version int
	err := q.QueryRow(ctx, `SELECT active_digest, calendar_version FROM mecc.close_month WHERE target_month=$1`, d.TargetMonth).Scan(&active, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %s has no decision", ErrDecisionMismatch, d.TargetMonth)
	}
	if err != nil {
		return err
	}
	if active != d.Digest {
		return fmt.Errorf("%w: received calendar v%d decision %s, active is calendar v%d decision %s", ErrDecisionMismatch, d.CalendarVersion, short(d.Digest), version, short(active))
	}
	return nil
}

func short(d string) string {
	if len(d) > 19 {
		return d[:19]
	}
	return d
}
