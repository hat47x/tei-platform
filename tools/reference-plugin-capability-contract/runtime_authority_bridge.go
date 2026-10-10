package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/realization"
)

type RuntimeContractQueryBinding struct {
	CapabilityID core.CanonicalID `json:"capabilityId"`
	Operation    string           `json:"operation"`
}

type RuntimeAuthorizedContractQueryExecutor struct {
	provider Provider
	plan     pruntime.EnvironmentPlan
	registry pruntime.Registry
	binding  RuntimeContractQueryBinding
	seq      atomic.Uint64

	mu          sync.Mutex
	invocations []pruntime.InvocationProvenance
}

func NewRuntimeAuthorizedContractQueryExecutor(
	provider Provider,
	plan pruntime.EnvironmentPlan,
	registry pruntime.Registry,
	binding RuntimeContractQueryBinding,
) (*RuntimeAuthorizedContractQueryExecutor, error) {
	if provider.Scope != ScopeApplicationRuntime {
		return nil, fmt.Errorf(
			"runtime-authorized Contract Query requires application-runtime provider, got %q",
			provider.Scope,
		)
	}
	if provider.ContractID != contractQueryCapabilityContractID {
		return nil, fmt.Errorf(
			"runtime-authorized Contract Query provider contract %q is not %q",
			provider.ContractID,
			contractQueryCapabilityContractID,
		)
	}
	if provider.PluginID == "" || provider.PluginVersion == "" ||
		provider.ImplementationID == "" || provider.ArtifactDigest == "" {
		return nil, fmt.Errorf("runtime-authorized Contract Query provider is incomplete")
	}
	if binding.CapabilityID == "" {
		return nil, fmt.Errorf("runtime-authorized Contract Query requires legacy Capability ID")
	}
	binding.Operation = strings.TrimSpace(binding.Operation)
	if binding.Operation == "" {
		return nil, fmt.Errorf("runtime-authorized Contract Query requires operation")
	}

	pluginID, err := core.ParseCanonicalID(provider.PluginID)
	if err != nil {
		return nil, fmt.Errorf("runtime-authorized provider plugin id: %w", err)
	}
	expectedPlugin := plugin.Identity{
		ID:      pluginID,
		Version: provider.PluginVersion,
	}
	capability, ok := plan.Capability(
		plugin.CapabilityContractQueryExecutor,
		binding.CapabilityID,
	)
	if !ok {
		return nil, fmt.Errorf(
			"Environment Plan does not uniquely authorize Contract Query Capability %s",
			binding.CapabilityID,
		)
	}
	if capability.Plugin != expectedPlugin {
		return nil, fmt.Errorf(
			"generic provider %s@%s does not match Environment Plan provider %s@%s",
			provider.PluginID,
			provider.PluginVersion,
			capability.Plugin.ID,
			capability.Plugin.Version,
		)
	}

	runtimePlugin, err := runtimePluginForIdentity(plan, expectedPlugin)
	if err != nil {
		return nil, err
	}
	if runtimePlugin.Artifact.Digest == "" {
		return nil, fmt.Errorf(
			"Environment Plan provider %s@%s has no exact artifact digest",
			expectedPlugin.ID,
			expectedPlugin.Version,
		)
	}
	if runtimePlugin.Artifact.Digest != provider.ArtifactDigest {
		return nil, fmt.Errorf(
			"generic provider artifact %s does not match Environment Plan artifact %s",
			provider.ArtifactDigest,
			runtimePlugin.Artifact.Digest,
		)
	}

	return &RuntimeAuthorizedContractQueryExecutor{
		provider: provider,
		plan:     plan,
		registry: registry,
		binding:  binding,
	}, nil
}

