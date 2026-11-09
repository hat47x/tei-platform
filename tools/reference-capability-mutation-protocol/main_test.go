package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type ledgerRecord struct {
	prepared  PreparedMutation
	delta     int
	committed bool
	aborted   bool
	receipt   ProviderCommitReceipt
}

type ledgerProvider struct {
	value                      int
	version                    int
	records                    map[string]*ledgerRecord
	prepareCount               int
	commitCount                int
	statusCount                int
	effectCount                int
	commitSawCanceledContext   bool
	abortCount                 int
	abortSawCanceledContext    bool
	abortError                 bool
	commitErrorAfterApply      bool
	malformedReceiptAfterApply bool
	forceNotCommitted          bool
	statusUnknown              bool
}

func newLedgerProvider() *ledgerProvider {
	return &ledgerProvider{records: map[string]*ledgerRecord{}}
}

func (p *ledgerProvider) Prepare(_ context.Context, request PrepareRequest) (PreparedMutation, error) {
	p.prepareCount++
	if request.Protocol != mutationProtocol {
		return PreparedMutation{}, fmt.Errorf("unsupported protocol")
	}
	delta, err := strconv.Atoi(string(request.Payload))
	if err != nil {
		return PreparedMutation{}, err
	}
	if existing, ok := p.records[request.InvocationKey]; ok {
		if existing.prepared.BindingFingerprint != request.BindingFingerprint {
			return PreparedMutation{}, fmt.Errorf("idempotency key reused across execution binding")
		}
		if existing.prepared.RequestDigest != request.RequestDigest {
			return PreparedMutation{}, fmt.Errorf("idempotency key reused with different request")
		}
		return clonePrepared(existing.prepared), nil
	}
	result := []byte(strconv.Itoa(p.value + delta))
	prepared := NewPreparedMutation(
		"prep-"+request.InvocationKey,
		request.InvocationKey,
		request.BindingFingerprint,
		request.RequestDigest,
		result,
		fmt.Sprintf("version-%d", p.version),
	)
	p.records[request.InvocationKey] = &ledgerRecord{
		prepared: clonePrepared(prepared),
		delta:    delta,
	}
	return clonePrepared(prepared), nil
}

func (p *ledgerProvider) Commit(ctx context.Context, request CommitRequest) (ProviderCommitReceipt, error) {
	p.commitCount++
	if ctx.Err() != nil {
		p.commitSawCanceledContext = true
	}
	record, ok := p.records[request.InvocationKey]
	if !ok ||
		record.prepared.PreparationID != request.PreparationID ||
		record.prepared.BindingFingerprint != request.BindingFingerprint {
		return ProviderCommitReceipt{}, fmt.Errorf("unknown preparation")
	}
	if record.aborted {
		return ProviderCommitReceipt{}, fmt.Errorf("preparation was aborted")
	}
	handle := preparedHandle(record.prepared)
	if record.committed {
		return record.receipt, nil
	}
	if request.BindingFingerprint != handle.BindingFingerprint ||
		request.RequestDigest != handle.RequestDigest ||
		request.ResultDigest != handle.ResultDigest ||
		request.ConcurrencyToken != handle.ConcurrencyToken {
		return ProviderCommitReceipt{}, fmt.Errorf("commit request does not match preparation")
	}
	if p.forceNotCommitted || request.ConcurrencyToken != fmt.Sprintf("version-%d", p.version) {
		receipt := NewProviderReceipt(
			OutcomeNotCommitted,
			handle,
			"",
			"",
		)
		record.receipt = receipt
		return receipt, nil
	}

	p.value += record.delta
	p.version++
	p.effectCount++
	record.committed = true
	record.receipt = NewProviderReceipt(
		OutcomeCommitted,
		handle,
		digest([]byte(fmt.Sprintf("value=%d;version=%d", p.value, p.version))),
		"commit-"+request.InvocationKey,
	)
	if p.commitErrorAfterApply {
		return ProviderCommitReceipt{}, fmt.Errorf("synthetic transport failure after apply")
	}
	if p.malformedReceiptAfterApply {
		bad := record.receipt
		bad.ResultDigest = digest([]byte("wrong-result"))
		bad.ReceiptFingerprint = providerReceiptFingerprint(bad)
		return bad, nil
	}
	return record.receipt, nil
}

