package runtime_test

import (
	"context"
	"testing"

	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

type permissionMutatingBuiltin struct {
	builtin
	mutate bool
}

func (b permissionMutatingBuiltin) Invoke(_ context.Context, request pruntime.Request) (pruntime.Response, error) {
	if b.mutate && len(request.Permissions) > 0 {
		request.Permissions[0].Scope = "forged-scope"
	}
	return pruntime.Response{Status: pruntime.ResponseOK}, nil
}

func TestBuiltinRequestCannotRewriteHostPermissionProvenance(t *testing.T) {
	perm := permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "workspace-source"}
	id, capID := cid("tei.mutation.test"), cid("mutation.test")
	def := plugin.Definition{ID: id, Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: capID, Actions: []deployment.ActionKind{deployment.ActionApplicationStart}, Permissions: []permission.Spec{perm}}}}
	env := environment.Environment{ID: cid("dev"), Plugins: []environment.ExtensionRequirement{{ID: id, CompatibleMajor: 1}}, PermissionGrants: []environment.PermissionGrant{{CapabilityKind: string(plugin.CapabilityDeploymentExecutor), CapabilityID: capID, Permission: perm}}, ResourceBindings: []environment.ResourceBinding{rb(permission.KindFilesystem, perm.Scope, "dev.source")}}
	cat := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, err := plugin.BuildLock(cat, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(cat, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []bool{false, true} {
		broker := &resourceBroker{}
		impl := permissionMutatingBuiltin{builtin: builtin{id: plugin.Identity{ID: id, Version: def.Version}, fp: plugin.DefinitionFingerprint(def)}, mutate: mutate}
		registry := pruntime.Registry{Builtins: map[string]pruntime.Builtin{"tei.mutation.test@1.0.0": impl}, Resources: broker}
		inv, err := registry.InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, capID, "execute", "permission-test", map[string]string{"value": "test"})
		if err != nil {
			t.Fatalf("mutate=%t: %v", mutate, err)
		}
		if len(inv.Provenance.Permissions) != 1 || inv.Provenance.Permissions[0] != perm {
			t.Fatalf("mutate=%t corrupted Host provenance: %+v", mutate, inv.Provenance.Permissions)
		}
		if broker.acquired != 1 || broker.released != 1 || !inv.Provenance.ResourceLeaseReleased {
			t.Fatalf("mutate=%t changed resource lifecycle: %+v %+v", mutate, broker, inv.Provenance)
		}
	}
	t.Log("normal and mutating builtin invocations preserve Host permission provenance and release the resource lease")
}
