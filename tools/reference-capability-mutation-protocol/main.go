package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	mutationProtocol                    = "prepare-validate-persist-commit/v0"
	mutationFingerprintCanonicalization = "go-json-marshal-struct/v0"
	rawPayloadDigestRule                = "raw-bytes+sha256/v0"

	OutcomeCommitted    MutationOutcome = "committed"
	OutcomeNotCommitted MutationOutcome = "not-committed"
	OutcomeUnknown      MutationOutcome = "unknown"
)

var (
	ErrResultValidation   = errors.New("prepared mutation result validation failed")
	ErrCommitNotCommitted = errors.New("mutation was not committed")
	ErrCommitUnknown      = errors.New("mutation commit outcome is unknown")
	ErrJournalWrite       = errors.New("mutation journal write failed")
)

type MutationOutcome string

type MutationRequest struct {
	InvocationKey string
	Payload       []byte
}

type PrepareRequest struct {
	Protocol      string
	InvocationKey string
	RequestDigest string
	Payload       []byte
}

type PreparedMutation struct {
	Protocol               string
	PreparationID          string
	InvocationKey          string
	RequestDigest          string
	ResultPayload          []byte
	ResultDigest           string
	ConcurrencyToken       string
	PreparationFingerprint string
}

type CommitRequest struct {
	Protocol         string
	PreparationID   string
	InvocationKey   string
	RequestDigest   string
	ResultDigest    string
	ConcurrencyToken string
}

type ProviderCommitReceipt struct {
	Protocol           string
	Outcome            MutationOutcome
	PreparationID      string
	InvocationKey      string
	RequestDigest      string
	ResultDigest       string
	EffectDigest       string
	CommitID           string
	ReceiptFingerprint string
}

type StatusRequest struct {
	Protocol      string
	PreparationID string
	InvocationKey string
	RequestDigest string
	ResultDigest  string
}

type MutationHandle struct {
	Protocol               string
	PreparationID          string
	InvocationKey          string
	RequestDigest          string
	ResultDigest           string
	ConcurrencyToken       string
	PreparationFingerprint string
}

type MutationReceipt struct {
	Protocol                   string
	Outcome                    MutationOutcome
	InvocationKey              string
	RequestDigest              string
	ResultDigest               string
	PreparationFingerprint     string
	ProviderReceiptFingerprint string
	EffectDigest               string
	CommitID                   string
	ReceiptFingerprint         string
}

type MutationExecution struct {
	Outcome MutationOutcome
	Result  []byte
	Handle  MutationHandle
	Receipt *MutationReceipt
}

type ResultValidator func([]byte) error

type MutationProvider interface {
	Prepare(context.Context, PrepareRequest) (PreparedMutation, error)
	Commit(context.Context, CommitRequest) (ProviderCommitReceipt, error)
	Status(context.Context, StatusRequest) (ProviderCommitReceipt, error)
	Abort(context.Context, MutationHandle) error
}

type MutationJournal interface {
	RecordPrepared(context.Context, MutationHandle) error
	RecordOutcome(context.Context, MutationReceipt) error
	LoadPrepared(context.Context, string) (MutationHandle, bool, error)
	LoadOutcome(context.Context, string) (MutationReceipt, bool, error)
}

type Executor struct {
	Provider MutationProvider
	Journal  MutationJournal
	Validate ResultValidator
}

