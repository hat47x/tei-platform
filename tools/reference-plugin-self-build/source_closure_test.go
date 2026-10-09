package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	sourceDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	sourceDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	sourceDigestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

func TestReferenceBuilderVersionsAreLogicalAndArtifactExact(t *testing.T) {
	packageBuilder := packageBuilderIdentity(sourceDigestA)
	if packageBuilder.Version != packageBuilderVersion {
		t.Fatalf("package builder version: got %s want %s", packageBuilder.Version, packageBuilderVersion)
	}
	if packageBuilder.ArtifactDigest != sourceDigestA {
		t.Fatalf("package builder artifact digest: %s", packageBuilder.ArtifactDigest)
	}

	nativeBuilder := nativeBuilderIdentity(sourceDigestB)
	if nativeBuilder.Version != nativeBuilderVersion {
		t.Fatalf("native builder version: got %s want %s", nativeBuilder.Version, nativeBuilderVersion)
	}
	if nativeBuilder.ArtifactDigest != sourceDigestB {
		t.Fatalf("native builder artifact digest: %s", nativeBuilder.ArtifactDigest)
	}

	changedPackage := packageBuilderIdentity(sourceDigestC)
	if changedPackage.Version != packageBuilder.Version {
		t.Fatal("exact builder artifact change unexpectedly changed logical builder version")
	}
	if changedPackage.ArtifactDigest == packageBuilder.ArtifactDigest {
		t.Fatal("exact builder artifact change was not reflected in BuilderIdentity")
	}
}

func TestPluginBuildPlanSourcesDoNotUseRepositoryTreeAsBuildInput(t *testing.T) {
	tests := []struct {
		name    string
		sources func() []string
	}{
		{
			name: "package",
			sources: func() []string {
				var kinds []string
				for _, source := range packagePlanSources("plugin.tei", sourceDigestA) {
					kinds = append(kinds, source.Kind)
				}
				return kinds
			},
		},
		{
			name: "native",
			sources: func() []string {
				var kinds []string
				for _, source := range nativePlanSources(
					"plugin.tei",
					sourceDigestA,
					"implementation.tei",
					sourceDigestB,
				) {
					kinds = append(kinds, source.Kind)
				}
				return kinds
			},
		},
		{
			name: "process",
			sources: func() []string {
				var kinds []string
				for _, source := range processPlanSources(
					"./tools/reference-process-plugin",
					sourceDigestA,
					"plugin.tei",
					sourceDigestB,
				) {
					kinds = append(kinds, source.Kind)
				}
				return kinds
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, kind := range tt.sources() {
				if kind == "git-tree" {
					t.Fatal("Repository tree leaked back into Build Plan sources")
				}
			}
		})
	}
}

func TestPackagePlanSourcesAreOnlyAuthoritativePluginDefinition(t *testing.T) {
	sources := packagePlanSources("reference/plugin.tei", sourceDigestA)
	if len(sources) != 1 {
		t.Fatalf("package sources: %#v", sources)
	}
	got := sources[0]
	if got.Kind != "plugin-definition" ||
		got.Locator != "reference/plugin.tei" ||
		got.Digest != sourceDigestA {
		t.Fatalf("package source: %#v", got)
	}
}

func TestNativePlanSourcesKeepDefinitionAndImplementation(t *testing.T) {
	sources := nativePlanSources(
		"reference/plugin.tei",
		sourceDigestA,
		"reference/implementation.tei",
		sourceDigestB,
	)
	if len(sources) != 2 {
		t.Fatalf("native sources: %#v", sources)
	}
	if sources[0].Kind != "plugin-definition" ||
		sources[0].Digest != sourceDigestA {
		t.Fatalf("native definition source: %#v", sources[0])
	}
	if sources[1].Kind != "plugin-implementation" ||
		sources[1].Digest != sourceDigestB {
		t.Fatalf("native implementation source: %#v", sources[1])
	}
}

