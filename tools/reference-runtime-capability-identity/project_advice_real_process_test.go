package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// This is the existing Go test executable, containing the actual project-owned
// customeradvice package. It is NOT a separately accepted Plugin artifact or
// a signed deployable binary. TestMain handles its one-shot protocol.
func runProjectAdviceProcessHelper(req pruntime.Request) int {
	dispatchID, err := processDispatchID(
		"tei.reference.project.customer-advice",
		"1.0.0",
		fingerprint("hypothetical-project-advice-contract-shape"),
		"project.customer-advice.go",
	)
	if err != nil {
		return 10
	}
	if req.CapabilityKind != plugin.CapabilityKind(genericProcessDispatchKind) ||
		req.CapabilityID != dispatchID || req.Operation != "evaluate-advice" {
		writeGenericProcessHelperError(req, "project advisory dispatch mismatch")
		return 0
	}
	var input customeradvice.Input
	if err := json.Unmarshal(req.Payload, &input); err != nil {
		writeGenericProcessHelperError(req, "invalid project advisory JSON")
		return 0
	}
	result, err := customeradvice.Evaluate(input)
	if err != nil {
		writeGenericProcessHelperError(req, "invalid project advisory input")
		return 0
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return 11
	}
	returnCode := pruntime.Response{
		ProtocolVersion:       req.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseOK,
		Payload:               payload,
	}
	if err := json.NewEncoder(os.Stdout).Encode(returnCode); err != nil {
		return 12
	}
	return 0
}

// This verifier checks the exact bytes of the test executable, rather than
// returning an unconditional DigestVerified=true as other synthetic fixtures
// do. It does not prove issuer trust, signature, provenance or pinned-FD launch.
type projectAdviceTestBinaryVerifier struct {
	path  string
	calls int
}

func (v *projectAdviceTestBinaryVerifier) Verify(
	ctx context.Context,
	_ plugin.Identity,
	artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	if err := ctx.Err(); err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	got, err := projectAdviceFileDigest(v.path)
	if err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	if got != artifact.Digest {
		return pruntime.ArtifactVerification{}, fmt.Errorf("test executable artifact digest mismatch")
	}
	return pruntime.ArtifactVerification{
		VerifierID:        "project-advice-test-executable-sha256",
		DigestVerified:    true,
		SignatureVerified: false,
		ProvenanceChecked: false,
	}, nil
}

func projectAdviceFileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func projectAdviceRealProcessFixture(t *testing.T) (
	RuntimeEnvironmentPlan, RuntimeProcessRealization,
	pruntime.HostSandboxLauncher, *projectAdviceTestBinaryVerifier,
) {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
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
	plan, err := buildRuntimeEnvironmentPlan(ref, profile, referenceAuthority(ref, nil))
	if err != nil {
		t.Fatal(err)
	}
	// Explicit Env avoids passing the host environment and credentials into
	// the diagnostic process. The existing StdioProcessLauncher owns process IO.
	launcher := pruntime.StdioProcessLauncher{Env: []string{genericProcessHelperEnv + "=project-advice"}}
	host := pruntime.HostSandboxLauncher{
		Network:    &recordingNetworkEnforcer{},
		Filesystem: &recordingFilesystemEnforcer{},
		Runner:     launcher,
	}
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin:                       candidate.Plugin,
		Artifact:                     plugin.Artifact{Digest: digest},
		Command:                      pruntime.ProcessCommand{Executable: path},
	}
	return plan, realization, host, &projectAdviceTestBinaryVerifier{path: path}
}

