package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	buildExecutionPlanFormatVersion = 1
	artifactSetFormatVersion         = 1
	buildExecutionResultFormatVersion = 1
	pluginSubjectBindingFormatVersion = 1
)

// BuildExecutionPlan is a Reference-only projection of exact build inputs.
// It deliberately excludes Plugin identity and Plugin definition identity.
type BuildExecutionPlan struct {
	FormatVersion  int                      `json:"formatVersion"`
	Sources        []plugin.BuildSource     `json:"sources"`
	Builder        plugin.BuilderIdentity   `json:"builder"`
	Recipe         plugin.BuildRecipe       `json:"recipe"`
	OutputContract string                   `json:"outputContract"`
	Dependencies   []plugin.BuildDependency `json:"dependencies,omitempty"`
	Target         string                   `json:"target"`
	Profile        string                   `json:"profile,omitempty"`
	Parameters     []plugin.BuildParameter  `json:"parameters,omitempty"`
}

// ArtifactMember is one physical output. Locator is intentionally excluded
// from content identity; it only describes where the exact bytes are found.
type ArtifactMember struct {
	Role    string `json:"role"`
	Format  string `json:"format"`
	Digest  string `json:"digest"`
	Locator string `json:"locator,omitempty"`
}

// ArtifactSet identifies the physical outputs of one build without embedding
// any domain subject identity.
type ArtifactSet struct {
	FormatVersion int              `json:"formatVersion"`
	Members       []ArtifactMember `json:"members"`
}

// BuildExecutionResult binds one exact execution plan to one exact content set.
// Same bytes from a different execution remain a different result identity.
type BuildExecutionResult struct {
	FormatVersion            int    `json:"formatVersion"`
	ExecutionPlanFingerprint string `json:"executionPlanFingerprint"`
	ArtifactSetFingerprint   string `json:"artifactSetFingerprint"`
}

// PluginSubjectBinding connects a logical Plugin to a domain-neutral build
// result. It grants no Acceptance or Runtime Authority.
type PluginSubjectBinding struct {
	FormatVersion               int             `json:"formatVersion"`
	Plugin                      plugin.Identity `json:"plugin"`
	PluginDefinitionFingerprint string          `json:"pluginDefinitionFingerprint"`
	ExecutionPlanFingerprint    string          `json:"executionPlanFingerprint"`
	ArtifactSetFingerprint      string          `json:"artifactSetFingerprint"`
	OutputRoles                 []string        `json:"outputRoles"`
}

func main() {}

func executionPlanFromPluginBuildPlan(plan plugin.BuildPlan) (BuildExecutionPlan, error) {
	if err := plan.Validate(); err != nil {
		return BuildExecutionPlan{}, err
	}
	normalized := plugin.NormalizeBuildPlan(plan)
	out := BuildExecutionPlan{
		FormatVersion:  buildExecutionPlanFormatVersion,
		Sources:        append([]plugin.BuildSource(nil), normalized.Sources...),
		Builder:        normalized.Builder,
		Recipe:         normalized.Recipe,
		OutputContract: string(normalized.OutputKind),
		Dependencies:   append([]plugin.BuildDependency(nil), normalized.Dependencies...),
		Target:         normalized.Target,
		Profile:        normalized.Profile,
		Parameters:     append([]plugin.BuildParameter(nil), normalized.Parameters...),
	}
	if err := out.Validate(); err != nil {
		return BuildExecutionPlan{}, err
	}
	return out, nil
}