func TestProcessPlanSourcesBindResolvedGoClosureAndDefinition(t *testing.T) {
	sources := processPlanSources(
		"./tools/reference-process-plugin",
		sourceDigestA,
		"reference/plugin.tei",
		sourceDigestB,
	)
	if len(sources) != 2 {
		t.Fatalf("process sources: %#v", sources)
	}
	if sources[0].Kind != "go-source-closure" ||
		sources[0].Locator != "./tools/reference-process-plugin" ||
		sources[0].Digest != sourceDigestA {
		t.Fatalf("process Go source closure: %#v", sources[0])
	}
	if sources[1].Kind != "plugin-definition" ||
		sources[1].Digest != sourceDigestB {
		t.Fatalf("process Plugin definition source: %#v", sources[1])
	}
}

func TestGoSourceClosureDigestIsOrderIndependentAndInputSensitive(t *testing.T) {
	a := []goSourceMember{
		{Path: "internal/z.go", Digest: sourceDigestB},
		{Path: "go.mod", Digest: sourceDigestA},
	}
	b := []goSourceMember{a[1], a[0]}
	if goSourceClosureDigest(a) != goSourceClosureDigest(b) {
		t.Fatal("Go source closure digest depends on member declaration order")
	}

	changed := append([]goSourceMember(nil), a...)
	changed[0].Digest = sourceDigestC
	if goSourceClosureDigest(a) == goSourceClosureDigest(changed) {
		t.Fatal("Go source closure digest ignored source content change")
	}

	renamed := append([]goSourceMember(nil), a...)
	renamed[0].Path = "internal/renamed.go"
	if goSourceClosureDigest(a) == goSourceClosureDigest(renamed) {
		t.Fatal("Go source closure digest ignored source path change")
	}
}

func TestGoPackageBuildFilesIncludesSelectedCompilerAndEmbedInputs(t *testing.T) {
	pkg := goListPackage{
		GoFiles:    []string{"z.go", "a.go"},
		SFiles:     []string{"asm.s"},
		EmbedFiles: []string{"schema.json"},
	}
	got := goPackageBuildFiles(pkg)
	want := map[string]bool{
		"a.go":        true,
		"z.go":        true,
		"asm.s":       true,
		"schema.json": true,
	}
	if len(got) != len(want) {
		t.Fatalf("build files: %#v", got)
	}
	for _, file := range got {
		if !want[file] {
			t.Fatalf("unexpected build input %q in %#v", file, got)
		}
	}
}


func TestRepositoryTreeSnapshotDigestTracksGitTreeListingNotWorktreeSemantics(t *testing.T) {
	first := []byte("100644 blob aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\tREADME.md\x00" +
		"120000 blob bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\tlink\x00")
	second := append([]byte(nil), first...)
	if repositoryTreeSnapshotDigest(first) != repositoryTreeSnapshotDigest(second) {
		t.Fatal("repository tree snapshot digest is not stable for identical Git tree listings")
	}

	changed := append([]byte(nil), first...)
	changed[len(changed)-2] = '2'
	if repositoryTreeSnapshotDigest(first) == repositoryTreeSnapshotDigest(changed) {
		t.Fatal("repository tree snapshot digest ignored Git tree listing change")
	}
}


func TestHashRepositorySnapshotTreeUsesCommittedHEADNotWorktreeBytes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git is unavailable: %v", err)
	}

	repo := t.TempDir()
	runSnapshotGit(t, repo, "init")
	runSnapshotGit(t, repo, "config", "user.name", "TEI Snapshot Test")
	runSnapshotGit(t, repo, "config", "user.email", "tei-snapshot@example.invalid")
	runSnapshotGit(t, repo, "config", "commit.gpgsign", "false")

	path := filepath.Join(repo, "tracked.txt")
	if err := os.WriteFile(path, []byte("committed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runSnapshotGit(t, repo, "add", "tracked.txt")
	runSnapshotGit(t, repo, "commit", "-m", "initial")

	before, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("worktree-only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	afterWorktreeChange, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if afterWorktreeChange != before {
		t.Fatalf("uncommitted worktree bytes changed repository snapshot digest: %s != %s", afterWorktreeChange, before)
	}

	runSnapshotGit(t, repo, "add", "tracked.txt")
	runSnapshotGit(t, repo, "commit", "-m", "change")
	afterCommit, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if afterCommit == before {
		t.Fatal("committed HEAD tree change did not change repository snapshot digest")
	}
}

func runSnapshotGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	all := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", all...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
