package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tei/internal/buildintent"
	"tei/internal/plugin"
	"tei/internal/syntax"
)

// reference-plugin-package-build-child is the P-1 child of SB-C0 §4.10: it
// consumes an exact BuildPlan, materializes the Plugin package from it, writes
// the artifact where `tei self-build` expects it, and reports the plan
// fingerprint it actually used.
//
// It exists because the earlier survey (§4.10.1) found that no tool both
// satisfied the child grammar (-artifact-out, -state, one JSON on stdout) and
// reported a plan fingerprint. Without such a child the cross-check can only be
// `unperformed`.
//
// The report is a claim about the plan this process loaded and validated, not
// about the meaning that produced it and not about the artifact being that
// plan's output. The boundary string says so.

const (
	childSchema   = "tei.reference.plugin-package-build-child/v0"
	childBoundary = "one child process that loaded and validated an exact BuildPlan and materialized its Plugin package; this is not acceptance, not L3, and not proof that the plan came from any particular declaration"
)

type childReport struct {
	SchemaVersion   string `json:"schemaVersion"`
	Nature          string `json:"nature"`
	PlanPath        string `json:"planPath"`
	PlanBytesDigest string `json:"planBytesDigest"`
	PlanFingerprint string `json:"planFingerprint"`
	PluginID        string `json:"pluginId"`
	PluginVersion   string `json:"pluginVersion"`
	OutputKind      string `json:"outputKind"`
	ArtifactPath    string `json:"artifactPath,omitempty"`
	ArtifactDigest  string `json:"artifactDigest,omitempty"`
	StateDir        string `json:"stateDir,omitempty"`
	Boundary        string `json:"claimBoundary"`
	// Lock is the binding this process forms and validates itself, so a caller
	// no longer has to trust two separate self-reports. It ties the plan this
	// process loaded to the artifact it wrote.
	Lock *selfLock `json:"buildArtifactLock,omitempty"`
	// The declaration readings. Present only when the caller names -intent, so
	// a caller that passes no declaration gets no values it might mistake for a
	// check that ran.
	IntentPath string `json:"intentPath,omitempty"`
	// DefinitionBytesDigest is the digest of the definition file this process
	// read, and DefinitionMatchesPlan says whether it equals the digest the plan
	// records for that source. A plan that names one file and a run that reads
	// another would otherwise produce an artifact the plan does not describe.
	DefinitionBytesDigest string `json:"definitionBytesDigest,omitempty"`
	DefinitionMatchesPlan bool   `json:"definitionMatchesPlan"`
	IntentBytesDigest     string `json:"intentBytesDigest,omitempty"`
	IntentFingerprint     string `json:"intentFingerprint,omitempty"`
	// IntentBoundToPlan says whether this process checked that the declaration
	// it read names the same subject and target as the plan it consumed. It is
	// the child's own check, not a proof that the plan came from that
	// declaration.
	IntentBoundToPlan bool   `json:"intentBoundToPlan"`
	IntentBindingNote string `json:"intentBindingNote,omitempty"`
}

const intentBindingBoundary = "this process read the declaration bytes and checked that its subject identity agrees with the plan it consumed; agreement of identity is not proof that this declaration produced that plan, and not acceptance or runtime authority"

// selfLock is the child's own validated plan-to-artifact binding. The child
// checks it with internal/plugin before writing anything, so a malformed
// digest or a missing identity field stops the run instead of being reported.
type selfLock struct {
	FormatVersion   int    `json:"formatVersion"`
	PlanFingerprint string `json:"planFingerprint"`
	ArtifactDigest  string `json:"artifactDigest"`
	LockFingerprint string `json:"lockFingerprint"`
	Validated       bool   `json:"validated"`
	Boundary        string `json:"boundary"`
}

const lockBoundary = "this process validated one BuildArtifactLock over the plan it loaded and the artifact it wrote; validation shows the lock is well formed, not that the plan produced the artifact"

