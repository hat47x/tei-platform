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
	dispatchSchema      = "tei.reference.capability-transport-dispatch/v0"
	resultSchema        = "tei.reference.capability-transport-result/v0"
	receiptSchema       = "tei.reference.capability-transport-receipt/v0"
	dispatchPlanSchema  = "tei.reference.capability-transport-dispatch-plan/v0"
)

type GenericProviderIdentity struct {
	PluginID         string `json:"pluginId"`
	PluginVersion    string `json:"pluginVersion"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest  string `json:"artifactDigest"`
}

type GenericInvocationIdentity struct {
	ContractID              string                  `json:"contractId"`
	ContractFingerprint     string                  `json:"contractFingerprint"`
	Target                  string                  `json:"target"`
	Provider                GenericProviderIdentity `json:"provider"`
	RequestShapeFingerprint string                  `json:"requestShapeFingerprint"`
	ResultShapeFingerprint  string                  `json:"resultShapeFingerprint"`
	PayloadCodec             string                  `json:"payloadCodec"`
}

type TransportDispatchPlan struct {
	Schema             string                    `json:"schema"`
	InvocationIdentity GenericInvocationIdentity `json:"invocationIdentity"`
	PlanFingerprint    string                    `json:"planFingerprint"`
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

type TransportCarrierReceipt struct {
	Kind                 string                  `json:"kind"`
	Plugin               *plugin.Identity        `json:"plugin,omitempty"`
	LegacyCapabilityKind plugin.CapabilityKind   `json:"legacyCapabilityKind,omitempty"`
	LegacyCapabilityID   core.CanonicalID        `json:"legacyCapabilityId,omitempty"`
}

type TransportReceipt struct {
	Schema                  string                    `json:"schema"`
	Transport               string                    `json:"transport"`
	DispatchPlanFingerprint string                    `json:"dispatchPlanFingerprint,omitempty"`
	Carrier            TransportCarrierReceipt   `json:"carrier"`
	RequestID          string                    `json:"requestId"`
	InvocationIdentity GenericInvocationIdentity `json:"invocationIdentity"`
	RequestDigest       string                    `json:"requestDigest"`
	ResultDigest        string                    `json:"resultDigest"`
	AuthorityLimits     []string                  `json:"authorityLimits"`
}

type TransportExecution struct {
	ResultPayload []byte           `json:"resultPayload"`
	Receipt       TransportReceipt `json:"receipt"`
}

func buildTransportDispatchPlan(
	identity GenericInvocationIdentity,
) (TransportDispatchPlan, error) {
	if err := validateGenericInvocationIdentity(identity); err != nil {
		return TransportDispatchPlan{}, err
	}
	plan := TransportDispatchPlan{
		Schema:             dispatchPlanSchema,
		InvocationIdentity: identity,
	}
	plan.PlanFingerprint = transportDispatchPlanFingerprint(plan)
	return plan, nil
}

func validateTransportDispatchPlan(plan TransportDispatchPlan) error {
	if plan.Schema != dispatchPlanSchema {
		return fmt.Errorf("transport dispatch plan schema %q is not supported", plan.Schema)
	}
	if err := validateGenericInvocationIdentity(plan.InvocationIdentity); err != nil {
		return fmt.Errorf("transport dispatch plan identity: %w", err)
	}
	if !validSHA256(plan.PlanFingerprint) {
		return fmt.Errorf("transport dispatch plan fingerprint is invalid")
	}
	if transportDispatchPlanFingerprint(plan) != plan.PlanFingerprint {
		return fmt.Errorf("transport dispatch plan fingerprint is stale")
	}
	return nil
}

func transportDispatchPlanFingerprint(plan TransportDispatchPlan) string {
	plan.PlanFingerprint = ""
	payload, _ := json.Marshal(plan)
	return digestBytes(payload)
}

func validateDispatchAgainstPlan(
	plan TransportDispatchPlan,
	dispatch GenericDispatch,
) error {
	if err := validateTransportDispatchPlan(plan); err != nil {
		return err
	}
	if err := validateGenericDispatch(dispatch); err != nil {
		return err
	}
	if genericDispatchIdentity(dispatch) != plan.InvocationIdentity {
		return fmt.Errorf("generic dispatch identity differs from transport dispatch plan")
	}
	return nil
}

func bindExecutionToTransportPlan(
	execution TransportExecution,
	plan TransportDispatchPlan,
) TransportExecution {
	execution.Receipt.DispatchPlanFingerprint = plan.PlanFingerprint
	execution.Receipt.AuthorityLimits = append(
		execution.Receipt.AuthorityLimits,
		"dispatch-plan-is-not-capability-conformance",
		"dispatch-plan-is-not-runtime-authority",
		"dispatch-plan-is-not-canonical-acceptance",
	)
	return execution
}

func invokePlannedOverProcessTransport(
	ctx context.Context,
	launcher pruntime.ProcessLauncher,
	command pruntime.ProcessCommand,
	plan TransportDispatchPlan,
	carrier LegacyCarrier,
	requestID string,
	dispatch GenericDispatch,
) (TransportExecution, error) {
	if err := validateDispatchAgainstPlan(plan, dispatch); err != nil {
		return TransportExecution{}, err
	}
	execution, err := invokeOverProcessTransport(
		ctx,
		launcher,
		command,
		carrier,
		requestID,
		dispatch,
	)
	if err != nil {
		return TransportExecution{}, err
	}
	return bindExecutionToTransportPlan(execution, plan), nil
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
	if err := validateCarrierProviderBinding(carrier, dispatch.Provider); err != nil {
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
	if err := validateGenericTransportResult(dispatch, result); err != nil {
		return TransportExecution{}, err
	}

	return TransportExecution{
		ResultPayload: append([]byte(nil), result.Payload...),
		Receipt: newTransportReceipt(
			"process-stdio-v1",
			TransportCarrierReceipt{
				Kind:                 "tei-runtime-legacy-capability",
				Plugin:               &carrier.Plugin,
				LegacyCapabilityKind: carrier.CapabilityKind,
				LegacyCapabilityID:   carrier.CapabilityID,
			},
			requestID,
			dispatch,
			result,
			[]string{
				"transport-success-is-not-capability-conformance",
				"transport-success-is-not-runtime-authority",
				"transport-success-is-not-canonical-acceptance",
				"legacy-carrier-identity-is-not-generic-contract-identity",
				"carrier-provider-package-match-is-not-runtime-authority",
				"generic-provider-and-shape-echo-is-not-runtime-authority",
				"process-response-does-not-echo-legacy-capability-kind",
				"process-stdio-is-one-realization-not-universal-capability-abi",
			},
		),
	}, nil
}

func genericDispatchIdentity(dispatch GenericDispatch) GenericInvocationIdentity {
	return GenericInvocationIdentity{
		ContractID:              dispatch.ContractID,
		ContractFingerprint:     dispatch.ContractFingerprint,
		Target:                  dispatch.Target,
		Provider:                dispatch.Provider,
		RequestShapeFingerprint: dispatch.RequestShapeFingerprint,
		ResultShapeFingerprint:  dispatch.ResultShapeFingerprint,
		PayloadCodec:             dispatch.PayloadCodec,
	}
}

func genericResultIdentity(result GenericTransportResult) GenericInvocationIdentity {
	return GenericInvocationIdentity{
		ContractID:              result.ContractID,
		ContractFingerprint:     result.ContractFingerprint,
		Target:                  result.Target,
		Provider:                result.Provider,
		RequestShapeFingerprint: result.RequestShapeFingerprint,
		ResultShapeFingerprint:  result.ResultShapeFingerprint,
		PayloadCodec:             result.PayloadCodec,
	}
}

func validateGenericTransportResult(
	dispatch GenericDispatch,
	result GenericTransportResult,
) error {
	if result.Schema != resultSchema ||
		genericResultIdentity(result) != genericDispatchIdentity(dispatch) {
		return fmt.Errorf("generic transport result identity drifted")
	}
	return nil
}

func newTransportReceipt(
	transport string,
	carrier TransportCarrierReceipt,
	requestID string,
	dispatch GenericDispatch,
	result GenericTransportResult,
	authorityLimits []string,
) TransportReceipt {
	return TransportReceipt{
		Schema:             receiptSchema,
		Transport:          transport,
		Carrier:            carrier,
		RequestID:          requestID,
		InvocationIdentity: genericDispatchIdentity(dispatch),
		RequestDigest:       digestBytes(dispatch.Payload),
		ResultDigest:        digestBytes(result.Payload),
		AuthorityLimits:     append([]string(nil), authorityLimits...),
	}
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

func validateCarrierProviderBinding(
	carrier LegacyCarrier,
	provider GenericProviderIdentity,
) error {
	if string(carrier.Plugin.ID) != provider.PluginID ||
		carrier.Plugin.Version != provider.PluginVersion {
		return fmt.Errorf(
			"generic provider %s@%s does not match legacy carrier plugin %s@%s",
			provider.PluginID,
			provider.PluginVersion,
			carrier.Plugin.ID,
			carrier.Plugin.Version,
		)
	}
	return nil
}

func validateGenericInvocationIdentity(identity GenericInvocationIdentity) error {
	if strings.TrimSpace(identity.ContractID) == "" ||
		!validSHA256(identity.ContractFingerprint) ||
		strings.TrimSpace(identity.Target) == "" {
		return fmt.Errorf("generic dispatch contract identity is incomplete")
	}
	if identity.Provider.PluginID == "" ||
		identity.Provider.PluginVersion == "" ||
		identity.Provider.ImplementationID == "" ||
		!validSHA256(identity.Provider.ArtifactDigest) {
		return fmt.Errorf("generic dispatch provider identity is incomplete")
	}
	if !validSHA256(identity.RequestShapeFingerprint) ||
		!validSHA256(identity.ResultShapeFingerprint) ||
		strings.TrimSpace(identity.PayloadCodec) == "" {
		return fmt.Errorf("generic dispatch shape identity is incomplete")
	}
	return nil
}

func validateGenericDispatch(dispatch GenericDispatch) error {
	if dispatch.Schema != dispatchSchema {
		return fmt.Errorf("generic dispatch schema %q is not supported", dispatch.Schema)
	}
	return validateGenericInvocationIdentity(genericDispatchIdentity(dispatch))
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
