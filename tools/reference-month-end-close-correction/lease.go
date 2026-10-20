package monthendclose

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tei/internal/resource"
)

// PgOwnershipStore implements the product's storage-neutral
// resource.SupervisorOwnershipStore on PostgreSQL. Every expiry decision is
// made by the database clock (clock_timestamp()); the hosts' clocks are only
// recorded. Fencing tokens are allocated from a per-key sequence row and
// strictly increase across expiry, release and takeover.
type PgOwnershipStore struct {
	Conn *pgx.Conn
	// HostNow is the host's clock, possibly skewed. It is recorded with each
	// event and used only to show what a host-clock design would decide.
	HostNow func() time.Time
}

var _ resource.SupervisorOwnershipStore = (*PgOwnershipStore)(nil)

// ErrLeaseHeld is returned when another owner holds an unexpired lease.
var ErrLeaseHeld = errors.New("lease is held by another owner")

// ErrFenced is returned when an owner's token is no longer current or its
// lease expired by the database clock.
var ErrFenced = errors.New("fenced")

const dbTimeLayout = time.RFC3339Nano

func (s *PgOwnershipStore) hostNow() time.Time {
	if s.HostNow != nil {
		return s.HostNow()
	}
	return time.Now()
}

func (s *PgOwnershipStore) event(ctx context.Context, q pgx.Tx, key, owner string, token uint64, state string, naive *bool, msg string) error {
	_, err := q.Exec(ctx, `INSERT INTO mecc.lease_event(job_key,owner_id,fencing_token,state,host_time,naive_would_take_over,message) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		key, owner, int64(token), state, s.hostNow(), naive, msg)
	return err
}

func ownershipFromRow(key, owner string, token int64, claimed, hb, exp time.Time) resource.SupervisorOwnership {
	return resource.SupervisorOwnership{LeaseID: key, OwnerID: owner, FencingToken: uint64(token),
		ClaimedAt: claimed.UTC().Format(dbTimeLayout), HeartbeatAt: hb.UTC().Format(dbTimeLayout), OwnerExpiresAt: exp.UTC().Format(dbTimeLayout)}
}

// Claim takes the lease when it is free or expired by the database clock.
// The same owner renews its own unexpired lease with the same token.
func (s *PgOwnershipStore) Claim(ctx context.Context, key, owner string, ttl time.Duration) (resource.SupervisorOwnership, bool, error) {
	if key == "" || owner == "" || ttl <= 0 {
		return resource.SupervisorOwnership{}, false, fmt.Errorf("lease key, owner and positive ttl are required")
	}
	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO mecc.job_lease_seq VALUES ($1,0) ON CONFLICT DO NOTHING`, key); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	var last int64
	if err := tx.QueryRow(ctx, `SELECT last_token FROM mecc.job_lease_seq WHERE job_key=$1 FOR UPDATE`, key).Scan(&last); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	var prevOwner string
	var prevToken int64
	var prevExp, prevHostExp time.Time
	var expired bool
	err = tx.QueryRow(ctx, `SELECT owner_id, fencing_token, expires_at, owner_host_expires_at, expires_at <= clock_timestamp() FROM mecc.job_lease WHERE job_key=$1`, key).
		Scan(&prevOwner, &prevToken, &prevExp, &prevHostExp, &expired)
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resource.SupervisorOwnership{}, false, err
	}
	hostNow := s.hostNow()
	if exists && !expired {
		if prevOwner != owner {
			// What a design comparing the holder's host-clock expiry with the
			// claimant's host clock would have decided.
			naive := hostNow.After(prevHostExp)
			_ = s.event(ctx, tx, key, owner, uint64(prevToken), "claim-refused", &naive, fmt.Sprintf("held by %s token %d until %s (db clock)", prevOwner, prevToken, prevExp.UTC().Format(dbTimeLayout)))
			if err := tx.Commit(ctx); err != nil {
				return resource.SupervisorOwnership{}, false, err
			}
			return resource.SupervisorOwnership{}, false, &LeaseHeldError{Owner: prevOwner, Token: uint64(prevToken), ExpiresAt: prevExp, NaiveWouldTakeOver: naive}
		}
		var o resource.SupervisorOwnership
		var claimed, hb, exp time.Time
		if err := tx.QueryRow(ctx, `UPDATE mecc.job_lease SET heartbeat_at=clock_timestamp(), expires_at=clock_timestamp()+($2::bigint * interval '1 microsecond'), owner_host_expires_at=$3
			WHERE job_key=$1 RETURNING claimed_at, heartbeat_at, expires_at`, key, ttl.Microseconds(), hostNow.Add(ttl)).Scan(&claimed, &hb, &exp); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		o = ownershipFromRow(key, owner, prevToken, claimed, hb, exp)
		if err := s.event(ctx, tx, key, owner, o.FencingToken, "renewed", nil, ""); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		return o, false, tx.Commit(ctx)
	}
	token := last + 1
	if exists && prevToken >= token {
		token = prevToken + 1
	}
	if _, err := tx.Exec(ctx, `UPDATE mecc.job_lease_seq SET last_token=$2 WHERE job_key=$1`, key, token); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	var claimed, hb, exp time.Time
	if err := tx.QueryRow(ctx, `INSERT INTO mecc.job_lease VALUES ($1,$2,$3,clock_timestamp(),clock_timestamp(),clock_timestamp()+($4::bigint * interval '1 microsecond'),$5)
		ON CONFLICT (job_key) DO UPDATE SET owner_id=EXCLUDED.owner_id, fencing_token=EXCLUDED.fencing_token, claimed_at=EXCLUDED.claimed_at,
		heartbeat_at=EXCLUDED.heartbeat_at, expires_at=EXCLUDED.expires_at, owner_host_expires_at=EXCLUDED.owner_host_expires_at
		RETURNING claimed_at, heartbeat_at, expires_at`, key, owner, token, ttl.Microseconds(), hostNow.Add(ttl)).Scan(&claimed, &hb, &exp); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	takeover := exists && prevOwner != owner
	state := "claimed"
	if takeover {
		state = "takeover"
	}
	msg := ""
	if exists {
		msg = fmt.Sprintf("previous %s token %d expired at %s (db clock)", prevOwner, prevToken, prevExp.UTC().Format(dbTimeLayout))
	}
	if err := s.event(ctx, tx, key, owner, uint64(token), state, nil, msg); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	return ownershipFromRow(key, owner, token, claimed, hb, exp), takeover, tx.Commit(ctx)
}

