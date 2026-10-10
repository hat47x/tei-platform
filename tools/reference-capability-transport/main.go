package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const (
	dispatchSchema = "tei.reference.capability-transport-dispatch/v0"
	receiptSchema  = "tei.reference.capability-transport-receipt/v0"
)

type GenericProviderIdentity struct {
	PluginID         string `json:"pluginId"`
	PluginVersion    string `json:"pluginVersion"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest  string `json:"artifactDigest"`
}

type GenericDispatch struct {
	Schema                  string                  `json:"schema"`
	ContractID              string                  `json:"contractId"`
	ContractFingerprint     string                  `json:"contractFingerprint"`
	Target                  string                  `json:"target"`
	Provider                GenericProviderIdentity `json:"provider"`
	RequestShapeFingerprint string                  `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string                  `json:"resultShapeFingerprint"`
	PayloadCodec             string                  `json:"payloadCodec"`
	Payload                  []byte                  `json:"payload"`
}

type GenericTransportResult struct {
	Schema                  string                  `json:"schema"`
	ContractID              string                  `json:"contractId"`
	ContractFingerprint     string                  `json:"contractFingerprint"`
	Target                  string                  `json:"target"`
	Provider                GenericProviderIdentity `json:"provider"`
	RequestShapeFingerprint string                  `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string                  `json:"resultShapeFingerprint"`
	PayloadCodec             string                  `json:"payloadCodec"`
	Payload                  []byte                  `json:"payload"`
}

type LegacyCarrier struct {
	Plugin                plugin.Identity       `json:"plugin"`
	CapabilityKind        plugin.CapabilityKind `json:"capabilityKind"`
	CapabilityID          core.CanonicalID       `json:"capabilityId"`
	DescriptorFingerprint string                 `json:"descriptorFingerprint"`
	Operation             string                 `json:"operation"`
}

type TransportReceipt struct {
	Schema                  string                  `json:"schema"`
	Transport               string                  `json:"transport"`
	RequestID               string                  `json:"requestId"`
	GenericContractID       string                  `json:"genericContractId"`
	ContractFingerprint     string                  `json:"contractFingerprint"`
	Target                  string                  `json:"target"`
	Provider                GenericProviderIdentity `json:"provider"`
	RequestShapeFingerprint string                  `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string                  `json:"resultShapeFingerprint"`
	PayloadCodec             string                  `json:"payloadCodec"`
	RequestDigest            string                  `json:"requestDigest"`
	ResultDigest             string                  `json:"resultDigest"`
	LegacyCapabilityKind     plugin.CapabilityKind   `json:"legacyCapabilityKind"`
	LegacyCapabilityID       core.CanonicalID        `json:"legacyCapabilityId"`
	AuthorityLimits          []string                `json:"authorityLimits"`
}

type TransportExecution struct {
	ResultPayload []byte           `json:"resultPayload"`
	Receipt       TransportReceipt `json:"receipt"`
}

func invokeOverProcessTransport(
	ctx context.Context,
	launcher pruntime.ProcessLauncher,
	command pruntime.ProcessCommand,
	carrier LegacyCarrier,
	requestID string,
	dispatch GenericDispatch,
) (TransportExecution, error) {
	if ctx == nil {
		return TransportExecution{}, fmt.Errorf("capability transport requires context")
	}
	if launcher == nil {
		return TransportExecution{}, fmt.Errorf("capability transport requires process launcher")
	}
	if err := validateLegacyCarrier(carrier); err != nil {
		return TransportExecution{}, err
	}
	if strings.TrimSpace(requestID) == "" {
		return TransportExecution{}, fmt.Errorf("capability transport requires request id")
	}
	if err := validateGenericDispatch(dispatch); err != nil {
		return TransportExecution{}, err
	}

	payload, err := json.Marshal(dispatch)
	if err != nil {
		return TransportExecution{}, fmt.Errorf("encode generic transport dispatch: %w", err)
	}
	req := pruntime.Request{
		ProtocolVersion:       pruntime.ProtocolVersion,
		RequestID:             requestID,
		Plugin:                carrier.Plugin,
		CapabilityKind:        carrier.CapabilityKind,
		CapabilityID:          carrier.CapabilityID,
		DescriptorFingerprint: carrier.DescriptorFingerprint,
		Operation:             carrier.Operation,
		Payload:               payload,
	}
	resp, err := launcher.Invoke(ctx, command, req)
	if err != nil {
		return TransportExecution{}, err
	}
	if err := validateProcessResponseIdentity(req, resp); err != nil {
		return TransportExecution{}, err
	}

	var result GenericTransportResult
	if err := decodeJSONExact(resp.Payload, &result); err != nil {
		return TransportExecution{}, fmt.Errorf("decode generic transport result: %w", err)
	}
	if result.Schema != dispatchSchema ||
		result.ContractID != dispatch.ContractID ||
		result.ContractFingerprint != dispatch.ContractFingerprint ||
		result.Target != dispatch.Target ||
		result.Provider != dispatch.Provider ||
		result.RequestShapeFingerprint != dispatch.RequestShapeFingerprint ||
		result.ResultShapeFingerprint != dispatch.ResultShapeFingerprint ||
		result.PayloadCodec != dispatch.PayloadCodec {
		return TransportExecution{}, fmt.Errorf("generic transport result identity drifted")
	}

	return TransportExecution{
		ResultPayload: append([]byte(nil), result.Payload...),
		Receipt: TransportReceipt{
			Schema:                  receiptSchema,
			Transport:               "process-stdio-v1",
			RequestID:               requestID,
			GenericContractID:       dispatch.ContractID,
			ContractFingerprint:     dispatch.ContractFingerprint,
			Target:                  dispatch.Target,
			Provider:                dispatch.Provider,
			RequestShapeFingerprint: dispatch.RequestShapeFingerprint,
			ResultShapeFingerprint:  dispatch.ResultShapeFingerprint,
			PayloadCodec:             dispatch.PayloadCodec,
			RequestDigest:            digestBytes(dispatch.Payload),
			ResultDigest:             digestBytes(result.Payload),
			LegacyCapabilityKind:     carrier.CapabilityKind,
			LegacyCapabilityID:       carrier.CapabilityID,
			AuthorityLimits: []string{
				"transport-success-is-not-capability-conformance",
				"transport-success-is-not-runtime-authority",
				"transport-success-is-not-canonical-acceptance",
				"legacy-carrier-identity-is-not-generic-contract-identity",
				"generic-provider-and-shape-echo-is-not-runtime-authority",
				"process-response-does-not-echo-legacy-capability-kind",
				"process-stdio-is-one-realization-not-universal-capability-abi",
			},
		},
	}, nil
}

