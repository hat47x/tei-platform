package runtime_test

import (
	"context"
	"encoding/json"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

type builtin struct {
	id plugin.Identity
	fp string
}

func (b builtin) Identity() plugin.Identity     { return b.id }
func (b builtin) DescriptorFingerprint() string { return b.fp }
func (b builtin) Invoke(_ context.Context, req pruntime.Request) (pruntime.Response, error) {
	result := executor.Result{AttemptID: "a1", Outcome: executor.OutcomeAccepted}
	raw, _ := json.Marshal(result)
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: raw}, nil
}

type launcher struct{ called bool }

func (l *launcher) Invoke(_ context.Context, command string, req pruntime.Request) (pruntime.Response, error) {
	l.called = true
	raw, _ := json.Marshal(map[string]any{"ok": true, "command": command})
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: raw}, nil
}

func cid(s string) core.CanonicalID { return core.MustCanonicalID(s) }

type resourceBroker struct {
	acquired int
	released int
}

func (b *resourceBroker) Acquire(_ context.Context, req resource.AcquireRequest) (resource.Lease, error) {
	b.acquired++
	l := resource.Lease{ID: "lease-test", Revision: 1, ExpiresAt: "2026-08-29T16:00:00+09:00", RuntimePolicyFingerprint: req.RuntimePolicyFingerprint}
	byKey := map[string]resource.Binding{}
	for _, x := range req.Bindings {
		byKey[x.Key()] = x
	}
	for _, p := range req.Permissions {
		r := resource.Resolved{Kind: p.Kind, Access: p.Access, Scope: p.Scope}
		binding := byKey[string(p.Kind)+"|"+p.Scope]
		r.ResourceID = binding.ResourceID
		r.SecretDelivery = binding.SecretDelivery
		r.SecretTarget = binding.SecretTarget
		switch p.Kind {
		case permission.KindSecret:
			r.SecretHandle = "secret-handle://lease-test"
		case permission.KindNetwork:
			r.NetworkEndpoint = "db.example.internal:5432"
		case permission.KindFilesystem:
			r.MountPath = "/tei/lease/workspace-source"
		}
		l.Resources = append(l.Resources, r)
	}
	return l, nil
}
func (b *resourceBroker) Release(_ context.Context, _ resource.Lease) error { b.released++; return nil }

func rb(kind permission.Kind, scope, id string) environment.ResourceBinding {
	return environment.ResourceBinding{Kind: kind, Scope: scope, ResourceID: cid(id)}
}

