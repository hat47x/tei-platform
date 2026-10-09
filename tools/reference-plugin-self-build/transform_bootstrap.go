package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

const (
	transformSeedBuilderPackage = "./tools/reference-transform-seed-builder"
	transformMetaSourcePath     = "reference/plugin-self-build/builder-sources/transform-source-realizer-v1.transform.json"
	transformPackageSourcePath  = "reference/plugin-self-build/builder-sources/plugin-package-v1.transform.json"
)

type transformBootstrapEvidence struct {
	RuntimeEngineSourceDigest      string                            `json:"runtimeEngineSourceDigest"`
	RuntimeEngine                  pruntime.RuntimeEngineIdentity    `json:"runtimeEngine"`
	RuntimeEngineFingerprint       string                            `json:"runtimeEngineFingerprint"`
	SeedBuilderFirstDigest         string                            `json:"seedBuilderFirstDigest"`
	SeedBuilderSecondDigest        string                            `json:"seedBuilderSecondDigest"`
	SeedBuilderReproducible        bool                              `json:"seedBuilderReproducible"`
	MetaSourceFingerprint          string                            `json:"metaSourceFingerprint"`
	Stage1ArtifactDigest           string                            `json:"stage1ArtifactDigest"`
	Stage2ArtifactDigest           string                            `json:"stage2ArtifactDigest"`
	ConvergenceEvidence            plugin.BootstrapConvergenceEvidence `json:"convergenceEvidence"`
	ConvergenceResult              plugin.BootstrapConvergenceResult   `json:"convergenceResult"`
	PackageBuilderSourceFingerprint string                           `json:"packageBuilderSourceFingerprint"`
	PackageBuilderArtifactDigest   string                            `json:"packageBuilderArtifactDigest"`
	PackageBuildPlan               plugin.BuildPlan                  `json:"packageBuildPlan"`
	PackageBuildPlanFingerprint    string                            `json:"packageBuildPlanFingerprint"`
	PackageFirstArtifactDigest     string                            `json:"packageFirstArtifactDigest"`
	PackageSecondArtifactDigest    string                            `json:"packageSecondArtifactDigest"`
	PackageReproducible            bool                              `json:"packageReproducible"`
	PackageOracleDigest            string                            `json:"packageOracleDigest"`
	PackageLockedBuild             plugin.LockedBuild                `json:"packageLockedBuild"`
	PackageLockFingerprint         string                            `json:"packageLockFingerprint"`
}

