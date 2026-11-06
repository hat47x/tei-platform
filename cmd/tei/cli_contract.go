package main

// Command discovery is deliberately limited to the CLI's initial read-only
// integration surface. It is not a second plugin or capability registry.
type cliCommandCatalog struct {
	SchemaVersion string             `json:"schemaVersion"`
	Scope         string             `json:"scope"`
	Commands      []cliCommandRecord `json:"commands"`
}

type cliCommandRecord struct {
	Name     string   `json:"name"`
	Effect   string   `json:"effect"`
	Required []string `json:"required"`
	Optional []string `json:"optional,omitempty"`
	Output   string   `json:"output"`
	// SchemaVersion is the schemaVersion the command returns, when it has one.
	// A command with an empty value is either text-only at this invocation or
	// outside the stable JSON contract; JSONContract says which.
	SchemaVersion string `json:"schemaVersion,omitempty"`
	// JSONContract is "enveloped" for the stable ok/error/schemaVersion envelope,
	// "raw" for a documented exception (world), and "none" for text-only output.
	JSONContract string `json:"jsonContract"`
}

func commandCatalog() cliCommandCatalog {
	return cliCommandCatalog{
		SchemaVersion: "tei.cli.discovery/v0",
		Scope:         "selected-read-only-commands",
		Commands: []cliCommandRecord{
			{Name: "version", Effect: "read-only", Required: []string{"none"}, Optional: []string{"-json"}, Output: "text; json when -json", SchemaVersion: "tei.cli.version/v0", JSONContract: "enveloped"},
			{Name: "model catalog", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-namespace"}, Output: "json", SchemaVersion: "tei.cli.model/v0", JSONContract: "enveloped"},
			{Name: "workspace-files", Effect: "read-only", Required: []string{"-workspace"}, Output: "json", SchemaVersion: "tei.cli.workspace-files/v0", JSONContract: "enveloped"},
			{Name: "inspect", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Output: "json", SchemaVersion: "tei.cli.inspect/v0", JSONContract: "enveloped"},
			{Name: "view", Effect: "read-only", Required: []string{"-workspace", "-application"}, Optional: []string{"-format", "at most one of -pattern, -package, -command", "-input and -execute with -command", "-period-state with -execute", "-allow-capability with -execute", "-operation-log with -execute", "-lifecycle-state and -lifecycle-command with -execute", "-lifecycle-owner and -lifecycle-generation with -lifecycle-state"}, Output: "text; json when -format json", SchemaVersion: "tei.cli.view/v0", JSONContract: "enveloped"},
			{Name: "trace", Effect: "read-only", Required: []string{"-workspace", "-scope", "-subject-kind", "-subject-id"}, Optional: []string{"-direction", "-mode", "-offset and -limit (full only)", "-expected-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.trace/v0", JSONContract: "enveloped"},
			{Name: "source-index", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind and -subject-id together"}, Output: "json", SchemaVersion: "tei.cli.source-index/v0", JSONContract: "enveloped"},
			{Name: "knowledge-context", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind", "-subject-id", "-disclose", "-expected-data-design-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-context/v0", JSONContract: "enveloped"},
			{Name: "knowledge-qualify", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-qualification-requirements"}, Output: "json", SchemaVersion: "tei.cli.knowledge-qualify/v0", JSONContract: "enveloped"},
			{Name: "knowledge-review-plan", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Optional: []string{"-expected-review-plan-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-review-plan/v0", JSONContract: "enveloped"},
			{Name: "knowledge-review-handoff", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-qualification-requirements"}, Optional: []string{"-expected-review-handoff-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-review-handoff/v0", JSONContract: "enveloped"},
			{Name: "knowledge-review-disclose", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-review-handoff"}, Optional: []string{"-expected-review-handoff-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-review-disclose/v0", JSONContract: "enveloped"},
			{Name: "knowledge-data-design-candidate-review", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-data-design-candidate-kind", "-data-design-candidate-fingerprint"}, Optional: []string{"-expected-data-design-fingerprint", "-expected-data-design-candidate-evidence-requirement-fingerprint", "-expected-data-design-candidate-review-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-data-design-candidate-review/v0", JSONContract: "enveloped"},
			{Name: "validate", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-json (workspace only)"}, Output: "text; json when -workspace -json", SchemaVersion: "tei.cli.validate/v0", JSONContract: "enveloped"},
			{Name: "world", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json", JSONContract: "raw"},
			{Name: "requirements", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json", SchemaVersion: "tei.cli.requirements/v0", JSONContract: "enveloped"},
			{Name: "plugin-lock", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-environment"}, Output: "json", SchemaVersion: "tei.cli.plugin-lock/v0", JSONContract: "enveloped"},
			{Name: "plugin-resolve", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment"}, Optional: []string{"-plugin-lock", "-diagnostic-candidates"}, Output: "json", SchemaVersion: "tei.cli.plugin-resolve/v0", JSONContract: "enveloped"},
			{Name: "plugin-runtime-plan", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment", "-plugin-lock"}, Output: "json", SchemaVersion: "tei.cli.plugin-runtime-plan/v0", JSONContract: "enveloped"},
		},
	}
}