func runtimePluginForIdentity(
	plan pruntime.EnvironmentPlan,
	identity plugin.Identity,
) (pruntime.PluginRuntime, error) {
	var matches []pruntime.PluginRuntime
	for _, runtimePlugin := range plan.Plugins {
		if runtimePlugin.Plugin == identity {
			matches = append(matches, runtimePlugin)
		}
	}
	if len(matches) != 1 {
		return pruntime.PluginRuntime{}, fmt.Errorf(
			"Environment Plan must contain exactly one runtime Plugin %s@%s; found %d",
			identity.ID,
			identity.Version,
			len(matches),
		)
	}
	return matches[0], nil
}

func (e *RuntimeAuthorizedContractQueryExecutor) ExecuteContractQuery(
	ctx context.Context,
	target realization.ContractQueryTarget,
	contract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	if e == nil {
		return driver.ContractQueryResult{}, fmt.Errorf(
			"runtime-authorized Contract Query executor is nil",
		)
	}
	if err := ctx.Err(); err != nil {
		return driver.ContractQueryResult{}, err
	}
	if !target.Valid() {
		return driver.ContractQueryResult{}, fmt.Errorf(
			"runtime-authorized Contract Query target is incomplete",
		)
	}
	if contract.Name == "" {
		return driver.ContractQueryResult{}, fmt.Errorf(
			"runtime-authorized Contract Query contract has no query name",
		)
	}

	requestID := fmt.Sprintf(
		"generic-contract-query-%d",
		e.seq.Add(1),
	)
	invocation, err := e.registry.InvokeWithProvenance(
		ctx,
		e.plan,
		plugin.CapabilityContractQueryExecutor,
		e.binding.CapabilityID,
		e.binding.Operation,
		requestID,
		ContractQueryInvocationRequest{
			Target:     target,
			Contract:   contract,
			Parameters: parameters,
		},
	)
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	if invocation.Provenance.Plugin.ID.String() != e.provider.PluginID ||
		invocation.Provenance.Plugin.Version != e.provider.PluginVersion ||
		invocation.Provenance.CapabilityKind != plugin.CapabilityContractQueryExecutor ||
		invocation.Provenance.CapabilityID != e.binding.CapabilityID {
		return driver.ContractQueryResult{}, fmt.Errorf(
			"production Runtime provenance does not match generic provider binding",
		)
	}

	var result driver.ContractQueryResult
	if err := decodeReferenceJSONExact(
		invocation.Response.Payload,
		&result,
	); err != nil {
		return driver.ContractQueryResult{}, fmt.Errorf(
			"decode production Runtime Contract Query result: %w",
			err,
		)
	}

	e.mu.Lock()
	e.invocations = append(e.invocations, invocation.Provenance)
	e.mu.Unlock()
	return result, nil
}

func (e *RuntimeAuthorizedContractQueryExecutor) InvocationProvenance() []pruntime.InvocationProvenance {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]pruntime.InvocationProvenance(nil), e.invocations...)
}

func RuntimeAuthorizedContractQueryInvocationImplementation(
	provider Provider,
	plan pruntime.EnvironmentPlan,
	registry pruntime.Registry,
	binding RuntimeContractQueryBinding,
) (InvocationImplementation, *RuntimeAuthorizedContractQueryExecutor, error) {
	executor, err := NewRuntimeAuthorizedContractQueryExecutor(
		provider,
		plan,
		registry,
		binding,
	)
	if err != nil {
		return InvocationImplementation{}, nil, err
	}
	implementation, err := ContractQueryInvocationImplementation(
		provider,
		executor,
	)
	if err != nil {
		return InvocationImplementation{}, nil, err
	}
	implementation.AuthorityLimits = uniqueSortedInvocationStrings(append(
		implementation.AuthorityLimits,
		"runtime-authority-is-enforced-by-production-environment-plan",
		"generic-provider-does-not-grant-permission-resource-or-sandbox-authority",
		"runtime-provenance-remains-outside-generic-invocation-envelope",
	))
	return implementation, executor, nil
}

var _ driver.ContractQueryExecutor = (*RuntimeAuthorizedContractQueryExecutor)(nil)

func runtimeContractQueryResponsePayload(result driver.ContractQueryResult) ([]byte, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode Runtime Contract Query result: %w", err)
	}
	return payload, nil
}