func (e Executor) Invoke(ctx context.Context, request MutationRequest) (MutationExecution, error) {
	if e.Provider == nil {
		return MutationExecution{}, fmt.Errorf("mutation provider is required")
	}
	if e.Journal == nil {
		return MutationExecution{}, fmt.Errorf("mutation journal is required")
	}
	if e.Validate == nil {
		return MutationExecution{}, fmt.Errorf("result validator is required")
	}
	if request.InvocationKey == "" {
		return MutationExecution{}, fmt.Errorf("invocation key is required")
	}
	if err := ctx.Err(); err != nil {
		return MutationExecution{Outcome: OutcomeNotCommitted}, err
	}

	payload := append([]byte(nil), request.Payload...)
	requestDigest := digest(payload)
	prepared, err := e.Provider.Prepare(ctx, PrepareRequest{
		Protocol:      mutationProtocol,
		InvocationKey: request.InvocationKey,
		RequestDigest: requestDigest,
		Payload:       payload,
	})
	if err != nil {
		return MutationExecution{Outcome: OutcomeNotCommitted}, err
	}
	handle, preparedResult, err := validatePrepared(request.InvocationKey, requestDigest, prepared)
	if err != nil {
		abortPrepared(e.Provider, ctx, preparedHandle(prepared))
		return MutationExecution{Outcome: OutcomeNotCommitted}, err
	}

	if err := e.Validate(preparedResult); err != nil {
		abortPrepared(e.Provider, ctx, handle)
		return MutationExecution{
			Outcome: OutcomeNotCommitted,
			Handle:  handle,
		}, fmt.Errorf("%w: %v", ErrResultValidation, err)
	}
	if err := ctx.Err(); err != nil {
		abortPrepared(e.Provider, ctx, handle)
		return MutationExecution{
			Outcome: OutcomeNotCommitted,
			Handle:  handle,
		}, err
	}
	if err := e.Journal.RecordPrepared(ctx, handle); err != nil {
		abortPrepared(e.Provider, ctx, handle)
		return MutationExecution{
			Outcome: OutcomeNotCommitted,
			Handle:  handle,
		}, fmt.Errorf("%w: prepared intent: %v", ErrJournalWrite, err)
	}

	providerReceipt, commitErr := e.Provider.Commit(ctx, CommitRequest{
		Protocol:          mutationProtocol,
		PreparationID:    handle.PreparationID,
		InvocationKey:    handle.InvocationKey,
		RequestDigest:    handle.RequestDigest,
		ResultDigest:     handle.ResultDigest,
		ConcurrencyToken: handle.ConcurrencyToken,
	})
	if commitErr != nil {
		return MutationExecution{
			Outcome: OutcomeUnknown,
			Handle:  handle,
		}, fmt.Errorf("%w: %v", ErrCommitUnknown, commitErr)
	}

	receipt, err := validateProviderReceipt(handle, providerReceipt)
	if err != nil {
		return MutationExecution{
			Outcome: OutcomeUnknown,
			Handle:  handle,
		}, fmt.Errorf("%w: invalid provider commit receipt: %v", ErrCommitUnknown, err)
	}
	switch receipt.Outcome {
	case OutcomeCommitted:
		if err := e.Journal.RecordOutcome(ctx, receipt); err != nil {
			return MutationExecution{
				Outcome: OutcomeCommitted,
				Handle:  handle,
				Receipt: &receipt,
			}, fmt.Errorf("%w: committed receipt: %v", ErrJournalWrite, err)
		}
		return MutationExecution{
			Outcome: OutcomeCommitted,
			Result:  append([]byte(nil), preparedResult...),
			Handle:  handle,
			Receipt: &receipt,
		}, nil
	case OutcomeNotCommitted:
		if err := e.Journal.RecordOutcome(ctx, receipt); err != nil {
			return MutationExecution{
				Outcome: OutcomeNotCommitted,
				Handle:  handle,
				Receipt: &receipt,
			}, fmt.Errorf("%w: not-committed receipt: %v", ErrJournalWrite, err)
		}
		return MutationExecution{
			Outcome: OutcomeNotCommitted,
			Handle:  handle,
			Receipt: &receipt,
		}, ErrCommitNotCommitted
	default:
		return MutationExecution{
			Outcome: OutcomeUnknown,
			Handle:  handle,
		}, ErrCommitUnknown
	}
}

