package main

import "regexp"

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
	// RequiredFlags and OptionalFlags name the flags themselves: a bare name,
	// without a leading dash, without prose and without a parenthetical. A
	// caller that needs the flag set could not use Required/Optional, because
	// those carry the rules as sentences ("exactly one of -workspace or
	// -world") and, in one place, a note glued to a name; passing such an
	// element as a flag fails with "flag provided but not defined" (#4166).
	//
	// Required/Optional stay as they are: they say why, and this pair says
	// which. The union is the command's whole accepted flag set.
	RequiredFlags []string `json:"requiredFlags"`
	OptionalFlags []string `json:"optionalFlags"`
	Output        string   `json:"output"`
	// SchemaVersion is the schemaVersion the command returns, when it has one.
	// A command with an empty value is either text-only at this invocation or
	// outside the stable JSON contract; JSONContract says which.
	SchemaVersion string `json:"schemaVersion,omitempty"`
	// JSONContract is "enveloped" for the stable ok/error/schemaVersion envelope,
	// "raw" for a documented exception (world), and "none" for text-only output.
	JSONContract string `json:"jsonContract"`
	// Modes lists every value the response's `mode` field can take for this
	// command, so a caller that branches on `mode` learns the closed set before
	// running anything. An explicit "" entry means the field may be absent in a
	// response. A command with no Modes never writes `mode`.
	//
	// It exists because `mode` changes the result shape for source-index (whole
	// index vs one subject) and knowledge-context (context vs disclosure);
	// plugin-lock and plugin-resolve switch shape with a flag and say so with
	// `mode` as well.
	Modes []string `json:"modes,omitempty"`
	// SuccessShape names the body a successful response has. "envelope" is
	// the ok/result/error envelope; "identity" is the `version -json` build
	// identity, which carries schemaVersion at the top level but no ok and no
	// result. Declaring it here stops a caller from assuming `ok` after reading
	// jsonContract "enveloped" (#2824).
	SuccessShape string `json:"successShape,omitempty"`
	// ResultShape describes the JSON type of result on success, not the outer
	// successShape envelope. "array" marks an otherwise surprising shape.
	// If omitted, no shape claim is made: callers must inspect the actual result.
	// In particular, omitted does not mean "object". This field is descriptive,
	// not a new validator or permission to infer semantic completeness (#3347).
	ResultShape string `json:"resultShape,omitempty"`
	// ArgumentValues lists the closed value sets of this command's flags. A
	// caller that needs -subject-kind, -format, -mode, -direction, -disclose or
	// -data-design-candidate-kind had to guess or read prose (#3242). Open-ended
	// values (paths, IDs, fingerprints) are not listed, because there is no set
	// to advertise.
	ArgumentValues map[string][]string `json:"argumentValues,omitempty"`
	// ArgumentType names the flag's accepted form for a name in ArgumentValues,
	// so a caller knows whether a comma-separated list is allowed.
	ArgumentType map[string]string `json:"argumentType,omitempty"`
	// ArgumentDefaults carries the value used when the flag is omitted. Absent
	// names have no default: the value is required or has to be given.
	ArgumentDefaults map[string]string `json:"argumentDefaults,omitempty"`
	// RepeatableFlags names the flags that collect several values by design.
	// Every other flag holds one value, and supplying it twice is refused with
	// INVALID_ARGUMENT rather than silently keeping the last one (#3675).
	// An empty list means no flag of this command repeats.
	RepeatableFlags []string `json:"repeatableFlags,omitempty"`
	// WriteFlags names the flags that make the process create or modify a file,
	// and where. `effect: "read-only"` means the command does not change the
	// canonical meaning; it does not mean the process writes nothing (#3182).
	// A caller that needs "no file writes at all" must also check that none of
	// these flags was passed. An empty value means the command writes nothing.
	WriteFlags []string `json:"writeFlags,omitempty"`
}

