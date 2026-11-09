package postgres

import (
	"context"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/commandadapter"
	"tei/internal/semantic"
)

func adapterWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "orders",
			Capabilities: []semantic.Capability{{
				ID:       "order.save",
				Symbol:   "save",
				Outcomes: []semantic.Outcome{{Name: "saved"}},
				Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "orders.order"},
			}},
		}},
		Applications: []application.Application{{
			ID: "orders",
			Patterns: []application.Pattern{{
				ID:         "orders.save",
				Kind:       application.PatternCommand,
				Capability: "order.save",
			}},
		}},
	}
}

func TestCommandAdapterCommitsThroughPostgres(t *testing.T) {
	fixture := openFixture(t)
	grant := fixture.grant(t)
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      adapterWorld(),
		Executor: commandadapter.Executor{
			Runner:          effectcommit.Runner{Provider: fixture.provider},
			Grant:           grant,
			OperationID:     fixture.req.OperationID,
			ExpectedVersion: fixture.req.ExpectedVersion,
			Payload:         fixture.req.Payload,
			Effect:          &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "orders.order"},
		},
	}
	if _, err := invoker.Invoke(context.Background(), command.Request{
		Application: "orders",
		Pattern:     "orders.save",
		Capability:  "order.save",
	}); err != nil {
		t.Fatal(err)
	}
}
