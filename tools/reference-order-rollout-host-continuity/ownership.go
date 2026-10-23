package orderrollouthost

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/resource"
)

// ErrFenced is returned when an owner's token is no longer the current one.
var ErrFenced = errors.New("fenced")

// ErrLeaseHeld is returned when another owner holds the supervisor ownership
// and the claimant has not yet seen it stop for a whole ttl.
var ErrLeaseHeld = errors.New("supervisor ownership is held by another owner")

// LeaseHeldError describes a refused claim.
type LeaseHeldError struct {
	Owner string
	Token uint64
	// NaiveHostClockTakeover is what a design comparing the holder's
	// host-clock expiry with the claimant's host clock would have decided.
	NaiveHostClockTakeover bool
}

func (e *LeaseHeldError) Error() string {
	return fmt.Sprintf("%v: %s token %d", ErrLeaseHeld, e.Owner, e.Token)
}

func (e *LeaseHeldError) Unwrap() error { return ErrLeaseHeld }

// execer is satisfied by a pool and a transaction.
type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type ownershipView struct {
	owner string
	token int64
	seq   int64
	since time.Time // monotonic reading of the claimant
}

// PgOwnershipStore implements the product's storage-neutral
// resource.SupervisorOwnershipStore on PostgreSQL.
//
// No wall clock decides anything. The holder's heartbeat advances
// heartbeat_seq. A different owner may take over only after its own Claim
// calls have seen the same (owner, token, heartbeat_seq) for at least ttl on
// the claimant's monotonic clock; the takeover is then a compare-and-set on
// that triple, so a heartbeat that lands first defeats it. Every authority
// check (Heartbeat, Assert, Release) is a comparison of owner and fencing
// token in the database. Fencing tokens come from a per-lease sequence row
// and strictly increase across release and takeover.
//
// The host clock (HostNow, possibly skewed) is only recorded, together with
// what a host-clock design would have decided.
type PgOwnershipStore struct {
	Pool    *pgxpool.Pool
	Actor   string
	HostNow func() time.Time
	// FailHeartbeat injects a store failure on every Heartbeat (step 2).
	FailHeartbeat bool

	mu       sync.Mutex
	observed map[string]ownershipView
}

var _ resource.SupervisorOwnershipStore = (*PgOwnershipStore)(nil)

func (s *PgOwnershipStore) hostNow() time.Time {
	if s.HostNow != nil {
		return s.HostNow()
	}
	return time.Now()
}

func (s *PgOwnershipStore) event(ctx context.Context, q execer, leaseID, op string, accepted bool, token, curToken int64, curOwner string, naive *bool, msg string) error {
	_, err := q.Exec(ctx, `INSERT INTO orhc.ownership_event(lease_id,actor,op,accepted,fencing_token,current_owner,current_token,host_time,naive_host_clock_takeover,message)
		VALUES ($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10)`, leaseID, s.Actor, op, accepted, token, curOwner, curToken, s.hostNow(), naive, msg)
	return err
}

func ownershipOut(leaseID, owner string, token int64, ttl time.Duration, hostNow time.Time) resource.SupervisorOwnership {
	now := hostNow.UTC()
	return resource.SupervisorOwnership{LeaseID: leaseID, OwnerID: owner, FencingToken: uint64(token),
		ClaimedAt: now.Format(time.RFC3339Nano), HeartbeatAt: now.Format(time.RFC3339Nano), OwnerExpiresAt: now.Add(ttl).Format(time.RFC3339Nano)}
}

