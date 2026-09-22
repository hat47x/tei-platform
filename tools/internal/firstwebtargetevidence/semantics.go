package firstwebtargetevidence

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Provenance is the Target-local provenance envelope shared by the First Web
// Target Gate C evidence and the R7B behavior replay path.
type Provenance struct {
	HandoverBaselineID         string `json:"handoverBaselineId"`
	BaselineRepositoryRevision string `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string `json:"targetBuildRevision"`
	TargetAdapter              string `json:"targetAdapter"`
	Host                       string `json:"host"`
}

const (
	ObservationClassReadResult                    = "read-result"
	ObservationClassBusinessResultPlusObservation = "business-result-plus-separate-observation"
	ObservationClassAuthenticationDenied          = "business-denial-authentication"
	ObservationClassAuthorizationDenied           = "business-denial-authorization"
	ObservationClassValidationNotSatisfied        = "validation-not-satisfied"
	ObservationClassExecutionError                = "command-invocation-execution-error"
)

func ValidateReadArtifact(data []byte, expected Provenance) error {
	var artifact readEnvelope
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("decode read.json: %w", err)
	}
	if err := requireProvenance("read.json", artifact.Provenance, expected); err != nil {
		return err
	}
	if artifact.Result.ScenarioID != "order-entry.read-only-web-slice" ||
		artifact.Result.ApplicationID != "order_entry" {
		return fmt.Errorf("read Scenario identity is unexpected: %+v", artifact.Result)
	}
	return requireRuntimeBindings(artifact.Result.RuntimeBindings)
}

func ValidateCommandSuccessArtifact(data []byte, expected Provenance) error {
	var artifact commandEnvelope
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("decode submit-success.json: %w", err)
	}
	if err := requireProvenance("submit-success.json", artifact.Provenance, expected); err != nil {
		return err
	}
	return requireSuccessfulCommand(artifact.Result)
}

func ValidateAuthenticationDeniedArtifact(data []byte, expected Provenance) error {
	var artifact commandEnvelope
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("decode submit-authentication-denied.json: %w", err)
	}
	if err := requireProvenance("submit-authentication-denied.json", artifact.Provenance, expected); err != nil {
		return err
	}
	if artifact.Result.Stage != "authentication-denied" ||
		artifact.Result.Authentication == nil ||
		artifact.Result.Authentication.Outcome != "denied" ||
		artifact.Result.Command != nil ||
		artifact.Result.Observation != nil {
		return fmt.Errorf("authentication denial boundary is not preserved")
	}
	return nil
}

func ValidateAuthorizationDeniedArtifact(data []byte, expected Provenance) error {
	var artifact commandEnvelope
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("decode submit-authorization-denied.json: %w", err)
	}
	if err := requireProvenance("submit-authorization-denied.json", artifact.Provenance, expected); err != nil {
		return err
	}
	if artifact.Result.Stage != "authorization-denied" ||
		artifact.Result.Authorization == nil ||
		artifact.Result.Authorization.Outcome != "denied" ||
		artifact.Result.Command != nil ||
		artifact.Result.Observation != nil {
		return fmt.Errorf("authorization denial boundary is not preserved")
	}
	return nil
}

func ValidateValidationNotSatisfiedArtifact(data []byte, expected Provenance) error {
	var artifact commandEnvelope
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("decode submit-validation-not-satisfied.json: %w", err)
	}
	if err := requireProvenance("submit-validation-not-satisfied.json", artifact.Provenance, expected); err != nil {
		return err
	}
	if artifact.Result.Stage != "validation-not-satisfied" ||
		len(artifact.Result.Validations) != 1 ||
		artifact.Result.Validations[0].Validation != "sales.order_line.quantity_positive" ||
		artifact.Result.Validations[0].Result != "not-satisfied" ||
		artifact.Result.Authorization != nil ||
		artifact.Result.Command != nil ||
		artifact.Result.Observation != nil {
		return fmt.Errorf("Validation not-satisfied boundary is not preserved")
	}
	return nil
}

func ValidateExecutionErrorArtifact(data []byte, expected Provenance) error {
	var artifact commandErrorEnvelope
	if err := json.Unmarshal(data, &artifact); err != nil {
		return fmt.Errorf("decode submit-execution-error.json: %w", err)
	}
	if err := requireProvenance("submit-execution-error.json", artifact.Provenance, expected); err != nil {
		return err
	}
	if artifact.Result.Stage != "command" ||
		artifact.Result.Authorization == nil ||
		artifact.Result.Authorization.Outcome != "allowed" ||
		artifact.Result.Command != nil ||
		artifact.Result.Observation != nil ||
		!strings.Contains(artifact.ExecutionError, "synthetic submit adapter unavailable for order 1003") {
		return fmt.Errorf("Command Execution Error boundary is not preserved")
	}
	return nil
}

// RejectReferenceCredentialMaterial prevents Target-local fixture credentials
// from becoming retained replay Evidence.
func RejectReferenceCredentialMaterial(name string, data []byte) error {
	content := string(data)
	for _, forbidden := range []string{
		"editor-secret",
		"viewer-secret",
		"wrong-secret",
		"locked-secret",
		"disabled-secret",
		"credential-ref://legacy/",
	} {
		if strings.Contains(content, forbidden) {
			return fmt.Errorf("%s contains Reference credential material %q", name, forbidden)
		}
	}
	return nil
}

type readEnvelope struct {
	Provenance
	Result readArtifact `json:"result"`
}

type readArtifact struct {
	ScenarioID      string           `json:"scenarioId"`
	ApplicationID   string           `json:"applicationId"`
	RuntimeBindings []runtimeBinding `json:"runtimeBindings"`
}

type runtimeBinding struct {
	Pattern            string `json:"pattern"`
	Environment        string `json:"environment,omitempty"`
	Instance           string `json:"instance,omitempty"`
	Provider           string `json:"provider,omitempty"`
	ExecutorCapability string `json:"executorCapability,omitempty"`
	ContractQuery      string `json:"contractQuery,omitempty"`
	Datasource         string `json:"datasource,omitempty"`
	Database           string `json:"database,omitempty"`
	Access             string `json:"access,omitempty"`
}

type commandEnvelope struct {
	Provenance
	Result commandArtifact `json:"result"`
}

type commandErrorEnvelope struct {
	Provenance
	Result         commandArtifact `json:"result"`
	ExecutionError string          `json:"executionError"`
}

type commandArtifact struct {
	ScenarioID     string             `json:"scenarioId"`
	ApplicationID  string             `json:"applicationId"`
	Stage          string             `json:"stage"`
	Authentication *businessResult    `json:"authentication,omitempty"`
	Validations    []validationResult `json:"validations,omitempty"`
	Authorization  *businessResult    `json:"authorization,omitempty"`
	Command        *businessResult    `json:"command,omitempty"`
	Observation    *observation       `json:"observation,omitempty"`
}

type validationResult struct {
	Validation string `json:"validation"`
	Result     string `json:"result"`
}

type businessResult struct {
	Capability string         `json:"capability"`
	Mode       string         `json:"mode"`
	Outcome    string         `json:"outcome,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type observation struct {
	ReadIdentity string `json:"readIdentity"`
	Nature       string `json:"nature"`
	Result       any    `json:"result"`
}

