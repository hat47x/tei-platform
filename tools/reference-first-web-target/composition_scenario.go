//go:build !tei_repair_only

package main

import (
	"context"
	"fmt"

	"tei/internal/composition"
)

var (
	readServiceContract           = scenarioContract("tei.reference.order-entry.read", "readScenarioInput -> readScenarioResult + error; read-only; RFC3339 from; nonnegative row; Driver browse/record/customer/header-detail")
	submitServiceContract         = scenarioContract("tei.reference.order-entry.submit", "commandScenarioInput -> commandScenarioResult + error; authentication/validation/authorization/command/separate-observation; synthetic state; credentials transient")
	postgresSubmitServiceContract = scenarioContract("tei.reference.order-entry.postgres-submit", "commandScenarioInput -> commandScenarioResult + error; current grant and fencing; canonical effect authorization; persistent PostgreSQL draft-to-submitted transition; separate external observation; unknown commit forbids automatic retry; credentials transient")
)

type composedReadScenario struct {
	model   *scenarioModel
	service *composition.Service[readScenarioInput, readScenarioResult]
}

func (*composedReadScenario) ApplicationID() string { return "order_entry" }
func (r *composedReadScenario) Run(ctx context.Context, input readScenarioInput) (readScenarioResult, error) {
	if err := r.model.Check(ctx); err != nil {
		return readScenarioResult{}, err
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

type composedCommandScenario struct {
	model   *scenarioModel
	service *composition.Service[commandScenarioInput, commandScenarioResult]
}

func (*composedCommandScenario) ApplicationID() string { return "order_entry" }
func (r *composedCommandScenario) RunCommand(ctx context.Context, input commandScenarioInput) (commandScenarioResult, error) {
	if err := r.model.Check(ctx); err != nil {
		return commandScenarioResult{}, err
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

// wireComposedScenarios is a trusted host assembly point. The caller supplies
// current instance permission; business authorization remains in the Driver.
// It does not turn the reference executors into real external providers.
func wireComposedScenarios(ctx context.Context, registry *composition.Registry, read readScenarioRunner, command *referenceCommandScenarioRunner, digest string, guard func(composition.Binding) error) (*composedReadScenario, *composedCommandScenario, error) {
	if read == nil || command == nil || registry == nil || guard == nil {
		return nil, nil, fmt.Errorf("reference composition requires runners, registry and host guard")
	}
	if err := requireApplicationMatch("composed read", read, "order_entry"); err != nil {
		return nil, nil, err
	}
	scope := "reference/order_entry/development"
	submitContract := submitServiceContract
	if command.persistence != nil {
		submitContract = postgresSubmitServiceContract
	}
	if _, err := registry.Apply(0, []composition.Provider{
		{ID: "order-read", Scope: scope, Offers: readServiceContract},
		{ID: "order-submit", Scope: scope, Offers: submitContract, Requires: []composition.Dependency{{Contract: readServiceContract, ProviderID: "order-read"}}},
	}, nil); err != nil {
		return nil, nil, err
	}
	lifecycle := composition.Lifecycle{
		Prepare: func(context.Context, *composition.Scope) error { return nil },
		Ready:   func(context.Context, composition.Binding) error { return nil },
	}
	selectedRead, err := registry.Select("order-read", scope)
	if err != nil {
		return nil, nil, err
	}
	readBinding := composition.Binding{ContractID: readServiceContract.ID, ContractVersion: "1", ProviderID: "order-read", ArtifactDigest: digest, Generation: 1}
	readService, err := composition.NewService(ctx, registry, selectedRead, readBinding, guard, read.Run, lifecycle)
	if err != nil {
		return nil, nil, err
	}
	cleanupRead := func(err error) (*composedReadScenario, *composedCommandScenario, error) {
		_ = readService.Withdraw(1)
		_ = readService.Reap(context.Background())
		return nil, nil, err
	}
	selectedCommand, err := registry.Select("order-submit", scope)
	if err != nil {
		return cleanupRead(err)
	}
	commandBinding := composition.Binding{ContractID: submitContract.ID, ContractVersion: "1", ProviderID: "order-submit", ArtifactDigest: digest, Generation: 1}
	// Bind the internal read to the same selected read service used by HTTP.
	// Copy the runner to preserve the caller's existing assembly.
	composedRead := &composedReadScenario{service: readService, model: command.model}
	linkedCommand := *command
	linkedCommand.read = composedRead
	commandService, err := composition.NewService(ctx, registry, selectedCommand, commandBinding, guard, linkedCommand.RunCommand, lifecycle)
	if err != nil {
		return cleanupRead(err)
	}
	return composedRead, &composedCommandScenario{service: commandService, model: command.model}, nil
}
