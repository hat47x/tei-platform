package workspace_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/workspace"
)

const packageBuilderDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

func TestReferenceRuntimePluginCanMaterializeSB1Package(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Reference Workspaceの読み込みに失敗しました: %v", err)
	}

	var source plugin.Definition
	var found bool
	for _, definition := range result.World.Plugins {
		if definition.ID == "tei.runtime.reference" && definition.Version == "1.0.0" {
			source = definition
			found = true
			break
		}
	}
	if !found {
		t.Fatal("tei.runtime.reference@1.0.0 not found")
	}
	if source.Artifact.Digest != "" {
		t.Fatalf("SB-1 source must be artifact-free: %q", source.Artifact.Digest)
	}

	const rel = "plugins/tei-runtime-reference/1.0.0/plugin.tei"
	src, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(src)
	sourceDigest := "sha256:" + hex.EncodeToString(sum[:])

	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: source.ID, Version: source.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(source),
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-definition",
			ID:      "tei-runtime-reference-definition",
			Locator: rel,
			Digest:  sourceDigest,
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.seed"),
			Version:         "reference-test",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  packageBuilderDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-plugin-package",
			Entrypoint: rel,
		},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
	}
	first, err := plugin.MaterializePluginPackage(source, plan)
	if err != nil {
		t.Fatal(err)
	}
	second, err := plugin.MaterializePluginPackage(source, plan)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("same Reference source produced different SB-1 package bytes")
	}

	locked, err := plugin.LockBuildOutput(plan, plugin.BuildOutputDigest(first))
	if err != nil {
		t.Fatal(err)
	}
	if locked.Plugin != plan.Plugin || locked.OutputKind != plugin.BuildOutputPluginPackage {
		t.Fatalf("locked package output: %#v", locked)
	}
}
