package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/xeadsource"
)

func receiptTestHasSHA256Digest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	digest, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(digest) == sha256.Size
}

func TestInvocationReceiptExcludesPayloadAndCopiesAuthorityLimits(t *testing.T) {
	secret := "synthetic-private-result"
	execution := InvocationExecution{
		RequestShape:    "reference.request/v0",
		ResultShape:     "reference.result/v0",
		RequestDigest:   "sha256:" + strings.Repeat("a", 64),
		ResultDigest:    "sha256:" + strings.Repeat("b", 64),
		ResultPayload:   []byte(secret),
		AuthorityLimits: []string{"invocation-is-not-runtime-authority"},
	}
	receipt := execution.Receipt()
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) ||
		strings.Contains(string(encoded), "resultPayload") {
		t.Fatalf("receipt disclosed invocation payload: %s", encoded)
	}
	if receipt.Schema != invocationReceiptSchema ||
		receipt.RequestDigest != execution.RequestDigest ||
		receipt.ResultDigest != execution.ResultDigest {
		t.Fatalf("receipt identity is incomplete: %+v", receipt)
	}
	receipt.AuthorityLimits[0] = "changed"
	if execution.AuthorityLimits[0] != "invocation-is-not-runtime-authority" {
		t.Fatal("receipt shares mutable authority limits with execution")
	}
}

func TestSourceAdapterReturnsExactPayloadFreeInvocationReceipt(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	privateText := "synthetic-private-xml-text"
	request := SourceAdapterRequest{
		Format: xeadsource.FormatXead,
		SourceBytes: []byte(
			`<System FormatVersion="1.2"><Table ID="T1"><VendorThing>` +
				privateText +
				`</VendorThing></Table></System>`,
		),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	}
	result, err := registry.ResolveAndInvoke(
		sourceAdapterResolveInput(contract, provider),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	receipt := result.InvocationReceipt
	if receipt.Schema != invocationReceiptSchema ||
		receipt.Resolution != result.Resolution ||
		receipt.RequestShape != contract.RequestShape ||
		receipt.ResultShape != contract.ResultShape ||
		receipt.RequestShapeFingerprint != contract.RequestShapeFingerprint ||
		receipt.ResultShapeFingerprint != contract.ResultShapeFingerprint {
		t.Fatalf("Source Adapter receipt lost contract or provider identity: %+v", receipt)
	}
	for name, value := range map[string]string{
		"request authority": receipt.RequestShapeAuthorityEvidenceFingerprint,
		"result authority": receipt.ResultShapeAuthorityEvidenceFingerprint,
		"request trust": receipt.RequestShapeTrustQualificationFingerprint,
		"result trust": receipt.ResultShapeTrustQualificationFingerprint,
		"request digest": receipt.RequestDigest,
		"result digest": receipt.ResultDigest,
	} {
		if !receiptTestHasSHA256Digest(value) {
			t.Fatalf("%s fingerprint is not exact: %q", name, value)
		}
	}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	requestSum := sha256.Sum256(payload)
	if want := "sha256:" + hex.EncodeToString(requestSum[:]); receipt.RequestDigest != want {
		t.Fatalf("request digest=%s want=%s", receipt.RequestDigest, want)
	}
	sourceSum := sha256.Sum256(request.SourceBytes)
	if want := "sha256:" + hex.EncodeToString(sourceSum[:]); result.SourceArtifactDigest != want {
		t.Fatalf("source digest=%s want=%s", result.SourceArtifactDigest, want)
	}

	inspectionBytes, err := json.Marshal(result.Inspection)
	if err != nil {
		t.Fatal(err)
	}
	resultSum := sha256.Sum256(inspectionBytes)
	if want := "sha256:" + hex.EncodeToString(resultSum[:]); receipt.ResultDigest != want {
		t.Fatalf("result digest=%s want=%s", receipt.ResultDigest, want)
	}

	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), privateText) ||
		strings.Contains(string(encoded), "resultPayload") ||
		strings.Contains(string(encoded), "sourceBytes") ||
		strings.Contains(string(encoded), "transcriptionContract") {
		t.Fatalf("Source Adapter receipt included private source or payload: %s", encoded)
	}
	if !hasSourceAdapterLimit(receipt.AuthorityLimits, "source-adapter-execution-is-read-only-observation") ||
		!hasSourceAdapterLimit(receipt.AuthorityLimits, "capability-resolution-is-not-canonical-acceptance") {
		t.Fatalf("receipt lost authority boundaries: %+v", receipt.AuthorityLimits)
	}
}

func TestSourceAdapterReceiptSeparatesSourceBytesFromCapabilityIdentity(t *testing.T) {
	contract, provider, registry := sourceAdapterRegistryFixture(t)
	request := SourceAdapterRequest{
		Format: xeadsource.FormatXead,
		SourceBytes: []byte(`<System FormatVersion="1.2"><Table ID="T1"/></System>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	}
	left, err := registry.ResolveAndInvoke(sourceAdapterResolveInput(contract, provider), request)
	if err != nil {
		t.Fatal(err)
	}
	request.SourceBytes = append(append([]byte(nil), request.SourceBytes...), ' ')
	right, err := registry.ResolveAndInvoke(sourceAdapterResolveInput(contract, provider), request)
	if err != nil {
		t.Fatal(err)
	}
	if left.InvocationReceipt.RequestDigest == right.InvocationReceipt.RequestDigest {
		t.Fatal("changed source bytes retained the old invocation request digest")
	}
	if left.InvocationReceipt.Resolution.ResolutionFingerprint !=
		right.InvocationReceipt.Resolution.ResolutionFingerprint {
		t.Fatal("source content changed the selected Capability Contract or Provider identity")
	}
}
