package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/permission"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/realization"
)

type runtimeAuthorityBuiltin struct {
	identity              plugin.Identity
	descriptorFingerprint string
	calls                 int
	lastRequest           pruntime.Request
	result                driver.ContractQueryResult
}

func (b *runtimeAuthorityBuiltin) Identity() plugin.Identity {
	return b.identity
}

func (b *runtimeAuthorityBuiltin) DescriptorFingerprint() string {
	return b.descriptorFingerprint
}

func (b *runtimeAuthorityBuiltin) Invoke(
	ctx context.Context,
	req pruntime.Request,
) (pruntime.Response, error) {
	if err := ctx.Err(); err != nil {
		return pruntime.Response{}, err
	}
	b.calls++
	b.lastRequest = req
	if req.Operation != "execute-contract-query" {
		return pruntime.Response{
			Status: pruntime.ResponseError,
			Error:  "unexpected operation",
		}, nil
	}
	var request ContractQueryInvocationRequest
	if err := json.Unmarshal(req.Payload, &request); err != nil {
		return pruntime.Response{}, err
	}
	if !request.Target.Valid() || request.Contract.Name == "" {
		return pruntime.Response{
			Status: pruntime.ResponseError,
			Error:  "invalid contract query request",
		}, nil
	}
	payload, err := runtimeContractQueryResponsePayload(b.result)
	if err != nil {
		return pruntime.Response{}, err
	}
	return pruntime.Response{
		Status:  pruntime.ResponseOK,
		Payload: payload,
	}, nil
}

type acceptingRuntimeArtifactVerifier struct {
	calls int
}

func (v *acceptingRuntimeArtifactVerifier) Verify(
	_ context.Context,
	_ plugin.Identity,
	_ plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	return pruntime.ArtifactVerification{
		VerifierID:        "reference.runtime-authority-test",
		DigestVerified:    true,
		SignatureVerified: true,
		ProvenanceChecked: true,
	}, nil
}

type runtimeAuthorityFixture struct {
	contract     Contract
	provider     Provider
	capabilityID core.CanonicalID
	plan         pruntime.EnvironmentPlan
	registry     pruntime.Registry
	builtin      *runtimeAuthorityBuiltin
	verifier     *acceptingRuntimeArtifactVerifier
}

func newRuntimeAuthorityFixture(t *testing.T) runtimeAuthorityFixture {
	t.Helper()
	contract := runtimeContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeApplicationRuntime,
		"reference.runtime.contract_query",
		"contract-query.executor.v1",
	)
	pluginID := core.MustCanonicalID(provider.PluginID)
	identity := plugin.Identity{
		ID:      pluginID,
		Version: provider.PluginVersion,
	}
	capabilityID := core.MustCanonicalID("database.query.reference")
	descriptorFingerprint := strings.Repeat("d", 64)

	builtin := &runtimeAuthorityBuiltin{
		identity:              identity,
		descriptorFingerprint: descriptorFingerprint,
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{{
				Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("customer.id"): "C-001",
				},
			}},
			Payload: map[string]any{"provider": "reference"},
		},
	}
	verifier := &acceptingRuntimeArtifactVerifier{}
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("development"),
		PluginLockFingerprint: strings.Repeat("e", 64),
		RuntimePolicyFingerprint: strings.Repeat("f", 64),
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                identity,
			DescriptorFingerprint: descriptorFingerprint,
			Boundary:              pruntime.BoundaryInProcess,
			Trust:                 pruntime.TrustHostTrusted,
			Artifact: plugin.Artifact{
				Digest: provider.ArtifactDigest,
			},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity,
			Capability: plugin.Capability{
				Kind: plugin.CapabilityContractQueryExecutor,
				ID:   capabilityID,
			},
			Boundary: pruntime.BoundaryInProcess,
		}},
	}
	registry := pruntime.Registry{
		Builtins: map[string]pruntime.Builtin{
			string(identity.ID) + "@" + identity.Version: builtin,
		},
		Verifier: verifier,
	}
	return runtimeAuthorityFixture{
		contract:     contract,
		provider:     provider,
		capabilityID: capabilityID,
		plan:         plan,
		registry:     registry,
		builtin:      builtin,
		verifier:     verifier,
	}
}