func (p *ledgerProvider) Status(_ context.Context, request StatusRequest) (ProviderCommitReceipt, error) {
	p.statusCount++
	record, ok := p.records[request.InvocationKey]
	if !ok || record.prepared.PreparationID != request.PreparationID {
		return ProviderCommitReceipt{}, fmt.Errorf("unknown preparation")
	}
	if p.statusUnknown {
		return NewProviderReceipt(
			OutcomeUnknown,
			preparedHandle(record.prepared),
			"",
			"",
		), nil
	}
	if record.committed {
		return record.receipt, nil
	}
	if record.receipt.Outcome == OutcomeNotCommitted {
		return record.receipt, nil
	}
	return NewProviderReceipt(
		OutcomeNotCommitted,
		preparedHandle(record.prepared),
		"",
		"",
	), nil
}

func (p *ledgerProvider) Abort(ctx context.Context, handle MutationHandle) error {
	p.abortCount++
	if ctx.Err() != nil {
		p.abortSawCanceledContext = true
	}
	if p.abortError {
		return fmt.Errorf("synthetic abort cleanup failure")
	}
	record, ok := p.records[handle.InvocationKey]
	if !ok {
		return nil
	}
	if !record.committed {
		record.aborted = true
		delete(p.records, handle.InvocationKey)
	}
	return nil
}

type memoryJournal struct {
	intents        map[string]MutationHandle
	outcomes       map[string]MutationReceipt
	failIntent     bool
	failOutcome    bool
	onRecordIntent func()
}

func newMemoryJournal() *memoryJournal {
	return &memoryJournal{
		intents:  map[string]MutationHandle{},
		outcomes: map[string]MutationReceipt{},
	}
}

func (j *memoryJournal) RecordCommitIntent(_ context.Context, handle MutationHandle) error {
	if j.failIntent {
		return fmt.Errorf("synthetic commit intent journal failure")
	}
	if existing, ok := j.intents[handle.InvocationKey]; ok {
		if existing != handle {
			return fmt.Errorf("commit intent conflict for invocation %s", handle.InvocationKey)
		}
		if j.onRecordIntent != nil {
			j.onRecordIntent()
		}
		return nil
	}
	j.intents[handle.InvocationKey] = cloneHandle(handle)
	if j.onRecordIntent != nil {
		j.onRecordIntent()
	}
	return nil
}

func (j *memoryJournal) RecordOutcome(_ context.Context, receipt MutationReceipt) error {
	if j.failOutcome {
		return fmt.Errorf("synthetic outcome journal failure")
	}
	if existing, ok := j.outcomes[receipt.InvocationKey]; ok {
		if existing != receipt {
			return fmt.Errorf("outcome conflict for invocation %s", receipt.InvocationKey)
		}
		return nil
	}
	j.outcomes[receipt.InvocationKey] = receipt
	return nil
}

func (j *memoryJournal) LoadCommitIntent(_ context.Context, invocationKey string) (MutationHandle, bool, error) {
	handle, ok := j.intents[invocationKey]
	if !ok {
		return MutationHandle{}, false, nil
	}
	return cloneHandle(handle), true, nil
}

func (j *memoryJournal) LoadOutcome(_ context.Context, invocationKey string) (MutationReceipt, bool, error) {
	receipt, ok := j.outcomes[invocationKey]
	return receipt, ok, nil
}

func numericResultValidator(value []byte) error {
	if _, err := strconv.Atoi(string(value)); err != nil {
		return fmt.Errorf("result is not an integer")
	}
	return nil
}

