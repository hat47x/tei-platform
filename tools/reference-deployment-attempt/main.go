package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"tei/internal/core"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
	"tei/internal/workspace"
)

type verifier struct{}

func (verifier) Verify(_ context.Context, _ plugin.Identity, _ plugin.Artifact) (pruntime.ArtifactVerification, error) {
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
func (secrets) RefreshSecrets(context.Context, []pruntime.SecretInjection, []pruntime.SecretInjection) error {
	return nil
}

type runner struct{}

func (runner) InvokeSandboxed(ctx context.Context, _ pruntime.ProcessCommand, _ pruntime.Request) (pruntime.Response, error) {
	// M0.21 deliberately does not call Renew or Refresh from plugin code.
	// The short delay gives the host refresh trigger a chance to fire; lease
	// lifecycle management remains entirely host-supervised.
	select {
	case <-ctx.Done():
		return pruntime.Response{}, ctx.Err()
	case <-time.After(20 * time.Millisecond):
	}
	out := executor.Result{AttemptID: "attempt-m0.21-atlas-accepted", Outcome: executor.OutcomeAccepted, ExternalRef: "reference://atlas/apply/sales-expand"}
	raw, _ := json.Marshal(out)
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: raw}, nil
}

type onceRefreshTrigger struct{ fired bool }

func (t *onceRefreshTrigger) WaitForRefresh(ctx context.Context, _ resource.Lease) error {
	if !t.fired {
		t.fired = true
		return nil
	}
	<-ctx.Done()
	return ctx.Err()
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
	workspacePath := flag.String("workspace", "reference/workspace", "workspace path")
	lockPath := flag.String("plugin-lock", "reference/workspace/.tei/plugin-lock.json", "plugin lock")
	runPath := flag.String("run", "reference/order_rollout.durable-run.m0.21.start.json", "durable run")
	flag.Parse()

	loaded, err := workspace.Load(*workspacePath)
	if err != nil {
		panic(err)
	}
	var lock plugin.Lock
	if raw, err := os.ReadFile(*lockPath); err != nil {
		panic(err)
	} else if err := json.Unmarshal(raw, &lock); err != nil {
		panic(err)
	}
	var run durable.Run
	if raw, err := os.ReadFile(*runPath); err != nil {
		panic(err)
	} else if err := json.Unmarshal(raw, &run); err != nil {
		panic(err)
	}
	env, ok := findEnv(loaded.World.Environments, run.Environment)
	if !ok {
		panic(fmt.Errorf("environment %s not found", run.Environment))
	}
	plan, err := pruntime.BuildPlan(plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}, env, lock)
	if err != nil {
		panic(err)
	}
	commands := durable.ReadyCommands(run)
	if len(commands) == 0 {
		panic("run has no ready command")
	}
	cmd := commands[0]
	if cmd.Kind != "database-release" {
		panic(fmt.Sprintf("expected first command database-release, got %s", cmd.Kind))
	}

	broker := &resource.AdapterBroker{TTL: 10 * time.Minute, Now: func() time.Time { return time.Now().UTC() }, Adapters: []resource.Adapter{
		&resource.ReferenceSecretManagerAdapter{Handles: map[core.CanonicalID]string{core.MustCanonicalID("production.sales_core.migration_credential"): "secret-manager://reference/sales-core"}},
		&resource.StaticAdapter{ResourceKind: permission.KindNetwork, Entries: []resource.StaticResource{{ResourceID: core.MustCanonicalID("production.sales_core.database_endpoint"), NetworkEndpoint: "sales-db.production.internal:5432"}}},
		&resource.StaticAdapter{ResourceKind: permission.KindFilesystem, Entries: []resource.StaticResource{{ResourceID: core.MustCanonicalID("production.sales_core.migration_workdir"), MountSourceHandle: "mount-source://reference/migration-workdir", MountPath: "/tei/migration"}}},
	}}
	sandbox := pruntime.HostSandboxLauncher{Network: network{}, Filesystem: filesystem{}, Secrets: secrets{}, Runner: runner{}, RefreshTrigger: &onceRefreshTrigger{}}
	ownerships := &resource.MemoryOwnershipStore{}
	adapter := pruntime.DeploymentExecutor{Plan: plan, Registry: pruntime.Registry{Sandboxes: sandbox, Verifier: verifier{}, Resources: broker, SupervisorOwnerships: ownerships, SupervisorOwnerID: "reference-host-a", SupervisorOwnershipTTL: 30 * time.Millisecond}, CapabilityID: core.MustCanonicalID("migration.atlas")}
	result, err := adapter.Execute(context.Background(), cmd)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		panic(err)
	}
}