func (e Executor) ReconcileByInvocationKey(
	ctx context.Context,
	invocationKey string,
) (MutationExecution, error) {
	if e.Journal == nil {
		return MutationExecution{}, fmt.Errorf("mutation journal is required")
	}
	if invocationKey == "" {
		return MutationExecution{}, fmt.Errorf("invocation key is required")
	}
	handle, ok, err := e.Journal.LoadPrepared(ctx, invocationKey)
	if err != nil {
		return MutationExecution{}, fmt.Errorf("%w: load prepared intent: %v", ErrJournalWrite, err)
	}
	if !ok {
		return MutationExecution{}, fmt.Errorf("prepared mutation %q is not journaled", invocationKey)
	}
	if receipt, found, err := e.Journal.LoadOutcome(ctx, invocationKey); err != nil {
		return MutationExecution{}, fmt.Errorf("%w: load outcome: %v", ErrJournalWrite, err)
	} else if found {
		if receipt.Outcome == OutcomeCommitted {
			return MutationExecution{
				Outcome: OutcomeCommitted,
				Handle:  handle,
				Receipt: &receipt,
			}, nil
		}
		if receipt.Outcome == OutcomeNotCommitted {
			return MutationExecution{
				Outcome: OutcomeNotCommitted,
				Handle:  handle,
				Receipt: &receipt,
			}, ErrCommitNotCommitted
		}
	}
	return e.Reconcile(ctx, handle)
}

func (e Executor) Reconcile(ctx context.Context, handle MutationHandle) (MutationExecution, error) {
	if e.Provider == nil {
		return MutationExecution{}, fmt.Errorf("mutation provider is required")
	}
	if e.Journal == nil {
		return MutationExecution{}, fmt.Errorf("mutation journal is required")
	}
	if err := validateHandle(handle); err != nil {
		return MutationExecution{}, err
	}
	providerReceipt, err := e.Provider.Status(ctx, StatusRequest{
		Protocol:      mutationProtocol,
		PreparationID: handle.PreparationID,
		InvocationKey: handle.InvocationKey,
		RequestDigest: handle.RequestDigest,
		ResultDigest:  handle.ResultDigest,
	})
	if err != nil {
		return MutationExecution{
			Outcome: OutcomeUnknown,
			Handle:  handle,
		}, fmt.Errorf("%w: status: %v", ErrCommitUnknown, err)
	}
	receipt, err := validateProviderReceipt(handle, providerReceipt)
	if err != nil {
		return MutationExecution{
			Outcome: OutcomeUnknown,
			Handle:  handle,
		}, fmt.Errorf("%w: invalid status receipt: %v", ErrCommitUnknown, err)
	}
	if receipt.Outcome == OutcomeUnknown {
		return MutationExecution{
			Outcome: OutcomeUnknown,
			Handle:  handle,
		}, ErrCommitUnknown
	}
	if err := e.Journal.RecordOutcome(ctx, receipt); err != nil {
		return MutationExecution{
			Outcome: receipt.Outcome,
			Handle:  handle,
			Receipt: &receipt,
		}, fmt.Errorf("%w: reconciled receipt: %v", ErrJournalWrite, err)
	}
	if receipt.Outcome == OutcomeNotCommitted {
		return MutationExecution{
			Outcome: OutcomeNotCommitted,
			Handle:  handle,
			Receipt: &receipt,
		}, ErrCommitNotCommitted
	}
	return MutationExecution{
		Outcome: OutcomeCommitted,
		Handle:  handle,
		Receipt: &receipt,
	}, nil
}

func validatePrepared(
	invocationKey,
	requestDigest string,
	prepared PreparedMutation,
) (MutationHandle, []byte, error) {
	if prepared.Protocol != mutationProtocol {
		return MutationHandle{}, nil, fmt.Errorf(
			"prepared mutation protocol %q is unsupported",
			prepared.Protocol,
		)
	}
	if prepared.PreparationID == "" ||
		prepared.InvocationKey != invocationKey ||
		prepared.RequestDigest != requestDigest ||
		prepared.ConcurrencyToken == "" {
		return MutationHandle{}, nil, fmt.Errorf("prepared mutation identity is inconsistent")
	}
	if prepared.ResultDigest != digest(prepared.ResultPayload) {
		return MutationHandle{}, nil, fmt.Errorf("prepared result digest is stale")
	}
	expected := preparedFingerprint(prepared)
	if prepared.PreparationFingerprint != expected {
		return MutationHandle{}, nil, fmt.Errorf("prepared mutation fingerprint is stale")
	}
	handle := preparedHandle(prepared)
	if err := validateHandle(handle); err != nil {
		return MutationHandle{}, nil, err
	}
	return handle, append([]byte(nil), prepared.ResultPayload...), nil
}

