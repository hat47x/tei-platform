package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/realization"
	"tei/internal/xeadsource"
)

type recordingContractQueryExecutor struct {
	calls      int
	target     realization.ContractQueryTarget
	contract   database.QueryContract
	parameters map[string]any
	result     driver.ContractQueryResult
	err        error
}

func (e *recordingContractQueryExecutor) ExecuteContractQuery(
	ctx context.Context,
	target realization.ContractQueryTarget,
	contract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	if err := ctx.Err(); err != nil {
		return driver.ContractQueryResult{}, err
	}
	e.calls++
	e.target = target
	e.contract = contract
	e.parameters = parameters
	if e.err != nil {
		return driver.ContractQueryResult{}, e.err
	}
	return e.result, nil
}

func genericInvocationRegistryFixture(
	t *testing.T,
) (
	*InvocationRegistry,
	Contract,
	Provider,
	Contract,
	Provider,
	*recordingContractQueryExecutor,
) {
	t.Helper()

	sourceContract := sourceAdapterContract()
	sourceProvider := sourceAdapterProvider(
		sourceContract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	sourceTyped, err := ReferenceXeadSourceAdapterImplementation(sourceProvider)
	if err != nil {
		t.Fatal(err)
	}
	sourceInvocation, err := sourceAdapterInvocationImplementation(sourceTyped)
	if err != nil {
		t.Fatal(err)
	}

	queryContract := runtimeContract()
	queryProvider := sourceAdapterProvider(
		queryContract,
		ScopeApplicationRuntime,
		"reference.runtime.contract-query",
		"contract-query.executor.v1",
	)
	queryExecutor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{{
				Attributes: map[core.CanonicalID]any{
					core.MustCanonicalID("customer.id"): "C-001",
				},
			}},
		},
	}
	queryInvocation, err := ContractQueryInvocationImplementation(
		queryProvider,
		queryExecutor,
	)
	if err != nil {
		t.Fatal(err)
	}

	registry, err := NewInvocationRegistry([]InvocationImplementation{
		sourceInvocation,
		queryInvocation,
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry, sourceContract, sourceProvider, queryContract, queryProvider, queryExecutor
}

func TestGenericInvocationRegistryRunsIndependentAuthoringAndRuntimeCapabilities(t *testing.T) {
	registry, sourceContract, sourceProvider, queryContract, queryProvider, queryExecutor :=
		genericInvocationRegistryFixture(t)

	sourceRequest, err := json.Marshal(SourceAdapterRequest{
		Format: xeadsource.FormatXead,
		SourceBytes: []byte(
			`<System FormatVersion="1.2"><Table ID="T1"/></System>`,
		),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	sourceExecution, err := registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(sourceContract, sourceProvider),
		sourceRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if sourceExecution.Resolution.Provider != sourceProvider ||
		sourceExecution.RequestShape != sourceAdapterRequestShape ||
		sourceExecution.RequestShapeFingerprint != sourceAdapterRequestShapeFingerprint ||
		sourceExecution.ResultShape != sourceAdapterResultShape ||
		sourceExecution.ResultShapeFingerprint != sourceAdapterResultShapeFingerprint ||
		sourceExecution.PayloadCodec != referenceJSONPayloadCodec {
		t.Fatalf("source invocation execution=%+v", sourceExecution)
	}
	var inspection xeadsource.Inspection
	if err := json.Unmarshal(sourceExecution.ResultPayload, &inspection); err != nil {
		t.Fatal(err)
	}
	if inspection.Document.Format != xeadsource.FormatXead ||
		inspection.Document.SHA256 == "" {
		t.Fatalf("source inspection=%+v", inspection)
	}

	queryTarget := realization.ContractQueryTarget{
		Database: core.MustCanonicalID("sales"),
		Contract: core.MustCanonicalID("customer_read"),
		Query:    core.MustLocalName("find_by_id"),
	}
	queryDefinition := database.QueryContract{
		Name:              "find_by_id",
		Result:            core.MustCanonicalID("customer"),
		ResultCardinality: database.QueryResultMany,
	}
	queryRequest, err := json.Marshal(ContractQueryInvocationRequest{
		Target:     queryTarget,
		Contract:   queryDefinition,
		Parameters: map[string]any{"customer_id": "C-001"},
	})
	if err != nil {
		t.Fatal(err)
	}
	queryExecution, err := registry.ResolveAndInvoke(
		context.Background(),
		ResolveInput{
			Scope:       ScopeApplicationRuntime,
			Requirement: Requirement{ContractID: queryContract.ID, CompatibleMajor: 1},
			Contracts:   []Contract{queryContract},
			Providers:   []Provider{queryProvider},
			Evidence: []ConformanceEvidence{
				evidenceFor(queryProvider, queryContract, "query-conformance"),
			},
		},
		queryRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if queryExecution.Resolution.Provider != queryProvider ||
		queryExecution.RequestShape != contractQueryRequestShape ||
		queryExecution.RequestShapeFingerprint != contractQueryRequestShapeFingerprint ||
		queryExecution.ResultShape != contractQueryResultShape ||
		queryExecution.ResultShapeFingerprint != contractQueryResultShapeFingerprint {
		t.Fatalf("query invocation execution=%+v", queryExecution)
	}
	var queryResult driver.ContractQueryResult
	if err := json.Unmarshal(queryExecution.ResultPayload, &queryResult); err != nil {
		t.Fatal(err)
	}
	if queryExecutor.calls != 1 ||
		queryExecutor.target != queryTarget ||
		queryExecutor.contract.Name != queryDefinition.Name ||
		queryExecutor.parameters["customer_id"] != "C-001" {
		t.Fatalf(
			"query executor calls=%d target=%+v contract=%+v parameters=%+v",
			queryExecutor.calls,
			queryExecutor.target,
			queryExecutor.contract,
			queryExecutor.parameters,
		)
	}
	if len(queryResult.Rows) != 1 ||
		queryResult.Rows[0].Attributes[core.MustCanonicalID("customer.id")] != "C-001" {
		t.Fatalf("query result=%+v", queryResult)
	}

	if sourceExecution.Resolution.Scope == queryExecution.Resolution.Scope {
		t.Fatal("independent Authority scopes collapsed in generic invocation")
	}
}

func TestGenericInvocationRejectsContractShapeDriftBeforeDomainCall(t *testing.T) {
	contract := sourceAdapterContract()
	contract.RequestShape = "tei.reference.source-adapter.request/v999"
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	typed, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := sourceAdapterInvocationImplementation(typed)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, provider),
		payload,
	)
	if err == nil || !strings.Contains(err.Error(), "invocation shape") {
		t.Fatalf("contract / implementation shape drift was accepted: %v", err)
	}
}

func TestGenericInvocationRejectsSameShapeNameWithDifferentSchemaFingerprint(t *testing.T) {
	contract := sourceAdapterContract()
	contract.RequestShapeFingerprint = "sha256:" + strings.Repeat("f", 64)
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	typed, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := sourceAdapterInvocationImplementation(typed)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, provider),
		payload,
	)
	if err == nil || !strings.Contains(err.Error(), "invocation shape") {
		t.Fatalf("same shape name with different schema fingerprint was accepted: %v", err)
	}
}