func commandCatalog() cliCommandCatalog {
	commands := []cliCommandRecord{
		{Name: "version", Effect: "read-only", Required: []string{"no flags are required"}, Optional: []string{"-json"}, Output: "text; json when -json", SchemaVersion: "tei.cli.version/v0", JSONContract: "enveloped", SuccessShape: "identity"},
		{Name: "model catalog", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-namespace"}, Output: "json", SchemaVersion: "tei.cli.model/v0", JSONContract: "enveloped"},
		{Name: "model overview", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-limit"}, Output: "json", SchemaVersion: "tei.cli.model-overview/v0", JSONContract: "enveloped"},
		{Name: "model locate", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Optional: []string{"-limit"}, Output: "json", SchemaVersion: "tei.cli.model-locate/v0", JSONContract: "enveloped"},
		{Name: "workspace-files", Effect: "read-only", Required: []string{"-workspace"}, Output: "json", SchemaVersion: "tei.cli.workspace-files/v0", JSONContract: "enveloped"},
		{Name: "inspect", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Output: "json", SchemaVersion: "tei.cli.inspect/v0", JSONContract: "enveloped", ArgumentValues: valueSets("inspect"), ArgumentType: valueTypes("inspect")},
		{Name: "view", Effect: "read-only", Required: []string{"-workspace", "-application"}, Optional: []string{"-format: text|json for -application alone or -package, and text|json|html|print for a browse -pattern (a command -pattern takes text|json)", "at most one of -pattern, -package, -command", "-input and -execute with -command", "-reference-store with -execute", "-period-state with -execute", "-allow-capability with -execute", "-operation-log with -execute", "-lifecycle-state and -lifecycle-command with -execute", "-lifecycle-owner and -lifecycle-generation with -lifecycle-state"}, Output: "text; json when -format json", SchemaVersion: "tei.cli.view/v0", JSONContract: "enveloped", ArgumentValues: valueSets("view"), ArgumentType: valueTypes("view"), ArgumentDefaults: valueDefaults("view"), RepeatableFlags: []string{"-input", "-allow-capability"}, WriteFlags: []string{"-operation-log", "-lifecycle-state"}},
		{Name: "trace", Effect: "read-only", Required: []string{"-workspace", "-scope", "-subject-kind", "-subject-id"}, Optional: []string{"-direction", "-mode", "-offset and -limit (full only)", "-expected-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.trace/v0", JSONContract: "enveloped", ArgumentValues: valueSets("trace"), ArgumentType: valueTypes("trace"), ArgumentDefaults: valueDefaults("trace")},
		{Name: "source-index", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind and -subject-id together"}, Output: "json", SchemaVersion: "tei.cli.source-index/v0", JSONContract: "enveloped", Modes: []string{"index", "subject"}},
		{Name: "knowledge-context", Effect: "read-only", Required: []string{"-workspace"}, Optional: []string{"-subject-kind", "-subject-id", "-disclose", "-expected-data-design-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-context/v0", JSONContract: "enveloped", Modes: []string{"context", "disclosure"}, ArgumentValues: valueSets("knowledge-context"), ArgumentType: valueTypes("knowledge-context")},
		{Name: "knowledge-qualify", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-qualification-requirements"}, Output: "json", SchemaVersion: "tei.cli.knowledge-qualify/v0", JSONContract: "enveloped", Modes: []string{"qualification"}},
		{Name: "knowledge-review-plan", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id"}, Optional: []string{"-expected-review-plan-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-review-plan/v0", JSONContract: "enveloped", Modes: []string{"plan"}},
		{Name: "knowledge-review-handoff", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-qualification-requirements"}, Optional: []string{"-expected-review-handoff-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-review-handoff/v0", JSONContract: "enveloped", Modes: []string{"handoff"}},
		{Name: "knowledge-review-disclose", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-review-handoff"}, Optional: []string{"-expected-review-handoff-fingerprint", "-expected-disclosure-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-review-disclose/v0", JSONContract: "enveloped", Modes: []string{"disclosure"}},
		{Name: "knowledge-data-design-candidate-review", Effect: "read-only", Required: []string{"-workspace", "-subject-kind", "-subject-id", "-data-design-candidate-kind", "-data-design-candidate-fingerprint"}, Optional: []string{"-expected-data-design-fingerprint", "-expected-data-design-candidate-evidence-requirement-fingerprint", "-expected-data-design-candidate-review-fingerprint"}, Output: "json", SchemaVersion: "tei.cli.knowledge-data-design-candidate-review/v0", JSONContract: "enveloped", Modes: []string{"review"}, ArgumentValues: valueSets("knowledge-data-design-candidate-review"), ArgumentType: valueTypes("knowledge-data-design-candidate-review")},
		{Name: "validate", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-json"}, Output: "text; json when -workspace -json", SchemaVersion: "tei.cli.validate/v0", JSONContract: "enveloped"},
		{Name: "world", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json", JSONContract: "raw"},
		{Name: "requirements", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Output: "json", SchemaVersion: "tei.cli.requirements/v0", JSONContract: "enveloped", ResultShape: "array"},
		{Name: "plugin-lock", Effect: "read-only", Required: []string{"exactly one of -workspace or -world"}, Optional: []string{"-environment"}, Output: "json", SchemaVersion: "tei.cli.plugin-lock/v0", JSONContract: "enveloped", Modes: []string{"", "environment"}},
		{Name: "plugin-resolve", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment"}, Optional: []string{"-plugin-lock", "-diagnostic-candidates"}, Output: "json", SchemaVersion: "tei.cli.plugin-resolve/v0", JSONContract: "enveloped", Modes: []string{"", "unvalidated-candidates"}},
		{Name: "plugin-runtime-plan", Effect: "read-only", Required: []string{"exactly one of -workspace or -world", "-environment", "-plugin-lock"}, Output: "json", SchemaVersion: "tei.cli.plugin-runtime-plan/v0", JSONContract: "enveloped"},
	}
	fillFlagNameSets(commands)
	return cliCommandCatalog{
		SchemaVersion: "tei.cli.discovery/v0",
		Scope:         "selected-read-only-commands",
		Commands:      commands,
	}
}

// selfParsedFlagNames lists the flags of the commands that parse their own
// arguments, so they are not in the shared allowlist. It is the one place those
// three grammars are named for machines.
var selfParsedFlagNames = map[string]struct {
	required []string
	optional []string
}{
	"model catalog":  {[]string{"workspace"}, []string{"namespace"}},
	"model locate":   {[]string{"workspace", "subject-kind", "subject-id"}, []string{"limit"}},
	"model overview": {[]string{"workspace"}, []string{"limit"}},
}

// flagNamePattern matches the flag names a rule sentence mentions.
var flagNamePattern = regexp.MustCompile(`-([a-z][a-z0-9-]*)`)

// flagNamesFromProse extracts the flag names a rule sentence mentions, in the
// order they appear. "exactly one of -workspace or -world" names two flags;
// "none" names none.
func flagNamesFromProse(elements []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, element := range elements {
		for _, match := range flagNamePattern.FindAllStringSubmatch(element, -1) {
			if seen[match[1]] {
				continue
			}
			seen[match[1]] = true
			out = append(out, match[1])
		}
	}
	return out
}

// fillFlagNameSets writes RequiredFlags/OptionalFlags for every record. The
// optional side is derived from the accepted set, so the advertisement cannot
// list a flag the command does not read.
func fillFlagNameSets(commands []cliCommandRecord) {
	for i := range commands {
		record := &commands[i]
		if declared, ok := selfParsedFlagNames[record.Name]; ok {
			record.RequiredFlags = append([]string(nil), declared.required...)
			record.OptionalFlags = append([]string(nil), declared.optional...)
			continue
		}
		allowed, ok := selectedReadOnlyFlagAllowlist[record.Name]
		if !ok {
			continue
		}
		accepted := make(map[string]bool, len(allowed))
		for _, name := range allowed {
			accepted[name] = true
		}
		isRequired := map[string]bool{}
		required := make([]string, 0, len(allowed))
		// A rule sentence may mention a flag this command does not read; the
		// accepted set decides, not the prose.
		for _, name := range flagNamesFromProse(record.Required) {
			if !accepted[name] || isRequired[name] {
				continue
			}
			isRequired[name] = true
			required = append(required, name)
		}
		optional := make([]string, 0, len(allowed))
		for _, name := range allowed {
			if !isRequired[name] {
				optional = append(optional, name)
			}
		}
		record.RequiredFlags = required
		record.OptionalFlags = optional
	}
}
