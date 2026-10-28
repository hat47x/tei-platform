//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
	"tei/internal/workspace"
)

// These tests are DB-free. A fake Provider/Transaction with call counters
// drives the real effectcommit.Runner; they are synthetic and prove nothing
// about a real database.

type fakeSubmitTx struct {
	p           *fakeSubmitProvider
	allowed     bool
	writeErr    error
	commitErr   error
	existing    *effectcommit.Receipt
	checkCalls  int
	writeCalls  int
	saveCalls   int
	commitCalls int
}

type fakeSubmitProvider struct{ tx *fakeSubmitTx }

func newFakeSubmit() *fakeSubmitProvider {
	p := &fakeSubmitProvider{}
	p.tx = &fakeSubmitTx{p: p, allowed: true}
	return p
}
func (p *fakeSubmitProvider) Begin(context.Context) (effectcommit.Transaction, error) {
	return p.tx, nil
}
func (t *fakeSubmitTx) CheckAuthority(context.Context, effectcommit.Scope, legacyauthorization.AuthorityEpochs) (bool, error) {
	t.checkCalls++
	return t.allowed, nil
}
func (t *fakeSubmitTx) FindReceipt(context.Context, string) (effectcommit.Receipt, bool, error) {
	if t.existing != nil {
		return *t.existing, true, nil
	}
	return effectcommit.Receipt{}, false, nil
}
func (t *fakeSubmitTx) WriteResource(context.Context, effectcommit.Request) (uint64, error) {
	t.writeCalls++
	if t.writeErr != nil {
		return 0, t.writeErr
	}
	return 1, nil
}
func (t *fakeSubmitTx) SaveReceipt(context.Context, effectcommit.Receipt) error {
	t.saveCalls++
	return nil
}
func (t *fakeSubmitTx) Commit(context.Context) error { t.commitCalls++; return t.commitErr }
func (t *fakeSubmitTx) Rollback() error              { return nil }

type countingRunner struct {
	inner    submitRunner
	executes int
}

func (c *countingRunner) Execute(ctx context.Context, r effectcommit.Request, g legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	c.executes++
	return c.inner.Execute(ctx, r, g)
}
func (c *countingRunner) Lookup(ctx context.Context, r effectcommit.Request, g legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	return c.inner.Lookup(ctx, r, g)
}

type statusReader struct {
	reads int
	err   error
}

func (s *statusReader) Submit(context.Context, int64) error { return errors.New("unused") }
func (s *statusReader) ReadStatus(context.Context, int64) (string, error) {
	s.reads++
	if s.err != nil {
		return "", s.err
	}
	return "submitted", nil
}

func syntheticSubmitGrant() legacyauthorization.AuthorizationGrant {
	return legacyauthorization.AuthorizationGrant{
		SubjectID: "legacy-order-editor", Action: "execute", Resource: runnerSubmitCapability,
		Epochs: &legacyauthorization.AuthorityEpochs{Source: "test-authority", SubjectLink: 1, Membership: 1, Policy: 1},
	}
}

func newFakeStore(t *testing.T, p *fakeSubmitProvider, grant submitGrantSource, st orderSubmitStore) (*runnerOrderSubmitStore, *countingRunner) {
	t.Helper()
	cr := &countingRunner{inner: effectcommit.Runner{Provider: p}}
	if grant == nil {
		grant = func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
			return syntheticSubmitGrant(), nil
		}
	}
	if st == nil {
		st = &statusReader{}
	}
	s, err := newRunnerOrderSubmitStore(cr, grant, nil, st)
	if err != nil {
		t.Fatal(err)
	}
	return s, cr
}

func TestRunnerOrderSubmitMapsEveryOutcome(t *testing.T) {
	const secret = "postgres://u:p@h/db driver detail"
	for _, tc := range []struct {
		name   string
		result effectcommit.Result
		err    error
		want   error
	}{
		{"committed", effectcommit.Result{Status: effectcommit.StatusCommitted}, nil, nil},
		{"replayed", effectcommit.Result{Status: effectcommit.StatusReplayed}, nil, errOrderNotSubmittable},
		{"denied", effectcommit.Result{Status: effectcommit.StatusDenied}, nil, errOrderNotSubmittable},
		{"conflict", effectcommit.Result{Status: effectcommit.StatusConflict}, nil, errOrderNotSubmittable},
		{"not-recorded", effectcommit.Result{Status: effectcommit.StatusNotRecorded}, nil, errRunnerSubmitFailed},
		{"uncertain", effectcommit.Result{}, &effectcommit.CommitUncertainError{Cause: errors.New(secret)}, errOrderSubmitCommitUnconfirmed},
		{"uncertain-wrapped", effectcommit.Result{}, errors.Join(errors.New("x"), &effectcommit.CommitUncertainError{Cause: errors.New(secret)}), errOrderSubmitCommitUnconfirmed},
		{"invalid", effectcommit.Result{}, effectcommit.ErrInvalidRequest, errRunnerSubmitFailed},
		{"missing-provider", effectcommit.Result{}, effectcommit.ErrMissingProvider, errRunnerSubmitFailed},
		{"arbitrary", effectcommit.Result{}, errors.New(secret), errRunnerSubmitFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mapSubmitOutcome(tc.result, tc.err)
			if !errors.Is(got, tc.want) && got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			if got != nil && (strings.Contains(got.Error(), "postgres://") || strings.Contains(got.Error(), "driver")) {
				t.Fatalf("leak: %v", got)
			}
		})
	}
}

