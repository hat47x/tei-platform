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
}

func commandCatalog() cliCommandCatalog {
	return cliCommandCatalog{
		SchemaVersion: "tei.cli.discovery/v0",
		Scope:         "selected-read-only-commands",
		Commands: []cliCommandRecord{
			{Name: "workspace-files", Effect: "read-only", Required: []string{"-workspace"}, Output: "json"},
			{Name: "inspect", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Output: "json"},
			{Name: "trace", Effect: "read-only", Required: []string{"-workspace", "-scope", "-subject-kind", "-subject-id"}, Optional: []string{"-direction", "-mode", "-offset and -limit (full only)", "-expected-fingerprint"}, Output: "json"},
			{Name: "source-index", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind and -subject-id together"}, Output: "json"},
			{Name: "knowledge-context", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind", "-subject-id", "-disclose", "-expected-data-design-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json"},
			{Name: "validate", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-json (workspace only)"}, Output: "text; json when -workspace -json"},
			{Name: "world", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json"},
			{Name: "requirements", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json"},
			{Name: "plugin-lock", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json"},
			{Name: "plugin-resolve", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment"}, Optional: []string{"-plugin-lock"}, Output: "json"},
			{Name: "plugin-runtime-plan", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment", "-plugin-lock"}, Output: "json"},
		},
	}
}
