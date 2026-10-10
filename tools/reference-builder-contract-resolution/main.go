package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	builderResolutionLockFormatVersion = 1
	scopeBuildTool                     = "build-tool"
	referenceSchema                    = "tei.reference.builder-contract-resolution/v0"
)

var authorityLimits = []string{
	"builder-resolution-is-not-build-execution",
	"builder-resolution-is-not-build-acceptance",
	"builder-resolution-is-not-runtime-authority",
	"builder-resolution-is-not-provider-trust",
	"locked-replay-is-not-fresh-resolution",
}

type BuilderContract struct {
	ID                 string `json:"id"`
	Version            string `json:"version"`
	InputSemantics     string `json:"inputSemantics"`
	OutputSemantics    string `json:"outputSemantics"`
	FailureSemantics   string `json:"failureSemantics"`
	ConformanceProfile string `json:"conformanceProfile"`
}

type BuilderIntent struct {
	ContractID      string `json:"contractId"`
	ContractVersion string `json:"contractVersion"`
	RecipeKind      string `json:"recipeKind"`
	Target          string `json:"target"`
	Profile         string `json:"profile"`
}

type BuilderProvider struct {
	ProviderID          core.CanonicalID `json:"providerId"`
	ProviderVersion     string           `json:"providerVersion"`
	ImplementationID    string           `json:"implementationId"`
	ArtifactDigest      string           `json:"artifactDigest"`
	Scope               string           `json:"scope"`
	ContractID          string           `json:"contractId"`
	ContractVersion     string           `json:"contractVersion"`
	ContractFingerprint string           `json:"contractFingerprint"`
	ConformanceProfile  string           `json:"conformanceProfile"`
	RecipeKinds         []string         `json:"recipeKinds"`
	Targets             []string         `json:"targets"`
	Profiles            []string         `json:"profiles"`
}

type BuilderConformanceEvidence struct {
	ProviderID          core.CanonicalID `json:"providerId"`
	ProviderVersion     string           `json:"providerVersion"`
	ImplementationID    string           `json:"implementationId"`
	ArtifactDigest      string           `json:"artifactDigest"`
	ContractFingerprint string           `json:"contractFingerprint"`
	ConformanceProfile  string           `json:"conformanceProfile"`
	RecipeKind          string           `json:"recipeKind"`
	Target              string           `json:"target"`
	Profile             string           `json:"profile"`
	EvidenceFingerprint string           `json:"evidenceFingerprint"`
}

type BuilderWiring struct {
	ContractID       string           `json:"contractId"`
	ProviderID       core.CanonicalID `json:"providerId"`
	ProviderVersion  string           `json:"providerVersion"`
	ImplementationID string           `json:"implementationId"`
	ArtifactDigest   string           `json:"artifactDigest,omitempty"`
}

type BuilderCatalog struct {
	Contracts []BuilderContract            `json:"contracts"`
	Providers []BuilderProvider            `json:"providers"`
	Evidence  []BuilderConformanceEvidence `json:"evidence"`
}

type BuilderResolution struct {
	IntentFingerprint     string                 `json:"intentFingerprint"`
	Contract              BuilderContract        `json:"contract"`
	ContractFingerprint   string                 `json:"contractFingerprint"`
	Provider              BuilderProvider        `json:"provider"`
	EvidenceFingerprint   string                 `json:"evidenceFingerprint"`
	Builder               plugin.BuilderIdentity `json:"builder"`
	ResolutionFingerprint string                 `json:"resolutionFingerprint"`
}

type BuilderResolutionLock struct {
	FormatVersion         int                    `json:"formatVersion"`
	IntentFingerprint     string                 `json:"intentFingerprint"`
	ContractID            string                 `json:"contractId"`
	ContractVersion       string                 `json:"contractVersion"`
	ContractFingerprint   string                 `json:"contractFingerprint"`
	ProviderID            core.CanonicalID       `json:"providerId"`
	ProviderVersion       string                 `json:"providerVersion"`
	ImplementationID      string                 `json:"implementationId"`
	ArtifactDigest        string                 `json:"artifactDigest"`
	EvidenceFingerprint   string                 `json:"evidenceFingerprint"`
	Builder               plugin.BuilderIdentity `json:"builder"`
	ResolutionFingerprint string                 `json:"resolutionFingerprint"`
}

type ReferenceEvidence struct {
	Schema                string                `json:"schema"`
	FreshResolution       BuilderResolution     `json:"freshResolution"`
	Lock                  BuilderResolutionLock `json:"lock"`
	ReplayResolution      BuilderResolution     `json:"replayResolution"`
	ExpandedFreshError    string                `json:"expandedFreshError"`
	AuthorityLimits       []string              `json:"authorityLimits"`
}

