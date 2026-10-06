package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/environment"
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
			Version:         "go1.26",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  buildDigestA,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "go-package",
			Entrypoint: "./tools/reference-process-plugin",
		},
		OutputKind:   plugin.BuildOutputRuntimeArtifact,
		Dependencies: []plugin.BuildDependency{
			{Kind: "sdk", ID: "example.builder.sdk", Version: "1.0.0", ArtifactDigest: buildDigestB},
			{Kind: "support", ID: "example.builder.support", Version: "2.0.0", ArtifactDigest: buildDigestA},
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
		{"builder version", func(p *plugin.BuildPlan) { p.Builder.Version = "go1.26.1" }},
		{"builder contract", func(p *plugin.BuildPlan) { p.Builder.ContractVersion = "tei.plugin-build/v2" }},
		{"dependency", func(p *plugin.BuildPlan) { p.Dependencies[0].Version = "1.0.1" }},
		{"recipe", func(p *plugin.BuildPlan) { p.Recipe.Entrypoint = "./tools/other-plugin" }},
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
	t.Run("builder contract required", func(t *testing.T) {
		p := buildPlanFixture()
		p.Builder.ContractVersion = ""
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected missing builder contract version to fail")
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
	t.Run("recipe required", func(t *testing.T) {
		p := buildPlanFixture()
		p.Recipe = plugin.BuildRecipe{}
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected missing build recipe to fail")
		}
	})
	t.Run("output kind required", func(t *testing.T) {
		p := buildPlanFixture()
		p.OutputKind = ""
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected missing build output kind to fail")
		}
	})
	t.Run("invalid plugin id", func(t *testing.T) {
		p := buildPlanFixture()
		p.Plugin.ID = core.CanonicalID("Invalid")
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected invalid plugin id to fail")
		}
	})
	t.Run("invalid builder id", func(t *testing.T) {
		p := buildPlanFixture()
		p.Builder.ID = core.CanonicalID("Invalid")
		if _, err := plugin.BuildPlanFingerprint(p); err == nil {
			t.Fatal("expected invalid builder id to fail")
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
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.z"), Version: "1.0.0"}, OutputKind: plugin.BuildOutputRuntimeArtifact, Target: "host-process", PlanFingerprint: buildFPA, ArtifactDigest: buildDigestA},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, OutputKind: plugin.BuildOutputRuntimeArtifact, Target: "wasm-component", Profile: "web", PlanFingerprint: buildFPB, ArtifactDigest: buildDigestB},
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

func TestBuildArtifactLockRejectsInvalidPluginID(t *testing.T) {
	l := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{{
			Plugin:          plugin.Identity{ID: core.CanonicalID("Invalid"), Version: "1.0.0"},
			OutputKind:      plugin.BuildOutputRuntimeArtifact,
			Target:          "host-process",
			PlanFingerprint: buildFPA,
			ArtifactDigest:  buildDigestA,
		}},
	}
	if _, err := plugin.BuildArtifactLockFingerprint(l); err == nil {
		t.Fatal("expected invalid locked plugin id to fail")
	}
}

func TestBuildLockRejectsDuplicatePluginTarget(t *testing.T) {
	l := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, OutputKind: plugin.BuildOutputRuntimeArtifact, Target: "host-process", PlanFingerprint: buildFPA, ArtifactDigest: buildDigestA},
			{Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"}, OutputKind: plugin.BuildOutputRuntimeArtifact, Target: "host-process", PlanFingerprint: buildFPB, ArtifactDigest: buildDigestB},
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
		{"output kind", func(_ *plugin.BuildPlan, b *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			b.OutputKind = plugin.BuildOutputPluginPackage
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


func TestRealizeBuiltDefinitionFeedsExistingRuntimePluginLock(t *testing.T) {
	source := buildDefinitionFixture()
	source.Artifact = plugin.Artifact{}
	sourceBuildFP := plugin.BuildDefinitionFingerprint(source)
	sourceRuntimeFP := plugin.DefinitionFingerprint(source)

	realized, err := plugin.RealizeBuiltDefinition(source, buildDigestB)
	if err != nil {
		t.Fatal(err)
	}
	if source.Artifact.Digest != "" {
		t.Fatal("source definition was mutated")
	}
	if realized.Artifact.Digest != buildDigestB {
		t.Fatalf("realized artifact digest: got %s", realized.Artifact.Digest)
	}
	if plugin.BuildDefinitionFingerprint(realized) != sourceBuildFP {
		t.Fatal("build definition identity changed after artifact realization")
	}
	if plugin.DefinitionFingerprint(realized) == sourceRuntimeFP {
		t.Fatal("runtime definition identity did not bind produced artifact")
	}

	env := environment.Environment{
		ID: core.MustCanonicalID("reference"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              source.ID,
			CompatibleMajor: 1,
		}},
	}
	runtimeLock, err := plugin.BuildLock(plugin.Catalog{Plugins: []plugin.Definition{realized}}, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeLock.Environments) != 1 || len(runtimeLock.Environments[0].Plugins) != 1 {
		t.Fatalf("runtime lock: %#v", runtimeLock)
	}
	locked := runtimeLock.Environments[0].Plugins[0]
	if locked.ArtifactDigest != buildDigestB {
		t.Fatalf("runtime lock artifact: got %s", locked.ArtifactDigest)
	}

	plan := buildPlanFixture()
	buildOutput, err := plugin.LockBuildOutput(plan, buildDigestB)
	if err != nil {
		t.Fatal(err)
	}
	if err := plugin.VerifyRuntimePluginBuild(plan, buildOutput, realized, locked); err != nil {
		t.Fatal(err)
	}
}

func TestRealizeBuiltDefinitionRejectsPredeclaredArtifactMismatch(t *testing.T) {
	source := buildDefinitionFixture()
	source.Artifact = plugin.Artifact{Digest: buildDigestA}

	if _, err := plugin.RealizeBuiltDefinition(source, buildDigestB); err == nil {
		t.Fatal("expected predeclared artifact mismatch to fail")
	}
	if _, err := plugin.RealizeBuiltDefinition(source, buildDigestA); err != nil {
		t.Fatal(err)
	}
}


func TestVerifyRuntimePluginBuildRejectsPackageOutput(t *testing.T) {
	plan := buildPlanFixture()
	plan.OutputKind = plugin.BuildOutputPluginPackage
	locked, err := plugin.LockBuildOutput(plan, buildDigestB)
	if err != nil {
		t.Fatal(err)
	}
	definition := buildDefinitionFixture()
	runtime := plugin.LockedPlugin{
		ID:             plan.Plugin.ID,
		Version:        plan.Plugin.Version,
		Fingerprint:    plugin.DefinitionFingerprint(definition),
		ArtifactDigest: buildDigestB,
	}
	if err := plugin.VerifyRuntimePluginBuild(plan, locked, definition, runtime); err == nil {
		t.Fatal("expected plugin-package output to be rejected as Runtime artifact")
	}
}

func TestBuildArtifactLockAllowsDifferentOutputKinds(t *testing.T) {
	l := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{
				Plugin:          plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"},
				OutputKind:      plugin.BuildOutputRuntimeArtifact,
				Target:          "host-process",
				PlanFingerprint: buildFPA,
				ArtifactDigest:  buildDigestA,
			},
			{
				Plugin:          plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"},
				OutputKind:      plugin.BuildOutputPluginPackage,
				Target:          "host-process",
				PlanFingerprint: buildFPB,
				ArtifactDigest:  buildDigestB,
			},
		},
	}
	if _, err := plugin.BuildArtifactLockFingerprint(l); err != nil {
		t.Fatal(err)
	}
}


func TestBuildDefinitionFingerprintDoesNotMutateSourceOrdering(t *testing.T) {
	source := buildDefinitionFixture()
	source.Runtime.Args = []string{"second", "first"}
	source.Provides = []plugin.Capability{{
		Kind: plugin.CapabilityDeploymentExecutor,
		ID:   core.MustCanonicalID("runtime.reference"),
		Actions: []deployment.ActionKind{
			deployment.ActionTrafficShift,
			deployment.ActionApplicationStart,
		},
	}}
	beforeFirst := source.Provides[0].Actions[0]
	beforeSecond := source.Provides[0].Actions[1]
	beforeArg0 := source.Runtime.Args[0]
	beforeArg1 := source.Runtime.Args[1]

	_ = plugin.BuildDefinitionFingerprint(source)

	if source.Provides[0].Actions[0] != beforeFirst || source.Provides[0].Actions[1] != beforeSecond {
		t.Fatalf("build fingerprint mutated capability action ordering: %#v", source.Provides[0].Actions)
	}
	if source.Runtime.Args[0] != beforeArg0 || source.Runtime.Args[1] != beforeArg1 {
		t.Fatalf("build fingerprint mutated runtime args: %#v", source.Runtime.Args)
	}
}
