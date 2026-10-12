package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

// These adapter descriptions exercise the shared BuildPlan contract. They do
// not claim that either source resolver or executable builder is implemented.
func TestBuildPlanKeepsResolvedSourceClosureLanguageNeutral(t *testing.T) {
	tests := []struct {
		name          string
		closureKind   string
		entrypoint    string
		recipeKind    string
		builderID     string
		target        string
	}{
		{
			name:        "tei-native",
			closureKind: "tei-source-closure",
			entrypoint:  "reference/plugin-implementations/implementation.tei",
			recipeKind:  "tei-native-realization",
			builderID:   "tei.builder.native.reference",
			target:      "host/tei-native",
		},
		{
			name:        "external-adapter",
			closureKind: "adapter-resolved-source-closure",
			entrypoint:  "reference/adapter/program.src",
			recipeKind:  "adapter-compiled-component",
			builderID:   "tei.builder.adapter.reference",
			target:      "wasm/component",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildPlanFixture()
			plan.Sources = []plugin.BuildSource{
				{
					Kind:    tt.closureKind,
					ID:      "resolved-inputs",
					Locator: tt.entrypoint,
					Digest:  buildDigestA,
				},
				{
					Kind:    "plugin-definition",
					ID:      "definition",
					Locator: "reference/plugin.tei",
					Digest:  buildDigestB,
				},
			}
			plan.Builder = plugin.BuilderIdentity{
				ID:              core.MustCanonicalID(tt.builderID),
				Version:         "1.0.0",
				ContractVersion: "tei.plugin-build/v1",
				ArtifactDigest:  buildDigestB,
			}
			plan.Recipe = plugin.BuildRecipe{
				Kind:       tt.recipeKind,
				Entrypoint: tt.entrypoint,
			}
			plan.Target = tt.target
			plan.Dependencies = nil
			plan.Parameters = nil

			base, err := plugin.BuildPlanFingerprint(plan)
			if err != nil {
				t.Fatalf("language-neutral BuildPlan rejected: %v", err)
			}
			locked, err := plugin.LockBuildOutput(plan, buildDigestA)
			if err != nil {
				t.Fatal(err)
			}
			if locked.PlanFingerprint != base || locked.ArtifactDigest != buildDigestA {
				t.Fatalf("lock lost the generic build plan or artifact binding: %+v", locked)
			}

			// The adapter resolves the inputs. The shared model only sorts
			// and binds its fully resolved, digest-pinned declarations.
			reordered := plan
			reordered.Sources = []plugin.BuildSource{plan.Sources[1], plan.Sources[0]}
			same, err := plugin.BuildPlanFingerprint(reordered)
			if err != nil || same != base {
				t.Fatalf("source declaration order changed the plan: %s, %v", same, err)
			}

			for _, tc := range []struct {
				name string
				edit func(*plugin.BuildPlan)
			}{
				{"resolved input content", func(p *plugin.BuildPlan) { p.Sources[0].Digest = buildDigestB }},
				{"resolved input location", func(p *plugin.BuildPlan) { p.Sources[0].Locator += ".other" }},
				{"non-Go builder implementation", func(p *plugin.BuildPlan) { p.Builder.ArtifactDigest = buildDigestA }},
				{"builder contract", func(p *plugin.BuildPlan) { p.Builder.ContractVersion = "tei.plugin-build/v2" }},
				{"target", func(p *plugin.BuildPlan) { p.Target += "/other" }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					changed := plan
					changed.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
					tc.edit(&changed)
					got, err := plugin.BuildPlanFingerprint(changed)
					if err != nil {
						t.Fatal(err)
					}
					if got == base {
						t.Fatal("artifact-affecting input did not invalidate BuildPlan")
					}
				})
			}

			for _, tc := range []struct {
				name string
				edit func(*plugin.BuildPlan)
			}{
				{"unresolved source", func(p *plugin.BuildPlan) { p.Sources[0].Digest = "" }},
				{"ambiguous input", func(p *plugin.BuildPlan) { p.Sources[1].ID = p.Sources[0].ID }},
				{"builder without artifact", func(p *plugin.BuildPlan) { p.Builder.ArtifactDigest = "" }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					invalid := plan
					invalid.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
					tc.edit(&invalid)
					if _, err := plugin.BuildPlanFingerprint(invalid); err == nil {
						t.Fatal("unresolved input was accepted into BuildPlan")
					}
				})
			}
		})
	}
}
