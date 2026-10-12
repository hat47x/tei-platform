//go:build !tei_order_only

package main

import (
	"context"
	"fmt"

	"tei/internal/composition"
)

var repairLookupContract = scenarioContract("tei.reference.repair-intake.customer-lookup", "customerLookupInput -> customerLookupResult + error; existing Driver reference Pattern; read-only synthetic Oracle; invalid input remains typed error; no authentication or command authority")

type composedCustomerLookup struct {
	model   *scenarioModel
	service *composition.Service[customerLookupInput, customerLookupResult]
}

func (*composedCustomerLookup) ApplicationID() string { return "repair_intake" }
func (r *composedCustomerLookup) RunCustomerLookup(ctx context.Context, input customerLookupInput) (customerLookupResult, error) {
	if err := r.model.Check(ctx); err != nil {
		return customerLookupResult{}, err
	}
	result, binding, err := r.service.Invoke(ctx, input)
	if err == nil {
		result.Composition = projectCompositionBinding(binding)
		if r.model != nil {
			result.Composition.ModelRevision = r.model.revision
		}
	}
	return result, err
}

func wireComposedCustomerLookup(ctx context.Context, registry *composition.Registry, runner customerLookupRunner, model *scenarioModel, digest string, guard func(composition.Binding) error) (*composedCustomerLookup, error) {
	if registry == nil || runner == nil || guard == nil {
		return nil, fmt.Errorf("repair composition requires registry, runner and guard")
	}
	if err := requireApplicationMatch("composed customer lookup", runner, "repair_intake"); err != nil {
		return nil, err
	}
	scope := "reference/repair_intake/development"
	if _, err := registry.Apply(0, []composition.Provider{{ID: "repair-customer", Scope: scope, Offers: repairLookupContract}}, nil); err != nil {
		return nil, err
	}
	selected, err := registry.Select("repair-customer", scope)
	if err != nil {
		return nil, err
	}
	binding := composition.Binding{ContractID: repairLookupContract.ID, ContractVersion: repairLookupContract.Version, ProviderID: "repair-customer", ArtifactDigest: digest, Generation: 1}
	service, err := composition.NewService(ctx, registry, selected, binding, guard, runner.RunCustomerLookup, composition.Lifecycle{
		Prepare: func(context.Context, *composition.Scope) error { return nil },
		Ready:   func(context.Context, composition.Binding) error { return nil },
	})
	if err != nil {
		return nil, err
	}
	return &composedCustomerLookup{model: model, service: service}, nil
}
