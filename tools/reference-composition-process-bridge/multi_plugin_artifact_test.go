package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/plugins/reference-bundle-alpha"
	"tei/plugins/reference-bundle-beta"
)

// An executable may contain more than one logical plugin. Exact artifact
// identity is necessary, but it is not a substitute for selected plugin or
// contract identity. This is a test-only host adapter, not a new resolver.
func TestTwoLogicalPluginsShareOnePhysicalExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference stdio process launcher forbids spaces in executable paths on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	path := filepath.Join(t.TempDir(), "tei-bundled-plugins")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", path, "../reference-plugin-bundle")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build bundled executable: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	const marker = "--tei-reference-marker=argv value preserved"
	type target struct {
		name     string
		id       plugin.Identity
		capID    core.CanonicalID
		provider string
		message  string
	}
	targets := []target{
		{"alpha", plugin.Identity{ID: core.MustCanonicalID(alpha.PluginID), Version: alpha.Version},
			core.MustCanonicalID(alpha.CapabilityID), "provider-alpha", "alpha accepted intent"},
		{"beta", plugin.Identity{ID: core.MustCanonicalID(beta.PluginID), Version: beta.Version},
			core.MustCanonicalID(beta.CapabilityID), "provider-beta", "beta accepted intent"},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "reference-multi-plugin-lock",
	}
	contract := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0",
		Shape: "reference-execute-request-result-v1",
	}
	var declarations []composition.Provider
	for _, tg := range targets {
		plan.Plugins = append(plan.Plugins, pruntime.PluginRuntime{
			Plugin: tg.id, DescriptorFingerprint: "descriptor-" + tg.name,
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: path, Args: []string{marker}, SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: digest},
		})
		plan.Capabilities = append(plan.Capabilities, pruntime.CapabilityRuntime{
			Plugin: tg.id, Boundary: pruntime.BoundaryProcess,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityDeploymentExecutor, ID: tg.capID,
				Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
			},
		})
		declarations = append(declarations, composition.Provider{
			ID: tg.provider, Scope: "bundle-reference", Offers: contract,
		})
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	if plan.Plugins[0].Artifact.Digest != plan.Plugins[1].Artifact.Digest ||
		plan.Plugins[0].Plugin == plan.Plugins[1].Plugin {
		t.Fatal("expected different logical identities with exactly one artifact digest")
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, declarations, nil); err != nil {
		t.Fatal(err)
	}
	launcher := &countingLauncher{}
	allowed := true
	denied := errors.New("host permission withdrawn")
	command := executor.Command{
		ID: "bundle-command", ActionID: "sales_expand/apply",
		IdempotencyKey: "bundle-key", Kind: string(deployment.ActionDatabaseRelease),
		Database: core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	coordinators := make(map[string]*composition.Coordinator)
	adapters := make(map[string]pruntime.DeploymentExecutor)
	for _, tg := range targets {
		selection, err := registry.Select(tg.provider, "bundle-reference")
		if err != nil {
			t.Fatal(err)
		}
		binding := composition.Binding{
			ContractID: contract.ID, ContractVersion: contract.Version,
			ProviderID: tg.provider, ArtifactDigest: digest, Generation: 1,
		}
		guard := func(b composition.Binding) error {
			if !allowed || b != binding {
				return denied
			}
			return nil
		}
		coordinator, err := composition.NewSelectedCoordinator(registry, selection, binding, composition.NewScope(), guard)
		if err != nil {
			t.Fatal(err)
		}
		coordinators[tg.provider] = coordinator
		adapters[tg.provider] = pruntime.DeploymentExecutor{
			Plan: plan, Registry: pruntime.Registry{
				Processes: launcher, Verifier: fileVerifier{path: path},
			}, CapabilityID: tg.capID,
		}
	}
	invoke := func(provider, executorProvider string) (executor.Result, error) {
		lease, err := coordinators[provider].Acquire()
		if err != nil {
			return executor.Result{}, err
		}
		defer lease.Release()
		// The explicit wiring lives in the host reference test only.
		var wanted plugin.Identity
		for _, tg := range targets {
			if tg.provider == provider {
				wanted = tg.id
			}
		}
		for _, tg := range targets {
			if tg.provider == executorProvider {
				if tg.id != wanted || lease.Binding.ProviderID != provider ||
					lease.Binding.ArtifactDigest != digest ||
					lease.Binding.ContractID != contract.ID ||
					lease.Binding.ContractVersion != contract.Version {
					return executor.Result{}, errors.New("bundled plugin wiring does not match selected provider")
				}
				return adapters[executorProvider].Execute(ctx, command)
			}
		}
		return executor.Result{}, errors.New("unknown executor provider")
	}
	for _, tg := range targets {
		result, err := invoke(tg.provider, tg.provider)
		if err != nil {
			t.Fatalf("invoke %s: %v", tg.name, err)
		}
		if result.Outcome != executor.OutcomeAccepted ||
			result.ActionID != command.ActionID || result.IdempotencyKey != command.IdempotencyKey ||
			!strings.Contains(result.Message, tg.message) {
			t.Fatalf("incorrect %s result: %+v", tg.name, result)
		}
		pv := result.PluginInvocation
		if pv == nil || pv.PluginID != tg.id.ID || pv.PluginVersion != tg.id.Version ||
			pv.ArtifactDigest != digest || !pv.DigestVerified ||
			pv.SignatureVerified || pv.Sandboxed {
			t.Fatalf("incorrect %s provenance: %+v", tg.name, pv)
		}
		if err := pruntime.ValidateDeploymentInvocation(plan, tg.capID, pv); err != nil {
			t.Fatalf("unverified %s result: %v", tg.name, err)
		}
	}
	if launcher.calls != 2 {
		t.Fatalf("one artifact did not receive both plugin requests: %d", launcher.calls)
	}
	before := launcher.calls
	if _, err := invoke("provider-alpha", "provider-beta"); err == nil || launcher.calls != before {
		t.Fatal("bundle artifact digest incorrectly authorized cross-plugin routing")
	}
	allowed = false
	for _, tg := range targets {
		if _, err := invoke(tg.provider, tg.provider); !errors.Is(err, denied) {
			t.Fatalf("host revocation ignored for %s: %v", tg.name, err)
		}
	}
	if launcher.calls != before {
		t.Fatal("revoked bundle executed an external process")
	}
	// No claim of production host isolation, signed bundles, or durable effects.
}
