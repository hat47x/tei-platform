package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestTransformSourceMutationPropagatesToBuildPlanAndEvidence(t *testing.T) {
	root := t.TempDir()
	put := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("internal/core/id.go", "id-v1")
	put("internal/core/version.go", "version-v1")
	put("internal/plugin/transform/eval.go", "evaluator-v1")
	put("internal/plugin/transform/source_budget.go", "budget-v1")

	const digestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const digestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const digestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	base := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.example.plugin"), Version: "1.0.0"},
		PluginDefinitionFingerprint: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Sources: []plugin.BuildSource{{
			Kind: "source", ID: "plugin-source", Locator: "reference/test.json", Digest: digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.original"), Version: "1.0.0",
			ContractVersion: "tei.plugin-build/v1", ArtifactDigest: digestB,
		},
		Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "reference/test.json"},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
		Dependencies: []plugin.BuildDependency{{
			Kind: "source", ID: "fixed-input", Version: "1.0.0", ArtifactDigest: digestA,
		}},
		Parameters: []plugin.BuildParameter{{Name: "encoding", Value: "json-v1"}},
	}
	builder := plugin.BuilderIdentity{
		ID: core.MustCanonicalID("tei.builder.transform_test"), Version: "1.0.0",
		ContractVersion: "tei.plugin-build/v1", ArtifactDigest: digestC,
	}
	type result struct {
		sourceDigest string
		engine       pruntime.RuntimeEngineIdentity
		engineFP     string
		plan         plugin.BuildPlan
		planFP       string
		evidence     transformBootstrapEvidence
	}
	build := func(toolchain string) result {
		t.Helper()
		sources, err := transformRuntimeEngineSources(root)
		if err != nil {
			t.Fatal(err)
		}
		sourceDigest, err := hashSourceFiles(root, sources)
		if err != nil {
			t.Fatal(err)
		}
		engine, err := referenceRuntimeEngineIdentity(
			core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
			transformRuntimeEngineVersion, "tei.runtime-engine/deterministic-transform-v1",
			sourceDigest, toolchain, "host/"+runtime.GOOS+"/"+runtime.GOARCH,
		)
		if err != nil {
			t.Fatal(err)
		}
		engineFP, err := pruntime.RuntimeEngineIdentityFingerprint(engine)
		if err != nil {
			t.Fatal(err)
		}
		plan, planFP, err := transformPackageBuildPlan(base, builder, engine, engineFP)
		if err != nil {
			t.Fatal(err)
		}
		if planFP == "" {
			t.Fatal("missing Build Plan fingerprint")
		}
		if got, err := plugin.BuildPlanFingerprint(plan); err != nil || got != planFP {
			t.Fatalf("Build Plan fingerprint changed when re-read: %q, %v", got, err)
		}
		var runtimeDep *plugin.BuildDependency
		for i := range plan.Dependencies {
			if plan.Dependencies[i].Kind == "runtime-engine" {
				runtimeDep = &plan.Dependencies[i]
			}
		}
		if runtimeDep == nil || runtimeDep.ID != string(engine.ID) ||
			runtimeDep.Version != engine.Version || runtimeDep.ArtifactDigest != engine.ImplementationDigest {
			t.Fatal("Runtime Engine identity was lost from the Build Plan dependencies")
		}
		parameters := map[string]string{}
		for _, param := range plan.Parameters {
			parameters[param.Name] = param.Value
		}
		if parameters["runtime-engine-contract"] != engine.ContractVersion ||
			parameters["runtime-engine-fingerprint"] != engineFP ||
			parameters["runtime-engine-target"] != engine.Target {
			t.Fatal("Runtime Engine identity was lost from Build Plan parameters")
		}
		var evidence transformBootstrapEvidence
		evidence.RuntimeEngineSourceDigest = sourceDigest
		evidence.RuntimeEngine = engine
		evidence.RuntimeEngineFingerprint = engineFP
		evidence.PackageBuildPlan = plan
		evidence.PackageBuildPlanFingerprint = planFP
		return result{sourceDigest, engine, engineFP, plan, planFP, evidence}
	}

	first := build(digestA)
	put("docs/unrelated.md", "changed documentation")
	unrelated := build(digestA)
	if first.sourceDigest != unrelated.sourceDigest ||
		first.engine != unrelated.engine || first.engineFP != unrelated.engineFP ||
		first.planFP != unrelated.planFP {
		t.Fatal("documentation changed the Runtime Engine or Build Plan identity")
	}

	put("internal/plugin/transform/source_budget.go", "budget-v2")
	second := build(digestA)
	if first.sourceDigest == second.sourceDigest ||
		first.engine.ImplementationDigest == second.engine.ImplementationDigest ||
		first.engineFP == second.engineFP || first.planFP == second.planFP {
		t.Fatal("resource-bound source change failed to reach Runtime Engine and Build Plan")
	}
	if first.evidence.RuntimeEngineSourceDigest != first.sourceDigest ||
		second.evidence.RuntimeEngine.ImplementationDigest != second.engine.ImplementationDigest ||
		second.evidence.RuntimeEngineFingerprint != second.engineFP ||
		second.evidence.PackageBuildPlanFingerprint != second.planFP {
		t.Fatal("Build Plan or Runtime Engine identity was lost from Evidence")
	}
	toolchainChange := build(digestB)
	if second.sourceDigest != toolchainChange.sourceDigest ||
		second.engineFP == toolchainChange.engineFP ||
		second.planFP == toolchainChange.planFP {
		t.Fatal("toolchain change did not invalidate Engine fingerprint and Build Plan")
	}
	if len(base.Dependencies) != 1 || len(base.Parameters) != 1 {
		t.Fatal("derived plan changed the base plan")
	}
}

func TestTransformPackageBuildPlanRejectsStaleEngineFingerprint(t *testing.T) {
	const source = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const toolchain = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	engine, err := referenceRuntimeEngineIdentity(
		core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
		transformRuntimeEngineVersion, "tei.runtime-engine/deterministic-transform-v1",
		source, toolchain, "host/"+runtime.GOOS+"/"+runtime.GOARCH,
	)
	if err != nil {
		t.Fatal(err)
	}
	engineFP, err := pruntime.RuntimeEngineIdentityFingerprint(engine)
	if err != nil {
		t.Fatal(err)
	}
	engine.ImplementationDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, _, err := transformPackageBuildPlan(plugin.BuildPlan{}, plugin.BuilderIdentity{}, engine, engineFP); err == nil {
		t.Fatal("stale fingerprint accepted after Runtime Engine identity change")
	}
}
