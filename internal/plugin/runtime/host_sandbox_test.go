package runtime_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

type netEnforcer struct {
	called  bool
	cleaned bool
	policy  pruntime.NetworkPolicy
}

func (e *netEnforcer) ApplyNetworkPolicy(_ context.Context, p pruntime.NetworkPolicy) (pruntime.Cleanup, error) {
	e.called, e.policy = true, p
	return func(context.Context) error { e.cleaned = true; return nil }, nil
}

type fsEnforcer struct {
	called  bool
	cleaned bool
	policy  pruntime.FilesystemPolicy
}

func (e *fsEnforcer) ApplyFilesystemPolicy(_ context.Context, p pruntime.FilesystemPolicy) (pruntime.Cleanup, error) {
	e.called, e.policy = true, p
	return func(context.Context) error { e.cleaned = true; return nil }, nil
}

type secretInjector struct {
	called  bool
	cleaned bool
	got     []pruntime.SecretInjection
}

func (i *secretInjector) InjectSecrets(_ context.Context, in []pruntime.SecretInjection) (pruntime.Cleanup, error) {
	i.called = true
	i.got = append([]pruntime.SecretInjection(nil), in...)
	return func(context.Context) error { i.cleaned = true; return nil }, nil
}

type sandboxRunner struct{ called bool }

