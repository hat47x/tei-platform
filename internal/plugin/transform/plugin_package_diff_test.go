package transform_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

const (
	transformDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	transformDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	transformDigestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func transformPackageFixture() (plugin.Definition, plugin.BuildPlan) {
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.transform.package_test"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{
			{
				Kind:         plugin.CapabilityObservationProvider,
				ID:           core.MustCanonicalID("runtime.z_observer"),
				Observations: []deployment.ObservationKind{deployment.ObservationDatabaseState, deployment.ObservationApplicationRuntime},
			},
			{
				Kind:    plugin.CapabilityDeploymentExecutor,
				ID:      core.MustCanonicalID("runtime.a_executor"),
				Actions: []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionApplicationStart},
				Permissions: []permission.Spec{
					{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "z"},
					{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "m"},
					{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "a"},
				},
			},
		},
		Requires: []plugin.Requirement{
			{Plugin: core.MustCanonicalID("tei.z_dependency"), CompatibleMajor: 1},
			{Plugin: core.MustCanonicalID("tei.a_dependency"), CompatibleMajor: 2},
		},
		External: []plugin.ExternalRequirement{
			{Scope: "runtime", Kind: "tool", Name: "z"},
			{Scope: "build", Kind: "toolchain", Name: "a"},
		},
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{
			{Kind: "plugin-definition", ID: "z-source", Locator: "z.tei", Digest: transformDigestA},
			{Kind: "git-tree", ID: "a-source", Locator: "git:HEAD", Digest: transformDigestB},
		},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.transform_test"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  transformDigestC,
		},
		Recipe: plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "plugin.tei"},
		OutputKind: plugin.BuildOutputPluginPackage,
		Dependencies: []plugin.BuildDependency{
			{Kind: "tool", ID: "z-tool", Version: "2", ArtifactDigest: transformDigestA},
			{Kind: "sdk", ID: "a-sdk", Version: "1", ArtifactDigest: transformDigestB},
		},
		Target:  "plugin-package",
		Profile: "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "zeta", Value: "2"},
			{Name: "alpha", Value: "1"},
		},
	}
	return definition, plan
}

func valueFromGoJSON(t *testing.T, value any) transform.Value {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	got, err := transform.ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestPluginPackageTransformMatchesGoOracleByteForByte(t *testing.T) {
	definition, plan := transformPackageFixture()
	oracle, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		t.Fatal(err)
	}
	result, err := transform.Evaluate(transform.PluginPackageSourceV1(), map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": valueFromGoJSON(t, definition),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output.Kind != transform.ValueBytes {
		t.Fatalf("output kind: got %s", result.Output.Kind)
	}
	if !bytes.Equal(result.Output.Bytes, oracle) {
		t.Fatalf("TEI-native transform differs from Go oracle")
	}
}
