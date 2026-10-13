package inputprofile

import (
	"encoding/json"
	"fmt"

	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

const (
	// PackageInputBindingSchema identifies observational input-binding evidence.
	// It grants neither build execution authority nor artifact acceptance.
	PackageInputBindingSchema      = "tei.plugin-package-input-binding/v1"
	PackageOutputObservationSchema = "tei.plugin-package-output-observation/v1"
)

// BuildInputSelection must be supplied by a trusted Host after resolving the
// application's explicit input profile, Source lock and exact BuildPlan lock.
// Values copied from an untrusted request do not constitute Host approval.
type BuildInputSelection struct {
	Profile                    string
	LockedSourceFingerprint    string
	LockedBuildPlanFingerprint string
}

// BuildInputEvidence binds the two immutable inputs of the shipped SB-1 Source
// to an explicitly selected normalization profile. It is observational only.
type BuildInputEvidence struct {
	Schema                      string          `json:"schema"`
	Profile                     string          `json:"profile"`
	SourceFingerprint           string          `json:"sourceFingerprint"`
	BuildPlanFingerprint        string          `json:"buildPlanFingerprint"`
	Plugin                      plugin.Identity `json:"plugin"`
	PluginDefinitionFingerprint string          `json:"pluginDefinitionFingerprint"`
	OriginalInputDigest         string          `json:"originalInputDigest"`
	CanonicalInputDigest        string          `json:"canonicalInputDigest"`
}

type PreparedPackageInputs struct {
	Inputs   map[string]transform.Value
	Evidence BuildInputEvidence
}

// PreparePluginPackageInputs is the explicit Host-facing boundary for the v2
// ingress profile. Neither the legacy transform.Evaluate path nor the shipped
// SB-1 Source invokes it implicitly. The Host must separately authorize the
// exact build and verify the Source lock's provenance before calling it.
func PreparePluginPackageInputs(selection BuildInputSelection, source transform.Source, plan plugin.BuildPlan, rawDefinition []byte) (PreparedPackageInputs, error) {
	if selection.Profile != PluginDefinitionJSONV2 {
		return PreparedPackageInputs{}, fmt.Errorf("unsupported package input profile %q", selection.Profile)
	}
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return PreparedPackageInputs{}, fmt.Errorf("invalid exact BuildPlan: %w", err)
	}
	if plan.OutputKind != plugin.BuildOutputPluginPackage {
		return PreparedPackageInputs{}, fmt.Errorf("package input binding requires plugin-package output kind")
	}
	if selection.LockedBuildPlanFingerprint == "" || selection.LockedBuildPlanFingerprint != planFingerprint {
		return PreparedPackageInputs{}, fmt.Errorf("Host-approved BuildPlan fingerprint does not match exact plan")
	}
	definitionValue, ingress, err := CanonicalizePluginDefinition(selection.Profile, source, selection.LockedSourceFingerprint, rawDefinition)
	if err != nil {
		return PreparedPackageInputs{}, err
	}
	definitionRaw, err := definitionValue.JSON()
	if err != nil {
		return PreparedPackageInputs{}, fmt.Errorf("encode canonical Plugin definition: %w", err)
	}
	var definition plugin.Definition
	if err := json.Unmarshal(definitionRaw, &definition); err != nil {
		return PreparedPackageInputs{}, fmt.Errorf("decode canonical Plugin definition: %w", err)
	}
	if definition.ID != plan.Plugin.ID || definition.Version != plan.Plugin.Version {
		return PreparedPackageInputs{}, fmt.Errorf("canonical Plugin identity does not match exact BuildPlan")
	}
	definitionFingerprint := plugin.BuildDefinitionFingerprint(definition)
	if definitionFingerprint != plan.PluginDefinitionFingerprint {
		return PreparedPackageInputs{}, fmt.Errorf("canonical Plugin definition fingerprint does not match exact BuildPlan")
	}
	planRaw, err := json.Marshal(plan)
	if err != nil {
		return PreparedPackageInputs{}, fmt.Errorf("encode exact BuildPlan: %w", err)
	}
	planValue, err := transform.ParseJSON(planRaw)
	if err != nil {
		return PreparedPackageInputs{}, fmt.Errorf("parse exact BuildPlan: %w", err)
	}
	return PreparedPackageInputs{
		Inputs: map[string]transform.Value{
			"buildPlan":        planValue,
			"pluginDefinition": definitionValue,
		},
		Evidence: BuildInputEvidence{
			Schema:                      PackageInputBindingSchema,
			Profile:                     selection.Profile,
			SourceFingerprint:           ingress.SourceFingerprint,
			BuildPlanFingerprint:        planFingerprint,
			Plugin:                      plan.Plugin,
			PluginDefinitionFingerprint: definitionFingerprint,
			OriginalInputDigest:         ingress.OriginalInputDigest,
			CanonicalInputDigest:        ingress.CanonicalInputDigest,
		},
	}, nil
}

// PackageOutputObservation relates an actual evaluation output to the exact
// admitted inputs. It is not a signed execution receipt, an ArtifactLock,
// proof of Host authorization, or permission to deploy the output.
type PackageOutputObservation struct {
	Schema       string             `json:"schema"`
	Input        BuildInputEvidence `json:"input"`
	OutputDigest string             `json:"outputDigest"`
	Steps        int                `json:"steps"`
}

// ObservePackageOutput checks the returned manifest's input bindings and
// records its digest. Callers must authenticate any external engine response
// separately; self-reported bytes cannot establish execution provenance.
func ObservePackageOutput(binding BuildInputEvidence, result transform.EvaluationResult) (PackageOutputObservation, error) {
	if binding.Schema != PackageInputBindingSchema ||
		binding.Profile != PluginDefinitionJSONV2 ||
		binding.SourceFingerprint == "" ||
		binding.BuildPlanFingerprint == "" ||
		binding.PluginDefinitionFingerprint == "" ||
		binding.OriginalInputDigest == "" ||
		binding.CanonicalInputDigest == "" {
		return PackageOutputObservation{}, fmt.Errorf("invalid Plugin package input-binding evidence")
	}
	if result.Output.Kind != transform.ValueBytes || result.Steps < 1 {
		return PackageOutputObservation{}, fmt.Errorf("Plugin package evaluation did not return verified output bytes")
	}
	var manifest plugin.PluginPackageManifest
	if err := json.Unmarshal(result.Output.Bytes, &manifest); err != nil {
		return PackageOutputObservation{}, fmt.Errorf("Plugin package output manifest: %w", err)
	}
	if manifest.FormatVersion != plugin.PluginPackageFormatVersion ||
		manifest.Plugin != binding.Plugin ||
		manifest.BuildPlanFingerprint != binding.BuildPlanFingerprint ||
		manifest.PluginDefinitionFingerprint != binding.PluginDefinitionFingerprint ||
		plugin.BuildDefinitionFingerprint(manifest.Definition) != binding.PluginDefinitionFingerprint {
		return PackageOutputObservation{}, fmt.Errorf("Plugin package output does not match admitted input fingerprints")
	}
	return PackageOutputObservation{
		Schema:       PackageOutputObservationSchema,
		Input:        binding,
		OutputDigest: plugin.BuildOutputDigest(result.Output.Bytes),
		Steps:        result.Steps,
	}, nil
}
