//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/submit"
)

// AT-1 characterization for issue #1174 (real PostgreSQL, TEST-ONLY DDL,
// SYNTHETIC grant; gated by TEI_TEST_POSTGRES_DSN, a skip is not evidence).
//
// The first test documents a KNOWN GAP, it does not close it: effectcommit.Runner
// does not reconcile Request.Payload with Request.Scope.Resource or the
// OperationID. The test-only apply_owner_effect routine takes the target order
// id from the payload, so a Request whose OperationID names order 1001 but whose
// payload targets order 1002 updates 1002. Hence the TARGET CALLER must keep
// them consistent. This is evidence for the open question (an Owner decision),
// NOT a claim that the gap is closed. If this test starts failing, the Runner or
// the Provider began to reconcile them: update the issue and this test together.
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

func TestRunnerOrderSubmitRealPostgresAT1CharacterizationKnownGapPayloadTargetNotReconciledWithScope(t *testing.T) {
	env := newRunnerPGEnv(t)
	grant := syntheticSubmitGrant()
	req, err := runnerSubmitRequest(grant, 1001) // Scope and OperationID name order 1001
	if err != nil {
		t.Fatal(err)
	}
	if req.OperationID != "order-submit/1001" {
		t.Fatalf("operation=%q", req.OperationID)
	}
	req.Payload = []byte(`{"orderId":1002,"to":"submitted"}`) // payload targets 1002

	res, err := effectcommit.Runner{Provider: env.provider(t)}.Execute(context.Background(), req, grant)
	if err != nil || res.Status != effectcommit.StatusCommitted {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	// KNOWN GAP (characterized): the Runner accepted it and the effect landed on 1002.
	if got := atSecondConnStatus(t, env, 1002); got != "submitted" {
		t.Fatalf("1002=%q: payload target was not applied; the gap may have been closed, update this characterization", got)
	}
	if got := atSecondConnStatus(t, env, 1001); got != "draft" {
		t.Fatalf("1001=%q", got)
	}
	// The receipt (audit trail) names operation 1001 although 1002 changed.
	if ops := atReceiptOperations(t, env); len(ops) != 1 || ops[0] != "order-submit/1001" {
		t.Fatalf("receipts=%v", ops)
	}
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