// formSelfLock builds and validates the lock from the values this process
// actually holds: the plan it loaded and the bytes it wrote.
func formSelfLock(plan plugin.BuildPlan, planFingerprint, artifactDigest string) (selfLock, error) {
	lock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{{
			Plugin:          plan.Plugin,
			OutputKind:      plan.OutputKind,
			Target:          plan.Target,
			Profile:         plan.Profile,
			PlanFingerprint: planFingerprint,
			ArtifactDigest:  artifactDigest,
		}},
	}
	if err := lock.Validate(); err != nil {
		return selfLock{}, fmt.Errorf("the plan-to-artifact lock does not validate: %w", err)
	}
	fingerprint, err := plugin.BuildArtifactLockFingerprint(lock)
	if err != nil {
		return selfLock{}, fmt.Errorf("the lock fingerprint could not be derived: %w", err)
	}
	return selfLock{
		FormatVersion:   plugin.BuildArtifactLockFormatVersion,
		PlanFingerprint: planFingerprint,
		ArtifactDigest:  artifactDigest,
		LockFingerprint: fingerprint,
		Validated:       true,
		Boundary:        lockBoundary,
	}, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-plugin-package-build-child", flag.ContinueOnError)
	fs.SetOutput(stderr)
	planPath := fs.String("plan", "", "exact BuildPlan JSON to consume (required)")
	pluginPath := fs.String("plugin", "", "Plugin definition (.tei) the plan names (required)")
	artifactOut := fs.String("artifact-out", "", "where to write the package artifact (required)")
	stateDir := fs.String("state", "", "isolated state directory for this attempt")
	intentPath := fs.String("intent", "", "Build Intent declaration this process also reads (optional)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *planPath == "" || *pluginPath == "" || *artifactOut == "" {
		fmt.Fprintln(stderr, "-plan, -plugin and -artifact-out are required")
		return 2
	}

	report, err := materialize(*planPath, *pluginPath, *artifactOut, *stateDir, *intentPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

// materialize reads the plan as bytes, hashes those bytes, decodes and
// validates the plan, then materializes the package. The byte digest and the
// fingerprint are recorded separately: the first is what this process held, the
// second is what internal/plugin derives from the normalized plan.
func materialize(planPath, pluginPath, artifactOut, stateDir, intentPath string) (childReport, error) {
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return childReport{}, fmt.Errorf("read plan: %w", err)
	}
	var plan plugin.BuildPlan
	decoder := json.NewDecoder(bytes.NewReader(planBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return childReport{}, fmt.Errorf("decode plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return childReport{}, errors.New("plan contains multiple JSON values")
		}
		return childReport{}, fmt.Errorf("decode trailing plan content: %w", err)
	}
	// Validate before touching the artifact: a plan that does not validate must
	// not produce output, so a failure cannot be read as "the build worked".
	if err := plan.Validate(); err != nil {
		return childReport{}, fmt.Errorf("plan does not validate: %w", err)
	}
	fingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return childReport{}, fmt.Errorf("plan fingerprint: %w", err)
	}

	definitionSource, err := os.ReadFile(pluginPath)
	if err != nil {
		return childReport{}, fmt.Errorf("read plugin definition: %w", err)
	}
	// The plan names an exact source closure. A run that reads a different file
	// would build something the plan does not describe, and no later observation
	// would notice: the plan fingerprint still matches (this process took it from
	// the plan) and the bytes are reproducible. So the match is checked here,
	// before the artifact exists.
	definitionDigest := buildintent.SingleFileSourceDigest(definitionSource)
	plannedDigest := plannedDefinitionDigest(plan)
	if plannedDigest == "" {
		return childReport{}, fmt.Errorf("the plan does not record a digest for the plugin definition source, so this run cannot show that it built the planned input")
	}
	if definitionDigest != plannedDigest {
		return childReport{}, fmt.Errorf("the definition read (%s) is not the one the plan names (%s); this run would build an artifact the plan does not describe", definitionDigest, plannedDigest)
	}
	definition, err := syntax.NewParser(string(definitionSource)).ParsePlugin()
	if err != nil {
		return childReport{}, fmt.Errorf("parse plugin definition: %w", err)
	}
	artifact, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		return childReport{}, err
	}

	if err := os.MkdirAll(filepath.Dir(artifactOut), 0o755); err != nil {
		return childReport{}, fmt.Errorf("prepare artifact directory: %w", err)
	}
	if err := os.WriteFile(artifactOut, artifact, 0o644); err != nil {
		return childReport{}, fmt.Errorf("write artifact: %w", err)
	}
	if stateDir != "" {
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return childReport{}, fmt.Errorf("prepare state directory: %w", err)
		}
	}

	artifactDigest := digestOf(artifact)
	// Form and validate the lock in this process, from the values it holds. A
	// lock that does not validate stops the run before the report is written,
	// so a caller never receives a binding this process could not check.
	lock, err := formSelfLock(plan, fingerprint, artifactDigest)
	if err != nil {
		return childReport{}, err
	}

	report := childReport{
		SchemaVersion:         childSchema,
		Nature:                "one child process that consumes an exact BuildPlan and materializes its Plugin package",
		PlanPath:              planPath,
		PlanBytesDigest:       digestOf(planBytes),
		DefinitionBytesDigest: definitionDigest,
		DefinitionMatchesPlan: true,
		PlanFingerprint:       fingerprint,
		PluginID:              string(plan.Plugin.ID),
		PluginVersion:         plan.Plugin.Version,
		OutputKind:            string(plan.OutputKind),
		ArtifactPath:          artifactOut,
		ArtifactDigest:        artifactDigest,
		StateDir:              stateDir,
		Boundary:              childBoundary,
		Lock:                  &lock,
	}

	// §4.14: when the caller names the declaration, this process reads it with
	// the shared strict reader and reports its own readings. The entry can then
	// compare them with the values it derived itself, instead of relying on the
	// builder's silence.
	if strings.TrimSpace(intentPath) != "" {
		intentBytes, err := os.ReadFile(intentPath)
		if err != nil {
			return childReport{}, fmt.Errorf("read declaration: %w", err)
		}
		intent, err := buildintent.ParseIntent(intentBytes)
		if err != nil {
			return childReport{}, fmt.Errorf("declaration: %w", err)
		}
		intentFingerprint, err := buildintent.IntentFingerprint(intent)
		if err != nil {
			return childReport{}, fmt.Errorf("declaration fingerprint: %w", err)
		}
		report.IntentPath = intentPath
		report.IntentBytesDigest = digestOf(intentBytes)
		report.IntentFingerprint = intentFingerprint
		report.IntentBindingNote = intentBindingBoundary
		// The identity check the child can make on its own: the declaration's
		// subject must be the plan's plugin. Anything else means the two inputs
		// describe different things.
		if string(intent.Subject.ID) != string(plan.Plugin.ID) || intent.Subject.Version != plan.Plugin.Version {
			return childReport{}, fmt.Errorf(
				"the declaration names %s@%s but the plan builds %s@%s",
				intent.Subject.ID, intent.Subject.Version, plan.Plugin.ID, plan.Plugin.Version,
			)
		}
		// §4.14の次の一歩: the declaration's build coordinates must match the
		// plan's. Identity alone says the two describe the same plugin; the
		// coordinates say they describe the same build of it. Each mismatch is
		// named separately so a caller can tell which coordinate disagreed.
		coordinates := []struct {
			name   string
			intent string
			plan   string
		}{
			{"outputKind", string(intent.OutputKind), string(plan.OutputKind)},
			{"target", intent.Target, plan.Target},
			{"profile", intent.Profile, plan.Profile},
		}
		for _, c := range coordinates {
			if c.intent != c.plan {
				return childReport{}, fmt.Errorf(
					"the declaration's %s %q does not match the plan's %q",
					c.name, c.intent, c.plan,
				)
			}
		}
		// §4.15の次の一歩: the remaining declaration fields that the plan also
		// carries. Sources and dependencies are compared by identity only: the
		// declaration names a reference, the plan carries the derived locator and
		// digest, so comparing those would compare two different things and call
		// every correct build a mismatch.
		if err := compareIntentFields(intent, plan); err != nil {
			return childReport{}, err
		}
		report.IntentBoundToPlan = true
	}
	return report, nil
}