func validateLegacyCarrier(carrier LegacyCarrier) error {
	if carrier.Plugin.ID == "" || strings.TrimSpace(carrier.Plugin.Version) == "" {
		return fmt.Errorf("legacy transport carrier requires plugin identity")
	}
	if carrier.CapabilityKind == "" || carrier.CapabilityID == "" {
		return fmt.Errorf("legacy transport carrier requires CapabilityKind and CapabilityID")
	}
	if strings.TrimSpace(carrier.DescriptorFingerprint) == "" {
		return fmt.Errorf("legacy transport carrier requires descriptor fingerprint")
	}
	if strings.TrimSpace(carrier.Operation) == "" {
		return fmt.Errorf("legacy transport carrier requires operation")
	}
	return nil
}

func validateGenericDispatch(dispatch GenericDispatch) error {
	if dispatch.Schema != dispatchSchema {
		return fmt.Errorf("generic dispatch schema %q is not supported", dispatch.Schema)
	}
	if strings.TrimSpace(dispatch.ContractID) == "" ||
		!validSHA256(dispatch.ContractFingerprint) ||
		strings.TrimSpace(dispatch.Target) == "" {
		return fmt.Errorf("generic dispatch contract identity is incomplete")
	}
	if dispatch.Provider.PluginID == "" ||
		dispatch.Provider.PluginVersion == "" ||
		dispatch.Provider.ImplementationID == "" ||
		!validSHA256(dispatch.Provider.ArtifactDigest) {
		return fmt.Errorf("generic dispatch provider identity is incomplete")
	}
	if !validSHA256(dispatch.RequestShapeFingerprint) ||
		!validSHA256(dispatch.ResultShapeFingerprint) ||
		strings.TrimSpace(dispatch.PayloadCodec) == "" {
		return fmt.Errorf("generic dispatch shape identity is incomplete")
	}
	return nil
}

func validateProcessResponseIdentity(
	req pruntime.Request,
	resp pruntime.Response,
) error {
	if resp.ProtocolVersion != pruntime.ProtocolVersion {
		return fmt.Errorf("process response protocol version %d does not match %d", resp.ProtocolVersion, pruntime.ProtocolVersion)
	}
	if resp.RequestID != req.RequestID {
		return fmt.Errorf("process response request id %q does not match %q", resp.RequestID, req.RequestID)
	}
	if resp.Plugin != req.Plugin {
		return fmt.Errorf("process response plugin identity does not match request")
	}
	if resp.CapabilityID != req.CapabilityID {
		return fmt.Errorf("process response CapabilityID does not match request")
	}
	if resp.DescriptorFingerprint != req.DescriptorFingerprint {
		return fmt.Errorf("process response descriptor fingerprint does not match request")
	}
	if resp.Status != pruntime.ResponseOK {
		if resp.Status == pruntime.ResponseError {
			return fmt.Errorf("process transport returned error status")
		}
		return fmt.Errorf("process transport returned invalid status %q", resp.Status)
	}
	return nil
}

func decodeJSONExact(payload []byte, out any) error {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
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

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
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