func runtimeAuthorityGenericRegistry(
	t *testing.T,
	fixture runtimeAuthorityFixture,
) (*InvocationRegistry, *RuntimeAuthorizedContractQueryExecutor) {
	t.Helper()
	implementation, executor, err :=
		RuntimeAuthorizedContractQueryInvocationImplementation(
			fixture.provider,
			fixture.plan,
			fixture.registry,
			RuntimeContractQueryBinding{
				CapabilityID: fixture.capabilityID,
				Operation:    "execute-contract-query",
			},
		)
	if err != nil {
		t.Fatal(err)
	}
	return newInvocationRegistryForTest(
		t,
		[]InvocationImplementation{implementation},
	), executor
}

func runtimeAuthorityRequestPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := json.Marshal(ContractQueryInvocationRequest{
		Target: realization.ContractQueryTarget{
			Database: core.MustCanonicalID("sales"),
			Contract: core.MustCanonicalID("customer_read"),
			Query:    core.MustLocalName("find_by_id"),
		},
		Contract: database.QueryContract{
			Name:              "find_by_id",
			Result:            core.MustCanonicalID("customer"),
			ResultCardinality: database.QueryResultMany,
		},
		Parameters: map[string]any{"customer_id": "C-001"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestGenericContractQueryCanReuseProductionRuntimeAuthority(t *testing.T) {
	fixture := newRuntimeAuthorityFixture(t)
	registry, executor := runtimeAuthorityGenericRegistry(t, fixture)

	execution, err := registry.ResolveAndInvoke(
		context.Background(),
		ResolveInput{
			Scope:       ScopeApplicationRuntime,
			Requirement: Requirement{
				ContractID:      fixture.contract.ID,
				CompatibleMajor: 1,
			},
			Contracts: []Contract{fixture.contract},
			Providers: []Provider{fixture.provider},
			Evidence: []ConformanceEvidence{
				evidenceFor(
					fixture.provider,
					fixture.contract,
					"runtime-authority-conformance",
				),
			},
		},
		runtimeAuthorityRequestPayload(t),
	)
	if err != nil {
		t.Fatal(err)
	}

	if fixture.builtin.calls != 1 {
		t.Fatalf("production Runtime builtin calls=%d want=1", fixture.builtin.calls)
	}
	if fixture.verifier.calls != 1 {
		t.Fatalf("production Runtime artifact verifier calls=%d want=1", fixture.verifier.calls)
	}
	if fixture.builtin.lastRequest.CapabilityKind != plugin.CapabilityContractQueryExecutor ||
		fixture.builtin.lastRequest.CapabilityID != fixture.capabilityID ||
		fixture.builtin.lastRequest.Plugin.ID != core.MustCanonicalID(fixture.provider.PluginID) {
		t.Fatalf("production Runtime request=%+v", fixture.builtin.lastRequest)
	}

	var result driver.ContractQueryResult
	if err := json.Unmarshal(execution.ResultPayload, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 ||
		result.Rows[0].Attributes[core.MustCanonicalID("customer.id")] != "C-001" {
		t.Fatalf("generic Runtime result=%+v", result)
	}

	provenance := executor.InvocationProvenance()
	if len(provenance) != 1 {
		t.Fatalf("Runtime provenance count=%d want=1", len(provenance))
	}
	if provenance[0].Environment != fixture.plan.Environment ||
		provenance[0].PluginLockFingerprint != fixture.plan.PluginLockFingerprint ||
		provenance[0].CapabilityKind != plugin.CapabilityContractQueryExecutor ||
		provenance[0].CapabilityID != fixture.capabilityID ||
		provenance[0].Boundary != pruntime.BoundaryInProcess ||
		provenance[0].Verification == nil ||
		!provenance[0].Verification.DigestVerified {
		t.Fatalf("Runtime provenance=%+v", provenance[0])
	}
	for _, limit := range []string{
		"runtime-authority-is-enforced-by-production-environment-plan",
		"generic-provider-does-not-grant-permission-resource-or-sandbox-authority",
		"runtime-provenance-remains-outside-generic-invocation-envelope",
	} {
		if !hasSourceAdapterLimit(execution.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q missing: %+v", limit, execution.AuthorityLimits)
		}
	}
}

func TestRuntimeAuthorityBridgeRejectsProviderIdentityMismatch(t *testing.T) {
	fixture := newRuntimeAuthorityFixture(t)
	mismatched := fixture.provider
	mismatched.PluginID = "reference.runtime.other"

	_, err := NewRuntimeAuthorizedContractQueryExecutor(
		mismatched,
		fixture.plan,
		fixture.registry,
		RuntimeContractQueryBinding{
			CapabilityID: fixture.capabilityID,
			Operation:    "execute-contract-query",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not match Environment Plan provider") {
		t.Fatalf("provider identity mismatch was accepted: %v", err)
	}
}

func TestRuntimeAuthorityBridgeRejectsProviderArtifactMismatch(t *testing.T) {
	fixture := newRuntimeAuthorityFixture(t)
	mismatched := fixture.provider
	mismatched.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)

	_, err := NewRuntimeAuthorizedContractQueryExecutor(
		mismatched,
		fixture.plan,
		fixture.registry,
		RuntimeContractQueryBinding{
			CapabilityID: fixture.capabilityID,
			Operation:    "execute-contract-query",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not match Environment Plan artifact") {
		t.Fatalf("provider artifact mismatch was accepted: %v", err)
	}
}

func TestRuntimeAuthorityBridgeRejectsCapabilityMissingFromEnvironmentPlan(t *testing.T) {
	fixture := newRuntimeAuthorityFixture(t)
	plan := fixture.plan
	plan.Capabilities = nil

	_, err := NewRuntimeAuthorizedContractQueryExecutor(
		fixture.provider,
		plan,
		fixture.registry,
		RuntimeContractQueryBinding{
			CapabilityID: fixture.capabilityID,
			Operation:    "execute-contract-query",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not uniquely authorize") {
		t.Fatalf("missing Environment Capability was accepted: %v", err)
	}
}

func TestGenericInvocationCannotBypassProductionRuntimePermissionGate(t *testing.T) {
	fixture := newRuntimeAuthorityFixture(t)
	required := permission.Spec{
		Kind:   permission.KindNetwork,
		Access: permission.AccessConnect,
		Scope:  "database.primary",
	}
	fixture.plan.Capabilities[0].Capability.Permissions = []permission.Spec{required}
	fixture.plan.Capabilities[0].Permissions = nil

	registry, executor := runtimeAuthorityGenericRegistry(t, fixture)
	_, err := registry.ResolveAndInvoke(
		context.Background(),
		ResolveInput{
			Scope:       ScopeApplicationRuntime,
			Requirement: Requirement{
				ContractID:      fixture.contract.ID,
				CompatibleMajor: 1,
			},
			Contracts: []Contract{fixture.contract},
			Providers: []Provider{fixture.provider},
			Evidence: []ConformanceEvidence{
				evidenceFor(
					fixture.provider,
					fixture.contract,
					"runtime-authority-conformance",
				),
			},
		},
		runtimeAuthorityRequestPayload(t),
	)
	if err == nil || !strings.Contains(err.Error(), "requires permission") {
		t.Fatalf("missing production Runtime permission was bypassed: %v", err)
	}
	if fixture.builtin.calls != 0 {
		t.Fatalf("domain implementation ran before Permission gate: calls=%d", fixture.builtin.calls)
	}
	if fixture.verifier.calls != 0 {
		t.Fatalf("artifact verification ran before Permission gate: calls=%d", fixture.verifier.calls)
	}
	if len(executor.InvocationProvenance()) != 0 {
		t.Fatalf("failed invocation was recorded as successful Runtime provenance")
	}
}

func TestRuntimeAuthorityBridgeDoesNotInferLegacyCapabilityFromImplementationID(t *testing.T) {
	fixture := newRuntimeAuthorityFixture(t)
	otherCapability := core.MustCanonicalID("database.query.other")

	_, err := NewRuntimeAuthorizedContractQueryExecutor(
		fixture.provider,
		fixture.plan,
		fixture.registry,
		RuntimeContractQueryBinding{
			CapabilityID: otherCapability,
			Operation:    "execute-contract-query",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not uniquely authorize") {
		t.Fatalf("legacy Capability was inferred from implementation identity: %v", err)
	}
}