func testExecutionBinding(marker string) string {
	return digest([]byte("execution-binding:" + marker))
}

func newExecutorWithBinding(
	provider *ledgerProvider,
	journal *memoryJournal,
	bindingFingerprint string,
) Executor {
	return Executor{
		BindingFingerprint: bindingFingerprint,
		Provider:           provider,
		Journal:            journal,
		Validate:           numericResultValidator,
	}
}

func newExecutor(provider *ledgerProvider, journal *memoryJournal) Executor {
	return newExecutorWithBinding(
		provider,
		journal,
		testExecutionBinding("default"),
	)
}

func clonePrepared(value PreparedMutation) PreparedMutation {
	value.ResultPayload = append([]byte(nil), value.ResultPayload...)
	return value
}

func cloneHandle(value MutationHandle) MutationHandle {
	return value
}

func TestPreparedResultIsValidatedBeforeCommit(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)
	executor.Validate = func([]byte) error {
		return fmt.Errorf("shape mismatch")
	}

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-1",
		Payload:       []byte("3"),
	})
	if err == nil || !errors.Is(err, ErrResultValidation) {
		t.Fatalf("result validation failure was not returned: execution=%+v err=%v", execution, err)
	}
	if execution.Outcome != OutcomeNotCommitted ||
		provider.value != 0 ||
		provider.effectCount != 0 ||
		provider.commitCount != 0 ||
		len(journal.intents) != 0 {
		t.Fatalf(
			"invalid result crossed commit boundary: execution=%+v value=%d effects=%d commits=%d prepared=%d",
			execution,
			provider.value,
			provider.effectCount,
			provider.commitCount,
			len(journal.intents),
		)
	}
	if provider.abortCount != 1 {
		t.Fatalf("invalid preparation was not aborted: %d", provider.abortCount)
	}
}

func TestCommitIntentMustBeJournaledBeforeCommit(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	journal.failIntent = true
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-2",
		Payload:       []byte("4"),
	})
	if err == nil || !errors.Is(err, ErrJournalWrite) {
		t.Fatalf("journal failure was not returned: execution=%+v err=%v", execution, err)
	}
	if provider.effectCount != 0 || provider.commitCount != 0 || provider.value != 0 {
		t.Fatalf("commit happened without prepared journal: %+v", provider)
	}
}

func TestCommitIntentJournalFailureCanRetryWithoutReusingAbortedPreparation(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	journal.failIntent = true
	executor := newExecutor(provider, journal)
	request := MutationRequest{
		InvocationKey: "order-journal-retry",
		Payload:       []byte("4"),
	}

	first, err := executor.Invoke(context.Background(), request)
	if err == nil || !errors.Is(err, ErrJournalWrite) ||
		first.Outcome != OutcomeNotCommitted {
		t.Fatalf("expected pre-commit journal failure: execution=%+v err=%v", first, err)
	}
	if provider.effectCount != 0 || provider.commitCount != 0 {
		t.Fatalf("journal failure crossed commit boundary: %+v", provider)
	}

	journal.failIntent = false
	second, err := executor.Invoke(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Outcome != OutcomeCommitted ||
		provider.value != 4 ||
		provider.effectCount != 1 {
		t.Fatalf("safe retry after aborted preparation failed: execution=%+v provider=%+v", second, provider)
	}
}

func TestSuccessfulMutationCommitsPreparedResultExactlyOnce(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	request := MutationRequest{InvocationKey: "order-3", Payload: []byte("5")}
	first, err := executor.Invoke(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != OutcomeCommitted ||
		string(first.Result) != "5" ||
		first.Receipt == nil ||
		provider.value != 5 ||
		provider.effectCount != 1 {
		t.Fatalf("first execution=%+v provider=%+v", first, provider)
	}
	if first.Receipt.ResultDigest != digest(first.Result) ||
		first.Receipt.RequestDigest != digest(request.Payload) ||
		first.Receipt.PreparationFingerprint != first.Handle.PreparationFingerprint {
		t.Fatalf("receipt lost prepared identity: %+v", first.Receipt)
	}

	second, err := executor.Invoke(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if second.Outcome != OutcomeCommitted ||
		provider.value != 5 ||
		provider.effectCount != 1 ||
		second.Receipt == nil ||
		second.Receipt.ReceiptFingerprint != first.Receipt.ReceiptFingerprint {
		t.Fatalf("idempotent retry duplicated effect: second=%+v provider=%+v", second, provider)
	}
}

func TestIdempotencyKeyCannotBeReusedForDifferentRequest(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	if _, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-4",
		Payload:       []byte("2"),
	}); err != nil {
		t.Fatal(err)
	}
	_, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-4",
		Payload:       []byte("7"),
	})
	if err == nil || !strings.Contains(err.Error(), "idempotency key") {
		t.Fatalf("different request reused key: %v", err)
	}
	if provider.value != 2 || provider.effectCount != 1 {
		t.Fatalf("conflicting retry changed state: %+v", provider)
	}
}