func (p BuildExecutionPlan) Validate() error {
	if p.FormatVersion != buildExecutionPlanFormatVersion {
		return fmt.Errorf("unsupported build execution plan format %d", p.FormatVersion)
	}
	if len(p.Sources) == 0 {
		return fmt.Errorf("build execution plan requires at least one source")
	}
	sourceKeys := map[string]struct{}{}
	for _, source := range p.Sources {
		if strings.TrimSpace(source.Kind) == "" || strings.TrimSpace(source.ID) == "" {
			return fmt.Errorf("build execution source kind and id are required")
		}
		if strings.TrimSpace(source.Locator) == "" {
			return fmt.Errorf("build execution source %s locator is required", source.ID)
		}
		if err := validateDigest("build execution source "+source.ID, source.Digest); err != nil {
			return err
		}
		key := source.Kind + "\x00" + source.ID
		if _, duplicate := sourceKeys[key]; duplicate {
			return fmt.Errorf("build execution source %s/%s is declared more than once", source.Kind, source.ID)
		}
		sourceKeys[key] = struct{}{}
	}
	if err := p.Builder.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(p.Recipe.Kind) == "" || strings.TrimSpace(p.Recipe.Entrypoint) == "" {
		return fmt.Errorf("build execution recipe kind and entrypoint are required")
	}
	if strings.TrimSpace(p.OutputContract) == "" {
		return fmt.Errorf("build execution output contract is required")
	}
	dependencyKeys := map[string]struct{}{}
	for _, dependency := range p.Dependencies {
		if strings.TrimSpace(dependency.Kind) == "" ||
			strings.TrimSpace(dependency.ID) == "" ||
			strings.TrimSpace(dependency.Version) == "" {
			return fmt.Errorf("build execution dependency kind, id, and version are required")
		}
		if err := validateDigest("build execution dependency "+dependency.ID, dependency.ArtifactDigest); err != nil {
			return err
		}
		key := dependency.Kind + "\x00" + dependency.ID + "\x00" + dependency.Version
		if _, duplicate := dependencyKeys[key]; duplicate {
			return fmt.Errorf("build execution dependency %s/%s@%s is declared more than once", dependency.Kind, dependency.ID, dependency.Version)
		}
		dependencyKeys[key] = struct{}{}
	}
	if strings.TrimSpace(p.Target) == "" {
		return fmt.Errorf("build execution target is required")
	}
	parameters := map[string]struct{}{}
	for _, parameter := range p.Parameters {
		if strings.TrimSpace(parameter.Name) == "" {
			return fmt.Errorf("build execution parameter name is required")
		}
		if _, duplicate := parameters[parameter.Name]; duplicate {
			return fmt.Errorf("build execution parameter %s is declared more than once", parameter.Name)
		}
		parameters[parameter.Name] = struct{}{}
	}
	return nil
}

