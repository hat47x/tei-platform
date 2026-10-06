package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
)

const (
	BuildPlanFormatVersion = 1
	BuildLockFormatVersion = 1
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
	ID             core.CanonicalID `json:"id"`
	Version        string           `json:"version"`
	ArtifactDigest string           `json:"artifactDigest"`
}

// BuilderIdentity identifies the exact builder used to produce an artifact.
// The builder may itself be implemented by TEI or by another language/toolchain.
type BuilderIdentity struct {
	ID             core.CanonicalID `json:"id"`
	Version        string           `json:"version"`
	ArtifactDigest string           `json:"artifactDigest"`
}

// BuildParameter contains a non-secret input that may affect the build output.
// Secrets and signing material belong to separate authority/provenance paths and
// must not be embedded here.
type BuildParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// BuildPlan fixes the resolved inputs for one Plugin artifact build. It does not
// grant Acceptance or Runtime Authority and does not execute the build.
type BuildPlan struct {
	FormatVersion         int               `json:"formatVersion"`
	Plugin                Identity          `json:"plugin"`
	DescriptorFingerprint string            `json:"descriptorFingerprint"`
	Sources               []BuildSource     `json:"sources"`
	Builder               BuilderIdentity   `json:"builder"`
	Dependencies          []BuildDependency `json:"dependencies,omitempty"`
	Target                string            `json:"target"`
	Profile               string            `json:"profile,omitempty"`
	Parameters            []BuildParameter  `json:"parameters,omitempty"`
}

// LockedBuild binds one BuildPlan fingerprint to the artifact produced from it.
type LockedBuild struct {
	Plugin          Identity `json:"plugin"`
	Target          string   `json:"target"`
	Profile         string   `json:"profile,omitempty"`
	PlanFingerprint string   `json:"planFingerprint"`
	ArtifactDigest  string   `json:"artifactDigest"`
}

// BuildLock is separate from Plugin Lock. Plugin Lock answers which exact
// implementation is used at runtime; Build Lock answers which exact build plan
// produced a particular artifact.
type BuildLock struct {
	FormatVersion int           `json:"formatVersion"`
	Builds        []LockedBuild `json:"builds"`
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
	if !core.ValidNumericVersion(p.Plugin.Version) {
		return fmt.Errorf("build plan plugin version %q must be major.minor.patch", p.Plugin.Version)
	}
	if err := validateHexFingerprint("descriptor fingerprint", p.DescriptorFingerprint); err != nil {
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
	if p.Builder.ID == "" {
		return fmt.Errorf("builder id is required")
	}
	if strings.TrimSpace(p.Builder.Version) == "" {
		return fmt.Errorf("builder version is required")
	}
	if err := validateBuildDigest("builder artifact digest", p.Builder.ArtifactDigest); err != nil {
		return err
	}
	dependencies := map[core.CanonicalID]struct{}{}
	for _, d := range p.Dependencies {
		if d.ID == "" {
			return fmt.Errorf("build dependency id is required")
		}
		if _, exists := dependencies[d.ID]; exists {
			return fmt.Errorf("build dependency %s is declared more than once", d.ID)
		}
		dependencies[d.ID] = struct{}{}
		if strings.TrimSpace(d.Version) == "" {
			return fmt.Errorf("build dependency %s version is required", d.ID)
		}
		if err := validateBuildDigest("build dependency "+string(d.ID)+" artifact digest", d.ArtifactDigest); err != nil {
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

func NormalizeBuildLock(l BuildLock) BuildLock {
	out := l
	out.Builds = append([]LockedBuild(nil), l.Builds...)
	sort.Slice(out.Builds, func(i, j int) bool {
		if out.Builds[i].Plugin.ID != out.Builds[j].Plugin.ID {
			return out.Builds[i].Plugin.ID < out.Builds[j].Plugin.ID
		}
		if out.Builds[i].Plugin.Version != out.Builds[j].Plugin.Version {
			return out.Builds[i].Plugin.Version < out.Builds[j].Plugin.Version
		}
		if out.Builds[i].Target != out.Builds[j].Target {
			return out.Builds[i].Target < out.Builds[j].Target
		}
		return out.Builds[i].Profile < out.Builds[j].Profile
	})
	return out
}

func (l BuildLock) Validate() error {
	if l.FormatVersion != BuildLockFormatVersion {
		return fmt.Errorf("unsupported build lock format %d", l.FormatVersion)
	}
	seen := map[string]struct{}{}
	for _, b := range l.Builds {
		if b.Plugin.ID == "" {
			return fmt.Errorf("locked build plugin id is required")
		}
		if !core.ValidNumericVersion(b.Plugin.Version) {
			return fmt.Errorf("locked build plugin version %q must be major.minor.patch", b.Plugin.Version)
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
		key := string(b.Plugin.ID) + "@" + b.Plugin.Version + "|" + b.Target + "|" + b.Profile
		if _, exists := seen[key]; exists {
			return fmt.Errorf("locked build %s is declared more than once", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// BuildLockFingerprint returns a stable fingerprint for exact build-plan and
// output-artifact bindings.
func BuildLockFingerprint(l BuildLock) (string, error) {
	if err := l.Validate(); err != nil {
		return "", err
	}
	n := NormalizeBuildLock(l)
	b, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
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
