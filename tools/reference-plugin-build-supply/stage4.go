package pluginbuildsupply

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/workspace"
)

const (
	selfBuildWorkspace  = "reference/plugin-self-build/workspace"
	processCapabilityID = "process.reference"
	processExecutable   = "tei-process-reference"
	processArgvMarker   = "--tei-reference-marker=argv value preserved"
)

// Stage4 is the call of the built process plugin from the bytes the store
// returned, under the host registration stage 3 produced.
type Stage4 struct {
	RetrievedDigest                 string `json:"retrievedDigest"`
	StoreLocator                    string `json:"storeLocator"`
	InvokedArtifactDigest           string `json:"invokedArtifactDigest"`
	DigestVerified                  bool   `json:"digestVerified"`
	BuildPlanFingerprint            string `json:"buildPlanFingerprint"`
	PluginLockFingerprint           string `json:"pluginLockFingerprint"`
	StaticResponseEngineFingerprint string `json:"staticResponseEngineFingerprint"`
	ProtocolVersion                 int    `json:"protocolVersion"`
	Outcome                         string `json:"outcome"`
	AttemptID                       string `json:"attemptId"`
	ArgvPreserved                   bool   `json:"argvPreserved"`
}

// fileVerifier hashes the exact file the launcher will execute.
type fileVerifier struct{ path string }

func (v fileVerifier) Verify(_ context.Context, _ plugin.Identity, artifact plugin.Artifact) (pruntime.ArtifactVerification, error) {
	digest, err := hashFile(v.path)
	if err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	if digest != artifact.Digest {
		return pruntime.ArtifactVerification{}, errors.New("plugin executable bytes do not match the locked digest")
	}
	return pruntime.ArtifactVerification{VerifierID: "plugin-build-supply.file-sha256", DigestVerified: true}, nil
}

// InvokeFromStore retrieves the artifact by digest, places it as the
// executable the descriptor names, and calls it through the product's
// DeploymentExecutor and StdioProcessLauncher (process protocol version 1).
// The environment plan is built from the self-build workspace and the
// runtime plugin lock stage 3 registered; nothing is rebuilt.
func InvokeFromStore(ctx context.Context, repo string, store Store, digest string, ev RunnerEvidence, workDir string) (Stage4, error) {
	if len(ev.NativeRuntimeEngineLock.Bindings) != 1 {
		return Stage4{}, fmt.Errorf("stage 3 engine lock must bind one plugin")
	}
	data, receipt, err := store.Get(digest)
	if err != nil {
		return Stage4{}, err
	}
	binDir := filepath.Join(workDir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return Stage4{}, err
	}
	executable := filepath.Join(binDir, processExecutable)
	if err := os.WriteFile(executable, data, 0o755); err != nil {
		return Stage4{}, err
	}
	loaded, err := workspace.Load(filepath.Join(repo, filepath.FromSlash(selfBuildWorkspace)))
	if err != nil {
		return Stage4{}, err
	}
	if len(loaded.World.Environments) != 1 {
		return Stage4{}, fmt.Errorf("self-build workspace must have one environment")
	}
	env := loaded.World.Environments[0]
	catalog := plugin.Catalog{Plugins: append([]plugin.Definition(nil), loaded.World.Plugins...)}
	for i, def := range catalog.Plugins {
		var artifact string
		switch def.ID {
		case ev.BuildPlan.Plugin.ID:
			artifact = receipt.Digest
		case ev.NativeRuntimeEngineLock.Bindings[0].Plugin.ID:
			artifact = ev.NativeFirstArtifactDigest
		default:
			continue
		}
		if catalog.Plugins[i], err = plugin.RealizeBuiltDefinition(def, artifact); err != nil {
			return Stage4{}, err
		}
	}
	plan, err := pruntime.BuildPlan(catalog, env, ev.RuntimePluginLock)
	if err != nil {
		return Stage4{}, fmt.Errorf("host plan from the registered lock: %w", err)
	}
	if plan, err = pruntime.BindRuntimeEngineLock(plan, ev.NativeRuntimeEngineLock); err != nil {
		return Stage4{}, err
	}
	engineFP := ""
	for _, p := range plan.Plugins {
		if p.Plugin == ev.NativeRuntimeEngineLock.Bindings[0].Plugin && p.RuntimeEngine != nil {
			if engineFP, err = pruntime.RuntimeEngineIdentityFingerprint(*p.RuntimeEngine); err != nil {
				return Stage4{}, err
			}
		}
	}
	capability := core.MustCanonicalID(processCapabilityID)
	adapter := pruntime.DeploymentExecutor{
		Plan: plan,
		Registry: pruntime.Registry{
			Processes: pruntime.StdioProcessLauncher{Env: []string{"PATH=" + binDir}, MaxExecutionDuration: 2 * time.Minute},
			Verifier:  fileVerifier{path: executable},
		},
		CapabilityID: capability,
	}
	command := executor.Command{
		ID: "plugin-build-supply-1", ActionID: "plugin_build_supply/invoke",
		IdempotencyKey: "plugin-build-supply-" + strings.TrimPrefix(digest, "sha256:")[:16],
		Kind:           string(deployment.ActionDatabaseRelease),
		Database:       core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	result, err := adapter.Execute(ctx, command)
	if err != nil {
		return Stage4{}, fmt.Errorf("invoke built process plugin: %w", err)
	}
	pi := result.PluginInvocation
	if pi == nil {
		return Stage4{}, fmt.Errorf("invocation has no provenance")
	}
	if err := pruntime.ValidateDeploymentInvocation(plan, capability, pi); err != nil {
		return Stage4{}, err
	}
	return Stage4{
		RetrievedDigest: receipt.Digest, StoreLocator: receipt.Locator,
		InvokedArtifactDigest: pi.ArtifactDigest, DigestVerified: pi.DigestVerified,
		BuildPlanFingerprint: pi.BuildPlanFingerprint, PluginLockFingerprint: plan.PluginLockFingerprint,
		StaticResponseEngineFingerprint: engineFP,
		ProtocolVersion:                 pruntime.ProtocolVersion,
		Outcome:                         string(result.Outcome), AttemptID: result.AttemptID,
		ArgvPreserved: strings.Contains(result.Message, processArgvMarker) && result.ActionID == command.ActionID && result.IdempotencyKey == command.IdempotencyKey,
	}, nil
}
