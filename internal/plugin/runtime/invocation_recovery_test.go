package runtime_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func recoveryPlan() pruntime.EnvironmentPlan {
	p := plugin.Identity{ID: core.MustCanonicalID("tei.migration.atlas"), Version: "1.0.0"}
	c := plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("migration.atlas"), Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease}}
	return pruntime.EnvironmentPlan{
		Environment:              core.MustCanonicalID("production"),
		PluginLockFingerprint:    "lock-fp",
		RuntimePolicyFingerprint: "policy-fp",
		Plugins:                  []pruntime.PluginRuntime{{Plugin: p, DescriptorFingerprint: "descriptor-fp", Boundary: pruntime.BoundaryProcess, Artifact: plugin.Artifact{Digest: "sha256:atlas"}}},
		Capabilities:             []pruntime.CapabilityRuntime{{Plugin: p, Capability: c, Boundary: pruntime.BoundaryProcess}},
	}
}

func recoveryHandle() pruntime.DurableInvocationHandle {
	return pruntime.DurableInvocationHandle{
		FormatVersion:            1,
		Revision:                 1,
		InvocationID:             "inv-1",
		RunID:                    "deploy-1",
		ActionID:                 "sales_expand/apply",
		RequestID:                "req-1",
		Environment:              core.MustCanonicalID("production"),
		Plugin:                   plugin.Identity{ID: core.MustCanonicalID("tei.migration.atlas"), Version: "1.0.0"},
		CapabilityID:             core.MustCanonicalID("migration.atlas"),
		DescriptorFingerprint:    "descriptor-fp",
		PluginLockFingerprint:    "lock-fp",
		RuntimePolicyFingerprint: "policy-fp",
		ArtifactDigest:           "sha256:atlas",
		RuntimeKind:              "process",
		RuntimeHandle:            "runtime://atlas/42",
		ResourceLeaseID:          "lease-1",
		SupervisorOwnerID:        "host-a",
		FencingToken:             1,
		RuntimeFencingToken:      1,
		Lifecycle:                pruntime.RuntimeRunning,
	}
}

func TestInvocationRecoveryReattachesWithNewFencingToken(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	// Seed through Save so revision semantics are exercised.
	h.Revision = 1
	if _, err := store.Save(ctx, h, 0); err != nil {
		t.Fatal(err)
	}
	rt := pruntime.NewReferenceReattachableRuntime()
	rt.Put(h.RuntimeHandle, pruntime.RuntimeRunning, 1)

	res, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != pruntime.RecoveryReattached {
		t.Fatalf("outcome=%s", res.Outcome)
	}
	if res.Handle.SupervisorOwnerID != "host-b" || res.Handle.FencingToken != 2 || res.Handle.RuntimeFencingToken != 2 || res.Handle.Revision != 2 {
		t.Fatalf("handle=%#v", res.Handle)
	}
	// Old host cannot retake authority with token 1 after runtime fencing moved to 2.
	if _, err := rt.ReattachInvocation(ctx, h.RuntimeHandle, 1); err == nil {
		t.Fatal("expected stale runtime fencing token rejection")
	}
}

func TestInvocationRecoveryDoesNotReplayMissingRuntime(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	if _, err := store.Save(ctx, h, 0); err != nil {
		t.Fatal(err)
	}
	rt := pruntime.NewReferenceReattachableRuntime() // handle deliberately absent

	res, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != pruntime.RecoveryRuntimeLost || res.Handle.Lifecycle != pruntime.RuntimeMissing {
		t.Fatalf("result=%#v", res)
	}
	if len(res.Findings) == 0 {
		t.Fatal("expected explicit no-replay finding")
	}
}

func TestInvocationRecoveryRejectsChangedRuntimePolicy(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	if _, err := store.Save(ctx, h, 0); err != nil {
		t.Fatal(err)
	}
	rt := pruntime.NewReferenceReattachableRuntime()
	rt.Put(h.RuntimeHandle, pruntime.RuntimeRunning, 1)
	plan := recoveryPlan()
	plan.RuntimePolicyFingerprint = "changed"
	if _, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, plan, h.InvocationID, "host-b", 2); err == nil {
		t.Fatal("expected runtime policy mismatch rejection")
	}
}

func TestInvocationHandleStoreUsesOptimisticConcurrency(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	saved, err := store.Save(ctx, h, 0)
	if err != nil {
		t.Fatal(err)
	}
	saved.LastMessage = "first"
	if _, err := store.Save(ctx, saved, 1); err != nil {
		t.Fatal(err)
	}
	saved.LastMessage = "stale"
	if _, err := store.Save(ctx, saved, 1); err == nil {
		t.Fatal("expected revision conflict")
	}
}

type observingRecoveryRuntime struct {
	observation         pruntime.RuntimeObservation
	reattachObservation pruntime.RuntimeObservation
	observeCalls        int
	reattachCalls       int
}

func (r *observingRecoveryRuntime) ObserveInvocation(context.Context, string) (pruntime.RuntimeObservation, error) {
	r.observeCalls++
	return r.observation, nil
}

func (r *observingRecoveryRuntime) ReattachInvocation(context.Context, string, uint64) (pruntime.RuntimeObservation, error) {
	r.reattachCalls++
	return r.reattachObservation, nil
}

