package main

// An Environment's concrete database-instance selection is an authored
// binding, not evidence of a live Provider, connection, or authorization.

import "tei/internal/ir"

type EnvironmentBindingObservation struct {
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	Environment      string `json:"environment"`
	Application      string `json:"application"`
	DatasourceAlias  string `json:"datasourceAlias,omitempty"`
	Instance         string `json:"instance,omitempty"`
	Definition       string `json:"definition,omitempty"`
	DatabaseProvider string `json:"databaseProvider,omitempty"`
}

// ObserveEnvironmentBinding consumes the existing Canonical Environment
// model. It requires a qualified Pattern's exact authored Contract dependency,
// and never uses an arbitrary instance selected by the caller.
func ObserveEnvironmentBinding(world ir.World, environmentID, applicationID string, required *DeclaredContractQuery) EnvironmentBindingObservation {
	out := EnvironmentBindingObservation{
		Status: "unresolved", Reason: "contract-query-requirement-not-established",
		Environment: environmentID, Application: applicationID,
	}
	if required == nil || required.Status != "declared-requirement" ||
		required.DatasourceAlias == "" || required.TargetDatabase == "" {
		return out
	}
	out.DatasourceAlias = required.DatasourceAlias

	envCount := 0
	var envIndex int
	for i := range world.Environments {
		if string(world.Environments[i].ID) == environmentID {
			envCount++
			envIndex = i
		}
	}
	if envCount != 1 {
		out.Reason = "environment-not-unique-or-missing"
		return out
	}
	env := world.Environments[envIndex]
	bindingCount := 0
	var instanceID string
	for _, b := range env.Bindings {
		if string(b.Application) == applicationID && string(b.Alias) == required.DatasourceAlias {
			bindingCount++
			instanceID = string(b.Instance)
		}
	}
	if bindingCount != 1 || instanceID == "" {
		out.Reason = "application-datasource-binding-not-unique-or-missing"
		return out
	}
	out.Instance = instanceID
	instanceCount := 0
	var definition, provider string
	for _, candidate := range env.Instances {
		if string(candidate.ID) == instanceID {
			instanceCount++
			definition = string(candidate.Definition)
			provider = string(candidate.Provider)
		}
	}
	if instanceCount != 1 {
		out.Reason = "database-instance-not-unique-or-missing"
		return out
	}
	out.Definition = definition
	if definition != required.TargetDatabase {
		out.Status = "incompatible"
		out.Reason = "database-instance-definition-mismatch"
		return out
	}
	if provider == "" {
		out.Reason = "database-provider-label-not-declared"
		return out
	}
	out.DatabaseProvider = provider
	out.Status = "declared-environment-binding"
	out.Reason = "matching-environment-binding-provider-conformance-not-verified"
	return out
}
