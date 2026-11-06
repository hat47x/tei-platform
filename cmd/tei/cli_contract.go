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
	// OutputSchemaVersion is the schemaVersion of the JSON this command
	// prints. It is empty only when OutsideContract is true. Added to
	// tei.cli.discovery/v0 as an optional field (#2824); readers that ignore
	// unknown fields are unaffected.
	OutputSchemaVersion string `json:"outputSchemaVersion,omitempty"`
	// OutsideContract marks JSON output that has no tei.cli.* envelope and is
	// not part of the stable external JSON contract (see
	// plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md section 3).
	OutsideContract bool `json:"outsideContract,omitempty"`
}

func commandCatalog() cliCommandCatalog {
	return cliCommandCatalog{
		SchemaVersion: "tei.cli.discovery/v0",
		Scope:         "selected-read-only-commands",
		Commands: []cliCommandRecord{
			{Name: "version", Effect: "read-only", Required: []string{"none"}, Optional: []string{"-json"}, Output: "text; json when -json", OutputSchemaVersion: "tei.cli.version/v0"},
			{Name: "model catalog", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-namespace"}, Output: "json", OutputSchemaVersion: "tei.cli.model/v0"},
			{Name: "workspace-files", Effect: "read-only", Required: []string{"-workspace"}, Output: "json", OutputSchemaVersion: "tei.cli.workspace-files/v0"},
			{Name: "inspect", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Output: "json", OutputSchemaVersion: "tei.cli.inspect/v0"},
			{Name: "view", Effect: "read-only", Required: []string{"-workspace", "-application"}, Optional: []string{"-format", "at most one of -pattern, -package, -command", "-input and -execute with -command", "-period-state with -execute", "-allow-capability with -execute"}, Output: "text; json when -format json", OutputSchemaVersion: "tei.cli.view/v0"},
			{Name: "trace", Effect: "read-only", Required: []string{"-workspace", "-scope", "-subject-kind", "-subject-id"}, Optional: []string{"-direction", "-mode", "-offset and -limit (full only)", "-expected-fingerprint"}, Output: "json", OutputSchemaVersion: "tei.cli.trace/v0"},
			{Name: "source-index", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind and -subject-id together"}, Output: "json", OutputSchemaVersion: "tei.cli.source-index/v0"},
			{Name: "knowledge-context", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind", "-subject-id", "-disclose", "-expected-data-design-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json", OutputSchemaVersion: "tei.cli.knowledge-context/v0"},
			{Name: "knowledge-qualify", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-qualification-requirements"}, Output: "json", OutputSchemaVersion: "tei.cli.knowledge-qualify/v0"},
			{Name: "knowledge-review-plan", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Optional: []string{"-expected-review-plan-fingerprint"}, Output: "json", OutputSchemaVersion: "tei.cli.knowledge-review-plan/v0"},
			{Name: "knowledge-review-handoff", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-qualification-requirements"}, Optional: []string{"-expected-review-handoff-fingerprint"}, Output: "json", OutputSchemaVersion: "tei.cli.knowledge-review-handoff/v0"},
			{Name: "knowledge-review-disclose", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-review-handoff"}, Optional: []string{"-expected-review-handoff-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json", OutputSchemaVersion: "tei.cli.knowledge-review-disclose/v0"},
			{Name: "knowledge-data-design-candidate-review", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-data-design-candidate-kind", "-data-design-candidate-fingerprint"}, Optional: []string{"-expected-data-design-fingerprint", "-expected-data-design-candidate-evidence-requirement-fingerprint", "-expected-data-design-candidate-review-fingerprint"}, Output: "json", OutputSchemaVersion: "tei.cli.knowledge-data-design-candidate-review/v0"},
			{Name: "validate", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-json (workspace only)"}, Output: "text; json when -workspace -json", OutputSchemaVersion: "tei.cli.validate/v0"},
			{Name: "world", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json", OutsideContract: true},
			{Name: "requirements", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json", OutputSchemaVersion: "tei.cli.requirements/v0"},
			{Name: "plugin-lock", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-environment"}, Output: "json", OutputSchemaVersion: "tei.cli.plugin-lock/v0"},
			{Name: "plugin-resolve", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment"}, Optional: []string{"-plugin-lock", "-diagnostic-candidates"}, Output: "json", OutputSchemaVersion: "tei.cli.plugin-resolve/v0"},
			{Name: "plugin-runtime-plan", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment", "-plugin-lock"}, Output: "json", OutputSchemaVersion: "tei.cli.plugin-runtime-plan/v0"},
		},
	}
}
