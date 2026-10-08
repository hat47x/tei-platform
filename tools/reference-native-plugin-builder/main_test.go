package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/plugin/implementation"
)

func TestReferenceNativePluginBuilderMaterializesDeterministicArtifact(t *testing.T) {
	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.tei")
	implementationPath := filepath.Join(root, "implementation.tei")
	firstPath := filepath.Join(root, "first.json")
	secondPath := filepath.Join(root, "second.json")

	writeBuilderFixture(t, pluginPath, `plugin tei.native.builder_test v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.builder_test {
        action application-start
    }
}
`)
	writeBuilderFixture(t, implementationPath, `plugin-implementation tei.native.builder_test v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.builder_test
    operation execute json {
        {
            "outcome": "accepted",
            "attemptId": "builder-test"
        }
    }
}
`)

	if err := run(implementationPath, pluginPath, firstPath); err != nil {
		t.Fatal(err)
	}
	if err := run(implementationPath, pluginPath, secondPath); err != nil {
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
		t.Fatalf("builder output differs:\n%s\n%s", first, second)
	}
	source, digest, err := implementation.ParseRealization(first)
	if err != nil {
		t.Fatal(err)
	}
	if source.Plugin.ID != "tei.native.builder_test" || digest == "" {
		t.Fatalf("realization: source=%#v digest=%q", source, digest)
	}
}

func TestReferenceNativePluginBuilderRejectsPredeclaredArtifact(t *testing.T) {
	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.tei")
	implementationPath := filepath.Join(root, "implementation.tei")
	outputPath := filepath.Join(root, "out.json")

	writeBuilderFixture(t, pluginPath, `plugin tei.native.builder_test v1.0.0 {
    runtime builtin
    artifact {
        digest "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    }
    provides deployment-executor runtime.builder_test {
        action application-start
    }
}
`)
	writeBuilderFixture(t, implementationPath, `plugin-implementation tei.native.builder_test v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.builder_test
    operation execute json { {"attemptId":"builder-test","outcome":"accepted"} }
}
`)

	if err := run(implementationPath, pluginPath, outputPath); err == nil {
		t.Fatal("expected predeclared artifact to fail")
	}
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("builder created output after failed validation: %v", err)
	}
}

func writeBuilderFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
