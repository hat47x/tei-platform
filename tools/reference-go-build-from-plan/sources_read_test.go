package main

import (
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// §4.46: the plan must name only the sources the build reads. This is the mirror
// of §4.41 — there, the build read something the plan did not name; here, the plan
// names something the build never reads. Both make the plan and the build disagree.

func commandPlanWith(kinds ...string) plugin.BuildPlan {
	sources := make([]plugin.BuildSource, 0, len(kinds))
	for i, kind := range kinds {
		sources = append(sources, plugin.BuildSource{
			Kind:    kind,
			ID:      "s" + string(rune('a'+i)),
			Locator: "cmd/probe",
			Digest:  "sha256:" + strings.Repeat("a", 64),
		})
	}
	return plugin.BuildPlan{
		Recipe:  plugin.BuildRecipe{Kind: "go-command", Entrypoint: "./cmd/probe"},
		Sources: sources,
	}
}

func TestCheckSourcesAreReadAcceptsTheSourcesACommandBuildUses(t *testing.T) {
	plan := commandPlanWith("go-package", "go-module")
	if err := checkSourcesAreRead(plan); err != nil {
		t.Fatalf("a plan naming only consumed sources was refused: %v", err)
	}
}

func TestCheckSourcesAreReadRefusesASourceTheCommandBuildNeverReads(t *testing.T) {
	plan := commandPlanWith("go-package", "plugin-definition")
	err := checkSourcesAreRead(plan)
	if err == nil {
		t.Fatal("a plan naming a declaration the build never reads was accepted")
	}
	if !strings.Contains(err.Error(), "plugin-definition") {
		t.Fatalf("the refusal does not name the unused source: %v", err)
	}
	if !strings.Contains(err.Error(), "does not read") {
		t.Fatalf("the refusal does not say why it is wrong: %v", err)
	}
}

// Other recipes keep the behaviour they had: the rule is defined for the command
// recipe, and this test says so rather than leaving it to inference.
func TestCheckSourcesAreReadLeavesOtherRecipesAlone(t *testing.T) {
	plan := commandPlanWith("plugin-definition", "go-package")
	plan.Recipe.Kind = "go-package"
	if err := checkSourcesAreRead(plan); err != nil {
		t.Fatalf("a plugin plan was refused by the command rule: %v", err)
	}
}

// The tool refuses the whole attempt, not just the record.
func TestRunRefusesAPlanWithAnUnusedSource(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	locator := "tools/reference-go-build-from-plan/testdata/probe"
	plan := planFor(t, root, locator, "./tools/reference-go-build-from-plan/testdata/probe", "runtime-artifact")

	// Add a declaration the command build never reads.
	definitionLocator := "reference/plugin-self-build/workspace/plugins/tei-native-runtime-reference/1.0.0/plugin.tei"
	resolved, resolveErr := (buildintent.DirSourceResolver{Root: root}).Resolve(buildintent.IntentSource{
		Kind: "plugin-definition", ID: "definition", Ref: definitionLocator,
	})
	if resolveErr != nil {
		t.Fatalf("resolve the declaration: %v", resolveErr)
	}
	plan.Sources = append(plan.Sources, plugin.BuildSource{
		Kind:    "plugin-definition",
		ID:      "definition",
		Locator: definitionLocator,
		Digest:  resolved.Source.Digest,
	})
	planPath := writePlan(t, dir, plan)

	_, runErr := runTool(t,
		"-plan", planPath,
		"-repo", root,
		"-artifact-out", filepath.Join(dir, "artifact"),
		"-state", filepath.Join(dir, "state"),
	)
	if runErr == nil {
		t.Fatal("a plan with an unused source was accepted")
	}
	if !strings.Contains(runErr.Error(), "does not read") {
		t.Fatalf("the refusal does not say why: %v", runErr)
	}
}
