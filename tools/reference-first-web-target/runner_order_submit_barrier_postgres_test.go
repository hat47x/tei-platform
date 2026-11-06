//go:build !tei_repair_only

package main

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/submit"
	"tei/internal/security/legacyauthorization"
)

// Barrier test (real PostgreSQL, TEST-ONLY DDL, synthetic grant, gated by
// TEI_TEST_POSTGRES_DSN; a skip is not evidence).
//
// From the test DDL: lock_authority takes SELECT ... FOR UPDATE on the authority
// row inside the Runner's transaction, and that lock is held until Commit. An
// UPDATE of the same row from another connection therefore must wait. The test
// holds the Runner's transaction right after the real CheckAuthority returned,
// issues the authority change from a second connection, and REQUIRES (t.Fatalf
// otherwise) that the change is seen waiting on a lock and completes only after
// the release event, then checks the end state. Removing FOR UPDATE from the
// fixture therefore fails this test. It covers one hold-after-CheckAuthority
// interleaving on the synthetic fixture and does not claim atomicity.

type barrierProvider struct {
	inner   effectcommit.Provider
	held    chan struct{}
	release chan struct{}
	once    sync.Once
}

type barrierTx struct {
	effectcommit.Transaction
	p *barrierProvider
}

func (p *barrierProvider) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	tx, err := p.inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return barrierTx{tx, p}, nil
}

