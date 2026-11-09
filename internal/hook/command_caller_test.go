package hook_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// commandCallerWorld declares one application with a command Pattern that
// reaches the capability a hook may call.
func commandCallerWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID:         "sales.order",
				Attributes: []semantic.Attribute{{ID: "sales.order.order_id", TypeRef: "std.identifier"}},
			}},
			Capabilities: []semantic.Capability{{
				ID:       "sales.submit_order",
				Symbol:   "submit_order",
				Inputs:   []semantic.Parameter{{Name: "order", Type: "std.identifier"}},
				Outcomes: []semantic.Outcome{{Name: "submitted", Payload: "sales.order"}},
				Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns: []application.Pattern{{
				ID:   "order_entry.submit_order",
				Kind: application.PatternCommand,
				Data: "sales.order",
				Interaction: &application.Interaction{Commands: []application.InteractionCommand{{
					Capability: "sales.submit_order",
					Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.order_id"}},
				}}},
			}},
		}},
	}
}

// commandCaller runs a hook's capability through the Command boundary with a
// reference executor, so the call inherits the invoker's rules.
func commandCaller() hook.CommandCaller {
	return hook.CommandCaller{
		Application: "order_entry",
		Invoker: command.Invoker{
			World:      commandCallerWorld(),
			Executor:   command.ReferenceExecutor{World: commandCallerWorld()},
			UnitOfWork: &command.MemoryUnitOfWork{},
			Authorizer: command.ReferenceAuthorizer{},
		},
		Pattern: func(capability core.CanonicalID) (core.CanonicalID, bool) {
			if capability == "sales.submit_order" {
				return "order_entry.submit_order", true
			}
			return "", false
		},
	}
}

// TestCommandCallerRunsThroughTheBoundary checks that a hook call reaches the
// Command boundary and returns its result, rather than touching data directly.
func TestCommandCallerRunsThroughTheBoundary(t *testing.T) {
	value, err := commandCaller().Call(context.Background(), "sales.submit_order", map[string]string{"order": "ORD-1"})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	result, ok := value.(command.Result)
	if !ok || result.Outcome == nil || result.Outcome.Name != "submitted" {
		t.Fatalf("value=%+v", value)
	}
}

// TestCommandCallerRequiresADeclaredPattern checks that a hook cannot invent a
// call target: a capability no Pattern declares is refused before the boundary.
func TestCommandCallerRequiresADeclaredPattern(t *testing.T) {
	_, err := commandCaller().Call(context.Background(), "sales.delete_order", nil)
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PATTERN_NOT_DECLARED" {
		t.Fatalf("err=%v", err)
	}
}

// TestCommandCallerKeepsTheBoundaryRefusal checks that the boundary's own
// refusals (input, authorization, effect) come through unchanged, so the hook
// does not report them as its own call failure.
func TestCommandCallerKeepsTheBoundaryRefusal(t *testing.T) {
	_, err := commandCaller().Call(context.Background(), "sales.submit_order", nil)
	var executionError *command.ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "INVALID_COMMAND_INPUT" {
		t.Fatalf("err=%v", err)
	}
}

// TestDeclaredCommandRefusesAnUndeclaredCapability checks that the hook's
// declaration is enforced before the Command boundary runs.
func TestDeclaredCommandRefusesAnUndeclaredCapability(t *testing.T) {
	caller := hook.DeclaredCommand{
		CommandCaller: commandCaller(),
		Hook:          &application.Hook{Phase: "before-create", Uses: []core.CanonicalID{"sales.submit_order"}},
	}
	_, err := caller.Call(context.Background(), "sales.delete_order", nil)
	var hookError *hook.Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_UNDECLARED_CALL" {
		t.Fatalf("err=%v", err)
	}
}

// TestDeclaredCommandRunsADeclaredCapability checks that a declared capability
// still runs through the boundary.
func TestDeclaredCommandRunsADeclaredCapability(t *testing.T) {
	caller := hook.DeclaredCommand{
		CommandCaller: commandCaller(),
		Hook:          &application.Hook{Phase: "before-create", Uses: []core.CanonicalID{"sales.submit_order"}},
	}
	if _, err := caller.Call(context.Background(), "sales.submit_order", map[string]string{"order": "ORD-1"}); err != nil {
		t.Fatalf("err=%v", err)
	}
}
