package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// TestProcessHostTrustRequirementsFailClosed verifies the real process boundary:
// matching file bytes alone never satisfy an explicitly required signature,
// provenance check or sandbox. The reference verifier only checks a SHA-256
// digest; no production trust anchor or OS sandbox is represented here.
func TestProcessHostTrustRequirementsFailClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "reference-process")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../plugins/reference-process-executor")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real process plugin: %v\n%s", err, out)
	}
	b, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	capabilityID := core.MustCanonicalID("process.reference")
	identity := plugin.Identity{
		ID: core.MustCanonicalID("tei.process.reference"),
		Version: "1.0.0",
	}
	capability := plugin.Capability{
		Kind:    plugin.CapabilityDeploymentExecutor,
		ID:      capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "reference-host-trust-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                identity,
			DescriptorFingerprint: "reference-process-descriptor",
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Command:               binary,
			Args:                  []string{referenceArgvMarker},
			SandboxPolicy:         plugin.SandboxNone,
			Artifact:              plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity, Capability: capability, Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	launcher := &countingLauncher{}
	adapter := pruntime.DeploymentExecutor{
		Plan: plan,
		Registry: pruntime.Registry{
			Processes: launcher,
			Verifier:  fileVerifier{path: binary},
		},
		CapabilityID: capabilityID,
	}
	command := executor.Command{
		ID:             "host-trust-reference",
		ActionID:       "sales_expand/apply",
		IdempotencyKey: "host-trust-once",
		Kind:           string(deployment.ActionDatabaseRelease),
		Database:       core.MustCanonicalID("sales_core"),
		Version:        "1.0.0",
	}
	first, err := adapter.Execute(ctx, command)
	if err != nil {
		t.Fatalf("verified unsigned reference executable rejected: %v", err)
	}
	if first.Outcome != executor.OutcomeAccepted ||
		first.PluginInvocation == nil || !first.PluginInvocation.DigestVerified ||
		first.PluginInvocation.SignatureVerified || first.PluginInvocation.Sandboxed {
		t.Fatalf("incorrect reference-only trust evidence: %+v", first.PluginInvocation)
	}
	if err := pruntime.ValidateDeploymentInvocation(plan, capabilityID, first.PluginInvocation); err != nil {
		t.Fatal(err)
	}
	if launcher.calls != 1 {
		t.Fatalf("expected one real process dispatch, got %d", launcher.calls)
	}

	cases := []struct {
		name         string
		change       func(*pruntime.DeploymentExecutor)
		wantError    string
	}{
		{
			name: "explicit-signature-required",
			change: func(a *pruntime.DeploymentExecutor) {
				a.Plan.Plugins[0].Artifact.SignatureRef = "signature-required-but-unverified"
			},
			wantError: "signature was not verified",
		},
		{
			name: "explicit-provenance-required",
			change: func(a *pruntime.DeploymentExecutor) {
				a.Plan.Plugins[0].Artifact.Provenance = "provenance-required-but-unchecked"
			},
			wantError: "provenance was not checked",
		},
		{
			name: "sandbox-required",
			change: func(a *pruntime.DeploymentExecutor) {
				a.Plan.Plugins[0].SandboxPolicy = plugin.SandboxRequired
			},
			wantError: "requires sandbox launcher",
		},
		{
			name: "verifier-missing",
			change: func(a *pruntime.DeploymentExecutor) {
				a.Registry.Verifier = nil
			},
			wantError: "no artifact verifier",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := adapter
			next.Plan.Plugins = append([]pruntime.PluginRuntime(nil), adapter.Plan.Plugins...)
			tc.change(&next)
			before := launcher.calls
			if _, err := next.Execute(ctx, command); err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected host refusal containing %q; got %v", tc.wantError, err)
			}
			if launcher.calls != before {
				t.Fatalf("untrusted process was started: before=%d after=%d", before, launcher.calls)
			}
		})
	}
}
