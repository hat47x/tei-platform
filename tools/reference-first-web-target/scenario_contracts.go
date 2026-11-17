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
	ScenarioID       string                  `json:"scenarioId"`
	Environment      string                  `json:"environment"`
	ApplicationID    string                  `json:"applicationId"`
	From             string                  `json:"from"`
	SelectedRowIndex int                     `json:"selectedRowIndex"`
	BrowseRows       []semanticRowProjection `json:"browseRows"`
	SelectedOrder    semanticRowProjection   `json:"selectedOrder"`
	CustomerRows     []semanticRowProjection `json:"customerRows"`
	OrderLineRows    []semanticRowProjection `json:"orderLineRows"`
	// RowAnswers is the answer of every declared rule for each order line that
	// was read, in the order of OrderLineRows. RelationAnswers holds the answers
	// of rules about the whole relation. Both are the same three-valued answers
	// `tei view -environment` reports, not a new judgement.
	RowAnswers      []rowAnswerProjection `json:"rowAnswers"`
	RelationAnswers []string              `json:"relationAnswers"`
	// ReadIdentity names the read this page shows: the selected order, the
	// header-detail Contract Query target, and the inputs that Query was
	// executed with. The answers are bound to it, so a list produced by a
	// different read is refused instead of rendered under these rows.
	ReadIdentity    readIdentityProjection     `json:"readIdentity"`
	RuntimeBindings []runtimeBindingProjection `json:"runtimeBindings"`
	AuthorityLimits []string                   `json:"authorityLimits"`
	Unresolved      []string                   `json:"unresolved"`
}

// readIdentityProjection is what makes a row answer locatable: which order was
// selected, which Contract Query was executed for its lines, and with which
// input. It is not a business key of the page, and no display position is
// derived from it.
type readIdentityProjection struct {
	SelectedOrderKey string `json:"selectedOrderKey"`
	Environment      string `json:"environment"`
	Pattern          string `json:"pattern"`
	Database         string `json:"database"`
	Contract         string `json:"contract"`
	Query            string `json:"query"`
	InputOrderID     string `json:"inputOrderId"`
	RowCount         int    `json:"rowCount"`
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
