package orderrollouthost

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/permission"
	"tei/internal/resource"
)

type localLease struct {
	req      resource.AcquireRequest
	lease    resource.Lease
	issued   time.Time // host clock at acquisition, without monotonic reading
	acquired time.Time // monotonic reading
}

// PgBroker is a resource broker whose lease authority (identity, revision,
// fencing token, state) lives in PostgreSQL, so a takeover host can fence and
// return a lease acquired by another process, and a stale host cannot.
//
// The concrete resource view (opaque secret handle, endpoint, mount source)
// is resolved by the host adapters and kept only in the memory of the
// acquiring process. The table keeps resource.Records, which carry logical
// identities only. A host that did not acquire the lease can fence and
// release it but cannot renew it; releasing it there does not call the
// adapters (the reference adapters hold no external state).
//
// Renewal is not time-gated by the broker; the maximum lifetime is measured
// on the acquiring process's monotonic clock. ExpiresAt is in host-clock
// terms because the product's lease supervisor schedules renewal from it; no
// other host reads it to decide anything. A renewed expiry is the issue time
// plus the monotonic time elapsed since the acquisition plus the TTL, so it
// always advances (the product rejects a renewal whose expiry does not
// advance, which a backward step of the wall clock would otherwise cause).
type PgBroker struct {
	Pool     *pgxpool.Pool
	Actor    string
	Adapters []resource.Adapter
	TTL      time.Duration
	Now      func() time.Time
	// FailRenew, when it returns true, injects a broker failure on Renew
	// (step 2).
	FailRenew func() bool

	mu    sync.Mutex
	local map[string]localLease
}

var _ resource.FenceableBroker = (*PgBroker)(nil)
var _ resource.RenewableBroker = (*PgBroker)(nil)

// ErrLeaseAuthority is returned when the lease authority refuses an operation.
var ErrLeaseAuthority = errors.New("resource lease authority refused")

func (b *PgBroker) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

func (b *PgBroker) adapter(k permission.Kind) (resource.Adapter, error) {
	for _, a := range b.Adapters {
		if a != nil && a.Kind() == k {
			return a, nil
		}
	}
	return nil, fmt.Errorf("host has no resource adapter for kind %s", k)
}

