//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
	effectpostgres "tei/internal/security/effectcommit/postgres"
	"tei/internal/security/legacyauthorization"
)

// Real PostgreSQL tests for the opt-in Evidence sink on runnerOrderSubmitStore
// (Issue #1174 requirements 6 and 8). Same gating and test-only DDL as
// runner_order_submit_postgres_test.go; they SKIP without TEI_TEST_POSTGRES_DSN
// and a skip is not evidence. The Provenance label is chosen by this test file
// (the composition), never by the store. Guarantee is the Provider's
// self-declaration; the trusted-owner postgres Provider declares
// atomic-with-commit with the Reason "provisional D10: ..." (provisional,
// unverified), which GuaranteeReason carries into Evidence. This is a reference
// result, not proof of production audit.

const evidenceCanary = "CANARY-SECRET-7f3a9c"

type recordingSink struct {
	mu  sync.Mutex
	got []effectcommit.Evidence
}

func (r *recordingSink) Record(e effectcommit.Evidence) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, e)
}

func (r *recordingSink) all() []effectcommit.Evidence {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]effectcommit.Evidence(nil), r.got...)
}

func (e *runnerPGEnv) auditedStore(t *testing.T, sink effectcommit.EvidenceSink) *runnerOrderSubmitStore {
	t.Helper()
	s, err := newPostgresRunnerOrderSubmitStore(context.Background(), e.dsn, e.dsn, runnerTestSchema, e.world,
		func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
			return syntheticSubmitGrant(), nil
		},
		withRunnerAudit(effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceRealPG}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

type storedReceipt struct {
	operationID string
	version     uint64
}

// storedReceipts reads the receipts table over env.db, a connection separate
// from the store's pools.
func (e *runnerPGEnv) storedReceipts(t *testing.T) []storedReceipt {
	t.Helper()
	rows, err := e.db.Query(`SELECT operation_id, resource_version FROM ` + runnerTestSchema + `.receipts ORDER BY operation_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []storedReceipt
	for rows.Next() {
		var r storedReceipt
		if err := rows.Scan(&r.operationID, &r.version); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// secretRenderings returns the forms in which a secret could leak through a
// future []byte/[]rune/etc. field: raw, decimal byte list, hex and base64.
func secretRenderings(secret string) []string {
	return []string{
		secret,
		fmt.Sprint([]byte(secret)),
		hex.EncodeToString([]byte(secret)),
		base64.StdEncoding.EncodeToString([]byte(secret)),
		base64.RawStdEncoding.EncodeToString([]byte(secret)),
	}
}

// evidenceLeaks reports whether any rendering (%v, %+v, %#v) of any Evidence
// contains the secret in any of the secretRenderings forms.
func evidenceLeaks(evs []effectcommit.Evidence, secrets []string) bool {
	for _, ev := range evs {
		for _, f := range []string{"%v", "%+v", "%#v"} {
			text := fmt.Sprintf(f, ev)
			for _, sec := range secrets {
				if sec == "" {
					continue
				}
				for _, n := range secretRenderings(sec) {
					if strings.Contains(text, n) {
						return true
					}
				}
			}
		}
	}
	return false
}

// assertSecretFree fails if any Evidence renders the canary or the DSN.
func assertSecretFree(t *testing.T, dsn string, evs []effectcommit.Evidence) {
	t.Helper()
	needles := []string{evidenceCanary, dsn}
	if u, err := url.Parse(dsn); err == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok && pw != "" {
			needles = append(needles, pw)
		}
	}
	if evidenceLeaks(evs, needles) {
		t.Fatalf("evidence contains a secret value")
	}
}

// TestEvidenceSecretCheckDetectsLeaks proves the check itself fires on a
// string leak and on a []byte leak (printed as decimals by %+v).
func TestEvidenceSecretCheckDetectsLeaks(t *testing.T) {
	if evidenceLeaks([]effectcommit.Evidence{{Operation: "order-submit/1"}}, []string{evidenceCanary}) {
		t.Fatal("clean evidence reported as leaking")
	}
	if !evidenceLeaks([]effectcommit.Evidence{{Operation: "x " + evidenceCanary}}, []string{evidenceCanary}) {
		t.Fatal("string leak not detected")
	}
	type leaky struct{ B []byte }
	text := fmt.Sprintf("%+v", leaky{B: []byte(evidenceCanary)})
	if strings.Contains(text, evidenceCanary) {
		t.Fatal("premise broken: verb plus-v printed the bytes as text")
	}
	found := false
	for _, n := range secretRenderings(evidenceCanary) {
		if strings.Contains(text, n) {
			found = true
		}
	}
	if !found {
		t.Fatal("[]byte leak not detected by the renderings")
	}
}

// assertBaseEvidence expects the trusted-owner Provider's provisional D10
// declaration. The ack-loss wrapper in this test file holds the Provider in a
// named field and declares nothing, so it uses assertBaseEvidenceGuarantee.
func assertBaseEvidence(t *testing.T, ev effectcommit.Evidence, op string) {
	t.Helper()
	assertBaseEvidenceGuarantee(t, ev, op, effectcommit.GuaranteeAtomicWithCommit, effectpostgres.ReasonProvisionalD10)
}

func assertBaseEvidenceGuarantee(t *testing.T, ev effectcommit.Evidence, op string, wantLevel effectcommit.GuaranteeLevel, wantReason string) {
	t.Helper()
	g := syntheticSubmitGrant()
	if ev.Operation != op {
		t.Fatalf("operation=%q want %q", ev.Operation, op)
	}
	if ev.Provenance != effectcommit.ProvenanceRealPG {
		t.Fatalf("provenance=%q", ev.Provenance)
	}
	if ev.Guarantee != wantLevel || ev.GuaranteeReason != wantReason {
		t.Fatalf("guarantee=%q reason=%q, want %q / %q", ev.Guarantee, ev.GuaranteeReason, wantLevel, wantReason)
	}
	want := effectcommit.Scope{Application: runnerSubmitApplication, Capability: runnerSubmitCapability,
		Subject: g.SubjectID, Action: g.Action, Resource: g.Resource}
	if ev.Scope != want {
		t.Fatalf("scope=%+v", ev.Scope)
	}
	if ev.Epochs != *g.Epochs {
		t.Fatalf("epochs=%+v", ev.Epochs)
	}
}

func TestRunnerOrderSubmitRealPostgresEvidenceHappyReplayAndObservation(t *testing.T) {
	env := newRunnerPGEnv(t)
	sink := &recordingSink{}
	store := env.auditedStore(t, sink)
	ctx := context.Background()

	if err := store.Submit(ctx, 1001); err != nil {
		t.Fatal(err)
	}
	evs := sink.all()
	if len(evs) != 1 {
		t.Fatalf("evidence after submit=%d", len(evs))
	}
	stored := env.storedReceipts(t)
	if len(stored) != 1 {
		t.Fatalf("stored=%+v", stored)
	}
	ev := evs[0]
	assertBaseEvidence(t, ev, "execute")
	if ev.Status != effectcommit.StatusCommitted || ev.ErrorClass != effectcommit.ErrorClassNone {
		t.Fatalf("ev=%+v", ev)
	}
	if ev.OperationID != stored[0].operationID || ev.ReceiptVersion != stored[0].version || ev.ReceiptVersion == 0 {
		t.Fatalf("evidence (%q,%d) != stored (%q,%d)", ev.OperationID, ev.ReceiptVersion, stored[0].operationID, stored[0].version)
	}

	// Observation: the separate read pool agrees with the committed Evidence.
	status, err := store.ReadStatus(ctx, 1001)
	if err != nil || status != "submitted" {
		t.Fatalf("status=%q err=%v", status, err)
	}
	if got := orderStatus(t, env.db, 1001); got != "submitted" || ev.Status != effectcommit.StatusCommitted {
		t.Fatalf("db=%q ev=%s", got, ev.Status)
	}

	// Replay with the same OperationID: one more Evidence, same identifier/version.
	if err := store.Submit(ctx, 1001); !errors.Is(err, errOrderNotSubmittable) {
		t.Fatalf("second submit err=%v", err)
	}
	evs = sink.all()
	if len(evs) != 2 {
		t.Fatalf("evidence after replay=%d", len(evs))
	}
	assertBaseEvidence(t, evs[1], "execute")
	if evs[1].Status != effectcommit.StatusReplayed || evs[1].OperationID != stored[0].operationID || evs[1].ReceiptVersion != stored[0].version {
		t.Fatalf("replay ev=%+v stored=%+v", evs[1], stored[0])
	}

	// Same OperationID with a canary payload: Conflict; the payload is not in Evidence.
	grant := syntheticSubmitGrant()
	req, _ := runnerSubmitRequest(grant, 1001)
	req.Payload = []byte(`{"orderId":1001,"to":"cancelled","note":"` + evidenceCanary + `"}`)
	res, err := store.runner.Execute(ctx, req, grant)
	if err != nil || res.Status != effectcommit.StatusConflict {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	evs = sink.all()
	if len(evs) != 3 || evs[2].Status != effectcommit.StatusConflict || evs[2].OperationID != stored[0].operationID {
		t.Fatalf("conflict evidence=%+v", evs)
	}

	// Lookup adds exactly one more Evidence with the same identifier.
	look, err := store.LookupSubmit(ctx, 1001)
	if err != nil || look.Status != effectcommit.StatusReplayed {
		t.Fatalf("lookup=%+v err=%v", look, err)
	}
	evs = sink.all()
	if len(evs) != 4 {
		t.Fatalf("evidence after lookup=%d", len(evs))
	}
	assertBaseEvidence(t, evs[3], "lookup")
	if evs[3].Status != effectcommit.StatusReplayed || evs[3].OperationID != stored[0].operationID || evs[3].ReceiptVersion != stored[0].version {
		t.Fatalf("lookup ev=%+v", evs[3])
	}
	if got := env.storedReceipts(t); len(got) != 1 || got[0] != stored[0] {
		t.Fatalf("receipts changed: %+v", got)
	}
	assertSecretFree(t, env.dsn, evs)
}

func TestRunnerOrderSubmitRealPostgresEvidenceDeniedHasNoReceipt(t *testing.T) {
	for name, stmt := range map[string]string{
		"epoch-bump": `UPDATE ` + runnerTestSchema + `.authority SET policy=policy+1`,
		"revoked":    `UPDATE ` + runnerTestSchema + `.authority SET enabled=FALSE`,
	} {
		t.Run(name, func(t *testing.T) {
			env := newRunnerPGEnv(t)
			if _, err := env.db.Exec(stmt); err != nil {
				t.Fatal(err)
			}
			sink := &recordingSink{}
			store := env.auditedStore(t, sink)
			err := store.Submit(context.Background(), 1001)
			if !errors.Is(err, errOrderNotSubmittable) || orderStatus(t, env.db, 1001) != "draft" {
				t.Fatalf("err=%v", err)
			}
			evs := sink.all()
			if len(evs) != 1 {
				t.Fatalf("evidence=%d", len(evs))
			}
			assertBaseEvidence(t, evs[0], "execute")
			if evs[0].Status != effectcommit.StatusDenied || evs[0].ErrorClass != effectcommit.ErrorClassNone ||
				evs[0].ReceiptVersion != 0 || evs[0].OperationID != "order-submit/1001" {
				t.Fatalf("ev=%+v", evs[0])
			}
			if got := env.storedReceipts(t); len(got) != 0 {
				t.Fatalf("receipt rows=%+v", got)
			}
			assertSecretFree(t, env.dsn, evs)
		})
	}
}

func TestRunnerOrderSubmitRealPostgresEvidenceUnknownCommitThenLookupSameOperationID(t *testing.T) {
	env := newRunnerPGEnv(t)
	sink := &recordingSink{}
	pool, err := pgxpool.New(context.Background(), env.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	provider, err := effectpostgres.NewWithTrustedOwnerRoutine(pool, runnerTestSchema)
	if err != nil {
		t.Fatal(err)
	}
	lossy, err := newRunnerOrderSubmitStore(
		effectcommit.Runner{
			Provider: ackLossProvider{provider},
			Audit:    effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceRealPG},
		},
		func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
			return syntheticSubmitGrant(), nil
		}, nil, &statusReader{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = lossy.Submit(ctx, 1001)
	if !errors.Is(err, errOrderSubmitCommitUnconfirmed) {
		t.Fatalf("err=%v", err)
	}
	evs := sink.all()
	if len(evs) != 1 {
		t.Fatalf("evidence after unknown commit=%d", len(evs))
	}
	// The ack-loss wrapper declares nothing (fail closed), so its Evidence is undeclared.
	assertBaseEvidenceGuarantee(t, evs[0], "execute", effectcommit.GuaranteeUndeclared, "")
	if evs[0].ErrorClass != effectcommit.ErrorClassCommitUncertain || evs[0].Status != "" {
		t.Fatalf("ev=%+v", evs[0])
	}
	stored := env.storedReceipts(t)
	if len(stored) != 1 || orderStatus(t, env.db, 1001) != "submitted" {
		t.Fatalf("stored=%+v", stored)
	}
	if evs[0].OperationID != stored[0].operationID {
		t.Fatalf("uncertain evidence op %q != stored %q", evs[0].OperationID, stored[0].operationID)
	}

	// Recovery through the audited real store, sharing the same sink.
	recov := env.auditedStore(t, sink)
	look, err := recov.LookupSubmit(ctx, 1001)
	if err != nil || look.Status != effectcommit.StatusReplayed {
		t.Fatalf("lookup=%+v err=%v", look, err)
	}
	evs = sink.all()
	if len(evs) != 2 {
		t.Fatalf("evidence after lookup=%d", len(evs))
	}
	assertBaseEvidence(t, evs[1], "lookup")
	if evs[1].Status != effectcommit.StatusReplayed || evs[1].OperationID != evs[0].OperationID ||
		evs[1].OperationID != stored[0].operationID || evs[1].ReceiptVersion != stored[0].version {
		t.Fatalf("lookup ev=%+v first=%+v stored=%+v", evs[1], evs[0], stored[0])
	}
	assertSecretFree(t, env.dsn, evs)
	for i, ev := range evs {
		if strings.Contains(fmt.Sprintf("%+v", ev), "lost-ack-secret") {
			t.Fatalf("evidence[%d] carries driver error text", i)
		}
	}
}

func TestRunnerOrderSubmitRealPostgresEvidenceIsOptIn(t *testing.T) {
	env := newRunnerPGEnv(t)
	// No option: identical behaviour, nothing to record into.
	if err := env.store(t).Submit(context.Background(), 1001); err != nil {
		t.Fatal(err)
	}
	if len(env.storedReceipts(t)) != 1 || orderStatus(t, env.db, 1001) != "submitted" {
		t.Fatal("default store behaviour changed")
	}
}
