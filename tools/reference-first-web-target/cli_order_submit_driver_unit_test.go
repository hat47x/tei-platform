//go:build !tei_repair_only

package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/submit"
	"tei/internal/security/legacyauthorization"
)

type scriptedRunner struct {
	exec, look effectcommit.Result
	execErr    error
	lookErr    error
	executes   int
}

func (r *scriptedRunner) Execute(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	r.executes++
	return r.exec, r.execErr
}
func (r *scriptedRunner) Lookup(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	return r.look, r.lookErr
}

// DB-free: the CLI driver prints one stable line and a distinct exit code per
// Disposition, and gives Replayed exit 0 where the Web store refuses.
func TestRunnerOrderSubmitCLIDriverLinePerDisposition(t *testing.T) {
	grants := func(context.Context) (legacyauthorization.AuthorizationGrant, error) { return cliSyntheticGrant(), nil }
	unc := &effectcommit.CommitUncertainError{Cause: errors.New("postgres://secret driver")}
	for _, tc := range []struct {
		name string
		r    *scriptedRunner
		args []string
		code int
		line string
	}{
		{"committed", &scriptedRunner{exec: effectcommit.Result{Status: effectcommit.StatusCommitted, Receipt: effectcommit.Receipt{OperationID: "op"}}}, []string{"submit-order", "7"}, cliExitOK, "submitted order 7 operation=op"},
		{"replayed", &scriptedRunner{exec: effectcommit.Result{Status: effectcommit.StatusReplayed, Receipt: effectcommit.Receipt{OperationID: "op"}}}, []string{"submit-order", "7"}, cliExitOK, "already submitted order 7 operation=op"},
		{"denied", &scriptedRunner{exec: effectcommit.Result{Status: effectcommit.StatusDenied}}, []string{"submit-order", "7"}, cliExitNotCommitted, "not submitted order 7 (denied)"},
		{"conflict", &scriptedRunner{exec: effectcommit.Result{Status: effectcommit.StatusConflict}}, []string{"submit-order", "7"}, cliExitNotCommitted, "not submitted order 7 (resource-conflict)"},
		{"uncertain", &scriptedRunner{execErr: unc}, []string{"submit-order", "7"}, cliExitUncertain, "unconfirmed order 7; run lookup-order 7"},
		{"failed", &scriptedRunner{execErr: errors.New("boom")}, []string{"submit-order", "7"}, cliExitFailed, "failed order 7"},
		{"lookup-not-recorded", &scriptedRunner{look: effectcommit.Result{Status: effectcommit.StatusNotRecorded}}, []string{"lookup-order", "7"}, cliExitNotRecorded, "no record for order 7"},
		{"lookup-denied", &scriptedRunner{look: effectcommit.Result{Status: effectcommit.StatusDenied}}, []string{"lookup-order", "7"}, cliExitNotCommitted, "not submitted order 7 (denied)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			code := cliOrderRun(context.Background(), submit.Caller{Runner: tc.r, Grants: grants, Spec: cliSpec}, tc.args, &out)
			if code != tc.code || strings.TrimSpace(out.String()) != tc.line || strings.Contains(out.String(), "secret") {
				t.Fatalf("code=%d out=%q", code, out.String())
			}
		})
	}
	for _, args := range [][]string{nil, {"submit-order"}, {"submit-order", "x"}, {"submit-order", "0"}, {"delete-order", "1"}} {
		r := &scriptedRunner{}
		var out bytes.Buffer
		if code := cliOrderRun(context.Background(), submit.Caller{Runner: r, Grants: grants, Spec: cliSpec}, args, &out); code != cliExitUsage || r.executes != 0 {
			t.Fatalf("args=%v code=%d", args, code)
		}
	}
	// A failing grant source stops before the Runner.
	r := &scriptedRunner{}
	var out bytes.Buffer
	failing := submit.Caller{Runner: r, Spec: cliSpec}
	if code := cliOrderRun(context.Background(), failing, []string{"submit-order", "7"}, &out); code != cliExitFailed || r.executes != 0 {
		t.Fatalf("code=%d executes=%d", code, r.executes)
	}
}
