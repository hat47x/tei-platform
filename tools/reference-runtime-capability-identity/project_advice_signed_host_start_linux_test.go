//go:build linux

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// E42 combines the existing Host Environment start-admission port (#1876)
// with the existing signer-generation start gate on ONE independently built
// project executable. All keys/Grants here are TEST fixtures: neither a
// production Contract acceptance nor an issuer/revocation source.
func TestProjectAdviceSignedStartAdmissionSeparatesEnvironmentAndSignerRevocation(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	candidate.ArtifactDigest = digest
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	dispatch, err := genericProcessDispatchID(ref)
	if err != nil {
		t.Fatal(err)
	}
	const signerRef = "test-host/start-double-authority"
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	trust := pruntime.NewLinuxHostSignerTrust()
	if err := trust.Allow(signerRef, public); err != nil {
		t.Fatal(err)
	}
	message, err := pruntime.LinuxPinnedSignatureMessage(candidate.Plugin, digest, signerRef)
	if err != nil {
		t.Fatal(err)
	}
	proof := pruntime.LinuxPinnedSignatureProof{
		SignatureRef: signerRef,
		Signature: ed25519.Sign(private, message),
	}
	pin := func() *pruntime.PinnedLinuxProcess {
		t.Helper()
		pinned, err := pruntime.PinSignedLinuxProcess(
			context.Background(), trust, candidate.Plugin, path, digest,
			proof, pruntime.StdioProcessLauncher{Env: []string{}},
		)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := pinned.Close(); err != nil {
				t.Error(err)
			}
		})
		return pinned
	}
	held := pin()
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin: candidate.Plugin,
		Artifact: plugin.Artifact{
			Digest: digest,
			SignatureRef: signerRef,
		},
		Command: pruntime.ProcessCommand{
			Executable: path,
			Args: []string{"--tei-expected-dispatch=" + string(dispatch)},
		},
	}
	grantA := referenceAuthority(ref, nil)
	envB := core.MustCanonicalID("reference_runtime_b")
	grantB := referenceAuthority(ref, nil)
	grantB.Environment = envB
	planA, err := buildRuntimeEnvironmentPlan(ref, profile, grantA)
	if err != nil {
		t.Fatal(err)
	}
	planB, err := buildRuntimeEnvironmentPlan(ref, profile, grantB)
	if err != nil {
		t.Fatal(err)
	}
	scopeA := &projectStartOnlyHost{environment: grantA.Environment, authority: grantA}
	scopeB := &projectStartOnlyHost{environment: envB, authority: grantB}
	type admittedRunner struct {
		process *pruntime.HostAdmittedPinnedLinuxProcess
		sandbox pruntime.HostSandboxLauncher
	}
	newRunner := func(scope *projectStartOnlyHost, plan RuntimeEnvironmentPlan, image *pruntime.PinnedLinuxProcess, justBeforeStart func()) admittedRunner {
		t.Helper()
		inner := scope.startAdmission(plan, profile)
		// The callback is Host-controlled and fires AFTER verifier.Verify
		// but BEFORE current Environment and signer checks at cmd.Start.
		gate := pruntime.LinuxHostStartAdmission(func(ctx context.Context, start func() error) error {
			if justBeforeStart != nil {
				justBeforeStart()
			}
			return inner(ctx, start)
		})
		process, err := pruntime.NewHostAdmittedPinnedLinuxProcess(image, gate)
		if err != nil {
			t.Fatal(err)
		}
		return admittedRunner{
			process: process,
			sandbox: pruntime.HostSandboxLauncher{
				Runner: process,
				Network: &recordingNetworkEnforcer{},
				Filesystem: &recordingFilesystemEnforcer{},
			},
		}
	}
	call := func(scope *projectStartOnlyHost, plan RuntimeEnvironmentPlan, runner admittedRunner, request string) (GenericInvocation, error) {
		return invokeWithCurrentHostEnvironment(
			context.Background(), scope.environment, plan, profile, scope.current,
			realization, runner.process, nil, runner.sandbox,
			"evaluate-advice", request,
			customeradvice.Input{CustomerCode: "VIP-123"},
		)
	}
	assertSigned := func(scope *projectStartOnlyHost, plan RuntimeEnvironmentPlan, runner admittedRunner, request string) {
		t.Helper()
		inv, err := call(scope, plan, runner, request)
		if err != nil {
			t.Fatal(err)
		}
		var result customeradvice.Result
		if err := json.Unmarshal(inv.Response.Payload, &result); err != nil {
			t.Fatal(err)
		}
		if result.Kind != "review_suggested" ||
			inv.Provenance.Environment != scope.environment ||
			inv.Provenance.ArtifactVerification == nil ||
			!inv.Provenance.ArtifactVerification.DigestVerified ||
			!inv.Provenance.ArtifactVerification.SignatureVerified ||
			inv.Provenance.ArtifactVerification.ProvenanceChecked {
			t.Fatalf("unexpected signed project response/provenance: %+v / %+v", result, inv.Provenance)
		}
	}

	runnerA := newRunner(scopeA, planA, held, nil)
	runnerB := newRunner(scopeB, planB, held, nil)
	assertSigned(scopeA, planA, runnerA, "signed-admission-a")
	assertSigned(scopeB, planB, runnerB, "signed-admission-b")
	if scopeA.starts.Load() != 1 || scopeB.starts.Load() != 1 {
		t.Fatalf("unexpected actual child starts: A=%d B=%d", scopeA.starts.Load(), scopeB.starts.Load())
	}

	// Environment A revokes AFTER the first current read and artifact
	// verification but BEFORE cmd.Start. B's same signature/image stays usable.
	didRevokeGrant := false
	revokeGrant := newRunner(scopeA, planA, held, func() {
		didRevokeGrant = true
		scopeA.revoke()
	})
	if _, err := call(scopeA, planA, revokeGrant, "signed-admission-a-revoked"); err == nil ||
		!strings.Contains(err.Error(), "OS Start admission rejected") {
		t.Fatalf("revoked Environment reached physical Start: %v", err)
	}
	if !didRevokeGrant || scopeA.starts.Load() != 1 {
		t.Fatal("Environment A revoked after verification but still started child")
	}
	assertSigned(scopeB, planB, runnerB, "signed-admission-b-after-a-revoke")

	// Trust revokes after a SUCCESSFUL signature verification but before
	// cmd.Start. Current Environment B remains authorized; signer start-gate
	// must still prevent OS launch.
	didRevokeSigner := false
	revokeSigner := newRunner(scopeB, planB, held, func() {
		didRevokeSigner = trust.Revoke(signerRef)
	})
	beforeB := scopeB.starts.Load()
	if _, err := call(scopeB, planB, revokeSigner, "signed-admission-b-signer-revoked"); err == nil ||
		!strings.Contains(err.Error(), "Host signer trust was revoked") {
		t.Fatalf("revoked signer reached physical Start: %v", err)
	}
	if !didRevokeSigner || scopeB.starts.Load() != beforeB {
		t.Fatalf("signer revocation failed to stop physical Start: starts=%d", scopeB.starts.Load())
	}
	// Re-allowing IDENTICAL public key is another signer generation;
	// old held image cannot silently inherit that trust.
	if err := trust.Allow(signerRef, public); err != nil {
		t.Fatal(err)
	}
	if _, err := call(scopeB, planB, runnerB, "signed-admission-old-generation"); err == nil ||
		!strings.Contains(err.Error(), "Host signer trust was revoked") {
		t.Fatalf("old signed image unexpectedly revived after key re-registration: %v", err)
	}

	fresh := pin()
	assertSigned(scopeB, planB, newRunner(scopeB, planB, fresh, nil), "signed-admission-new-generation")
	// Restoring signer trust must NOT revive Environment A's revoked Grant.
	if _, err := call(scopeA, planA, newRunner(scopeA, planA, fresh, nil), "signed-admission-a-still-revoked"); err == nil ||
		!strings.Contains(err.Error(), "Execution Grant") {
		t.Fatalf("Environment Grant incorrectly revived by signer change: %v", err)
	}
}
