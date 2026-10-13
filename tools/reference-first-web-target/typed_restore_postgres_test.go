//go:build !tei_repair_only && !tei_frozen_world

package main

import (
	"context"
	"testing"

	"tei/internal/composition"
)

func TestPostgresScenarioTypedRestoreNeedsNewDatabaseAdmission(t *testing.T) {
	state := postgresTestState(t)
	ctx := context.Background()
	registry := composition.NewRegistry()
	const scope = "reference/order_entry/development"
	if _, err := registry.Apply(0, []composition.Provider{{ID: "order-read", Scope: scope, Offers: readServiceContract}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("order-read", scope)
	if err != nil {
		t.Fatal(err)
	}
	binding := composition.Binding{ContractID: readServiceContract.ID, ContractVersion: readServiceContract.Version, ProviderID: "order-read", ArtifactDigest: state.digest, Generation: 1}
	allowed := uint64(1)
	guard := func(b composition.Binding) error {
		if b.Generation != allowed {
			return composition.ErrSelectionMismatch
		}
		return nil
	}
	lifecycle := composition.Lifecycle{Prepare: func(context.Context, *composition.Scope) error { return nil }, Ready: func(context.Context, composition.Binding) error { return nil }}
	service, err := composition.NewService(ctx, registry, selected, binding, guard, (&postgresReadScenario{state: state}).Run, lifecycle)
	if err != nil {
		t.Fatal(err)
	}
	read := &composedReadScenario{service: service}
	input := readScenarioInput{From: postgresSubmitInput().From}
	before, err := read.Run(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := service.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := read.Run(ctx, input); err == nil {
		t.Fatal("withdrawn read ran")
	}
	allowed = 2
	binding.Generation = 2
	newState := &postgresScenarioState{db: state.db, world: state.world, digest: state.digest, fence: 2}
	if err := service.Restore(ctx, selected, 1, binding, (&postgresReadScenario{state: newState}).Run, lifecycle); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Withdraw(2); _ = service.Reap(ctx) })
	if _, err := read.Run(ctx, input); err == nil {
		t.Fatal("old database grant admitted new instance")
	}
	if _, err := state.db.ExecContext(ctx, "UPDATE public.tei_reference_execution_grants SET generation=2,fencing_token=2 WHERE provider_id='order-read'"); err != nil {
		t.Fatal(err)
	}
	after, err := read.Run(ctx, input)
	if err != nil || after.Composition.Generation != 2 || after.Composition.ArtifactDigest != state.digest || after.StorageRevision != before.StorageRevision || len(after.OrderLineRows) != len(before.OrderLineRows) {
		t.Fatalf("restored read: %+v %v", after, err)
	}
}