func main() {
	contract, providers, evidence, wiring := referenceFixture()
	intent := BuilderIntent{
		ContractID:      contract.ID,
		ContractVersion: contract.Version,
		RecipeKind:      "deterministic-copy",
		Target:          "host/reference",
		Profile:         "reference",
	}
	catalog := BuilderCatalog{
		Contracts: []BuilderContract{contract},
		Providers: []BuilderProvider{providers[0]},
		Evidence:  []BuilderConformanceEvidence{evidence[0]},
	}
	resolution, err := resolveBuilder(intent, catalog, &wiring)
	if err != nil {
		panic(err)
	}
	lock, err := lockBuilderResolution(resolution)
	if err != nil {
		panic(err)
	}

	expanded := catalog
	expanded.Providers = append(append([]BuilderProvider(nil), catalog.Providers...), providers[1])
	expanded.Evidence = append(append([]BuilderConformanceEvidence(nil), catalog.Evidence...), evidence[1])
	replay, err := replayBuilderResolution(intent, expanded, lock)
	if err != nil {
		panic(err)
	}
	_, freshErr := resolveBuilder(intent, expanded, nil)
	if freshErr == nil {
		panic("expanded fresh resolution unexpectedly remained unambiguous")
	}

	out := ReferenceEvidence{
		Schema:             referenceSchema,
		FreshResolution:    resolution,
		Lock:               lock,
		ReplayResolution:   replay,
		ExpandedFreshError: freshErr.Error(),
		AuthorityLimits:    append([]string(nil), authorityLimits...),
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		panic(err)
	}
}

func resolveBuilder(
	intent BuilderIntent,
	catalog BuilderCatalog,
	wiring *BuilderWiring,
) (BuilderResolution, error) {
	if err := validateIntent(intent); err != nil {
		return BuilderResolution{}, err
	}
	contract, err := exactContract(catalog.Contracts, intent.ContractID, intent.ContractVersion)
	if err != nil {
		return BuilderResolution{}, err
	}
	contractFingerprint, err := builderContractFingerprint(contract)
	if err != nil {
		return BuilderResolution{}, err
	}
	intentFingerprint, err := builderIntentFingerprint(intent)
	if err != nil {
		return BuilderResolution{}, err
	}

	if wiring != nil {
		if err := validateWiring(*wiring); err != nil {
			return BuilderResolution{}, err
		}
		if wiring.ContractID != intent.ContractID {
			return BuilderResolution{}, fmt.Errorf("builder wiring contract %s does not match intent contract %s", wiring.ContractID, intent.ContractID)
		}
		matches := matchingProviders(catalog.Providers, *wiring)
		switch len(matches) {
		case 0:
			return BuilderResolution{}, fmt.Errorf("wired builder provider is not present")
		case 1:
			provider := matches[0]
			evidence, err := validateEligibleProvider(intent, contract, contractFingerprint, provider, catalog.Evidence)
			if err != nil {
				return BuilderResolution{}, fmt.Errorf("wired builder provider is not eligible: %w", err)
			}
			return buildResolution(intentFingerprint, contract, contractFingerprint, provider, evidence)
		default:
			return BuilderResolution{}, fmt.Errorf("explicit builder wiring is ambiguous across physical artifacts")
		}
	}

	type candidate struct {
		provider BuilderProvider
		evidence BuilderConformanceEvidence
	}
	var candidates []candidate
	for _, provider := range catalog.Providers {
		evidence, err := validateEligibleProvider(intent, contract, contractFingerprint, provider, catalog.Evidence)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate{provider: provider, evidence: evidence})
	}
	switch len(candidates) {
	case 0:
		return BuilderResolution{}, fmt.Errorf("no conformant build-tool provider for %s@%s", intent.ContractID, intent.ContractVersion)
	case 1:
		return buildResolution(intentFingerprint, contract, contractFingerprint, candidates[0].provider, candidates[0].evidence)
	default:
		return BuilderResolution{}, fmt.Errorf("builder provider is ambiguous for %s@%s", intent.ContractID, intent.ContractVersion)
	}
}

