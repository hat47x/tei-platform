package orderwebmember

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Grant is the authorization of one principal as read from the DB in one
// statement. Seq is the value of owdm.clock in the same snapshot: rows created
// after the grant was read have created_seq > Seq and are never visible with
// this grant, whatever scope it still lists.
type Grant struct {
	PrincipalID string
	Kind        string
	Active      bool
	Version     int64
	Accounts    []int64
	Seq         int64
	ReadAt      time.Time // DB clock of the read
	readStart   time.Time // local, before the read was issued
}

var (
	ErrUnauthenticated = errors.New("unknown session")
	ErrRevoked         = errors.New("principal revoked")
)

// AuthzCache keeps grants for at most MaxAge (measured from before the read)
// and drops a principal's grants when NOTIFY owdm_authz names it. Declared
// propagation bound of a revocation or a scope reduction: MaxAge, even when
// notifications are lost.
type AuthzCache struct {
	pool       *pgxpool.Pool
	MaxAge     time.Duration
	DropNotify bool
	now        func() time.Time

	mu      sync.Mutex
	byToken map[string]Grant
	loads   int64
	notices int64
}

func NewAuthzCache(pool *pgxpool.Pool, maxAge time.Duration) *AuthzCache {
	return &AuthzCache{pool: pool, MaxAge: maxAge, now: time.Now, byToken: map[string]Grant{}}
}

// Resolve returns the grant of a session token. A revoked principal yields
// ErrRevoked, an unknown token ErrUnauthenticated.
func (a *AuthzCache) Resolve(ctx context.Context, token string) (Grant, error) {
	a.mu.Lock()
	if g, ok := a.byToken[token]; ok && a.now().Sub(g.readStart) <= a.MaxAge {
		a.mu.Unlock()
		return g.check()
	}
	a.loads++
	a.mu.Unlock()
	start := a.now()
	g := Grant{readStart: start}
	err := a.pool.QueryRow(ctx, `
		SELECT p.principal_id, p.kind, p.active, p.authz_version,
		       COALESCE(array_agg(s.account_id ORDER BY s.account_id) FILTER (WHERE s.account_id IS NOT NULL), '{}'),
		       (SELECT seq FROM owdm.clock WHERE id = 1), clock_timestamp()
		FROM owdm.session ss
		JOIN owdm.principal p ON p.principal_id = ss.principal_id
		LEFT JOIN owdm.principal_scope s ON s.principal_id = p.principal_id
		WHERE ss.token = $1
		GROUP BY p.principal_id, p.kind, p.active, p.authz_version`, token).
		Scan(&g.PrincipalID, &g.Kind, &g.Active, &g.Version, &g.Accounts, &g.Seq, &g.ReadAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Grant{}, ErrUnauthenticated
	}
	if err != nil {
		return Grant{}, err
	}
	a.mu.Lock()
	a.byToken[token] = g
	a.mu.Unlock()
	return g.check()
}

func (g Grant) check() (Grant, error) {
	if !g.Active {
		return g, ErrRevoked
	}
	return g, nil
}

// InvalidatePrincipal drops every cached grant of a principal.
func (a *AuthzCache) InvalidatePrincipal(principal string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for tok, g := range a.byToken {
		if g.PrincipalID == principal {
			delete(a.byToken, tok)
		}
	}
}

func (a *AuthzCache) notice(principal string) {
	a.mu.Lock()
	a.notices++
	drop := a.DropNotify
	a.mu.Unlock()
	if !drop {
		a.InvalidatePrincipal(principal)
	}
}

// Stats returns loads and received notifications.
func (a *AuthzCache) Stats() map[string]int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]int64{"loads": a.loads, "notices": a.notices}
}