func TestCommitTransportFailureReturnsUnknownAndReconciles(t *testing.T) {
	provider := newLedgerProvider()
	provider.commitErrorAfterApply = true
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-5",
		Payload:       []byte("6"),
	})
	if err == nil || !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("commit transport failure was not unknown: execution=%+v err=%v", execution, err)
	}
	if execution.Outcome != OutcomeUnknown ||
		provider.value != 6 ||
		provider.effectCount != 1 ||
		execution.Receipt != nil {
		t.Fatalf("unknown outcome was misreported: execution=%+v provider=%+v", execution, provider)
	}

	provider.commitErrorAfterApply = false
	reconciled, err := executor.Reconcile(context.Background(), execution.Handle)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Outcome != OutcomeCommitted ||
		len(reconciled.Result) != 0 ||
		reconciled.Receipt == nil ||
		provider.effectCount != 1 {
		t.Fatalf("reconcile did not recover committed effect: %+v provider=%+v", reconciled, provider)
	}
}

func TestMalformedPostCommitReceiptReturnsUnknownUntilReconciled(t *testing.T) {
	provider := newLedgerProvider()
	provider.malformedReceiptAfterApply = true
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-6",
		Payload:       []byte("8"),
	})
	if err == nil || !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("malformed receipt was not treated as unknown: execution=%+v err=%v", execution, err)
	}
	if execution.Outcome != OutcomeUnknown || provider.value != 8 || provider.effectCount != 1 {
		t.Fatalf("post-commit uncertainty was misclassified: execution=%+v provider=%+v", execution, provider)
	}

	provider.malformedReceiptAfterApply = false
	reconciled, err := executor.Reconcile(context.Background(), execution.Handle)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Outcome != OutcomeCommitted || provider.effectCount != 1 {
		t.Fatalf("reconcile duplicated or lost effect: %+v provider=%+v", reconciled, provider)
	}
}

func TestNotCommittedOutcomeDoesNotExposePreparedResult(t *testing.T) {
	provider := newLedgerProvider()
	provider.forceNotCommitted = true
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-7",
		Payload:       []byte("9"),
	})
	if err == nil || !errors.Is(err, ErrCommitNotCommitted) {
		t.Fatalf("not-committed outcome was not returned: execution=%+v err=%v", execution, err)
	}
	if execution.Outcome != OutcomeNotCommitted ||
		len(execution.Result) != 0 ||
		provider.value != 0 ||
		provider.effectCount != 0 {
		t.Fatalf("not-committed mutation exposed committed semantics: execution=%+v provider=%+v", execution, provider)
	}
}

