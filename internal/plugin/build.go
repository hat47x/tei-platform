package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/permission"
)

const (
	BuildPlanFormatVersion         = 1
	BuildArtifactLockFormatVersion = 1
)

// BuildSource is one exact member of the source closure that affects a Plugin
// build. ID is the stable source identity used by the build definition.
// Locator tells the builder where that member is obtained. Digest fixes the
// bytes used by the build.
type BuildSource struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Locator string `json:"locator"`
	Digest  string `json:"digest"`
}

// BuildDependency is an exact build-time dependency. It is deliberately
// separate from Definition.Requires, which describes Plugin runtime/package
// dependencies.
type BuildDependency struct {
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	Version        string `json:"version"`
	ArtifactDigest string `json:"artifactDigest"`
}

// BuilderIdentity identifies the exact builder used to produce an artifact.
// The builder may itself be implemented by TEI or by another language/toolchain.
type BuilderIdentity struct {
	ID              core.CanonicalID `json:"id"`
	Version         string           `json:"version"`
	ContractVersion string           `json:"contractVersion"`
	ArtifactDigest  string           `json:"artifactDigest"`
}

func (b BuilderIdentity) Validate() error {
	if b.ID == "" {
		return fmt.Errorf("builder id is required")
	}
	if _, err := core.ParseCanonicalID(string(b.ID)); err != nil {
		return fmt.Errorf("builder id: %w", err)
	}
	if strings.TrimSpace(b.Version) == "" {
		return fmt.Errorf("builder version is required")
	}
	if strings.TrimSpace(b.ContractVersion) == "" {
		return fmt.Errorf("builder contract version is required")
	}
	if err := validateBuildDigest("builder artifact digest", b.ArtifactDigest); err != nil {
		return err
	}
	return nil
}

// BuildParameter contains a non-secret input that may affect the build output.
// Secrets and signing material belong to separate authority/provenance paths and
// must not be embedded here.
type BuildParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// BuildRecipe identifies what the selected builder is asked to build without
// embedding an arbitrary shell command in the canonical build contract.
// Kind is interpreted by the builder adapter; Entrypoint identifies the
// language/toolchain-local source unit such as a Go package or TEI module.
type BuildRecipe struct {
	Kind       string `json:"kind"`
	Entrypoint string `json:"entrypoint"`
}

// BuildOutputKind distinguishes artifacts that have different authority and
// consumption paths. The value is intentionally extensible; Runtime binding
// explicitly accepts only BuildOutputRuntimeArtifact.
type BuildOutputKind string

const (
	BuildOutputRuntimeArtifact BuildOutputKind = "runtime-artifact"
	BuildOutputPluginPackage   BuildOutputKind = "plugin-package"
	BuildOutputBuilderArtifact BuildOutputKind = "builder-artifact"
)

// BuildPlan fixes the resolved inputs for one Plugin artifact build. It does not
// grant Acceptance or Runtime Authority and does not execute the build.
type BuildPlan struct {
	FormatVersion               int               `json:"formatVersion"`
	Plugin                      Identity          `json:"plugin"`
	PluginDefinitionFingerprint string            `json:"pluginDefinitionFingerprint"`
	Sources                     []BuildSource     `json:"sources"`
	Builder                     BuilderIdentity   `json:"builder"`
	Recipe                      BuildRecipe       `json:"recipe"`
	OutputKind                  BuildOutputKind   `json:"outputKind"`
	Dependencies                []BuildDependency `json:"dependencies,omitempty"`
	Target                      string            `json:"target"`
	Profile                     string            `json:"profile,omitempty"`
	Parameters                  []BuildParameter  `json:"parameters,omitempty"`
}

// LockedBuild binds one BuildPlan fingerprint to the artifact produced from it.
type LockedBuild struct {
	Plugin          Identity        `json:"plugin"`
	OutputKind      BuildOutputKind `json:"outputKind"`
	Target          string          `json:"target"`
	Profile         string          `json:"profile,omitempty"`
	PlanFingerprint string          `json:"planFingerprint"`
	ArtifactDigest  string          `json:"artifactDigest"`
}

// BuildArtifactLock is separate from Plugin Lock. Plugin Lock answers which
// exact implementation is used at runtime; BuildArtifactLock answers which
// exact build plan produced a particular artifact. The longer Go name avoids
// colliding with the existing BuildLock function that constructs Runtime
// Plugin Lock.
type BuildArtifactLock struct {
	FormatVersion int           `json:"formatVersion"`
	Builds        []LockedBuild `json:"builds"`
}