func TestProjectAdviceActualGoCodeViaExistingRuntimeProcessBoundary(t *testing.T) {
	plan, realization, host, verifier := projectAdviceRealProcessFixture(t)
	for _, tc := range []struct {
		code string
		note string
		kind string
	}{
		{"VIP-123", "", "review_suggested"},
		{"VIP-123", "reviewed", "no_suggestion"},
	} {
		inv, err := invokeWithExistingHostBoundary(
			context.Background(), plan, realization, verifier, nil,
			host, "evaluate-advice", "project-advice-"+tc.kind,
			customeradvice.Input{CustomerCode: tc.code, Note: tc.note},
		)
		if err != nil {
			t.Fatal(err)
		}
		var result customeradvice.Result
		if err := json.Unmarshal(inv.Response.Payload, &result); err != nil {
			t.Fatal(err)
		}
		if !result.Advisory || result.Kind != tc.kind ||
			result.Schema != "tei.reference.project-customer-advice/v0" {
			t.Fatalf("actual Go project result=%+v", result)
		}
		if inv.Provenance.ArtifactVerification == nil ||
			!inv.Provenance.ArtifactVerification.DigestVerified ||
			inv.Provenance.RuntimeCapability.ArtifactDigest != realization.Artifact.Digest ||
			inv.Provenance.ProcessDispatchID != inv.Response.CapabilityID ||
			inv.Provenance.ExecutionAuthorityEvidenceFingerprint == "" ||
			len(inv.Provenance.Resources) != 0 ||
			inv.Provenance.ResourceLeaseID != "" {
			t.Fatalf("unexpected process authority/provenance=%+v", inv.Provenance)
		}
	}
	if verifier.calls != 2 {
		t.Fatalf("expected real byte verification for both process invocations: %d", verifier.calls)
	}
}

func TestProjectAdviceRealProcessRejectsIncorrectArtifactBeforeDispatch(t *testing.T) {
	plan, realization, _, _ := projectAdviceRealProcessFixture(t)
	// A self-consistent but untrusted substitute Candidate is insufficient:
	// the verifier must compare its claimed digest with the current test bytes.
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	changed := projectAdviceCandidate(profile)
	changed.ArtifactDigest = "sha256:" + strings.Repeat("0", 64)
	ref, err := deriveRuntimeCapabilityRef(changed, profile)
	if err != nil {
		t.Fatal(err)
	}
	badPlan, err := buildRuntimeEnvironmentPlan(ref, profile, referenceAuthority(ref, nil))
	if err != nil {
		t.Fatal(err)
	}
	realization.RuntimeCapabilityFingerprint = ref.Fingerprint
	realization.Artifact.Digest = changed.ArtifactDigest
	network := &recordingNetworkEnforcer{}
	filesystem := &recordingFilesystemEnforcer{}
	runner := &recordingSandboxRunner{}
	host := pruntime.HostSandboxLauncher{
		Network: network, Filesystem: filesystem, Runner: runner,
	}
	verifier := &projectAdviceTestBinaryVerifier{path: realization.Command.Executable}
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), badPlan, realization, verifier, nil, host,
		"evaluate-advice", "project-advice-drift",
		customeradvice.Input{CustomerCode: "VIP-123"},
	); err == nil || !strings.Contains(err.Error(), "artifact digest mismatch") {
		t.Fatalf("untrusted artifact claim was accepted: %v", err)
	}
	if verifier.calls != 1 || runner.Calls != 0 ||
		network.Calls != 0 || filesystem.Calls != 0 {
		t.Fatal("unverified project binary reached HostSandbox/dispatch")
	}
	// The accepted fixture remains an independent test plan, not an authority
	// or implicit fallback for the substituted Candidate.
	if plan.RuntimeCapability.Fingerprint == badPlan.RuntimeCapability.Fingerprint {
		t.Fatal("artifact change silently reused original Runtime Capability identity")
	}
}

func TestProjectAdviceRealProcessInvalidInputDoesNotBecomeAcceptedAction(t *testing.T) {
	plan, realization, host, verifier := projectAdviceRealProcessFixture(t)
	if _, err := invokeWithExistingHostBoundary(
		context.Background(), plan, realization, verifier, nil,
		host, "evaluate-advice", "project-advice-invalid",
		customeradvice.Input{CustomerCode: "<script>"},
	); err == nil || !strings.Contains(err.Error(), "invalid project advisory input") {
		t.Fatalf("invalid project input was reported as accepted: %v", err)
	}
}