func TestRunnerOrderSubmitDeniedBeforeEffectWritesNothing(t *testing.T) {
	p := newFakeSubmit()
	p.tx.allowed = false
	s, _ := newFakeStore(t, p, nil, nil)
	if err := s.Submit(context.Background(), 1001); !errors.Is(err, errOrderNotSubmittable) {
		t.Fatalf("err=%v", err)
	}
	// The authority check was reached, and nothing was written after it denied.
	if p.tx.checkCalls != 1 || p.tx.writeCalls != 0 || p.tx.saveCalls != 0 {
		t.Fatalf("check=%d write=%d save=%d", p.tx.checkCalls, p.tx.writeCalls, p.tx.saveCalls)
	}
}

func TestRunnerOrderSubmitCommitsThroughRunner(t *testing.T) {
	p := newFakeSubmit()
	s, _ := newFakeStore(t, p, nil, nil)
	if err := s.Submit(context.Background(), 1001); err != nil {
		t.Fatal(err)
	}
	if p.tx.writeCalls != 1 || p.tx.saveCalls != 1 || p.tx.commitCalls != 1 {
		t.Fatalf("%+v", p.tx)
	}
}

func TestRunnerOrderSubmitReplayAndHashConflict(t *testing.T) {
	p := newFakeSubmit()
	s, _ := newFakeStore(t, p, nil, nil)
	grant := syntheticSubmitGrant()
	req, err := runnerSubmitRequest(grant, 1001)
	if err != nil {
		t.Fatal(err)
	}
	// Same request: Replayed, no second write. The stored receipt carries the
	// request hash a first commit would have saved; captureHash derives it with a
	// recording transaction because SaveReceipt's argument is not exposed.
	rec := effectcommit.Receipt{Scope: req.Scope, OperationID: req.OperationID, ResourceVersion: 1}
	rec.RequestHash = captureHash(t, req, grant)
	p.tx.existing = &rec
	if err := s.Submit(context.Background(), 1001); !errors.Is(err, errOrderNotSubmittable) {
		t.Fatalf("replay err=%v", err)
	}
	if p.tx.writeCalls != 0 {
		t.Fatal("replay wrote again")
	}
	// Different payload under the same OperationID: Conflict.
	other := req
	other.Payload = []byte(`{"orderId":1001,"to":"cancelled"}`)
	res, err := (effectcommit.Runner{Provider: p}).Execute(context.Background(), other, grant)
	if err != nil || res.Status != effectcommit.StatusConflict || p.tx.writeCalls != 0 {
		t.Fatalf("res=%+v err=%v writes=%d", res, err, p.tx.writeCalls)
	}
	// LookupSubmit returns Replayed from the receipt.
	look, err := s.LookupSubmit(context.Background(), 1001)
	if err != nil || look.Status != effectcommit.StatusReplayed {
		t.Fatalf("lookup %+v %v", look, err)
	}
}

type hashCapture struct {
	*fakeSubmitTx
	hash string
}

func (h *hashCapture) SaveReceipt(ctx context.Context, r effectcommit.Receipt) error {
	h.hash = r.RequestHash
	return nil
}

type hashProvider struct{ tx *hashCapture }

func (p hashProvider) Begin(context.Context) (effectcommit.Transaction, error) { return p.tx, nil }

func captureHash(t *testing.T, req effectcommit.Request, grant legacyauthorization.AuthorizationGrant) string {
	t.Helper()
	tx := &hashCapture{fakeSubmitTx: &fakeSubmitTx{allowed: true}}
	if _, err := (effectcommit.Runner{Provider: hashProvider{tx}}).Execute(context.Background(), req, grant); err != nil {
		t.Fatal(err)
	}
	return tx.hash
}