func TestRuntimePlanDerivesTrustBoundaryFromLockedDescriptor(t *testing.T) {
	p1 := plugin.Definition{ID: cid("tei.runtime"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.main"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}}}}
	p2 := plugin.Definition{ID: cid("tei.sql"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "python3 -m worker"}, Provides: []plugin.Capability{{Kind: plugin.CapabilityEditorIntelligence, ID: cid("editor.sql"), Features: []environment.Feature{environment.FeatureSQLAnalysis}}}}
	env := environment.Environment{ID: cid("dev"), Plugins: []environment.ExtensionRequirement{{ID: p1.ID, CompatibleMajor: 1}, {ID: p2.ID, CompatibleMajor: 1}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{p1, p2}}
	lock, err := plugin.BuildLock(cat, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Plugins) != 2 || plan.PluginLockFingerprint == "" {
		t.Fatalf("plan: %#v", plan)
	}
	var processFound bool
	for _, p := range plan.Plugins {
		if p.Plugin.ID == p2.ID {
			processFound = true
			if p.Boundary != pruntime.BoundaryProcess || p.Trust != pruntime.TrustExternalProcess {
				t.Fatalf("process boundary: %#v", p)
			}
		}
	}
	if !processFound {
		t.Fatal("missing process plugin")
	}
}

func TestBuiltinDeploymentAdapterUsesExactLockedPlugin(t *testing.T) {
	def := plugin.Definition{ID: cid("tei.runtime"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.main"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}}}}
	env := environment.Environment{ID: cid("prod"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	reg := pruntime.Registry{Builtins: map[string]pruntime.Builtin{"tei.runtime@1.0.0": builtin{id: plugin.Identity{ID: def.ID, Version: def.Version}, fp: plugin.DefinitionFingerprint(def)}}}
	a := pruntime.DeploymentExecutor{Plan: plan, Registry: reg, CapabilityID: cid("runtime.main")}
	res, err := a.Execute(context.Background(), executor.Command{ActionID: "app/start", IdempotencyKey: "k", Kind: string(deployment.ActionApplicationStart)})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutorID != "runtime.main" || res.ActionID != "app/start" || res.IdempotencyKey != "k" {
		t.Fatalf("result: %#v", res)
	}
	if res.PluginInvocation == nil || res.PluginInvocation.PluginID != def.ID || res.PluginInvocation.PluginVersion != def.Version || res.PluginInvocation.PluginLockFingerprint == "" {
		t.Fatalf("plugin invocation provenance: %#v", res.PluginInvocation)
	}
}

func TestProcessPluginRequiresHostLauncher(t *testing.T) {
	def := plugin.Definition{ID: cid("tei.sql"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "python3 -m worker"}, Provides: []plugin.Capability{{Kind: plugin.CapabilityEditorIntelligence, ID: cid("editor.sql"), Features: []environment.Feature{environment.FeatureSQLAnalysis}}}}
	env := environment.Environment{ID: cid("dev"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	_, err := (pruntime.Registry{}).Invoke(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.sql"), "analyze", "r1", map[string]string{"sql": "select 1"})
	if err == nil {
		t.Fatal("expected missing process launcher")
	}
	l := &launcher{}
	_, err = (pruntime.Registry{Processes: l}).Invoke(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.sql"), "analyze", "r1", map[string]string{"sql": "select 1"})
	if err != nil || !l.called {
		t.Fatalf("launcher: %v %v", l.called, err)
	}
}

func TestBuiltinDescriptorFingerprintMismatchIsRejected(t *testing.T) {
	def := plugin.Definition{ID: cid("tei.runtime.fp"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.fp"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}}}}
	env := environment.Environment{ID: cid("prod_fp"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	reg := pruntime.Registry{Builtins: map[string]pruntime.Builtin{"tei.runtime.fp@1.0.0": builtin{id: plugin.Identity{ID: def.ID, Version: def.Version}, fp: "wrong"}}}
	a := pruntime.DeploymentExecutor{Plan: plan, Registry: reg, CapabilityID: cid("runtime.fp")}
	if _, err := a.Execute(context.Background(), executor.Command{ActionID: "app/start", IdempotencyKey: "k", Kind: string(deployment.ActionApplicationStart)}); err == nil {
		t.Fatal("expected descriptor fingerprint mismatch")
	}
}

type sandboxLauncher struct {
	called bool
	got    pruntime.SandboxSpec
}

func (s *sandboxLauncher) Invoke(_ context.Context, spec pruntime.SandboxSpec, req pruntime.Request) (pruntime.Response, error) {
	s.called = true
	s.got = spec
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

type verifier struct{ called bool }

func (v *verifier) Verify(_ context.Context, _ plugin.Identity, _ plugin.Artifact) (pruntime.ArtifactVerification, error) {
	v.called = true
	return pruntime.ArtifactVerification{VerifierID: "test-verifier", DigestVerified: true, SignatureVerified: true, ProvenanceChecked: true}, nil
}

func TestCapabilityPermissionLeaseSandboxAndArtifactProvenance(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "workspace-source"}
	def := plugin.Definition{
		ID: cid("tei.sql.secure"), Version: "1.0.0",
		Runtime:  plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "python3 -m worker", Sandbox: plugin.SandboxRequired},
		Artifact: plugin.Artifact{Digest: "sha256:5af655bd19fffee7c3564b060f92056e1a6bd521c0b15bd1155dd244cd77de43", SignatureRef: "sigstore://test", Provenance: "source://test"},
		Provides: []plugin.Capability{{Kind: plugin.CapabilityEditorIntelligence, ID: cid("editor.sql.secure"), Features: []environment.Feature{environment.FeatureSQLAnalysis}, Permissions: []permission.Spec{perm}}},
	}
	env := environment.Environment{ID: cid("secure_dev"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityEditorIntelligence), CapabilityID: cid("editor.sql.secure"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{rb(permission.KindFilesystem, "workspace-source", "secure_dev.workspace_source")}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, err := plugin.BuildLock(cat, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	sb := &sandboxLauncher{}
	vf := &verifier{}
	broker := &resourceBroker{}
	inv, err := (pruntime.Registry{Sandboxes: sb, Verifier: vf, Resources: broker}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.sql.secure"), "analyze", "r-secure", map[string]string{"sql": "select 1"})
	if err != nil {
		t.Fatal(err)
	}
	if !sb.called || !vf.called || !inv.Provenance.Sandboxed {
		t.Fatalf("sandbox/verifier/provenance: %#v", inv.Provenance)
	}
	if len(inv.Provenance.Permissions) != 1 || inv.Provenance.Permissions[0].Key() != perm.Key() {
		t.Fatalf("permissions: %#v", inv.Provenance.Permissions)
	}
	if inv.Provenance.Verification == nil || !inv.Provenance.Verification.SignatureVerified {
		t.Fatalf("verification: %#v", inv.Provenance.Verification)
	}
	if len(sb.got.Permissions) != 1 || sb.got.ResourceLease == nil || len(sb.got.Profile.Mounts) != 1 {
		t.Fatalf("sandbox spec: %#v", sb.got)
	}
	if broker.acquired != 1 || broker.released != 1 || !inv.Provenance.ResourceLeaseReleased {
		t.Fatalf("lease lifecycle: broker=%+v provenance=%+v", broker, inv.Provenance)
	}
	if got := inv.Provenance.ResourceLeaseSupervision; len(got) == 0 || got[len(got)-1].State != resource.SupervisionRevoked {
		t.Fatalf("lease supervision: %#v", got)
	}
}

func TestMissingPermissionGrantAndRequiredSandboxAreRejected(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "named-secret"}
	def := plugin.Definition{ID: cid("tei.secure.proc"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityEditorIntelligence, ID: cid("editor.secure"), Features: []environment.Feature{environment.FeatureSQLAnalysis}, Permissions: []permission.Spec{perm}}}}
	env := environment.Environment{ID: cid("dev_secure"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	if _, err := (pruntime.Registry{Processes: &launcher{}}).Invoke(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.secure"), "x", "r", nil); err == nil {
		t.Fatal("expected missing permission grant")
	}
	env.PermissionGrants = []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityEditorIntelligence), CapabilityID: cid("editor.secure"), Permission: perm}}
	env.ResourceBindings = []environment.ResourceBinding{rb(permission.KindSecret, "named-secret", "dev_secure.named_secret")}
	lock, _ = plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ = pruntime.BuildPlan(cat, env, lock)
	if _, err := (pruntime.Registry{Processes: &launcher{}, Resources: &resourceBroker{}}).Invoke(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.secure"), "x", "r", nil); err == nil {
		t.Fatal("expected required sandbox rejection")
	}
}

func TestValidateDeploymentInvocationAgainstLockedRuntimePlan(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint"}
	def := plugin.Definition{
		ID: cid("tei.deploy.secure"), Version: "1.0.0",
		Runtime:  plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "deploy-worker", Sandbox: plugin.SandboxRequired},
		Artifact: plugin.Artifact{Digest: "sha256:5af655bd19fffee7c3564b060f92056e1a6bd521c0b15bd1155dd244cd77de43", SignatureRef: "sigstore://deploy", Provenance: "source://deploy"},
		Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("deploy.secure"), Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease}, Permissions: []permission.Spec{perm}}},
	}
	env := environment.Environment{ID: cid("prod_secure"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("deploy.secure"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{rb(permission.KindNetwork, "database-endpoint", "prod_secure.database_endpoint")}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, err := plugin.BuildLock(cat, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	pi := &executor.PluginInvocation{
		PluginID: def.ID, PluginVersion: def.Version,
		DescriptorFingerprint: plugin.DefinitionFingerprint(def),
		PluginLockFingerprint: plan.PluginLockFingerprint, RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint,
		ArtifactDigest: def.Artifact.Digest, ArtifactSignatureRef: def.Artifact.SignatureRef, ArtifactProvenance: def.Artifact.Provenance,
		Boundary: string(pruntime.BoundaryProcess), Sandboxed: true,
		PermissionLease: []permission.Spec{perm},
		ResourceLeaseID: "lease-deploy", ResourceLeaseReleased: true,
		Resources:                []resource.Record{{Kind: perm.Kind, Access: perm.Access, Scope: perm.Scope, ResourceID: cid("prod_secure.database_endpoint")}},
		ResourceLeaseSupervision: []resource.SupervisionRecord{{State: resource.SupervisionActive}, {State: resource.SupervisionRevokePending}, {State: resource.SupervisionRevoked}},
		VerifierID:               "test-verifier", DigestVerified: true, SignatureVerified: true, ProvenanceChecked: true,
	}
	if err := pruntime.ValidateDeploymentInvocation(plan, cid("deploy.secure"), pi); err != nil {
		t.Fatalf("valid invocation rejected: %v", err)
	}
	bad := *pi
	bad.Sandboxed = false
	if err := pruntime.ValidateDeploymentInvocation(plan, cid("deploy.secure"), &bad); err == nil {
		t.Fatal("expected sandbox provenance mismatch")
	}
	bad = *pi
	bad.PermissionLease = nil
	if err := pruntime.ValidateDeploymentInvocation(plan, cid("deploy.secure"), &bad); err == nil {
		t.Fatal("expected permission lease mismatch")
	}
	bad = *pi
	bad.SignatureVerified = false
	if err := pruntime.ValidateDeploymentInvocation(plan, cid("deploy.secure"), &bad); err == nil {
		t.Fatal("expected artifact signature verification mismatch")
	}
	bad = *pi
	bad.ResourceLeaseRevision = 2
	bad.ResourceLeaseRenewals = nil
	if err := pruntime.ValidateDeploymentInvocation(plan, cid("deploy.secure"), &bad); err == nil {
		t.Fatal("expected forged resource lease renewal history mismatch")
	}
	bad = *pi
	bad.ResourceLeaseSupervision = []resource.SupervisionRecord{{State: resource.SupervisionActive}, {State: resource.SupervisionRevokePending}}
	if err := pruntime.ValidateDeploymentInvocation(plan, cid("deploy.secure"), &bad); err == nil {
		t.Fatal("expected missing final revoked supervision state")
	}
}

func TestTrustPreflightHappensBeforeResourceLeaseAcquisition(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "named-secret"}
	def := plugin.Definition{
		ID: cid("tei.secure.preflight"), Version: "1.0.0",
		Runtime:  plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired},
		Artifact: plugin.Artifact{Digest: "sha256:5af655bd19fffee7c3564b060f92056e1a6bd521c0b15bd1155dd244cd77de43", SignatureRef: "sigstore://preflight"},
		Provides: []plugin.Capability{{Kind: plugin.CapabilityEditorIntelligence, ID: cid("editor.secure.preflight"), Features: []environment.Feature{environment.FeatureSQLAnalysis}, Permissions: []permission.Spec{perm}}},
	}
	env := environment.Environment{ID: cid("preflight_dev"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityEditorIntelligence), CapabilityID: cid("editor.secure.preflight"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{rb(permission.KindSecret, "named-secret", "preflight_dev.named_secret")}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)

	broker := &resourceBroker{}
	// Missing verifier fails before the required sandbox/resource path is touched.
	if _, err := (pruntime.Registry{Sandboxes: &sandboxLauncher{}, Resources: broker}).Invoke(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.secure.preflight"), "x", "r", nil); err == nil {
		t.Fatal("expected missing artifact verifier")
	}
	if broker.acquired != 0 {
		t.Fatalf("resource lease acquired before artifact verification: %+v", broker)
	}

	broker = &resourceBroker{}
	// With verification available but the required sandbox unavailable, resource
	// acquisition must still not happen.
	if _, err := (pruntime.Registry{Verifier: &verifier{}, Processes: &launcher{}, Resources: broker}).Invoke(context.Background(), plan, plugin.CapabilityEditorIntelligence, cid("editor.secure.preflight"), "x", "r", nil); err == nil {
		t.Fatal("expected missing required sandbox")
	}
	if broker.acquired != 0 {
		t.Fatalf("resource lease acquired before sandbox preflight: %+v", broker)
	}
}

type failingSandbox struct{}

func (failingSandbox) Invoke(context.Context, pruntime.SandboxSpec, pruntime.Request) (pruntime.Response, error) {
	return pruntime.Response{}, context.Canceled
}

func TestResourceLeaseIsReleasedWhenPluginInvocationFails(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "database-endpoint"}
	def := plugin.Definition{ID: cid("tei.fail.proc"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "worker", Sandbox: plugin.SandboxRequired}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("deploy.fail"), Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease}, Permissions: []permission.Spec{perm}}}}
	env := environment.Environment{ID: cid("fail_prod"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("deploy.fail"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{rb(permission.KindNetwork, "database-endpoint", "fail_prod.database_endpoint")}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	broker := &resourceBroker{}
	_, err := (pruntime.Registry{Sandboxes: failingSandbox{}, Resources: broker}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, cid("deploy.fail"), "execute", "r-fail", map[string]string{"x": "y"})
	if err == nil {
		t.Fatal("expected invocation failure")
	}
	if broker.acquired != 1 || broker.released != 1 {
		t.Fatalf("resource lease not cleaned up on failure: %+v", broker)
	}
}

type blockingReleaseBroker struct{ resourceBroker }

func (b *blockingReleaseBroker) Release(ctx context.Context, _ resource.Lease) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestRevokeTimeoutTurnsCleanupFailureIntoInvocationError(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "credential"}
	def := plugin.Definition{ID: cid("tei.revoke.timeout"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("revoke.timeout"), Permissions: []permission.Spec{perm}}}}
	env := environment.Environment{ID: cid("revoke_env"), Plugins: []environment.ExtensionRequirement{{ID: def.ID, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("revoke.timeout"), Permission: perm}}, ResourceBindings: []environment.ResourceBinding{{Kind: permission.KindSecret, Scope: perm.Scope, ResourceID: cid("revoke_env.secret")}}, LeasePolicies: []environment.CapabilityLeasePolicy{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: cid("revoke.timeout"), Policy: resource.LeasePolicy{MaxLifetime: "1m", RenewBefore: "10s", RevokeTimeout: "5ms", RenewalFailure: resource.RenewalFailClosed}}}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, _ := plugin.BuildLock(cat, []environment.Environment{env})
	plan, _ := pruntime.BuildPlan(cat, env, lock)
	id := plugin.Identity{ID: def.ID, Version: def.Version}
	reg := pruntime.Registry{Builtins: map[string]pruntime.Builtin{"tei.revoke.timeout@1.0.0": builtin{id: id, fp: plugin.DefinitionFingerprint(def)}}, Resources: &blockingReleaseBroker{}}
	_, err := reg.InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, cid("revoke.timeout"), "execute", "revoke-timeout", nil)
	if err == nil {
		t.Fatal("expected revoke timeout to fail invocation")
	}
}
