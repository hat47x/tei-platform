package runtime_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

// Existing synthetic broker and sandbox fixtures intentionally remain the
// source of all runtime effects. No X-TEA Script or Owner business mapping is
// accepted by these boundary tests.
func cancellationRuntimePlan(t *testing.T) pruntime.EnvironmentPlan {
	t.Helper()
	perm := permission.Spec{
		Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint",
	}
	def := plugin.Definition{
		ID: cid("tei.runtime.cancel"), Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDeploymentExecutor, ID: cid("deploy.runtime.cancel"),
			Actions:     []deployment.ActionKind{deployment.ActionDatabaseRelease},
			Permissions: []permission.Spec{perm},
		}},
	}
	env := environment.Environment{
		ID:      cid("runtime_cancel_test"),
		Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}},
		PermissionGrants: []environment.PermissionGrant{{
			CapabilityKind: string(plugin.CapabilityDeploymentExecutor),
			CapabilityID:   cid("deploy.runtime.cancel"), Permission: perm,
		}},
		ResourceBindings: []environment.ResourceBinding{
			rb(permission.KindNetwork, "database-endpoint", "runtime_cancel_test.database_endpoint"),
		},
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

func TestCancelledRuntimeInvocationSkipsResourceAcquisition(t *testing.T) {
	plan := cancellationRuntimePlan(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	broker := &resourceBroker{}
	sandbox := &sandboxLauncher{}
	inv, err := (pruntime.Registry{Sandboxes: sandbox, Resources: broker}).InvokeWithProvenance(
		ctx, plan, plugin.CapabilityDeploymentExecutor, cid("deploy.runtime.cancel"),
		"execute", "pre-cancel", nil,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled invocation must propagate context.Canceled: inv=%#v err=%v", inv, err)
	}
	if broker.acquired != 0 || broker.released != 0 || sandbox.called {
		t.Fatalf("cancelled invocation crossed resource/dispatch boundary: broker=%+v invoked=%t", broker, sandbox.called)
	}
}

type cancellationOnAcquireBroker struct {
	resourceBroker
	cancel                   context.CancelFunc
	releaseContextUncanceled bool
}

func (b *cancellationOnAcquireBroker) Acquire(ctx context.Context, req resource.AcquireRequest) (resource.Lease, error) {
	l, err := b.resourceBroker.Acquire(ctx, req)
	if err == nil {
		b.cancel()
	}
	return l, err
}
func (b *cancellationOnAcquireBroker) Release(ctx context.Context, l resource.Lease) error {
	b.releaseContextUncanceled = ctx.Err() == nil
	return b.resourceBroker.Release(ctx, l)
}

func TestCancelledDuringRuntimeLeaseAcquisitionReleasesBeforeDispatch(t *testing.T) {
	plan := cancellationRuntimePlan(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	broker := &cancellationOnAcquireBroker{cancel: cancel}
	sandbox := &sandboxLauncher{}
	inv, err := (pruntime.Registry{Sandboxes: sandbox, Resources: broker}).InvokeWithProvenance(
		ctx, plan, plugin.CapabilityDeploymentExecutor, cid("deploy.runtime.cancel"),
		"execute", "cancel-on-acquire", nil,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled acquisition must propagate context.Canceled: inv=%#v err=%v", inv, err)
	}
	if sandbox.called || broker.acquired != 1 || broker.released != 1 {
		t.Fatalf("lease was used or leaked during cancellation: broker=%+v invoked=%t", broker, sandbox.called)
	}
	if !broker.releaseContextUncanceled || !inv.Provenance.ResourceLeaseReleased {
		t.Fatalf("cleanup after cancellation lacks uncancelled context or release evidence: inv=%#v broker=%+v", inv, broker)
	}
}