func normalizeBuildExecutionPlan(plan BuildExecutionPlan) BuildExecutionPlan {
	out := plan
	out.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
	out.Dependencies = append([]plugin.BuildDependency(nil), plan.Dependencies...)
	out.Parameters = append([]plugin.BuildParameter(nil), plan.Parameters...)
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

func buildExecutionPlanFingerprint(plan BuildExecutionPlan) (string, error) {
	if err := plan.Validate(); err != nil {
		return "", err
	}
	normalized := normalizeBuildExecutionPlan(plan)
	type identitySource struct {
		Kind   string `json:"kind"`
		ID     string `json:"id"`
		Digest string `json:"digest"`
	}
	sources := make([]identitySource, 0, len(normalized.Sources))
	for _, source := range normalized.Sources {
		sources = append(sources, identitySource{
			Kind:   source.Kind,
			ID:     source.ID,
			Digest: source.Digest,
		})
	}
	return fingerprintJSON(struct {
		FormatVersion  int                      `json:"formatVersion"`
		Sources        []identitySource         `json:"sources"`
		Builder        plugin.BuilderIdentity   `json:"builder"`
		Recipe         plugin.BuildRecipe       `json:"recipe"`
		OutputContract string                   `json:"outputContract"`
		Dependencies   []plugin.BuildDependency `json:"dependencies,omitempty"`
		Target         string                   `json:"target"`
		Profile        string                   `json:"profile,omitempty"`
		Parameters     []plugin.BuildParameter  `json:"parameters,omitempty"`
	}{
		FormatVersion:  normalized.FormatVersion,
		Sources:        sources,
		Builder:        normalized.Builder,
		Recipe:         normalized.Recipe,
		OutputContract: normalized.OutputContract,
		Dependencies:   normalized.Dependencies,
		Target:         normalized.Target,
		Profile:        normalized.Profile,
		Parameters:     normalized.Parameters,
	})
}

func (s ArtifactSet) Validate() error {
	if s.FormatVersion != artifactSetFormatVersion {
		return fmt.Errorf("unsupported artifact set format %d", s.FormatVersion)
	}
	if len(s.Members) == 0 {
		return fmt.Errorf("artifact set requires at least one member")
	}
	roles := map[string]struct{}{}
	for _, member := range s.Members {
		if strings.TrimSpace(member.Role) == "" {
			return fmt.Errorf("artifact member role is required")
		}
		if strings.TrimSpace(member.Format) == "" {
			return fmt.Errorf("artifact member %s format is required", member.Role)
		}
		if err := validateDigest("artifact member "+member.Role, member.Digest); err != nil {
			return err
		}
		if _, duplicate := roles[member.Role]; duplicate {
			return fmt.Errorf("artifact member role %s is declared more than once", member.Role)
		}
		roles[member.Role] = struct{}{}
	}
	return nil
}

func artifactSetFingerprint(set ArtifactSet) (string, error) {
	if err := set.Validate(); err != nil {
		return "", err
	}
	type identityMember struct {
		Role   string `json:"role"`
		Format string `json:"format"`
		Digest string `json:"digest"`
	}
	members := make([]identityMember, 0, len(set.Members))
	for _, member := range set.Members {
		members = append(members, identityMember{
			Role:   member.Role,
			Format: member.Format,
			Digest: member.Digest,
		})
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].Role != members[j].Role {
			return members[i].Role < members[j].Role
		}
		if members[i].Format != members[j].Format {
			return members[i].Format < members[j].Format
		}
		return members[i].Digest < members[j].Digest
	})
	return fingerprintJSON(struct {
		FormatVersion int              `json:"formatVersion"`
		Members       []identityMember `json:"members"`
	}{
		FormatVersion: set.FormatVersion,
		Members:       members,
	})
}

func (r BuildExecutionResult) Validate() error {
	if r.FormatVersion != buildExecutionResultFormatVersion {
		return fmt.Errorf("unsupported build execution result format %d", r.FormatVersion)
	}
	if err := validateFingerprint("execution plan fingerprint", r.ExecutionPlanFingerprint); err != nil {
		return err
	}
	if err := validateFingerprint("artifact set fingerprint", r.ArtifactSetFingerprint); err != nil {
		return err
	}
	return nil
}

func buildExecutionResultFingerprint(result BuildExecutionResult) (string, error) {
	if err := result.Validate(); err != nil {
		return "", err
	}
	return fingerprintJSON(result)
}

func (b PluginSubjectBinding) Validate() error {
	if b.FormatVersion != pluginSubjectBindingFormatVersion {
		return fmt.Errorf("unsupported plugin subject binding format %d", b.FormatVersion)
	}
	if b.Plugin.ID == "" {
		return fmt.Errorf("plugin subject binding id is required")
	}
	if _, err := core.ParseCanonicalID(string(b.Plugin.ID)); err != nil {
		return fmt.Errorf("plugin subject binding id: %w", err)
	}
	if !core.ValidNumericVersion(b.Plugin.Version) {
		return fmt.Errorf("plugin subject binding version %q must be major.minor.patch", b.Plugin.Version)
	}
	if err := validateFingerprint("plugin definition fingerprint", b.PluginDefinitionFingerprint); err != nil {
		return err
	}
	if err := validateFingerprint("execution plan fingerprint", b.ExecutionPlanFingerprint); err != nil {
		return err
	}
	if err := validateFingerprint("artifact set fingerprint", b.ArtifactSetFingerprint); err != nil {
		return err
	}
	if len(b.OutputRoles) == 0 {
		return fmt.Errorf("plugin subject binding requires at least one output role")
	}
	seen := map[string]struct{}{}
	for _, role := range b.OutputRoles {
		if strings.TrimSpace(role) == "" {
			return fmt.Errorf("plugin subject binding output role is required")
		}
		if _, duplicate := seen[role]; duplicate {
			return fmt.Errorf("plugin subject binding output role %s is declared more than once", role)
		}
		seen[role] = struct{}{}
	}
	return nil
}

