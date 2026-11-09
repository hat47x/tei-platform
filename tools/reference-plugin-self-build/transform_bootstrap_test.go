package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

func bootstrapTestMetaSource(t *testing.T) transform.Source {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(transformMetaSourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func bootstrapTestEngine() pruntime.RuntimeEngineIdentity {
	return pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.transform_test"),
		Version:              "1.0.0",
		ContractVersion:      "tei.runtime-engine/deterministic-transform-v1",
		ImplementationDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Target:               "host/test/test",
	}
}

func TestTransformBootstrapComparisonFingerprintIgnoresStageExecution(t *testing.T) {
	source := bootstrapTestMetaSource(t)
	base := transformBootstrapComparisonFingerprint(
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		source.Identity,
	)
	if got := transformBootstrapComparisonFingerprint(
		"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		source.Identity,
	); got == base {
		t.Fatal("comparison fingerprint ignored source closure")
	}
	if got := transformBootstrapComparisonFingerprint(
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		source.Identity,
	); got == base {
		t.Fatal("comparison fingerprint ignored Source fingerprint")
	}
	changedOutput := source.Identity
	changedOutput.Version = "1.0.1"
	if got := transformBootstrapComparisonFingerprint(
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		changedOutput,
	); got == base {
		t.Fatal("comparison fingerprint ignored logical output identity")
	}
}

func TestTransformBootstrapStagePlanFingerprintBindsExactExecution(t *testing.T) {
	source := bootstrapTestMetaSource(t)
	seed := plugin.BuilderIdentity{
		ID:              core.MustCanonicalID("tei.builder.transform_seed.test"),
		Version:         "seed-a",
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	stage1 := transformBootstrapStagePlan{
		Stage:               plugin.BootstrapStage1,
		Builder:             seed,
		SourceClosureDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		SourceFingerprint:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Execution: transformBootstrapStageExecution{
			Kind:   transformBootstrapExecutionNativeSeed,
			Target: "host/test/test",
		},
		Output: source.Identity,
	}
	stage1FP, err := transformBootstrapStagePlanFingerprint(stage1)
	if err != nil {
		t.Fatal(err)
	}

	changedBuilder := stage1
	changedBuilder.Builder.ArtifactDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	if got, err := transformBootstrapStagePlanFingerprint(changedBuilder); err != nil {
		t.Fatal(err)
	} else if got == stage1FP {
		t.Fatal("stage plan fingerprint ignored builder identity")
	}

	changedTarget := stage1
	changedTarget.Execution.Target = "host/other/test"
	if got, err := transformBootstrapStagePlanFingerprint(changedTarget); err != nil {
		t.Fatal(err)
	} else if got == stage1FP {
		t.Fatal("stage plan fingerprint ignored native execution target")
	}

	engine := bootstrapTestEngine()
	stage2 := transformBootstrapStagePlan{
		Stage:               plugin.BootstrapStage2,
		Builder:             seed,
		SourceClosureDigest: stage1.SourceClosureDigest,
		SourceFingerprint:   stage1.SourceFingerprint,
		Execution: transformBootstrapStageExecution{
			Kind:          transformBootstrapExecutionInterpreted,
			Target:        engine.Target,
			RuntimeEngine: &engine,
		},
		Output: source.Identity,
	}
	stage2FP, err := transformBootstrapStagePlanFingerprint(stage2)
	if err != nil {
		t.Fatal(err)
	}
	changedEngine := stage2
	changedEngine.Execution.RuntimeEngine = new(pruntime.RuntimeEngineIdentity)
	*changedEngine.Execution.RuntimeEngine = engine
	changedEngine.Execution.RuntimeEngine.ImplementationDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if got, err := transformBootstrapStagePlanFingerprint(changedEngine); err != nil {
		t.Fatal(err)
	} else if got == stage2FP {
		t.Fatal("stage2 plan fingerprint ignored Runtime Engine identity")
	}

	if stage1.Execution.RuntimeEngine != nil {
		t.Fatal("native seed stage must not report an interpreted Runtime Engine")
	}
	if stage2.Execution.RuntimeEngine == nil || *stage2.Execution.RuntimeEngine != engine {
		t.Fatal("interpreted stage lost exact Runtime Engine identity")
	}
}

func TestBuilderIdentityFromTransformSourcePreservesLogicalIdentity(t *testing.T) {
	source := bootstrapTestMetaSource(t)
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	got := builderIdentityFromTransformSource(source, digest)
	if got.ID != source.Identity.ID ||
		got.Version != source.Identity.Version ||
		got.ContractVersion != source.Identity.ContractVersion ||
		got.ArtifactDigest != digest {
		t.Fatalf("builder identity: %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTransformSeedBuilderVersionSeparatesRepositoryRevision(t *testing.T) {
	const artifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	builder := transformSeedBuilderIdentity(artifactDigest)
	if builder.ID != core.MustCanonicalID("tei.builder.transform_seed.reference") ||
		builder.Version != "1.0.0" ||
		builder.ContractVersion != "tei.plugin-build/v1" ||
		builder.ArtifactDigest != artifactDigest {
		t.Fatalf("unexpected logical seed Builder identity: %+v", builder)
	}
	if err := builder.Validate(); err != nil {
		t.Fatal(err)
	}

	source := bootstrapTestMetaSource(t)
	fingerprint := func(identity plugin.BuilderIdentity) string {
		t.Helper()
		plan := transformBootstrapStagePlan{
			Stage:               plugin.BootstrapStage1,
			Builder:             identity,
			SourceClosureDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			SourceFingerprint:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			Execution: transformBootstrapStageExecution{
				Kind:   transformBootstrapExecutionNativeSeed,
				Target: "host/test/test",
			},
			Output: source.Identity,
		}
		got, err := transformBootstrapStagePlanFingerprint(plan)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	baseFingerprint := fingerprint(builder)

	// Keep the repository revision in evidence, but never use it as the
	// identity's Version or as an input to the stage plan fingerprint.
	before := evidence{RepositoryRevision: "revision-before-docs"}
	after := evidence{RepositoryRevision: "revision-after-docs"}
	if before.RepositoryRevision == after.RepositoryRevision {
		t.Fatal("test must use distinct repository revisions")
	}
	sameArtifactBuilder := transformSeedBuilderIdentity(artifactDigest)
	if sameArtifactBuilder != builder || fingerprint(sameArtifactBuilder) != baseFingerprint {
		t.Fatal("repository-only revision unexpectedly altered seed builder identity or stage plan")
	}

	changedArtifact := transformSeedBuilderIdentity("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	changedVersion := builder
	changedVersion.Version = "1.0.1"
	changedContract := builder
	changedContract.ContractVersion = "tei.plugin-build/v2"
	for name, changed := range map[string]plugin.BuilderIdentity{
		"seed executable": changedArtifact,
		"logical version": changedVersion,
		"build contract":  changedContract,
	} {
		if got := fingerprint(changed); got == baseFingerprint {
			t.Errorf("%s did not affect the stage plan fingerprint", name)
		}
	}
}

func TestTransformBootstrapStagePlanRejectsExecutionIdentityConfusion(t *testing.T) {
	source := bootstrapTestMetaSource(t)
	seed := plugin.BuilderIdentity{
		ID:              core.MustCanonicalID("tei.builder.transform_seed.test"),
		Version:         "1.0.0",
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	base := transformBootstrapStagePlan{
		Stage:               plugin.BootstrapStage1,
		Builder:             seed,
		SourceClosureDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		SourceFingerprint:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Execution: transformBootstrapStageExecution{
			Kind:   transformBootstrapExecutionNativeSeed,
			Target: "host/test/test",
		},
		Output: source.Identity,
	}
	engine := bootstrapTestEngine()

	tests := []struct {
		name string
		edit func(*transformBootstrapStagePlan)
	}{
		{"stage1 engine", func(plan *transformBootstrapStagePlan) {
			plan.Execution.RuntimeEngine = &engine
		}},
		{"stage1 interpreted kind", func(plan *transformBootstrapStagePlan) {
			plan.Execution.Kind = transformBootstrapExecutionInterpreted
		}},
		{"stage2 missing engine", func(plan *transformBootstrapStagePlan) {
			plan.Stage = plugin.BootstrapStage2
			plan.Execution.Kind = transformBootstrapExecutionInterpreted
		}},
		{"stage2 engine target mismatch", func(plan *transformBootstrapStagePlan) {
			plan.Stage = plugin.BootstrapStage2
			plan.Execution.Kind = transformBootstrapExecutionInterpreted
			plan.Execution.RuntimeEngine = &engine
			plan.Execution.Target = "host/other/test"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := base
			tt.edit(&plan)
			if _, err := transformBootstrapStagePlanFingerprint(plan); err == nil {
				t.Fatal("expected invalid stage execution evidence to fail")
			}
		})
	}
}