func TestOutcomeJournalFailureAfterCommitDoesNotPretendEffectWasRolledBack(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	journal.failOutcome = true
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-8",
		Payload:       []byte("10"),
	})
	if err == nil || !errors.Is(err, ErrJournalWrite) {
		t.Fatalf("outcome journal failure was not reported: execution=%+v err=%v", execution, err)
	}
	if execution.Outcome != OutcomeCommitted ||
		execution.Receipt == nil ||
		provider.value != 10 ||
		provider.effectCount != 1 {
		t.Fatalf("committed effect was falsely rolled back: execution=%+v provider=%+v", execution, provider)
	}

	journal.failOutcome = false
	retried, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-8",
		Payload:       []byte("10"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if retried.Outcome != OutcomeCommitted || provider.effectCount != 1 {
		t.Fatalf("retry after receipt persistence failure duplicated effect: %+v provider=%+v", retried, provider)
	}
}

func TestReconcilePreservesUnknownStatus(t *testing.T) {
	provider := newLedgerProvider()
	provider.commitErrorAfterApply = true
	provider.statusUnknown = true
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-9",
		Payload:       []byte("11"),
	})
	if err == nil || execution.Outcome != OutcomeUnknown {
		t.Fatalf("expected unknown execution: %+v err=%v", execution, err)
	}
	reconciled, err := executor.Reconcile(context.Background(), execution.Handle)
	if err == nil || !errors.Is(err, ErrCommitUnknown) ||
		reconciled.Outcome != OutcomeUnknown {
		t.Fatalf("unknown status was collapsed: %+v err=%v", reconciled, err)
	}
}

func TestTamperedMutationHandleCannotBeReconciled(t *testing.T) {
	provider := newLedgerProvider()
	provider.commitErrorAfterApply = true
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	execution, _ := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-10",
		Payload:       []byte("12"),
	})
	tampered := execution.Handle
	tampered.ResultDigest = digest([]byte("999"))
	if _, err := executor.Reconcile(context.Background(), tampered); err == nil {
		t.Fatal("tampered handle was accepted")
	}
}

func TestCanceledInvocationDoesNotPrepareOrCommit(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	execution, err := executor.Invoke(ctx, MutationRequest{
		InvocationKey: "order-11",
		Payload:       []byte("13"),
	})
	if !errors.Is(err, context.Canceled) ||
		execution.Outcome != OutcomeNotCommitted ||
		provider.prepareCount != 0 ||
		provider.commitCount != 0 {
		t.Fatalf("canceled invocation crossed mutation boundary: execution=%+v err=%v provider=%+v", execution, err, provider)
	}
}

