package postgres

import (
	"context"
	"errors"
	"testing"

	"tei/internal/command"
	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/commandadapter"
)

// RCAS-04 (#2564) on the product path: the same operation identity with a
// different payload is a business conflict, not a second effect and not an
// unknown outcome. The Command adapter returns the conflict, the lifecycle
// observes a rejection, and the stored value and receipt count stay unchanged.
func TestPostgresCommandOperationIdentityRejectsDifferentPayload(t *testing.T) {
	f := openFixture(t)
	ctx := context.Background()
	// openFixture seeds its own "orders"/"order.save" scope; seed the Command
	// scope with the same authority versions so the fixture grant stays valid.
	for _, statement := range []string{
		"INSERT INTO " + f.schema + ".authority(application,capability,subject,action,resource,source,subject_link,membership,policy,enabled) " +
			"VALUES ('order_entry','sales.submit','editor','write','order-1','postgres-authority',1,1,1,TRUE)",
		"INSERT INTO " + f.schema + ".objects(application,resource,version,value) VALUES ('order_entry','order-1',0,'original')",
	} {
		if _, err := f.pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	grant := f.grant(t)
	lifecycle := command.NewLifecycle()

	newInvoker := func(payload string, expected uint64) command.LifecycleInvoker {
		return command.LifecycleInvoker{
			Lifecycle: lifecycle,
			Invoker: command.Invoker{
				World: commandWorldE2E(),
				Executor: commandadapter.Executor{
					Runner:          effectcommit.Runner{Provider: f.provider},
					Grant:           grant,
					OperationID:     "same-operation",
					ExpectedVersion: expected,
					Payload:         []byte(payload),
				},
			},
		}
	}

	first := newInvoker("updated", 0)
	if _, err := first.Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if value, version := readE2EObject(t, f); value != "updated" || version != 1 {
		t.Fatalf("first effect value=%q version=%d", value, version)
	}

	// A second logical command reuses the operation identity with different
	// content. The receipt binds the operation to the first request.
	second := newInvoker("different", 1)
	_, err := second.Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: "cmd-2", Owner: "owner-a", Generation: 1,
	})
	if !errors.Is(err, effectcommit.ErrResourceConflict) {
		t.Fatalf("second error=%v", err)
	}
	if command.EffectMayBeUnknown(err) {
		t.Fatal("an operation-identity conflict was treated as an unknown effect")
	}
	attempt, _ := lifecycle.Attempt("cmd-2")
	if attempt.Stage != command.StageObserved || attempt.Outcome != command.OutcomeRejected {
		t.Fatalf("second attempt=%+v", attempt)
	}

	if value, version := readE2EObject(t, f); value != "updated" || version != 1 {
		t.Fatalf("conflict changed the effect value=%q version=%d", value, version)
	}
	var receipts int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM `+f.schema+`.receipts`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 {
		t.Fatalf("receipts=%d, want 1", receipts)
	}
	t.Logf("operation identity rejected a different payload: stage=%s outcome=%s receipts=%d",
		attempt.Stage, attempt.Outcome, receipts)
}
