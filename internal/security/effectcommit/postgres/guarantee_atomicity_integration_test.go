package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"tei/internal/security/effectcommit"
)

// Real PostgreSQL tests that back the provisional D10 declaration: a Runner
// built through effectcommit.NewRequiredRunner(AtomicWithCommit) on a
// NewWithTrustedOwnerRoutine Provider keeps authority check, owner effects and
// receipt in one commit under lost acknowledgement and concurrency. They SKIP
// without TEI_EFFECT_POSTGRES_DSN, and a skip is not evidence. Passing them
// does not prove the declaration for a real deployment (see README.md).

type collectingSink struct {
	mu  sync.Mutex
	got []effectcommit.Evidence
}

func (c *collectingSink) Record(e effectcommit.Evidence) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.got = append(c.got, e)
}

func (c *collectingSink) all() []effectcommit.Evidence {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]effectcommit.Evidence(nil), c.got...)
}

func requiredRunner(t *testing.T, p effectcommit.Provider, sink effectcommit.EvidenceSink) effectcommit.Runner {
	t.Helper()
	r, err := effectcommit.NewRequiredRunner(p, effectcommit.GuaranteeAtomicWithCommit,
		effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceRealPG})
	if err != nil {
		t.Fatalf("declared provider refused: %v", err)
	}
	return r
}

// declaredUncertain loses the commit acknowledgement AND states its own level
// explicitly (a wrapper must not rely on promotion), copying the inner
// declaration so the Required Runner accepts it.
type declaredUncertain struct{ uncertainProvider }

func (d declaredUncertain) Guarantee() effectcommit.Guarantee {
	return effectcommit.DeclaredGuarantee(d.inner)
}

func assertEvidenceDeclaresProvisional(t *testing.T, evs []effectcommit.Evidence) {
	t.Helper()
	if len(evs) == 0 {
		t.Fatal("no evidence recorded")
	}
	for i, ev := range evs {
		if ev.Guarantee != effectcommit.GuaranteeAtomicWithCommit || ev.GuaranteeReason != ReasonProvisionalD10 ||
			ev.Provenance != effectcommit.ProvenanceRealPG {
			t.Fatalf("evidence[%d] guarantee=%q reason=%q provenance=%q", i, ev.Guarantee, ev.GuaranteeReason, ev.Provenance)
		}
	}
}

func TestDeclaredProviderRequiredRunnerAckLossThenReplay(t *testing.T) {
	f := openFixture(t)
	installTrustedOwnerFixture(t, f)
	ctx := context.Background()
	req := f.req
	req.Payload = []byte("3")
	grant := f.grant(t)
	owner, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	// The plain ack-loss wrapper declares nothing, so a Required Runner refuses it.
	if _, err := effectcommit.NewRequiredRunner(uncertainProvider{inner: owner}, effectcommit.GuaranteeAtomicWithCommit,
		effectcommit.AuditConfig{}); !errors.Is(err, effectcommit.ErrGuaranteeUndeclared) {
		t.Fatalf("undeclaring wrapper was not refused: %v", err)
	}

	sink := &collectingSink{}
	lossy := requiredRunner(t, declaredUncertain{uncertainProvider{inner: owner}}, sink)
	got, err := lossy.Execute(ctx, req, grant)
	var uncertain *effectcommit.CommitUncertainError
	if got != (effectcommit.Result{}) || !errors.As(err, &uncertain) {
		t.Fatalf("lost acknowledgement was not uncertain: %+v err=%v", got, err)
	}
	a, reservations, history, outbox, receipts := ownerCounts(t, f)
	value, version := f.readObject(t)
	if a != 0 || reservations != 1 || history != 1 || outbox != 1 || receipts != 1 || value != "3" || version != 1 {
		t.Fatalf("effects and receipt not committed together: %d %d %d %d %d %q %d", a, reservations, history, outbox, receipts, value, version)
	}

	retry, err := requiredRunner(t, owner, sink).Execute(ctx, req, grant)
	if err != nil || retry.Status != effectcommit.StatusReplayed || retry.Receipt.ResourceVersion != 1 {
		t.Fatalf("retry after ack loss: %+v err=%v", retry, err)
	}
	a, reservations, history, outbox, receipts = ownerCounts(t, f)
	if a != 0 || reservations != 1 || history != 1 || outbox != 1 || receipts != 1 {
		t.Fatalf("retry duplicated effects: %d %d %d %d %d", a, reservations, history, outbox, receipts)
	}
	evs := sink.all()
	if len(evs) != 2 || evs[0].ErrorClass != effectcommit.ErrorClassCommitUncertain || evs[1].Status != effectcommit.StatusReplayed {
		t.Fatalf("evidence=%+v", evs)
	}
	assertEvidenceDeclaresProvisional(t, evs)
}

