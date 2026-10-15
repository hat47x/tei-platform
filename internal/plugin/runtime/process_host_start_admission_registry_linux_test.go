//go:build linux

package runtime_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// E43 deliberately uses the existing production Registry invocation path.
// It tests a TEST-ONLY Host-owned admission callback. It does not establish
// that production Registries universally require the optional E41 adapter.
func TestRegistryHostStartAdmissionBindsExactVerifierAndRunner(t *testing.T) {
	t.Setenv("TEI_PROCESS_HOST_SECRET_CANARY", "host-secret-not-for-plugin")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	req := processRequest()
	command := pruntime.ProcessCommand{Executable: executable, Args: []string{"--tei-process-env-probe"}}
	pinned, err := pruntime.PinLinuxProcess(
		context.Background(), req.Plugin, executable, digest, pruntime.StdioProcessLauncher{},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.Close()

	var allowed atomic.Bool
	allowed.Store(true)
	var starts atomic.Int32
	admit := func(ctx context.Context, start func() error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !allowed.Load() {
			return errors.New("Host Environment Execution Grant revoked before OS Start")
		}
		if err := start(); err != nil {
			return err
		}
		starts.Add(1)
		return nil
	}
	protected, err := pruntime.NewHostAdmittedPinnedLinuxProcess(pinned, admit)
	if err != nil {
		t.Fatal(err)
	}
	// A separate adapter wrapping the SAME image must not replace the exact
	// Host-selected admission verifier and runner at Registry dispatch.
	other, err := pruntime.NewHostAdmittedPinnedLinuxProcess(
		pinned, func(context.Context, func() error) error {
			return errors.New("unselected Host start admission")
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	plan := pruntime.EnvironmentPlan{
		Environment: core.MustCanonicalID("reference"),
		PluginLockFingerprint: req.PluginLockFingerprint,
		Plugins: []pruntime.PluginRuntime{{
			Plugin: req.Plugin,
			DescriptorFingerprint: req.DescriptorFingerprint,
			Boundary: pruntime.BoundaryProcess,
			Trust: pruntime.TrustExternalProcess,
			Artifact: plugin.Artifact{Digest: digest},
			Command: command.Executable,
			Args: command.Args,
			SandboxPolicy: plugin.SandboxRequired,
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: req.Plugin,
			Capability: plugin.Capability{Kind: req.CapabilityKind, ID: req.CapabilityID},
			Boundary: pruntime.BoundaryProcess,
		}},
	}
	invoke := func(reg pruntime.Registry, p pruntime.EnvironmentPlan) (pruntime.Invocation, error) {
		return reg.InvokeWithProvenance(
			context.Background(), p, req.CapabilityKind, req.CapabilityID,
			req.Operation, req.RequestID, map[string]string{"value": "ok"},
		)
	}
	makeHost := func(runner pruntime.SandboxedProcessRunner, net *netEnforcer, fs *fsEnforcer) pruntime.HostSandboxLauncher {
		return pruntime.HostSandboxLauncher{Runner: runner, Network: net, Filesystem: fs}
	}
	net, fs := &netEnforcer{}, &fsEnforcer{}
	selected := pruntime.Registry{
		Verifier: protected,
		Sandboxes: makeHost(protected, net, fs),
	}
	result, err := invoke(selected, plan)
	if err != nil {
		t.Fatal(err)
	}
	if starts.Load() != 1 ||
		!result.Provenance.Sandboxed ||
		result.Provenance.Verification == nil ||
		!result.Provenance.Verification.DigestVerified {
		t.Fatalf("Registry did not use the exact admitted, verified image: starts=%d provenance=%+v", starts.Load(), result.Provenance)
	}
	checkProcessEnvProbe(t, result.Response, "")
	if !net.called || !net.cleaned || !fs.called || !fs.cleaned {
		t.Fatal("Host sandbox policy adapters were not installed and cleaned")
	}

	// The same accepted plan and digest MUST NOT bypass a Host-owned Start
	// denial even after Registry's initial Artifact verification succeeded.
	allowed.Store(false)
	before := starts.Load()
	if _, err := invoke(selected, plan); err == nil ||
		!strings.Contains(err.Error(), "Host Environment Execution Grant revoked before OS Start") {
		t.Fatalf("Registry launched a process after Host admission was revoked: %v", err)
	}
	if starts.Load() != before {
		t.Fatal("revoked Registry call crossed the physical OS Start gate")
	}

	for _, tc := range []struct {
		name     string
		verifier pruntime.ArtifactVerifier
		process  pruntime.ProcessLauncher
		runner   pruntime.SandboxedProcessRunner
		policy   plugin.SandboxPolicy
	}{
		{
			name: "unprotected verifier with protected runner",
			verifier: pinned, runner: protected,
			policy: plugin.SandboxRequired,
		},
		{
			name: "protected verifier with unprotected runner",
			verifier: protected, runner: pinned,
			policy: plugin.SandboxRequired,
		},
		{
			name: "protected verifier with separately admitted runner using same image",
			verifier: protected, runner: other,
			policy: plugin.SandboxRequired,
		},
		{
			name: "protected sandbox runner but separate unprotected process handle",
			verifier: protected, runner: protected, process: pinned,
			policy: plugin.SandboxPreferred,
		},
		{
			name: "protected verifier with unprotected unsandboxed process",
			verifier: protected, process: pinned,
			policy: plugin.SandboxNone,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			net, fs := &netEnforcer{}, &fsEnforcer{}
			p := plan
			p.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
			p.Plugins[0].SandboxPolicy = tc.policy
			reg := pruntime.Registry{Verifier: tc.verifier, Processes: tc.process}
			if tc.runner != nil {
				reg.Sandboxes = makeHost(tc.runner, net, fs)
			}
			if _, err := invoke(reg, p); err == nil ||
				!strings.Contains(err.Error(), "must hold the same image") {
				t.Fatalf("Registry did not reject mismatched admitted image: %v", err)
			}
			if net.called || fs.called || starts.Load() != before {
				t.Fatal("mismatched adapter crossed Host sandbox or OS Start gate")
			}
		})
	}
}