func runTransformBootstrap(
	root, goPath, goVersion, goToolchainDigest, revision string,
	packageDefinition plugin.Definition,
	basePackagePlan plugin.BuildPlan,
) (transformBootstrapEvidence, error) {
	engineSourceDigest, err := hashSourceFiles(root, []string{
		"internal/plugin/transform/value.go",
		"internal/plugin/transform/model.go",
		"internal/plugin/transform/eval.go",
		"internal/plugin/transform/source.go",
		"internal/plugin/transform/self_host.go",
	})
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("hash deterministic transform Runtime Engine source closure: %w", err)
	}
	engine := pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
		Version:              revision,
		ContractVersion:      "tei.runtime-engine/deterministic-transform-v1",
		ImplementationDigest: sha256Digest([]byte(engineSourceDigest + "\n" + goToolchainDigest + "\n")),
		Target:               "host/" + runtime.GOOS + "/" + runtime.GOARCH,
	}
	engineFingerprint, err := pruntime.RuntimeEngineIdentityFingerprint(engine)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	if err := requireNoExternalModules(root, goPath, transformSeedBuilderPackage); err != nil {
		return transformBootstrapEvidence{}, err
	}

	metaRaw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(transformMetaSourcePath)))
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	metaSource, err := transform.ParseSourceJSON(metaRaw)
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("parse transform meta-builder Source: %w", err)
	}
	metaSourceFingerprint, err := transform.SourceFingerprint(metaSource)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	metaSourceClosureDigest, err := hashSourceFiles(root, []string{transformMetaSourcePath})
	if err != nil {
		return transformBootstrapEvidence{}, err
	}

	packageSourceRaw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(transformPackageSourcePath)))
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	packageSource, err := transform.ParseSourceJSON(packageSourceRaw)
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("parse Plugin package transform Source: %w", err)
	}
	packageSourceFingerprint, err := transform.SourceFingerprint(packageSource)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}

	tempDir, err := os.MkdirTemp("", "tei-transform-bootstrap-*")
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	defer os.RemoveAll(tempDir)

	seedFirstPath := filepath.Join(tempDir, "seed-first")
	seedSecondPath := filepath.Join(tempDir, "seed-second")
	if runtime.GOOS == "windows" {
		seedFirstPath += ".exe"
		seedSecondPath += ".exe"
	}
	if err := buildGoPackage(
		root,
		goPath,
		transformSeedBuilderPackage,
		seedFirstPath,
		filepath.Join(tempDir, "seed-state-first"),
	); err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("build first transform seed builder: %w", err)
	}
	if err := buildGoPackage(
		root,
		goPath,
		transformSeedBuilderPackage,
		seedSecondPath,
		filepath.Join(tempDir, "seed-state-second"),
	); err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("build second transform seed builder: %w", err)
	}
	seedFirstDigest, err := hashFile(seedFirstPath)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	seedSecondDigest, err := hashFile(seedSecondPath)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	if seedFirstDigest != seedSecondDigest {
		return transformBootstrapEvidence{}, fmt.Errorf(
			"transform stage0 seed builder is not reproducible: %s != %s",
			seedFirstDigest,
			seedSecondDigest,
		)
	}

	stage1FirstPath := filepath.Join(tempDir, "stage1-first.transform.json")
	stage1SecondPath := filepath.Join(tempDir, "stage1-second.transform.json")
	if err := runTransformSeedBuilder(root, seedFirstPath, transformMetaSourcePath, stage1FirstPath); err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("materialize first stage1 builder: %w", err)
	}
	if err := runTransformSeedBuilder(root, seedSecondPath, transformMetaSourcePath, stage1SecondPath); err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("materialize second stage1 builder: %w", err)
	}
	stage1FirstBytes, err := os.ReadFile(stage1FirstPath)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	stage1SecondBytes, err := os.ReadFile(stage1SecondPath)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	stage1FirstDigest, err := hashFile(stage1FirstPath)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	stage1SecondDigest, err := hashFile(stage1SecondPath)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	if !bytes.Equal(stage1FirstBytes, stage1SecondBytes) || stage1FirstDigest != stage1SecondDigest {
		return transformBootstrapEvidence{}, fmt.Errorf(
			"stage0 seed produced non-reproducible stage1 builder: %s != %s",
			stage1FirstDigest,
			stage1SecondDigest,
		)
	}
	stage1Source, err := transform.ParseSourceJSON(stage1FirstBytes)
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("parse stage1 builder artifact: %w", err)
	}
	stage1SourceFingerprint, err := transform.SourceFingerprint(stage1Source)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	if stage1SourceFingerprint != metaSourceFingerprint {
		return transformBootstrapEvidence{}, fmt.Errorf("stage1 builder Source fingerprint diverged from authoritative meta-builder Source")
	}

	stage1Value, err := transform.SourceValue(stage1Source)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	stage2Result, err := transform.Evaluate(stage1Source, map[string]transform.Value{
		"transformSource": stage1Value,
	})
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("execute stage1 meta-builder: %w", err)
	}
	if stage2Result.Output.Kind != transform.ValueBytes {
		return transformBootstrapEvidence{}, fmt.Errorf("stage2 builder output kind is %s, expected bytes", stage2Result.Output.Kind)
	}
	stage2Bytes := stage2Result.Output.Bytes
	stage2Digest := plugin.BuildOutputDigest(stage2Bytes)
	if !bytes.Equal(stage1FirstBytes, stage2Bytes) || stage1FirstDigest != stage2Digest {
		return transformBootstrapEvidence{}, fmt.Errorf(
			"transform meta-builder did not converge: stage1=%s stage2=%s",
			stage1FirstDigest,
			stage2Digest,
		)
	}

	seedIdentity := plugin.BuilderIdentity{
		ID:              core.MustCanonicalID("tei.builder.transform_seed.reference"),
		Version:         revision,
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  seedFirstDigest,
	}
	stage1Output := builderIdentityFromTransformSource(metaSource, stage1FirstDigest)
	buildInputFingerprint := transformBootstrapInputFingerprint(
		metaSourceFingerprint,
		engineFingerprint,
		engine.Target,
	)
	convergenceEvidence := plugin.BootstrapConvergenceEvidence{
		FormatVersion: plugin.BootstrapConvergenceFormatVersion,
		Stage0Seed: plugin.BootstrapSeedEvidence{
			Builder:       seedIdentity,
			TrustBoundary: "external-go-seed-and-deterministic-transform-runtime-engine",
		},
		Stage1: plugin.BootstrapStageEvidence{
			Stage:                 plugin.BootstrapStage1,
			Builder:               seedIdentity,
			SourceClosureDigest:   metaSourceClosureDigest,
			BuildInputFingerprint: buildInputFingerprint,
			BuildPlanFingerprint: transformBootstrapStageFingerprint(
				"stage1",
				seedIdentity,
				metaSourceFingerprint,
				engineFingerprint,
			),
			Output: stage1Output,
		},
		Stage2: plugin.BootstrapStageEvidence{
			Stage:                 plugin.BootstrapStage2,
			Builder:               stage1Output,
			SourceClosureDigest:   metaSourceClosureDigest,
			BuildInputFingerprint: buildInputFingerprint,
			BuildPlanFingerprint: transformBootstrapStageFingerprint(
				"stage2",
				stage1Output,
				metaSourceFingerprint,
				engineFingerprint,
			),
			Output: builderIdentityFromTransformSource(metaSource, stage2Digest),
		},
	}
	convergenceResult, err := plugin.EvaluateBootstrapConvergence(convergenceEvidence)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	if !convergenceResult.Converged {
		return transformBootstrapEvidence{}, fmt.Errorf("transform bootstrap did not converge")
	}

	packageSourceValue, err := transform.SourceValue(packageSource)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	packageBuilderResult, err := transform.Evaluate(stage1Source, map[string]transform.Value{
		"transformSource": packageSourceValue,
	})
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("realize Plugin package builder with converged meta-builder: %w", err)
	}
	if packageBuilderResult.Output.Kind != transform.ValueBytes {
		return transformBootstrapEvidence{}, fmt.Errorf("Plugin package builder realization output is %s", packageBuilderResult.Output.Kind)
	}
	packageBuilderBytes := packageBuilderResult.Output.Bytes
	packageBuilderDigest := plugin.BuildOutputDigest(packageBuilderBytes)
	parsedPackageBuilder, err := transform.ParseSourceJSON(packageBuilderBytes)
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("parse realized Plugin package builder: %w", err)
	}
	parsedPackageFP, err := transform.SourceFingerprint(parsedPackageBuilder)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	if parsedPackageFP != packageSourceFingerprint {
		return transformBootstrapEvidence{}, fmt.Errorf("realized Plugin package builder Source fingerprint diverged")
	}

	packageBuilderIdentity := builderIdentityFromTransformSource(packageSource, packageBuilderDigest)
	teiPackagePlan := basePackagePlan
	teiPackagePlan.Builder = packageBuilderIdentity
	teiPackagePlan.Dependencies = append(
		append([]plugin.BuildDependency(nil), basePackagePlan.Dependencies...),
		plugin.BuildDependency{
			Kind:           "runtime-engine",
			ID:             string(engine.ID),
			Version:        engine.Version,
			ArtifactDigest: engine.ImplementationDigest,
		},
	)
	teiPackagePlan.Parameters = append(
		append([]plugin.BuildParameter(nil), basePackagePlan.Parameters...),
		plugin.BuildParameter{Name: "runtime-engine-contract", Value: engine.ContractVersion},
		plugin.BuildParameter{Name: "runtime-engine-fingerprint", Value: engineFingerprint},
		plugin.BuildParameter{Name: "runtime-engine-target", Value: engine.Target},
	)
	teiPackagePlanFingerprint, err := plugin.BuildPlanFingerprint(teiPackagePlan)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}

	planValue, err := transformValueFromJSON(teiPackagePlan)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	definitionValue, err := transformValueFromJSON(packageDefinition)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	firstPackageResult, err := transform.Evaluate(parsedPackageBuilder, map[string]transform.Value{
		"buildPlan":        planValue,
		"pluginDefinition": definitionValue,
	})
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("execute TEI-native Plugin package builder: %w", err)
	}
	secondPackageResult, err := transform.Evaluate(parsedPackageBuilder, map[string]transform.Value{
		"buildPlan":        planValue,
		"pluginDefinition": definitionValue,
	})
	if err != nil {
		return transformBootstrapEvidence{}, fmt.Errorf("repeat TEI-native Plugin package builder: %w", err)
	}
	if firstPackageResult.Output.Kind != transform.ValueBytes || secondPackageResult.Output.Kind != transform.ValueBytes {
		return transformBootstrapEvidence{}, fmt.Errorf("TEI-native Plugin package builder did not return bytes")
	}
	firstPackageBytes := firstPackageResult.Output.Bytes
	secondPackageBytes := secondPackageResult.Output.Bytes
	firstPackageDigest := plugin.BuildOutputDigest(firstPackageBytes)
	secondPackageDigest := plugin.BuildOutputDigest(secondPackageBytes)
	if !bytes.Equal(firstPackageBytes, secondPackageBytes) || firstPackageDigest != secondPackageDigest {
		return transformBootstrapEvidence{}, fmt.Errorf(
			"TEI-native Plugin package builder is not reproducible: %s != %s",
			firstPackageDigest,
			secondPackageDigest,
		)
	}
	oracleBytes, err := plugin.MaterializePluginPackage(packageDefinition, teiPackagePlan)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	oracleDigest := plugin.BuildOutputDigest(oracleBytes)
	if !bytes.Equal(firstPackageBytes, oracleBytes) || firstPackageDigest != oracleDigest {
		return transformBootstrapEvidence{}, fmt.Errorf(
			"TEI-native Plugin package builder diverged from Go oracle: %s != %s",
			firstPackageDigest,
			oracleDigest,
		)
	}
	lockedBuild, err := plugin.LockBuildOutput(teiPackagePlan, firstPackageDigest)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}
	artifactLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{lockedBuild},
	}
	lockFingerprint, err := plugin.BuildArtifactLockFingerprint(artifactLock)
	if err != nil {
		return transformBootstrapEvidence{}, err
	}

	return transformBootstrapEvidence{
		RuntimeEngineSourceDigest:       engineSourceDigest,
		RuntimeEngine:                   engine,
		RuntimeEngineFingerprint:        engineFingerprint,
		SeedBuilderFirstDigest:          seedFirstDigest,
		SeedBuilderSecondDigest:         seedSecondDigest,
		SeedBuilderReproducible:         true,
		MetaSourceFingerprint:           metaSourceFingerprint,
		Stage1ArtifactDigest:            stage1FirstDigest,
		Stage2ArtifactDigest:            stage2Digest,
		ConvergenceEvidence:             convergenceEvidence,
		ConvergenceResult:               convergenceResult,
		PackageBuilderSourceFingerprint: packageSourceFingerprint,
		PackageBuilderArtifactDigest:    packageBuilderDigest,
		PackageBuildPlan:                teiPackagePlan,
		PackageBuildPlanFingerprint:     teiPackagePlanFingerprint,
		PackageFirstArtifactDigest:      firstPackageDigest,
		PackageSecondArtifactDigest:     secondPackageDigest,
		PackageReproducible:             true,
		PackageOracleDigest:             oracleDigest,
		PackageLockedBuild:              lockedBuild,
		PackageLockFingerprint:          lockFingerprint,
	}, nil
}