func TestGenericInvocationRejectsResolvedButUninstalledExactArtifact(t *testing.T) {
	registry, contract, installed, _, _, _ := genericInvocationRegistryFixture(t)
	uninstalled := installed
	uninstalled.ArtifactDigest = "sha256:" + strings.Repeat("b", 64)
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, uninstalled),
		payload,
	)
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("uninstalled exact artifact was executed: %v", err)
	}
}

func TestGenericInvocationForwardsCancellationWithoutFallback(t *testing.T) {
	registry, sourceContract, sourceProvider, _, _, _ := genericInvocationRegistryFixture(t)
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = registry.ResolveAndInvoke(
		ctx,
		sourceAdapterResolveInput(sourceContract, sourceProvider),
		payload,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled invocation did not remain canceled: %v", err)
	}
}

func TestInvocationPayloadCodecIsContractScopedRatherThanJSONHardCoded(t *testing.T) {
	contract := Contract{
		ID:                 "build.reference.opaque-copy",
		Version:            "1.0.0",
		RequestSemantics:   "opaque-bytes",
		ResultSemantics:    "same-opaque-bytes",
		FailureSemantics:   "execution-error-is-not-success",
		RequestShape:            "tei.reference.opaque.request/v0",
		RequestShapeFingerprint: "sha256:c43afa7741de2f84dd9d149ef6f4fde2d0bd2f347d7a377176cb1aa24adab8b2",
		ResultShape:             "tei.reference.opaque.result/v0",
		ResultShapeFingerprint:  "sha256:c43afa7741de2f84dd9d149ef6f4fde2d0bd2f347d7a377176cb1aa24adab8b2",
		PayloadCodec:            "application/octet-stream;profile=tei-reference-v0",
		ConformanceProfile: "tei.reference.opaque-copy/v0",
	}
	provider := sourceAdapterProvider(
		contract,
		ScopeBuildTool,
		"reference.build.opaque",
		"opaque.copy.v1",
	)
	implementation := InvocationImplementation{
		Provider:                provider,
		RequestShape:            contract.RequestShape,
		RequestShapeFingerprint: contract.RequestShapeFingerprint,
		ResultShape:             contract.ResultShape,
		ResultShapeFingerprint:  contract.ResultShapeFingerprint,
		PayloadCodec:            contract.PayloadCodec,
		Invoke: func(ctx context.Context, payload []byte) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return append([]byte(nil), payload...), nil
		},
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte{0x00, 0xff, 0x10, 0x20}
	execution, err := registry.ResolveAndInvoke(
		context.Background(),
		ResolveInput{
			Scope:       ScopeBuildTool,
			Requirement: Requirement{ContractID: contract.ID, CompatibleMajor: 1},
			Contracts:   []Contract{contract},
			Providers:   []Provider{provider},
			Evidence: []ConformanceEvidence{
				evidenceFor(provider, contract, "opaque-conformance"),
			},
		},
		payload,
	)
	if err != nil {
		t.Fatal(err)
	}
	if execution.PayloadCodec != contract.PayloadCodec ||
		string(execution.ResultPayload) != string(payload) {
		t.Fatalf("opaque invocation execution=%+v", execution)
	}
	if execution.PayloadCodec == referenceJSONPayloadCodec {
		t.Fatal("generic invocation hard-coded the Reference JSON codec")
	}
}

