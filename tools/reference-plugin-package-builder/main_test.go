package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/syntax"
)

func TestReferencePluginPackageBuilderMaterializesDeterministicPackage(t *testing.T) {
	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.tei")
	planPath := filepath.Join(root, "plan.json")
	firstPath := filepath.Join(root, "first.json")
	secondPath := filepath.Join(root, "second.json")

	const pluginSource = `plugin tei.package.builder_test v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.package_test {
        action application-start
    }
}
`
	if err := os.WriteFile(pluginPath, []byte(pluginSource), 0o644); err != nil {
		t.Fatal(err)
	}
	definition, err := syntax.NewParser(pluginSource).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-definition",
			ID:      "package-builder-test-definition",
			Locator: "plugin.tei",
			Digest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.reference.seed"),
			Version:         "test",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-plugin-package",
			Entrypoint: "plugin.tei",
		},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, planBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run(pluginPath, planPath, firstPath); err != nil {
		t.Fatal(err)
	}
	if err := run(pluginPath, planPath, secondPath); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("Plugin package differs:\n%s\n%s", first, second)
	}
	var manifest plugin.PluginPackageManifest
	if err := json.Unmarshal(first, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Plugin.ID != definition.ID || manifest.BuildPlanFingerprint == "" {
		t.Fatalf("package manifest: %#v", manifest)
	}
}

func TestReferencePluginPackageBuilderRejectsWrongBuildPlan(t *testing.T) {
	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.tei")
	planPath := filepath.Join(root, "plan.json")
	outputPath := filepath.Join(root, "out.json")

	const pluginSource = `plugin tei.package.builder_test v1.0.0 {
    runtime builtin
}
`
	if err := os.WriteFile(pluginPath, []byte(pluginSource), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.other"), Version: "1.0.0"},
		PluginDefinitionFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-definition",
			ID:      "package-builder-test-definition",
			Locator: "plugin.tei",
			Digest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.reference.seed"),
			Version:         "test",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "plugin.tei"},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, planBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run(pluginPath, planPath, outputPath); err == nil {
		t.Fatal("expected mismatched Build Plan to fail")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("package builder created output after failed validation: %v", err)
	}
}