// Claim takes a free ownership, renews the caller's own ownership with the
// same token, or takes over an ownership whose heartbeat the caller has seen
// stand still for ttl.
func (s *PgOwnershipStore) Claim(ctx context.Context, leaseID, owner string, ttl time.Duration) (resource.SupervisorOwnership, bool, error) {
	if leaseID == "" || owner == "" || ttl <= 0 {
		return resource.SupervisorOwnership{}, false, fmt.Errorf("lease id, owner and positive ttl are required")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO orhc.ownership_seq VALUES ($1,0) ON CONFLICT DO NOTHING`, leaseID); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	var last int64
	if err := tx.QueryRow(ctx, `SELECT last_token FROM orhc.ownership_seq WHERE lease_id=$1 FOR UPDATE`, leaseID).Scan(&last); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	var cur ownershipView
	var curHostExp time.Time
	err = tx.QueryRow(ctx, `SELECT owner_id, fencing_token, heartbeat_seq, owner_host_expires_at FROM orhc.supervisor_ownership WHERE lease_id=$1 FOR UPDATE`, leaseID).
		Scan(&cur.owner, &cur.token, &cur.seq, &curHostExp)
	exists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return resource.SupervisorOwnership{}, false, err
	}
	hostNow := s.hostNow()
	if exists && cur.owner == owner {
		if _, err := tx.Exec(ctx, `UPDATE orhc.supervisor_ownership SET heartbeat_seq=heartbeat_seq+1, heartbeat_at=clock_timestamp(), owner_host_expires_at=$2 WHERE lease_id=$1`, leaseID, hostNow.Add(ttl)); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		if err := s.event(ctx, tx, leaseID, "claim-renew", true, cur.token, cur.token, cur.owner, nil, ""); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		return ownershipOut(leaseID, owner, cur.token, ttl, hostNow), false, tx.Commit(ctx)
	}
	if exists {
		naive := hostNow.After(curHostExp)
		s.mu.Lock()
		if s.observed == nil {
			s.observed = map[string]ownershipView{}
		}
		prev, seen := s.observed[leaseID]
		stood := seen && prev.owner == cur.owner && prev.token == cur.token && prev.seq == cur.seq
		if !stood {
			cur.since = time.Now()
			s.observed[leaseID] = cur
		}
		elapsed := time.Duration(0)
		if stood {
			elapsed = time.Since(prev.since)
		}
		s.mu.Unlock()
		if !stood || elapsed < ttl {
			msg := fmt.Sprintf("heartbeat %d unchanged for %s of %s on the claimant's monotonic clock", cur.seq, elapsed.Round(time.Millisecond), ttl)
			if err := s.event(ctx, tx, leaseID, "claim-refused", false, cur.token, cur.token, cur.owner, &naive, msg); err != nil {
				return resource.SupervisorOwnership{}, false, err
			}
			if err := tx.Commit(ctx); err != nil {
				return resource.SupervisorOwnership{}, false, err
			}
			return resource.SupervisorOwnership{}, false, &LeaseHeldError{Owner: cur.owner, Token: uint64(cur.token), NaiveHostClockTakeover: naive}
		}
		token := max(last, cur.token) + 1
		if _, err := tx.Exec(ctx, `UPDATE orhc.supervisor_ownership SET owner_id=$2, fencing_token=$3, heartbeat_seq=0, claimed_at=clock_timestamp(), heartbeat_at=clock_timestamp(), owner_host_expires_at=$4
			WHERE lease_id=$1 AND owner_id=$5 AND fencing_token=$6 AND heartbeat_seq=$7`, leaseID, owner, token, hostNow.Add(ttl), cur.owner, cur.token, cur.seq); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		if _, err := tx.Exec(ctx, `UPDATE orhc.ownership_seq SET last_token=$2 WHERE lease_id=$1`, leaseID, token); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		msg := fmt.Sprintf("previous %s token %d heartbeat %d stood still for %s (claimant monotonic clock)", cur.owner, cur.token, cur.seq, elapsed.Round(time.Millisecond))
		if err := s.event(ctx, tx, leaseID, "takeover", true, token, cur.token, cur.owner, &naive, msg); err != nil {
			return resource.SupervisorOwnership{}, false, err
		}
		s.mu.Lock()
		delete(s.observed, leaseID)
		s.mu.Unlock()
		return ownershipOut(leaseID, owner, token, ttl, hostNow), true, tx.Commit(ctx)
	}
	token := last + 1
	if _, err := tx.Exec(ctx, `INSERT INTO orhc.supervisor_ownership(lease_id,owner_id,fencing_token,owner_host_expires_at) VALUES ($1,$2,$3,$4)`, leaseID, owner, token, hostNow.Add(ttl)); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	if _, err := tx.Exec(ctx, `UPDATE orhc.ownership_seq SET last_token=$2 WHERE lease_id=$1`, leaseID, token); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	if err := s.event(ctx, tx, leaseID, "claimed", true, token, 0, "", nil, ""); err != nil {
		return resource.SupervisorOwnership{}, false, err
	}
	return ownershipOut(leaseID, owner, token, ttl, hostNow), false, tx.Commit(ctx)
}

func (s *PgOwnershipStore) rejected(ctx context.Context, o resource.SupervisorOwnership, op string) error {
	var curOwner string
	var curToken int64
	_ = s.Pool.QueryRow(ctx, `SELECT owner_id, fencing_token FROM orhc.supervisor_ownership WHERE lease_id=$1`, o.LeaseID).Scan(&curOwner, &curToken)
	_ = s.event(ctx, s.Pool, o.LeaseID, op, false, int64(o.FencingToken), curToken, curOwner, nil, "")
	return fmt.Errorf("%w: %s by %s token %d; current %s token %d", ErrFenced, op, o.OwnerID, o.FencingToken, curOwner, curToken)
}

// Heartbeat advances the heartbeat of the same owner and token.
func (s *PgOwnershipStore) Heartbeat(ctx context.Context, o resource.SupervisorOwnership, ttl time.Duration) (resource.SupervisorOwnership, error) {
	if s.FailHeartbeat {
		_ = s.event(ctx, s.Pool, o.LeaseID, "heartbeat-injected-failure", false, int64(o.FencingToken), 0, "", nil, "injected store failure")
		return resource.SupervisorOwnership{}, fmt.Errorf("injected ownership store failure")
	}
	hostNow := s.hostNow()
	tag, err := s.Pool.Exec(ctx, `UPDATE orhc.supervisor_ownership SET heartbeat_seq=heartbeat_seq+1, heartbeat_at=clock_timestamp(), owner_host_expires_at=$4
		WHERE lease_id=$1 AND owner_id=$2 AND fencing_token=$3`, o.LeaseID, o.OwnerID, int64(o.FencingToken), hostNow.Add(ttl))
	if err != nil {
		return resource.SupervisorOwnership{}, err
	}
	if tag.RowsAffected() == 0 {
		return resource.SupervisorOwnership{}, s.rejected(ctx, o, "heartbeat")
	}
	out := o
	out.HeartbeatAt = hostNow.UTC().Format(time.RFC3339Nano)
	out.OwnerExpiresAt = hostNow.Add(ttl).UTC().Format(time.RFC3339Nano)
	return out, nil
}

// Assert checks that the ownership is still the current one.
func (s *PgOwnershipStore) Assert(ctx context.Context, o resource.SupervisorOwnership) error {
	var n int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM orhc.supervisor_ownership WHERE lease_id=$1 AND owner_id=$2 AND fencing_token=$3`, o.LeaseID, o.OwnerID, int64(o.FencingToken)).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return s.rejected(ctx, o, "assert")
	}
	return nil
}

// Release drops the ownership if it is still this owner's. The token
// sequence is kept, so the next claim gets a larger token.
func (s *PgOwnershipStore) Release(ctx context.Context, o resource.SupervisorOwnership) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM orhc.supervisor_ownership WHERE lease_id=$1 AND owner_id=$2 AND fencing_token=$3`, o.LeaseID, o.OwnerID, int64(o.FencingToken))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.rejected(ctx, o, "release")
	}
	return s.event(ctx, s.Pool, o.LeaseID, "released", true, int64(o.FencingToken), 0, "", nil, "")
}