func (r *sandboxRunner) InvokeSandboxed(_ context.Context, _ pruntime.ProcessCommand, _ pruntime.Request) (pruntime.Response, error) {
	r.called = true
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

func TestHostSandboxLauncherAppliesNetworkMountAndSecretInjection(t *testing.T) {
	net := &netEnforcer{}
	fs := &fsEnforcer{}
	sec := &secretInjector{}
	runner := &sandboxRunner{}
	launcher := pruntime.HostSandboxLauncher{Network: net, Filesystem: fs, Secrets: sec, Runner: runner}
	spec := pruntime.SandboxSpec{Command: pruntime.ProcessCommand{Executable: "worker"}, Profile: pruntime.SandboxProfile{
		DenyNetworkByDefault: true, DenyFilesystemByDefault: true,
		NetworkAllowlist: []string{"db.internal:5432"},
		Mounts:           []pruntime.SandboxMount{{Scope: "migration-workdir", ResourceID: core.MustCanonicalID("prod.workdir"), SourceHandle: "mount://source", Target: "/tei/migration", Access: permission.AccessReadWrite}},
		SecretInjections: []pruntime.SecretInjection{{Scope: "database-credential", ResourceID: core.MustCanonicalID("prod.secret"), Handle: "vault://opaque", Delivery: resource.SecretDeliveryEnvironment, Target: "TEI_DATABASE_CREDENTIAL"}},
	}}
	if _, err := launcher.Invoke(context.Background(), spec, pruntime.Request{}); err != nil {
		t.Fatal(err)
	}
	if !net.called || !fs.called || !sec.called || !runner.called || !net.cleaned || !fs.cleaned || !sec.cleaned {
		t.Fatalf("host enforcement lifecycle incomplete: net=%+v fs=%+v secret=%+v runner=%+v", net, fs, sec, runner)
	}
	if !net.policy.DenyByDefault || len(net.policy.Allowlist) != 1 || net.policy.Allowlist[0] != "db.internal:5432" {
		t.Fatalf("network policy: %#v", net.policy)
	}
	if len(fs.policy.Mounts) != 1 || fs.policy.Mounts[0].Access != permission.AccessReadWrite {
		t.Fatalf("filesystem policy: %#v", fs.policy)
	}
	if len(sec.got) != 1 || sec.got[0].Delivery != resource.SecretDeliveryEnvironment || sec.got[0].Target != "TEI_DATABASE_CREDENTIAL" {
		t.Fatalf("secret injection: %#v", sec.got)
	}
}

type renewingSandbox struct{ renewed resource.Lease }

func (s *renewingSandbox) Invoke(ctx context.Context, spec pruntime.SandboxSpec, _ pruntime.Request) (pruntime.Response, error) {
	if spec.LeaseController == nil {
		return pruntime.Response{}, context.Canceled
	}
	lease, err := spec.LeaseController.Renew(ctx)
	if err != nil {
		return pruntime.Response{}, err
	}
	s.renewed = lease
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

func TestSandboxLauncherCanExplicitlyRenewSameAuthorityLease(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	def := plugin.Definition{ID: cid("tei.long.migration"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("migration.long"), Permissions: []permission.Spec{perm}}}}
	binding := environment.ResourceBinding{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("prod.long.secret"), SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL"}
	env := environment.Environment{ID: cid("prod_long"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("migration.long"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{binding}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, err := plugin.BuildLock(cat, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 29, 15, 0, 0, 0, time.UTC)
	adapter := &resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: binding.ResourceID, SecretHandle: "vault://opaque"}}}
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{adapter}, TTL: time.Minute, Now: func() time.Time { now = now.Add(time.Minute); return now }}
	sb := &renewingSandbox{}
	inv, err := (pruntime.Registry{Sandboxes: sb, Resources: broker}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, cid("migration.long"), "execute", "renew-test", map[string]string{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Provenance.ResourceLeaseRevision != 2 || len(inv.Provenance.ResourceLeaseRenewals) != 1 || !inv.Provenance.ResourceLeaseReleased {
		t.Fatalf("renewal provenance: %#v", inv.Provenance)
	}
	if sb.renewed.Resources[0].SecretHandle != "vault://opaque" {
		t.Fatalf("in-place renewal changed concrete secret handle: %#v", sb.renewed)
	}
	if broker.ActiveLeaseCount() != 0 {
		t.Fatal("renewed lease was not released")
	}
}

func TestHostSandboxLauncherRejectsMissingDefaultDenyEnforcer(t *testing.T) {
	launcher := pruntime.HostSandboxLauncher{Runner: &sandboxRunner{}}
	_, err := launcher.Invoke(context.Background(), pruntime.SandboxSpec{Profile: pruntime.SandboxProfile{DenyNetworkByDefault: true, DenyFilesystemByDefault: true}}, pruntime.Request{})
	if err == nil {
		t.Fatal("expected missing host network/filesystem enforcer rejection")
	}
}


func TestZeroPermissionSandboxedCapabilityDoesNotStartLeaseSupervisor(t *testing.T) {
	def := plugin.Definition{
		ID:      cid("tei.zero_permission"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "worker",
			Sandbox: plugin.SandboxRequired,
		},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDeploymentExecutor,
			ID:   cid("zero.permission"),
		}},
	}
	env := environment.Environment{
		ID: cid("zero_permission_env"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
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

	runner := &sandboxRunner{}
	registry := pruntime.Registry{
		Sandboxes: pruntime.HostSandboxLauncher{
			Network:    &netEnforcer{},
			Filesystem: &fsEnforcer{},
			Runner:     runner,
		},
	}
	inv, err := registry.InvokeWithProvenance(
		context.Background(),
		plan,
		plugin.CapabilityDeploymentExecutor,
		cid("zero.permission"),
		"execute",
		"zero-permission-sandbox",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !runner.called || !inv.Provenance.Sandboxed {
		t.Fatalf("zero-permission sandbox invocation did not complete: %#v", inv.Provenance)
	}
	if inv.Provenance.ResourceLeaseID != "" {
		t.Fatalf("zero-permission capability unexpectedly acquired lease %q", inv.Provenance.ResourceLeaseID)
	}
}

func TestRuntimePolicyFingerprintIncludesSecretInjectionTarget(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	def := plugin.Definition{ID: cid("tei.inject.target"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("inject.target"), Permissions: []permission.Spec{perm}}}}
	mk := func(target string) environment.Environment {
		return environment.Environment{ID: cid("inject_prod"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("inject.target"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("inject_prod.secret"), SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: target}}}
	}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	envA := mk("TEI_DATABASE_CREDENTIAL")
	lock, err := plugin.BuildLock(cat, []environment.Environment{envA})
	if err != nil {
		t.Fatal(err)
	}
	planA, err := pruntime.BuildPlan(cat, envA, lock)
	if err != nil {
		t.Fatal(err)
	}
	envB := mk("OTHER_CREDENTIAL_TARGET")
	planB, err := pruntime.BuildPlan(cat, envB, lock)
	if err != nil {
		t.Fatal(err)
	}
	if planA.RuntimePolicyFingerprint == planB.RuntimePolicyFingerprint {
		t.Fatal("secret injection target change did not change runtime policy fingerprint")
	}
}

type refreshSecretInjector struct {
	secretInjector
	refreshed bool
	from, to  []pruntime.SecretInjection
	fail      bool
}

func (i *refreshSecretInjector) RefreshSecrets(_ context.Context, from, to []pruntime.SecretInjection) error {
	i.refreshed = true
	i.from = append([]pruntime.SecretInjection(nil), from...)
	i.to = append([]pruntime.SecretInjection(nil), to...)
	if i.fail {
		return context.Canceled
	}
	return nil
}

type refreshRunner struct{ ignored bool }

func (r *refreshRunner) InvokeSandboxed(ctx context.Context, _ pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	if req.LeaseController == nil {
		return pruntime.Response{}, context.Canceled
	}
	_, err := req.LeaseController.Refresh(ctx)
	if err != nil {
		r.ignored = true
	}
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

func TestLiveSecretRefreshIsExplicitAndAppliedBySandbox(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	def := plugin.Definition{ID: cid("tei.refresh.secret"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("migration.refresh"), Permissions: []permission.Spec{perm}}}}
	binding := environment.ResourceBinding{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("prod.refresh.secret"), SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL", Refresh: resource.RefreshLive}
	env := environment.Environment{ID: cid("prod_refresh"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("migration.refresh"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{binding}, LeasePolicies: []environment.CapabilityLeasePolicy{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("migration.refresh"), Policy: resource.LeasePolicy{MaxLifetime: "20m", RenewBefore: "3m", RevokeTimeout: "2s", RenewalFailure: resource.RenewalFailClosed}}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 29, 17, 0, 0, 0, time.UTC)
	adapter := &resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: binding.ResourceID, SecretHandle: "vault://v1"}}, RefreshSuffix: "#v"}
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{adapter}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	sec := &refreshSecretInjector{}
	runner := &refreshRunner{}
	host := &pruntime.HostSandboxLauncher{Network: &netEnforcer{}, Filesystem: &fsEnforcer{}, Secrets: sec, Runner: runner, Now: func() time.Time { return now }}
	inv, err := (pruntime.Registry{Sandboxes: host, Resources: broker}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, cid("migration.refresh"), "execute", "refresh-test", map[string]string{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	if !sec.refreshed || len(sec.from) != 1 || len(sec.to) != 1 || sec.from[0].Handle == sec.to[0].Handle {
		t.Fatalf("refresh not applied: %#v", sec)
	}
	if inv.Provenance.ResourceLeaseRevision != 2 || len(inv.Provenance.ResourceLeaseRefreshes) != 1 || inv.Provenance.ResourceLeaseExpiresAt == "" || !inv.Provenance.ResourceLeaseReleased {
		t.Fatalf("provenance=%#v", inv.Provenance)
	}
	if broker.ActiveLeaseCount() != 0 {
		t.Fatal("refreshed lease not released")
	}
}

func TestLiveRefreshEnforcementFailureIsFailClosedEvenIfPluginIgnoresIt(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "database-credential"}
	def := plugin.Definition{ID: cid("tei.refresh.fail"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("migration.refresh.fail"), Permissions: []permission.Spec{perm}}}}
	binding := environment.ResourceBinding{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("prod.refresh.fail.secret"), SecretDelivery: resource.SecretDeliveryEnvironment, SecretTarget: "TEI_DATABASE_CREDENTIAL", Refresh: resource.RefreshLive}
	env := environment.Environment{ID: cid("prod_refresh_fail"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("migration.refresh.fail"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{binding}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	now := time.Date(2026, 8, 29, 17, 0, 0, 0, time.UTC)
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{&resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: binding.ResourceID, SecretHandle: "vault://v1"}}, RefreshSuffix: "#v"}}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	sec := &refreshSecretInjector{fail: true}
	runner := &refreshRunner{}
	host := &pruntime.HostSandboxLauncher{Network: &netEnforcer{}, Filesystem: &fsEnforcer{}, Secrets: sec, Runner: runner, Now: func() time.Time { return now }}
	_, err := (pruntime.Registry{Sandboxes: host, Resources: broker}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, cid("migration.refresh.fail"), "execute", "refresh-fail", nil)
	if err == nil || !runner.ignored {
		t.Fatalf("expected fail-closed refresh error; ignored=%v err=%v", runner.ignored, err)
	}
	if broker.ActiveLeaseCount() != 0 {
		t.Fatal("latest refreshed lease was not revoked after fail-closed error")
	}
}

type ignoringRenewRunner struct{}

func (r *ignoringRenewRunner) InvokeSandboxed(ctx context.Context, _ pruntime.ProcessCommand, req pruntime.Request) (pruntime.Response, error) {
	_, _ = req.LeaseController.Renew(ctx)
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

func TestRenewalFailureIsFailClosedEvenWhenPluginIgnoresError(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "credential"}
	def := plugin.Definition{ID: cid("tei.renew.failclosed"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("renew.failclosed"), Permissions: []permission.Spec{perm}}}}
	binding := environment.ResourceBinding{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("prod.renew.secret")}
	policy := resource.LeasePolicy{MaxLifetime: "20m", RenewBefore: "2m", RevokeTimeout: "2s", RenewalFailure: resource.RenewalFailClosed}
	env := environment.Environment{ID: cid("prod_renew_fail"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("renew.failclosed"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{binding}, LeasePolicies: []environment.CapabilityLeasePolicy{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("renew.failclosed"), Policy: policy}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	now := time.Date(2026, 8, 29, 17, 0, 0, 0, time.UTC)
	broker := &resource.AdapterBroker{Adapters: []resource.Adapter{&resource.StaticAdapter{ResourceKind: permission.KindSecret, Entries: []resource.StaticResource{{ResourceID: binding.ResourceID, SecretHandle: "opaque"}}}}, TTL: 10 * time.Minute, Now: func() time.Time { return now }}
	host := &pruntime.HostSandboxLauncher{Network: &netEnforcer{}, Filesystem: &fsEnforcer{}, Secrets: &secretInjector{}, Runner: &ignoringRenewRunner{}}
	_, err := (pruntime.Registry{Sandboxes: host, Resources: broker}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, cid("renew.failclosed"), "execute", "renew-too-early", nil)
	if err == nil {
		t.Fatal("expected ignored renewal failure to fail invocation")
	}
	if broker.ActiveLeaseCount() != 0 {
		t.Fatal("lease not revoked after renewal fail-closed")
	}
}

func TestRuntimePolicyFingerprintIncludesLeasePolicyAndRefreshMode(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "credential"}
	def := plugin.Definition{ID: cid("tei.policy.fp"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("policy.fp"), Permissions: []permission.Spec{perm}}}}
	mk := func(refresh resource.RefreshMode, max string) environment.Environment {
		return environment.Environment{ID: cid("policy_env"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("policy.fp"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("policy_env.secret"), Refresh: refresh}}, LeasePolicies: []environment.CapabilityLeasePolicy{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("policy.fp"), Policy: resource.LeasePolicy{MaxLifetime: max, RenewBefore: "2m", RevokeTimeout: "2s", RenewalFailure: resource.RenewalFailClosed}}}}
	}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	a := mk(resource.RefreshNone, "20m")
	lock, _ := plugin.BuildLock(cat, []environment.Environment{a})
	pa, _ := pruntime.BuildPlan(cat, a, lock)
	b := mk(resource.RefreshLive, "20m")
	pb, _ := pruntime.BuildPlan(cat, b, lock)
	c := mk(resource.RefreshNone, "30m")
	pc, _ := pruntime.BuildPlan(cat, c, lock)
	if pa.RuntimePolicyFingerprint == pb.RuntimePolicyFingerprint {
		t.Fatal("refresh mode did not affect runtime policy fingerprint")
	}
	if pa.RuntimePolicyFingerprint == pc.RuntimePolicyFingerprint {
		t.Fatal("lease max-lifetime did not affect runtime policy fingerprint")
	}
}

type failingSupervisorController struct{ lease resource.Lease }

func (c *failingSupervisorController) Current() resource.Lease { return c.lease }
func (c *failingSupervisorController) Renew(context.Context) (resource.Lease, error) {
	return resource.Lease{}, fmt.Errorf("host renewal unavailable")
}
func (c *failingSupervisorController) Refresh(context.Context) (resource.Lease, error) {
	return resource.Lease{}, fmt.Errorf("refresh not used")
}

type cancelAwareRunner struct{ canceled chan struct{} }

func (r *cancelAwareRunner) InvokeSandboxed(ctx context.Context, _ pruntime.ProcessCommand, _ pruntime.Request) (pruntime.Response, error) {
	<-ctx.Done()
	close(r.canceled)
	return pruntime.Response{}, ctx.Err()
}

func TestHostLeaseSupervisorCancelsInvocationWhenAutomaticRenewalFails(t *testing.T) {
	now := time.Now().UTC()
	ctrl := &failingSupervisorController{lease: resource.Lease{ID: "lease-supervisor-fail", Revision: 1, IssuedAt: now.Format(time.RFC3339Nano), ExpiresAt: now.Add(80 * time.Millisecond).Format(time.RFC3339Nano)}}
	runner := &cancelAwareRunner{canceled: make(chan struct{})}
	host := pruntime.HostSandboxLauncher{Network: &netEnforcer{}, Filesystem: &fsEnforcer{}, Secrets: &secretInjector{}, Runner: runner}
	spec := pruntime.SandboxSpec{Command: pruntime.ProcessCommand{Executable: "reference"}, LeaseController: ctrl, LeasePolicy: resource.LeasePolicy{MaxLifetime: "1s", RenewBefore: "50ms", RevokeTimeout: "1s", RenewalFailure: resource.RenewalFailClosed}, Profile: pruntime.SandboxProfile{DenyNetworkByDefault: true, DenyFilesystemByDefault: true}}
	_, err := host.Invoke(context.Background(), spec, pruntime.Request{LeaseController: ctrl})
	if err == nil {
		t.Fatal("expected supervised renewal failure")
	}
	select {
	case <-runner.canceled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("runner context was not canceled by supervisor")
	}
}