func lockBuilderResolution(resolution BuilderResolution) (BuilderResolutionLock, error) {
	if err := validateResolution(resolution); err != nil {
		return BuilderResolutionLock{}, err
	}
	return BuilderResolutionLock{
		FormatVersion:         builderResolutionLockFormatVersion,
		IntentFingerprint:     resolution.IntentFingerprint,
		ContractID:            resolution.Contract.ID,
		ContractVersion:       resolution.Contract.Version,
		ContractFingerprint:   resolution.ContractFingerprint,
		ProviderID:            resolution.Provider.ProviderID,
		ProviderVersion:       resolution.Provider.ProviderVersion,
		ImplementationID:      resolution.Provider.ImplementationID,
		ArtifactDigest:        canonicalDigest(resolution.Provider.ArtifactDigest),
		EvidenceFingerprint:   canonicalFingerprint(resolution.EvidenceFingerprint),
		Builder:               canonicalBuilderIdentity(resolution.Builder),
		ResolutionFingerprint: canonicalFingerprint(resolution.ResolutionFingerprint),
	}, nil
}

func replayBuilderResolution(
	intent BuilderIntent,
	catalog BuilderCatalog,
	lock BuilderResolutionLock,
) (BuilderResolution, error) {
	if err := validateLock(lock); err != nil {
		return BuilderResolution{}, err
	}
	intentFingerprint, err := builderIntentFingerprint(intent)
	if err != nil {
		return BuilderResolution{}, err
	}
	if canonicalFingerprint(intentFingerprint) != canonicalFingerprint(lock.IntentFingerprint) {
		return BuilderResolution{}, fmt.Errorf("builder resolution lock intent fingerprint does not match")
	}
	contract, err := exactContract(catalog.Contracts, lock.ContractID, lock.ContractVersion)
	if err != nil {
		return BuilderResolution{}, err
	}
	contractFingerprint, err := builderContractFingerprint(contract)
	if err != nil {
		return BuilderResolution{}, err
	}
	if canonicalFingerprint(contractFingerprint) != canonicalFingerprint(lock.ContractFingerprint) {
		return BuilderResolution{}, fmt.Errorf("builder resolution lock contract fingerprint does not match")
	}

	var providers []BuilderProvider
	for _, provider := range catalog.Providers {
		if provider.ProviderID == lock.ProviderID &&
			provider.ProviderVersion == lock.ProviderVersion &&
			provider.ImplementationID == lock.ImplementationID &&
			digestEqual(provider.ArtifactDigest, lock.ArtifactDigest) {
			providers = append(providers, provider)
		}
	}
	if len(providers) != 1 {
		return BuilderResolution{}, fmt.Errorf("locked builder provider must resolve to exactly one physical provider, got %d", len(providers))
	}
	evidence, err := validateEligibleProvider(intent, contract, contractFingerprint, providers[0], catalog.Evidence)
	if err != nil {
		return BuilderResolution{}, fmt.Errorf("locked builder provider is no longer eligible: %w", err)
	}
	if canonicalFingerprint(evidence.EvidenceFingerprint) != canonicalFingerprint(lock.EvidenceFingerprint) {
		return BuilderResolution{}, fmt.Errorf("builder resolution lock conformance Evidence fingerprint does not match")
	}
	resolution, err := buildResolution(intentFingerprint, contract, contractFingerprint, providers[0], evidence)
	if err != nil {
		return BuilderResolution{}, err
	}
	if canonicalBuilderIdentity(resolution.Builder) != canonicalBuilderIdentity(lock.Builder) {
		return BuilderResolution{}, fmt.Errorf("builder resolution lock exact BuilderIdentity does not match")
	}
	if canonicalFingerprint(resolution.ResolutionFingerprint) != canonicalFingerprint(lock.ResolutionFingerprint) {
		return BuilderResolution{}, fmt.Errorf("builder resolution lock fingerprint does not match replay")
	}
	return resolution, nil
}

func buildResolution(
	intentFingerprint string,
	contract BuilderContract,
	contractFingerprint string,
	provider BuilderProvider,
	evidence BuilderConformanceEvidence,
) (BuilderResolution, error) {
	builder := plugin.BuilderIdentity{
		ID:              provider.ProviderID,
		Version:         provider.ProviderVersion,
		ContractVersion: contract.ID + "@" + contract.Version,
		ArtifactDigest:  canonicalDigest(provider.ArtifactDigest),
	}
	if err := builder.Validate(); err != nil {
		return BuilderResolution{}, err
	}
	resolution := BuilderResolution{
		IntentFingerprint:   canonicalFingerprint(intentFingerprint),
		Contract:            normalizeContract(contract),
		ContractFingerprint: canonicalFingerprint(contractFingerprint),
		Provider:            normalizeProvider(provider),
		EvidenceFingerprint: canonicalFingerprint(evidence.EvidenceFingerprint),
		Builder:             builder,
	}
	fingerprint, err := resolutionFingerprint(resolution)
	if err != nil {
		return BuilderResolution{}, err
	}
	resolution.ResolutionFingerprint = fingerprint
	return resolution, nil
}

