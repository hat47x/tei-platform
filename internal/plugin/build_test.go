package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	buildDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	buildDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	buildFPA      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	buildFPB      = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func buildPlanFixture() plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion:         plugin.BuildPlanFormatVersion,
		Plugin:                plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"},
		DescriptorFingerprint: buildFPA,
		Sources: []plugin.BuildSource{
			{Kind: "implementation", ID: "main", Locator: "tools/reference-process-plugin/main.go", Digest: buildDigestA},
			{Kind: "module-lock", ID: "go.sum", Locator: "go.sum", Digest: buildDigestB},
		},
		Builder: plugin.BuilderIdentity{
			ID:             core.MustCanonicalID("tei.builder.go"),
			Version:        "go1.26",
			ArtifactDigest: buildDigestA,
		},
		Dependencies: []plugin.BuildDependency{
			{ID: core.MustCanonicalID("tei.builder.sdk"), Version: "1.0.0", ArtifactDigest: buildDigestB},
			{ID: core.MustCanonicalID("tei.builder.support"), Version: "2.0.0", ArtifactDigest: buildDigestA},
		},
		Target:  "host-process",
		Profile: "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "trimpath", Value: "true"},
			{Name: "cgo", Value: "disabled"},
		},
	}
}

func TestBuildPlanFingerprintIsIndependentOfDeclarationOrder(t *testing.T) {
	a := buildPlanFixture()
	b := buildPlanFixture()
	b.Sources[0], b.Sources[1] = b.Sources[1], b.Sources[0]
	b.Dependencies[0], b.Dependencies[1] = b.Dependencies[1], b.Dependencies[0]
	b.Parameters[0], b.Parameters[1] = b.Parameters[1], b.Parameters[0]

	afp, err := plugin.BuildPlanFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	bfp, err := plugin.BuildPlanFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if afp != bfp {
		t.Fatalf("same resolved build inputs produced different fingerprints: %s != %s", afp, bfp)
	}
}

func TestBuildPlanFingerprintChangesWhenBuildInputChanges(t *testing.T) {
	base := buildPlanFixture()
	baseFP, err := plugin.BuildPlanFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		edit func(*plugin.BuildPlan)
	}{
		{"source digest", func(p *plugin.BuildPlan) { p.Sources[0].Digest = buildDigestB }},
		{"source locator", func(p *plugin.BuildPlan) { p.Sources[0].Locator = "other/main.go" }},
		{"builder", func(p *plugin.BuildPlan) { p.Builder.Version = "go1.26.1" }},
		{"dependency", func(p *plugin.BuildPlan) { p.Dependencies[0].Version = "1.0.1" }},
		{"target", func(p *plugin.BuildPlan) { p.Target = "wasm-component" }},
		{"parameter", func(p *plugin.BuildPlan) { p.Parameters[0].Value = "false" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := buildPlanFixture()
			tt.edit(&p)
			got, err := plugin.BuildPlanFingerprint(p)
			if err != nil {
				t.Fatal(err)
			}
			if got == baseFP {
				t.Fatalf("%s change did not change build plan fingerprint", tt.name)
			}
		})
	}
}

func TestBuildPlanRejectsAmbiguousOrUnpinnedInputs(t *testing.T) {
	t.Run("source digest required", func(t *testing.T) {
		p := buildPlanFixture()
		p.Sources[0].Digest = ""
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected missing source digest to fail")
		}
	})
	t.Run("builder digest required", func(t *testing.T) {
		p := buildPlanFixture()
		p.Builder.ArtifactDigest = ""
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected missing builder digest to fail")
		}
	})
	t.Run("duplicate source identity", func(t *testing.T) {
		p := buildPlanFixture()
		p.Sources[1].ID = p.Sources[0].ID
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected duplicate source identity to fail")
		}
	})
	t.Run("duplicate build dependency", func(t *testing.T) {
		p := buildPlanFixture()
		p.Dependencies[1].ID = p.Dependencies[0].ID
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected duplicate build dependency to fail")
		}
	})
	t.Run("duplicate parameter", func(t *testing.T) {
		p := buildPlanFixture()
		p.Parameters[1].Name = p.Parameters[0].Name
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected duplicate build parameter to fail")
		}
	})
}

func TestBuildLockFingerprintIsStableAndBindsArtifact(t *testing.T) {
	a := plugin.BuildLock{
		FormatVersion: plugin.BuildLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.z"), Version: "1.0.0"}, Target: "host-process", PlanFingerprint: buildFPA, ArtifactDigest: buildDigestA},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, Target: "wasm-component", Profile: "web", PlanFingerprint: buildFPB, ArtifactDigest: buildDigestB},
		},
	}
	b := a
	b.Builds = append([]plugin.LockedBuild(nil), a.Builds...)
	b.Builds[0], b.Builds[1] = b.Builds[1], b.Builds[0]

	afp, err := plugin.BuildLockFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	bfp, err := plugin.BuildLockFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if afp != bfp {
		t.Fatalf("same lock entries produced different fingerprints: %s != %s", afp, bfp)
	}

	changed := b
	changed.Builds = append([]plugin.LockedBuild(nil), b.Builds...)
	changed.Builds[0].ArtifactDigest = buildDigestA
	changedFP, err := plugin.BuildLockFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == afp {
		t.Fatal("artifact digest change did not change build lock fingerprint")
	}
}

func TestBuildLockRejectsDuplicatePluginTarget(t *testing.T) {
	l := plugin.BuildLock{
		FormatVersion: plugin.BuildLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, Target: "host-process", PlanFingerprint: buildFPA, ArtifactDigest: buildDigestA},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, Target: "host-process", PlanFingerprint: buildFPB, ArtifactDigest: buildDigestB},
		},
	}
	if _, err := plugin.BuildLockFingerprint(l); err == nil {
		t.Fatal("expected duplicate plugin/target build lock entry to fail")
	}
}
