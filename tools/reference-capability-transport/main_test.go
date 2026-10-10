package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestTransportHelperProcess(t *testing.T) {
	mode := os.Getenv("TEI_TRANSPORT_HELPER")
	if mode == "" {
		return
	}

	var req pruntime.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "decode request:", err)
		os.Exit(20)
	}
	if mode == "fail-secret" {
		fmt.Fprintln(os.Stderr, "transport-secret-do-not-expose")
		os.Exit(23)
	}

	var dispatch GenericDispatch
	if err := json.Unmarshal(req.Payload, &dispatch); err != nil {
		fmt.Fprintln(os.Stderr, "decode dispatch:", err)
		os.Exit(21)
	}
	result := GenericTransportResult{
		Schema:                  dispatch.Schema,
		ContractID:              dispatch.ContractID,
		ContractFingerprint:     dispatch.ContractFingerprint,
		Target:                  dispatch.Target,
		Provider:                dispatch.Provider,
		RequestShapeFingerprint: dispatch.RequestShapeFingerprint,
		ResultShapeFingerprint:  dispatch.ResultShapeFingerprint,
		PayloadCodec:             dispatch.PayloadCodec,
		Payload:                  append([]byte(nil), dispatch.Payload...),
	}
	switch mode {
	case "echo":
	case "tamper-target":
		result.Target = "tampered-target"
	case "tamper-contract":
		result.ContractFingerprint = "sha256:" + strings.Repeat("f", 64)
	case "tamper-provider":
		result.Provider.ImplementationID = "tampered.implementation"
	default:
		fmt.Fprintln(os.Stderr, "unsupported helper mode")
		os.Exit(24)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode result:", err)
		os.Exit(22)
	}
	resp := pruntime.Response{
		ProtocolVersion:       pruntime.ProtocolVersion,
		RequestID:             req.RequestID,
		Plugin:                req.Plugin,
		CapabilityID:          req.CapabilityID,
		DescriptorFingerprint: req.DescriptorFingerprint,
		Status:                pruntime.ResponseOK,
		Payload:               payload,
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		fmt.Fprintln(os.Stderr, "encode response:", err)
		os.Exit(25)
	}
	os.Exit(0)
}

func transportCarrier() LegacyCarrier {
	return LegacyCarrier{
		Plugin: plugin.Identity{
			ID:      core.MustCanonicalID("tei.reference.transport"),
			Version: "1.0.0",
		},
		CapabilityKind:        plugin.CapabilityContractQueryExecutor,
		CapabilityID:          core.MustCanonicalID("transport.legacy-carrier"),
		DescriptorFingerprint: strings.Repeat("d", 64),
		Operation:             "generic-dispatch-v0",
	}
}

func genericDispatch(payload []byte) GenericDispatch {
	return GenericDispatch{
		Schema:              dispatchSchema,
		ContractID:          "reference.generic.echo",
		ContractFingerprint: "sha256:" + strings.Repeat("a", 64),
		Target:              "desktop-test",
		Provider: GenericProviderIdentity{
			PluginID:         "tei.reference.transport",
			PluginVersion:    "1.0.0",
			ImplementationID: "generic.echo.v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("b", 64),
		},
		RequestShapeFingerprint: "sha256:" + strings.Repeat("c", 64),
		ResultShapeFingerprint:  "sha256:" + strings.Repeat("e", 64),
		PayloadCodec:             "application/octet-stream",
		Payload:                  append([]byte(nil), payload...),
	}
}

func helperLauncher(mode string) (pruntime.StdioProcessLauncher, pruntime.ProcessCommand) {
	return pruntime.StdioProcessLauncher{
			Env: append(os.Environ(), "TEI_TRANSPORT_HELPER="+mode),
		},
		pruntime.ProcessCommand{
			Executable: os.Args[0],
			Args:       []string{"-test.run=TestTransportHelperProcess"},
		}
}

