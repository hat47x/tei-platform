package commandadapter

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"tei/internal/command"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
)

// These are characterization tests, NOT an acceptance test for safe writes.
// The independent expectation is ORD-1 in the Command and ORD-2 in the
// explicitly supplied payload. The test records whether the existing adapter
// distinguishes them; a passing test does not authorize that mismatch.
type characterizationProvider struct {
	transaction *characterizationTransaction
}

func (p *characterizationProvider) Begin(context.Context) (effectcommit.Transaction, error) {
	p.transaction = &characterizationTransaction{}
	return p.transaction, nil
}

type characterizationTransaction struct {
	request   effectcommit.Request
	writes    int
	saved     int
	commits   int
	rollbacks int
}

func (*characterizationTransaction) CheckAuthority(context.Context, effectcommit.Scope, legacyauthorization.AuthorityEpochs) (bool, error) {
	return true, nil
}

func (*characterizationTransaction) FindReceipt(context.Context, string) (effectcommit.Receipt, bool, error) {
	return effectcommit.Receipt{}, false, nil
}

func (t *characterizationTransaction) WriteResource(_ context.Context, req effectcommit.Request) (uint64, error) {
	t.request = req
	t.writes++
	return req.ExpectedVersion + 1, nil
}

func (t *characterizationTransaction) SaveReceipt(context.Context, effectcommit.Receipt) error {
	t.saved++
	return nil
}

func (t *characterizationTransaction) Commit(context.Context) error {
	t.commits++
	return nil
}

// The runner's deferred rollback cannot undo the fake transaction's committed state.
func (t *characterizationTransaction) Rollback() error {
	t.rollbacks++
	return nil
}

func TestCharacterizationExplicitPayloadDiffersFromCommandInput(t *testing.T) {
	provider := &characterizationProvider{}
	commandRequest := request()
	// This is deliberately NOT the input the caller selected.
	explicitPayload := []byte(`{"order":"ORD-2"}`)
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: provider},
			Grant:       grant(),
			OperationID: "op-characterization-target",
			Payload:     explicitPayload,
		},
	}
	if _, err := invoker.Invoke(context.Background(), commandRequest); err != nil {
		t.Fatalf("adapter execution: %v", err)
	}
	tx := provider.transaction
	if tx == nil || tx.writes != 1 || tx.commits != 1 || tx.saved != 1 {
		t.Fatalf("expected one simulated write/receipt/commit: %+v", tx)
	}
	var actual map[string]string
	if err := json.Unmarshal(tx.request.Payload, &actual); err != nil {
		t.Fatal(err)
	}
	if commandRequest.Inputs["order"] != "ORD-1" || actual["order"] != "ORD-2" {
		t.Fatalf("fixture drift: selected=%q effectivePayload=%q", commandRequest.Inputs["order"], actual["order"])
	}
	if actual["order"] == commandRequest.Inputs["order"] {
		t.Fatal("fixture no longer demonstrates a target mismatch")
	}
	// At present there is no check binding explicit Payload to the selected
	// Command target. This result is a detected gap, not a safety PASS.
}

type failingOuterUnitOfWork struct {
	commits   int
	rollbacks int
}

func (u *failingOuterUnitOfWork) Begin(context.Context) (command.Transaction, error) {
	return u, nil
}

func (u *failingOuterUnitOfWork) Commit(context.Context) error {
	u.commits++
	return errors.New("outer commit failed after inner effectcommit")
}

func (u *failingOuterUnitOfWork) Rollback(context.Context) error {
	u.rollbacks++
	return nil
}

func TestCharacterizationInnerCommitCanPrecedeOuterCommitFailure(t *testing.T) {
	provider := &characterizationProvider{}
	outer := &failingOuterUnitOfWork{}
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: provider},
			Grant:       grant(),
			OperationID: "op-characterization-double-tx",
		},
		UnitOfWork: outer,
	}
	_, err := invoker.Invoke(context.Background(), request())
	var executionErr *command.ExecutionError
	if !errors.As(err, &executionErr) || executionErr.Code != "COMMAND_COMMIT_FAILED" {
		t.Fatalf("expected outer commit error, got %T %v", err, err)
	}
	if outer.commits != 1 || outer.rollbacks != 1 {
		t.Fatalf("outer unit of work calls: commits=%d rollbacks=%d", outer.commits, outer.rollbacks)
	}
	tx := provider.transaction
	if tx == nil || tx.writes != 1 || tx.commits != 1 || tx.saved != 1 {
		t.Fatalf("inner effectcommit did not commit before outer failure: %+v", tx)
	}
	// The outer rollback cannot undo this simulated inner commit. The test
	// does NOT prove any real DB durability, fencing, or receipt behavior.
}
