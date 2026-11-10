package commandadapter

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tei/internal/command"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
)

// The real Runner keeps committed and replayed distinct. The current
// Command adapter returns an empty command.Result for both. This test records
// information loss, NOT an approved product-facing result contract.
type replayCharacterizationProvider struct {
	receipt   effectcommit.Receipt
	hasRecord bool
	writes    int
	saves     int
	commits   int
}

func (p *replayCharacterizationProvider) Begin(context.Context) (effectcommit.Transaction, error) {
	return &replayCharacterizationTx{provider: p}, nil
}

type replayCharacterizationTx struct {
	provider *replayCharacterizationProvider
}

func (*replayCharacterizationTx) CheckAuthority(context.Context, effectcommit.Scope, legacyauthorization.AuthorityEpochs) (bool, error) {
	return true, nil
}

func (tx *replayCharacterizationTx) FindReceipt(_ context.Context, operationID string) (effectcommit.Receipt, bool, error) {
	if tx.provider.hasRecord && tx.provider.receipt.OperationID == operationID {
		return tx.provider.receipt, true, nil
	}
	return effectcommit.Receipt{}, false, nil
}

func (tx *replayCharacterizationTx) WriteResource(_ context.Context, req effectcommit.Request) (uint64, error) {
	tx.provider.writes++
	return req.ExpectedVersion + 1, nil
}

func (tx *replayCharacterizationTx) SaveReceipt(_ context.Context, receipt effectcommit.Receipt) error {
	tx.provider.receipt = receipt
	tx.provider.hasRecord = true
	tx.provider.saves++
	return nil
}

func (tx *replayCharacterizationTx) Commit(context.Context) error {
	tx.provider.commits++
	return nil
}

func (*replayCharacterizationTx) Rollback() error { return nil }

func TestCharacterizationCommittedAndReplayedHaveIdenticalCommandResults(t *testing.T) {
	provider := &replayCharacterizationProvider{}
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: provider},
			Grant:       grant(),
			OperationID: "op-characterization-replay",
		},
	}
	first, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if !provider.hasRecord || provider.writes != 1 || provider.saves != 1 {
		t.Fatalf("expected one initial fake effect and receipt: %+v", provider)
	}
	second, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if provider.writes != 1 || provider.saves != 1 || provider.commits != 2 {
		t.Fatalf("same operation must replay without a second write: %+v", provider)
	}
	if !reflect.DeepEqual(first, second) || first.Outcome != nil || first.Effect != nil || len(first.Applied) != 0 {
		t.Fatalf("current adapter should return identical Command results without outcome, effect or applied operations, first=%+v second=%+v", first, second)
	}
	// An actual receipt was retained inside the fake Provider, but no status
	// or receipt from either Runner call survives in command.Result. A Command
	// consumer therefore cannot infer first-write vs replay from this value.
}

type uncertainCharacterizationProvider struct{}

func (uncertainCharacterizationProvider) Begin(context.Context) (effectcommit.Transaction, error) {
	return &uncertainCharacterizationTx{}, nil
}

type uncertainCharacterizationTx struct{}

func (*uncertainCharacterizationTx) CheckAuthority(context.Context, effectcommit.Scope, legacyauthorization.AuthorityEpochs) (bool, error) {
	return true, nil
}
func (*uncertainCharacterizationTx) FindReceipt(context.Context, string) (effectcommit.Receipt, bool, error) {
	return effectcommit.Receipt{}, false, nil
}
func (*uncertainCharacterizationTx) WriteResource(_ context.Context, req effectcommit.Request) (uint64, error) {
	return req.ExpectedVersion + 1, nil
}
func (*uncertainCharacterizationTx) SaveReceipt(context.Context, effectcommit.Receipt) error {
	return nil
}
func (*uncertainCharacterizationTx) Commit(context.Context) error {
	return errors.New("ack lost after commit request")
}
func (*uncertainCharacterizationTx) Rollback() error { return nil }

func TestCharacterizationCommitUnknownRetainsTypedCauseUnderGenericCommandError(t *testing.T) {
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: uncertainCharacterizationProvider{}},
			Grant:       grant(),
			OperationID: "op-characterization-unknown",
		},
	}
	_, err := invoker.Invoke(context.Background(), request())
	var executionErr *command.ExecutionError
	if !errors.As(err, &executionErr) || executionErr.Code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("expected generic Command execution error wrapper, got %T %v", err, err)
	}
	var uncertain *effectcommit.CommitUncertainError
	if !errors.As(err, &uncertain) {
		t.Fatalf("typed commit uncertainty should survive errors.As: %v", err)
	}
	// The UI/Host must not treat the generic Command code as proof of failure
	// or automatically retry under a new OperationID. This fake cannot tell
	// whether the database committed; recovery needs Runner.Lookup.
}
