package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
)

// The legacy process envelope is only the carrier. Its reference fingerprints
// are NOT the generic Service Contract / candidate shape fingerprints.
func quoteLegacyFingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func quoteLegacyProcessDispatch(ctx context.Context, executable string, provider Provider, payload []byte) ([]byte, error) {
	return quoteLegacyProcessDispatchUsing(ctx, provider, payload, func(ctx context.Context, req pruntime.Request) (pruntime.Response, error) {
		return (pruntime.StdioProcessLauncher{}).Invoke(
			ctx, pruntime.ProcessCommand{Executable: executable}, req,
		)
	})
}

// Shared wire encoder/decoder: the callback chooses whether the same legacy
// carrier goes directly to the process or through the product Host Runtime.
// Neither path treats the legacy fingerprint as a Service Contract identity.
func quoteLegacyProcessDispatchUsing(
	ctx context.Context, provider Provider, payload []byte,
	invoke func(context.Context, pruntime.Request) (pruntime.Response, error),
) ([]byte, error) {
	carrierPlugin := plugin.Identity{
		ID: core.MustCanonicalID(provider.PluginID), Version: provider.PluginVersion,
	}
	const descriptor = "reference-order-quote.transport-carrier/v1"
	dispatch := struct {
		Schema              string `json:"schema"`
		ContractID          string `json:"contractId"`
		ContractFingerprint string `json:"contractFingerprint"`
		Target              string `json:"target"`
		Provider            struct {
			PluginID         string `json:"pluginId"`
			PluginVersion    string `json:"pluginVersion"`
			ImplementationID string `json:"implementationId"`
			ArtifactDigest   string `json:"artifactDigest"`
		} `json:"provider"`
		RequestShapeFingerprint string `json:"requestShapeFingerprint"`
		ResultShapeFingerprint  string `json:"resultShapeFingerprint"`
		PayloadCodec            string `json:"payloadCodec"`
		Payload                 []byte `json:"payload"`
	}{
		Schema:                  "tei.reference.capability-transport-dispatch/v0",
		ContractID:              provider.ContractID,
		ContractFingerprint:     quoteLegacyFingerprint("reference.order.quote-lines.contract/v1"),
		Target:                  "reference/order_entry",
		RequestShapeFingerprint: quoteLegacyFingerprint("reference.order.quote-lines.request/v1"),
		ResultShapeFingerprint:  quoteLegacyFingerprint("reference.order.quote-lines.result/v1"),
		PayloadCodec:            "application/json",
		Payload:                 payload,
	}
	dispatch.Provider.PluginID = provider.PluginID
	dispatch.Provider.PluginVersion = provider.PluginVersion
	dispatch.Provider.ImplementationID = provider.ImplementationID
	dispatch.Provider.ArtifactDigest = provider.ArtifactDigest
	encoded, err := json.Marshal(dispatch)
	if err != nil {
		return nil, err
	}
	req := pruntime.Request{
		ProtocolVersion:       1,
		RequestID:             "quote-candidate-real-process",
		Plugin:                carrierPlugin,
		CapabilityKind:        plugin.CapabilityContractQueryExecutor,
		CapabilityID:          core.MustCanonicalID("transport.order_quote_carrier"),
		DescriptorFingerprint: quoteLegacyFingerprint(descriptor),
		Operation:             "generic-dispatch-v0",
		Payload:               encoded,
	}
	resp, err := invoke(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Status != pruntime.ResponseOK ||
		resp.ProtocolVersion != req.ProtocolVersion ||
		resp.RequestID != req.RequestID ||
		resp.Plugin != req.Plugin ||
		resp.CapabilityID != req.CapabilityID ||
		resp.DescriptorFingerprint != req.DescriptorFingerprint {
		return nil, fmt.Errorf("legacy process carrier response identity drifted or failed")
	}
	var result struct {
		Schema              string `json:"schema"`
		ContractID          string `json:"contractId"`
		ContractFingerprint string `json:"contractFingerprint"`
		Target              string `json:"target"`
		Provider            struct {
			PluginID         string `json:"pluginId"`
			PluginVersion    string `json:"pluginVersion"`
			ImplementationID string `json:"implementationId"`
			ArtifactDigest   string `json:"artifactDigest"`
		} `json:"provider"`
		RequestShapeFingerprint string `json:"requestShapeFingerprint"`
		ResultShapeFingerprint  string `json:"resultShapeFingerprint"`
		PayloadCodec            string `json:"payloadCodec"`
		Payload                 []byte `json:"payload"`
	}
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return nil, err
	}
	if result.Schema != "tei.reference.capability-transport-result/v0" ||
		result.ContractID != dispatch.ContractID ||
		result.ContractFingerprint != dispatch.ContractFingerprint ||
		result.Target != dispatch.Target ||
		result.RequestShapeFingerprint != dispatch.RequestShapeFingerprint ||
		result.ResultShapeFingerprint != dispatch.ResultShapeFingerprint ||
		result.PayloadCodec != dispatch.PayloadCodec ||
		result.Provider.PluginID != dispatch.Provider.PluginID ||
		result.Provider.PluginVersion != dispatch.Provider.PluginVersion ||
		result.Provider.ImplementationID != dispatch.Provider.ImplementationID ||
		result.Provider.ArtifactDigest != dispatch.Provider.ArtifactDigest {
		return nil, fmt.Errorf("legacy process generic result identity drifted")
	}
	return result.Payload, nil
}

