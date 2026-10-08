package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/deployment"
	"tei/internal/plugin"
	"tei/internal/workspace"
)

func TestPluginSelfBuildReferenceLoadsArtifactFreeSourceDefinition(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "plugin-self-build", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatalf("Plugin self-build Referenceの読み込みに失敗しました: %v", err)
	}
	// The Reference declares the process Plugin (SB-2) and the TEI-native builtin Plugin (SB-3).
	if len(result.World.Plugins) != 2 {
		t.Fatalf("plugins: %#v", result.World.Plugins)
	}
	var definition plugin.Definition
	for _, candidate := range result.World.Plugins {
		if candidate.ID == "tei.process.reference" {
			definition = candidate
		}
	}
	if definition.ID != "tei.process.reference" || definition.Version != "1.0.0" {
		t.Fatalf("plugin identity: %#v", definition)
	}
	if definition.Artifact.Digest != "" {
		t.Fatalf("source definition must not predeclare build output: %q", definition.Artifact.Digest)
	}
	if definition.Runtime.Kind != plugin.RuntimeProcess || definition.Runtime.Command != "tei-process-reference" {
		t.Fatalf("runtime: %#v", definition.Runtime)
	}
	if len(definition.Runtime.Args) != 1 || definition.Runtime.Args[0] != "--tei-reference-marker=argv value preserved" {
		t.Fatalf("runtime args: %#v", definition.Runtime.Args)
	}

	var executorFound bool
	for _, capability := range definition.Provides {
		if capability.Kind == plugin.CapabilityDeploymentExecutor &&
			capability.ID == "process.reference" &&
			capability.SupportsAction(deployment.ActionDatabaseRelease) {
			executorFound = true
			break
		}
	}
	if !executorFound {
		t.Fatalf("deployment executor capability not found: %#v", definition.Provides)
	}

	location, ok := result.Sources["plugin:tei.process.reference@1.0.0"]
	if !ok {
		t.Fatal("plugin source location not indexed")
	}
	if location.File != "plugins/tei-process-reference/1.0.0/plugin.tei" {
		t.Fatalf("plugin source location: %#v", location)
	}

	if len(result.World.Environments) != 1 || result.World.Environments[0].ID != "reference" {
		t.Fatalf("environment: %#v", result.World.Environments)
	}
	if _, err := os.Stat(filepath.Join(root, "tools", "reference-process-plugin", "main.go")); err != nil {
		t.Fatalf("existing process implementation source is unavailable: %v", err)
	}
}
