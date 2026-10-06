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

func buildDefinitionFixture() plugin.Definition {
	return plugin.Definition{
		ID:      core.MustCanonicalID("tei.process.reference"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "reference-plugin",
		},
		Artifact: plugin.Artifact{Digest: buildDigestB},
	}
}

func buildPlanFixture() plugin.BuildPlan {
	definition := buildDefinitionFixture()
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
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
	a := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.z"), Version: "1.0.0"}, Target: "host-process", PlanFingerprint: buildFPA, ArtifactDigest: buildDigestA},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, Target: "wasm-component", Profile: "web", PlanFingerprint: buildFPB, ArtifactDigest: buildDigestB},
		},
	}
	b := a
	b.Builds = append([]plugin.LockedBuild(nil), a.Builds...)
	b.Builds[0], b.Builds[1] = b.Builds[1], b.Builds[0]

	afp, err := plugin.BuildArtifactLockFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	bfp, err := plugin.BuildArtifactLockFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if afp != bfp {
		t.Fatalf("same lock entries produced different fingerprints: %s != %s", afp, bfp)
	}

	changed := b
	changed.Builds = append([]plugin.LockedBuild(nil), b.Builds...)
	changed.Builds[0].ArtifactDigest = buildDigestA
	changedFP, err := plugin.BuildArtifactLockFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == afp {
		t.Fatal("artifact digest change did not change build lock fingerprint")
	}
}

func TestBuildLockRejectsDuplicatePluginTarget(t *testing.T) {
	l := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, Target: "host-process", PlanFingerprint: buildFPA, ArtifactDigest: buildDigestA},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, Target: "host-process", PlanFingerprint: buildFPB, ArtifactDigest: buildDigestB},
		},
	}
	if _, err := plugin.BuildArtifactLockFingerprint(l); err == nil {
		t.Fatal("expected duplicate plugin/target build lock entry to fail")
	}
}


func TestLockBuildOutputAndVerifyRuntimePluginBuild(t *testing.T) {
	plan := buildPlanFixture()
	locked, err := plugin.LockBuildOutput(plan, buildDigestB)
	if err != nil {
		t.Fatal(err)
	}
	planFP, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	if locked.PlanFingerprint != planFP {
		t.Fatalf("plan fingerprint: got %s want %s", locked.PlanFingerprint, planFP)
	}
	definition := buildDefinitionFixture()
	runtime := plugin.LockedPlugin{
		ID:             plan.Plugin.ID,
		Version:        plan.Plugin.Version,
		Fingerprint:    plugin.DefinitionFingerprint(definition),
		ArtifactDigest: buildDigestB,
	}
	if err := plugin.VerifyRuntimePluginBuild(plan, locked, definition, runtime); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRuntimePluginBuildFailsClosedOnDivergence(t *testing.T) {
	plan := buildPlanFixture()
	locked, err := plugin.LockBuildOutput(plan, buildDigestB)
	if err != nil {
		t.Fatal(err)
	}
	definition := buildDefinitionFixture()
	baseRuntime := plugin.LockedPlugin{
		ID:             plan.Plugin.ID,
		Version:        plan.Plugin.Version,
		Fingerprint:    plugin.DefinitionFingerprint(definition),
		ArtifactDigest: buildDigestB,
	}

	tests := []struct {
		name string
		edit func(*plugin.BuildPlan, *plugin.LockedBuild, *plugin.LockedPlugin)
	}{
		{"runtime identity", func(_ *plugin.BuildPlan, _ *plugin.LockedBuild, r *plugin.LockedPlugin) {
			r.ID = core.MustCanonicalID("tei.other")
		}},
		{"descriptor fingerprint", func(_ *plugin.BuildPlan, _ *plugin.LockedBuild, r *plugin.LockedPlugin) {
			r.Fingerprint = buildFPB
		}},
		{"plan fingerprint", func(_ *plugin.BuildPlan, b *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			b.PlanFingerprint = buildFPB
		}},
		{"target", func(_ *plugin.BuildPlan, b *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			b.Target = "wasm-component"
		}},
		{"artifact", func(_ *plugin.BuildPlan, _ *plugin.LockedBuild, r *plugin.LockedPlugin) {
			r.ArtifactDigest = buildDigestA
		}},
		{"missing runtime artifact", func(_ *plugin.BuildPlan, _ *plugin.LockedBuild, r *plugin.LockedPlugin) {
			r.ArtifactDigest = ""
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := plan
			b := locked
			r := baseRuntime
			tt.edit(&p, &b, &r)
			if err := plugin.VerifyRuntimePluginBuild(p, b, definition, r); err == nil {
				t.Fatal("expected divergence to fail")
			}
		})
	}
}


func TestBuildDefinitionFingerprintExcludesProducedArtifact(t *testing.T) {
	a := buildDefinitionFixture()
	b := a
	b.Artifact = plugin.Artifact{Digest: buildDigestA}

	if got, want := plugin.BuildDefinitionFingerprint(a), plugin.BuildDefinitionFingerprint(b); got != want {
		t.Fatalf("build definition fingerprint changed with produced artifact: %s != %s", got, want)
	}
	if plugin.DefinitionFingerprint(a) == plugin.DefinitionFingerprint(b) {
		t.Fatal("runtime definition fingerprint did not change with artifact")
	}
}