func TestPreparedResultCannotCommitAgainstStaleConcurrencyState(t *testing.T) {
	provider := newLedgerProvider()

	left, err := provider.Prepare(context.Background(), PrepareRequest{
		Protocol:           mutationProtocol,
		InvocationKey:      "concurrent-left",
		BindingFingerprint: testExecutionBinding("default"),
		RequestDigest:      digest([]byte("2")),
		Payload:            []byte("2"),
	})
	if err != nil {
		t.Fatal(err)
	}
	right, err := provider.Prepare(context.Background(), PrepareRequest{
		Protocol:           mutationProtocol,
		InvocationKey:      "concurrent-right",
		BindingFingerprint: testExecutionBinding("default"),
		RequestDigest:      digest([]byte("3")),
		Payload:            []byte("3"),
	})
	if err != nil {
		t.Fatal(err)
	}

	leftHandle, _, err := validatePrepared(
		"concurrent-left",
		testExecutionBinding("default"),
		digest([]byte("2")),
		left,
	)
	if err != nil {
		t.Fatal(err)
	}
	leftReceipt, err := provider.Commit(context.Background(), CommitRequest{
		Protocol:           mutationProtocol,
		PreparationID:      leftHandle.PreparationID,
		InvocationKey:      leftHandle.InvocationKey,
		BindingFingerprint: leftHandle.BindingFingerprint,
		RequestDigest:      leftHandle.RequestDigest,
		ResultDigest:       leftHandle.ResultDigest,
		ConcurrencyToken:   leftHandle.ConcurrencyToken,
	})
	if err != nil || leftReceipt.Outcome != OutcomeCommitted {
		t.Fatalf("first commit failed: receipt=%+v err=%v", leftReceipt, err)
	}

	rightHandle, _, err := validatePrepared(
		"concurrent-right",
		testExecutionBinding("default"),
		digest([]byte("3")),
		right,
	)
	if err != nil {
		t.Fatal(err)
	}
	rightReceipt, err := provider.Commit(context.Background(), CommitRequest{
		Protocol:           mutationProtocol,
		PreparationID:      rightHandle.PreparationID,
		InvocationKey:      rightHandle.InvocationKey,
		BindingFingerprint: rightHandle.BindingFingerprint,
		RequestDigest:      rightHandle.RequestDigest,
		ResultDigest:       rightHandle.ResultDigest,
		ConcurrencyToken:   rightHandle.ConcurrencyToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rightReceipt.Outcome != OutcomeNotCommitted ||
		provider.value != 2 ||
		provider.effectCount != 1 {
		t.Fatalf(
			"stale prepared result committed against changed state: receipt=%+v provider=%+v",
			rightReceipt,
			provider,
		)
	}
}

func TestUnknownCommitCanBeRecoveredFromJournalAfterExecutorRestart(t *testing.T) {
	provider := newLedgerProvider()
	provider.commitErrorAfterApply = true
	journal := newMemoryJournal()
	firstExecutor := newExecutor(provider, journal)

	execution, err := firstExecutor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-restart",
		Payload:       []byte("14"),
	})
	if err == nil || !errors.Is(err, ErrCommitUnknown) ||
		execution.Outcome != OutcomeUnknown {
		t.Fatalf("expected unknown after synthetic transport loss: execution=%+v err=%v", execution, err)
	}

	provider.commitErrorAfterApply = false
	restartedExecutor := newExecutor(provider, journal)
	reconciled, err := restartedExecutor.ReconcileByInvocationKey(
		context.Background(),
		"order-restart",
	)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Outcome != OutcomeCommitted ||
		len(reconciled.Result) != 0 ||
		reconciled.Receipt == nil ||
		provider.effectCount != 1 {
		t.Fatalf("restart recovery failed: reconciled=%+v provider=%+v", reconciled, provider)
	}
}

func TestReconcileByInvocationKeyUsesJournaledOutcomeWithoutProviderReplay(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	first, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-journaled",
		Payload:       []byte("15"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != OutcomeCommitted {
		t.Fatalf("first execution=%+v", first)
	}
	beforeCommits := provider.commitCount

	restarted := newExecutor(provider, journal)
	reconciled, err := restarted.ReconcileByInvocationKey(
		context.Background(),
		"order-journaled",
	)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Outcome != OutcomeCommitted ||
		len(reconciled.Result) != 0 ||
		reconciled.Receipt == nil ||
		reconciled.Receipt.ReceiptFingerprint != first.Receipt.ReceiptFingerprint {
		t.Fatalf("journaled outcome was not recovered: %+v", reconciled)
	}
	if provider.commitCount != beforeCommits {
		t.Fatalf("journaled outcome triggered provider commit replay: before=%d after=%d", beforeCommits, provider.commitCount)
	}
}

func TestCancellationAfterPrepareUsesNonCanceledCleanupContext(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)
	ctx, cancel := context.WithCancel(context.Background())
	executor.Validate = func([]byte) error {
		cancel()
		return nil
	}

	execution, err := executor.Invoke(ctx, MutationRequest{
		InvocationKey: "order-cancel-after-prepare",
		Payload:       []byte("16"),
	})
	if !errors.Is(err, context.Canceled) ||
		execution.Outcome != OutcomeNotCommitted {
		t.Fatalf("post-prepare cancellation outcome=%+v err=%v", execution, err)
	}
	if provider.commitCount != 0 ||
		provider.effectCount != 0 ||
		provider.abortCount != 1 {
		t.Fatalf("post-prepare cancellation crossed commit boundary: %+v", provider)
	}
	if provider.abortSawCanceledContext {
		t.Fatal("Abort cleanup received the canceled request context")
	}
}