func TestInvocationRecoveryObservesNonRunningStatesWithoutReattach(t *testing.T) {
	cases := []struct {
		name        string
		lifecycle   pruntime.RuntimeLifecycle
		want        pruntime.InvocationRecoveryOutcome
		wantFinding bool
	}{
		{name: "succeeded", lifecycle: pruntime.RuntimeSucceeded, want: pruntime.RecoveryAlreadySucceeded},
		{name: "failed", lifecycle: pruntime.RuntimeFailed, want: pruntime.RecoveryAlreadyFailed},
		{name: "unknown", lifecycle: pruntime.RuntimeUnknown, want: pruntime.RecoveryWaitingObserve, wantFinding: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := pruntime.NewMemoryInvocationHandleStore()
			h := recoveryHandle()
			if _, err := store.Save(ctx, h, 0); err != nil {
				t.Fatal(err)
			}
			rt := &observingRecoveryRuntime{observation: pruntime.RuntimeObservation{Lifecycle: tc.lifecycle}}
			res, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
			if err != nil {
				t.Fatal(err)
			}
			if res.Outcome != tc.want || res.Handle.Lifecycle != tc.lifecycle {
				t.Fatalf("result=%#v", res)
			}
			if rt.reattachCalls != 0 {
				t.Fatalf("reattach calls=%d", rt.reattachCalls)
			}
			if tc.wantFinding && len(res.Findings) == 0 {
				t.Fatal("expected unresolved-state finding")
			}
		})
	}
}

func TestInvocationRecoveryReattachesStartingRuntime(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	if _, err := store.Save(ctx, h, 0); err != nil {
		t.Fatal(err)
	}
	rt := &observingRecoveryRuntime{
		observation:         pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeStarting},
		reattachObservation: pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeStarting},
	}
	res, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != pruntime.RecoveryReattached || rt.reattachCalls != 1 {
		t.Fatalf("result=%#v reattachCalls=%d", res, rt.reattachCalls)
	}
}

func TestInvocationRecoveryDoesNotReplayRuntimeLostDuringReattach(t *testing.T) {
	ctx := context.Background()
	store := pruntime.NewMemoryInvocationHandleStore()
	h := recoveryHandle()
	if _, err := store.Save(ctx, h, 0); err != nil {
		t.Fatal(err)
	}
	rt := &observingRecoveryRuntime{
		observation:         pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeRunning},
		reattachObservation: pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeMissing},
	}
	res, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, "host-b", 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != pruntime.RecoveryRuntimeLost || len(res.Findings) == 0 || rt.reattachCalls != 1 {
		t.Fatalf("result=%#v reattachCalls=%d", res, rt.reattachCalls)
	}
}

func TestInvocationRecoveryRejectsChangedExecutionIdentityBeforeObservation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*pruntime.EnvironmentPlan)
	}{
		{name: "environment", mutate: func(p *pruntime.EnvironmentPlan) { p.Environment = core.MustCanonicalID("staging") }},
		{name: "plugin-lock", mutate: func(p *pruntime.EnvironmentPlan) { p.PluginLockFingerprint = "changed-lock" }},
		{name: "plugin", mutate: func(p *pruntime.EnvironmentPlan) {
			p.Capabilities[0].Plugin = plugin.Identity{ID: core.MustCanonicalID("tei.migration.other"), Version: "1.0.0"}
		}},
		{name: "descriptor", mutate: func(p *pruntime.EnvironmentPlan) { p.Plugins[0].DescriptorFingerprint = "changed-descriptor" }},
		{name: "artifact", mutate: func(p *pruntime.EnvironmentPlan) { p.Plugins[0].Artifact.Digest = "sha256:changed" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := pruntime.NewMemoryInvocationHandleStore()
			h := recoveryHandle()
			if _, err := store.Save(ctx, h, 0); err != nil {
				t.Fatal(err)
			}
			plan := recoveryPlan()
			tc.mutate(&plan)
			rt := &observingRecoveryRuntime{observation: pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeRunning}}
			if _, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, plan, h.InvocationID, "host-b", 2); err == nil {
				t.Fatal("expected execution identity mismatch")
			}
			if rt.observeCalls != 0 || rt.reattachCalls != 0 {
				t.Fatalf("runtime was touched before identity validation: observe=%d reattach=%d", rt.observeCalls, rt.reattachCalls)
			}
		})
	}
}

func TestInvocationRecoveryRejectsStaleOrConflictingHandleOwnershipBeforeObservation(t *testing.T) {
	cases := []struct {
		name    string
		ownerID string
		token   uint64
	}{
		{name: "stale-token", ownerID: "host-b", token: 1},
		{name: "same-token-different-owner", ownerID: "host-b", token: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := pruntime.NewMemoryInvocationHandleStore()
			h := recoveryHandle()
			h.FencingToken = 2
			h.SupervisorOwnerID = "host-a"
			if _, err := store.Save(ctx, h, 0); err != nil {
				t.Fatal(err)
			}
			rt := &observingRecoveryRuntime{observation: pruntime.RuntimeObservation{Lifecycle: pruntime.RuntimeRunning}}
			if _, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, recoveryPlan(), h.InvocationID, tc.ownerID, tc.token); err == nil {
				t.Fatal("expected ownership conflict")
			}
			if rt.observeCalls != 0 || rt.reattachCalls != 0 {
				t.Fatalf("runtime was touched before ownership validation: observe=%d reattach=%d", rt.observeCalls, rt.reattachCalls)
			}
		})
	}
}
