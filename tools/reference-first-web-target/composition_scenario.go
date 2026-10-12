//go:build !tei_repair_only

package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"tei/internal/composition"
)

// These contracts belong to this reference consumer, not to a kernel enum.
// Fingerprints describe the reference typed API, not semantic conformance.
func scenarioContract(id, shape string) composition.Contract {
	return composition.Contract{ID: id, Version: "1", Shape: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(shape)))}
}

var (
	readServiceContract   = scenarioContract("tei.reference.order-entry.read", "readScenarioInput -> readScenarioResult + error; read-only; RFC3339 from; nonnegative row; Driver browse/record/customer/header-detail")
	submitServiceContract = scenarioContract("tei.reference.order-entry.submit", "commandScenarioInput -> commandScenarioResult + error; authentication/validation/authorization/command/separate-observation; synthetic state; credentials transient")
)

func projectCompositionBinding(binding composition.Binding) *compositionTrace {
	return &compositionTrace{ContractID: binding.ContractID, ContractVersion: binding.ContractVersion, ProviderID: binding.ProviderID, ArtifactDigest: binding.ArtifactDigest, Generation: binding.Generation}
}

type composedReadScenario struct {
	service *composition.Service[readScenarioInput, readScenarioResult]
}

func (*composedReadScenario) ApplicationID() string { return "order_entry" }
func (r *composedReadScenario) Run(ctx context.Context, input readScenarioInput) (readScenarioResult, error) {
	result, binding, err := r.service.Invoke(ctx, input)
	if err == nil {
		result.Composition = projectCompositionBinding(binding)
	}
	return result, err
}

type composedCommandScenario struct {
	service *composition.Service[commandScenarioInput, commandScenarioResult]
}

func (*composedCommandScenario) ApplicationID() string { return "order_entry" }
func (r *composedCommandScenario) RunCommand(ctx context.Context, input commandScenarioInput) (commandScenarioResult, error) {
	result, binding, err := r.service.Invoke(ctx, input)
	if err == nil {
		result.Composition = projectCompositionBinding(binding)
	}
	return result, err
}

// The linked executable is the actual artifact for this reference. Hashing
// bytes proves identity only: it is neither publisher trust nor authorization.
func runningExecutableDigest() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// wireComposedScenarios is a trusted host assembly point. The caller supplies
// current instance permission; business authorization remains in the Driver.
// It does not turn the reference executors into real external providers.
func wireComposedScenarios(ctx context.Context, registry *composition.Registry, read *referenceReadScenarioRunner, command *referenceCommandScenarioRunner, digest string, guard func(composition.Binding) error) (*composedReadScenario, *composedCommandScenario, error) {
	if read == nil || command == nil || registry == nil || guard == nil {
		return nil, nil, fmt.Errorf("reference composition requires runners, registry and host guard")
	}
	scope := "reference/order_entry/development"
	if _, err := registry.Apply(0, []composition.Provider{
		{ID: "order-read", Scope: scope, Offers: readServiceContract},
		{ID: "order-submit", Scope: scope, Offers: submitServiceContract, Requires: []composition.Dependency{{Contract: readServiceContract, ProviderID: "order-read"}}},
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
	commandBinding := composition.Binding{ContractID: submitServiceContract.ID, ContractVersion: "1", ProviderID: "order-submit", ArtifactDigest: digest, Generation: 1}
	// Bind the internal read to the same selected read service used by HTTP.
	// Copy the runner to preserve the caller's existing assembly.
	composedRead := &composedReadScenario{service: readService}
	linkedCommand := *command
	linkedCommand.read = composedRead
	commandService, err := composition.NewService(ctx, registry, selectedCommand, commandBinding, guard, linkedCommand.RunCommand, lifecycle)
	if err != nil {
		return cleanupRead(err)
	}
	return composedRead, &composedCommandScenario{service: commandService}, nil
}