func validateResolution(resolution BuilderResolution) error {
	if err := validateFingerprint("intent fingerprint", resolution.IntentFingerprint); err != nil {
		return err
	}
	if err := validateContract(resolution.Contract); err != nil {
		return err
	}
	if err := validateFingerprint("contract fingerprint", resolution.ContractFingerprint); err != nil {
		return err
	}
	expectedContractFingerprint, err := builderContractFingerprint(resolution.Contract)
	if err != nil {
		return err
	}
	if canonicalFingerprint(expectedContractFingerprint) != canonicalFingerprint(resolution.ContractFingerprint) {
		return fmt.Errorf("resolution contract fingerprint does not match Contract")
	}
	if err := validateProvider(resolution.Provider); err != nil {
		return err
	}
	if resolution.Provider.Scope != scopeBuildTool {
		return fmt.Errorf("resolution provider scope %q is not build-tool", resolution.Provider.Scope)
	}
	if resolution.Provider.ContractID != resolution.Contract.ID ||
		resolution.Provider.ContractVersion != resolution.Contract.Version ||
		canonicalFingerprint(resolution.Provider.ContractFingerprint) != canonicalFingerprint(resolution.ContractFingerprint) ||
		resolution.Provider.ConformanceProfile != resolution.Contract.ConformanceProfile {
		return fmt.Errorf("resolution Provider does not match Contract")
	}
	if err := validateFingerprint("Evidence fingerprint", resolution.EvidenceFingerprint); err != nil {
		return err
	}
	if err := resolution.Builder.Validate(); err != nil {
		return err
	}
	expectedBuilder := plugin.BuilderIdentity{
		ID:              resolution.Provider.ProviderID,
		Version:         resolution.Provider.ProviderVersion,
		ContractVersion: resolution.Contract.ID + "@" + resolution.Contract.Version,
		ArtifactDigest:  canonicalDigest(resolution.Provider.ArtifactDigest),
	}
	if canonicalBuilderIdentity(resolution.Builder) != canonicalBuilderIdentity(expectedBuilder) {
		return fmt.Errorf("resolution BuilderIdentity does not match selected Provider")
	}
	if err := validateFingerprint("resolution fingerprint", resolution.ResolutionFingerprint); err != nil {
		return err
	}
	expectedResolutionFingerprint, err := resolutionFingerprint(resolution)
	if err != nil {
		return err
	}
	if canonicalFingerprint(expectedResolutionFingerprint) != canonicalFingerprint(resolution.ResolutionFingerprint) {
		return fmt.Errorf("resolution fingerprint does not match resolution fields")
	}
	return nil
}

func validateLock(lock BuilderResolutionLock) error {
	if lock.FormatVersion != builderResolutionLockFormatVersion {
		return fmt.Errorf("unsupported builder resolution lock format %d", lock.FormatVersion)
	}
	if err := validateFingerprint("lock intent fingerprint", lock.IntentFingerprint); err != nil {
		return err
	}
	if _, err := core.ParseCanonicalID(lock.ContractID); err != nil {
		return fmt.Errorf("lock contract id: %w", err)
	}
	if !core.ValidNumericVersion(lock.ContractVersion) {
		return fmt.Errorf("lock contract version %q must be major.minor.patch", lock.ContractVersion)
	}
	if err := validateFingerprint("lock contract fingerprint", lock.ContractFingerprint); err != nil {
		return err
	}
	if lock.ProviderID == "" || !core.ValidNumericVersion(lock.ProviderVersion) || strings.TrimSpace(lock.ImplementationID) == "" {
		return fmt.Errorf("invalid locked builder provider identity")
	}
	if _, err := core.ParseCanonicalID(string(lock.ProviderID)); err != nil {
		return fmt.Errorf("lock provider id: %w", err)
	}
	if _, err := canonicalDigestChecked(lock.ArtifactDigest); err != nil {
		return err
	}
	if err := validateFingerprint("lock Evidence fingerprint", lock.EvidenceFingerprint); err != nil {
		return err
	}
	if err := lock.Builder.Validate(); err != nil {
		return err
	}
	expectedBuilder := plugin.BuilderIdentity{
		ID:              lock.ProviderID,
		Version:         lock.ProviderVersion,
		ContractVersion: lock.ContractID + "@" + lock.ContractVersion,
		ArtifactDigest:  canonicalDigest(lock.ArtifactDigest),
	}
	if canonicalBuilderIdentity(lock.Builder) != canonicalBuilderIdentity(expectedBuilder) {
		return fmt.Errorf("lock BuilderIdentity does not match locked Provider")
	}
	if err := validateFingerprint("lock resolution fingerprint", lock.ResolutionFingerprint); err != nil {
		return err
	}
	return nil
}

