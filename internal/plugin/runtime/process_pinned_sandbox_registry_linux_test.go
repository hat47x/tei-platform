//go:build linux

package runtime_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// Registry must bind the digest verifier to the exact held image that the
// HostSandboxLauncher will execute, not merely to a matching file path/digest.
func TestRegistryPinnedSandboxRequiresSameVerifiedImage(t *testing.T) {
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
	pinnedA, err := pruntime.PinLinuxProcess(context.Background(), req.Plugin, executable, digest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer pinnedA.Close()
	// Same bytes but a different held image. It is not equivalent to the
	// particular file descriptor whose verification Registry accepted.
	pinnedB, err := pruntime.PinLinuxProcess(context.Background(), req.Plugin, executable, digest, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer pinnedB.Close()

	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: req.PluginLockFingerprint,
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                req.Plugin,
			DescriptorFingerprint: req.DescriptorFingerprint,
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Artifact:              plugin.Artifact{Digest: digest},
			Command:               command.Executable,
			Args:                  command.Args,
			SandboxPolicy:         plugin.SandboxRequired,
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin:     req.Plugin,
			Capability: plugin.Capability{Kind: req.CapabilityKind, ID: req.CapabilityID},
			Boundary:   pruntime.BoundaryProcess,
		}},
	}
	makeHost := func(runner pruntime.SandboxedProcessRunner, net *netEnforcer, fs *fsEnforcer) pruntime.HostSandboxLauncher {
		return pruntime.HostSandboxLauncher{Runner: runner, Network: net, Filesystem: fs}
	}
	invoke := func(reg pruntime.Registry, p pruntime.EnvironmentPlan) (pruntime.Invocation, error) {
		return reg.InvokeWithProvenance(context.Background(), p, req.CapabilityKind, req.CapabilityID,
			req.Operation, req.RequestID, map[string]string{"value": "ok"})
	}
	t.Run("exact held image is accepted in sandbox", func(t *testing.T) {
		net, fs := &netEnforcer{}, &fsEnforcer{}
		reg := pruntime.Registry{Verifier: pinnedA, Sandboxes: makeHost(pinnedA, net, fs)}
		inv, err := invoke(reg, plan)
		if err != nil {
			t.Fatal(err)
		}
		if !inv.Provenance.Sandboxed || inv.Provenance.Verification == nil || !inv.Provenance.Verification.DigestVerified {
			t.Fatal("sandboxed pinned process has no verified provenance")
		}
		checkProcessEnvProbe(t, inv.Response, "")
		if !net.called || !net.cleaned || !fs.called || !fs.cleaned {
			t.Fatal("sandbox host policy adapters were not invoked and cleaned")
		}
	})
	t.Run("preferred sandbox also retains the verified image", func(t *testing.T) {
		preferred := plan
		preferred.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
		preferred.Plugins[0].SandboxPolicy = plugin.SandboxPreferred
		net, fs := &netEnforcer{}, &fsEnforcer{}
		reg := pruntime.Registry{Verifier: pinnedA, Processes: pinnedA, Sandboxes: makeHost(pinnedA, net, fs)}
		inv, err := invoke(reg, preferred)
		if err != nil {
			t.Fatal(err)
		}
		if !inv.Provenance.Sandboxed || !net.called || !fs.called || !net.cleaned || !fs.cleaned {
			t.Fatal("preferred sandbox did not use the verified pinned runner")
		}
		checkProcessEnvProbe(t, inv.Response, "")
	})
	for _, tc := range []struct {
		name    string
		make    func(*netEnforcer, *fsEnforcer) pruntime.Registry
		wantErr string
	}{
		{
			name: "different held image with same digest",
			make: func(n *netEnforcer, f *fsEnforcer) pruntime.Registry {
				return pruntime.Registry{Verifier: pinnedA, Sandboxes: makeHost(pinnedB, n, f)}
			},
			wantErr: "verifier and sandbox runner must hold the same image",
		},
		{
			name: "pathname runner cannot replace pinned image",
			make: func(n *netEnforcer, f *fsEnforcer) pruntime.Registry {
				return pruntime.Registry{Verifier: pinnedA, Sandboxes: makeHost(pruntime.StdioProcessLauncher{}, n, f)}
			},
			wantErr: "verifier and sandbox runner must hold the same image",
		},
		{
			name: "unbound sandbox launcher cannot assert pinned image",
			make: func(_ *netEnforcer, _ *fsEnforcer) pruntime.Registry {
				return pruntime.Registry{Verifier: pinnedA, Sandboxes: &sandboxLauncher{}}
			},
			wantErr: "verifier and sandbox runner must hold the same image",
		},
		{
			name: "independent bound process launcher cannot bypass selected sandbox",
			make: func(n *netEnforcer, f *fsEnforcer) pruntime.Registry {
				return pruntime.Registry{Verifier: pinnedA, Processes: pinnedB, Sandboxes: makeHost(pinnedA, n, f)}
			},
			wantErr: "verifier and sandbox runner must hold the same image",
		},
		{
			name: "unbound verifier cannot vouch for held sandbox image",
			make: func(n *netEnforcer, f *fsEnforcer) pruntime.Registry {
				return pruntime.Registry{Verifier: &verifier{}, Sandboxes: makeHost(pinnedA, n, f)}
			},
			wantErr: "verifier and sandbox runner must hold the same image",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			net, fs := &netEnforcer{}, &fsEnforcer{}
			if _, err := invoke(tc.make(net, fs), plan); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected fail-closed image binding, got %v", err)
			}
			if net.called || fs.called {
				t.Fatal("sandbox adapters ran before image binding was confirmed")
			}
		})
	}
	t.Run("unsandboxed exact process binding remains supported", func(t *testing.T) {
		unsandboxed := plan
		unsandboxed.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
		unsandboxed.Plugins[0].SandboxPolicy = plugin.SandboxNone
		reg := pruntime.Registry{Verifier: pinnedA, Processes: pinnedA}
		inv, err := invoke(reg, unsandboxed)
		if err != nil {
			t.Fatal(err)
		}
		if inv.Provenance.Sandboxed {
			t.Fatal("plain pinned invocation must not be labelled sandboxed")
		}
		checkProcessEnvProbe(t, inv.Response, "")
	})
}
