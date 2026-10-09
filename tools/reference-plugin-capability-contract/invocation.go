package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	referenceJSONPayloadCodec = "application/json;profile=tei-reference-v0"
	invocationEffectReadOnly  = "read-only"
)

type InvocationShapeAuthorityEvidence struct {
	Issuer              InvocationShapeIssuer `json:"issuer"`
	ShapeID             string                `json:"shapeId"`
	ShapeFingerprint    string                `json:"shapeFingerprint"`
	EvidenceFingerprint string                `json:"evidenceFingerprint"`
}

type InvocationImplementation struct {
	Provider                Provider
	RequestShape            string
	RequestShapeFingerprint string
	RequestShapeArtifact    InvocationShapeArtifact
	ResultShape             string
	ResultShapeFingerprint  string
	ResultShapeArtifact     InvocationShapeArtifact
	PayloadCodec            string
	AuthorityLimits         []string
	Invoke                  func(context.Context, []byte) ([]byte, error)
}

type InvocationExecution struct {
	Resolution                              Resolution `json:"resolution"`
	RequestShape                            string     `json:"requestShape"`
	RequestShapeFingerprint                 string     `json:"requestShapeFingerprint"`
	RequestShapeAuthorityEvidenceFingerprint string    `json:"requestShapeAuthorityEvidenceFingerprint"`
	ResultShape                             string     `json:"resultShape"`
	ResultShapeFingerprint                  string     `json:"resultShapeFingerprint"`
	ResultShapeAuthorityEvidenceFingerprint string     `json:"resultShapeAuthorityEvidenceFingerprint"`
	PayloadCodec                            string     `json:"payloadCodec"`
	RequestDigest     string     `json:"requestDigest"`
	ResultDigest      string     `json:"resultDigest"`
	ResultPayload     []byte     `json:"resultPayload"`
	AuthorityLimits   []string   `json:"authorityLimits"`
}

type InvocationRegistry struct {
	implementations  map[string]InvocationImplementation
	shapeAuthorities map[string]InvocationShapeAuthorityEvidence
}

