package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"tei/internal/core"
)

const PluginPackageFormatVersion = 1

// PluginPackageManifest is the first deterministic SB-1 package profile.
// It packages the normalized, artifact-free Plugin definition and binds it to
// the exact BuildPlan that materialized the package. Runtime implementation
// artifacts are deliberately outside this profile.
type PluginPackageManifest struct {
	FormatVersion               int        `json:"formatVersion"`
	Plugin                      Identity   `json:"plugin"`
	BuildPlanFingerprint        string     `json:"buildPlanFingerprint"`
	PluginDefinitionFingerprint string     `json:"pluginDefinitionFingerprint"`
	Definition                  Definition `json:"definition"`
}

// MaterializePluginPackage produces deterministic bytes for the first SB-1
// descriptor package profile. The authoritative source Definition must not
// already bind a produced Runtime artifact; SB-2 artifacts are a separate build
// output and authority path.
func MaterializePluginPackage(source Definition, plan BuildPlan) ([]byte, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if plan.OutputKind != BuildOutputPluginPackage {
		return nil, fmt.Errorf("plugin package materialization requires %s output, got %s", BuildOutputPluginPackage, plan.OutputKind)
	}
	if source.ID != plan.Plugin.ID || source.Version != plan.Plugin.Version {
		return nil, fmt.Errorf(
			"plugin package source %s@%s does not match build plan %s@%s",
			source.ID, source.Version, plan.Plugin.ID, plan.Plugin.Version,
		)
	}
	if _, err := core.ParseCanonicalID(string(source.ID)); err != nil {
		return nil, fmt.Errorf("plugin package source id: %w", err)
	}
	if !core.ValidNumericVersion(source.Version) {
		return nil, fmt.Errorf("plugin package source version %q must be major.minor.patch", source.Version)
	}
	if err := source.Artifact.Validate(); err != nil {
		return nil, err
	}
	if source.Artifact.Digest != "" || source.Artifact.SignatureRef != "" || source.Artifact.Provenance != "" {
		return nil, fmt.Errorf("SB-1 plugin package source must be artifact-free")
	}

	definitionFingerprint := BuildDefinitionFingerprint(source)
	if definitionFingerprint != plan.PluginDefinitionFingerprint {
		return nil, fmt.Errorf("plugin package definition fingerprint does not match build plan")
	}
	planFingerprint, err := BuildPlanFingerprint(plan)
	if err != nil {
		return nil, err
	}

	normalized := NormalizeDefinition(cloneBuildDefinition(source))
	normalized.Artifact = Artifact{}
	manifest := PluginPackageManifest{
		FormatVersion:               PluginPackageFormatVersion,
		Plugin:                      plan.Plugin,
		BuildPlanFingerprint:        planFingerprint,
		PluginDefinitionFingerprint: definitionFingerprint,
		Definition:                  normalized,
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// BuildOutputDigest returns the digest format used by BuildArtifactLock.
func BuildOutputDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
