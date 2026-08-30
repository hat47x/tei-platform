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

type fakeExec struct{ id string }

func (f fakeExec) ID() string { return f.id }
func (f fakeExec) Supports(c executor.Command) bool {
	return c.Kind == string(deployment.ActionDatabaseRelease)
}
func (f fakeExec) Execute(_ context.Context, c executor.Command) (executor.Result, error) {
	return executor.Result{ExecutorID: f.id, AttemptID: "a1", Outcome: executor.OutcomeAccepted}, nil
}

func TestExplicitExecutorBinding(t *testing.T) {
	env := environment.Environment{ID: "production", DeploymentExecutors: []environment.DeploymentExecutorBinding{{Action: deployment.ActionDatabaseRelease, Executor: "migration.atlas"}}}
	cmd := executor.Command{ActionID: "db/apply", IdempotencyKey: "k", Kind: string(deployment.ActionDatabaseRelease)}
	r, err := routing.RouteCommand(env, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if r.ExecutorID != "migration.atlas" {
		t.Fatalf("unexpected route: %+v", r)
	}
	router := routing.Router{Executors: map[core.CanonicalID]executor.Executor{"migration.atlas": fakeExec{id: "migration.atlas"}}}
	res, err := router.Dispatch(context.Background(), env, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if res.ActionID != cmd.ActionID || res.IdempotencyKey != cmd.IdempotencyKey {
		t.Fatalf("result identity not normalized: %+v", res)
	}
}

func TestMissingBindingDoesNotGuess(t *testing.T) {
	env := environment.Environment{ID: "production"}
	_, err := routing.RouteCommand(env, executor.Command{Kind: string(deployment.ActionDatabaseRelease)})
	if err == nil {
		t.Fatal("expected missing binding error")
	}
}

func TestTargetSpecificBindingOverridesGlobalFallback(t *testing.T) {
	env := environment.Environment{ID: "production", DeploymentExecutors: []environment.DeploymentExecutorBinding{
		{Action: deployment.ActionDatabaseRelease, Scope: environment.ExecutorScopeGlobal, Executor: "migration.default"},
		{Action: deployment.ActionDatabaseRelease, Scope: environment.ExecutorScopeDatabase, Target: "sales_core", Executor: "migration.atlas"},
	}}
	r, err := routing.RouteCommand(env, executor.Command{Kind: string(deployment.ActionDatabaseRelease), Database: "sales_core"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ExecutorID != "migration.atlas" {
		t.Fatalf("specific binding must win: %+v", r)
	}
	r, err = routing.RouteCommand(env, executor.Command{Kind: string(deployment.ActionDatabaseRelease), Database: "other_db"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ExecutorID != "migration.default" {
		t.Fatalf("global fallback expected: %+v", r)
	}
}