func TestDurableMutationHandleDoesNotContainPreparedResultPayload(t *testing.T) {
	prepared := NewPreparedMutation(
		"prep-sensitive",
		"sensitive-invocation",
		testExecutionBinding("sensitive"),
		digest([]byte("request")),
		[]byte("sensitive-result-payload"),
		"version-1",
	)
	handle := preparedHandle(prepared)
	rendered := fmt.Sprintf("%+v", handle)
	if strings.Contains(rendered, "sensitive-result-payload") {
		t.Fatalf("durable handle leaked prepared result payload: %s", rendered)
	}
	if handle.ResultDigest != digest(prepared.ResultPayload) {
		t.Fatalf("durable handle lost result identity: %+v", handle)
	}
}

func TestPreparedFingerprintDoesNotCollideAcrossFieldBoundaries(t *testing.T) {
	common := PreparedMutation{
		Protocol:           mutationProtocol,
		BindingFingerprint: testExecutionBinding("collision"),
		RequestDigest:      digest([]byte("request")),
		ResultDigest:       digest([]byte("result")),
		ConcurrencyToken:   "version-1",
	}
	left := common
	left.PreparationID = "a\x00b"
	left.InvocationKey = "c"

	right := common
	right.PreparationID = "a"
	right.InvocationKey = "b\x00c"

	if preparedFingerprint(left) == preparedFingerprint(right) {
		t.Fatalf(
			"structured fingerprint collapsed distinct field boundaries: left=%q right=%q",
			left.PreparationID,
			right.InvocationKey,
		)
	}
}

func TestAbortCleanupFailureIsVisibleWithoutChangingBusinessOutcome(t *testing.T) {
	provider := newLedgerProvider()
	provider.abortError = true
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)
	executor.Validate = func([]byte) error {
		return fmt.Errorf("shape mismatch")
	}

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-abort-error",
		Payload:       []byte("17"),
	})
	if execution.Outcome != OutcomeNotCommitted ||
		!errors.Is(err, ErrResultValidation) ||
		!errors.Is(err, ErrAbortCleanup) {
		t.Fatalf("cleanup failure was hidden or changed business outcome: execution=%+v err=%v", execution, err)
	}
	if provider.commitCount != 0 || provider.effectCount != 0 {
		t.Fatalf("cleanup failure crossed commit boundary: %+v", provider)
	}
}

func TestReconcileRejectsExecutionBindingDriftBeforeProviderStatus(t *testing.T) {
	provider := newLedgerProvider()
	provider.commitErrorAfterApply = true
	journal := newMemoryJournal()
	first := newExecutorWithBinding(
		provider,
		journal,
		testExecutionBinding("binding-a"),
	)

	execution, err := first.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-binding-drift",
		Payload:       []byte("18"),
	})
	if err == nil || !errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("expected unknown first execution: execution=%+v err=%v", execution, err)
	}
	beforeStatus := provider.statusCount

	restarted := newExecutorWithBinding(
		provider,
		journal,
		testExecutionBinding("binding-b"),
	)
	_, err = restarted.ReconcileByInvocationKey(
		context.Background(),
		"order-binding-drift",
	)
	if err == nil || !strings.Contains(err.Error(), "execution binding") {
		t.Fatalf("binding drift was accepted: %v", err)
	}
	if provider.statusCount != beforeStatus {
		t.Fatalf(
			"binding drift reached Provider.Status: before=%d after=%d",
			beforeStatus,
			provider.statusCount,
		)
	}
}

