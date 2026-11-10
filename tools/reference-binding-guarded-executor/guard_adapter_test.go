package bindingguardedexecutor_test

import (
	"context"
	"encoding/json"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	bindingguardedexecutor "tei/tools/reference-binding-guarded-executor"
)

// recordingBuiltin answers with a fixed payload and counts entries, so the test
// can tell "the adapter ran" from "the adapter was never handed an executor".
type recordingBuiltin struct {
	id      plugin.Identity
	fp      string
	payload string
	calls   int
}

func (b *recordingBuiltin) Identity() plugin.Identity     { return b.id }
func (b *recordingBuiltin) DescriptorFingerprint() string { return b.fp }
func (b *recordingBuiltin) Invoke(_ context.Context, _ pruntime.Request) (pruntime.Response, error) {
	b.calls++
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: []byte(b.payload)}, nil
}

func cid(s string) core.CanonicalID { return core.MustCanonicalID(s) }

// adapterFixture builds a real DeploymentExecutor bound to "migration.atlas",
// as the existing reference sites do.
func adapterFixture(t *testing.T, builtin *recordingBuiltin) pruntime.DeploymentExecutor {
	t.Helper()
	def := plugin.Definition{
		ID: cid("tei.migration.atlas"), Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDeploymentExecutor, ID: cid("migration.atlas"),
			Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
		}},
	}
	env := environment.Environment{
		ID:      cid("development"),
		Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}},
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action: deployment.ActionDatabaseRelease, Executor: cid("migration.atlas"),
		}},
	}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, err := plugin.BuildLock(cat, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	builtin.id = plugin.Identity{ID: def.ID, Version: def.Version}
	builtin.fp = plugin.DefinitionFingerprint(def)
	reg := pruntime.Registry{Builtins: map[string]pruntime.Builtin{
		"tei.migration.atlas@1.0.0": builtin,
	}}
	// The hard-coded id mirrors the existing reference sites.
	return pruntime.DeploymentExecutor{
		Plan: plan, Registry: reg, CapabilityID: cid("migration.atlas"),
	}
}

func guardInput(t *testing.T) bindingguardedexecutor.Input {
	t.Helper()
	env := environment.Environment{
		ID: cid("development"),
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action: deployment.ActionDatabaseRelease, Executor: cid("migration.atlas"),
		}},
	}
	return bindingguardedexecutor.Input{
		Environment: env,
		Registered:  []core.CanonicalID{cid("migration.atlas")},
	}
}

// The guard serves the adapter's id when the binding agrees, and the real
// adapter then runs.
func TestTheAdapterRunsWhenTheGuardServes(t *testing.T) {
	builtin := &recordingBuiltin{payload: `{"attemptId":"a1","outcome":"accepted"}`}
	adapter := adapterFixture(t, builtin)
	in := guardInput(t)
	in.Requests = []bindingguardedexecutor.Request{{
		Command:     executor.Command{ActionID: "db/apply", Kind: string(deployment.ActionDatabaseRelease)},
		RequestedID: cid("migration.atlas"),
	}}

	report := bindingguardedexecutor.Run(in)
	if report.Served != 1 || report.Refused != 0 {
		t.Fatalf("report = %+v", report)
	}

	res, err := adapter.Execute(context.Background(), executor.Command{
		ActionID: "db/apply", Kind: string(deployment.ActionDatabaseRelease),
		IdempotencyKey: "k",
	})
	if err != nil {
		t.Fatalf("the served adapter failed: %v", err)
	}
	if builtin.calls != 1 {
		t.Fatalf("the builtin was entered %d times", builtin.calls)
	}
	if res.ExecutorID != "migration.atlas" {
		t.Fatalf("executor id = %q", res.ExecutorID)
	}
}

// This is the blindspot, end to end: the binding selects another executor, so
// the guard refuses. Without the guard, the hard-coded adapter would have run
// the wrong one — which the second half of the test demonstrates.
func TestTheGuardStopsTheAdapterTheBindingDidNotSelect(t *testing.T) {
	builtin := &recordingBuiltin{payload: `{"attemptId":"a1","outcome":"accepted"}`}
	adapter := adapterFixture(t, builtin)

	// The Host's environment now binds a different executor.
	in := guardInput(t)
	in.Environment.DeploymentExecutors = []environment.DeploymentExecutorBinding{{
		Action: deployment.ActionDatabaseRelease, Executor: cid("migration.other"),
	}}
	in.Registered = []core.CanonicalID{cid("migration.atlas"), cid("migration.other")}
	in.Requests = []bindingguardedexecutor.Request{{
		Command:     executor.Command{ActionID: "db/apply", Kind: string(deployment.ActionDatabaseRelease)},
		RequestedID: cid("migration.atlas"),
	}}

	report := bindingguardedexecutor.Run(in)
	if report.Refused != 1 {
		t.Fatalf("the guard did not refuse: %+v", report)
	}
	if report.Outcomes[0].Status != bindingguardedexecutor.StatusNotMatched {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if builtin.calls != 0 {
		t.Fatalf("the guard ran the executor %d times", builtin.calls)
	}

	// The counterfactual: the adapter itself has no idea. Called directly, it
	// runs the hard-coded executor even though the binding names another.
	if _, err := adapter.Execute(context.Background(), executor.Command{
		ActionID: "db/apply", Kind: string(deployment.ActionDatabaseRelease),
		IdempotencyKey: "k",
	}); err != nil {
		t.Fatalf("the adapter should still run when called directly: %v", err)
	}
	if builtin.calls != 1 {
		t.Fatalf("the adapter did not run the hard-coded executor: calls=%d", builtin.calls)
	}
}

// keep json referenced for the payload literal shape.
var _ = json.RawMessage{}
