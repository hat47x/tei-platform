package main

import (
	"context"
	"encoding/json"
	"fmt"

	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/realization"
)

const (
	contractQueryCapabilityContractID = "runtime.contract-query-executor"
	contractQueryRequestShape         = "tei.reference.contract-query-execution.request/v0"
	contractQueryResultShape          = "tei.reference.contract-query-execution.result/v0"
)

var (
	contractQueryRequestShapeFingerprint = invocationShapeFingerprint(
		contractQueryRequestShapeArtifact(),
	)
	contractQueryResultShapeFingerprint = invocationShapeFingerprint(
		contractQueryResultShapeArtifact(),
	)
)

type ContractQueryInvocationRequest struct {
	Target     realization.ContractQueryTarget `json:"target"`
	Contract   database.QueryContract          `json:"contract"`
	Parameters map[string]any                  `json:"parameters,omitempty"`
}

func ContractQueryInvocationImplementation(
	provider Provider,
	executor driver.ContractQueryExecutor,
) (InvocationImplementation, error) {
	if executor == nil {
		return InvocationImplementation{}, fmt.Errorf(
			"contract query invocation requires executor",
		)
	}
	if provider.Scope != ScopeApplicationRuntime ||
		provider.ContractID != contractQueryCapabilityContractID ||
		provider.PluginID == "" ||
		provider.PluginVersion == "" ||
		provider.ImplementationID == "" ||
		provider.ArtifactDigest == "" ||
		provider.ContractVersion == "" ||
		provider.ContractFingerprint == "" ||
		provider.ConformanceProfile == "" {
		return InvocationImplementation{}, fmt.Errorf(
			"provider %s@%s/%s is not a complete Runtime Contract Query provider",
			provider.PluginID,
			provider.PluginVersion,
			provider.ImplementationID,
		)
	}
	return InvocationImplementation{
		Provider:                provider,
		RequestShape:            contractQueryRequestShape,
		RequestShapeFingerprint: contractQueryRequestShapeFingerprint,
		RequestShapeArtifact:    contractQueryRequestShapeArtifact(),
		ResultShape:             contractQueryResultShape,
		ResultShapeFingerprint:  contractQueryResultShapeFingerprint,
		ResultShapeArtifact:     contractQueryResultShapeArtifact(),
		PayloadCodec:            referenceJSONPayloadCodec,
		AuthorityLimits: []string{
			"application-runtime-capability-does-not-grant-authoring-authority",
			"contract-query-invocation-does-not-bypass-database-provider-binding",
			"contract-query-invocation-does-not-create-environment-authority",
		},
		Invoke: func(ctx context.Context, payload []byte) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			var request ContractQueryInvocationRequest
			if err := decodeReferenceJSONExact(payload, &request); err != nil {
				return nil, fmt.Errorf("decode contract query request: %w", err)
			}
			if !request.Target.Valid() {
				return nil, fmt.Errorf("contract query invocation target is incomplete")
			}
			if request.Contract.Name == "" {
				return nil, fmt.Errorf("contract query invocation contract has no query name")
			}
			result, err := executor.ExecuteContractQuery(
				ctx,
				request.Target,
				request.Contract,
				request.Parameters,
			)
			if err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				return nil, fmt.Errorf("encode contract query result: %w", err)
			}
			return encoded, nil
		},
	}, nil
}
