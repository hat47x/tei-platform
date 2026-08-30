package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const referenceArgvMarker = "--tei-reference-marker=argv value preserved"

func main() {
	pluginPath := flag.String("plugin", "", "path to the reference process plugin executable")
	flag.Parse()
	if *pluginPath == "" {
		fmt.Fprintln(os.Stderr, "-plugin is required")
		os.Exit(2)
	}

	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind:    plugin.CapabilityDeploymentExecutor,
		ID:      capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:              core.MustCanonicalID("reference"),
		PluginLockFingerprint:    "reference-process-lock",
		RuntimePolicyFingerprint: "reference-process-policy",
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                identity,
			DescriptorFingerprint: "reference-process-descriptor",
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Command:               *pluginPath,
			Args:                  []string{referenceArgvMarker},
			SandboxPolicy:         plugin.SandboxNone,
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin:     identity,
			Capability: capability,
			Boundary:   pruntime.BoundaryProcess,
		}},
	}
	adapter := pruntime.DeploymentExecutor{
		Plan:         plan,
		Registry:     pruntime.Registry{Processes: pruntime.StdioProcessLauncher{}},
		CapabilityID: capabilityID,
	}
	command := executor.Command{
		ID:             "reference-process-command-1",
		ActionID:       "sales_expand/apply",
		IdempotencyKey: "reference-process-idempotency-1",
		Kind:           string(deployment.ActionDatabaseRelease),
		Database:       core.MustCanonicalID("sales_core"),
		Version:        "1.0.0",
	}
	result, err := adapter.Execute(context.Background(), command)
	if err != nil {
		panic(err)
	}
	if result.Outcome != executor.OutcomeAccepted {
		panic(fmt.Errorf("unexpected reference process outcome %s", result.Outcome))
	}
	if err := pruntime.ValidateDeploymentInvocation(plan, capabilityID, result.PluginInvocation); err != nil {
		panic(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		panic(err)
	}
}
