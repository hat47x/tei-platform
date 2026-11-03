//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/submit"
)

// AT-1 for issue #1174 (real PostgreSQL, TEST-ONLY DDL, SYNTHETIC grant; gated
// by TEI_TEST_POSTGRES_DSN, a skip is not evidence).
//
// effectcommit.Runner does not itself reconcile Request.Payload with
// Request.Scope.Resource or the OperationID, and the test-only apply_owner_effect
// routine takes the target order id from the payload. Owner provisional decision
// D11 therefore puts the guarantee on the TARGET that composes the Intent: the
// shared checkOrderSubmitIntent is installed as submit.Spec.ValidateIntent, and a
// mismatching Intent is refused as NotCommitted before the Runner is reached.
// The first test is the regression test of that: it replaced the earlier
// characterization of the known gap. It does not claim the Runner or the Provider
// reconcile the two; a direct Runner call with a mismatching Request is not
// covered here.
//
// The second test shows that the Web and CLI callers, which build the payload
// from the order id themselves, keep payload, OperationID and effect consistent.

func atSecondConnStatus(t *testing.T, env *runnerPGEnv, id int64) string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, env.dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var s string
	if err := conn.QueryRow(ctx, `SELECT status FROM public.orders WHERE order_id=$1`, id).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func atReceiptOperations(t *testing.T, env *runnerPGEnv) []string {
	t.Helper()
	rows, err := env.db.Query(`SELECT operation_id FROM ` + runnerTestSchema + `.receipts ORDER BY operation_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ops []string
	for rows.Next() {
		var op string
		if err := rows.Scan(&op); err != nil {
			t.Fatal(err)
		}
		ops = append(ops, op)
	}
	return ops
}

func TestRunnerOrderSubmitRealPostgresAT1TargetRefusesPayloadNotReconciledWithOperationBeforeExecution(t *testing.T) {
	mismatches := map[string][]byte{
		"other-order":  []byte(`{"orderId":1002,"to":"submitted"}`),
		"other-state":  []byte(`{"orderId":1001,"to":"draft"}`),
		"extra-key":    []byte(`{"orderId":1001,"to":"submitted","x":1}`),
		"missing-to":   []byte(`{"orderId":1001}`),
		"not-json":     []byte(`order 1001`),
		"empty":        nil,
		"duplicate-id": []byte(`{"orderId":1002,"orderId":1001,"to":"submitted"}`),
	}
	for _, kind := range []string{"web", "cli"} {
		for name, payload := range mismatches {
			t.Run(kind+"/"+name, func(t *testing.T) {
				env := newRunnerPGEnv(t)
				for _, q := range []string{cliAuthoritySQL, atSeedEpochsSQL} {
					if _, err := env.db.Exec(q); err != nil {
						t.Fatal(err)
					}
				}
				sink := &atSink{}
				rec := &atRecRunner{inner: effectcommit.Runner{Provider: env.provider(t),
					Audit: effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceRealPG}}}
				spec, in := runnerSubmitSpec, effectIntentFor(kind, 1001)
				if kind == "cli" {
					spec = cliSpec
				}
				in.Payload = payload // OperationID names 1001, the payload does not agree
				caller := submit.Caller{Runner: rec, Grants: atGrants(kind, nil), Spec: spec}
				grant, err := caller.Grant(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				out, err := caller.Execute(context.Background(), grant, in)
				if !errors.Is(err, submit.ErrIntentRejected) || out.Disposition != submit.NotCommitted {
					t.Fatalf("out=%+v err=%v", out, err)
				}
				if kind == "web" {
					if _, err := runnerSubmitRequestFromIntent(grant, in); !errors.Is(err, submit.ErrIntentRejected) {
						t.Fatalf("web request builder err=%v", err)
					}
				}
				if rec.calls != 0 {
					t.Fatalf("Runner reached %d times", rec.calls)
				}
				if len(sink.got) != 0 {
					t.Fatalf("evidence emitted: %d", len(sink.got))
				}
				for _, id := range []int64{1001, 1002} {
					if got := atSecondConnStatus(t, env, id); got != "draft" {
						t.Fatalf("%d=%q: the database changed", id, got)
					}
				}
				if ops := atReceiptOperations(t, env); len(ops) != 0 {
					t.Fatalf("receipts=%v", ops)
				}
			})
		}
	}
}

// effectIntentFor returns the Intent the given caller composes for an order.
func effectIntentFor(kind string, id int64) submit.Intent {
	if kind == "cli" {
		return cliIntent(id)
	}
	in, err := runnerSubmitIntent(id)
	if err != nil {
		panic(err)
	}
	return in
}

func TestRunnerOrderSubmitRealPostgresAT1CallersKeepPayloadTargetConsistentWithOperation(t *testing.T) {
	type payload struct {
		OrderID int64  `json:"orderId"`
		To      string `json:"to"`
	}
	for _, kind := range []string{"web", "cli"} {
		for _, id := range []int64{1001, 1002} {
			t.Run(kind+"/"+itoa(id), func(t *testing.T) {
				env := newRunnerPGEnv(t)
				for _, q := range []string{cliAuthoritySQL, atSeedEpochsSQL} { // atBaseGrant uses the distinct epochs
					if _, err := env.db.Exec(q); err != nil {
						t.Fatal(err)
					}
				}
				// 1. The payload each caller builds for order N targets N, as does the OperationID.
				var raw []byte
				var op string
				if kind == "web" {
					req, err := runnerSubmitRequest(syntheticSubmitGrant(), id)
					if err != nil {
						t.Fatal(err)
					}
					raw, op = req.Payload, req.OperationID
				} else {
					in := cliIntent(id)
					raw, op = in.Payload, in.OperationID
				}
				var p payload
				if err := json.Unmarshal(raw, &p); err != nil || p.OrderID != id || p.To != "submitted" {
					t.Fatalf("payload=%s err=%v", raw, err)
				}
				if !strings.HasSuffix(op, "/"+itoa(id)) {
					t.Fatalf("operation=%q", op)
				}
				// 2. End to end: only order N changes and the receipt names N.
				run := atSubmit(t, env, kind, env.provider(t), atGrants(kind, nil), id)
				if run.label != "committed" || run.rec.calls != 1 ||
					submit.Classify(run.rec.result, run.rec.err).Disposition != submit.Committed {
					t.Fatalf("run=%+v", run)
				}
				other := int64(1001)
				if id == 1001 {
					other = 1002
				}
				if atSecondConnStatus(t, env, id) != "submitted" || atSecondConnStatus(t, env, other) != "draft" {
					t.Fatalf("statuses %d=%q %d=%q", id, atSecondConnStatus(t, env, id), other, atSecondConnStatus(t, env, other))
				}
				if ops := atReceiptOperations(t, env); len(ops) != 1 || ops[0] != op {
					t.Fatalf("receipts=%v want [%s]", ops, op)
				}
			})
		}
	}
}