func (b *PgBroker) event(ctx context.Context, q execer, leaseID, op string, accepted bool, revision int, token, current int64, msg string) {
	_, _ = q.Exec(ctx, `INSERT INTO orhc.lease_event(lease_id,actor,op,accepted,revision,fencing_token,current_token,message) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		leaseID, b.Actor, op, accepted, revision, token, current, msg)
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

// Acquire resolves every permission through the host adapters and records
// the lease authority.
func (b *PgBroker) Acquire(ctx context.Context, req resource.AcquireRequest) (resource.Lease, error) {
	req = resource.NormalizeAcquireRequest(req)
	bindings, err := resource.MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return resource.Lease{}, err
	}
	byKey := map[string]resource.Binding{}
	for _, v := range bindings {
		byKey[v.Key()] = v
	}
	policy := req.LeasePolicy.WithDefaults()
	if err := policy.Validate(); err != nil {
		return resource.Lease{}, err
	}
	id := "lease-" + randomHex(8)
	issued := b.now()
	expires := issued.Add(b.TTL)
	if d := issued.Add(policy.MaxLifetimeDuration()); expires.After(d) {
		expires = d
	}
	l := resource.Lease{ID: id, Revision: 1, IssuedAt: issued.Format(time.RFC3339Nano), ExpiresAt: expires.Format(time.RFC3339Nano), RuntimePolicyFingerprint: req.RuntimePolicyFingerprint}
	for _, p := range req.Permissions {
		a, err := b.adapter(p.Kind)
		if err != nil {
			return resource.Lease{}, err
		}
		v, err := a.Acquire(ctx, resource.AdapterRequest{Environment: req.Environment, CapabilityKind: req.CapabilityKind, CapabilityID: req.CapabilityID, Permission: p, Binding: byKey[string(p.Kind)+"|"+p.Scope], LeaseID: id, Revision: 1})
		if err != nil {
			return resource.Lease{}, fmt.Errorf("resource adapter %s acquire failed for %s: %w", p.Kind, p.Scope, err)
		}
		l.Resources = append(l.Resources, v)
	}
	records, _ := json.Marshal(resource.Records(l.Resources))
	if _, err := b.Pool.Exec(ctx, `INSERT INTO orhc.resource_lease(lease_id,revision,state,request_fingerprint,runtime_policy_fingerprint,records,issued_at,expires_at,acquired_by)
		VALUES ($1,1,'active',$2,$3,$4,$5,$6,$7)`, id, resource.AcquireRequestFingerprint(req), req.RuntimePolicyFingerprint, records, l.IssuedAt, l.ExpiresAt, b.Actor); err != nil {
		return resource.Lease{}, err
	}
	b.event(ctx, b.Pool, id, "acquire", true, 1, 0, 0, "")
	b.mu.Lock()
	if b.local == nil {
		b.local = map[string]localLease{}
	}
	b.local[id] = localLease{req: req, lease: l, issued: issued, acquired: time.Now()}
	b.mu.Unlock()
	return l, nil
}

type leaseRow struct {
	revision int
	token    int64
	state    string
	reqFP    string
	issued   string
	expires  string
	policyFP string
}

func lockLease(ctx context.Context, tx pgx.Tx, id string) (leaseRow, bool, error) {
	var r leaseRow
	err := tx.QueryRow(ctx, `SELECT revision, fencing_token, state, request_fingerprint, issued_at, expires_at, runtime_policy_fingerprint FROM orhc.resource_lease WHERE lease_id=$1 FOR UPDATE`, id).
		Scan(&r.revision, &r.token, &r.state, &r.reqFP, &r.issued, &r.expires, &r.policyFP)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	return r, err == nil, err
}

// refuse records a refused operation in its own statement and returns the
// refusal. The caller's transaction is rolled back.
func (b *PgBroker) refuse(ctx context.Context, id, op string, revision int, token, current int64, why string) error {
	b.event(ctx, b.Pool, id, op, false, revision, token, current, why)
	return fmt.Errorf("%w: %s of %s by %s (token %d, current %d): %s", ErrLeaseAuthority, op, id, b.Actor, token, current, why)
}

// Renew extends the lease in place. Only the acquiring process can renew,
// and only with the current revision and fencing token.
func (b *PgBroker) Renew(ctx context.Context, req resource.AcquireRequest, prev resource.Lease) (resource.Lease, error) {
	if b.FailRenew != nil && b.FailRenew() {
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), 0, "injected broker failure")
	}
	req = resource.NormalizeAcquireRequest(req)
	tx, err := b.Pool.Begin(ctx)
	if err != nil {
		return resource.Lease{}, err
	}
	defer tx.Rollback(ctx)
	row, ok, err := lockLease(ctx, tx, prev.ID)
	if err != nil {
		return resource.Lease{}, err
	}
	switch {
	case !ok || row.state != "active":
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), row.token, "lease is not active")
	case row.token != 0 && int64(prev.FencingToken) != row.token:
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), row.token, "stale fencing token")
	case prev.Revision != row.revision:
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), row.token, "stale revision")
	case row.reqFP != resource.AcquireRequestFingerprint(req):
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), row.token, "request differs from the original")
	}
	b.mu.Lock()
	loc, isLocal := b.local[prev.ID]
	b.mu.Unlock()
	if !isLocal {
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), row.token, "lease was acquired by another process")
	}
	policy := req.LeasePolicy.WithDefaults()
	if time.Since(loc.acquired) >= policy.MaxLifetimeDuration() {
		return resource.Lease{}, b.refuse(ctx, prev.ID, "renew", prev.Revision, int64(prev.FencingToken), row.token, "max lifetime reached")
	}
	bindings, err := resource.MatchBindings(req.Permissions, req.Bindings)
	if err != nil {
		return resource.Lease{}, err
	}
	byKey := map[string]resource.Binding{}
	for _, v := range bindings {
		byKey[v.Key()] = v
	}
	old := map[string]resource.Resolved{}
	for _, v := range loc.lease.Resources {
		old[string(v.Kind)+"|"+v.Scope] = v
	}
	next := resource.Lease{ID: prev.ID, Revision: row.revision + 1, FencingToken: uint64(row.token), IssuedAt: row.issued, RuntimePolicyFingerprint: row.policyFP}
	expires := loc.issued.Add(time.Since(loc.acquired)).Add(b.TTL)
	if d := loc.issued.Add(policy.MaxLifetimeDuration()); expires.After(d) {
		expires = d
	}
	next.ExpiresAt = expires.Format(time.RFC3339Nano)
	for _, p := range req.Permissions {
		a, err := b.adapter(p.Kind)
		if err != nil {
			return resource.Lease{}, err
		}
		v, err := a.Renew(ctx, resource.AdapterRequest{Environment: req.Environment, CapabilityKind: req.CapabilityKind, CapabilityID: req.CapabilityID, Permission: p, Binding: byKey[string(p.Kind)+"|"+p.Scope], LeaseID: prev.ID, Revision: next.Revision}, old[string(p.Kind)+"|"+p.Scope])
		if err != nil {
			return resource.Lease{}, fmt.Errorf("resource adapter %s renew failed for %s: %w", p.Kind, p.Scope, err)
		}
		next.Resources = append(next.Resources, v)
	}
	if err := resource.ValidateInPlaceRenewal(req, loc.lease, next); err != nil {
		return resource.Lease{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE orhc.resource_lease SET revision=$2, expires_at=$3 WHERE lease_id=$1`, prev.ID, next.Revision, next.ExpiresAt); err != nil {
		return resource.Lease{}, err
	}
	b.event(ctx, tx, prev.ID, "renew", true, next.Revision, row.token, row.token, "")
	if err := tx.Commit(ctx); err != nil {
		return resource.Lease{}, err
	}
	b.mu.Lock()
	loc.lease = next
	b.local[prev.ID] = loc
	b.mu.Unlock()
	return next, nil
}