func TestRunnerOrderSubmitUnknownCommitKeepsCategoryAndDoesNotRetry(t *testing.T) {
	p := newFakeSubmit()
	p.tx.commitErr = errors.New("postgres://fixture-secret@internal-db lost")
	st := &statusReader{}
	s, cr := newFakeStore(t, p, nil, st)
	runner := newOrderSubmitRunner(t, s)
	result, err := runner.RunCommand(context.Background(), orderSubmitInput)
	if !errors.Is(err, errOrderSubmitCommitUnconfirmed) || result.Stage != "command-commit-unconfirmed" ||
		result.Command != nil || result.Observation != nil || cr.executes != 1 || st.reads != 0 {
		t.Fatalf("result=%+v err=%v executes=%d reads=%d", result, err, cr.executes, st.reads)
	}
	if strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("leak")
	}
	// HTTP: 500, no secrets.
	p2 := newFakeSubmit()
	p2.tx.commitErr = errors.New("postgres://fixture-secret@internal-db lost")
	s2, _ := newFakeStore(t, p2, nil, st)
	loaded, err := workspace.Load(referenceWorkspacePath(t))
	if err != nil {
		t.Fatal(err)
	}
	projection, err := buildWebTargetProjection(loaded.World, projectionConfig{
		HandoverBaselineID: testBaselineID, BaselineRepositoryRevision: testBaselineRevision,
		TargetBuildRevision: testTargetRevision, ApplicationID: core.MustCanonicalID("order_entry"),
	})
	if err != nil {
		t.Fatal(err)
	}
	server, err := newWebTargetServer(projection, newReferenceReadScenarioRunner(loaded.World), newOrderSubmitRunner(t, s2))
	if err != nil {
		t.Fatal(err)
	}
	body := `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/scenario/submit", strings.NewReader(body)))
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "fixture-secret") ||
		strings.Contains(w.Body.String(), "postgres://") || st.reads != 0 {
		t.Fatalf("HTTP=%d body=%s reads=%d", w.Code, w.Body.String(), st.reads)
	}
	var got webCommandScenarioErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.Result.Stage != "command-commit-unconfirmed" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestRunnerOrderSubmitObservationFailureAfterCommit(t *testing.T) {
	p := newFakeSubmit()
	st := &statusReader{err: errors.New("postgres://account:password@internal-db/private reset")}
	s, _ := newFakeStore(t, p, nil, st)
	result, err := newOrderSubmitRunner(t, s).RunCommand(context.Background(), orderSubmitInput)
	if err == nil || result.Stage != "observation" || result.Command == nil || result.Observation != nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if strings.Contains(err.Error(), "postgres://") || p.tx.commitCalls != 1 {
		t.Fatalf("err=%v commits=%d", err, p.tx.commitCalls)
	}
}

func TestRunnerOrderSubmitHostGrantDeniedNeverCallsRunner(t *testing.T) {
	p := newFakeSubmit()
	s, cr := newFakeStore(t, p, nil, nil)
	runner := newOrderSubmitRunner(t, s)
	runner.hostEffect = func(context.Context, string) error { return errors.New("revoked") }
	result, err := runner.RunCommand(context.Background(), orderSubmitInput)
	if err == nil || result.Stage != "host-effect-grant-denied" || cr.executes != 0 || p.tx.writeCalls != 0 {
		t.Fatalf("result=%+v err=%v executes=%d", result, err, cr.executes)
	}
}

func TestRunnerOrderSubmitMissingGrantSourceFailsClosed(t *testing.T) {
	p := newFakeSubmit()
	if _, err := newRunnerOrderSubmitStore(effectcommit.Runner{Provider: p}, nil, nil, &statusReader{}); err == nil {
		t.Fatal("nil grant source accepted")
	}
	failing := func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
		return legacyauthorization.AuthorizationGrant{}, errors.New("postgres://secret")
	}
	s, cr := newFakeStore(t, p, failing, nil)
	err := s.Submit(context.Background(), 1001)
	if err == nil || strings.Contains(err.Error(), "secret") || cr.executes != 0 {
		t.Fatalf("err=%v executes=%d", err, cr.executes)
	}
	nilEpochs := func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
		g := syntheticSubmitGrant()
		g.Epochs = nil
		return g, nil
	}
	s, _ = newFakeStore(t, p, nilEpochs, nil)
	err = s.Submit(context.Background(), 1001)
	if err == nil || p.tx.checkCalls != 0 || p.tx.writeCalls != 0 {
		t.Fatalf("nil-epochs grant not refused: err=%v checks=%d", err, p.tx.checkCalls)
	}
}

func TestRunnerOrderSubmitSchemaNotReadyRefusesBeforeRunner(t *testing.T) {
	p := newFakeSubmit()
	cr := &countingRunner{inner: effectcommit.Runner{Provider: p}}
	s, err := newRunnerOrderSubmitStore(cr, func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
		return syntheticSubmitGrant(), nil
	}, func(context.Context) error { return errors.New("catalog detail") }, &statusReader{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Submit(context.Background(), 1001); !errors.Is(err, errPhysicalSchemaNotReady) || cr.executes != 0 {
		t.Fatalf("err=%v executes=%d", err, cr.executes)
	}
}
