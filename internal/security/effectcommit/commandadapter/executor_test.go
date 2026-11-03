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
	allowed bool
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
	return request.ExpectedVersion + 1, nil
}

func (t *fakeTransaction) SaveReceipt(context.Context, effectcommit.Receipt) error { return nil }
func (t *fakeTransaction) Commit(context.Context) error                            { return nil }
func (t *fakeTransaction) Rollback() error                                         { return nil }

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
		World: adapterWorld(),
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
		World: adapterWorld(),
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