// BuildDefinitionFingerprint identifies the build-relevant Plugin definition
// without the produced artifact identity. Keeping Artifact outside this
// fingerprint avoids a cycle where a BuildPlan would need the output digest
// before the artifact exists. Runtime DefinitionFingerprint intentionally keeps
// Artifact included.
func BuildDefinitionFingerprint(d Definition) string {
	n := NormalizeDefinition(cloneBuildDefinition(d))
	n.Artifact = Artifact{}
	b, _ := json.Marshal(n)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func cloneBuildDefinition(d Definition) Definition {
	out := d
	out.Runtime.Args = append([]string(nil), d.Runtime.Args...)
	out.Requires = append([]Requirement(nil), d.Requires...)
	out.External = append([]ExternalRequirement(nil), d.External...)
	out.Provides = make([]Capability, len(d.Provides))
	for i, capability := range d.Provides {
		out.Provides[i] = capability
		out.Provides[i].Actions = append([]deployment.ActionKind(nil), capability.Actions...)
		out.Provides[i].Observations = append([]deployment.ObservationKind(nil), capability.Observations...)
		out.Provides[i].Evidence = append([]deployment.EvidenceKind(nil), capability.Evidence...)
		out.Provides[i].DatabaseProviders = append([]environment.DatabaseProvider(nil), capability.DatabaseProviders...)
		out.Provides[i].MigrationTools = append([]migration.Tool(nil), capability.MigrationTools...)
		out.Provides[i].Features = append([]environment.Feature(nil), capability.Features...)
		out.Provides[i].Permissions = append([]permission.Spec(nil), capability.Permissions...)
	}
	return out
}

func NormalizeBuildPlan(p BuildPlan) BuildPlan {
	out := p
	out.Sources = append([]BuildSource(nil), p.Sources...)
	out.Dependencies = append([]BuildDependency(nil), p.Dependencies...)
	out.Parameters = append([]BuildParameter(nil), p.Parameters...)

	sort.Slice(out.Sources, func(i, j int) bool {
		if out.Sources[i].Kind != out.Sources[j].Kind {
			return out.Sources[i].Kind < out.Sources[j].Kind
		}
		if out.Sources[i].ID != out.Sources[j].ID {
			return out.Sources[i].ID < out.Sources[j].ID
		}
		return out.Sources[i].Locator < out.Sources[j].Locator
	})
	sort.Slice(out.Dependencies, func(i, j int) bool {
		if out.Dependencies[i].Kind != out.Dependencies[j].Kind {
			return out.Dependencies[i].Kind < out.Dependencies[j].Kind
		}
		if out.Dependencies[i].ID != out.Dependencies[j].ID {
			return out.Dependencies[i].ID < out.Dependencies[j].ID
		}
		return out.Dependencies[i].Version < out.Dependencies[j].Version
	})
	sort.Slice(out.Parameters, func(i, j int) bool {
		return out.Parameters[i].Name < out.Parameters[j].Name
	})
	return out
}

func (p BuildPlan) Validate() error {
	if p.FormatVersion != BuildPlanFormatVersion {
		return fmt.Errorf("unsupported build plan format %d", p.FormatVersion)
	}
	if p.Plugin.ID == "" {
		return fmt.Errorf("build plan plugin id is required")
	}
	if _, err := core.ParseCanonicalID(string(p.Plugin.ID)); err != nil {
		return fmt.Errorf("build plan plugin id: %w", err)
	}
	if !core.ValidNumericVersion(p.Plugin.Version) {
		return fmt.Errorf("build plan plugin version %q must be major.minor.patch", p.Plugin.Version)
	}
	if err := validateHexFingerprint("plugin definition fingerprint", p.PluginDefinitionFingerprint); err != nil {
		return err
	}
	if len(p.Sources) == 0 {
		return fmt.Errorf("build plan requires at least one source")
	}
	sourceIDs := map[string]struct{}{}
	for _, s := range p.Sources {
		if strings.TrimSpace(s.Kind) == "" {
			return fmt.Errorf("build source kind is required")
		}
		if strings.TrimSpace(s.ID) == "" {
			return fmt.Errorf("build source id is required")
		}
		if _, exists := sourceIDs[s.ID]; exists {
			return fmt.Errorf("build source %q is declared more than once", s.ID)
		}
		sourceIDs[s.ID] = struct{}{}
		if strings.TrimSpace(s.Locator) == "" {
			return fmt.Errorf("build source %q locator is required", s.ID)
		}
		if err := validateBuildDigest("build source "+s.ID+" digest", s.Digest); err != nil {
			return err
		}
	}
	if err := p.Builder.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(p.Recipe.Kind) == "" {
		return fmt.Errorf("build recipe kind is required")
	}
	if strings.TrimSpace(p.Recipe.Entrypoint) == "" {
		return fmt.Errorf("build recipe entrypoint is required")
	}
	if strings.TrimSpace(string(p.OutputKind)) == "" {
		return fmt.Errorf("build output kind is required")
	}
	dependencies := map[string]struct{}{}
	for _, d := range p.Dependencies {
		if strings.TrimSpace(d.Kind) == "" {
			return fmt.Errorf("build dependency kind is required")
		}
		if strings.TrimSpace(d.ID) == "" {
			return fmt.Errorf("build dependency id is required")
		}
		if strings.TrimSpace(d.Version) == "" {
			return fmt.Errorf("build dependency %s version is required", d.ID)
		}
		key := d.Kind + "|" + d.ID + "|" + d.Version
		if _, exists := dependencies[key]; exists {
			return fmt.Errorf("build dependency %s is declared more than once", key)
		}
		dependencies[key] = struct{}{}
		if err := validateBuildDigest("build dependency "+d.ID+" artifact digest", d.ArtifactDigest); err != nil {
			return err
		}
	}
	if strings.TrimSpace(p.Target) == "" {
		return fmt.Errorf("build target is required")
	}
	parameters := map[string]struct{}{}
	for _, param := range p.Parameters {
		if strings.TrimSpace(param.Name) == "" {
			return fmt.Errorf("build parameter name is required")
		}
		if _, exists := parameters[param.Name]; exists {
			return fmt.Errorf("build parameter %q is declared more than once", param.Name)
		}
		parameters[param.Name] = struct{}{}
	}
	return nil
}

// BuildPlanInputFingerprint returns a stable fingerprint for every resolved
// build input except the exact Builder implementation identity (ID, version
// and artifact digest). Builder ContractVersion remains part of the
// fingerprint because it defines how the BuildPlan is interpreted. Bootstrap
// stage1/stage2 use this value to prove they are rebuilding the same target
// with the same contract, sources, recipe, dependencies and parameters while
// intentionally changing only the builder implementation that performs it.
func BuildPlanInputFingerprint(p BuildPlan) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	n := NormalizeBuildPlan(p)
	n.Builder = BuilderIdentity{ContractVersion: p.Builder.ContractVersion}
	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// BuildPlanFingerprint returns the stable fingerprint for the exact resolved
// build inputs. Slice declaration order does not affect the result.
func BuildPlanFingerprint(p BuildPlan) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	n := NormalizeBuildPlan(p)
	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func NormalizeBuildArtifactLock(l BuildArtifactLock) BuildArtifactLock {
	out := l
	out.Builds = append([]LockedBuild(nil), l.Builds...)
	sort.Slice(out.Builds, func(i, j int) bool {
		if out.Builds[i].Plugin.ID != out.Builds[j].Plugin.ID {
			return out.Builds[i].Plugin.ID < out.Builds[j].Plugin.ID
		}
		if out.Builds[i].Plugin.Version != out.Builds[j].Plugin.Version {
			return out.Builds[i].Plugin.Version < out.Builds[j].Plugin.Version
		}
		if out.Builds[i].OutputKind != out.Builds[j].OutputKind {
			return out.Builds[i].OutputKind < out.Builds[j].OutputKind
		}
		if out.Builds[i].Target != out.Builds[j].Target {
			return out.Builds[i].Target < out.Builds[j].Target
		}
		return out.Builds[i].Profile < out.Builds[j].Profile
	})
	return out
}

func (l BuildArtifactLock) Validate() error {
	if l.FormatVersion != BuildArtifactLockFormatVersion {
		return fmt.Errorf("unsupported build lock format %d", l.FormatVersion)
	}
	seen := map[string]struct{}{}
	for _, b := range l.Builds {
		if b.Plugin.ID == "" {
			return fmt.Errorf("locked build plugin id is required")
		}
		if _, err := core.ParseCanonicalID(string(b.Plugin.ID)); err != nil {
			return fmt.Errorf("locked build plugin id: %w", err)
		}
		if !core.ValidNumericVersion(b.Plugin.Version) {
			return fmt.Errorf("locked build plugin version %q must be major.minor.patch", b.Plugin.Version)
		}
		if strings.TrimSpace(string(b.OutputKind)) == "" {
			return fmt.Errorf("locked build output kind is required")
		}
		if strings.TrimSpace(b.Target) == "" {
			return fmt.Errorf("locked build target is required")
		}
		if err := validateHexFingerprint("build plan fingerprint", b.PlanFingerprint); err != nil {
			return err
		}
		if err := validateBuildDigest("locked build artifact digest", b.ArtifactDigest); err != nil {
			return err
		}
		key := string(b.Plugin.ID) + "@" + b.Plugin.Version + "|" + string(b.OutputKind) + "|" + b.Target + "|" + b.Profile
		if _, exists := seen[key]; exists {
			return fmt.Errorf("locked build %s is declared more than once", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// BuildArtifactLockFingerprint returns a stable fingerprint for exact
// build-plan and output-artifact bindings.
func BuildArtifactLockFingerprint(l BuildArtifactLock) (string, error) {
	if err := l.Validate(); err != nil {
		return "", err
	}
	n := NormalizeBuildArtifactLock(l)
	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// RealizeBuiltDefinition returns a derived Runtime definition that binds a
// produced artifact to an authoritative Plugin definition without mutating the
// source definition. If the source already pins an artifact, the produced
// artifact must match it exactly.
func RealizeBuiltDefinition(source Definition, artifactDigest string) (Definition, error) {
	if err := source.Artifact.Validate(); err != nil {
		return Definition{}, err
	}
	if err := validateBuildDigest("built plugin artifact digest", artifactDigest); err != nil {
		return Definition{}, err
	}
	if source.Artifact.Digest != "" && source.Artifact.Digest != artifactDigest {
		return Definition{}, fmt.Errorf(
			"built plugin artifact digest does not match source definition: got %s want %s",
			artifactDigest, source.Artifact.Digest,
		)
	}
	out := source
	out.Artifact.Digest = artifactDigest
	return out, nil
}

// LockBuildOutput binds one validated BuildPlan to an exact produced artifact.
// It does not add the artifact to Runtime Plugin Lock or grant any Runtime
// authority.
func LockBuildOutput(plan BuildPlan, artifactDigest string) (LockedBuild, error) {
	planFingerprint, err := BuildPlanFingerprint(plan)
	if err != nil {
		return LockedBuild{}, err
	}
	if err := validateBuildDigest("build output artifact digest", artifactDigest); err != nil {
		return LockedBuild{}, err
	}
	return LockedBuild{
		Plugin:          plan.Plugin,
		OutputKind:      plan.OutputKind,
		Target:          plan.Target,
		Profile:         plan.Profile,
		PlanFingerprint: planFingerprint,
		ArtifactDigest:  artifactDigest,
	}, nil
}

// BindRuntimeBuild records which exact BuildPlan produced the Runtime artifact
// selected by one Environment's Plugin Lock entry. Existing format-2 locks are
// accepted and upgraded to format 3. Other Environment entries are preserved.
func BindRuntimeBuild(lock Lock, environmentID core.CanonicalID, build LockedBuild) (Lock, error) {
	if !supportedLockFormatVersion(lock.FormatVersion) {
		return Lock{}, fmt.Errorf("unsupported plugin lock format %d", lock.FormatVersion)
	}
	if err := (BuildArtifactLock{
		FormatVersion: BuildArtifactLockFormatVersion,
		Builds:        []LockedBuild{build},
	}).Validate(); err != nil {
		return Lock{}, err
	}
	if build.OutputKind != BuildOutputRuntimeArtifact {
		return Lock{}, fmt.Errorf("runtime lock build binding requires %s output, got %s", BuildOutputRuntimeArtifact, build.OutputKind)
	}

	out := lock
	out.Environments = make([]EnvironmentLock, len(lock.Environments))
	foundEnvironment := false
	foundPlugin := false
	for i, environmentLock := range lock.Environments {
		out.Environments[i] = environmentLock
		out.Environments[i].Plugins = append([]LockedPlugin(nil), environmentLock.Plugins...)
		if environmentLock.Environment != environmentID {
			continue
		}
		foundEnvironment = true
		for j := range out.Environments[i].Plugins {
			locked := &out.Environments[i].Plugins[j]
			if locked.ID != build.Plugin.ID || locked.Version != build.Plugin.Version {
				continue
			}
			foundPlugin = true
			if locked.ArtifactDigest == "" {
				return Lock{}, fmt.Errorf(
					"runtime plugin %s@%s has no artifact digest to bind to build plan",
					locked.ID, locked.Version,
				)
			}
			if locked.ArtifactDigest != build.ArtifactDigest {
				return Lock{}, fmt.Errorf(
					"runtime plugin %s@%s artifact digest does not match build output",
					locked.ID, locked.Version,
				)
			}
			if locked.BuildPlanFingerprint != "" && locked.BuildPlanFingerprint != build.PlanFingerprint {
				return Lock{}, fmt.Errorf(
					"runtime plugin %s@%s is already bound to a different build plan",
					locked.ID, locked.Version,
				)
			}
			locked.BuildPlanFingerprint = build.PlanFingerprint
		}
	}
	if !foundEnvironment {
		return Lock{}, fmt.Errorf("plugin lock has no environment %s", environmentID)
	}
	if !foundPlugin {
		return Lock{}, fmt.Errorf(
			"plugin lock environment %s has no plugin %s@%s",
			environmentID, build.Plugin.ID, build.Plugin.Version,
		)
	}
	out.FormatVersion = BuildTraceLockFormatVersion
	return out, nil
}

// VerifyRuntimePluginBuild checks that a Runtime Plugin Lock entry can be
// traced back to this exact BuildPlan and locked build output. This is an
// explicit cross-check only; neither BuildPlan nor BuildArtifactLock is Runtime
// authority by itself.
func VerifyRuntimePluginBuild(plan BuildPlan, build LockedBuild, definition Definition, runtime LockedPlugin) error {
	expectedPlanFingerprint, err := BuildPlanFingerprint(plan)
	if err != nil {
		return err
	}
	if err := (BuildArtifactLock{
		FormatVersion: BuildArtifactLockFormatVersion,
		Builds:        []LockedBuild{build},
	}).Validate(); err != nil {
		return err
	}
	if definition.ID != plan.Plugin.ID || definition.Version != plan.Plugin.Version {
		return fmt.Errorf(
			"plugin definition %s@%s does not match build plan %s@%s",
			definition.ID, definition.Version, plan.Plugin.ID, plan.Plugin.Version,
		)
	}
	if plan.Plugin.ID != runtime.ID || plan.Plugin.Version != runtime.Version {
		return fmt.Errorf(
			"build plan plugin %s@%s does not match runtime plugin %s@%s",
			plan.Plugin.ID, plan.Plugin.Version, runtime.ID, runtime.Version,
		)
	}
	if plan.OutputKind != BuildOutputRuntimeArtifact {
		return fmt.Errorf("runtime plugin verification requires %s output, got %s", BuildOutputRuntimeArtifact, plan.OutputKind)
	}
	if build.OutputKind != plan.OutputKind {
		return fmt.Errorf("locked build output kind %s does not match build plan %s", build.OutputKind, plan.OutputKind)
	}
	if build.Plugin != plan.Plugin {
		return fmt.Errorf(
			"locked build plugin %s@%s does not match build plan %s@%s",
			build.Plugin.ID, build.Plugin.Version, plan.Plugin.ID, plan.Plugin.Version,
		)
	}
	if build.Target != plan.Target || build.Profile != plan.Profile {
		return fmt.Errorf("locked build target/profile does not match build plan")
	}
	if build.PlanFingerprint != expectedPlanFingerprint {
		return fmt.Errorf("locked build plan fingerprint diverged")
	}
	if BuildDefinitionFingerprint(definition) != plan.PluginDefinitionFingerprint {
		return fmt.Errorf("plugin definition fingerprint does not match build plan")
	}
	if DefinitionFingerprint(definition) != runtime.Fingerprint {
		return fmt.Errorf("runtime plugin descriptor fingerprint diverged from definition")
	}
	if definition.Artifact.Digest != runtime.ArtifactDigest {
		return fmt.Errorf("runtime plugin artifact digest diverged from definition")
	}
	if runtime.ArtifactDigest == "" {
		return fmt.Errorf("runtime plugin %s@%s has no artifact digest", runtime.ID, runtime.Version)
	}
	if runtime.ArtifactDigest != build.ArtifactDigest {
		return fmt.Errorf("runtime plugin artifact digest does not match locked build output")
	}
	if runtime.BuildPlanFingerprint == "" {
		return fmt.Errorf("runtime plugin %s@%s is not bound to a build plan", runtime.ID, runtime.Version)
	}
	if runtime.BuildPlanFingerprint != build.PlanFingerprint {
		return fmt.Errorf("runtime plugin build plan fingerprint does not match locked build output")
	}
	return nil
}

func validateHexFingerprint(label, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s must contain 64 hex characters", label)
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return fmt.Errorf("%s contains non-hex character", label)
		}
	}
	return nil
}

func validateBuildDigest(label, value string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("%s must use sha256:<64-hex>", label)
	}
	return validateHexFingerprint(label, strings.TrimPrefix(value, "sha256:"))
}
