package dispatch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Claim represents an exclusive lease on an issue for a role/model assignment.
type Claim struct {
	IssueNumber  int64     `json:"issueNumber"`
	Role         string    `json:"role"`
	ModelID      string    `json:"modelId"`
	Harness      string    `json:"harness"`
	SessionID    string    `json:"sessionId,omitempty"`
	LeaseExpires time.Time `json:"leaseExpires"`
	HeartbeatAt  time.Time `json:"heartbeatAt"`
	State        string    `json:"state"`
	BaseSHA      string    `json:"baseSha,omitempty"`
	HeadSHA      string    `json:"headSha,omitempty"`
	PRNumber     *int64    `json:"prNumber,omitempty"`
}

// LeaseStore persists claims in a local SQLite database.
type LeaseStore struct {
	db *sql.DB
}

// NewLeaseStore opens (or creates) the SQLite lease database.
func NewLeaseStore(path string) (*LeaseStore, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	// Two goroutines claiming at once contend for the write lock. Without a busy
	// timeout the loser gets SQLITE_BUSY immediately and reports it as an
	// unexpected error, so the exclusivity this store exists to provide looks
	// like a broken database. The DSN carries the timeout; the PRAGMA keeps the
	// connection honest if a future driver ignores the parameter.
	dsn := path + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open lease db: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("set wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec(leaseSchema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &LeaseStore{db: db}, nil
}

// Close releases the underlying database handle.
func (s *LeaseStore) Close() error {
	return s.db.Close()
}

// ClaimIssue tries to claim an issue exclusively for the given role/model.
// It returns ErrAlreadyClaimed if another active claim exists.
func (s *LeaseStore) ClaimIssue(ctx context.Context, issueNumber int64, role, modelID, harness, sessionID string, ttl time.Duration) (*Claim, error) {
	now := time.Now()
	expires := now.Add(ttl)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	// Remove terminal claims so the same issue can be reclaimed after release/expiry.
	if _, err := tx.ExecContext(ctx, `DELETE FROM claims WHERE issue_number=? AND state IN ('released','expired')`, issueNumber); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("clear terminal claims: %w", err)
	}

	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM claims WHERE issue_number=? AND state IN ('ready','working','review','blocked')`, issueNumber).Scan(&existing); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing > 0 {
		_ = tx.Rollback()
		return nil, ErrAlreadyClaimed
	}

	if _, err := tx.ExecContext(ctx, insertClaim, issueNumber, role, modelID, harness, sessionID, expires.Format(time.RFC3339), now.Format(time.RFC3339), "working"); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("insert claim: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}
	return &Claim{
		IssueNumber:  issueNumber,
		Role:         role,
		ModelID:      modelID,
		Harness:      harness,
		SessionID:    sessionID,
		LeaseExpires: expires,
		HeartbeatAt:  now,
		State:        "working",
	}, nil
}

// GetClaim returns the current claim for an issue, if any.
func (s *LeaseStore) GetClaim(ctx context.Context, issueNumber int64) (*Claim, error) {
	row := s.db.QueryRowContext(ctx, `SELECT issue_number, role, model_id, harness, session_id, lease_expires, heartbeat_at, state, base_sha, head_sha, pr_number FROM claims WHERE issue_number=? ORDER BY updated_at DESC LIMIT 1`, issueNumber)
	return scanClaim(row)
}

// Release marks a claim as released.
func (s *LeaseStore) Release(ctx context.Context, issueNumber int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE claims SET state='released', updated_at=datetime('now') WHERE issue_number=? AND state IN ('ready','working','review','blocked')`, issueNumber)
	if err != nil {
		return fmt.Errorf("release claim: %w", err)
	}
	return nil
}

// Heartbeat renews the lease for an active claim.
func (s *LeaseStore) Heartbeat(ctx context.Context, issueNumber int64, ttl time.Duration) error {
	now := time.Now()
	expires := now.Add(ttl)
	_, err := s.db.ExecContext(ctx, `UPDATE claims SET heartbeat_at=?, lease_expires=?, updated_at=datetime('now') WHERE issue_number=? AND state IN ('working','review','blocked')`, now.Format(time.RFC3339), expires.Format(time.RFC3339), issueNumber)
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	return nil
}

