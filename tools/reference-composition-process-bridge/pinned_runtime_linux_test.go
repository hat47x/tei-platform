//go:build linux

package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
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

// この試験は製品のRuntimeが提供する選択式起動器を使用する。
// 先行する参照用pinnedReferenceHostの別実装に成功判定を依存させない。
func invokeProductPinned(t *testing.T, ctx context.Context, host *pruntime.PinnedLinuxProcess, path, digest string, signature, provenance string, sandbox plugin.SandboxPolicy) (executor.Result, error) {
	return invokePinnedWithRegistry(t, ctx, pruntime.Registry{Verifier: host, Processes: host}, path, digest, signature, provenance, sandbox)
}

func invokePinnedWithRegistry(t *testing.T, ctx context.Context, registry pruntime.Registry, path, digest string, signature, provenance string, sandbox plugin.SandboxPolicy) (executor.Result, error) {
	t.Helper()
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "pinned-product-reference-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "pinned-product-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: path, Args: []string{referenceArgvMarker},
			SandboxPolicy: sandbox,
			Artifact:      plugin.Artifact{Digest: digest, SignatureRef: signature, Provenance: provenance},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
				Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
			},
			Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	adapter := pruntime.DeploymentExecutor{
		Plan:         plan,
		Registry:     registry,
		CapabilityID: capabilityID,
	}
	return adapter.Execute(ctx, executor.Command{
		ID: "pinned-production-reference", ActionID: "sales_expand/apply",
		IdempotencyKey: "pinned-production-once",
		Kind:           string(deployment.ActionDatabaseRelease),
		Database:       core.MustCanonicalID("sales_core"), Version: "1.0.0",
	})
}

func TestProductPinnedLinuxProcessKeepsVerifiedImageOnReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dir := t.TempDir()
	path, digest := buildPinnedFixture(t, ctx, dir, "approved")
	replacement, replacementDigest := buildPinnedFixture(t, ctx, dir, "replaced")
	if digest == replacementDigest {
		t.Fatal("fixtures unexpectedly contain identical bytes")
	}
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	host, err := pruntime.PinLinuxProcess(ctx, identity, path, digest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatalf("pin product runtime executable: %v", err)
	}
	defer host.Close()

	first, err := invokeProductPinned(t, ctx, host, path, digest, "", "", plugin.SandboxNone)
	if err != nil || !strings.Contains(first.Message, "revision=approved") {
		t.Fatalf("did not launch approved image: result=%+v error=%v", first, err)
	}
	if first.PluginInvocation == nil || !first.PluginInvocation.DigestVerified ||
		first.PluginInvocation.SignatureVerified || first.PluginInvocation.Sandboxed {
		t.Fatalf("invalid pinned invocation evidence: %+v", first.PluginInvocation)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if "sha256:"+hex.EncodeToString(sum[:]) != replacementDigest {
		t.Fatal("source path did not point to replacement bytes")
	}
	second, err := invokeProductPinned(t, ctx, host, path, digest, "", "", plugin.SandboxNone)
	if err != nil || !strings.Contains(second.Message, "revision=approved") ||
		strings.Contains(second.Message, "revision=replaced") {
		t.Fatalf("different executable launched after path replacement: result=%+v error=%v", second, err)
	}

	// 同一インスタンスが検証者と起動器を兼ねる場合も、
	// 別の指紋や未検証の署名・来歴を肯定しない。
	for _, rejected := range []struct {
		digest, signature, provenance string
	}{
		{digest: replacementDigest},
		{digest: digest, signature: "unverified-signature"},
		{digest: digest, provenance: "unverified-provenance"},
	} {
		if _, err := invokeProductPinned(t, ctx, host, path, rejected.digest, rejected.signature, rejected.provenance, plugin.SandboxNone); err == nil {
			t.Fatalf("unverified artifact was accepted: %+v", rejected)
		}
	}
	if _, err := invokeProductPinned(t, ctx, host, path, digest, "", "", plugin.SandboxRequired); err == nil {
		t.Fatal("sandbox-required executable was launched without a sandbox")
	}
	otherIdentity := plugin.Identity{ID: core.MustCanonicalID("tei.process.other"), Version: "1.0.0"}
	if _, err := host.Verify(ctx, otherIdentity, plugin.Artifact{Digest: digest}); err == nil {
		t.Fatal("artifact verifier approved another plugin identity")
	}
	if err := host.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeProductPinned(t, ctx, host, path, digest, "", "", plugin.SandboxNone); err == nil {
		t.Fatal("closed executable was launched")
	}
	if err := host.Close(); err != nil {
		t.Fatalf("second close should be harmless: %v", err)
	}
}