// LeaseHeldError describes a refused claim.
type LeaseHeldError struct {
	Owner              string
	Token              uint64
	ExpiresAt          time.Time
	NaiveWouldTakeOver bool
}

func (e *LeaseHeldError) Error() string {
	return fmt.Sprintf("%v: %s token %d until %s (db clock)", ErrLeaseHeld, e.Owner, e.Token, e.ExpiresAt.UTC().Format(dbTimeLayout))
}

func (e *LeaseHeldError) Unwrap() error { return ErrLeaseHeld }

// Heartbeat extends an unexpired lease of the same owner and token. It never
// resurrects an expired lease.
func (s *PgOwnershipStore) Heartbeat(ctx context.Context, o resource.SupervisorOwnership, ttl time.Duration) (resource.SupervisorOwnership, error) {
	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return resource.SupervisorOwnership{}, err
	}
	defer tx.Rollback(ctx)
	var claimed, hb, exp time.Time
	err = tx.QueryRow(ctx, `UPDATE mecc.job_lease SET heartbeat_at=clock_timestamp(), expires_at=clock_timestamp()+($4::bigint * interval '1 microsecond'), owner_host_expires_at=$5
		WHERE job_key=$1 AND owner_id=$2 AND fencing_token=$3 AND expires_at > clock_timestamp() RETURNING claimed_at, heartbeat_at, expires_at`,
		o.LeaseID, o.OwnerID, int64(o.FencingToken), ttl.Microseconds(), s.hostNow().Add(ttl)).Scan(&claimed, &hb, &exp)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = s.event(ctx, tx, o.LeaseID, o.OwnerID, o.FencingToken, "heartbeat-fenced", nil, "")
		_ = tx.Commit(ctx)
		return resource.SupervisorOwnership{}, fmt.Errorf("%w: %s token %d is not current or expired", ErrFenced, o.OwnerID, o.FencingToken)
	}
	if err != nil {
		return resource.SupervisorOwnership{}, err
	}
	return ownershipFromRow(o.LeaseID, o.OwnerID, int64(o.FencingToken), claimed, hb, exp), tx.Commit(ctx)
}

// Assert checks that the ownership is current and unexpired.
func (s *PgOwnershipStore) Assert(ctx context.Context, o resource.SupervisorOwnership) error {
	var ok bool
	err := s.Conn.QueryRow(ctx, `SELECT expires_at > clock_timestamp() FROM mecc.job_lease WHERE job_key=$1 AND owner_id=$2 AND fencing_token=$3`,
		o.LeaseID, o.OwnerID, int64(o.FencingToken)).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !ok) {
		return fmt.Errorf("%w: %s token %d", ErrFenced, o.OwnerID, o.FencingToken)
	}
	return err
}

// Release drops the lease if it is still this owner's. The token sequence is
// kept, so the next claim gets a larger token.
func (s *PgOwnershipStore) Release(ctx context.Context, o resource.SupervisorOwnership) error {
	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM mecc.job_lease WHERE job_key=$1 AND owner_id=$2 AND fencing_token=$3`, o.LeaseID, o.OwnerID, int64(o.FencingToken))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: release by %s token %d", ErrFenced, o.OwnerID, o.FencingToken)
	}
	if err := s.event(ctx, tx, o.LeaseID, o.OwnerID, o.FencingToken, "released", nil, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
