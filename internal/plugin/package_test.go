package plugin_test

import (
	"bytes"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/plugin"
)

func pluginPackagePlan(source plugin.Definition) plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: source.ID, Version: source.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(source),
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-definition",
			ID:      "source",
			Locator: "plugins/tei-runtime-reference/1.0.0/plugin.tei",
			Digest:  buildDigestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.seed"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  buildDigestB,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-plugin-package",
			Entrypoint: "plugins/tei-runtime-reference/1.0.0/plugin.tei",
		},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
	}
}

func artifactFreePackageSource() plugin.Definition {
	return plugin.Definition{
		ID:      core.MustCanonicalID("tei.runtime.reference"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDeploymentExecutor,
			ID:   core.MustCanonicalID("runtime.reference"),
			Actions: []deployment.ActionKind{
				deployment.ActionTrafficShift,
				deployment.ActionApplicationStart,
			},
		}},
	}
}

func TestMaterializePluginPackageIsDeterministicAcrossSemanticOrdering(t *testing.T) {
	a := artifactFreePackageSource()
	b := artifactFreePackageSource()
	b.Provides[0].Actions[0], b.Provides[0].Actions[1] = b.Provides[0].Actions[1], b.Provides[0].Actions[0]

	planA := pluginPackagePlan(a)
	planB := pluginPackagePlan(b)
	gotA, err := plugin.MaterializePluginPackage(a, planA)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := plugin.MaterializePluginPackage(b, planB)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, gotB) {
		t.Fatalf("semantic ordering changed package bytes:\n%s\n%s", gotA, gotB)
	}
	if plugin.BuildOutputDigest(gotA) != plugin.BuildOutputDigest(gotB) {
		t.Fatal("semantic ordering changed package digest")
	}
}

func TestMaterializePluginPackageBindsExactBuildPlan(t *testing.T) {
	source := artifactFreePackageSource()
	plan := pluginPackagePlan(source)
	first, err := plugin.MaterializePluginPackage(source, plan)
	if err != nil {
		t.Fatal(err)
	}

	changed := plan
	changed.Profile = "other"
	second, err := plugin.MaterializePluginPackage(source, changed)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("BuildPlan change did not change package bytes")
	}
}

func TestMaterializePluginPackageRejectsRuntimeArtifactAndWrongAuthority(t *testing.T) {
	t.Run("source runtime artifact", func(t *testing.T) {
		source := artifactFreePackageSource()
		source.Artifact = plugin.Artifact{Digest: buildDigestA}
		plan := pluginPackagePlan(source)
		if _, err := plugin.MaterializePluginPackage(source, plan); err == nil {
			t.Fatal("expected artifact-bearing source to fail")
		}
	})
	t.Run("runtime output kind", func(t *testing.T) {
		source := artifactFreePackageSource()
		plan := pluginPackagePlan(source)
		plan.OutputKind = plugin.BuildOutputRuntimeArtifact
		if _, err := plugin.MaterializePluginPackage(source, plan); err == nil {
			t.Fatal("expected runtime output kind to fail")
		}
	})
	t.Run("definition fingerprint", func(t *testing.T) {
		source := artifactFreePackageSource()
		plan := pluginPackagePlan(source)
		plan.PluginDefinitionFingerprint = buildFPB
		if _, err := plugin.MaterializePluginPackage(source, plan); err == nil {
			t.Fatal("expected definition fingerprint mismatch to fail")
		}
	})
	t.Run("identity", func(t *testing.T) {
		source := artifactFreePackageSource()
		plan := pluginPackagePlan(source)
		plan.Plugin.ID = core.MustCanonicalID("tei.other")
		if _, err := plugin.MaterializePluginPackage(source, plan); err == nil {
			t.Fatal("expected source/plan identity mismatch to fail")
		}
	})
}

func TestPluginPackageOutputCanBeLockedWithoutRuntimeBinding(t *testing.T) {
	source := artifactFreePackageSource()
	plan := pluginPackagePlan(source)
	pkg, err := plugin.MaterializePluginPackage(source, plan)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := plugin.LockBuildOutput(plan, plugin.BuildOutputDigest(pkg))
	if err != nil {
		t.Fatal(err)
	}
	if locked.OutputKind != plugin.BuildOutputPluginPackage {
		t.Fatalf("output kind: got %s", locked.OutputKind)
	}
	lock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{locked},
	}
	if _, err := plugin.BuildArtifactLockFingerprint(lock); err != nil {
		t.Fatal(err)
	}
}
