package routing_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/routing"
	"tei/internal/environment"
)

// forgingExec returns a result that names another executor.
type forgingExec struct{ id, claims string }

func (f forgingExec) ID() string { return f.id }
func (f forgingExec) Supports(c executor.Command) bool {
	return c.Kind == string(deployment.ActionDatabaseRelease)
}
func (f forgingExec) Execute(_ context.Context, _ executor.Command) (executor.Result, error) {
	return executor.Result{
		ExecutorID: f.claims, AttemptID: "a1", Outcome: executor.OutcomeAccepted,
	}, nil
}

func dispatchEnv() environment.Environment {
	return environment.Environment{
		ID: "production",
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action: deployment.ActionDatabaseRelease, Executor: "migration.atlas",
		}},
	}
}

func dispatchCmd() executor.Command {
	return executor.Command{
		ActionID: "db/apply", IdempotencyKey: "k",
		Kind: string(deployment.ActionDatabaseRelease),
	}
}

// Dispatch fills an empty identity but does NOT validate it. This is the
// documented three-stage split (RouteCommand -> Dispatch -> ValidateResult):
// Dispatch runs the executor, ValidateResult decides whether the answer is
// attributable. Pinning it here keeps a later change from silently merging the
// stages, which would make a forged answer impossible to detect.
func TestDispatchDoesNotValidateTheResultIdentity(t *testing.T) {
	env := dispatchEnv()
	cmd := dispatchCmd()
	router := routing.Router{Executors: map[core.CanonicalID]executor.Executor{
		"migration.atlas": forgingExec{id: "migration.atlas", claims: "someone.else"},
	}}

	res, err := router.Dispatch(context.Background(), env, cmd)
	if err != nil {
		t.Fatalf("Dispatch should not reject the answer; that is ValidateResult's job: %v", err)
	}
	// The forged identity survives Dispatch, which is why ValidateResult must run.
	if res.ExecutorID != "someone.else" {
		t.Fatalf("Dispatch overwrote the answer's identity: %q", res.ExecutorID)
	}

	// And ValidateResult is what rejects it.
	if err := routing.ValidateResult(env, cmd, res); err == nil {
		t.Fatal("ValidateResult accepted an answer from another executor")
	}
}

// The other half: Dispatch fills fields the executor left empty, so a caller
// that skips normalization still gets a usable identity.
func TestDispatchFillsAnEmptyResultIdentity(t *testing.T) {
	env := dispatchEnv()
	cmd := dispatchCmd()
	router := routing.Router{Executors: map[core.CanonicalID]executor.Executor{
		"migration.atlas": emptyIdentityExec{},
	}}

	res, err := router.Dispatch(context.Background(), env, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutorID != "migration.atlas" {
		t.Fatalf("executor id = %q", res.ExecutorID)
	}
	if res.ActionID != cmd.ActionID {
		t.Fatalf("action id = %q", res.ActionID)
	}
	if res.IdempotencyKey != cmd.IdempotencyKey {
		t.Fatalf("idempotency key = %q", res.IdempotencyKey)
	}
	if err := routing.ValidateResult(env, cmd, res); err != nil {
		t.Fatalf("a filled identity should validate: %v", err)
	}
}

type emptyIdentityExec struct{}

func (emptyIdentityExec) ID() string { return "migration.atlas" }
func (emptyIdentityExec) Supports(c executor.Command) bool {
	return c.Kind == string(deployment.ActionDatabaseRelease)
}
func (emptyIdentityExec) Execute(_ context.Context, _ executor.Command) (executor.Result, error) {
	return executor.Result{AttemptID: "a1", Outcome: executor.OutcomeAccepted}, nil
}
