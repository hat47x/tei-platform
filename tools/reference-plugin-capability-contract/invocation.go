package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
)

const referenceJSONPayloadCodec = "application/json;profile=tei-reference-v0"

type InvocationImplementation struct {
	Provider        Provider
	RequestShape    string
	ResultShape     string
	PayloadCodec    string
	AuthorityLimits []string
	Invoke          func(context.Context, []byte) ([]byte, error)
}

type InvocationExecution struct {
	Resolution        Resolution `json:"resolution"`
	RequestShape      string     `json:"requestShape"`
	ResultShape       string     `json:"resultShape"`
	PayloadCodec      string     `json:"payloadCodec"`
	RequestDigest     string     `json:"requestDigest"`
	ResultDigest      string     `json:"resultDigest"`
	ResultPayload     []byte     `json:"resultPayload"`
	AuthorityLimits   []string   `json:"authorityLimits"`
}

type InvocationRegistry struct {
	implementations map[string]InvocationImplementation
}

func NewInvocationRegistry(
	implementations []InvocationImplementation,
) (*InvocationRegistry, error) {
	registry := &InvocationRegistry{
		implementations: map[string]InvocationImplementation{},
	}
	for i, implementation := range implementations {
		provider := implementation.Provider
		if implementation.Invoke == nil {
			return nil, fmt.Errorf("capability implementation[%d] has no invoker", i)
		}
		if provider.PluginID == "" ||
			provider.PluginVersion == "" ||
			provider.ImplementationID == "" ||
			provider.ArtifactDigest == "" ||
			!provider.Scope.Valid() ||
			provider.ContractID == "" ||
			provider.ContractVersion == "" ||
			provider.ContractFingerprint == "" ||
			provider.ConformanceProfile == "" {
			return nil, fmt.Errorf(
				"capability implementation[%d] has invalid provider declaration",
				i,
			)
		}
		if implementation.RequestShape == "" ||
			implementation.ResultShape == "" ||
			implementation.PayloadCodec == "" {
			return nil, fmt.Errorf(
				"capability implementation[%d] has incomplete invocation shape",
				i,
			)
		}
		key := providerKey(provider)
		if _, duplicate := registry.implementations[key]; duplicate {
			return nil, fmt.Errorf(
				"capability implementation %s is registered more than once",
				key,
			)
		}
		registry.implementations[key] = implementation
	}
	return registry, nil
}

func (r *InvocationRegistry) ResolveAndInvoke(
	ctx context.Context,
	input ResolveInput,
	requestPayload []byte,
) (InvocationExecution, error) {
	if r == nil {
		return InvocationExecution{}, fmt.Errorf("capability invocation registry is nil")
	}
	if ctx == nil {
		return InvocationExecution{}, fmt.Errorf("capability invocation requires context")
	}
	resolution, err := resolveCapability(input)
	if err != nil {
		return InvocationExecution{}, err
	}
	contract := resolution.Contract
	if contract.RequestShape == "" ||
		contract.ResultShape == "" ||
		contract.PayloadCodec == "" {
		return InvocationExecution{}, fmt.Errorf(
			"capability contract %s@%s has no machine-readable invocation shape",
			contract.ID,
			contract.Version,
		)
	}

	implementation, ok := r.implementations[providerKey(resolution.Provider)]
	if !ok {
		return InvocationExecution{}, fmt.Errorf(
			"resolved provider %s@%s/%s artifact %q is not installed in capability invocation registry",
			resolution.Provider.PluginID,
			resolution.Provider.PluginVersion,
			resolution.Provider.ImplementationID,
			resolution.Provider.ArtifactDigest,
		)
	}
	if implementation.Provider != resolution.Provider {
		return InvocationExecution{}, fmt.Errorf(
			"capability invocation provider declaration differs from resolved provider",
		)
	}
	if implementation.RequestShape != contract.RequestShape ||
		implementation.ResultShape != contract.ResultShape ||
		implementation.PayloadCodec != contract.PayloadCodec {
		return InvocationExecution{}, fmt.Errorf(
			"capability implementation invocation shape does not match contract %s@%s",
			contract.ID,
			contract.Version,
		)
	}

	resultPayload, err := implementation.Invoke(ctx, append([]byte(nil), requestPayload...))
	if err != nil {
		return InvocationExecution{}, err
	}
	requestSum := sha256.Sum256(requestPayload)
	resultSum := sha256.Sum256(resultPayload)

	limits := append([]string(nil), implementation.AuthorityLimits...)
	limits = append(
		limits,
		"capability-resolution-is-not-canonical-acceptance",
		"capability-invocation-does-not-grant-additional-authority",
		"payload-codec-is-contract-scoped-not-core-semantic-model",
	)
	limits = uniqueSortedInvocationStrings(limits)

	return InvocationExecution{
		Resolution:      resolution,
		RequestShape:    contract.RequestShape,
		ResultShape:     contract.ResultShape,
		PayloadCodec:    contract.PayloadCodec,
		RequestDigest:   "sha256:" + hex.EncodeToString(requestSum[:]),
		ResultDigest:    "sha256:" + hex.EncodeToString(resultSum[:]),
		ResultPayload:   append([]byte(nil), resultPayload...),
		AuthorityLimits: limits,
	}, nil
}

func uniqueSortedInvocationStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
