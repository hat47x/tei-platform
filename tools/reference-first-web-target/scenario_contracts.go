package main

import "context"

// These JSON projections and minimal Runner interfaces form the existing
// combined Target's transport-facing contract. Move them out of concrete
// Application Runner sources so later compile-time entry points can select
// implementations without inventing new business meaning or runtime authority.
// Request credentials remain transient input; they are never persisted here.

type readScenarioInput struct {
	From     string
	RowIndex int
}

type readScenarioResult struct {
	ScenarioID       string                     `json:"scenarioId"`
	Environment      string                     `json:"environment"`
	ApplicationID    string                     `json:"applicationId"`
	From             string                     `json:"from"`
	SelectedRowIndex int                        `json:"selectedRowIndex"`
	BrowseRows       []semanticRowProjection    `json:"browseRows"`
	SelectedOrder    semanticRowProjection      `json:"selectedOrder"`
	CustomerRows     []semanticRowProjection    `json:"customerRows"`
	OrderLineRows    []semanticRowProjection    `json:"orderLineRows"`
	RuntimeBindings  []runtimeBindingProjection `json:"runtimeBindings"`
	AuthorityLimits  []string                   `json:"authorityLimits"`
	Unresolved       []string                   `json:"unresolved"`
}

type readScenarioRunner interface {
	Run(context.Context, readScenarioInput) (readScenarioResult, error)
}

type commandScenarioInput struct {
	LoginID    string
	Credential string
	From       string
	RowIndex   int
}

type commandScenarioResult struct {
	ScenarioID          string                       `json:"scenarioId"`
	ApplicationID       string                       `json:"applicationId"`
	Stage               string                       `json:"stage"`
	Authentication      *businessResultProjection    `json:"authentication,omitempty"`
	Read                *readScenarioResult          `json:"read,omitempty"`
	Validations         []validationResultProjection `json:"validations,omitempty"`
	Authorization       *businessResultProjection    `json:"authorization,omitempty"`
	EffectAuthorization *businessResultProjection    `json:"effectAuthorization,omitempty"`
	Command             *businessResultProjection    `json:"command,omitempty"`
	Observation         *observationProjection       `json:"observation,omitempty"`
	AuthorityLimits     []string                     `json:"authorityLimits"`
	Unresolved          []string                     `json:"unresolved"`
}

type validationResultProjection struct {
	Validation string `json:"validation"`
	Result     string `json:"result"`
}

type businessResultProjection struct {
	Capability string         `json:"capability"`
	Mode       string         `json:"mode"`
	Outcome    string         `json:"outcome,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type observationProjection struct {
	ReadIdentity string `json:"readIdentity"`
	Nature       string `json:"nature"`
	Result       any    `json:"result"`
}

type commandScenarioRunner interface {
	RunCommand(context.Context, commandScenarioInput) (commandScenarioResult, error)
}

type customerLookupInput struct {
	CustomerID string
}

type customerLookupResult struct {
	ScenarioID          string                     `json:"scenarioId"`
	Environment         string                     `json:"environment"`
	ApplicationID       string                     `json:"applicationId"`
	RequestedCustomerID string                     `json:"requestedCustomerId"`
	Intake              semanticRowProjection      `json:"intake"`
	CustomerRows        []semanticRowProjection    `json:"customerRows"`
	RuntimeBindings     []runtimeBindingProjection `json:"runtimeBindings"`
	AuthorityLimits     []string                   `json:"authorityLimits"`
	Unresolved          []string                   `json:"unresolved"`
}

type customerLookupRunner interface {
	RunCustomerLookup(context.Context, customerLookupInput) (customerLookupResult, error)
}
