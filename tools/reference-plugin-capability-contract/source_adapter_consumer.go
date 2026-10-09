package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/xeadsource"
)

const (
	sourceAdapterCapabilityContractID   = "authoring.source-adapter.xead"
	referenceXeadReaderImplementationID = "xead.reader.v0"
	sourceAdapterRequestShape           = "tei.reference.source-adapter.request/v0"
	sourceAdapterResultShape            = "tei.reference.source-adapter.inspection/v0"
)

var (
	sourceAdapterRequestShapeFingerprint = invocationShapeFingerprint(
		sourceAdapterRequestShapeArtifact(),
	)
	sourceAdapterResultShapeFingerprint = invocationShapeFingerprint(
		sourceAdapterResultShapeArtifact(),
	)
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
	invocations *InvocationRegistry
}

func NewSourceAdapterRegistry(
	implementations []SourceAdapterImplementation,
	shapeAuthorities []InvocationShapeAuthorityEvidence,
) (*SourceAdapterRegistry, error) {
	generic := make([]InvocationImplementation, 0, len(implementations))
	for i, implementation := range implementations {
		invocation, err := sourceAdapterInvocationImplementation(implementation)
		if err != nil {
			return nil, fmt.Errorf("source adapter implementation[%d]: %w", i, err)
		}
		generic = append(generic, invocation)
	}
	registry, err := NewInvocationRegistry(generic, shapeAuthorities)
	if err != nil {
		return nil, err
	}
	return &SourceAdapterRegistry{invocations: registry}, nil
}

func sourceAdapterInvocationImplementation(
	implementation SourceAdapterImplementation,
) (InvocationImplementation, error) {
	provider := implementation.Provider
	if implementation.Invoke == nil {
		return InvocationImplementation{}, fmt.Errorf("has no invoker")
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
		return InvocationImplementation{}, fmt.Errorf("has invalid provider declaration")
	}
	return InvocationImplementation{
		Provider:                provider,
		RequestShape:            sourceAdapterRequestShape,
		RequestShapeFingerprint: sourceAdapterRequestShapeFingerprint,
		RequestShapeArtifact:    sourceAdapterRequestShapeArtifact(),
		ResultShape:             sourceAdapterResultShape,
		ResultShapeFingerprint:  sourceAdapterResultShapeFingerprint,
		ResultShapeArtifact:     sourceAdapterResultShapeArtifact(),
		PayloadCodec:            referenceJSONPayloadCodec,
		AuthorityLimits: []string{
			"authoring-tool-capability-is-not-application-runtime-authority",
			"source-adapter-execution-is-read-only-observation",
			"source-adapter-inspection-does-not-replace-rmt-publication-receipt",
			"source-adapter-inspection-is-local-private-evidence",
		},
		Invoke: func(ctx context.Context, payload []byte) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			var request SourceAdapterRequest
			if err := decodeReferenceJSONExact(payload, &request); err != nil {
				return nil, fmt.Errorf("decode source adapter request: %w", err)
			}
			inspection, err := implementation.Invoke(request)
			if err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(inspection)
			if err != nil {
				return nil, fmt.Errorf("encode source adapter result: %w", err)
			}
			return encoded, nil
		},
	}, nil
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
	if r == nil || r.invocations == nil {
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

	payload, err := json.Marshal(request)
	if err != nil {
		return SourceAdapterExecution{}, fmt.Errorf("encode source adapter request: %w", err)
	}
	execution, err := r.invocations.ResolveAndInvoke(
		context.Background(),
		input,
		payload,
	)
	if err != nil {
		return SourceAdapterExecution{}, err
	}
	var inspection xeadsource.Inspection
	if err := decodeReferenceJSONExact(execution.ResultPayload, &inspection); err != nil {
		return SourceAdapterExecution{}, fmt.Errorf("decode source adapter result: %w", err)
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
	limits = append(limits, execution.AuthorityLimits...)
	limits = uniqueSortedSourceAdapterStrings(limits)

	return SourceAdapterExecution{
		Resolution:           execution.Resolution,
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
