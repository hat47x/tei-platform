package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/plugin"
	"tei/internal/syntax"
)

const ReceiptFormatVersion = 1

// SourceResolver は論理Source参照からexactなBuildSourceを導出する入口である。
// 言語固有のSource closure解決（#1096）はここへ接続する。
type SourceResolver interface {
	Resolve(IntentSource) (plugin.BuildSource, error)
	ReadFile(ref string) ([]byte, error)
}

// BuilderRequest はBuilder Contract解決（#1101）への入力である。
type BuilderRequest struct {
	Contract        string
	CompatibleMajor int
	Provider        *ProviderWiring
	Pin             *ArtifactPin
	RecipeKind      string
	Target          string
	Profile         string
}

// BuilderResolution は解決されたexact Builderと、その観測Evidenceの識別である。
type BuilderResolution struct {
	Identity       plugin.BuilderIdentity `json:"identity"`
	Implementation string                 `json:"implementation"`
	Pinned         bool                   `json:"pinned"`
}

type BuilderResolver interface {
	Resolve(BuilderRequest) (BuilderResolution, error)
	CatalogFingerprint() (string, error)
}

// DependencyResolver はBuild graph（#1103）のproducer outputを解決する入口である。
type DependencyResolver interface {
	Resolve(DependencyRequirement) (plugin.BuildDependency, error)
	CatalogFingerprint() (string, error)
}

// TargetCatalog はTarget / Profileの存在を確認する。
type TargetCatalog interface {
	Check(target, profile string) error
	CatalogFingerprint() (string, error)
}

// ResolutionContext はResolverが観測する外部状態への入口をまとめる。
type ResolutionContext struct {
	Sources      SourceResolver
	Builders     BuilderResolver
	Dependencies DependencyResolver
	Targets      TargetCatalog
}

// ResolutionContextFingerprints はContextの各部分とResolver版のfingerprintである。
// 現在時刻、読込順、Host pathは含めない。
type ResolutionContextFingerprints struct {
	ResolverVersion      string `json:"resolverVersion"`
	SourceClosure        string `json:"sourceClosure"`
	BuilderCatalog       string `json:"builderCatalog"`
	DependencyCatalog    string `json:"dependencyCatalog"`
	TargetProfileCatalog string `json:"targetProfileCatalog"`
}

// ResolvedBuildReceipt はIntent、観測Context、導出したexact BuildPlanを束ねる。
// これは再生成可能なderived cacheであり、Canonical Sourceではない。
type ResolvedBuildReceipt struct {
	FormatVersion                   int                           `json:"formatVersion"`
	Authority                       string                        `json:"authority"`
	IntentFingerprint               string                        `json:"intentFingerprint"`
	ResolutionContext               ResolutionContextFingerprints `json:"resolutionContext"`
	ResolutionContextFingerprint    string                        `json:"resolutionContextFingerprint"`
	PlanFingerprint                 string                        `json:"planFingerprint"`
	SourceClosureFingerprint        string                        `json:"sourceClosureFingerprint"`
	BuilderResolution               BuilderResolution             `json:"builderResolution"`
	BuilderResolutionFingerprint    string                        `json:"builderResolutionFingerprint"`
	DependencyResolutionFingerprint string                        `json:"dependencyResolutionFingerprint"`
	SecretRequirementNames          []string                      `json:"secretRequirementNames,omitempty"`
	Plan                            plugin.BuildPlan              `json:"plan"`
}

// AuthorityDerivedCache は、Receiptが編集してよい正本ではないことを示す。
const AuthorityDerivedCache = "derived-cache"