// This bridges the EXISTING #951 candidate resolver and shape gates to an
// actually built independent business executable. It does not publish the
// candidate shapes or turn synthetic conformance into production Authority.
func TestQuoteCandidateContractInvokesIndependentBusinessProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the reference stdio launcher restricts executable path whitespace on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "quote-provider")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./plugins/reference-order-quote")
	build.Dir = filepath.Join("..", "..")
	build.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent quote provider: %v\n%s", err, output)
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(hash[:])

	ignoredStubCalls := 0
	contract, provider, implementation := quoteCandidateRegistryFixture(t, &ignoredStubCalls)
	provider.ArtifactDigest = digest
	implementation.Provider = provider
	hostAllowed := true
	processCalls := 0
	implementation.Invoke = func(ctx context.Context, request []byte) ([]byte, error) {
		if !hostAllowed {
			return nil, fmt.Errorf("test-only Host approval withdrawn")
		}
		image, err := os.ReadFile(binary)
		if err != nil {
			return nil, fmt.Errorf("read selected artifact: %w", err)
		}
		sum := sha256.Sum256(image)
		if "sha256:"+hex.EncodeToString(sum[:]) != provider.ArtifactDigest {
			return nil, fmt.Errorf("selected executable no longer matches provider artifact")
		}
		processCalls++
		return quoteLegacyProcessDispatch(ctx, binary, provider, request)
	}
	implementation.AuthorityLimits = []string{
		"shape-publisher-trust-is-synthetic-test-only",
		"provider-conformance-evidence-is-synthetic-test-only",
		"host-approval-and-executable-digest-check-are-test-only",
		"legacy-carrier-does-not-prove-semantic-contract-fingerprint",
		"no-production-sandbox-or-permission-was-verified",
	}
	// These grants are synthesized by the test fixture. A candidate in the
	// repository has no authority to supply or bless these grants itself.
	registry := newInvocationRegistryForTest(t, []InvocationImplementation{implementation})
	resolve := ResolveInput{
		Scope:       ScopeApplicationRuntime,
		Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
		Contracts:   []Contract{contract},
		Providers:   []Provider{provider},
	}
	payload := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":3,"unitPriceMinor":250},{"sku":"BOOK","quantity":2,"unitPriceMinor":400}]}`)

	if _, err := registry.ResolveAndInvoke(ctx, resolve, payload); err == nil ||
		!strings.Contains(err.Error(), "no conformant provider") || processCalls != 0 {
		t.Fatalf("synthetic shape trust alone invoked process: err=%v calls=%d", err, processCalls)
	}
	resolve.Evidence = []ConformanceEvidence{
		evidenceFor(provider, contract, "synthetic-quote-process-conformance"),
	}
	t.Run("shape drift must stop before child process", func(t *testing.T) {
		invalid := []byte(`{"currency":"JPY","lines":[{"sku":"PEN","quantity":"3","unitPriceMinor":250}]}`)
		if _, err := registry.ResolveAndInvoke(ctx, resolve, invalid); err == nil ||
			!strings.Contains(err.Error(), "request does not conform") || processCalls != 0 {
			t.Fatalf("shape-invalid request reached executable: err=%v calls=%d", err, processCalls)
		}
	})
	t.Run("wrong artifact evidence cannot use installed executable", func(t *testing.T) {
		untrusted := resolve
		bad := provider
		bad.ArtifactDigest = quoteLegacyFingerprint("different-quote-executable")
		untrusted.Providers = []Provider{bad}
		untrusted.Evidence = []ConformanceEvidence{
			evidenceFor(bad, contract, "synthetic-wrong-artifact"),
		}
		if _, err := registry.ResolveAndInvoke(ctx, untrusted, payload); err == nil || processCalls != 0 {
			t.Fatalf("different physical artifact was allowed: err=%v calls=%d", err, processCalls)
		}
	})
	t.Run("Host denial cannot be overridden by resolution", func(t *testing.T) {
		hostAllowed = false
		defer func() { hostAllowed = true }()
		if _, err := registry.ResolveAndInvoke(ctx, resolve, payload); err == nil || processCalls != 0 {
			t.Fatalf("revoked test Host permission reached process: err=%v calls=%d", err, processCalls)
		}
	})
	execution, err := registry.ResolveAndInvoke(ctx, resolve, payload)
	if err != nil {
		t.Fatal(err)
	}
	var quote struct {
		Currency      string `json:"currency"`
		LineCount     int    `json:"lineCount"`
		SubtotalMinor int64  `json:"subtotalMinor"`
	}
	if err := json.Unmarshal(execution.ResultPayload, &quote); err != nil {
		t.Fatal(err)
	}
	if processCalls != 1 || ignoredStubCalls != 0 ||
		quote.Currency != "JPY" || quote.LineCount != 2 || quote.SubtotalMinor != 1550 ||
		execution.Resolution.Provider != provider ||
		execution.ResultShapeAuthorityEvidenceFingerprint == "" ||
		execution.ResultShapeTrustQualificationFingerprint == "" {
		t.Fatalf("candidate contract did not reach real business result: quote=%+v calls=%d resolution=%+v", quote, processCalls, execution.Resolution)
	}
	for _, limit := range implementation.AuthorityLimits {
		if !containsString(execution.AuthorityLimits, limit) {
			t.Fatalf("reference authority limitation %s was concealed", limit)
		}
	}
	// A changed executable must be rejected BEFORE a new child starts.
	file, err := os.OpenFile(binary, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("untrusted byte")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.ResolveAndInvoke(ctx, resolve, payload); err == nil ||
		!strings.Contains(err.Error(), "no longer matches provider artifact") || processCalls != 1 {
		t.Fatalf("modified selected executable reached process: err=%v calls=%d", err, processCalls)
	}
}
