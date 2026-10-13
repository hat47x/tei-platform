package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/syntax"
)

// This is an actual three-artifact build probe, not a synthetic input hash:
// the standard package and TEI-native realization use their production builder
// executables, and the Go process uses the exact source-closure resolver and
// compiler invocation. Repository revisions remain separate provenance.
func TestDocsOnlyCommitLeavesActualPluginArtifactsAndBuildPlansUnchanged(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git unavailable: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	teiRoot := filepath.Clean(filepath.Join(cwd, "..", ".."))
	repo := t.TempDir()
	outDir := t.TempDir()
	packageBuilder := filepath.Join(outDir, "package-builder")
	nativeBuilder := filepath.Join(outDir, "native-builder")
	if runtime.GOOS == "windows" {
		packageBuilder += ".exe"
		nativeBuilder += ".exe"
	}
	for _, item := range []struct{ source, target string }{
		{packageBuilderPackage, packageBuilder},
		{nativeBuilderPackage, nativeBuilder},
	} {
		if err := buildGoPackage(teiRoot, goPath, item.source, item.target, t.TempDir()); err != nil {
			t.Fatalf("build actual %s: %v", item.source, err)
		}
	}
	packageBuilderDigest, err := hashFile(packageBuilder)
	if err != nil {
		t.Fatal(err)
	}
	nativeBuilderDigest, err := hashFile(nativeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	goToolDigest, err := hashFile(goPath)
	if err != nil {
		t.Fatal(err)
	}

	runSnapshotGit(t, repo, "init")
	runSnapshotGit(t, repo, "config", "user.name", "TEI Artifact Isolation Test")
	runSnapshotGit(t, repo, "config", "user.email", "tei-artifact-isolation@example.invalid")
	runSnapshotGit(t, repo, "config", "commit.gpgsign", "false")
	write := func(rel string, data []byte) {
		t.Helper()
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{packageDescriptor, nativeDescriptor, nativeImplementation, referenceDescriptor} {
		data, err := os.ReadFile(filepath.Join(teiRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read real builder fixture %s: %v", rel, err)
		}
		write(rel, data)
	}
	write("go.mod", []byte("module example.invalid/tei-artifact-isolation\n\ngo 1.23.2\n"))
	write("cmd/demo/main.go", []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"fixture-v1\") }\n"))
	write("docs/notes.md", []byte("# initial notes\n"))
	commit := func(message string) {
		t.Helper()
		runSnapshotGit(t, repo, "add", "-A")
		runSnapshotGit(t, repo, "commit", "-m", message)
	}
	commit("baseline artifact inputs")
	type artifactState struct {
		planFingerprints map[string]string
		artifactDigests  map[string]string
	}
	definition := func(rel string) plugin.Definition {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		result, err := syntax.NewParser(string(data)).ParsePlugin()
		if err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}
		return result
	}
	digest := func(rel string) string {
		t.Helper()
		result, err := hashFile(filepath.Join(repo, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	capture := func(label string) artifactState {
		t.Helper()
		out := artifactState{planFingerprints: make(map[string]string), artifactDigests: make(map[string]string)}
		packageDefinition := definition(packageDescriptor)
		nativeDefinition := definition(nativeDescriptor)
		processDefinition := definition(referenceDescriptor)
		_, goSourceDigest, err := resolveGoPackageSourceClosure(repo, goPath, "./cmd/demo", t.TempDir())
		if err != nil {
			t.Fatalf("resolve actual Go inputs: %v", err)
		}
		makeFingerprint := func(name string, plan plugin.BuildPlan) {
			t.Helper()
			fp, err := plugin.BuildPlanFingerprint(plan)
			if err != nil {
				t.Fatalf("%s plan: %v", name, err)
			}
			out.planFingerprints[name] = fp
		}

		packagePlan := plugin.BuildPlan{
			FormatVersion: plugin.BuildPlanFormatVersion,
			Plugin: plugin.Identity{ID: packageDefinition.ID, Version: packageDefinition.Version},
			PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(packageDefinition),
			Sources: packagePlanSources(packageDescriptor, digest(packageDescriptor)),
			Builder: packageBuilderIdentity(packageBuilderDigest),
			Recipe: plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: packageDescriptor},
			OutputKind: plugin.BuildOutputPluginPackage, Target: "plugin-package", Profile: "reference",
			Parameters: []plugin.BuildParameter{{Name: "encoding", Value: "tei-plugin-package-json-v1"}},
		}
		makeFingerprint("package", packagePlan)
		nativePlan := plugin.BuildPlan{
			FormatVersion: plugin.BuildPlanFormatVersion,
			Plugin: plugin.Identity{ID: nativeDefinition.ID, Version: nativeDefinition.Version},
			PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(nativeDefinition),
			Sources: nativePlanSources(nativeDescriptor, digest(nativeDescriptor), nativeImplementation, digest(nativeImplementation)),
			Builder: nativeBuilderIdentity(nativeBuilderDigest),
			Recipe: plugin.BuildRecipe{Kind: "tei-native-realization", Entrypoint: nativeImplementation},
			OutputKind: plugin.BuildOutputRuntimeArtifact, Target: "host/reference", Profile: "reference",
		}
		makeFingerprint("native", nativePlan)
		processPlan := plugin.BuildPlan{
			FormatVersion: plugin.BuildPlanFormatVersion,
			Plugin: plugin.Identity{ID: processDefinition.ID, Version: processDefinition.Version},
			PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(processDefinition),
			Sources: processPlanSources("./cmd/demo", goSourceDigest, referenceDescriptor, digest(referenceDescriptor)),
			Builder: plugin.BuilderIdentity{
				ID: processDefinition.ID, Version: "reference-go-toolchain", ContractVersion: "tei.plugin-build/v1",
				ArtifactDigest: goToolDigest,
			},
			Recipe: plugin.BuildRecipe{Kind: "go-process", Entrypoint: "./cmd/demo"},
			OutputKind: plugin.BuildOutputRuntimeArtifact, Target: "host/reference", Profile: "reference",
		}
		makeFingerprint("process", processPlan)

		dir := filepath.Join(outDir, label)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		planJSON, err := json.Marshal(packagePlan)
		if err != nil {
			t.Fatal(err)
		}
		planFile := filepath.Join(dir, "package-plan.json")
		if err := os.WriteFile(planFile, planJSON, 0o600); err != nil {
			t.Fatal(err)
		}
		packageFile := filepath.Join(dir, "package.json")
		if err := runPluginPackageBuilder(repo, packageBuilder, planFile, packageFile); err != nil {
			t.Fatalf("actual package builder: %v", err)
		}
		data, err := os.ReadFile(packageFile)
		if err != nil {
			t.Fatal(err)
		}
		var manifest plugin.PluginPackageManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Plugin.ID != packageDefinition.ID || manifest.BuildPlanFingerprint != out.planFingerprints["package"] {
			t.Fatalf("package output does not bind its BuildPlan: %#v", manifest)
		}
		nativeFile := filepath.Join(dir, "native.json")
		if err := runNativePluginBuilder(repo, nativeBuilder, nativeFile); err != nil {
			t.Fatalf("actual native builder: %v", err)
		}
		processFile := filepath.Join(dir, "process")
		if runtime.GOOS == "windows" {
			processFile += ".exe"
		}
		if err := buildGoPackage(repo, goPath, "./cmd/demo", processFile, t.TempDir()); err != nil {
			t.Fatalf("actual Go process build: %v", err)
		}
		for name, path := range map[string]string{"package": packageFile, "native": nativeFile, "process": processFile} {
			value, err := hashFile(path)
			if err != nil {
				t.Fatalf("hash actual %s output: %v", name, err)
			}
			out.artifactDigests[name] = value
		}
		if value, err := exec.Command(processFile).Output(); err != nil || !strings.HasPrefix(string(value), "fixture-v") {
			t.Fatalf("actual process output %q: %v", value, err)
		}
		return out
	}

	snapshotBefore, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	before := capture("before")

	write("docs/notes.md", []byte("# second revision: documentation only\n"))
	commit("documentation-only edit")
	snapshotDocs, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if snapshotBefore == snapshotDocs {
		t.Fatal("documentation revision failed to change repository Evidence")
	}
	afterDocs := capture("docs-only")
	for _, name := range []string{"package", "native", "process"} {
		if before.planFingerprints[name] != afterDocs.planFingerprints[name] {
			t.Errorf("%s BuildPlan fingerprint depends on documentation-only commit", name)
		}
		if before.artifactDigests[name] != afterDocs.artifactDigests[name] {
			t.Errorf("%s actual artifact bytes depend on documentation-only commit", name)
		}
	}

	write("cmd/demo/main.go", []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"fixture-v2\") }\n"))
	commit("selected Go compiler input changed")
	afterGo := capture("compiler-input")
	if afterGo.planFingerprints["process"] == afterDocs.planFingerprints["process"] ||
		afterGo.artifactDigests["process"] == afterDocs.artifactDigests["process"] {
		t.Fatal("changing a selected Go source did not affect its BuildPlan and actual artifact")
	}
	for _, name := range []string{"package", "native"} {
		if afterGo.planFingerprints[name] != afterDocs.planFingerprints[name] ||
			afterGo.artifactDigests[name] != afterDocs.artifactDigests[name] {
			t.Errorf("unrelated %s artifact was invalidated by Go-only change", name)
		}
	}
}