func TestProcessTransportRoundTripPreservesGenericIdentityAndOpaquePayload(t *testing.T) {
	launcher, command := helperLauncher("echo")
	payload := []byte{0x00, 0x01, 0x7f, 0x80, 0xff, 'T', 'E', 'I'}
	dispatch := genericDispatch(payload)

	execution, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		transportCarrier(),
		"transport-roundtrip-1",
		dispatch,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(execution.ResultPayload, payload) {
		t.Fatalf("result payload=%v want=%v", execution.ResultPayload, payload)
	}
	receipt := execution.Receipt
	if receipt.GenericContractID != dispatch.ContractID ||
		receipt.ContractFingerprint != dispatch.ContractFingerprint ||
		receipt.Target != dispatch.Target ||
		receipt.Provider != dispatch.Provider ||
		receipt.RequestShapeFingerprint != dispatch.RequestShapeFingerprint ||
		receipt.ResultShapeFingerprint != dispatch.ResultShapeFingerprint ||
		receipt.PayloadCodec != dispatch.PayloadCodec {
		t.Fatalf("generic transport identity drifted: %+v", receipt)
	}
	if receipt.RequestDigest != digestBytes(payload) ||
		receipt.ResultDigest != digestBytes(payload) {
		t.Fatalf("payload digests=%+v", receipt)
	}
	if receipt.LegacyCapabilityKind != plugin.CapabilityContractQueryExecutor ||
		receipt.LegacyCapabilityID != core.MustCanonicalID("transport.legacy-carrier") {
		t.Fatalf("legacy carrier identity was hidden: %+v", receipt)
	}
}

func TestProcessTransportRejectsGenericTargetDrift(t *testing.T) {
	launcher, command := helperLauncher("tamper-target")
	_, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		transportCarrier(),
		"transport-target-drift",
		genericDispatch([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "identity drifted") {
		t.Fatalf("generic Target drift was accepted: %v", err)
	}
}

func TestProcessTransportRejectsGenericContractDrift(t *testing.T) {
	launcher, command := helperLauncher("tamper-contract")
	_, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		transportCarrier(),
		"transport-contract-drift",
		genericDispatch([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "identity drifted") {
		t.Fatalf("generic Contract drift was accepted: %v", err)
	}
}

func TestProcessTransportRejectsGenericProviderDrift(t *testing.T) {
	launcher, command := helperLauncher("tamper-provider")
	_, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		transportCarrier(),
		"transport-provider-drift",
		genericDispatch([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "identity drifted") {
		t.Fatalf("generic Provider drift was accepted: %v", err)
	}
}

func TestProcessTransportRequiresLegacyCarrierIdentity(t *testing.T) {
	launcher, command := helperLauncher("echo")
	carrier := transportCarrier()
	carrier.CapabilityKind = ""
	_, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		carrier,
		"transport-no-carrier",
		genericDispatch([]byte("payload")),
	)
	if err == nil || !strings.Contains(err.Error(), "CapabilityKind and CapabilityID") {
		t.Fatalf("missing legacy carrier identity was accepted: %v", err)
	}
}

func TestProcessTransportDoesNotExposeRawProcessStderr(t *testing.T) {
	launcher, command := helperLauncher("fail-secret")
	_, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		transportCarrier(),
		"transport-secret-error",
		genericDispatch([]byte("payload")),
	)
	if err == nil {
		t.Fatal("failing child process was accepted")
	}
	if strings.Contains(err.Error(), "transport-secret-do-not-expose") {
		t.Fatalf("raw child stderr leaked into Host error: %v", err)
	}
	if !strings.Contains(err.Error(), "stderrSHA256=") ||
		!strings.Contains(err.Error(), "stderrBytes=") {
		t.Fatalf("redacted process diagnostic lost digest metadata: %v", err)
	}
}

func TestTransportReceiptDoesNotPromoteCarrierToGenericAuthority(t *testing.T) {
	launcher, command := helperLauncher("echo")
	execution, err := invokeOverProcessTransport(
		context.Background(),
		launcher,
		command,
		transportCarrier(),
		"transport-authority-limits",
		genericDispatch([]byte("payload")),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"transport-success-is-not-capability-conformance",
		"transport-success-is-not-runtime-authority",
		"transport-success-is-not-canonical-acceptance",
		"legacy-carrier-identity-is-not-generic-contract-identity",
		"generic-provider-and-shape-echo-is-not-runtime-authority",
		"process-response-does-not-echo-legacy-capability-kind",
		"process-stdio-is-one-realization-not-universal-capability-abi",
	} {
		found := false
		for _, got := range execution.Receipt.AuthorityLimits {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("authority limit %q missing: %+v", want, execution.Receipt.AuthorityLimits)
		}
	}
}
