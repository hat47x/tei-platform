package runtime_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/routing"
	"tei/internal/environment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// reportingBuiltin returns a fixed payload, so the adapter's own identity
// handling is what the test observes.
type reportingBuiltin struct {
	id      plugin.Identity
	fp      string
	payload string
}

func (b reportingBuiltin) Identity() plugin.Identity     { return b.id }
func (b reportingBuiltin) DescriptorFingerprint() string { return b.fp }
func (b reportingBuiltin) Invoke(context.Context, pruntime.Request) (pruntime.Response, error) {
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: []byte(b.payload)}, nil
}

func executorPlan(t *testing.T) pruntime.EnvironmentPlan {
	t.Helper()
	def := plugin.Definition{
		ID: core.MustCanonicalID("tei.runtime"), Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("runtime.main"),
			Actions: []deployment.ActionKind{deployment.ActionApplicationStart},
		}},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("prod"),
		Plugins: []environment.ExtensionRequirement{
			{ID: def.ID, CompatibleMajor: 1},
		},
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action: deployment.ActionApplicationStart, Executor: core.MustCanonicalID("runtime.main"),
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
	return plan
}

func adapterWith(t *testing.T, payload string) pruntime.DeploymentExecutor {
	t.Helper()
	plan := executorPlan(t)
	id := plugin.Identity{ID: core.MustCanonicalID("tei.runtime"), Version: "1.0.0"}
	reg := pruntime.Registry{Builtins: map[string]pruntime.Builtin{
		"tei.runtime@1.0.0": reportingBuiltin{
			id: id, fp: plugin.DefinitionFingerprint(plugin.Definition{
				ID: id.ID, Version: id.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
				Provides: []plugin.Capability{{
					Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("runtime.main"),
					Actions: []deployment.ActionKind{deployment.ActionApplicationStart},
				}},
			}),
			payload: payload,
		},
	}}
	return pruntime.DeploymentExecutor{
		Plan: plan, Registry: reg, CapabilityID: core.MustCanonicalID("runtime.main"),
	}
}

// An executor that leaves the identity empty gets it filled.
func TestAnEmptyExecutorIdentityIsFilled(t *testing.T) {
	a := adapterWith(t, `{"actionId":"app/start","idempotencyKey":"k","outcome":"accepted"}`)
	res, err := a.Execute(context.Background(),
		executor.Command{ActionID: "app/start", IdempotencyKey: "k", Kind: string(deployment.ActionApplicationStart)})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutorID != "runtime.main" {
		t.Fatalf("executor id = %q, want the adapter's capability id", res.ExecutorID)
	}
}

// An executor that names ANOTHER identity keeps it. The adapter must not
// overwrite it, because routing.ValidateResult compares exactly this field.
// This is the regression the fix closes.
func TestAForeignExecutorIdentityIsNotOverwritten(t *testing.T) {
	a := adapterWith(t, `{"executorId":"someone.else","actionId":"app/start","idempotencyKey":"k","outcome":"accepted"}`)
	res, err := a.Execute(context.Background(),
		executor.Command{ActionID: "app/start", IdempotencyKey: "k", Kind: string(deployment.ActionApplicationStart)})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutorID != "someone.else" {
		t.Fatalf("a foreign executor id was overwritten with %q", res.ExecutorID)
	}
}

// The point of preserving the identity: ValidateResult can now reject a result
// from another executor. With the old overwrite it could never fire.
func TestAForeignExecutorIdentityIsRejectedByValidateResult(t *testing.T) {
	a := adapterWith(t, `{"executorId":"someone.else","actionId":"app/start","idempotencyKey":"k","outcome":"accepted"}`)
	cmd := executor.Command{ActionID: "app/start", IdempotencyKey: "k", Kind: string(deployment.ActionApplicationStart)}
	res, err := a.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	env := executorPlanEnv(t)
	if err := routing.ValidateResult(env, cmd, res); err == nil {
		t.Fatal("a result from another executor was accepted")
	}
}

func executorPlanEnv(t *testing.T) environment.Environment {
	t.Helper()
	return environment.Environment{
		ID: core.MustCanonicalID("prod"),
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action: deployment.ActionApplicationStart, Executor: core.MustCanonicalID("runtime.main"),
		}},
	}
}
