package postgres

import (
	"context"
	"testing"

	"tei/internal/command"
	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/commandadapter"
	"tei/internal/semantic"
)

// TestPostgresCommandObservesACommittedReceiptByLookup runs the product Command
// boundary against the real PostgreSQL effect-commit provider and configures
// the adapter's independent Lookup observation. A committed receipt should
// become persistence:"observed"; the test then removes the receipt and checks
// that the same configuration reports "not-observed" instead of claiming the
// effect is visible.
func TestPostgresCommandObservesACommittedReceiptByLookup(t *testing.T) {
	f := openFixture(t)
	ctx := context.Background()
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

	newExecutor := func(operation string) commandadapter.Executor {
		return commandadapter.Executor{
			Runner:          effectcommit.Runner{Provider: f.provider},
			Grant:           grant,
			OperationID:     operation,
			ExpectedVersion: 0,
			Payload:         []byte("updated"),
			Effect:          &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			Lookup:          true,
		}
	}
	newInvoker := func(operation string) command.Invoker {
		executor := newExecutor(operation)
		return command.Invoker{
			Authorizer:          command.ReferenceAuthorizer{},
			World:               commandWorldE2E(),
			Executor:            executor,
			DurableStore:        true,
			PersistenceObserver: executor,
		}
	}

	// 1. Commit succeeds, then the independent receipt lookup observes it.
	result, err := newInvoker("observe-op-1").Invoke(ctx, commandRequestE2E())
	if err != nil {
		t.Fatal(err)
	}
	if result.Persistence != command.PersistenceObserved {
		t.Fatalf("persistence=%q, want observed", result.Persistence)
	}
	if value, version := readE2EObject(t, f); value != "updated" || version != 1 {
		t.Fatalf("committed object=%q/%d", value, version)
	}

	// 2. The lookup reads a receipt, not the resource's current value. Removing
	// the receipt makes the same configuration report not-observed even though
	// the object still holds the committed value.
	if _, err := f.pool.Exec(ctx, "DELETE FROM "+f.schema+".receipts WHERE operation_id='observe-op-1'"); err != nil {
		t.Fatal(err)
	}
	// Calling Execute again would hit a resource conflict (the object is at
	// version 1), so probe Observe directly: the observation, not the write, is
	// what the receipt removal invalidates.
	if err := newExecutor("observe-op-1").Observe(ctx, commandRequestE2E(), semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}); err == nil {
		t.Fatal("receipt lookup succeeded after the receipt was removed")
	}
	if value, version := readE2EObject(t, f); value != "updated" || version != 1 {
		t.Fatalf("receipt removal changed the object: %q/%d", value, version)
	}
}