func TestDeclaredProviderRequiredRunnerConcurrentSameOperationCommitsOnce(t *testing.T) {
	f := openFixture(t)
	installTrustedOwnerFixture(t, f)
	ctx := context.Background()
	req := f.req
	req.Payload = []byte("3")
	grant := f.grant(t)
	owner, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	sink := &collectingSink{}
	runner := requiredRunner(t, owner, sink)
	const workers = 6
	type attempt struct {
		res effectcommit.Result
		err error
	}
	out := make(chan attempt, workers)
	begin := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-begin
			r, err := runner.Execute(ctx, req, grant)
			out <- attempt{r, err}
		}()
	}
	close(begin)
	wg.Wait()
	close(out)
	committed, replayed := 0, 0
	for x := range out {
		if x.err != nil {
			t.Fatalf("concurrent execute error: %v", x.err)
		}
		switch x.res.Status {
		case effectcommit.StatusCommitted:
			committed++
		case effectcommit.StatusReplayed:
			replayed++
		default:
			t.Fatalf("unexpected status: %+v", x.res)
		}
	}
	if committed != 1 || replayed != workers-1 {
		t.Fatalf("committed=%d replayed=%d", committed, replayed)
	}
	a, reservations, history, outbox, receipts := ownerCounts(t, f)
	if a != 0 || reservations != 1 || history != 1 || outbox != 1 || receipts != 1 {
		t.Fatalf("duplicate owner effects: %d %d %d %d %d", a, reservations, history, outbox, receipts)
	}
	assertEvidenceDeclaresProvisional(t, sink.all())
}

func TestDeclaredProviderRequiredRunnerConcurrentRevokeIsAllOrNothing(t *testing.T) {
	for iteration := 0; iteration < 8; iteration++ {
		f := openFixture(t)
		installTrustedOwnerFixture(t, f)
		ctx := context.Background()
		req := f.req
		req.Payload = []byte("3")
		grant := f.grant(t)
		owner, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
		if err != nil {
			t.Fatal(err)
		}
		runner := requiredRunner(t, owner, nil)
		begin := make(chan struct{})
		var wg sync.WaitGroup
		var result effectcommit.Result
		var writeErr, revokeErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-begin
			result, writeErr = runner.Execute(ctx, req, grant)
		}()
		go func() {
			defer wg.Done()
			<-begin
			_, revokeErr = f.pool.Exec(ctx, "UPDATE "+f.schema+".subjects SET role='VIEWER' WHERE subject=$1", req.Scope.Subject)
		}()
		close(begin)
		wg.Wait()
		if writeErr != nil || revokeErr != nil {
			t.Fatalf("concurrent database failure: %v / %v", writeErr, revokeErr)
		}
		a, reservations, history, outbox, receipts := ownerCounts(t, f)
		_, version := f.readObject(t)
		switch result.Status {
		case effectcommit.StatusCommitted:
			if a != 0 || reservations != 1 || history != 1 || outbox != 1 || receipts != 1 || version != 1 {
				t.Fatalf("committed but partial: %d %d %d %d %d v%d", a, reservations, history, outbox, receipts, version)
			}
		case effectcommit.StatusDenied:
			if a != 3 || reservations != 0 || history != 0 || outbox != 0 || receipts != 0 || version != 0 {
				t.Fatalf("denied but effects remain: %d %d %d %d %d v%d", a, reservations, history, outbox, receipts, version)
			}
		default:
			t.Fatalf("unexpected status: %+v", result)
		}
	}
}

