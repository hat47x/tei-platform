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
	if res.Handle.SupervisorOwnerID != "host-b" || res.Handle.FencingToken != 2 || res.Handle.Revision != 2 {
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
