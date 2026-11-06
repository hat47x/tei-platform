package command

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func commandWorld() ir.World {
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

type fakeExecutor struct {
	result Result
	err    error
}

func (f fakeExecutor) Execute(context.Context, Request) (Result, error) {
	return f.result, f.err
}

func request() Request {
	return Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1"},
	}
}

func executionCode(t *testing.T, err error) string {
	t.Helper()
	var execution *ExecutionError
	if !errors.As(err, &execution) {
		t.Fatalf("error is not ExecutionError: %T %v", err, err)
	}
	return execution.Code
}

func TestInvokeReturnsDeclaredOutcome(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor: fakeExecutor{result: Result{
			Outcome: &semantic.Outcome{Name: "submitted", Payload: "sales.order"},
			Effect:  &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
		}},
	}
	result, err := invoker.Invoke(context.Background(), request())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("result=%+v", result)
	}
}

func TestInvokeRejectsIncompleteInput(t *testing.T) {
	invoker := Invoker{World: commandWorld(), Authorizer: ReferenceAuthorizer{}, Executor: fakeExecutor{}}
	invalid := request()
	invalid.Inputs = nil
	_, err := invoker.Invoke(context.Background(), invalid)
	if code := executionCode(t, err); code != "INVALID_COMMAND_INPUT" {
		t.Fatalf("code=%s err=%v", code, err)
	}
}

func TestInvokeRequiresExecutor(t *testing.T) {
	invoker := Invoker{Authorizer: ReferenceAuthorizer{}, World: commandWorld()}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_EXECUTOR_UNAVAILABLE" {
		t.Fatalf("code=%s err=%v", code, err)
	}
}

func TestInvokeSeparatesExecutionFailure(t *testing.T) {
	invoker := Invoker{World: commandWorld(), Authorizer: ReferenceAuthorizer{}, Executor: fakeExecutor{err: errors.New("connection lost")}}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "COMMAND_EXECUTION_FAILED" {
		t.Fatalf("code=%s err=%v", code, err)
	}
}

func TestInvokeRejectsUndeclaredOutcome(t *testing.T) {
	invoker := Invoker{
		Authorizer: ReferenceAuthorizer{},
		World:      commandWorld(),
		Executor:   fakeExecutor{result: Result{Outcome: &semantic.Outcome{Name: "unknown", Payload: "sales.order"}}},
	}
	_, err := invoker.Invoke(context.Background(), request())
	if code := executionCode(t, err); code != "UNDECLARED_COMMAND_OUTCOME" {
		t.Fatalf("code=%s err=%v", code, err)
	}
}

func TestInvokeRejectsUnknownCommand(t *testing.T) {
	invoker := Invoker{World: commandWorld(), Authorizer: ReferenceAuthorizer{}, Executor: fakeExecutor{}}
	invalid := request()
	invalid.Capability = "sales.missing"
	_, err := invoker.Invoke(context.Background(), invalid)
	if code := executionCode(t, err); code != "COMMAND_NOT_DECLARED" {
		t.Fatalf("code=%s err=%v", code, err)
	}
}

// effectlessCommandWorld declares a capability with no DataEffect so a test
// can separate "nothing to compare" from "the executor could not say".
func effectlessCommandWorld() ir.World {
	world := commandWorld()
	world.Semantics[0].Capabilities[0].Effect = nil
	return world
}

func requestForEffectlessWorld() Request {
	return request()
}