// SweepExpired moves expired claims to 'expired' state so they can be reclaimed.
func (s *LeaseStore) SweepExpired(ctx context.Context) ([]Claim, error) {
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE claims SET state='expired', updated_at=datetime('now') WHERE state IN ('working','review','blocked') AND lease_expires < ?`, now)
	if err != nil {
		return nil, fmt.Errorf("sweep expired: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT issue_number, role, model_id, harness, session_id, lease_expires, heartbeat_at, state, base_sha, head_sha, pr_number FROM claims WHERE state='expired' AND updated_at = datetime('now')`)
	if err != nil {
		return nil, fmt.Errorf("query expired: %w", err)
	}
	defer rows.Close()
	var expired []Claim
	for rows.Next() {
		c, err := scanClaimRow(rows)
		if err != nil {
			return nil, err
		}
		expired = append(expired, *c)
	}
	return expired, nil
}

// ListActive returns all claims in ready/working/review/blocked state for a role.
func (s *LeaseStore) ListActive(ctx context.Context, role string) ([]Claim, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT issue_number, role, model_id, harness, session_id, lease_expires, heartbeat_at, state, base_sha, head_sha, pr_number FROM claims WHERE role=? AND state IN ('ready','working','review','blocked')`, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Claim
	for rows.Next() {
		c, err := scanClaimRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

// UpdateState updates the state of a claim.
func (s *LeaseStore) UpdateState(ctx context.Context, issueNumber int64, state string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE claims SET state=?, updated_at=datetime('now') WHERE issue_number=?`, state, issueNumber)
	return err
}

// UpdatePR sets the PR number on a claim.
func (s *LeaseStore) UpdatePR(ctx context.Context, issueNumber int64, prNumber int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE claims SET pr_number=?, updated_at=datetime('now') WHERE issue_number=?`, prNumber, issueNumber)
	return err
}

// UpdateHeadSHA sets the head SHA on a claim.
func (s *LeaseStore) UpdateHeadSHA(ctx context.Context, issueNumber int64, headSHA string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE claims SET head_sha=?, updated_at=datetime('now') WHERE issue_number=?`, headSHA, issueNumber)
	return err
}

func scanClaim(row *sql.Row) (*Claim, error) {
	var c Claim
	var leaseExpires, heartbeatAt string
	var baseSHA, headSHA sql.NullString
	var prNumber sql.NullInt64
	if err := row.Scan(&c.IssueNumber, &c.Role, &c.ModelID, &c.Harness, &c.SessionID, &leaseExpires, &heartbeatAt, &c.State, &baseSHA, &headSHA, &prNumber); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.LeaseExpires, _ = time.Parse(time.RFC3339, leaseExpires)
	c.HeartbeatAt, _ = time.Parse(time.RFC3339, heartbeatAt)
	if baseSHA.Valid {
		c.BaseSHA = baseSHA.String
	}
	if headSHA.Valid {
		c.HeadSHA = headSHA.String
	}
	if prNumber.Valid {
		c.PRNumber = &prNumber.Int64
	}
	return &c, nil
}

func scanClaimRow(rows *sql.Rows) (*Claim, error) {
	var c Claim
	var leaseExpires, heartbeatAt string
	var baseSHA, headSHA sql.NullString
	var prNumber sql.NullInt64
	if err := rows.Scan(&c.IssueNumber, &c.Role, &c.ModelID, &c.Harness, &c.SessionID, &leaseExpires, &heartbeatAt, &c.State, &baseSHA, &headSHA, &prNumber); err != nil {
		return nil, err
	}
	c.LeaseExpires, _ = time.Parse(time.RFC3339, leaseExpires)
	c.HeartbeatAt, _ = time.Parse(time.RFC3339, heartbeatAt)
	if baseSHA.Valid {
		c.BaseSHA = baseSHA.String
	}
	if headSHA.Valid {
		c.HeadSHA = headSHA.String
	}
	if prNumber.Valid {
		c.PRNumber = &prNumber.Int64
	}
	return &c, nil
}

var ErrAlreadyClaimed = errors.New("issue already claimed")

const leaseSchema = `CREATE TABLE IF NOT EXISTS claims (
    issue_number INTEGER PRIMARY KEY,
    role TEXT NOT NULL,
    model_id TEXT NOT NULL,
    harness TEXT NOT NULL,
    session_id TEXT,
    lease_expires TEXT NOT NULL,
    heartbeat_at TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'working',
    base_sha TEXT,
    head_sha TEXT,
    pr_number INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_claims_role_state ON claims(role, state);
CREATE INDEX IF NOT EXISTS idx_claims_lease ON claims(lease_expires);`

const insertClaim = `INSERT INTO claims (issue_number, role, model_id, harness, session_id, lease_expires, heartbeat_at, state) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