func TestJournaledOutcomeIsRevalidatedBeforeReplay(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutor(provider, journal)

	_, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-tampered-journal",
		Payload:       []byte("19"),
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt := journal.outcomes["order-tampered-journal"]
	receipt.CommitID = "tampered-commit"
	journal.outcomes["order-tampered-journal"] = receipt

	beforeStatus := provider.statusCount
	_, err = executor.ReconcileByInvocationKey(
		context.Background(),
		"order-tampered-journal",
	)
	if err == nil || !errors.Is(err, ErrJournalCorrupt) {
		t.Fatalf("tampered Journal outcome was accepted: %v", err)
	}
	if provider.statusCount != beforeStatus {
		t.Fatalf(
			"tampered Journal outcome triggered Provider.Status: before=%d after=%d",
			beforeStatus,
			provider.statusCount,
		)
	}
}

func TestExecutorRequiresExactExecutionBinding(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	executor := newExecutorWithBinding(provider, journal, "not-a-digest")

	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "order-invalid-binding",
		Payload:       []byte("20"),
	})
	if err == nil || !strings.Contains(err.Error(), "execution binding fingerprint") {
		t.Fatalf("invalid execution binding was accepted: execution=%+v err=%v", execution, err)
	}
	if provider.prepareCount != 0 || provider.commitCount != 0 {
		t.Fatalf("invalid execution binding reached Provider: %+v", provider)
	}
}

func TestCommitIntentJournalRejectsDifferentBindingForSameInvocationKey(t *testing.T) {
	journal := newMemoryJournal()
	firstBinding := testExecutionBinding("journal-binding-a")
	secondBinding := testExecutionBinding("journal-binding-b")

	oldPrepared := NewPreparedMutation(
		"prep-shared-key",
		"shared-key",
		firstBinding,
		digest([]byte("21")),
		[]byte("21"),
		"version-0",
	)
	if err := journal.RecordCommitIntent(
		context.Background(),
		preparedHandle(oldPrepared),
	); err != nil {
		t.Fatal(err)
	}

	provider := newLedgerProvider()
	executor := newExecutorWithBinding(provider, journal, secondBinding)
	execution, err := executor.Invoke(context.Background(), MutationRequest{
		InvocationKey: "shared-key",
		Payload:       []byte("21"),
	})
	if err == nil || !errors.Is(err, ErrJournalWrite) ||
		!strings.Contains(err.Error(), "commit intent conflict") {
		t.Fatalf("conflicting durable intent was overwritten: execution=%+v err=%v", execution, err)
	}
	if execution.Outcome != OutcomeNotCommitted ||
		provider.commitCount != 0 ||
		provider.effectCount != 0 ||
		provider.abortCount != 1 {
		t.Fatalf("journal conflict crossed commit boundary: execution=%+v provider=%+v", execution, provider)
	}
}

func TestCancellationAfterDurableCommitIntentDoesNotAbandonDecision(t *testing.T) {
	provider := newLedgerProvider()
	journal := newMemoryJournal()
	ctx, cancel := context.WithCancel(context.Background())
	journal.onRecordIntent = cancel
	executor := newExecutor(provider, journal)

	execution, err := executor.Invoke(ctx, MutationRequest{
		InvocationKey: "order-cancel-after-intent",
		Payload:       []byte("22"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Outcome != OutcomeCommitted ||
		provider.value != 22 ||
		provider.effectCount != 1 ||
		provider.commitCount != 1 {
		t.Fatalf("durable commit intent was abandoned after caller cancellation: execution=%+v provider=%+v", execution, provider)
	}
	if provider.commitSawCanceledContext {
		t.Fatal("commit coordinator received the canceled caller context after durable intent")
	}
	if provider.abortCount != 0 {
		t.Fatalf("durable commit intent was incorrectly aborted: %d", provider.abortCount)
	}
}
