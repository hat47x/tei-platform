package transform_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

const bootstrapSeedDigest = "sha256:9999999999999999999999999999999999999999999999999999999999999999"

func referenceTransformSource(t *testing.T, name string) (transform.Source, []byte) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "reference", "plugin-self-build", "builder-sources", name))
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return source, raw
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprintText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func builderFromSource(source transform.Source, digest string) plugin.BuilderIdentity {
	return plugin.BuilderIdentity{
		ID:              source.Identity.ID,
		Version:         source.Identity.Version,
		ContractVersion: source.Identity.ContractVersion,
		ArtifactDigest:  digest,
	}
}

func TestTransformSourceRealizerReferenceMatchesProgrammaticSource(t *testing.T) {
	fromFile, _ := referenceTransformSource(t, "transform-source-realizer-v1.transform.json")
	fileFP, err := transform.SourceFingerprint(fromFile)
	if err != nil {
		t.Fatal(err)
	}
	constructedFP, err := transform.SourceFingerprint(transform.TransformSourceRealizerV1())
	if err != nil {
		t.Fatal(err)
	}
	if fileFP != constructedFP {
		t.Fatalf("Reference meta-builder Source diverged: %s != %s", fileFP, constructedFP)
	}
}

func TestTransformSourceRealizerConvergesStage1AndStage2(t *testing.T) {
	metaSource, rawSource := referenceTransformSource(t, "transform-source-realizer-v1.transform.json")

	stage1Bytes, stage1Digest, err := transform.MaterializeSource(metaSource)
	if err != nil {
		t.Fatal(err)
	}
	stage1Source, err := transform.ParseSourceJSON(stage1Bytes)
	if err != nil {
		t.Fatal(err)
	}
	stage1Value, err := transform.SourceValue(stage1Source)
	if err != nil {
		t.Fatal(err)
	}
	stage2Result, err := transform.Evaluate(stage1Source, map[string]transform.Value{
		"transformSource": stage1Value,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stage2Result.Output.Kind != transform.ValueBytes {
		t.Fatalf("stage2 output kind: %s", stage2Result.Output.Kind)
	}
	stage2Bytes := stage2Result.Output.Bytes
	stage2Digest := digestBytes(stage2Bytes)
	if !bytes.Equal(stage1Bytes, stage2Bytes) || stage1Digest != stage2Digest {
		t.Fatalf("meta-builder did not converge: stage1=%s stage2=%s", stage1Digest, stage2Digest)
	}

	sourceFP, err := transform.SourceFingerprint(metaSource)
	if err != nil {
		t.Fatal(err)
	}
	sourceClosureDigest := digestBytes(rawSource)
	seed := plugin.BuilderIdentity{
		ID:              core.MustCanonicalID("tei.builder.transform_seed"),
		Version:         "reference-go-seed",
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  bootstrapSeedDigest,
	}
	stage1Output := builderFromSource(metaSource, stage1Digest)
	stage2Output := builderFromSource(metaSource, stage2Digest)
	evidence := plugin.BootstrapConvergenceEvidence{
		FormatVersion: plugin.BootstrapConvergenceFormatVersion,
		Stage0Seed: plugin.BootstrapSeedEvidence{
			Builder:       seed,
			TrustBoundary: "external-reference-runtime-engine-and-go-seed",
		},
		Stage1: plugin.BootstrapStageEvidence{
			Stage:                 plugin.BootstrapStage1,
			Builder:               seed,
			SourceClosureDigest:   sourceClosureDigest,
			BuildInputFingerprint: sourceFP,
			BuildPlanFingerprint:  fingerprintText("transform-source-realizer/stage1/" + sourceFP),
			Output:                stage1Output,
		},
		Stage2: plugin.BootstrapStageEvidence{
			Stage:                 plugin.BootstrapStage2,
			Builder:               stage1Output,
			SourceClosureDigest:   sourceClosureDigest,
			BuildInputFingerprint: sourceFP,
			BuildPlanFingerprint:  fingerprintText("transform-source-realizer/stage2/" + sourceFP),
			Output:                stage2Output,
		},
	}
	convergence, err := plugin.EvaluateBootstrapConvergence(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !convergence.Converged {
		t.Fatalf("expected stage1/stage2 convergence: %#v", convergence)
	}
	if evidence.Stage0Seed.Builder.ArtifactDigest == stage1Output.ArtifactDigest {
		t.Fatal("stage0 seed was incorrectly collapsed into self-built output")
	}
}

func TestConvergedMetaBuilderRealizesPluginPackageBuilderSource(t *testing.T) {
	metaSource, _ := referenceTransformSource(t, "transform-source-realizer-v1.transform.json")
	packageSource, _ := referenceTransformSource(t, "plugin-package-v1.transform.json")

	packageValue, err := transform.SourceValue(packageSource)
	if err != nil {
		t.Fatal(err)
	}
	got, err := transform.Evaluate(metaSource, map[string]transform.Value{
		"transformSource": packageValue,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Output.Kind != transform.ValueBytes {
		t.Fatalf("package builder realization kind: %s", got.Output.Kind)
	}
	want, wantDigest, err := transform.MaterializeSource(packageSource)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Output.Bytes, want) || digestBytes(got.Output.Bytes) != wantDigest {
		t.Fatal("meta-builder did not reproduce canonical package builder realization")
	}
}

func TestTransformSourceRealizerDetectsSourceDrift(t *testing.T) {
	metaSource, rawSource := referenceTransformSource(t, "transform-source-realizer-v1.transform.json")
	stage1Bytes, stage1Digest, err := transform.MaterializeSource(metaSource)
	if err != nil {
		t.Fatal(err)
	}

	drifted := metaSource
	drifted.Limits.MaxSteps++
	driftedValue, err := transform.SourceValue(drifted)
	if err != nil {
		t.Fatal(err)
	}
	stage2, err := transform.Evaluate(metaSource, map[string]transform.Value{
		"transformSource": driftedValue,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(stage1Bytes, stage2.Output.Bytes) || stage1Digest == digestBytes(stage2.Output.Bytes) {
		t.Fatal("source drift did not change builder realization")
	}

	sourceFP, _ := transform.SourceFingerprint(metaSource)
	driftedRaw, _, err := transform.MaterializeSource(drifted)
	if err != nil {
		t.Fatal(err)
	}
	stage1Output := builderFromSource(metaSource, stage1Digest)
	seed := plugin.BuilderIdentity{
		ID:              core.MustCanonicalID("tei.builder.transform_seed"),
		Version:         "reference-go-seed",
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  bootstrapSeedDigest,
	}
	evidence := plugin.BootstrapConvergenceEvidence{
		FormatVersion: plugin.BootstrapConvergenceFormatVersion,
		Stage0Seed: plugin.BootstrapSeedEvidence{Builder: seed, TrustBoundary: "external-reference-seed"},
		Stage1: plugin.BootstrapStageEvidence{
			Stage: plugin.BootstrapStage1, Builder: seed,
			SourceClosureDigest: digestBytes(rawSource), BuildInputFingerprint: sourceFP,
			BuildPlanFingerprint: fingerprintText("stage1"), Output: stage1Output,
		},
		Stage2: plugin.BootstrapStageEvidence{
			Stage: plugin.BootstrapStage2, Builder: stage1Output,
			SourceClosureDigest: digestBytes(driftedRaw), BuildInputFingerprint: sourceFP,
			BuildPlanFingerprint: fingerprintText("stage2"),
			Output: builderFromSource(metaSource, digestBytes(stage2.Output.Bytes)),
		},
	}
	if _, err := plugin.EvaluateBootstrapConvergence(evidence); err == nil {
		t.Fatal("expected source-closure drift to reject bootstrap comparison")
	}
}

func TestSourceValueRejectsInvalidTransformSource(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	source.Profile = "unsupported"
	if _, err := transform.SourceValue(source); err == nil {
		t.Fatal("expected invalid transform Source to fail")
	}
}