// TestDeclaredProviderRequiredRunnerBlocksOnUncommittedRevocation is the
// deterministic check of the authority row lock. Another transaction revokes
// (UPDATE, uncommitted) and holds the authority row; the declared Runner's
// CheckAuthority must wait on that row lock, and after the revocation commits it
// must see the changed version and Deny with no effect and no receipt. If the
// lock is not taken (FOR UPDATE removed, or CheckAuthority not calling the lock
// function) Execute returns while the revocation is still uncommitted.
func TestDeclaredProviderRequiredRunnerBlocksOnUncommittedRevocation(t *testing.T) {
	f := openFixture(t)
	installTrustedOwnerFixture(t, f)
	ctx := context.Background()
	req := f.req
	req.Payload = []byte("3")
	grant := f.grant(t)
	owner, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	runner := requiredRunner(t, owner, nil)

	revoker, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = revoker.Rollback(context.Background()) }()
	if _, err := revoker.Exec(ctx, "UPDATE "+f.schema+".subjects SET role='VIEWER' WHERE subject=$1", req.Scope.Subject); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		res effectcommit.Result
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		r, err := runner.Execute(ctx, req, grant)
		done <- outcome{r, err}
	}()
	select {
	case o := <-done:
		t.Fatalf("Execute did not wait for the uncommitted revocation (authority row not locked): %+v err=%v", o.res, o.err)
	case <-time.After(500 * time.Millisecond):
	}
	if err := revoker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case o := <-done:
		if o.err != nil || o.res.Status != effectcommit.StatusDenied || o.res.Receipt != (effectcommit.Receipt{}) {
			t.Fatalf("after revocation commit: %+v err=%v", o.res, o.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Execute stayed blocked after the revocation committed")
	}
	a, reservations, history, outbox, receipts := ownerCounts(t, f)
	value, version := f.readObject(t)
	if a != 3 || reservations != 0 || history != 0 || outbox != 0 || receipts != 0 || value != "original" || version != 0 {
		t.Fatalf("denied but state changed: %d %d %d %d %d %q %d", a, reservations, history, outbox, receipts, value, version)
	}
}

// failBeforeCommit rolls back and returns an error instead of committing, and
// declares its own level explicitly. Authority check, owner effect and receipt
// have all been executed on the transaction by then; none may survive. This is
// what detects a receipt INSERT or an owner effect that ran outside the commit.
type failBeforeCommit struct{ inner Provider }

func (failBeforeCommit) Guarantee() effectcommit.Guarantee {
	return effectcommit.Guarantee{Level: effectcommit.GuaranteeAtomicWithCommit, Reason: "test wrapper"}
}

func (w failBeforeCommit) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	tx, err := w.inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return failBeforeCommitTx{Transaction: tx}, nil
}

type failBeforeCommitTx struct{ effectcommit.Transaction }

func (t failBeforeCommitTx) Commit(context.Context) error {
	_ = t.Transaction.Rollback()
	return errors.New("commit not reached")
}

func TestDeclaredProviderRequiredRunnerPreCommitFailureLeavesNoReceiptNorOwnerEffect(t *testing.T) {
	f := openFixture(t)
	installTrustedOwnerFixture(t, f)
	ctx := context.Background()
	req := f.req
	req.Payload = []byte("3")
	grant := f.grant(t)
	owner, err := NewWithTrustedOwnerRoutine(f.pool, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	runner := requiredRunner(t, failBeforeCommit{inner: owner}, nil)
	got, err := runner.Execute(ctx, req, grant)
	if err == nil || got.Status == effectcommit.StatusCommitted {
		t.Fatalf("pre-commit failure reported as success: %+v err=%v", got, err)
	}
	a, reservations, history, outbox, receipts := ownerCounts(t, f)
	value, version := f.readObject(t)
	if a != 3 || reservations != 0 || history != 0 || outbox != 0 || receipts != 0 || value != "original" || version != 0 {
		t.Fatalf("effect or receipt survived without commit: stock=%d reservations=%d history=%d outbox=%d receipts=%d object=%q v%d",
			a, reservations, history, outbox, receipts, value, version)
	}
}