func (t barrierTx) CheckAuthority(ctx context.Context, s effectcommit.Scope, e legacyauthorization.AuthorityEpochs) (bool, error) {
	ok, err := t.Transaction.CheckAuthority(ctx, s, e)
	if err != nil || !ok {
		return ok, err
	}
	t.p.once.Do(func() { close(t.p.held) })
	select {
	case <-t.p.release:
		return ok, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

type eventLog struct {
	seq    atomic.Int64
	mu     sync.Mutex
	events []string
	at     map[string]int64
}

func (l *eventLog) mark(name string) {
	n := l.seq.Add(1)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.at == nil {
		l.at = map[string]int64{}
	}
	l.at[name] = n
	l.events = append(l.events, name)
}

func (l *eventLog) get(name string) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.at[name]
}

// waitingOnLock reports whether some other backend is waiting on a heavyweight
// lock while running an UPDATE of the authority table.
func waitingOnLock(ctx context.Context, db *sql.DB) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity
WHERE datname = current_database() AND pid <> pg_backend_pid()
  AND wait_event_type = 'Lock' AND query ILIKE 'UPDATE %authority%'`).Scan(&n)
	return n > 0, err
}

func TestRunnerOrderSubmitRealPostgresBarrierAuthorityChangeVersusHeldTransaction(t *testing.T) {
	for name, stmt := range map[string]string{
		"epoch-bump": `UPDATE ` + runnerTestSchema + `.authority SET policy=policy+1`,
		"revoked":    `UPDATE ` + runnerTestSchema + `.authority SET enabled=FALSE`,
	} {
		t.Run(name, func(t *testing.T) {
			env := newRunnerPGEnv(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			log := &eventLog{}

			bp := &barrierProvider{inner: env.provider(t), held: make(chan struct{}), release: make(chan struct{})}
			var releaseOnce sync.Once
			doRelease := func() { releaseOnce.Do(func() { close(bp.release) }) }
			defer doRelease() // never leave the held transaction blocked

			held, err := newRunnerOrderSubmitStore(effectcommit.Runner{Provider: bp},
				func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
					return syntheticSubmitGrant(), nil
				}, nil, &statusReader{})
			if err != nil {
				t.Fatal(err)
			}

			submitDone := make(chan error, 1)
			go func() {
				err := held.Submit(ctx, 1001)
				log.mark("submit-returned")
				submitDone <- err
			}()

			select {
			case <-bp.held:
				log.mark("held-after-check-authority")
			case err := <-submitDone:
				t.Fatalf("submit returned before the hold: %v", err)
			case <-ctx.Done():
				t.Fatal("timeout waiting for the held transaction")
			}

			// Second connection, dedicated, for the authority change.
			conn, err := pgx.Connect(ctx, env.dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close(context.Background())
			changeDone := make(chan error, 1)
			go func() {
				log.mark("change-issued")
				_, err := conn.Exec(ctx, stmt)
				log.mark("change-completed")
				changeDone <- err
			}()

			// Bounded polling (no bare sleep): wait until the change is either
			// visibly blocked on a lock or has already completed.
			blocked := false
			finishedEarly := false
			poll := time.NewTicker(10 * time.Millisecond)
			defer poll.Stop()
			deadline := time.After(5 * time.Second)
		wait:
			for {
				select {
				case err := <-changeDone:
					if err != nil {
						t.Fatalf("change failed: %v", err)
					}
					finishedEarly = true
					changeDone <- nil
					break wait
				case <-poll.C:
					w, err := waitingOnLock(ctx, env.db)
					if err != nil {
						t.Fatal(err)
					}
					if w {
						blocked = true
						log.mark("change-observed-waiting")
						break wait
					}
				case <-deadline:
					t.Fatal("neither a lock wait nor completion was observed in 5s")
				}
			}
			t.Logf("observed interleaving: change blocked behind held transaction=%v, completed before release=%v", blocked, finishedEarly)

			log.mark("release")
			doRelease()
			var submitErr error
			select {
			case submitErr = <-submitDone:
			case <-ctx.Done():
				t.Fatal("timeout waiting for the held submit")
			}
			select {
			case err := <-changeDone:
				if err != nil {
					t.Fatalf("change failed: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for the change")
			}

			// HARD assertions for this fixture: lock_authority holds FOR UPDATE until
			// Commit, so the authority change MUST wait for the held transaction. If it
			// did not wait, the fixture lost its lock (a TOCTOU window) and the test fails.
			if !blocked || finishedEarly {
				t.Fatalf("authority change did not wait for the held transaction (blocked=%v completedBeforeRelease=%v); events=%v", blocked, finishedEarly, log.events)
			}
			if log.get("change-observed-waiting") == 0 || log.get("release") == 0 || log.get("change-completed") == 0 {
				t.Fatalf("missing ordering evidence: events=%v", log.events)
			}
			if log.get("change-completed") < log.get("release") {
				t.Fatalf("change completed before release: %v", log.events)
			}

			// End state that does not depend on the interleaving: the held
			// transaction committed once; the later change took effect afterwards.
			if submitErr != nil {
				t.Fatalf("held submit err=%v events=%v", submitErr, log.events)
			}
			if got := env.snapshot(t); got != (pgSnapshot{"submitted", "draft", 1, 1}) {
				t.Fatalf("snapshot=%+v events=%v", got, log.events)
			}
			fresh, err := newRunnerOrderSubmitStore(effectcommit.Runner{Provider: env.provider(t)},
				func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
					return syntheticSubmitGrant(), nil
				}, nil, &statusReader{})
			if err != nil {
				t.Fatal(err)
			}
			if err := fresh.Submit(ctx, 1002); !errors.Is(err, errOrderNotSubmittable) {
				t.Fatalf("fresh submit of 1002 err=%v", err)
			}
			if got := env.snapshot(t); got != (pgSnapshot{"submitted", "draft", 1, 1}) {
				t.Fatalf("after fresh submit snapshot=%+v", got)
			}
			// After the change a Lookup of the held order is Denied, not a stale Replayed.
			res, lerr := fresh.LookupSubmit(ctx, 1001)
			out := submit.ClassifyLookup(res, lerr)
			if out.Disposition != submit.NotCommitted || out.Status != effectcommit.StatusDenied {
				t.Fatalf("lookup after change=%+v", out)
			}
		})
	}
}