// Fence installs a newer supervisor fencing token. It never moves backwards.
func (b *PgBroker) Fence(ctx context.Context, l resource.Lease, token uint64) (resource.Lease, error) {
	if token == 0 {
		return resource.Lease{}, fmt.Errorf("fencing token must be positive")
	}
	tx, err := b.Pool.Begin(ctx)
	if err != nil {
		return resource.Lease{}, err
	}
	defer tx.Rollback(ctx)
	row, ok, err := lockLease(ctx, tx, l.ID)
	if err != nil {
		return resource.Lease{}, err
	}
	switch {
	case !ok || row.state != "active":
		return resource.Lease{}, b.refuse(ctx, l.ID, "fence", l.Revision, int64(token), row.token, "lease is not active")
	case l.Revision != 0 && l.Revision != row.revision:
		return resource.Lease{}, b.refuse(ctx, l.ID, "fence", l.Revision, int64(token), row.token, "stale revision")
	case int64(token) < row.token:
		return resource.Lease{}, b.refuse(ctx, l.ID, "fence", l.Revision, int64(token), row.token, "fencing token would move backwards")
	}
	out := resource.Lease{ID: l.ID, Revision: row.revision, FencingToken: token, IssuedAt: row.issued, ExpiresAt: row.expires, RuntimePolicyFingerprint: row.policyFP}
	b.mu.Lock()
	loc, isLocal := b.local[l.ID]
	b.mu.Unlock()
	if isLocal {
		out.Resources = loc.lease.Resources
	}
	if int64(token) > row.token {
		if _, err := tx.Exec(ctx, `UPDATE orhc.resource_lease SET fencing_token=$2 WHERE lease_id=$1`, l.ID, int64(token)); err != nil {
			return resource.Lease{}, err
		}
		b.event(ctx, tx, l.ID, "fence", true, row.revision, int64(token), row.token, "")
	}
	if err := tx.Commit(ctx); err != nil {
		return resource.Lease{}, err
	}
	if isLocal {
		loc.lease = out
		b.mu.Lock()
		b.local[l.ID] = loc
		b.mu.Unlock()
	}
	return out, nil
}

// Release returns the lease. Only the holder of the current fencing token
// (and revision, when given) can return it.
func (b *PgBroker) Release(ctx context.Context, l resource.Lease) error {
	tx, err := b.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	row, ok, err := lockLease(ctx, tx, l.ID)
	if err != nil {
		return err
	}
	switch {
	case !ok || row.state != "active":
		return b.refuse(ctx, l.ID, "release", l.Revision, int64(l.FencingToken), row.token, "lease is not active")
	case row.token != 0 && int64(l.FencingToken) != row.token:
		return b.refuse(ctx, l.ID, "release", l.Revision, int64(l.FencingToken), row.token, "stale fencing token")
	case l.Revision != 0 && l.Revision != row.revision:
		return b.refuse(ctx, l.ID, "release", l.Revision, int64(l.FencingToken), row.token, "stale revision")
	}
	b.mu.Lock()
	loc, isLocal := b.local[l.ID]
	b.mu.Unlock()
	if isLocal {
		bindings, err := resource.MatchBindings(loc.req.Permissions, loc.req.Bindings)
		if err != nil {
			return err
		}
		byKey := map[string]resource.Binding{}
		for _, v := range bindings {
			byKey[v.Key()] = v
		}
		for _, v := range loc.lease.Resources {
			a, err := b.adapter(v.Kind)
			if err != nil {
				return err
			}
			p := permission.Spec{Kind: v.Kind, Access: v.Access, Scope: v.Scope}
			if err := a.Release(ctx, resource.AdapterRequest{Environment: loc.req.Environment, CapabilityKind: loc.req.CapabilityKind, CapabilityID: loc.req.CapabilityID, Permission: p, Binding: byKey[string(v.Kind)+"|"+v.Scope], LeaseID: l.ID, Revision: row.revision}, v); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE orhc.resource_lease SET state='released', released_by=$2 WHERE lease_id=$1`, l.ID, b.Actor); err != nil {
		return err
	}
	b.event(ctx, tx, l.ID, "release", true, row.revision, int64(l.FencingToken), row.token, "")
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	delete(b.local, l.ID)
	b.mu.Unlock()
	return nil
}

// LeaseAuthority reads the persisted authority of a lease.
func LeaseAuthority(ctx context.Context, pool *pgxpool.Pool, id string) (revision int, token int64, state string, err error) {
	err = pool.QueryRow(ctx, `SELECT revision, fencing_token, state FROM orhc.resource_lease WHERE lease_id=$1`, id).Scan(&revision, &token, &state)
	return
}