// compareIntentFields checks the declaration fields that the plan also carries,
// beyond identity and build coordinates.
//
// Compared: recipe kind and entrypoint, source identity (kind and id),
// dependency identity (kind and id), and parameters as a name/value map.
//
// Parameters are compared after measuring what normalisation does to them.
// Both sides sort parameters by name and nothing else:
// buildintent.NormalizeIntent sorts the declaration's, plugin.NormalizeBuildPlan
// sorts the plan's, and neither changes a value. So a difference in values is a
// real difference, and comparison by name avoids depending on declaration order.
//
// The builder contract is compared through the correspondence the resolver
// already applies: a requirement of contract C at major N resolves to a
// BuilderIdentity whose ContractVersion is "C/vN" (fixture.go builds it with
// fmt.Sprintf("%s/v%d", p.Contract, p.ContractMajor)). That correspondence is a
// rule, not a measurement, so it is written here explicitly: if the resolver
// ever changes the spelling, this comparison is the place that breaks, and the
// test that pins it says so.
//
// Not compared: a source's locator and digest (the plan derives them from
// bytes, the declaration names a reference) and a dependency's version and
// artifact digest (derived). Comparing those would call every correct build a
// mismatch. The builder's identity and artifact digest are also not compared:
// the declaration names a selection policy, not the exact builder.
func compareIntentFields(intent buildintent.BuildIntent, plan plugin.BuildPlan) error {
	// The declaration states a contract and a major; the plan states the
	// resolved ContractVersion. Compare them through the rule the resolver uses.
	if want := intent.Builder.Contract + "/v" + strconv.Itoa(intent.Builder.CompatibleMajor); want != plan.Builder.ContractVersion {
		return fmt.Errorf(
			"the declaration's builder contract %s at major %d resolves to %q but the plan's builder declares %q",
			intent.Builder.Contract, intent.Builder.CompatibleMajor, want, plan.Builder.ContractVersion,
		)
	}
	if intent.Recipe.Kind != plan.Recipe.Kind {
		return fmt.Errorf("the declaration's recipe kind %q does not match the plan's %q", intent.Recipe.Kind, plan.Recipe.Kind)
	}
	if intent.Recipe.Entrypoint != plan.Recipe.Entrypoint {
		return fmt.Errorf("the declaration's recipe entrypoint %q does not match the plan's %q", intent.Recipe.Entrypoint, plan.Recipe.Entrypoint)
	}

	planSources := make(map[string]string, len(plan.Sources))
	for _, s := range plan.Sources {
		planSources[s.ID] = s.Kind
	}
	for _, s := range intent.Sources {
		kind, ok := planSources[s.ID]
		if !ok {
			return fmt.Errorf("the declaration's source %q is not in the plan's sources", s.ID)
		}
		if kind != s.Kind {
			return fmt.Errorf("the declaration's source %q has kind %q but the plan's has %q", s.ID, s.Kind, kind)
		}
	}
	if len(intent.Sources) != len(plan.Sources) {
		return fmt.Errorf("the declaration names %d sources but the plan has %d", len(intent.Sources), len(plan.Sources))
	}

	planDependencies := make(map[string]string, len(plan.Dependencies))
	for _, d := range plan.Dependencies {
		planDependencies[d.ID] = d.Kind
	}
	for _, d := range intent.Dependencies {
		kind, ok := planDependencies[d.ID]
		if !ok {
			return fmt.Errorf("the declaration's dependency %q is not in the plan's dependencies", d.ID)
		}
		if kind != d.Kind {
			return fmt.Errorf("the declaration's dependency %q has kind %q but the plan's has %q", d.ID, d.Kind, kind)
		}
	}
	if len(intent.Dependencies) != len(plan.Dependencies) {
		return fmt.Errorf("the declaration names %d dependencies but the plan has %d", len(intent.Dependencies), len(plan.Dependencies))
	}

	// Parameters: compared by name, because both sides sort by name and neither
	// changes a value (see the note above).
	planParameters := make(map[string]string, len(plan.Parameters))
	for _, p := range plan.Parameters {
		planParameters[p.Name] = p.Value
	}
	for _, p := range intent.Parameters {
		value, ok := planParameters[p.Name]
		if !ok {
			return fmt.Errorf("the declaration's parameter %q is not in the plan's parameters", p.Name)
		}
		if value != p.Value {
			return fmt.Errorf("the declaration's parameter %q is %q but the plan's is %q", p.Name, p.Value, value)
		}
	}
	if len(intent.Parameters) != len(plan.Parameters) {
		return fmt.Errorf("the declaration names %d parameters but the plan has %d", len(intent.Parameters), len(plan.Parameters))
	}

	// Builder: the declaration states a requirement (contract + major, and
	// optionally a provider or an exact pin); the plan holds the resolved
	// identity. The correspondence is not invented here — the resolver filters
	// by contract and major and renders the identity with
	// buildintent.BuilderContractVersion, so comparing that same form tests the
	// requirement rather than a format.
	wantContract := buildintent.BuilderContractVersion(intent.Builder.Contract, intent.Builder.CompatibleMajor)
	if plan.Builder.ContractVersion != wantContract {
		return fmt.Errorf(
			"the declaration requires builder contract %q but the plan resolved %q",
			wantContract, plan.Builder.ContractVersion,
		)
	}
	if provider := intent.Builder.Provider; provider != nil {
		if string(plan.Builder.ID) != string(provider.Plugin) {
			return fmt.Errorf(
				"the declaration's builder provider is %s but the plan resolved %s",
				provider.Plugin, plan.Builder.ID,
			)
		}
		// provider.implementation has no counterpart in plugin.BuilderIdentity,
		// so it is not compared. Naming the gap here keeps a later reader from
		// assuming it was checked.
	}
	if pin := intent.Builder.Pin; pin != nil {
		if plan.Builder.ArtifactDigest != pin.ArtifactDigest {
			return fmt.Errorf(
				"the declaration pins builder artifact %s but the plan resolved %s",
				pin.ArtifactDigest, plan.Builder.ArtifactDigest,
			)
		}
	}
	return nil
}

// plannedDefinitionDigest returns the digest the plan records for its plugin
// definition source, or "" when the plan does not name one.
func plannedDefinitionDigest(plan plugin.BuildPlan) string {
	for _, source := range plan.Sources {
		if source.Kind == "plugin-definition" && source.Digest != "" {
			return source.Digest
		}
	}
	return ""
}

func digestOf(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