func requireProvenance(name string, actual, expected Provenance) error {
	if actual != expected {
		return fmt.Errorf("%s provenance=%+v, want %+v", name, actual, expected)
	}
	return nil
}

func requireRuntimeBindings(bindings []runtimeBinding) error {
	byPattern := make(map[string]runtimeBinding, len(bindings))
	for _, binding := range bindings {
		byPattern[binding.Pattern] = binding
	}
	browse := byPattern["order_entry.recent_orders"]
	if browse.Datasource != "sales" || browse.Database != "sales_core" || browse.Access != "read-write" {
		return fmt.Errorf("browse binding=%+v", browse)
	}
	if browse.Environment != "" || browse.Instance != "" {
		return fmt.Errorf("browse Environment was inferred: %+v", browse)
	}
	customer := byPattern["order_entry.customer_reference"]
	if customer.Environment != "development" ||
		customer.Instance != "master_data.dev" ||
		customer.Provider != "oracle" ||
		customer.ExecutorCapability != "database.query.oracle" ||
		customer.ContractQuery != "master_data/master_data.customer/find_by_id" {
		return fmt.Errorf("customer runtime binding=%+v", customer)
	}
	lines := byPattern["order_entry.order_lines"]
	if lines.Environment != "development" ||
		lines.Instance != "sales_core.dev" ||
		lines.Provider != "postgres" ||
		lines.ExecutorCapability != "database.query.postgres" ||
		lines.ContractQuery != "sales_core/sales_core.order_line/find_by_order" {
		return fmt.Errorf("order-lines runtime binding=%+v", lines)
	}
	return nil
}

func requireSuccessfulCommand(result commandArtifact) error {
	if result.ScenarioID != "order-entry.submit-order-web-slice" ||
		result.ApplicationID != "order_entry" ||
		result.Stage != "completed" {
		return fmt.Errorf("successful command Scenario identity/stage=%+v", result)
	}
	if result.Authentication == nil ||
		result.Authentication.Capability != "security.authenticate" ||
		result.Authentication.Outcome != "authenticated" {
		return fmt.Errorf("successful command Authentication=%+v", result.Authentication)
	}
	if len(result.Validations) != 2 {
		return fmt.Errorf("successful command Validation count=%d", len(result.Validations))
	}
	for _, validation := range result.Validations {
		if validation.Result != "satisfied" {
			return fmt.Errorf("successful command Validation=%+v", validation)
		}
	}
	if result.Authorization == nil ||
		result.Authorization.Capability != "security.authorize" ||
		result.Authorization.Outcome != "allowed" {
		return fmt.Errorf("successful command Authorization=%+v", result.Authorization)
	}
	if result.Command == nil ||
		result.Command.Capability != "sales.order.submit" ||
		result.Command.Mode != "output" {
		return fmt.Errorf("successful command Business Result=%+v", result.Command)
	}
	if result.Observation == nil ||
		result.Observation.ReadIdentity != "synthetic.order_state.read" ||
		result.Observation.Nature != "synthetic" ||
		fmt.Sprint(result.Observation.Result) != "submitted" {
		return fmt.Errorf("successful command Observation=%+v", result.Observation)
	}
	return nil
}
