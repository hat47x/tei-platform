package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func main() {
	ctx := context.Background()
	pid := plugin.Identity{ID: core.MustCanonicalID("tei.migration.atlas"), Version: "1.0.0"}
	capability := plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("migration.atlas"), Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease}}
	plan := pruntime.EnvironmentPlan{
		Environment:              core.MustCanonicalID("production"),
		PluginLockFingerprint:    "example-plugin-lock",
		RuntimePolicyFingerprint: "example-runtime-policy",
		Plugins:                  []pruntime.PluginRuntime{{Plugin: pid, DescriptorFingerprint: "example-descriptor", Boundary: pruntime.BoundaryProcess, Artifact: plugin.Artifact{Digest: "sha256:example-atlas"}}},
		Capabilities:             []pruntime.CapabilityRuntime{{Plugin: pid, Capability: capability, Boundary: pruntime.BoundaryProcess}},
	}

	store := pruntime.NewMemoryInvocationHandleStore()
	handle := pruntime.DurableInvocationHandle{
		FormatVersion:            1,
		Revision:                 1,
		InvocationID:             "invocation-atlas-001",
		RunID:                    "deployment-run-001",
		ActionID:                 "sales_expand/apply",
		RequestID:                "request-atlas-001",
		Environment:              plan.Environment,
		Plugin:                   pid,
		CapabilityID:             capability.ID,
		DescriptorFingerprint:    "example-descriptor",
		PluginLockFingerprint:    plan.PluginLockFingerprint,
		RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint,
		ArtifactDigest:           "sha256:example-atlas",
		RuntimeKind:              "process",
		RuntimeHandle:            "runtime://reference/atlas-001",
		ResourceLeaseID:          "lease-atlas-001",
		SupervisorOwnerID:        "host-a",
		FencingToken:             1,
		Lifecycle:                pruntime.RuntimeRunning,
	}
	if _, err := store.Save(ctx, handle, 0); err != nil {
		panic(err)
	}

	rt := pruntime.NewReferenceReattachableRuntime()
	rt.Put(handle.RuntimeHandle, pruntime.RuntimeRunning, 1)

	result, err := pruntime.ReconcileDurableInvocation(ctx, store, rt, plan, handle.InvocationID, "host-b", 2)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		panic(err)
	}

	if _, err := rt.ReattachInvocation(ctx, handle.RuntimeHandle, 1); err != nil {
		fmt.Fprintln(os.Stderr, "stale host rejected:", err)
	}
}