func NewInvocationRegistry(
	implementations []InvocationImplementation,
	authorities []InvocationShapeAuthorityEvidence,
) (*InvocationRegistry, error) {
	shapeAuthorities, err := normalizeInvocationShapeAuthorities(authorities)
	if err != nil {
		return nil, err
	}
	registry := &InvocationRegistry{
		implementations:  map[string]InvocationImplementation{},
		shapeAuthorities: shapeAuthorities,
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
			!validInvocationShapeFingerprint(implementation.RequestShapeFingerprint) ||
			implementation.ResultShape == "" ||
			!validInvocationShapeFingerprint(implementation.ResultShapeFingerprint) ||
			implementation.PayloadCodec == "" {
			return nil, fmt.Errorf(
				"capability implementation[%d] has incomplete invocation shape",
				i,
			)
		}
		if err := validateInvocationShapeArtifact(
			implementation.RequestShapeArtifact,
		); err != nil {
			return nil, fmt.Errorf(
				"capability implementation[%d] request shape artifact: %w",
				i,
				err,
			)
		}
		if err := validateInvocationShapeArtifact(
			implementation.ResultShapeArtifact,
		); err != nil {
			return nil, fmt.Errorf(
				"capability implementation[%d] result shape artifact: %w",
				i,
				err,
			)
		}
		if implementation.RequestShapeArtifact.ID != implementation.RequestShape ||
			invocationShapeFingerprint(implementation.RequestShapeArtifact) !=
				implementation.RequestShapeFingerprint ||
			implementation.RequestShapeArtifact.PayloadCodec !=
				implementation.PayloadCodec {
			return nil, fmt.Errorf(
				"capability implementation[%d] request shape artifact does not match declaration",
				i,
			)
		}
		if implementation.ResultShapeArtifact.ID != implementation.ResultShape ||
			invocationShapeFingerprint(implementation.ResultShapeArtifact) !=
				implementation.ResultShapeFingerprint ||
			implementation.ResultShapeArtifact.PayloadCodec !=
				implementation.PayloadCodec {
			return nil, fmt.Errorf(
				"capability implementation[%d] result shape artifact does not match declaration",
				i,
			)
		}
		if _, ok := shapeAuthorities[
			invocationShapeAuthorityKey(
				implementation.RequestShapeArtifact.Issuer,
				implementation.RequestShape,
				implementation.RequestShapeFingerprint,
			),
		]; !ok {
			return nil, fmt.Errorf(
				"capability implementation[%d] request shape %s has no accepted Shape Authority Evidence",
				i,
				implementation.RequestShape,
			)
		}
		if _, ok := shapeAuthorities[
			invocationShapeAuthorityKey(
				implementation.ResultShapeArtifact.Issuer,
				implementation.ResultShape,
				implementation.ResultShapeFingerprint,
			),
		]; !ok {
			return nil, fmt.Errorf(
				"capability implementation[%d] result shape %s has no accepted Shape Authority Evidence",
				i,
				implementation.ResultShape,
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
		!validInvocationShapeFingerprint(contract.RequestShapeFingerprint) ||
		contract.ResultShape == "" ||
		!validInvocationShapeFingerprint(contract.ResultShapeFingerprint) ||
		contract.PayloadCodec == "" {
		return InvocationExecution{}, fmt.Errorf(
			"capability contract %s@%s has no machine-readable invocation shape",
			contract.ID,
			contract.Version,
		)
	}
	if contract.ShapeIssuer == nil ||
		contract.ShapeIssuer.ID == "" ||
		contract.ShapeIssuer.Version == "" {
		return InvocationExecution{}, fmt.Errorf(
			"capability contract %s@%s has no invocation shape issuer",
			contract.ID,
			contract.Version,
		)
	}
	if contract.InvocationEffect != invocationEffectReadOnly {
		return InvocationExecution{}, fmt.Errorf(
			"capability contract %s@%s invocation effect %q is not supported by the Reference generic boundary",
			contract.ID,
			contract.Version,
			contract.InvocationEffect,
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
		implementation.RequestShapeFingerprint != contract.RequestShapeFingerprint ||
		implementation.ResultShape != contract.ResultShape ||
		implementation.ResultShapeFingerprint != contract.ResultShapeFingerprint ||
		implementation.PayloadCodec != contract.PayloadCodec {
		return InvocationExecution{}, fmt.Errorf(
			"capability implementation invocation shape does not match contract %s@%s",
			contract.ID,
			contract.Version,
		)
	}
	if implementation.RequestShapeArtifact.Issuer != *contract.ShapeIssuer ||
		implementation.ResultShapeArtifact.Issuer != *contract.ShapeIssuer {
		return InvocationExecution{}, fmt.Errorf(
			"capability invocation shape issuer does not match contract %s@%s",
			contract.ID,
			contract.Version,
		)
	}
	requestShapeAuthority, err := r.shapeAuthorityFor(
		implementation.RequestShapeArtifact,
	)
	if err != nil {
		return InvocationExecution{}, err
	}
	resultShapeAuthority, err := r.shapeAuthorityFor(
		implementation.ResultShapeArtifact,
	)
	if err != nil {
		return InvocationExecution{}, err
	}
	if err := validateInvocationPayload(
		implementation.RequestShapeArtifact,
		requestPayload,
	); err != nil {
		return InvocationExecution{}, fmt.Errorf(
			"capability request does not conform to shape %s: %w",
			contract.RequestShape,
			err,
		)
	}

	resultPayload, err := implementation.Invoke(ctx, append([]byte(nil), requestPayload...))
	if err != nil {
		return InvocationExecution{}, err
	}
	if err := validateInvocationPayload(
		implementation.ResultShapeArtifact,
		resultPayload,
	); err != nil {
		return InvocationExecution{}, fmt.Errorf(
			"capability result does not conform to shape %s: %w",
			contract.ResultShape,
			err,
		)
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
		Resolution:                               resolution,
		RequestShape:                             contract.RequestShape,
		RequestShapeFingerprint:                  contract.RequestShapeFingerprint,
		RequestShapeAuthorityEvidenceFingerprint: requestShapeAuthority.EvidenceFingerprint,
		ResultShape:                              contract.ResultShape,
		ResultShapeFingerprint:                   contract.ResultShapeFingerprint,
		ResultShapeAuthorityEvidenceFingerprint:  resultShapeAuthority.EvidenceFingerprint,
		PayloadCodec:                             contract.PayloadCodec,
		RequestDigest:   "sha256:" + hex.EncodeToString(requestSum[:]),
		ResultDigest:    "sha256:" + hex.EncodeToString(resultSum[:]),
		ResultPayload:   append([]byte(nil), resultPayload...),
		AuthorityLimits: limits,
	}, nil
}

func normalizeInvocationShapeAuthorities(
	values []InvocationShapeAuthorityEvidence,
) (map[string]InvocationShapeAuthorityEvidence, error) {
	out := map[string]InvocationShapeAuthorityEvidence{}
	for i, value := range values {
		value.Issuer.ID = strings.TrimSpace(value.Issuer.ID)
		value.Issuer.Version = strings.TrimSpace(value.Issuer.Version)
		value.ShapeID = strings.TrimSpace(value.ShapeID)
		value.ShapeFingerprint = strings.TrimSpace(value.ShapeFingerprint)
		value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)
		if value.Issuer.ID == "" || value.Issuer.Version == "" ||
			value.ShapeID == "" ||
			!validInvocationShapeFingerprint(value.ShapeFingerprint) ||
			!validInvocationShapeFingerprint(value.EvidenceFingerprint) {
			return nil, fmt.Errorf(
				"shape authority evidence[%d] is incomplete or invalid",
				i,
			)
		}
		if _, err := parseVersion(value.Issuer.Version); err != nil {
			return nil, fmt.Errorf(
				"shape authority evidence[%d] issuer version: %w",
				i,
				err,
			)
		}
		key := invocationShapeAuthorityKey(
			value.Issuer,
			value.ShapeID,
			value.ShapeFingerprint,
		)
		if existing, duplicate := out[key]; duplicate {
			return nil, fmt.Errorf(
				"shape %s authority is ambiguous across accepted Evidence %s and %s",
				value.ShapeID,
				existing.EvidenceFingerprint,
				value.EvidenceFingerprint,
			)
		}
		out[key] = value
	}
	return out, nil
}

func invocationShapeAuthorityKey(
	issuer InvocationShapeIssuer,
	shapeID string,
	shapeFingerprint string,
) string {
	return strings.Join(
		[]string{
			issuer.ID,
			issuer.Version,
			shapeID,
			shapeFingerprint,
		},
		"\x00",
	)
}

func (r *InvocationRegistry) shapeAuthorityFor(
	artifact InvocationShapeArtifact,
) (InvocationShapeAuthorityEvidence, error) {
	if r == nil {
		return InvocationShapeAuthorityEvidence{}, fmt.Errorf(
			"capability invocation registry is nil",
		)
	}
	fingerprint := invocationShapeFingerprint(artifact)
	key := invocationShapeAuthorityKey(
		artifact.Issuer,
		artifact.ID,
		fingerprint,
	)
	evidence, ok := r.shapeAuthorities[key]
	if !ok {
		return InvocationShapeAuthorityEvidence{}, fmt.Errorf(
			"invocation shape %s has no accepted Shape Authority Evidence",
			artifact.ID,
		)
	}
	return evidence, nil
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

func validInvocationShapeFingerprint(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, r := range strings.TrimPrefix(value, "sha256:") {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func decodeReferenceJSONExact(payload []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value is not allowed")
		}
		return fmt.Errorf("decode trailing JSON value: %w", err)
	}
	return nil
}
