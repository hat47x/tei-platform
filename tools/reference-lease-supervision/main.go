package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
	"tei/internal/workspace"
)

type verifier struct{}

func (verifier) Verify(context.Context, plugin.Identity, plugin.Artifact) (pruntime.ArtifactVerification, error) {
	return pruntime.ArtifactVerification{VerifierID: "reference-artifact-verifier", DigestVerified: true, SignatureVerified: true, ProvenanceChecked: true}, nil
}

type network struct{}

func (network) ApplyNetworkPolicy(context.Context, pruntime.NetworkPolicy) (pruntime.Cleanup, error) {
	return func(context.Context) error { return nil }, nil
}

type filesystem struct{}

func (filesystem) ApplyFilesystemPolicy(context.Context, pruntime.FilesystemPolicy) (pruntime.Cleanup, error) {
	return func(context.Context) error { return nil }, nil
}

type secrets struct{}

func (secrets) InjectSecrets(context.Context, []pruntime.SecretInjection) (pruntime.Cleanup, error) {
	return func(context.Context) error { return nil }, nil
}

type slowRunner struct{}

func (slowRunner) InvokeSandboxed(ctx context.Context, _ pruntime.ProcessCommand, _ pruntime.Request) (pruntime.Response, error) {
	select {
	case <-ctx.Done():
		return pruntime.Response{}, ctx.Err()
	case <-time.After(260 * time.Millisecond):
		return pruntime.Response{Status: pruntime.ResponseOK}, nil
	}
}

func findEnv(xs []environment.Environment, id core.CanonicalID) (environment.Environment, bool) {
	for _, e := range xs {
		if e.ID == id {
			return e, true
		}
	}
	return environment.Environment{}, false
}

func main() {
	loaded, err := workspace.Load("reference/workspace")
	if err != nil {
		panic(err)
	}
	var lock plugin.Lock
	raw, err := os.ReadFile("reference/workspace/.tei/plugin-lock.json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		panic(err)
	}
	env, ok := findEnv(loaded.World.Environments, core.MustCanonicalID("production"))
	if !ok {
		panic("production environment not found")
	}
	for i := range env.LeasePolicies {
		if env.LeasePolicies[i].CapabilityKind == string(plugin.CapabilityDeploymentExecutor) && env.LeasePolicies[i].CapabilityID == core.MustCanonicalID("migration.atlas") {
			env.LeasePolicies[i].Policy = resource.LeasePolicy{MaxLifetime: "900ms", RenewBefore: "200ms", RevokeTimeout: "100ms", RenewalFailure: resource.RenewalFailClosed}
		}
	}
	plan, err := pruntime.BuildPlan(plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}, env, lock)
	if err != nil {
		panic(err)
	}
	secret := core.MustCanonicalID("production.sales_core.migration_credential")
	networkID := core.MustCanonicalID("production.sales_core.database_endpoint")
	workdir := core.MustCanonicalID("production.sales_core.migration_workdir")
	broker := &resource.AdapterBroker{TTL: 350 * time.Millisecond, Adapters: []resource.Adapter{
		&resource.ReferenceSecretManagerAdapter{Handles: map[core.CanonicalID]string{secret: "secret-manager://reference/sales-core"}},
		&resource.StaticAdapter{ResourceKind: permission.KindNetwork, Entries: []resource.StaticResource{{ResourceID: networkID, NetworkEndpoint: "sales-db.production.internal:5432"}}},
		&resource.StaticAdapter{ResourceKind: permission.KindFilesystem, Entries: []resource.StaticResource{{ResourceID: workdir, MountSourceHandle: "mount-source://reference/migration-workdir", MountPath: "/tei/migration"}}},
	}}
	host := pruntime.HostSandboxLauncher{Network: network{}, Filesystem: filesystem{}, Secrets: secrets{}, Runner: slowRunner{}}
	inv, err := (pruntime.Registry{Sandboxes: host, Verifier: verifier{}, Resources: broker, SupervisorOwnerships: &resource.MemoryOwnershipStore{}, SupervisorOwnerID: "reference-supervisor-host", SupervisorOwnershipTTL: 90 * time.Millisecond}).InvokeWithProvenance(context.Background(), plan, plugin.CapabilityDeploymentExecutor, core.MustCanonicalID("migration.atlas"), "supervision-reference", "m0.21-supervision", map[string]string{"purpose": "host-supervised-renewal"})
	if err != nil {
		panic(err)
	}
	if len(inv.Provenance.ResourceLeaseRenewals) == 0 {
		panic("host lease supervisor did not renew")
	}
	out := struct {
		RuntimePolicyFingerprint string                        `json:"runtimePolicyFingerprint"`
		Provenance               pruntime.InvocationProvenance `json:"provenance"`
		BoundaryNotes            []string                      `json:"boundaryNotes"`
	}{
		RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint,
		Provenance:               inv.Provenance,
		BoundaryNotes: []string{
			"The plugin runner never calls Renew or Refresh; the HostSandboxLauncher lease supervisor renews automatically.",
			"Supervisor ownership is heartbeated independently and fenced with a monotonic token; the final ownership record is released.",
			"The reference uses millisecond lease policy only to make the supervision transition executable quickly; production policy remains unchanged.",
			"Concrete secret handles, network endpoints, and mount handles are absent from durable provenance.",
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
