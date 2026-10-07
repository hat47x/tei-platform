package workspace_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	pluginbuiltin "tei/internal/plugin/implementation/builtin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	nativeReferenceBuilderDigest = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	nativeReferenceEngineDigest  = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
)

func TestTEINativePluginImplementationReferenceRunsThroughExistingBuiltinABI(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "plugin-self-build", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Plugin self-build Referenceの読み込みに失敗しました: %v", err)
	}

	if len(result.PluginImplementations) != 1 {
		t.Fatalf("plugin implementations: %#v", result.PluginImplementations)
	}
	source := result.PluginImplementations[0]
	if source.Plugin.ID != "tei.native.runtime.reference" || source.CapabilityID != "runtime.native_reference" {
		t.Fatalf("implementation source: %#v", source)
	}

	sourceDefinition, found := exactReferencePlugin(result.World.Plugins, source.Plugin)
	if !found {
		t.Fatal("matching Plugin definition not found")
	}
	if sourceDefinition.Artifact.Digest != "" {
		t.Fatalf("authoritative source definition must be artifact-free: %q", sourceDefinition.Artifact.Digest)
	}

	const (
		descriptorRel     = "plugins/tei-native-runtime-reference/1.0.0/plugin.tei"
		implementationRel = "plugin-implementations/tei-native-runtime-reference/1.0.0/implementation.tei"
	)
	descriptorDigest := referenceFileDigest(t, filepath.Join(ws, filepath.FromSlash(descriptorRel)))
	implementationDigest := referenceFileDigest(t, filepath.Join(ws, filepath.FromSlash(implementationRel)))

	buildPlan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      source.Plugin,
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(sourceDefinition),
		Sources: []plugin.BuildSource{
			{
				Kind:    "plugin-definition",
				ID:      "tei-native-runtime-reference-definition",
				Locator: descriptorRel,
				Digest:  descriptorDigest,
			},
			{
				Kind:    "plugin-implementation",
				ID:      "tei-native-runtime-reference-implementation",
				Locator: implementationRel,
				Digest:  implementationDigest,
			},
		},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.native.seed"),
			Version:         "reference-test",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  nativeReferenceBuilderDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-native-static-response",
			Entrypoint: implementationRel,
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "interpreted-builtin/host",
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "execution-model", Value: "interpreted-builtin"},
		},
	}

	firstBytes, firstDigest, err := implementation.Materialize(source)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, secondDigest, err := implementation.Materialize(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) || firstDigest != secondDigest {
		t.Fatalf("TEI-native realization is not deterministic: %s != %s", firstDigest, secondDigest)
	}

	lockedBuild, err := plugin.LockBuildOutput(buildPlan, firstDigest)
	if err != nil {
		t.Fatal(err)
	}
	realizedDefinition, err := plugin.RealizeBuiltDefinition(sourceDefinition, firstDigest)
	if err != nil {
		t.Fatal(err)
	}

	catalog := plugin.Catalog{Plugins: append([]plugin.Definition(nil), result.World.Plugins...)}
	for i := range catalog.Plugins {
		if catalog.Plugins[i].ID == realizedDefinition.ID && catalog.Plugins[i].Version == realizedDefinition.Version {
			catalog.Plugins[i] = realizedDefinition
		}
	}
	if len(result.World.Environments) != 1 {
		t.Fatalf("environments: %#v", result.World.Environments)
	}
	env := result.World.Environments[0]
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	lock, err = plugin.BindRuntimeBuild(lock, env.ID, lockedBuild)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pruntime.BuildPlan(catalog, env, lock)
	if err != nil {
		t.Fatal(err)
	}
	engine := pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.static_response.reference"),
		Version:              "reference-test",
		ContractVersion:      "tei.runtime-engine/static-response-v1",
		ImplementationDigest: nativeReferenceEngineDigest,
		Target:               "host/reference",
	}
	engineLock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: realizedDefinition.ID, Version: realizedDefinition.Version},
			Engine: engine,
		}},
	}
	plan, err = pruntime.BindRuntimeEngineLock(plan, engineLock)
	if err != nil {
		t.Fatal(err)
	}

	builtin, err := pluginbuiltin.New(firstBytes, realizedDefinition)
	if err != nil {
		t.Fatal(err)
	}
	registry := pruntime.Registry{
		BuiltinRegistrations: map[string]pruntime.BuiltinRegistration{
			"tei.native.runtime.reference@1.0.0": {
				Builtin:              builtin,
				BuildPlanFingerprint: lockedBuild.PlanFingerprint,
				RealizationDigest:    firstDigest,
				RuntimeEngine:        engine,
			},
		},
	}
	adapter := pruntime.DeploymentExecutor{
		Plan:         plan,
		Registry:     registry,
		CapabilityID: core.MustCanonicalID("runtime.native_reference"),
	}
	command := executor.Command{
		ActionID:       "application/start",
		IdempotencyKey: "native-reference-key",
		Kind:           string(deployment.ActionApplicationStart),
	}
	got, err := adapter.Execute(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if got.AttemptID != "tei-native-reference" || got.Outcome != executor.OutcomeAccepted {
		t.Fatalf("result: %#v", got)
	}
	if got.ActionID != command.ActionID || got.IdempotencyKey != command.IdempotencyKey {
		t.Fatalf("request identity was not preserved by adapter: %#v", got)
	}
	engineFingerprint, err := pruntime.RuntimeEngineIdentityFingerprint(engine)
	if err != nil {
		t.Fatal(err)
	}
	if got.PluginInvocation == nil ||
		got.PluginInvocation.BuildPlanFingerprint != lockedBuild.PlanFingerprint ||
		got.PluginInvocation.RealizationDigest != firstDigest ||
		got.PluginInvocation.RuntimeEngineFingerprint != engineFingerprint {
		t.Fatalf("build realization/runtime engine provenance: %#v", got.PluginInvocation)
	}
	if err := pruntime.ValidateDeploymentInvocation(plan, core.MustCanonicalID("runtime.native_reference"), got.PluginInvocation); err != nil {
		t.Fatal(err)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatal(err)
	}
	var implementationIndexed bool
	for _, ref := range idx.References {
		if ref.Subject.Kind != "plugin-implementation" || ref.Subject.ID != source.Identity() {
			continue
		}
		if ref.Role == sourceindex.RoleDefinition {
			t.Fatal("TEI-native implementation source must not be indexed as canonical definition")
		}
		if ref.Role == sourceindex.RoleImplementation {
			implementationIndexed = true
		}
	}
	if !implementationIndexed {
		t.Fatal("TEI-native implementation source is not indexed as implementation evidence")
	}
}

func exactReferencePlugin(definitions []plugin.Definition, identity plugin.Identity) (plugin.Definition, bool) {
	for _, definition := range definitions {
		if definition.ID == identity.ID && definition.Version == identity.Version {
			return definition, true
		}
	}
	return plugin.Definition{}, false
}

func referenceFileDigest(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