// Resolve はBuildIntentとResolutionContextからexact BuildPlanを決定的に導出する。
// Provider選択が曖昧な場合、読込順では選ばず停止する。
func Resolve(intent BuildIntent, ctx ResolutionContext) (ResolvedBuildReceipt, error) {
	if err := intent.Validate(); err != nil {
		return ResolvedBuildReceipt{}, err
	}
	if ctx.Sources == nil || ctx.Builders == nil || ctx.Dependencies == nil || ctx.Targets == nil {
		return ResolvedBuildReceipt{}, fmt.Errorf("resolution context is incomplete")
	}
	intentFP, err := IntentFingerprint(intent)
	if err != nil {
		return ResolvedBuildReceipt{}, err
	}
	n := NormalizeIntent(intent)

	sources := make([]plugin.BuildSource, 0, len(n.Sources))
	var definitionRef string
	for _, s := range n.Sources {
		resolved, err := ctx.Sources.Resolve(s)
		if err != nil {
			return ResolvedBuildReceipt{}, fmt.Errorf("resolve source %q: %w", s.ID, err)
		}
		sources = append(sources, resolved)
		if s.ID == n.Subject.DefinitionSource {
			definitionRef = s.Ref
		}
	}
	// Source closureの観測値は、Source自体のidentity（Kind/ID/Locator/Digest）である。
	sourceFP, err := fingerprintJSON(plugin.NormalizeBuildPlan(plugin.BuildPlan{Sources: sources}).Sources)
	if err != nil {
		return ResolvedBuildReceipt{}, err
	}

	definitionRaw, err := ctx.Sources.ReadFile(definitionRef)
	if err != nil {
		return ResolvedBuildReceipt{}, fmt.Errorf("read plugin definition: %w", err)
	}
	definition, err := syntax.NewParser(string(definitionRaw)).ParsePlugin()
	if err != nil {
		return ResolvedBuildReceipt{}, fmt.Errorf("parse plugin definition: %w", err)
	}
	if definition.ID != n.Subject.ID || definition.Version != n.Subject.Version {
		return ResolvedBuildReceipt{}, fmt.Errorf(
			"plugin definition %s@%s does not match build intent subject %s@%s",
			definition.ID, definition.Version, n.Subject.ID, n.Subject.Version,
		)
	}

	if err := ctx.Targets.Check(n.Target, n.Profile); err != nil {
		return ResolvedBuildReceipt{}, err
	}
	builder, err := ctx.Builders.Resolve(BuilderRequest{
		Contract:        n.Builder.Contract,
		CompatibleMajor: n.Builder.CompatibleMajor,
		Provider:        n.Builder.Provider,
		Pin:             n.Builder.Pin,
		RecipeKind:      n.Recipe.Kind,
		Target:          n.Target,
		Profile:         n.Profile,
	})
	if err != nil {
		return ResolvedBuildReceipt{}, fmt.Errorf("resolve builder: %w", err)
	}

	dependencies := make([]plugin.BuildDependency, 0, len(n.Dependencies))
	for _, requirement := range n.Dependencies {
		resolved, err := ctx.Dependencies.Resolve(requirement)
		if err != nil {
			return ResolvedBuildReceipt{}, fmt.Errorf("resolve build dependency %s/%s: %w", requirement.Kind, requirement.ID, err)
		}
		dependencies = append(dependencies, resolved)
	}

	plan := plugin.NormalizeBuildPlan(plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: n.Subject.ID, Version: n.Subject.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources:                     sources,
		Builder:                     builder.Identity,
		Recipe:                      n.Recipe,
		OutputKind:                  n.OutputKind,
		Dependencies:                dependencies,
		Target:                      n.Target,
		Profile:                     n.Profile,
		Parameters:                  n.Parameters,
	})
	planFP, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return ResolvedBuildReceipt{}, fmt.Errorf("derived build plan is invalid: %w", err)
	}

	contextFPs := ResolutionContextFingerprints{ResolverVersion: ResolverVersion, SourceClosure: sourceFP}
	if contextFPs.BuilderCatalog, err = ctx.Builders.CatalogFingerprint(); err != nil {
		return ResolvedBuildReceipt{}, err
	}
	if contextFPs.DependencyCatalog, err = ctx.Dependencies.CatalogFingerprint(); err != nil {
		return ResolvedBuildReceipt{}, err
	}
	if contextFPs.TargetProfileCatalog, err = ctx.Targets.CatalogFingerprint(); err != nil {
		return ResolvedBuildReceipt{}, err
	}
	contextFP, err := fingerprintJSON(contextFPs)
	if err != nil {
		return ResolvedBuildReceipt{}, err
	}
	builderFP, err := fingerprintJSON(builder)
	if err != nil {
		return ResolvedBuildReceipt{}, err
	}
	dependencyFP, err := fingerprintJSON(plan.Dependencies)
	if err != nil {
		return ResolvedBuildReceipt{}, err
	}

	secretNames := make([]string, 0, len(n.SecretRequirements))
	for _, s := range n.SecretRequirements {
		secretNames = append(secretNames, s.Name)
	}
	sort.Strings(secretNames)

	return ResolvedBuildReceipt{
		FormatVersion:                   ReceiptFormatVersion,
		Authority:                       AuthorityDerivedCache,
		IntentFingerprint:               intentFP,
		ResolutionContext:               contextFPs,
		ResolutionContextFingerprint:    contextFP,
		PlanFingerprint:                 planFP,
		SourceClosureFingerprint:        sourceFP,
		BuilderResolution:               builder,
		BuilderResolutionFingerprint:    builderFP,
		DependencyResolutionFingerprint: dependencyFP,
		SecretRequirementNames:          secretNames,
		Plan:                            plan,
	}, nil
}

// MarshalReceipt はReceiptを決定的なJSONへ直列化する。
func MarshalReceipt(r ResolvedBuildReceipt) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// VerifyDerivedReceipt は保存済みReceiptがIntentとContextからの再導出と
// bytes一致することを確認する。人が編集したReceiptはSourceへ昇格せず、
// 乖離として拒否する。
func VerifyDerivedReceipt(intent BuildIntent, ctx ResolutionContext, stored []byte) error {
	fresh, err := Resolve(intent, ctx)
	if err != nil {
		return err
	}
	want, err := MarshalReceipt(fresh)
	if err != nil {
		return err
	}
	var parsed ResolvedBuildReceipt
	dec := json.NewDecoder(bytes.NewReader(stored))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&parsed); err != nil {
		return fmt.Errorf("stored derived receipt is not a receipt: %w", err)
	}
	got, err := MarshalReceipt(parsed)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("stored derived receipt diverges from the intent resolution; edit the build intent and re-derive, never the derived receipt")
	}
	return nil
}