func TestProductPinnedLinuxProcessSymlinkAndPreparationRejection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	original, digest := buildPinnedFixture(t, ctx, dir, "approved")
	replacement, _ := buildPinnedFixture(t, ctx, dir, "replaced")
	symlink := filepath.Join(dir, "selected")
	if err := os.Symlink(original, symlink); err != nil {
		t.Fatal(err)
	}
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	host, err := pruntime.PinLinuxProcess(ctx, identity, symlink, digest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if err := os.Remove(symlink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, symlink); err != nil {
		t.Fatal(err)
	}
	out, err := invokeProductPinned(t, ctx, host, symlink, digest, "", "", plugin.SandboxNone)
	if err != nil || !strings.Contains(out.Message, "revision=approved") {
		t.Fatalf("symlink retarget reached other bytes: result=%+v error=%v", out, err)
	}
	if _, err := host.Invoke(ctx, pruntime.ProcessCommand{Executable: replacement}, pruntime.Request{}); err == nil {
		t.Fatal("different command path bypassed pinned executable")
	}
	for _, bad := range []string{"", "sha256:invalid", "sha256:" + strings.Repeat("0", 64)} {
		p, err := pruntime.PinLinuxProcess(ctx, identity, replacement, bad, pruntime.StdioProcessLauncher{})
		if err == nil {
			p.Close()
			t.Fatalf("accepted an invalid executable fingerprint: %q", bad)
		}
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	_, err = pruntime.PinLinuxProcess(stopped, identity, original, digest, pruntime.StdioProcessLauncher{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled preparation was not rejected: %v", err)
	}
}

func TestProductPinnedLinuxProcessRejectsMismatchedRegistryWiring(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	path, originalDigest := buildPinnedFixture(t, ctx, dir, "approved")
	replacement, replacementDigest := buildPinnedFixture(t, ctx, dir, "replaced")
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	pinned, err := pruntime.PinLinuxProcess(ctx, identity, path, originalDigest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()
	otherPinned, err := pruntime.PinLinuxProcess(ctx, identity, replacement, replacementDigest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer otherPinned.Close()

	// Make path execution observably dangerous: a mismatched verifier would
	// attest to "approved" while an ordinary launcher starts "replaced".
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		verifier pruntime.ArtifactVerifier
		launcher pruntime.ProcessLauncher
		digest   string
	}{
		{name: "pinned-verifier-with-ordinary-path-launcher", verifier: pinned, launcher: pruntime.StdioProcessLauncher{}, digest: originalDigest},
		{name: "ordinary-verifier-with-pinned-launcher", verifier: fileVerifier{path: path}, launcher: pinned, digest: replacementDigest},
		{name: "two-distinct-pinned-images", verifier: pinned, launcher: otherPinned, digest: originalDigest},
		{name: "missing-launcher", verifier: pinned, digest: originalDigest},
		{name: "missing-verifier", launcher: pinned, digest: originalDigest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := invokePinnedWithRegistry(t, ctx, pruntime.Registry{
				Verifier: tc.verifier, Processes: tc.launcher,
			}, path, tc.digest, "", "", plugin.SandboxNone)
			if err == nil {
				t.Fatal("invalid Host wiring unexpectedly dispatched")
			}
			// Missing verifier/launcher is also rejected by the existing
			// required-component guard, before the image-binding guard.
			if tc.verifier != nil && tc.launcher != nil && !strings.Contains(err.Error(), "pinned process") {
				t.Fatalf("mixed Host implementations bypassed binding check: %v", err)
			}
		})
	}
	// A correctly bound Host continues to execute the originally accepted
	// bytes even after its source pathname now references different bytes.
	ok, err := invokeProductPinned(t, ctx, pinned, path, originalDigest, "", "", plugin.SandboxNone)
	if err != nil || !strings.Contains(ok.Message, "revision=approved") {
		t.Fatalf("valid Host binding stopped executing the held image: %+v %v", ok, err)
	}
}

// pinnedSandboxProbe detects any sandbox dispatch that would lose the pinned FD.
// It does not provide an OS sandbox and should never be invoked in this test.
type pinnedSandboxProbe struct{ calls int }

func (s *pinnedSandboxProbe) Invoke(_ context.Context, _ pruntime.SandboxSpec, _ pruntime.Request) (pruntime.Response, error) {
	s.calls++
	return pruntime.Response{}, errors.New("unexpected sandbox dispatch")
}

func TestProductPinnedLinuxProcessRejectsUnboundSandboxRoute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	path, digest := buildPinnedFixture(t, ctx, dir, "approved")
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	pinned, err := pruntime.PinLinuxProcess(ctx, identity, path, digest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()
	sandbox := &pinnedSandboxProbe{}
	registry := pruntime.Registry{Verifier: pinned, Processes: pinned, Sandboxes: sandbox}
	// A preferred sandbox would dispatch the unbound command path, ignoring
	// the image held by the verification/launch pair; reject before dispatch.
	if _, err := invokePinnedWithRegistry(t, ctx, registry, path, digest, "", "", plugin.SandboxPreferred); err == nil ||
		!strings.Contains(err.Error(), "pinned process image") {
		t.Fatalf("unbound sandbox route was not denied: %v", err)
	}
	if sandbox.calls != 0 {
		t.Fatal("separate sandbox launcher received the process")
	}
	// A configured sandbox must not be used accidentally for sandbox=none:
	// the correctly bound image still reaches the validated process launcher.
	out, err := invokePinnedWithRegistry(t, ctx, registry, path, digest, "", "", plugin.SandboxNone)
	if err != nil || !strings.Contains(out.Message, "revision=approved") || sandbox.calls != 0 {
		t.Fatalf("valid pinned route did not remain available: %+v err=%v sandboxCalls=%d", out, err, sandbox.calls)
	}
}
