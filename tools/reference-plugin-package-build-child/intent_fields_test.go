package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/syntax"
)

// These tests pin the field-level comparison added on top of identity and build
// coordinates (SB-C0 §4.15's next step).
//
// The comparison covers the declaration fields the plan also carries. Sources
// and dependencies are compared by identity only: the declaration names a
// reference, the plan carries a derived locator and digest, so comparing those
// would call every correct build a mismatch. Parameters are excluded because the
// resolver may normalise them, and a mismatch there would not distinguish a wrong
// declaration from a different normalisation.

// declarationFor builds a declaration matching the fixture plan, with a hook to
// change one field so a single difference can be exercised.
func declarationFor(t *testing.T, mutate func(*buildintent.BuildIntent)) buildintent.BuildIntent {
	t.Helper()
	definition, err := syntax.NewParser(testPluginSource).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	intent := buildintent.BuildIntent{
		FormatVersion: buildintent.IntentFormatVersion,
		Subject: buildintent.IntentSubject{
			Kind:             "plugin",
			ID:               definition.ID,
			Version:          definition.Version,
			DefinitionSource: "definition",
		},
		Sources: []buildintent.IntentSource{
			{Kind: "plugin-definition", ID: "definition", Ref: "plugin.tei"},
		},
		Builder:    buildintent.BuilderRequirement{Contract: "tei.plugin-build", CompatibleMajor: 1},
		Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "plugin.tei"},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
	}
	if mutate != nil {
		mutate(&intent)
	}
	if err := intent.Validate(); err != nil {
		t.Fatalf("the fixture declaration does not validate: %v", err)
	}
	return intent
}

func fixturePlan(t *testing.T) plugin.BuildPlan {
	t.Helper()
	definition, err := syntax.NewParser(testPluginSource).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-definition",
			ID:      "definition",
			Locator: "plugin.tei",
			Digest:  "sha256:" + strings.Repeat("a", 64),
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.reference.seed"),
			Version:         "test",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("b", 64),
		},
		Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "plugin.tei"},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
	}
}

// A declaration that agrees on every compared field passes.
func TestCompareIntentFieldsAcceptsAnAgreeingDeclaration(t *testing.T) {
	if err := compareIntentFields(declarationFor(t, nil), fixturePlan(t)); err != nil {
		t.Fatalf("an agreeing declaration was refused: %v", err)
	}
}

// Each compared field, changed on its own, is refused and named.
func TestCompareIntentFieldsNamesEachMismatch(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*buildintent.BuildIntent)
		expect string
	}{
		{
			name:   "recipe kind",
			mutate: func(i *buildintent.BuildIntent) { i.Recipe.Kind = "go-package" },
			expect: "recipe kind",
		},
		{
			name:   "recipe entrypoint",
			mutate: func(i *buildintent.BuildIntent) { i.Recipe.Entrypoint = "impl" },
			expect: "recipe entrypoint",
		},
		{
			// The definition source must stay a plugin-definition for the
			// declaration to validate, so the second source carries the changed
			// kind instead.
			name: "source kind",
			mutate: func(i *buildintent.BuildIntent) {
				i.Sources = append(i.Sources, buildintent.IntentSource{Kind: "go-package", ID: "extra", Ref: "impl"})
				i.Sources[0].ID = "definition"
			},
			expect: "source",
		},
		{
			name: "source count",
			mutate: func(i *buildintent.BuildIntent) {
				i.Sources = append(i.Sources, buildintent.IntentSource{Kind: "go-package", ID: "extra", Ref: "impl"})
			},
			expect: "sources",
		},
		{
			name: "dependency count",
			mutate: func(i *buildintent.BuildIntent) {
				i.Dependencies = append(i.Dependencies, buildintent.DependencyRequirement{Kind: "build-artifact", ID: "tei.support.codec"})
			},
			expect: "dependencies",
		},
	}
	plan := fixturePlan(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := compareIntentFields(declarationFor(t, c.mutate), plan)
			if err == nil {
				t.Fatalf("%s mismatch was accepted", c.name)
			}
			if !strings.Contains(err.Error(), c.expect) {
				t.Fatalf("%s mismatch was refused without naming it: %v", c.name, err)
			}
		})
	}
}

// A source that is in the plan but not in the declaration is refused: the counts
// differ even when every named source agrees.
func TestCompareIntentFieldsRefusesAnUnnamedPlanSource(t *testing.T) {
	plan := fixturePlan(t)
	plan.Sources = append(plan.Sources, plugin.BuildSource{
		Kind:    "go-package",
		ID:      "unnamed",
		Locator: "impl",
		Digest:  "sha256:" + strings.Repeat("c", 64),
	})
	if err := compareIntentFields(declarationFor(t, nil), plan); err == nil {
		t.Fatal("a plan source the declaration never named was accepted")
	}
}

// A source whose locator and digest were derived is not a mismatch: those are
// what the resolver produces, not what the declaration says.
func TestCompareIntentFieldsIgnoresDerivedSourceValues(t *testing.T) {
	plan := fixturePlan(t)
	plan.Sources[0].Locator = "a/different/locator"
	plan.Sources[0].Digest = "sha256:" + strings.Repeat("d", 64)
	if err := compareIntentFields(declarationFor(t, nil), plan); err != nil {
		t.Fatalf("a derived locator and digest were treated as a mismatch: %v", err)
	}
}

// Dependencies are compared by identity; their derived version and digest are not.
func TestCompareIntentFieldsIgnoresDerivedDependencyValues(t *testing.T) {
	intent := declarationFor(t, func(i *buildintent.BuildIntent) {
		i.Dependencies = []buildintent.DependencyRequirement{{Kind: "build-artifact", ID: "tei.support.codec"}}
	})
	plan := fixturePlan(t)
	plan.Dependencies = []plugin.BuildDependency{{
		Kind:           "build-artifact",
		ID:             "tei.support.codec",
		Version:        "1.4.0",
		ArtifactDigest: "sha256:" + strings.Repeat("e", 64),
	}}
	if err := compareIntentFields(intent, plan); err != nil {
		t.Fatalf("a derived dependency version and digest were treated as a mismatch: %v", err)
	}
}

// The child reports the binding as performed only when the field comparison also
// passed, so the report cannot say "bound" about a declaration it refused.
func TestBuildChildRefusesADeclarationWithADifferentRecipe(t *testing.T) {
	pluginPath, planPath := writeFixture(t)
	root := t.TempDir()
	artifactPath := filepath.Join(root, "package.json")
	intentPath := writeDeclaration(t, fixtureCoordinates(t))

	// Rewrite the declaration with a recipe the plan does not name.
	body, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := strings.Replace(string(body), `"kind": "tei-plugin-package"`, `"kind": "go-package"`, 1)
	if rewritten == string(body) {
		t.Fatal("the fixture recipe was not found")
	}
	if err := os.WriteFile(intentPath, []byte(rewritten), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"-plan", planPath,
		"-plugin", pluginPath,
		"-artifact-out", artifactPath,
		"-intent", intentPath,
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("a declaration with a different recipe was accepted: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "recipe kind") {
		t.Fatalf("the refusal did not name the recipe kind: %s", stderr.String())
	}
	// No report, so no "bound" claim about a declaration the child refused.
	if stdout.Len() != 0 {
		t.Fatalf("a refused declaration still produced a report: %s", stdout.String())
	}
	var report childReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err == nil && report.IntentBoundToPlan {
		t.Fatal("a refused declaration was reported as bound")
	}
}
