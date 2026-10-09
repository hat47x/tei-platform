package main

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

func bootstrapTestEngine() pruntime.RuntimeEngineIdentity {
	return pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.transform_test"),
		Version:              "revision-a",
		ContractVersion:      "tei.runtime-engine/deterministic-transform-v1",
		ImplementationDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Target:               "host/test/test",
	}
}

func TestTransformBootstrapInputFingerprintBindsClosureEngineAndTarget(t *testing.T) {
	base := transformBootstrapInputFingerprint(
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"host/test/test",
	)
	cases := []string{
		transformBootstrapInputFingerprint(
			"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"host/test/test",
		),
		transformBootstrapInputFingerprint(
			"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"host/test/test",
		),
		transformBootstrapInputFingerprint(
			"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			"host/test/test",
		),
		transformBootstrapInputFingerprint(
			"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"host/other/test",
		),
	}
	for i, got := range cases {
		if got == base {
			t.Fatalf("input fingerprint did not bind changed field %d", i)
		}
	}
}

func TestTransformBootstrapStagePlanFingerprintBindsBuilderAndEngine(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
	seed := plugin.BuilderIdentity{
		ID:              core.MustCanonicalID("tei.builder.transform_seed.test"),
		Version:         "seed-a",
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	base := transformBootstrapStagePlan{
		Stage:               plugin.BootstrapStage1,
		Builder:             seed,
		SourceClosureDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		SourceFingerprint:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		RuntimeEngine:       bootstrapTestEngine(),
		Target:              "host/test/test",
		Output:              source.Identity,
	}
	baseFP := transformBootstrapStagePlanFingerprint(base)

	changedBuilder := base
	changedBuilder.Builder.ArtifactDigest = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	if got := transformBootstrapStagePlanFingerprint(changedBuilder); got == baseFP {
		t.Fatal("stage plan fingerprint ignored builder identity")
	}

	changedEngine := base
	changedEngine.RuntimeEngine.ImplementationDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if got := transformBootstrapStagePlanFingerprint(changedEngine); got == baseFP {
		t.Fatal("stage plan fingerprint ignored Runtime Engine identity")
	}

	changedOutput := base
	changedOutput.Output.Version = "1.0.1"
	if got := transformBootstrapStagePlanFingerprint(changedOutput); got == baseFP {
		t.Fatal("stage plan fingerprint ignored logical output identity")
	}
}

func TestBuilderIdentityFromTransformSourcePreservesLogicalIdentity(t *testing.T) {
	source := transform.TransformSourceRealizerV1()
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
