package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"tei/internal/xeadsource"
)

const (
	sourceAdapterCapabilityContractID   = "authoring.source-adapter.xead"
	referenceXeadReaderImplementationID = "xead.reader.v0"
)

type SourceAdapterRequest struct {
	Format                xeadsource.Format
	SourceBytes           []byte
	TranscriptionContract []byte
}

type SourceAdapterImplementation struct {
	Provider Provider
	Invoke   func(SourceAdapterRequest) (xeadsource.Inspection, error)
}

type SourceAdapterExecution struct {
	Resolution           Resolution            `json:"resolution"`
	SourceArtifactDigest string                `json:"sourceArtifactDigest"`
	Inspection           xeadsource.Inspection `json:"inspection"`
	AuthorityLimits      []string              `json:"authorityLimits"`
}

type SourceAdapterRegistry struct {
	implementations map[string]SourceAdapterImplementation
}

func NewSourceAdapterRegistry(
	implementations []SourceAdapterImplementation,
) (*SourceAdapterRegistry, error) {
	registry := &SourceAdapterRegistry{
		implementations: map[string]SourceAdapterImplementation{},
	}
	for i, implementation := range implementations {
		provider := implementation.Provider
		if implementation.Invoke == nil {
			return nil, fmt.Errorf("source adapter implementation[%d] has no invoker", i)
		}
		if provider.Scope != ScopeAuthoringTool ||
			provider.ContractID != sourceAdapterCapabilityContractID ||
			provider.PluginID == "" ||
			provider.PluginVersion == "" ||
			provider.ImplementationID == "" ||
			provider.ArtifactDigest == "" ||
			provider.ContractVersion == "" ||
			provider.ContractFingerprint == "" ||
			provider.ConformanceProfile == "" {
			return nil, fmt.Errorf(
				"source adapter implementation[%d] has invalid provider declaration",
				i,
			)
		}
		key := providerKey(provider)
		if _, duplicate := registry.implementations[key]; duplicate {
			return nil, fmt.Errorf("source adapter implementation %s is registered more than once", key)
		}
		registry.implementations[key] = implementation
	}
	return registry, nil
}

func ReferenceXeadSourceAdapterImplementation(
	provider Provider,
) (SourceAdapterImplementation, error) {
	if provider.Scope != ScopeAuthoringTool ||
		provider.ContractID != sourceAdapterCapabilityContractID ||
		provider.ImplementationID != referenceXeadReaderImplementationID {
		return SourceAdapterImplementation{}, fmt.Errorf(
			"provider %s@%s/%s is not the Reference Xead/Xeaf reader",
			provider.PluginID,
			provider.PluginVersion,
			provider.ImplementationID,
		)
	}
	return SourceAdapterImplementation{
		Provider: provider,
		Invoke: func(request SourceAdapterRequest) (xeadsource.Inspection, error) {
			if len(request.SourceBytes) == 0 {
				return xeadsource.Inspection{}, fmt.Errorf("source adapter requires exact source bytes")
			}
			if len(request.TranscriptionContract) == 0 {
				return xeadsource.Inspection{}, fmt.Errorf("source adapter requires transcription contract bytes")
			}
			switch request.Format {
			case xeadsource.FormatXead, xeadsource.FormatXeaf:
			default:
				return xeadsource.Inspection{}, fmt.Errorf(
					"unsupported source adapter format %q",
					request.Format,
				)
			}
			return xeadsource.InspectWithContract(
				request.Format,
				request.SourceBytes,
				request.TranscriptionContract,
			)
		},
	}, nil
}

func (r *SourceAdapterRegistry) ResolveAndInvoke(
	input ResolveInput,
	request SourceAdapterRequest,
) (SourceAdapterExecution, error) {
	if r == nil {
		return SourceAdapterExecution{}, fmt.Errorf("source adapter registry is nil")
	}
	if input.Scope != ScopeAuthoringTool {
		return SourceAdapterExecution{}, fmt.Errorf(
			"source adapter execution requires %q scope, got %q",
			ScopeAuthoringTool,
			input.Scope,
		)
	}
	if input.Requirement.ContractID != sourceAdapterCapabilityContractID {
		return SourceAdapterExecution{}, fmt.Errorf(
			"source adapter execution requires contract %q, got %q",
			sourceAdapterCapabilityContractID,
			input.Requirement.ContractID,
		)
	}

	resolution, err := resolveCapability(input)
	if err != nil {
		return SourceAdapterExecution{}, err
	}
	if resolution.Contract.ID != sourceAdapterCapabilityContractID ||
		resolution.Provider.Scope != ScopeAuthoringTool {
		return SourceAdapterExecution{}, fmt.Errorf(
			"resolved capability is not an authoring Source Adapter",
		)
	}

	implementation, ok := r.implementations[providerKey(resolution.Provider)]
	if !ok {
		return SourceAdapterExecution{}, fmt.Errorf(
			"resolved provider %s@%s/%s artifact %q is not installed in source adapter registry",
			resolution.Provider.PluginID,
			resolution.Provider.PluginVersion,
			resolution.Provider.ImplementationID,
			resolution.Provider.ArtifactDigest,
		)
	}
	if implementation.Provider != resolution.Provider {
		return SourceAdapterExecution{}, fmt.Errorf(
			"source adapter registry provider declaration differs from resolved provider",
		)
	}

	inspection, err := implementation.Invoke(request)
	if err != nil {
		return SourceAdapterExecution{}, err
	}
	sum := sha256.Sum256(request.SourceBytes)
	expectedDigest := hex.EncodeToString(sum[:])
	if inspection.Document.SHA256 != expectedDigest {
		return SourceAdapterExecution{}, fmt.Errorf(
			"source adapter inspection digest %q differs from exact input digest %q",
			inspection.Document.SHA256,
			expectedDigest,
		)
	}
	if inspection.Document.Format != request.Format {
		return SourceAdapterExecution{}, fmt.Errorf(
			"source adapter inspection format %q differs from requested format %q",
			inspection.Document.Format,
			request.Format,
		)
	}

	limits := append([]string(nil), inspection.AuthorityLimits...)
	limits = append(
		limits,
		"capability-resolution-is-not-canonical-acceptance",
		"authoring-tool-capability-is-not-application-runtime-authority",
		"source-adapter-execution-is-read-only-observation",
	)
	limits = uniqueSortedSourceAdapterStrings(limits)

	return SourceAdapterExecution{
		Resolution:           resolution,
		SourceArtifactDigest: "sha256:" + expectedDigest,
		Inspection:           inspection,
		AuthorityLimits:      limits,
	}, nil
}

func uniqueSortedSourceAdapterStrings(values []string) []string {
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
