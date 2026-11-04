package postgres

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/commandadapter"
	"tei/internal/semantic"
)

// #2564 product-side end-to-end: the product Command boundary (Invoker +
// Lifecycle) drives a real PostgreSQL effect-commit provider. The effectcommit
// Runner owns the transaction, so the product Invoker does not add a second
// UnitOfWork. Success, business conflict, and a lost commit response are
// observed as committed / rejected / unknown respectively.
func TestPostgresCommandLifecycleEndToEnd(t *testing.T) {
	f := openFixture(t)
	ctx := context.Background()
	// The fixture seeds authority/objects for its own "orders"/"order.save"
	// scope. Seed the Command scope instead, keeping the same authority
	// versions so the grant computed by the fixture remains valid.
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO " + f.schema + ".authority(application,capability,subject,action,resource,source,subject_link,membership,policy,enabled) " +
			"VALUES ('order_entry','sales.submit','editor','write','order-1','postgres-authority',1,1,1,TRUE)", nil},
		{"INSERT INTO " + f.schema + ".objects(application,resource,version,value) VALUES ('order_entry','order-1',0,'original')", nil},
	} {
		if _, err := f.pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	grant := f.grant(t)
	lifecycle := command.NewLifecycle()

	newInvoker := func(runner effectcommit.Runner, operation string, expected uint64) command.LifecycleInvoker {
		return command.LifecycleInvoker{
			Lifecycle: lifecycle,
			Invoker: command.Invoker{
				World: commandWorldE2E(),
				Executor: commandadapter.Executor{
					Runner:          runner,
					Grant:           grant,
					OperationID:     operation,
					ExpectedVersion: expected,
					Payload:         []byte("updated"),
				},
			},
		}
	}

	// 1. A committed CAS is observed as committed.
	success := newInvoker(effectcommit.Runner{Provider: f.provider}, "e2e-op-1", 0)
	if _, err := success.Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: "cmd-1", Owner: "owner-a", Generation: 1,
	}); err != nil {
		t.Fatal(err)
	}
	attempt, _ := lifecycle.Attempt("cmd-1")
	if attempt.Stage != command.StageObserved || attempt.Outcome != command.OutcomeCommitted {
		t.Fatalf("committed attempt=%+v", attempt)
	}
	if value, version := readE2EObject(t, f); value != "updated" || version != 1 {
		t.Fatalf("committed object=%q/%d", value, version)
	}

	// 2. A stale expected version is a business conflict, not an unknown effect.
	conflict := newInvoker(effectcommit.Runner{Provider: f.provider}, "e2e-op-2", 0)
	_, err := conflict.Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: "cmd-2", Owner: "owner-a", Generation: 1,
	})
	if !errors.Is(err, effectcommit.ErrResourceConflict) {
		t.Fatalf("conflict error=%v", err)
	}
	if command.EffectMayBeUnknown(err) {
		t.Fatal("a resource conflict was treated as an unknown effect")
	}
	attempt, _ = lifecycle.Attempt("cmd-2")
	if attempt.Stage != command.StageObserved || attempt.Outcome != command.OutcomeRejected {
		t.Fatalf("conflict attempt=%+v", attempt)
	}
	if value, version := readE2EObject(t, f); value != "updated" || version != 1 {
		t.Fatalf("conflict changed the object: %q/%d", value, version)
	}

	// 3. A lost commit response is unknown: the data is applied but the caller
	// cannot tell. The same attempt must not replay.
	uncertain := newInvoker(effectcommit.Runner{Provider: uncertainProvider{inner: f.provider}}, "e2e-op-3", 1)
	options := command.DispatchOptions{CommandID: "cmd-3", Owner: "owner-a", Generation: 1}
	_, err = uncertain.Invoke(ctx, commandRequestE2E(), options)
	if !command.EffectMayBeUnknown(err) {
		t.Fatalf("commit uncertainty was dropped: %v", err)
	}
	var commitUncertain *effectcommit.CommitUncertainError
	if !errors.As(err, &commitUncertain) {
		t.Fatalf("CommitUncertainError is not preserved: %v", err)
	}
	attempt, _ = lifecycle.Attempt("cmd-3")
	if attempt.Stage != command.StageUnknown {
		t.Fatalf("uncertain attempt=%+v", attempt)
	}
	if lifecycle.MayReplay("cmd-3") {
		t.Fatal("an uncertain commit was reported as replayable")
	}
	if _, err := uncertain.Invoke(ctx, commandRequestE2E(), options); err == nil {
		t.Fatal("the same attempt was replayed after an unknown outcome")
	}
	if attempt, _ := lifecycle.Attempt("cmd-3"); attempt.Stage != command.StageUnknown {
		t.Fatalf("refused replay changed the stage to %s", attempt.Stage)
	}
	// The uncertain transaction had committed on the server.
	if value, version := readE2EObject(t, f); value != "updated" || version != 2 {
		t.Fatalf("uncertain commit object=%q/%d", value, version)
	}

	// 4. A takeover with a higher generation re-observes via the operation
	// receipt. Replay is idempotent: the version does not move again.
	takeover := newInvoker(effectcommit.Runner{Provider: f.provider}, "e2e-op-3", 1)
	if _, err := takeover.Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: "cmd-3", Owner: "owner-b", Generation: 2, Takeover: true,
	}); err != nil {
		t.Fatal(err)
	}
	attempt, _ = lifecycle.Attempt("cmd-3")
	if attempt.Stage != command.StageObserved || attempt.Outcome != command.OutcomeCommitted ||
		attempt.Owner != "owner-b" || attempt.Generation != 2 || attempt.TakeoverFrom != 1 {
		t.Fatalf("takeover attempt=%+v", attempt)
	}
	if value, version := readE2EObject(t, f); value != "updated" || version != 2 {
		t.Fatalf("takeover duplicated the effect: %q/%d", value, version)
	}
}

func readE2EObject(t *testing.T, f *postgresFixture) (string, int64) {
	t.Helper()
	var value []byte
	var version int64
	err := f.pool.QueryRow(context.Background(),
		"SELECT value,version FROM "+f.schema+".objects WHERE application='order_entry' AND resource='order-1'").
		Scan(&value, &version)
	if err != nil {
		t.Fatal(err)
	}
	return string(value), version
}

func commandWorldE2E() ir.World {
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

func commandRequestE2E() command.Request {
	return command.Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1"},
	}
}