func pluginSubjectBindingFingerprint(binding PluginSubjectBinding) (string, error) {
	if err := binding.Validate(); err != nil {
		return "", err
	}
	normalized := binding
	normalized.OutputRoles = append([]string(nil), binding.OutputRoles...)
	sort.Strings(normalized.OutputRoles)
	return fingerprintJSON(normalized)
}

func verifyPluginSubjectBinding(
	binding PluginSubjectBinding,
	plan BuildExecutionPlan,
	set ArtifactSet,
) error {
	if err := binding.Validate(); err != nil {
		return err
	}
	executionFingerprint, err := buildExecutionPlanFingerprint(plan)
	if err != nil {
		return err
	}
	if binding.ExecutionPlanFingerprint != executionFingerprint {
		return fmt.Errorf("plugin subject binding execution plan fingerprint does not match")
	}
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		return err
	}
	if binding.ArtifactSetFingerprint != setFingerprint {
		return fmt.Errorf("plugin subject binding artifact set fingerprint does not match")
	}
	roles := map[string]struct{}{}
	for _, member := range set.Members {
		roles[member.Role] = struct{}{}
	}
	for _, role := range binding.OutputRoles {
		if _, ok := roles[role]; !ok {
			return fmt.Errorf("plugin subject binding output role %s is not present in artifact set", role)
		}
	}
	return nil
}

func projectPluginBuild(
	plan plugin.BuildPlan,
	build plugin.LockedBuild,
) (BuildExecutionPlan, ArtifactSet, BuildExecutionResult, PluginSubjectBinding, error) {
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}
	if build.Plugin != plan.Plugin {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, fmt.Errorf("locked build Plugin does not match BuildPlan")
	}
	if build.OutputKind != plan.OutputKind || build.Target != plan.Target || build.Profile != plan.Profile {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, fmt.Errorf("locked build output coordinates do not match BuildPlan")
	}
	if build.PlanFingerprint != planFingerprint {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, fmt.Errorf("locked build plan fingerprint does not match BuildPlan")
	}
	if err := validateDigest("locked build artifact", build.ArtifactDigest); err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}

	executionPlan, err := executionPlanFromPluginBuildPlan(plan)
	if err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}
	executionFingerprint, err := buildExecutionPlanFingerprint(executionPlan)
	if err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{{
			Role:   "primary",
			Format: "application/octet-stream",
			Digest: build.ArtifactDigest,
		}},
	}
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}
	result := BuildExecutionResult{
		FormatVersion:            buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: executionFingerprint,
		ArtifactSetFingerprint:   setFingerprint,
	}
	if err := result.Validate(); err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}
	binding := PluginSubjectBinding{
		FormatVersion:               pluginSubjectBindingFormatVersion,
		Plugin:                      plan.Plugin,
		PluginDefinitionFingerprint: plan.PluginDefinitionFingerprint,
		ExecutionPlanFingerprint:    executionFingerprint,
		ArtifactSetFingerprint:      setFingerprint,
		OutputRoles:                 []string{"primary"},
	}
	if err := verifyPluginSubjectBinding(binding, executionPlan, set); err != nil {
		return BuildExecutionPlan{}, ArtifactSet{}, BuildExecutionResult{}, PluginSubjectBinding{}, err
	}
	return executionPlan, set, result, binding, nil
}

func fingerprintJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func validateFingerprint(label, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s must be 64 lowercase hex characters", label)
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return fmt.Errorf("%s must be 64 lowercase hex characters", label)
		}
	}
	return nil
}

func validateDigest(label, value string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("%s must use sha256:<64-hex>", label)
	}
	return validateFingerprint(label, strings.TrimPrefix(value, "sha256:"))
}