func TestGenericInvocationDoesNotAddAuthorityBeyondResolvedCapability(t *testing.T) {
	registry, contract, provider, _, _, _ := genericInvocationRegistryFixture(t)
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, provider),
		payload,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"capability-resolution-is-not-canonical-acceptance",
		"capability-invocation-does-not-grant-additional-authority",
		"payload-codec-is-contract-scoped-not-core-semantic-model",
		"authoring-tool-capability-is-not-application-runtime-authority",
	} {
		if !hasSourceAdapterLimit(execution.AuthorityLimits, want) {
			t.Fatalf("authority limit %q missing: %+v", want, execution.AuthorityLimits)
		}
	}
}


func TestReferenceJSONInvocationRejectsUnknownRequestField(t *testing.T) {
	registry, contract, provider, _, _, _ := genericInvocationRegistryFixture(t)
	payload, err := json.Marshal(map[string]any{
		"Format":                xeadsource.FormatXead,
		"SourceBytes":           []byte(`<System FormatVersion="1.2"/>`),
		"TranscriptionContract": sourceAdapterTranscriptionContract(),
		"Unexpected":            true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, provider),
		payload,
	)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown Reference JSON field was accepted: %v", err)
	}
}

func TestReferenceJSONInvocationRejectsTrailingJSONValue(t *testing.T) {
	registry, contract, provider, _, _, _ := genericInvocationRegistryFixture(t)
	payload, err := json.Marshal(SourceAdapterRequest{
		Format:                xeadsource.FormatXead,
		SourceBytes:           []byte(`<System FormatVersion="1.2"/>`),
		TranscriptionContract: sourceAdapterTranscriptionContract(),
	})
	if err != nil {
		t.Fatal(err)
	}
	payload = append(payload, []byte(" {}")...)
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, provider),
		payload,
	)
	if err == nil || !strings.Contains(err.Error(), "trailing JSON value") {
		t.Fatalf("trailing Reference JSON value was accepted: %v", err)
	}
}


func TestGenericInvocationRejectsResolvedContractWithoutInvocationShape(t *testing.T) {
	contract := sourceAdapterContract()
	contract.RequestShape = ""
	contract.RequestShapeFingerprint = ""
	contract.ResultShape = ""
	contract.ResultShapeFingerprint = ""
	contract.PayloadCodec = ""
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	typed, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := sourceAdapterInvocationImplementation(typed)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewInvocationRegistry([]InvocationImplementation{implementation})
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ResolveAndInvoke(
		context.Background(),
		sourceAdapterResolveInput(contract, provider),
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "machine-readable invocation shape") {
		t.Fatalf("resolved contract without invocation shape was executed: %v", err)
	}
}

func TestInvocationRegistryRejectsMalformedShapeFingerprint(t *testing.T) {
	contract := sourceAdapterContract()
	provider := sourceAdapterProvider(
		contract,
		ScopeAuthoringTool,
		"reference.compat.xead.source",
		referenceXeadReaderImplementationID,
	)
	typed, err := ReferenceXeadSourceAdapterImplementation(provider)
	if err != nil {
		t.Fatal(err)
	}
	implementation, err := sourceAdapterInvocationImplementation(typed)
	if err != nil {
		t.Fatal(err)
	}
	implementation.RequestShapeFingerprint = "not-a-sha256"
	if _, err := NewInvocationRegistry([]InvocationImplementation{implementation}); err == nil ||
		!strings.Contains(err.Error(), "incomplete invocation shape") {
		t.Fatalf("malformed invocation shape fingerprint was accepted: %v", err)
	}
}
