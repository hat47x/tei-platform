package commandadapter

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
)

func adapterWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.order_id", TypeRef: "std.identifier"},
				},
			}},
			Capabilities: []semantic.Capability{{
				ID:       "sales.submit",
				Symbol:   "submit",
				Inputs:   []semantic.Parameter{{Name: "order", Type: "std.identifier"}},
				Outcomes: []semantic.Outcome{{Name: "submitted", Payload: "sales.order"}},
				Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Items: []application.InteractionItem{{Attribute: "sales.order.order_id"}},
					Commands: []application.InteractionCommand{{
						Capability: "sales.submit",
						Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.order_id"}},
					}},
				},
			}},
		}},
	}
}

type fakeProvider struct {
	allowed   bool
	conflict  bool
	commitErr error
}

func (p fakeProvider) Begin(context.Context) (effectcommit.Transaction, error) {
	return &fakeTransaction{provider: p}, nil
}

type fakeTransaction struct {
	provider fakeProvider
}

func (t *fakeTransaction) CheckAuthority(context.Context, effectcommit.Scope, legacyauthorization.AuthorityEpochs) (bool, error) {
	return t.provider.allowed, nil
}

func (t *fakeTransaction) FindReceipt(context.Context, string) (effectcommit.Receipt, bool, error) {
	return effectcommit.Receipt{}, false, nil
}

func (t *fakeTransaction) WriteResource(_ context.Context, request effectcommit.Request) (uint64, error) {
	if t.provider.conflict {
		return 0, effectcommit.ErrResourceConflict
	}
	return request.ExpectedVersion + 1, nil
}

func (t *fakeTransaction) SaveReceipt(context.Context, effectcommit.Receipt) error { return nil }
func (t *fakeTransaction) Commit(context.Context) error {
	return t.provider.commitErr
}
func (t *fakeTransaction) Rollback() error { return nil }

func grant() legacyauthorization.AuthorizationGrant {
	return legacyauthorization.AuthorizationGrant{
		SubjectID: "user-1",
		Action:    "execute",
		Resource:  "sales.order",
		Epochs: &legacyauthorization.AuthorityEpochs{
			Source:      "test",
			SubjectLink: 1,
			Membership:  1,
			Policy:      1,
		},
	}
}

func request() command.Request {
	return command.Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1"},
	}
}

func TestExecutorCommitsThroughEffectCommit(t *testing.T) {
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: fakeProvider{allowed: true}},
			Grant:       grant(),
			OperationID: "op-1",
		},
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
}

func TestExecutorReportsBusinessDenial(t *testing.T) {
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: fakeProvider{allowed: false}},
			Grant:       grant(),
			OperationID: "op-1",
		},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if err == nil {
		t.Fatal("denied effect was accepted")
	}
	if !errors.Is(err, effectcommit.ErrBusinessDenied) {
		t.Fatalf("error does not wrap business denial: %v", err)
	}
}

// #2564: a lost commit response must reach the Command lifecycle as an
// unknown effect, not as a clean rejection, so the same attempt cannot replay.
func TestExecutorPreservesCommitUncertainty(t *testing.T) {
	lifecycle := command.NewLifecycle()
	invoker := command.LifecycleInvoker{
		Lifecycle: lifecycle,
		Invoker: command.Invoker{
			Authorizer: command.ReferenceAuthorizer{},
			World:      adapterWorld(),
			Executor: Executor{
				Runner:      effectcommit.Runner{Provider: fakeProvider{allowed: true, commitErr: errors.New("connection reset after send")}},
				Grant:       grant(),
				OperationID: "op-1",
			},
		},
	}
	_, err := invoker.Invoke(context.Background(), request(), command.DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	})
	if !command.EffectMayBeUnknown(err) {
		t.Fatalf("commit uncertainty was dropped: %v", err)
	}
	var execution *command.ExecutionError
	if !errors.As(err, &execution) || execution.Code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("error is not the command boundary error: %v", err)
	}
	var uncertain *effectcommit.CommitUncertainError
	if !errors.As(err, &uncertain) {
		t.Fatalf("CommitUncertainError is not preserved as the cause: %v", err)
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != command.StageUnknown {
		t.Fatalf("stage=%s, want unknown", attempt.Stage)
	}
	if lifecycle.MayReplay("cmd-1") {
		t.Fatal("an uncertain commit was reported as replayable")
	}
}

func TestExecutorReportsResourceConflict(t *testing.T) {
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: fakeProvider{allowed: true, conflict: true}},
			Grant:       grant(),
			OperationID: "op-1",
		},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if err == nil {
		t.Fatal("resource conflict was accepted")
	}
	if !errors.Is(err, effectcommit.ErrResourceConflict) {
		t.Fatalf("error does not wrap resource conflict: %v", err)
	}
}

// TestExecutorReportsTheCallerSuppliedDeclaredEffect checks that a committed
// effect can be reported to the Command boundary when the caller supplies the
// plan's declared effect. The Runner returns only a receipt, so the adapter
// must not invent the effect from the receipt.
func TestExecutorReportsTheCallerSuppliedDeclaredEffect(t *testing.T) {
	recorder := &command.MemoryRecorder{}
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: fakeProvider{allowed: true}},
			Grant:       grant(),
			OperationID: "op-1",
			Effect:      &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
		},
		Recorder: recorder,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Records) != 1 {
		t.Fatalf("records=%d", len(recorder.Records))
	}
	record := recorder.Records[0]
	if record.Reported == nil || record.Reported.Kind != semantic.DataEffectUpdate || record.Reported.Data != "sales.order" {
		t.Fatalf("reported=%+v", record.Reported)
	}
	if record.ReportedEffectMissing {
		t.Fatal("a supplied declared effect was recorded as missing")
	}
}

// TestExecutorWithoutEffectKeepsTheReceiptOnlyResult is the negative control:
// a nil Effect still reports nothing, and the Command record marks it missing.
func TestExecutorWithoutEffectKeepsTheReceiptOnlyResult(t *testing.T) {
	recorder := &command.MemoryRecorder{}
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: Executor{
			Runner:      effectcommit.Runner{Provider: fakeProvider{allowed: true}},
			Grant:       grant(),
			OperationID: "op-1",
		},
		Recorder: recorder,
	}
	if _, err := invoker.Invoke(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if !recorder.Records[0].ReportedEffectMissing {
		t.Fatal("receipt-only executor was not recorded as missing its declared effect")
	}
}
