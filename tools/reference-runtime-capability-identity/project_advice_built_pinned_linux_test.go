//go:build linux

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

func buildStandaloneProjectAdviceExecutable(t *testing.T) (string, string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	path := filepath.Join(t.TempDir(), "reference-project-advice-process")
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false",
		"-o", path, "./plugins/reference-project-customer-advice/cmd/reference-project-advice-process")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building independent project advisory executable: %v: %s", err, output)
	}
	digest, err := projectAdviceFileDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, digest
}

func projectAdvicePinnedFixture(t *testing.T, path, digest string) (
	RuntimeEnvironmentPlan, RuntimeProcessRealization,
	pruntime.HostSandboxLauncher, *pruntime.PinnedLinuxProcess,
) {
	t.Helper()
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	candidate.ArtifactDigest = digest
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, referenceAuthority(ref, nil))
	if err != nil {
		t.Fatal(err)
	}
	dispatch, err := genericProcessDispatchID(ref)
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := pruntime.PinLinuxProcess(
		context.Background(), candidate.Plugin, path, digest,
		pruntime.StdioProcessLauncher{Env: []string{}},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pinned.Close(); err != nil {
			t.Error(err)
		}
	})
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     pinned,
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin:                       candidate.Plugin,
		Artifact:                     plugin.Artifact{Digest: digest},
		Command: pruntime.ProcessCommand{
			Executable: path,
			Args:       []string{"--tei-expected-dispatch=" + string(dispatch)},
		},
	}
	return plan, realization, host, pinned
}

func TestProjectAdviceIndependentlyBuiltPinnedLinuxExecutable(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	plan, realization, host, pinned := projectAdvicePinnedFixture(t, path, digest)
	for _, tc := range []struct {
		code, note, expected string
	}{
		{"VIP-123", "", "review_suggested"},
		{"VIP-123", "checked", "no_suggestion"},
	} {
		inv, err := invokeWithExistingHostBoundary(
			context.Background(), plan, realization, pinned, nil, host,
			"evaluate-advice", "project-advice-pinned-"+tc.expected,
			customeradvice.Input{CustomerCode: tc.code, Note: tc.note},
		)
		if err != nil {
			t.Fatal(err)
		}
		var result customeradvice.Result
		if err := json.Unmarshal(inv.Response.Payload, &result); err != nil {
			t.Fatal(err)
		}
		if result.Kind != tc.expected || !result.Advisory ||
			inv.Provenance.ArtifactVerification == nil ||
			!inv.Provenance.ArtifactVerification.DigestVerified ||
			inv.Provenance.ArtifactVerification.SignatureVerified ||
			inv.Provenance.RuntimeCapability.ArtifactDigest != digest ||
			inv.Provenance.ProcessDispatchID != inv.Response.CapabilityID ||
			inv.Provenance.ResourceLeaseID != "" {
			t.Fatalf("unexpected project advice/provenance result=%+v provenance=%+v", result, inv.Provenance)
		}
	}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, pinned, nil, host,
		"evaluate-advice", "project-advice-pinned-invalid",
		customeradvice.Input{CustomerCode: "<script>"},
	); err == nil || !strings.Contains(err.Error(), "invalid project advisory input") {
		t.Fatalf("child process accepted invalid advisory request: %v", err)
	}
}

func TestProjectAdvicePinnedLinuxRejectsDigestDriftBeforeLaunch(t *testing.T) {
	path, _ := buildStandaloneProjectAdviceExecutable(t)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	// A valid-format but intentionally wrong digest must never produce a held,
	// executable process image. No alternate pathname process is dispatched.
	if _, err := pruntime.PinLinuxProcess(
		context.Background(), candidate.Plugin, path, candidate.ArtifactDigest,
		pruntime.StdioProcessLauncher{},
	); err == nil || !strings.Contains(err.Error(), "differ from expected") {
		t.Fatalf("bad digest unexpectedly pinned an executable: %v", err)
	}
}

func TestProjectAdvicePinnedLinuxSourceReplacementDoesNotChangeExecution(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	plan, realization, host, pinned := projectAdvicePinnedFixture(t, path, digest)
	replacement := path + ".replacement"
	if err := os.WriteFile(replacement, []byte("untrusted replacement"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	inv, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, pinned, nil, host,
		"evaluate-advice", "project-advice-pinned-source-replaced",
		customeradvice.Input{CustomerCode: "VIP-123"},
	)
	if err != nil {
		t.Fatalf("pinned image unexpectedly followed mutable source path: %v", err)
	}
	var result customeradvice.Result
	if err := json.Unmarshal(inv.Response.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Kind != "review_suggested" {
		t.Fatalf("replaced source changed pinned executable behavior: %+v", result)
	}
}
