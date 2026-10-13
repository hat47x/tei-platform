package runtime_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// A different realization or Engine must be rejected before an accepted
// in-process registration can execute, even when logical Plugin ID is stable.
func TestInterpretedRuntimeReplayRequiresExactRealizationAndEngine(t *testing.T) {
	command := executor.Command{
		ActionID:       "app/start",
		IdempotencyKey: "exact-runtime-replay",
		Kind:           string(deployment.ActionApplicationStart),
	}
	for _, tc := range []struct {
		name   string
		change func(*pruntime.BuiltinRegistration)
		accept bool
	}{
		{name: "same realization and engine replay", accept: true},
		{
			name: "same realization with changed engine digest",
			change: func(r *pruntime.BuiltinRegistration) {
				r.RuntimeEngine.ImplementationDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			},
		},
		{
			name: "same engine with changed realization",
			change: func(r *pruntime.BuiltinRegistration) {
				r.RealizationDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
			},
		},
		{
			name: "same realization with changed engine target",
			change: func(r *pruntime.BuiltinRegistration) {
				r.RuntimeEngine.Target = "wasm/reference"
			},
		},
		{
			name: "same realization with changed engine contract",
			change: func(r *pruntime.BuiltinRegistration) {
				r.RuntimeEngine.ContractVersion = "tei.runtime-engine/other-v1"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newBuildBoundBuiltinFixture(t)
			calls := 0
			reg := fx.reg
			reg.Builtin = countingBuiltin{
				id:      fx.reg.Builtin.Identity(),
				fp:      fx.reg.Builtin.DescriptorFingerprint(),
				invoked: &calls,
			}
			if tc.change != nil {
				tc.change(&reg)
			}
			adapter := pruntime.DeploymentExecutor{
				Plan: fx.plan,
				Registry: pruntime.Registry{BuiltinRegistrations: map[string]pruntime.BuiltinRegistration{
					"tei.native.runtime.reference@1.0.0": reg,
				}},
				CapabilityID: core.MustCanonicalID("runtime.native_reference"),
			}
			if !tc.accept {
				if _, err := adapter.Execute(context.Background(), command); err == nil {
					t.Fatal("changed physical runtime identity was accepted")
				}
				if calls != 0 {
					t.Fatalf("unaccepted implementation executed %d times", calls)
				}
				return
			}
			for i := 1; i <= 2; i++ {
				result, err := adapter.Execute(context.Background(), command)
				if err != nil {
					t.Fatalf("identical replay %d: %v", i, err)
				}
				if calls != i {
					t.Fatalf("exact registration invoked %d times, expected %d", calls, i)
				}
				if result.PluginInvocation == nil {
					t.Fatal("exact replay omitted runtime invocation provenance")
				}
				fp, err := pruntime.RuntimeEngineIdentityFingerprint(reg.RuntimeEngine)
				if err != nil {
					t.Fatal(err)
				}
				if result.PluginInvocation.RuntimeEngineFingerprint != fp ||
					result.PluginInvocation.RealizationDigest != reg.RealizationDigest ||
					result.PluginInvocation.RuntimeEngineLockFingerprint != fx.plan.RuntimeEngineLockFingerprint {
					t.Fatal("exact replay lost realization or runtime engine identity")
				}
			}
		})
	}
}

// A logical Plugin and its exact realized bytes can remain unchanged while a
// target selects a different physical Engine. This tests identity binding,
// not actual execution of a WebAssembly binary.
func TestRuntimeEngineTargetChangesPhysicalLockNotLogicalPlugin(t *testing.T) {
	base := engineLockPlan(pruntime.BoundaryInProcess)
	identity := plugin.Identity{
		ID:      core.MustCanonicalID("tei.native.reference"),
		Version: "1.0.0",
	}
	engine := engineLockIdentity("tei.runtime.engine.reference")
	bind := func(target string) pruntime.EnvironmentPlan {
		t.Helper()
		chosen := engine
		chosen.Target = target
		lock := pruntime.RuntimeEngineLock{
			FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
			Bindings: []pruntime.RuntimeEngineBinding{{
				Plugin: identity,
				Engine: chosen,
			}},
		}
		plan, err := pruntime.BindRuntimeEngineLock(base, lock)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Plugins[0].Plugin != identity ||
			plan.Plugins[0].Artifact.Digest != base.Plugins[0].Artifact.Digest ||
			plan.Plugins[0].BuildPlanFingerprint != base.Plugins[0].BuildPlanFingerprint {
			t.Fatal("target selection altered logical Plugin identity or realization")
		}
		if plan.Plugins[0].RuntimeEngine == nil ||
			plan.Plugins[0].RuntimeEngine.Target != target {
			t.Fatal("target selection lost exact runtime engine")
		}
		return plan
	}
	host := bind("host/reference")
	wasm := bind("wasm/reference")
	if host.RuntimeEngineLockFingerprint == wasm.RuntimeEngineLockFingerprint ||
		host.RuntimePolicyFingerprint == wasm.RuntimePolicyFingerprint {
		t.Fatal("distinct physical Engine targets produced the same binding")
	}
}
