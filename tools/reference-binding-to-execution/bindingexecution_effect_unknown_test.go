package bindingexecution

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
)

// This is an in-memory counterfactual, not an external effect receipt.
// The acknowledgement can be lost or invalid *after* Execute changed state.
type effectUnknownExecutor struct {
	support bool
	change  bool
	fail    bool
	wrongID bool
	calls   int
	state   string
}

func (e *effectUnknownExecutor) ID() string                     { return "migration.atlas" }
func (e *effectUnknownExecutor) Supports(executor.Command) bool { return e.support }

func (e *effectUnknownExecutor) Execute(_ context.Context, cmd executor.Command) (executor.Result, error) {
	e.calls++
	if e.change {
		e.state = "applied"
	}
	if e.fail {
		return executor.Result{}, errors.New("synthetic acknowledgement lost after possible effect")
	}
	id := e.ID()
	if e.wrongID {
		id = "unrelated.executor"
	}
	return executor.Result{
		ExecutorID:     id,
		ActionID:       cmd.ActionID,
		IdempotencyKey: cmd.IdempotencyKey,
		AttemptID:      "synthetic-attempt",
		Outcome:        executor.OutcomeAccepted,
	}, nil
}

// BXE-01..06: the reference must never call a post-Execute error
// a preflight refusal, and an acknowledged call is not an effect receipt.
func TestBindingExecutionDifferentiatesPreflightFromPostInvokeUnknown(t *testing.T) {
	cases := []struct {
		name         string
		support      bool
		change       bool
		fail         bool
		wrongID      bool
		wantRan      int
		wantRefused  int
		wantUnknown  int
		wantState    string
		wantResolved bool
	}{
		{"BXE-01-unsupported-before-Execute-is-refused", false, false, false, false, 0, 1, 0, "pending", false},
		{"BXE-02-effect-then-error-is-unverified", true, true, true, false, 1, 0, 1, "applied", false},
		{"BXE-03-error-before-effect-is-still-unverified", true, false, true, false, 1, 0, 1, "pending", false},
		{"BXE-04-effect-then-invalid-ack-is-unverified", true, true, false, true, 1, 0, 1, "applied", false},
		{"BXE-05-accepted-without-effect-is-invoked", true, false, false, false, 1, 0, 0, "pending", true},
		{"BXE-06-accepted-with-effect-is-invoked", true, true, false, false, 1, 0, 0, "applied", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			impl := &effectUnknownExecutor{
				support: tc.support,
				change:  tc.change,
				fail:    tc.fail,
				wrongID: tc.wrongID,
				state:   "pending",
			}
			report, err := Run(writeWorkspace(t), "development", []executor.Command{{
				ActionID:       "db/apply",
				IdempotencyKey: "synthetic-key",
				Kind:           string(deployment.ActionDatabaseRelease),
				Database:       "sales_core",
			}}, map[core.CanonicalID]executor.Executor{"migration.atlas": impl})
			if err != nil {
				t.Fatalf("reference Run failed: %v", err)
			}
			if report.Schema != Schema || report.Authority != Authority ||
				report.Ran != tc.wantRan ||
				report.Refused != tc.wantRefused ||
				report.Unverified != tc.wantUnknown ||
				len(report.Outcomes) != 1 {
				t.Fatalf("wrong counters or scope: %+v", report)
			}
			out := report.Outcomes[0]
			if impl.calls != tc.wantRan || out.ExecutorInvoked != (tc.wantRan == 1) ||
				out.BindingResolved != tc.wantResolved || impl.state != tc.wantState {
				t.Fatalf("wrong execution evidence: outcome=%+v calls=%d state=%s",
					out, impl.calls, impl.state)
			}
			if tc.wantResolved {
				if out.Error != "" || out.RawOutcome != string(executor.OutcomeAccepted) {
					t.Fatalf("validated acknowledgement disappeared: %+v", out)
				}
			} else if out.Error == "" {
				t.Fatalf("failed path did not retain its reason: %+v", out)
			}
			// No retry or external observation takes place here. Distinct
			// states after the same error prove why a caller cannot infer
			// safe retry or no-effect from acknowledgement status.
		})
	}
}
