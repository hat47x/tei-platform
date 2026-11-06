package legacyauthorization_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
)

// authorizationCommandWorld declares a Pattern that reaches a capability, so a
// denial can be shown to come from the rules and not from a missing declaration.
func authorizationCommandWorld() ir.World {
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

func authorizationCommandRequest() command.Request {
	return command.Request{
		Application: "order_entry",
		Pattern:     "order_entry.submit_order",
		Capability:  "sales.submit_order",
		Inputs:      map[string]string{"order": "ORD-1"},
	}
}

func authorizedSubject() legacyauthorization.SubjectStore {
	return authorizationStoreFunc(func(_ context.Context, subject string) (legacyauth.User, bool, error) {
		return legacyauth.User{SubjectID: subject, LoginID: "clerk", RoleCode: "sales"}, true, nil
	})
}

func commandAuthorizer(allowed bool, reason string) legacyauthorization.CommandAuthorizer {
	return legacyauthorization.CommandAuthorizer{
		Subject: "user-1",
		Store:   authorizedSubject(),
		Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
			return allowed, reason, nil
		}),
		Resource: func(command.Request) string { return "sales.order" },
	}
}

// TestCommandAuthorizerLetsAGrantedSubjectRun checks that a granted subject
// reaches the executor, so the denial test below is about the grant, not wiring.
func TestCommandAuthorizerLetsAGrantedSubjectRun(t *testing.T) {
	executor := &recordingExecutor{}
	invoker := command.Invoker{
		World:      authorizationCommandWorld(),
		Executor:   executor,
		UnitOfWork: &command.MemoryUnitOfWork{},
		Authorizer: commandAuthorizer(true, ""),
	}
	if _, err := invoker.Invoke(context.Background(), authorizationCommandRequest()); err != nil {
		t.Fatalf("allowed invoke err=%v", err)
	}
	if !executor.called {
		t.Fatal("allowed invoke did not reach the executor")
	}
}

// TestInvokerRejectsUngrantedCommandBeforeExecutor checks that the declared
// Pattern does not grant access. A subject without a grant is denied before the
// executor runs, even though the Pattern advertises the command.
func TestInvokerRejectsUngrantedCommandBeforeExecutor(t *testing.T) {
	executor := &recordingExecutor{}
	invoker := command.Invoker{
		World:      authorizationCommandWorld(),
		Executor:   executor,
		UnitOfWork: &command.MemoryUnitOfWork{},
		Authorizer: commandAuthorizer(false, "role_lacks_permission"),
	}
	_, err := invoker.Invoke(context.Background(), authorizationCommandRequest())
	var executionError *command.ExecutionError
	if !errors.As(err, &executionError) || executionError.Code != "COMMAND_UNAUTHORIZED" {
		t.Fatalf("err=%v", err)
	}
	if !errors.Is(err, legacyauthorization.ErrNoGrant) {
		t.Fatalf("denial is not reported as a grant decision: %v", err)
	}
	if executor.called {
		t.Fatal("unauthorized invoke reached the executor")
	}
}

// TestCommandAuthorizerStopsWhenRulesCannotBeEvaluated checks that a rule
// failure is not turned into a business denial.
func TestCommandAuthorizerStopsWhenRulesCannotBeEvaluated(t *testing.T) {
	authorizer := legacyauthorization.CommandAuthorizer{
		Subject: "user-1",
		Store:   authorizedSubject(),
		Rule: authorizationRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
			return false, "", errors.New("rule store unavailable")
		}),
		Resource: func(command.Request) string { return "sales.order" },
	}
	err := authorizer.Authorize(context.Background(), authorizationCommandRequest())
	if err == nil || errors.Is(err, legacyauthorization.ErrNoGrant) {
		t.Fatalf("rule failure was accepted as a denial: %v", err)
	}
}

type recordingExecutor struct {
	called bool
}

func (e *recordingExecutor) Execute(context.Context, command.Request) (command.Result, error) {
	e.called = true
	return command.Result{Effect: &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"}}, nil
}
