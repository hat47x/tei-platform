package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
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
				Kind:              plugin.CapabilityObservationProvider,
				ID:                core.MustCanonicalID("runtime.z_observer"),
				Observations:      []deployment.ObservationKind{deployment.ObservationDatabaseState, deployment.ObservationApplicationRuntime},
				Evidence:          []deployment.EvidenceKind{deployment.EvidenceRolloutHealth},
				DatabaseProviders: []environment.DatabaseProvider{environment.ProviderSQLite, environment.ProviderPostgres},
				MigrationTools:    []migration.Tool{migration.ToolLiquibase, migration.ToolAtlas},
				Features:          []environment.Feature{environment.FeatureDatabaseWorkbench, environment.FeatureSQLAnalysis},
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
		Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "plugin.tei"},
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

func referencePluginPackageTransformSource(t *testing.T) transform.Source {
	t.Helper()
	source, _ := referenceTransformSource(t, "plugin-package-v1.transform.json")
	return source
}

func TestPluginPackageTransformMatchesGoOracleByteForByte(t *testing.T) {
	definition, plan := transformPackageFixture()
	oracle, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		t.Fatal(err)
	}
	result, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
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

// Repository Source is authoritative; editing its program must change the
// output without changing any builder-specific Go constructor.
func TestPluginPackageTransformRejectsDuplicateRequirements(t *testing.T) {
	for _, tc := range []struct {
		name   string
		majors [2]int
	}{
		{name: "same major", majors: [2]int{1, 1}},
		{name: "different majors", majors: [2]int{1, 2}},
		{name: "reversed conflicting majors", majors: [2]int{2, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			id := core.MustCanonicalID("tei.duplicate_dependency")
			definition.Requires = []plugin.Requirement{
				{Plugin: id, CompatibleMajor: tc.majors[0]},
				{Plugin: id, CompatibleMajor: tc.majors[1]},
			}
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			if _, err := plugin.MaterializePluginPackage(definition, plan); err == nil || !strings.Contains(err.Error(), "more than once") {
				t.Fatalf("Go materializer must reject repeated dependency, got %v", err)
			}
			if _, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan": valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			}); err == nil || !strings.Contains(err.Error(), "duplicate key") {
				t.Fatalf("TEI-native materializer must reject the duplicate key, got %v", err)
			}
		})
	}
}

func TestPluginPackageTransformValidRequirementPermutation(t *testing.T) {
	definition, plan := transformPackageFixture()
	reversed := definition
	reversed.Requires = []plugin.Requirement{definition.Requires[1], definition.Requires[0]}
	reversedPlan := plan
	reversedPlan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(reversed)
	goOriginal, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		t.Fatal(err)
	}
	goReversed, err := plugin.MaterializePluginPackage(reversed, reversedPlan)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(goOriginal, goReversed) {
		t.Fatal("valid requirement ordering changed Go package bytes")
	}
	teiReversed, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
		"buildPlan": valueFromGoJSON(t, reversedPlan),
		"pluginDefinition": valueFromGoJSON(t, reversed),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(teiReversed.Output.Bytes, goReversed) {
		t.Fatal("valid reordered requirements diverged from TEI-native builder")
	}
}

func TestReferencePluginPackageSourceControlsOutput(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	before, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	edited := source
	edited.Program.Output.Suffix = "\n#source-change\n"
	after, err := transform.SourceFingerprint(edited)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("builder Source change did not change its fingerprint")
	}
	definition, plan := transformPackageFixture()
	result, err := transform.Evaluate(edited, map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": valueFromGoJSON(t, definition),
	})
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.Output.Kind != transform.ValueBytes || bytes.Equal(result.Output.Bytes, oracle) {
		t.Fatal("editing TEI-owned builder Source did not change the produced bytes")
	}
}
