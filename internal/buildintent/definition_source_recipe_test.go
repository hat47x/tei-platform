package buildintent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/plugin"
)

// §4.40: the source that defines the subject follows from the recipe. A plugin
// build is defined by its declaration; a command build by the package it
// compiles. These tests pin the rule on both sides of that split.

func commandIntent(definitionRef string) BuildIntent {
	return BuildIntent{
		FormatVersion: 1,
		Subject: IntentSubject{
			Kind:             "plugin",
			ID:               "tei.command.probe",
			Version:          "1.0.0",
			DefinitionSource: "definition",
		},
		Sources: []IntentSource{
			{Kind: "go-package", ID: "definition", Ref: definitionRef},
		},
		Builder:    BuilderRequirement{Contract: "tei.plugin-build", CompatibleMajor: 1},
		Recipe:     plugin.BuildRecipe{Kind: RecipeKindCommand, Entrypoint: "./cmd/probe"},
		OutputKind: "runtime-artifact",
		Target:     "host/linux/amd64",
		Profile:    "release",
	}
}

func pluginIntent(definitionRef string) BuildIntent {
	return BuildIntent{
		FormatVersion: 1,
		Subject: IntentSubject{
			Kind:             "plugin",
			ID:               "tei.plugin.probe",
			Version:          "1.0.0",
			DefinitionSource: "definition",
		},
		Sources: []IntentSource{
			{Kind: "plugin-definition", ID: "definition", Ref: definitionRef},
			{Kind: "go-package", ID: "implementation", Ref: "impl"},
		},
		Builder:    BuilderRequirement{Contract: "tei.plugin-build", CompatibleMajor: 1},
		Recipe:     plugin.BuildRecipe{Kind: "go-package", Entrypoint: "."},
		OutputKind: "plugin-package",
		Target:     "host/linux/amd64",
		Profile:    "release",
	}
}

func TestValidateAcceptsTheDefinitionKindEachRecipeNeeds(t *testing.T) {
	if err := pluginIntent("plugin.tei").Validate(); err != nil {
		t.Fatalf("a plugin build with a plugin-definition was refused: %v", err)
	}
	if err := commandIntent("cmd/probe").Validate(); err != nil {
		t.Fatalf("a command build with a go-package was refused: %v", err)
	}
}

// The rule is strict in both directions: the old single rule is not simply
// relaxed, it is replaced by one rule per recipe.
func TestValidateRefusesTheDefinitionKindTheRecipeDoesNotUse(t *testing.T) {
	commandWithDeclaration := commandIntent("cmd/probe")
	commandWithDeclaration.Sources[0].Kind = "plugin-definition"
	if err := commandWithDeclaration.Validate(); err == nil {
		t.Fatal("a command build accepted a plugin-definition as its definition source")
	} else if !strings.Contains(err.Error(), "go-package") {
		t.Fatalf("the refusal does not name the kind the recipe needs: %v", err)
	}

	pluginWithPackage := pluginIntent("plugin.tei")
	pluginWithPackage.Sources[0].Kind = "go-package"
	if err := pluginWithPackage.Validate(); err == nil {
		t.Fatal("a plugin build accepted a go-package as its definition source")
	} else if !strings.Contains(err.Error(), "plugin-definition") {
		t.Fatalf("the refusal does not name the kind the recipe needs: %v", err)
	}
}

// The definition source must still be declared: neither recipe may name a source
// that is not there.
func TestValidateStillRequiresADeclaredDefinitionSource(t *testing.T) {
	intent := commandIntent("cmd/probe")
	intent.Subject.DefinitionSource = "missing"
	if err := intent.Validate(); err == nil {
		t.Fatal("an undeclared definitionSource was accepted")
	}
}

// The resolver derives the plan's definition identity from the declaration for a
// plugin build, and from the package source for a command build. What matters is
// that the value exists, is stable for the same input, and changes when the input
// changes.
func TestResolveDerivesTheDefinitionIdentityPerRecipe(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "probe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cmd", "probe", "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog := BuildCatalog{
		FormatVersion: 1,
		Providers: []BuildProvider{{
			Plugin:         "tei.builder.go",
			Version:        "1.2.0",
			Implementation: "go.build-from-plan",
			Contract:       "tei.plugin-build",
			ContractMajor:  1,
			AuthorityScope: "build-tool",
			ArtifactDigest: "sha256:" + strings.Repeat("a", 64),
			RecipeKinds:    []string{RecipeKindCommand},
			Targets:        []string{"host/linux/amd64"},
			Profiles:       []string{"release"},
		}},
		Targets: []BuildTargetEntry{{Target: "host/linux/amd64", Profiles: []string{"release"}}},
	}
	ctx := NewBuildContext(root, catalog)

	intent := commandIntent("cmd/probe")
	receipt, err := Resolve(intent, ctx)
	if err != nil {
		t.Fatalf("resolve a command build: %v", err)
	}
	if receipt.Plan.PluginDefinitionFingerprint == "" {
		t.Fatal("the plan carries no definition identity")
	}
	again, err := Resolve(intent, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if again.Plan.PluginDefinitionFingerprint != receipt.Plan.PluginDefinitionFingerprint {
		t.Fatal("the same input produced a different definition identity")
	}

	// Change the defining input: the identity must follow it.
	if err := os.WriteFile(filepath.Join(root, "cmd", "probe", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := Resolve(intent, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Plan.PluginDefinitionFingerprint == receipt.Plan.PluginDefinitionFingerprint {
		t.Fatal("changed source bytes left the definition identity unchanged")
	}
}