func validateHandle(handle MutationHandle) error {
	if handle.Protocol != mutationProtocol ||
		handle.PreparationID == "" ||
		handle.InvocationKey == "" ||
		handle.RequestDigest == "" ||
		handle.ResultDigest == "" ||
		handle.ConcurrencyToken == "" ||
		handle.PreparationFingerprint == "" {
		return fmt.Errorf("mutation handle is incomplete")
	}
	prepared := PreparedMutation{
		Protocol:               handle.Protocol,
		PreparationID:          handle.PreparationID,
		InvocationKey:          handle.InvocationKey,
		RequestDigest:          handle.RequestDigest,
		ResultDigest:           handle.ResultDigest,
		ConcurrencyToken:       handle.ConcurrencyToken,
		PreparationFingerprint: handle.PreparationFingerprint,
	}
	if preparedFingerprint(prepared) != handle.PreparationFingerprint {
		return fmt.Errorf("mutation handle fingerprint is stale")
	}
	return nil
}

func validateProviderReceipt(handle MutationHandle, provider ProviderCommitReceipt) (MutationReceipt, error) {
	if provider.Protocol != mutationProtocol ||
		provider.PreparationID != handle.PreparationID ||
		provider.InvocationKey != handle.InvocationKey ||
		provider.RequestDigest != handle.RequestDigest ||
		provider.ResultDigest != handle.ResultDigest {
		return MutationReceipt{}, fmt.Errorf("provider receipt does not match prepared mutation")
	}
	if provider.Outcome != OutcomeCommitted &&
		provider.Outcome != OutcomeNotCommitted &&
		provider.Outcome != OutcomeUnknown {
		return MutationReceipt{}, fmt.Errorf("provider receipt has invalid outcome %q", provider.Outcome)
	}
	if provider.Outcome == OutcomeCommitted &&
		(provider.EffectDigest == "" || provider.CommitID == "") {
		return MutationReceipt{}, fmt.Errorf("committed provider receipt is incomplete")
	}
	if provider.ReceiptFingerprint != providerReceiptFingerprint(provider) {
		return MutationReceipt{}, fmt.Errorf("provider receipt fingerprint is stale")
	}
	receipt := MutationReceipt{
		Protocol:                   mutationProtocol,
		Outcome:                    provider.Outcome,
		InvocationKey:              handle.InvocationKey,
		RequestDigest:              handle.RequestDigest,
		ResultDigest:               handle.ResultDigest,
		PreparationFingerprint:     handle.PreparationFingerprint,
		ProviderReceiptFingerprint: provider.ReceiptFingerprint,
		EffectDigest:               provider.EffectDigest,
		CommitID:                   provider.CommitID,
	}
	receipt.ReceiptFingerprint = mutationReceiptFingerprint(receipt)
	return receipt, nil
}

func preparedHandle(prepared PreparedMutation) MutationHandle {
	return MutationHandle{
		Protocol:               prepared.Protocol,
		PreparationID:          prepared.PreparationID,
		InvocationKey:          prepared.InvocationKey,
		RequestDigest:          prepared.RequestDigest,
		ResultDigest:           prepared.ResultDigest,
		ConcurrencyToken:       prepared.ConcurrencyToken,
		PreparationFingerprint: prepared.PreparationFingerprint,
	}
}

func NewPreparedMutation(
	preparationID,
	invocationKey,
	requestDigest string,
	resultPayload []byte,
	concurrencyToken string,
) PreparedMutation {
	prepared := PreparedMutation{
		Protocol:         mutationProtocol,
		PreparationID:    preparationID,
		InvocationKey:    invocationKey,
		RequestDigest:    requestDigest,
		ResultPayload:    append([]byte(nil), resultPayload...),
		ResultDigest:     digest(resultPayload),
		ConcurrencyToken: concurrencyToken,
	}
	prepared.PreparationFingerprint = preparedFingerprint(prepared)
	return prepared
}