func runTransformSeedBuilder(root, builderPath, sourceRelative, output string) error {
	cmd := exec.Command(
		builderPath,
		"-source",
		filepath.Join(root, filepath.FromSlash(sourceRelative)),
		"-out",
		output,
	)
	cmd.Dir = root
	cmd.Env = []string{}
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("transform seed builder: %w: %s", err, bytes.TrimSpace(combined))
	}
	return nil
}

func builderIdentityFromTransformSource(source transform.Source, digest string) plugin.BuilderIdentity {
	return plugin.BuilderIdentity{
		ID:              source.Identity.ID,
		Version:         source.Identity.Version,
		ContractVersion: source.Identity.ContractVersion,
		ArtifactDigest:  digest,
	}
}

func transformValueFromJSON(value any) (transform.Value, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return transform.Value{}, err
	}
	return transform.ParseJSON(raw)
}

func transformBootstrapInputFingerprint(sourceFingerprint, engineFingerprint, target string) string {
	raw := sourceFingerprint + "\n" + engineFingerprint + "\n" + target + "\n"
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func transformBootstrapStageFingerprint(
	stage string,
	builder plugin.BuilderIdentity,
	sourceFingerprint, engineFingerprint string,
) string {
	payload := struct {
		Stage             string                 `json:"stage"`
		Builder           plugin.BuilderIdentity `json:"builder"`
		SourceFingerprint string                 `json:"sourceFingerprint"`
		EngineFingerprint string                 `json:"engineFingerprint"`
	}{
		Stage:             stage,
		Builder:           builder,
		SourceFingerprint: sourceFingerprint,
		EngineFingerprint: engineFingerprint,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
