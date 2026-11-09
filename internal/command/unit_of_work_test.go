package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/semantic"
)

type fakeTransaction struct {
	commits   int
	rollbacks int
	commitErr error
}

func (t *fakeTransaction) Commit(context.Context) error {
	t.commits++
	return t.commitErr
}

func (t *fakeTransaction) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type fakeUnitOfWork struct {
	transaction *fakeTransaction
	beginErr    error
}

func (u fakeUnitOfWork) Begin(context.Context) (Transaction, error) {
	if u.beginErr != nil {
		return nil, u.beginErr
	}
	return u.transaction, nil
}

func TestInvokeCommitsOnSuccess(t *testing.T) {
	transaction := &fakeTransaction{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		UnitOfWork: fakeUnitOfWork{transaction: transaction},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if transaction.commits != 1 || transaction.rollbacks != 0 {
		t.Fatalf("commits=%d rollbacks=%d", transaction.commits, transaction.rollbacks)
	}
}

func TestInvokeRollsBackOnExecutionFailure(t *testing.T) {
	transaction := &fakeTransaction{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{err: errors.New("connection lost")},
		UnitOfWork: fakeUnitOfWork{transaction: transaction},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("code=%s", code)
	}
	if transaction.commits != 0 || transaction.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", transaction.commits, transaction.rollbacks)
	}
}

func TestInvokeRollsBackOnUndeclaredOutcome(t *testing.T) {
	transaction := &fakeTransaction{}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "unknown", Payload: "sales.order"}}},
		UnitOfWork: fakeUnitOfWork{transaction: transaction},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_OUTCOME" {
		t.Fatalf("code=%s", code)
	}
	if transaction.commits != 0 || transaction.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", transaction.commits, transaction.rollbacks)
	}
}

func TestInvokeRejectsTransactionBeginFailure(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{},
		UnitOfWork: fakeUnitOfWork{beginErr: errors.New("cannot begin")},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_TRANSACTION_BEGIN_FAILED" {
		t.Fatalf("code=%s", code)
	}
}

func TestInvokeRollsBackOnCommitFailure(t *testing.T) {
	transaction := &fakeTransaction{commitErr: errors.New("commit failed")}
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"}}},
		UnitOfWork: fakeUnitOfWork{transaction: transaction},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_COMMIT_FAILED" {
		t.Fatalf("code=%s", code)
	}
	if transaction.commits != 1 || transaction.rollbacks != 1 {
		t.Fatalf("commits=%d rollbacks=%d", transaction.commits, transaction.rollbacks)
	}
}