func NewProviderReceipt(
	outcome MutationOutcome,
	handle MutationHandle,
	effectDigest,
	commitID string,
) ProviderCommitReceipt {
	receipt := ProviderCommitReceipt{
		Protocol:      mutationProtocol,
		Outcome:       outcome,
		PreparationID: handle.PreparationID,
		InvocationKey: handle.InvocationKey,
		RequestDigest: handle.RequestDigest,
		ResultDigest:  handle.ResultDigest,
		EffectDigest:  effectDigest,
		CommitID:      commitID,
	}
	receipt.ReceiptFingerprint = providerReceiptFingerprint(receipt)
	return receipt
}

func abortPrepared(provider MutationProvider, ctx context.Context, handle MutationHandle) {
	if provider == nil {
		return
	}
	// Request cancellation must not prevent cleanup of uncommitted prepared state.
	// Values are preserved for Host authority checks, while cancellation/deadline
	// are intentionally removed for this best-effort cleanup call.
	_ = provider.Abort(context.WithoutCancel(ctx), handle)
}

func preparedFingerprint(prepared PreparedMutation) string {
	return structuredFingerprint(struct {
		Canonicalization string `json:"canonicalization"`
		Protocol         string `json:"protocol"`
		PreparationID    string `json:"preparationId"`
		InvocationKey    string `json:"invocationKey"`
		RequestDigest    string `json:"requestDigest"`
		ResultDigest     string `json:"resultDigest"`
		ConcurrencyToken string `json:"concurrencyToken"`
	}{
		Canonicalization: mutationFingerprintCanonicalization,
		Protocol:         prepared.Protocol,
		PreparationID:    prepared.PreparationID,
		InvocationKey:    prepared.InvocationKey,
		RequestDigest:    prepared.RequestDigest,
		ResultDigest:     prepared.ResultDigest,
		ConcurrencyToken: prepared.ConcurrencyToken,
	})
}

func providerReceiptFingerprint(receipt ProviderCommitReceipt) string {
	return structuredFingerprint(struct {
		Canonicalization string          `json:"canonicalization"`
		Protocol         string          `json:"protocol"`
		Outcome          MutationOutcome `json:"outcome"`
		PreparationID    string          `json:"preparationId"`
		InvocationKey    string          `json:"invocationKey"`
		RequestDigest    string          `json:"requestDigest"`
		ResultDigest     string          `json:"resultDigest"`
		EffectDigest     string          `json:"effectDigest,omitempty"`
		CommitID         string          `json:"commitId,omitempty"`
	}{
		Canonicalization: mutationFingerprintCanonicalization,
		Protocol:         receipt.Protocol,
		Outcome:          receipt.Outcome,
		PreparationID:    receipt.PreparationID,
		InvocationKey:    receipt.InvocationKey,
		RequestDigest:    receipt.RequestDigest,
		ResultDigest:     receipt.ResultDigest,
		EffectDigest:     receipt.EffectDigest,
		CommitID:         receipt.CommitID,
	})
}

func mutationReceiptFingerprint(receipt MutationReceipt) string {
	return structuredFingerprint(struct {
		Canonicalization           string          `json:"canonicalization"`
		Protocol                   string          `json:"protocol"`
		Outcome                    MutationOutcome `json:"outcome"`
		InvocationKey              string          `json:"invocationKey"`
		RequestDigest              string          `json:"requestDigest"`
		ResultDigest               string          `json:"resultDigest"`
		PreparationFingerprint     string          `json:"preparationFingerprint"`
		ProviderReceiptFingerprint string          `json:"providerReceiptFingerprint"`
		EffectDigest               string          `json:"effectDigest,omitempty"`
		CommitID                   string          `json:"commitId,omitempty"`
	}{
		Canonicalization:           mutationFingerprintCanonicalization,
		Protocol:                   receipt.Protocol,
		Outcome:                    receipt.Outcome,
		InvocationKey:              receipt.InvocationKey,
		RequestDigest:              receipt.RequestDigest,
		ResultDigest:               receipt.ResultDigest,
		PreparationFingerprint:     receipt.PreparationFingerprint,
		ProviderReceiptFingerprint: receipt.ProviderReceiptFingerprint,
		EffectDigest:               receipt.EffectDigest,
		CommitID:                   receipt.CommitID,
	})
}

func structuredFingerprint(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return digest(encoded)
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func main() {}