func validateEligibleProvider(
	intent BuilderIntent,
	contract BuilderContract,
	contractFingerprint string,
	provider BuilderProvider,
	evidence []BuilderConformanceEvidence,
) (BuilderConformanceEvidence, error) {
	if err := validateProvider(provider); err != nil {
		return BuilderConformanceEvidence{}, err
	}
	if provider.Scope != scopeBuildTool {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider scope %q is not build-tool", provider.Scope)
	}
	if provider.ContractID != contract.ID || provider.ContractVersion != contract.Version {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider contract does not match")
	}
	if canonicalFingerprint(provider.ContractFingerprint) != canonicalFingerprint(contractFingerprint) {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider contract fingerprint drifted")
	}
	if provider.ConformanceProfile != contract.ConformanceProfile {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider conformance profile does not match")
	}
	if !contains(provider.RecipeKinds, intent.RecipeKind) {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider does not support recipe %s", intent.RecipeKind)
	}
	if !contains(provider.Targets, intent.Target) {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider does not support target %s", intent.Target)
	}
	if !contains(provider.Profiles, intent.Profile) {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider does not support profile %s", intent.Profile)
	}

	var matches []BuilderConformanceEvidence
	for _, item := range evidence {
		if item.ProviderID == provider.ProviderID &&
			item.ProviderVersion == provider.ProviderVersion &&
			item.ImplementationID == provider.ImplementationID &&
			digestEqual(item.ArtifactDigest, provider.ArtifactDigest) &&
			canonicalFingerprint(item.ContractFingerprint) == canonicalFingerprint(contractFingerprint) &&
			item.ConformanceProfile == contract.ConformanceProfile &&
			item.RecipeKind == intent.RecipeKind &&
			item.Target == intent.Target &&
			item.Profile == intent.Profile {
			if err := validateEvidence(item); err != nil {
				return BuilderConformanceEvidence{}, err
			}
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 {
		return BuilderConformanceEvidence{}, fmt.Errorf("provider requires exactly one matching Conformance Evidence, got %d", len(matches))
	}
	return matches[0], nil
}

func matchingProviders(providers []BuilderProvider, wiring BuilderWiring) []BuilderProvider {
	var out []BuilderProvider
	for _, provider := range providers {
		if provider.ContractID != wiring.ContractID ||
			provider.ProviderID != wiring.ProviderID ||
			provider.ProviderVersion != wiring.ProviderVersion ||
			provider.ImplementationID != wiring.ImplementationID {
			continue
		}
		if wiring.ArtifactDigest != "" && !digestEqual(provider.ArtifactDigest, wiring.ArtifactDigest) {
			continue
		}
		out = append(out, provider)
	}
	return out
}

func exactContract(contracts []BuilderContract, id, version string) (BuilderContract, error) {
	var matches []BuilderContract
	for _, contract := range contracts {
		if err := validateContract(contract); err != nil {
			return BuilderContract{}, err
		}
		if contract.ID == id && contract.Version == version {
			matches = append(matches, contract)
		}
	}
	if len(matches) != 1 {
		return BuilderContract{}, fmt.Errorf("builder contract %s@%s must resolve exactly once, got %d", id, version, len(matches))
	}
	return matches[0], nil
}

func validateWiring(wiring BuilderWiring) error {
	if _, err := core.ParseCanonicalID(wiring.ContractID); err != nil {
		return fmt.Errorf("builder wiring contract id: %w", err)
	}
	if wiring.ProviderID == "" {
		return fmt.Errorf("builder wiring provider id is required")
	}
	if _, err := core.ParseCanonicalID(string(wiring.ProviderID)); err != nil {
		return fmt.Errorf("builder wiring provider id: %w", err)
	}
	if !core.ValidNumericVersion(wiring.ProviderVersion) {
		return fmt.Errorf("builder wiring provider version %q must be major.minor.patch", wiring.ProviderVersion)
	}
	if strings.TrimSpace(wiring.ImplementationID) == "" {
		return fmt.Errorf("builder wiring implementation id is required")
	}
	if wiring.ArtifactDigest != "" {
		if _, err := canonicalDigestChecked(wiring.ArtifactDigest); err != nil {
			return fmt.Errorf("builder wiring artifact digest: %w", err)
		}
	}
	return nil
}

func validateIntent(intent BuilderIntent) error {
	if _, err := core.ParseCanonicalID(intent.ContractID); err != nil {
		return fmt.Errorf("builder intent contract id: %w", err)
	}
	if !core.ValidNumericVersion(intent.ContractVersion) {
		return fmt.Errorf("builder intent contract version %q must be major.minor.patch", intent.ContractVersion)
	}
	if strings.TrimSpace(intent.RecipeKind) == "" ||
		strings.TrimSpace(intent.Target) == "" ||
		strings.TrimSpace(intent.Profile) == "" {
		return fmt.Errorf("builder intent recipe, target, and profile are required")
	}
	return nil
}

func validateContract(contract BuilderContract) error {
	if _, err := core.ParseCanonicalID(contract.ID); err != nil {
		return fmt.Errorf("builder contract id: %w", err)
	}
	if !core.ValidNumericVersion(contract.Version) {
		return fmt.Errorf("builder contract version %q must be major.minor.patch", contract.Version)
	}
	if strings.TrimSpace(contract.InputSemantics) == "" ||
		strings.TrimSpace(contract.OutputSemantics) == "" ||
		strings.TrimSpace(contract.FailureSemantics) == "" ||
		strings.TrimSpace(contract.ConformanceProfile) == "" {
		return fmt.Errorf("builder contract semantics and conformance profile are required")
	}
	return nil
}

func validateProvider(provider BuilderProvider) error {
	if provider.ProviderID == "" {
		return fmt.Errorf("builder provider id is required")
	}
	if _, err := core.ParseCanonicalID(string(provider.ProviderID)); err != nil {
		return fmt.Errorf("builder provider id: %w", err)
	}
	if !core.ValidNumericVersion(provider.ProviderVersion) {
		return fmt.Errorf("builder provider version %q must be major.minor.patch", provider.ProviderVersion)
	}
	if strings.TrimSpace(provider.ImplementationID) == "" {
		return fmt.Errorf("builder provider implementation id is required")
	}
	if _, err := canonicalDigestChecked(provider.ArtifactDigest); err != nil {
		return err
	}
	if strings.TrimSpace(provider.Scope) == "" ||
		strings.TrimSpace(provider.ContractID) == "" ||
		strings.TrimSpace(provider.ContractVersion) == "" ||
		strings.TrimSpace(provider.ContractFingerprint) == "" ||
		strings.TrimSpace(provider.ConformanceProfile) == "" {
		return fmt.Errorf("builder provider contract identity is incomplete")
	}
	if len(provider.RecipeKinds) == 0 || len(provider.Targets) == 0 || len(provider.Profiles) == 0 {
		return fmt.Errorf("builder provider supported recipe, target, and profile lists are required")
	}
	if hasBlankOrDuplicate(provider.RecipeKinds) || hasBlankOrDuplicate(provider.Targets) || hasBlankOrDuplicate(provider.Profiles) {
		return fmt.Errorf("builder provider support lists contain blank or duplicate values")
	}
	if err := validateFingerprint("provider contract fingerprint", provider.ContractFingerprint); err != nil {
		return err
	}
	return nil
}

func validateEvidence(evidence BuilderConformanceEvidence) error {
	if evidence.ProviderID == "" || !core.ValidNumericVersion(evidence.ProviderVersion) ||
		strings.TrimSpace(evidence.ImplementationID) == "" ||
		strings.TrimSpace(evidence.ConformanceProfile) == "" ||
		strings.TrimSpace(evidence.RecipeKind) == "" ||
		strings.TrimSpace(evidence.Target) == "" ||
		strings.TrimSpace(evidence.Profile) == "" {
		return fmt.Errorf("invalid builder Conformance Evidence identity")
	}
	if _, err := core.ParseCanonicalID(string(evidence.ProviderID)); err != nil {
		return fmt.Errorf("Evidence provider id: %w", err)
	}
	if _, err := canonicalDigestChecked(evidence.ArtifactDigest); err != nil {
		return err
	}
	if err := validateFingerprint("Evidence contract fingerprint", evidence.ContractFingerprint); err != nil {
		return err
	}
	if err := validateFingerprint("Evidence fingerprint", evidence.EvidenceFingerprint); err != nil {
		return err
	}
	expected, err := conformanceEvidenceFingerprint(evidence)
	if err != nil {
		return err
	}
	if canonicalFingerprint(expected) != canonicalFingerprint(evidence.EvidenceFingerprint) {
		return fmt.Errorf("Evidence fingerprint does not match Conformance Evidence claims")
	}
	return nil
}

func conformanceEvidenceFingerprint(evidence BuilderConformanceEvidence) (string, error) {
	if evidence.ProviderID == "" || !core.ValidNumericVersion(evidence.ProviderVersion) ||
		strings.TrimSpace(evidence.ImplementationID) == "" ||
		strings.TrimSpace(evidence.ConformanceProfile) == "" ||
		strings.TrimSpace(evidence.RecipeKind) == "" ||
		strings.TrimSpace(evidence.Target) == "" ||
		strings.TrimSpace(evidence.Profile) == "" {
		return "", fmt.Errorf("invalid builder Conformance Evidence identity")
	}
	if _, err := core.ParseCanonicalID(string(evidence.ProviderID)); err != nil {
		return "", fmt.Errorf("Evidence provider id: %w", err)
	}
	digest, err := canonicalDigestChecked(evidence.ArtifactDigest)
	if err != nil {
		return "", err
	}
	if err := validateFingerprint("Evidence contract fingerprint", evidence.ContractFingerprint); err != nil {
		return "", err
	}
	value := evidence
	value.ArtifactDigest = digest
	value.ContractFingerprint = canonicalFingerprint(value.ContractFingerprint)
	value.EvidenceFingerprint = ""
	return fingerprint(value)
}

func builderContractFingerprint(contract BuilderContract) (string, error) {
	if err := validateContract(contract); err != nil {
		return "", err
	}
	return fingerprint(normalizeContract(contract))
}

func builderIntentFingerprint(intent BuilderIntent) (string, error) {
	if err := validateIntent(intent); err != nil {
		return "", err
	}
	return fingerprint(intent)
}

func resolutionFingerprint(resolution BuilderResolution) (string, error) {
	type providerIdentity struct {
		ProviderID          core.CanonicalID `json:"providerId"`
		ProviderVersion     string           `json:"providerVersion"`
		ImplementationID    string           `json:"implementationId"`
		ArtifactDigest      string           `json:"artifactDigest"`
		Scope               string           `json:"scope"`
		ContractID          string           `json:"contractId"`
		ContractVersion     string           `json:"contractVersion"`
		ContractFingerprint string           `json:"contractFingerprint"`
		ConformanceProfile  string           `json:"conformanceProfile"`
	}
	provider := normalizeProvider(resolution.Provider)
	return fingerprint(struct {
		IntentFingerprint   string                 `json:"intentFingerprint"`
		ContractFingerprint string                 `json:"contractFingerprint"`
		Provider            providerIdentity       `json:"provider"`
		EvidenceFingerprint string                 `json:"evidenceFingerprint"`
		Builder             plugin.BuilderIdentity `json:"builder"`
	}{
		IntentFingerprint:   canonicalFingerprint(resolution.IntentFingerprint),
		ContractFingerprint: canonicalFingerprint(resolution.ContractFingerprint),
		Provider: providerIdentity{
			ProviderID:          provider.ProviderID,
			ProviderVersion:     provider.ProviderVersion,
			ImplementationID:    provider.ImplementationID,
			ArtifactDigest:      provider.ArtifactDigest,
			Scope:               provider.Scope,
			ContractID:          provider.ContractID,
			ContractVersion:     provider.ContractVersion,
			ContractFingerprint: provider.ContractFingerprint,
			ConformanceProfile:  provider.ConformanceProfile,
		},
		EvidenceFingerprint: canonicalFingerprint(resolution.EvidenceFingerprint),
		Builder:             canonicalBuilderIdentity(resolution.Builder),
	})
}

func normalizeContract(contract BuilderContract) BuilderContract {
	return contract
}

func normalizeProvider(provider BuilderProvider) BuilderProvider {
	out := provider
	out.ArtifactDigest = canonicalDigest(provider.ArtifactDigest)
	out.ContractFingerprint = canonicalFingerprint(provider.ContractFingerprint)
	out.RecipeKinds = sortedCopy(provider.RecipeKinds)
	out.Targets = sortedCopy(provider.Targets)
	out.Profiles = sortedCopy(provider.Profiles)
	return out
}

func canonicalBuilderIdentity(builder plugin.BuilderIdentity) plugin.BuilderIdentity {
	out := builder
	out.ArtifactDigest = canonicalDigest(builder.ArtifactDigest)
	return out
}

func fingerprint(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func canonicalDigest(value string) string {
	normalized, err := canonicalDigestChecked(value)
	if err != nil {
		return value
	}
	return normalized
}

func canonicalDigestChecked(value string) (string, error) {
	if !strings.HasPrefix(value, "sha256:") {
		return "", fmt.Errorf("digest must use sha256:<64-hex>")
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if err := validateFingerprint("digest", hexPart); err != nil {
		return "", err
	}
	return "sha256:" + strings.ToLower(hexPart), nil
}

func digestEqual(left, right string) bool {
	a, errA := canonicalDigestChecked(left)
	b, errB := canonicalDigestChecked(right)
	return errA == nil && errB == nil && a == b
}

func canonicalFingerprint(value string) string {
	return strings.ToLower(value)
}

func validateFingerprint(label, value string) error {
	if len(value) != 64 {
		return fmt.Errorf("%s must be 64 hex characters", label)
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'f') ||
			(r >= 'A' && r <= 'F')) {
			return fmt.Errorf("%s must be 64 hex characters", label)
		}
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasBlankOrDuplicate(values []string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
		if _, duplicate := seen[value]; duplicate {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func referenceFixture() (BuilderContract, []BuilderProvider, []BuilderConformanceEvidence, BuilderWiring) {
	contract := BuilderContract{
		ID:                 "tei.reference.builder.copy",
		Version:            "1.0.0",
		InputSemantics:     "explicit-source-bytes",
		OutputSemantics:    "deterministic-copy-artifact",
		FailureSemantics:   "fail-closed",
		ConformanceProfile: "tei.reference.builder-copy-conformance/v1",
	}
	contractFingerprint, _ := builderContractFingerprint(contract)
	providers := []BuilderProvider{
		{
			ProviderID:          core.MustCanonicalID("tei.builder.copy_a"),
			ProviderVersion:     "1.0.0",
			ImplementationID:    "copy-a-native",
			ArtifactDigest:      "sha256:" + strings.Repeat("a", 64),
			Scope:               scopeBuildTool,
			ContractID:          contract.ID,
			ContractVersion:     contract.Version,
			ContractFingerprint: contractFingerprint,
			ConformanceProfile:  contract.ConformanceProfile,
			RecipeKinds:         []string{"deterministic-copy"},
			Targets:             []string{"host/reference"},
			Profiles:            []string{"reference"},
		},
		{
			ProviderID:          core.MustCanonicalID("tei.builder.copy_b"),
			ProviderVersion:     "1.0.0",
			ImplementationID:    "copy-b-portable",
			ArtifactDigest:      "sha256:" + strings.Repeat("b", 64),
			Scope:               scopeBuildTool,
			ContractID:          contract.ID,
			ContractVersion:     contract.Version,
			ContractFingerprint: contractFingerprint,
			ConformanceProfile:  contract.ConformanceProfile,
			RecipeKinds:         []string{"deterministic-copy"},
			Targets:             []string{"host/reference"},
			Profiles:            []string{"reference"},
		},
	}
	evidence := []BuilderConformanceEvidence{
		{
			ProviderID: providers[0].ProviderID, ProviderVersion: providers[0].ProviderVersion,
			ImplementationID: providers[0].ImplementationID, ArtifactDigest: providers[0].ArtifactDigest,
			ContractFingerprint: contractFingerprint, ConformanceProfile: contract.ConformanceProfile,
			RecipeKind: "deterministic-copy", Target: "host/reference", Profile: "reference",
		},
		{
			ProviderID: providers[1].ProviderID, ProviderVersion: providers[1].ProviderVersion,
			ImplementationID: providers[1].ImplementationID, ArtifactDigest: providers[1].ArtifactDigest,
			ContractFingerprint: contractFingerprint, ConformanceProfile: contract.ConformanceProfile,
			RecipeKind: "deterministic-copy", Target: "host/reference", Profile: "reference",
		},
	}
	for i := range evidence {
		fingerprint, err := conformanceEvidenceFingerprint(evidence[i])
		if err != nil {
			panic(err)
		}
		evidence[i].EvidenceFingerprint = fingerprint
	}
	wiring := BuilderWiring{
		ContractID:       contract.ID,
		ProviderID:       providers[0].ProviderID,
		ProviderVersion:  providers[0].ProviderVersion,
		ImplementationID: providers[0].ImplementationID,
		ArtifactDigest:   providers[0].ArtifactDigest,
	}
	return contract, providers, evidence, wiring
}
