package main

import "testing"

const (
	sourceDigestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	sourceDigestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	sourceDigestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

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
		GoFiles:      []string{"z.go", "a.go"},
		SFiles:       []string{"asm.s"},
		EmbedFiles:   []string{"schema.json"},
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
