//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
	"tei/internal/security/provisionaleffectgrant"
)

// The Runner form of the order submit store must be refused at startup when the
// physical orders table does not match the accepted declaration (S02), like the
// direct store. The per-write check stays (see ...SchemaNotReadyRefusesBeforeWrite).

type startupGateSink struct{}

func (startupGateSink) Record(effectcommit.Evidence) {}

type startupGateStatus struct{}

func (startupGateStatus) ReadStatus(context.Context, int64) (string, error) { return "draft", nil }

type startupGateFakeRunner struct{}

func (startupGateFakeRunner) Execute(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	return effectcommit.Result{}, errors.New("must not be reached")
}

func (startupGateFakeRunner) Lookup(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	return effectcommit.Result{}, errors.New("must not be reached")
}

func TestRunnerOrderSubmitRealPostgresStartupSchemaGate(t *testing.T) {
	env := newRunnerPGEnv(t)
	ctx := context.Background()
	grant := func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
		return syntheticSubmitGrant(), nil
	}
	builders := map[string]func(t *testing.T) *runnerOrderSubmitStore{
		"standard": func(t *testing.T) *runnerOrderSubmitStore {
			s, err := newPostgresRunnerOrderSubmitStore(ctx, env.dsn, env.dsn, runnerTestSchema, env.world, grant)
			if err != nil {
				t.Fatal(err)
			}
			return s
		},
		"provisional": func(t *testing.T) *runnerOrderSubmitStore {
			s, err := newPostgresProvisionalRunnerOrderSubmitStore(ctx, env.dsn, env.dsn, runnerTestSchema, env.world,
				provisionaleffectgrant.FileSource{}, "order_entry", "legacy-order-editor", startupGateSink{})
			if err != nil {
				t.Fatal(err)
			}
			return s
		},
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			resetDeclaredOrders(t, env.db, env.world)
			store := build(t)
			t.Cleanup(func() { _ = store.Close() })

			// The declared table starts normally.
			got, err := requireOrderSubmitSchema(ctx, env.world, store)
			if err != nil || got != orderSubmitStore(store) {
				t.Fatalf("a matching table was refused: %v", err)
			}

			// A declared index is missing: startup is refused with the same code as
			// the direct store, and no DSN reaches the error.
			if _, err := env.db.Exec(`DROP INDEX public.orders_customer_idx`); err != nil {
				t.Fatal(err)
			}
			got, err = requireOrderSubmitSchema(ctx, env.world, store)
			if !errors.Is(err, errPhysicalSchemaNotReady) || got != nil {
				t.Fatalf("a table without a declared index started: %v", err)
			}
			if err.Error() != "PHYSICAL_SCHEMA_NOT_READY" || strings.Contains(err.Error(), env.dsn) {
				t.Fatalf("unexpected error text: %q", err.Error())
			}

			// Restored: starts again.
			resetDeclaredOrders(t, env.db, env.world)
			if _, err := requireOrderSubmitSchema(ctx, env.world, store); err != nil {
				t.Fatalf("restored table was refused: %v", err)
			}
		})
	}
}

// A Runner store with no schemaCheck (a fake wired by a test) has nothing to
// observe: the startup gate leaves it as it was.
func TestRunnerOrderSubmitStartupSchemaGateLeavesUnwiredStoreAlone(t *testing.T) {
	store, err := newRunnerOrderSubmitStore(startupGateFakeRunner{},
		func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
			return legacyauthorization.AuthorizationGrant{}, errors.New("unused")
		}, nil, startupGateStatus{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := requireOrderSubmitSchema(context.Background(), ir.World{}, store)
	if err != nil || got != orderSubmitStore(store) {
		t.Fatalf("unwired Runner store changed: %v", err)
	}
}
