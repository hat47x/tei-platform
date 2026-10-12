package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

// A committed documentation change must change snapshot provenance without
// changing artifact-affecting inputs in any of the three reference BuildPlans.
func TestReferenceBuildPlansDoNotInvalidateOnCommittedDocsOnlyChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git is unavailable: %v", err)
	}
	root := t.TempDir()
	runSnapshotGit(t, root, "init")
	runSnapshotGit(t, root, "config", "user.name", "TEI Source Closure Test")
	runSnapshotGit(t, root, "config", "user.email", "tei-closure@example.invalid")
	runSnapshotGit(t, root, "config", "commit.gpgsign", "false")

	write := func(path, contents string) {
		t.Helper()
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(message string) {
		t.Helper()
		runSnapshotGit(t, root, "add", "-A")
		runSnapshotGit(t, root, "commit", "-m", message)
	}
	write("go.mod", "module example.invalid/reference\n\ngo 1.23.2\n")
	write("pkg/program.go", "package pkg\nfunc Value() string { return \"before\" }\n")
	write("reference/plugin.tei", "definition-before\n")
	write("reference/implementation.tei", "implementation-before\n")
	write("docs/notes.md", "# original\n")
	commit("initial")

	digestFile := func(path string) string {
		t.Helper()
		bytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return sha256Digest(bytes)
	}
	fingerprints := func() map[string]string {
		t.Helper()
		// These two paths represent the resolved compiler-input closure of
		// this fixture; other tests check the actual go list resolver.
		goDigest, err := hashSourceFiles(root, []string{"go.mod", "pkg/program.go"})
		if err != nil {
			t.Fatal(err)
		}
		definitionDigest := digestFile("reference/plugin.tei")
		implementationDigest := digestFile("reference/implementation.tei")
		base := plugin.BuildPlan{
			FormatVersion:               plugin.BuildPlanFormatVersion,
			Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.reference.closure_test"), Version: "1.0.0"},
			PluginDefinitionFingerprint: strings.Repeat("a", 64),
			Recipe:                      plugin.BuildRecipe{Kind: "reference-test", Entrypoint: "reference/plugin.tei"},
			OutputKind:                  plugin.BuildOutputRuntimeArtifact,
			Target:                      "host/test",
		}
		packagePlan := base
		packagePlan.Sources = packagePlanSources("reference/plugin.tei", definitionDigest)
		packagePlan.Builder = packageBuilderIdentity(sourceDigestA)
		packagePlan.OutputKind = plugin.BuildOutputPluginPackage

		nativePlan := base
		nativePlan.Sources = nativePlanSources(
			"reference/plugin.tei", definitionDigest,
			"reference/implementation.tei", implementationDigest,
		)
		nativePlan.Builder = nativeBuilderIdentity(sourceDigestB)

		processPlan := base
		processPlan.Sources = processPlanSources(
			"./pkg", goDigest, "reference/plugin.tei", definitionDigest,
		)
		processPlan.Builder = plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.go.reference"),
			Version:         "go1.23.2",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  sourceDigestC,
		}
		out := make(map[string]string)
		for name, plan := range map[string]plugin.BuildPlan{
			"package": packagePlan,
			"native":  nativePlan,
			"process": processPlan,
		} {
			fp, err := plugin.BuildPlanFingerprint(plan)
			if err != nil {
				t.Fatalf("%s BuildPlan fingerprint: %v", name, err)
			}
			out[name] = fp
		}
		return out
	}
	assertChanges := func(before, after map[string]string, changed ...string) {
		t.Helper()
		changeSet := make(map[string]bool)
		for _, name := range changed {
			changeSet[name] = true
		}
		for name, value := range before {
			if got := after[name]; (got != value) != changeSet[name] {
				t.Errorf("%s fingerprint changed=%t, expected=%t", name, got != value, changeSet[name])
			}
		}
	}

	snapshotBefore, err := hashRepositorySnapshotTree(root)
	if err != nil {
		t.Fatal(err)
	}
	before := fingerprints()

	write("docs/notes.md", "# changed documentation only\n")
	commit("docs only")
	snapshotAfterDocs, err := hashRepositorySnapshotTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if snapshotAfterDocs == snapshotBefore {
		t.Fatal("committed documentation did not change repository snapshot provenance")
	}
	afterDocs := fingerprints()
	assertChanges(before, afterDocs)

	write("pkg/program.go", "package pkg\nfunc Value() string { return \"after\" }\n")
	commit("compiler input changed")
	afterGo := fingerprints()
	assertChanges(afterDocs, afterGo, "process")

	write("reference/implementation.tei", "implementation-after\n")
	commit("native implementation changed")
	afterImplementation := fingerprints()
	assertChanges(afterGo, afterImplementation, "native")

	write("reference/plugin.tei", "definition-after\n")
	commit("plugin definition changed")
	afterDefinition := fingerprints()
	assertChanges(afterImplementation, afterDefinition, "package", "native", "process")
}
