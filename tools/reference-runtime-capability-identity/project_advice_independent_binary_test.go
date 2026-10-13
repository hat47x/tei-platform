package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// Unlike the previous os.Executable() test, this builds an independent binary
// from project-owned source, not a copy of the Go test executable. It is still
// unsigned, unaccepted test material, without pinned-FD launch or real OS sandbox.
func buildIndependentProjectAdviceBinary(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("the independent process reference is qualified for Linux only")
	}
	path := filepath.Join(t.TempDir(), "project-advice-process")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", path,
		"./plugins/reference-project-customer-advice/cmd/reference-advice-process")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build separate project advisory executable: %v\n%s", err, out)
	}
	return path
}

func independentProjectAdviceFixture(t *testing.T, path string) (
	RuntimeEnvironmentPlan, RuntimeProcessRealization,
	pruntime.HostSandboxLauncher, *projectAdviceTestBinaryVerifier,
	ExecutionProfile, RuntimeEnvironmentAuthority,
) {
	t.Helper()
	digest, err := projectAdviceFileDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	candidate.ArtifactDigest = digest
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure the separately compiled program's fixed fixture dispatch matches
	// the existing generic Host derivation, not a newly invented resolver.
	dispatchID, err := genericProcessDispatchID(ref)
	if err != nil {
		t.Fatal(err)
	}
	if dispatchID != core.CanonicalID(
		"runtime_dispatch.d_ce048e07732d7e7aa5d21453c66a94d1d5fadd94e033e3fcd8a1eaec8438af56") {
		t.Fatalf("project process and generic Host dispatch diverged: %s", dispatchID)
	}
	authority := referenceAuthority(ref, nil)
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, authority)
	if err != nil {
		t.Fatal(err)
	}
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     pruntime.StdioProcessLauncher{Env: []string{}},
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin:                       candidate.Plugin,
		Artifact:                     plugin.Artifact{Digest: digest},
		Command:                      pruntime.ProcessCommand{Executable: path},
	}
	return plan, realization, host, &projectAdviceTestBinaryVerifier{path: path},
		profile, authority
}

func TestIndependentProjectAdviceBinaryRunsThroughExistingHostBoundary(t *testing.T) {
	path := buildIndependentProjectAdviceBinary(t)
	plan, realization, host, verifier, profile, authority :=
		independentProjectAdviceFixture(t, path)
	if err := verifyRuntimeEnvironmentPlan(
		plan, plan.RuntimeCapability, profile, authority); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, code, note, kind string
	}{
		{"suggest", "VIP-123", "", "review_suggested"},
		{"no_suggestion", "VIP-123", "reviewed", "no_suggestion"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inv, err := invokeWithExistingHostBoundary(
				context.Background(), plan, realization, verifier, nil,
				host, "evaluate-advice", "standalone-"+tc.name,
				customeradvice.Input{CustomerCode: tc.code, Note: tc.note},
			)
			if err != nil {
				t.Fatal(err)
			}
			var got customeradvice.Result
			if err := json.Unmarshal(inv.Response.Payload, &got); err != nil {
				t.Fatal(err)
			}
			if !got.Advisory || got.Kind != tc.kind ||
				got.Schema != "tei.reference.project-customer-advice/v0" {
				t.Fatalf("independent process returned %+v", got)
			}
			if inv.Provenance.ArtifactVerification == nil ||
				!inv.Provenance.ArtifactVerification.DigestVerified ||
				inv.Provenance.ArtifactVerification.SignatureVerified ||
				inv.Provenance.ArtifactVerification.ProvenanceChecked ||
				inv.Provenance.RuntimeCapability.ArtifactDigest != realization.Artifact.Digest ||
				inv.Provenance.ProcessDispatchID != inv.Response.CapabilityID {
				t.Fatalf("wrong independent process provenance %+v", inv.Provenance)
			}
		})
	}
	if verifier.calls != 2 {
		t.Fatalf("each process invocation needs byte verification, got %d", verifier.calls)
	}
	// A matching file does not make a withdrawn synthetic grant current.
	revoked := authority
	revoked.ExecutionGrants = nil
	if err := verifyRuntimeEnvironmentPlan(
		plan, plan.RuntimeCapability, profile, revoked); err == nil {
		t.Fatal("stale Environment authority was treated as current")
	}
}

func TestIndependentProjectAdviceBinaryRejectsChangedBytesBeforeLaunch(t *testing.T) {
	path := buildIndependentProjectAdviceBinary(t)
	plan, realization, _, verifier, _, _ := independentProjectAdviceFixture(t, path)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("untrusted post-build modification")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	network := &recordingNetworkEnforcer{}
	filesystem := &recordingFilesystemEnforcer{}
	runner := &recordingSandboxRunner{}
	host := pruntime.HostSandboxLauncher{Network: network, Filesystem: filesystem, Runner: runner}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, verifier, nil, host,
		"evaluate-advice", "standalone-bytes-changed",
		customeradvice.Input{CustomerCode: "VIP-123"},
	); err == nil || !strings.Contains(err.Error(), "artifact digest mismatch") {
		t.Fatalf("changed executable was accepted: %v", err)
	}
	if verifier.calls != 1 || runner.Calls != 0 ||
		network.Calls != 0 || filesystem.Calls != 0 {
		t.Fatal("changed executable reached Host Sandbox before digest verification")
	}
}

func TestIndependentProjectAdviceBinaryRejectsUnboundDispatchAndInput(t *testing.T) {
	path := buildIndependentProjectAdviceBinary(t)
	plan, realization, _, _, _, _ := independentProjectAdviceFixture(t, path)
	launcher := pruntime.StdioProcessLauncher{Env: []string{}}
	originalDispatch, err := genericProcessDispatchID(plan.RuntimeCapability)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		req  pruntime.Request
	}{
		{"foreign-dispatch", pruntime.Request{
			ProtocolVersion: pruntime.ProtocolVersion,
			RequestID: "foreign", Plugin: realization.Plugin,
			CapabilityKind: plugin.CapabilityKind(genericProcessDispatchKind),
			CapabilityID: core.MustCanonicalID("runtime_dispatch.forged"),
			Operation: "evaluate-advice",
			Payload: json.RawMessage(`{"customerCode":"VIP-123"}`),
		}},
		{"unknown-operation", pruntime.Request{
			ProtocolVersion: pruntime.ProtocolVersion,
			RequestID: "other-operation", Plugin: realization.Plugin,
			CapabilityKind: plugin.CapabilityKind(genericProcessDispatchKind),
			CapabilityID: originalDispatch, Operation: "write-order",
			Payload: json.RawMessage(`{"customerCode":"VIP-123"}`),
		}},
		{"invalid-business-input", pruntime.Request{
			ProtocolVersion: pruntime.ProtocolVersion,
			RequestID: "invalid-input", Plugin: realization.Plugin,
			CapabilityKind: plugin.CapabilityKind(genericProcessDispatchKind),
			CapabilityID: originalDispatch, Operation: "evaluate-advice",
			Payload: json.RawMessage(`{"customerCode":"<script>"}`),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := launcher.Invoke(context.Background(), realization.Command, tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.Status != pruntime.ResponseError || len(resp.Payload) != 0 {
				t.Fatalf("unbound project operation returned successful payload: %+v", resp)
			}
		})
	}
}
