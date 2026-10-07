package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	bootstrapDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	bootstrapDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	bootstrapSource  = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	bootstrapPlan1   = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	bootstrapPlan2   = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

func bootstrapBuilder(id, version, digest string) plugin.BuilderIdentity {
	return plugin.BuilderIdentity{
		ID:              core.MustCanonicalID(id),
		Version:         version,
		ContractVersion: "tei.plugin-build/v1",
		ArtifactDigest:  digest,
	}
}

func bootstrapEvidenceFixture() plugin.BootstrapConvergenceEvidence {
	seed := bootstrapBuilder("tei.builder.seed", "go1.23.2", bootstrapDigestB)
	stage1Output := bootstrapBuilder("tei.builder.native", "current", bootstrapDigestA)
	return plugin.BootstrapConvergenceEvidence{
		FormatVersion: plugin.BootstrapConvergenceFormatVersion,
		Stage0Seed:    seed,
		Stage1: plugin.BootstrapStageEvidence{
			Stage:                plugin.BootstrapStage1,
			Builder:              seed,
			SourceClosureDigest:  bootstrapSource,
			BuildPlanFingerprint: bootstrapPlan1,
			Output:               stage1Output,
		},
		Stage2: plugin.BootstrapStageEvidence{
			Stage:                plugin.BootstrapStage2,
			Builder:              stage1Output,
			SourceClosureDigest:  bootstrapSource,
			BuildPlanFingerprint: bootstrapPlan2,
			Output:               stage1Output,
		},
	}
}

func TestEvaluateBootstrapConvergenceAcceptsExactStageChain(t *testing.T) {
	evidence := bootstrapEvidenceFixture()
	result, err := plugin.EvaluateBootstrapConvergence(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Converged {
		t.Fatalf("expected exact stage convergence: %#v", result)
	}
	if result.Stage1ArtifactDigest != bootstrapDigestA ||
		result.Stage2ArtifactDigest != bootstrapDigestA {
		t.Fatalf("unexpected convergence digests: %#v", result)
	}
}

func TestEvaluateBootstrapConvergencePreservesNonConvergence(t *testing.T) {
	evidence := bootstrapEvidenceFixture()
	evidence.Stage2.Output.ArtifactDigest = bootstrapDigestB

	result, err := plugin.EvaluateBootstrapConvergence(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if result.Converged {
		t.Fatal("different stage1/stage2 artifact digests were reported as converged")
	}
	if result.Stage1ArtifactDigest == result.Stage2ArtifactDigest {
		t.Fatalf("non-convergence evidence lost digest difference: %#v", result)
	}
}

func TestBootstrapConvergenceRejectsIncomparableStages(t *testing.T) {
	tests := []struct {
		name string
		edit func(*plugin.BootstrapConvergenceEvidence)
	}{
		{"format", func(e *plugin.BootstrapConvergenceEvidence) {
			e.FormatVersion = 99
		}},
		{"stage1 label", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage1.Stage = plugin.BootstrapStage2
		}},
		{"stage0 seed", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage1.Builder.ArtifactDigest = bootstrapDigestA
		}},
		{"stage2 builder", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage2.Builder.ArtifactDigest = bootstrapDigestB
		}},
		{"source closure", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage2.SourceClosureDigest = bootstrapDigestB
		}},
		{"output builder id", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage2.Output.ID = core.MustCanonicalID("tei.builder.other")
		}},
		{"output builder version", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage2.Output.Version = "other"
		}},
		{"output builder contract", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage2.Output.ContractVersion = "tei.plugin-build/v2"
		}},
		{"source digest", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage1.SourceClosureDigest = "sha256:bad"
		}},
		{"plan fingerprint", func(e *plugin.BootstrapConvergenceEvidence) {
			e.Stage2.BuildPlanFingerprint = "bad"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := bootstrapEvidenceFixture()
			tt.edit(&evidence)
			if _, err := plugin.EvaluateBootstrapConvergence(evidence); err == nil {
				t.Fatal("expected incomparable bootstrap stages to fail")
			}
		})
	}
}

func TestBootstrapConvergenceDoesNotTreatSeedTrustAsConvergence(t *testing.T) {
	evidence := bootstrapEvidenceFixture()
	result, err := plugin.EvaluateBootstrapConvergence(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Converged {
		t.Fatal("fixture must converge")
	}

	// Convergence says only that stage1 and stage2 rebuild the same builder
	// artifact from the same source closure. The trusted stage0 seed remains
	// explicit in the Evidence and is not erased by convergence.
	if evidence.Stage0Seed.ArtifactDigest == evidence.Stage1.Output.ArtifactDigest {
		t.Fatal("fixture unexpectedly collapsed stage0 seed into current builder")
	}
}
