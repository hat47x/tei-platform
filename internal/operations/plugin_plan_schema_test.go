package operations

import (
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// TestPluginPlanSchemaVersionIsPinned keeps the plan's schema identifier named by
// a test. The version is part of the external contract: a caller that reads
// tei.plugin-management.plan/v0 must be able to tell when the shape changes, so
// a change to the constant has to be a deliberate act with a test behind it
// rather than a silent one.
func TestPluginPlanSchemaVersionIsPinned(t *testing.T) {
	if PluginPlanSchemaVersion != "tei.plugin-management.plan/v0" {
		t.Fatalf("PluginPlanSchemaVersion = %q, want tei.plugin-management.plan/v0; "+
			"changing it changes the external contract", PluginPlanSchemaVersion)
	}
	b := pbuiltin(t, "b")
	env := penv(t, "development", "b")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{b},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if plan.SchemaVersion != PluginPlanSchemaVersion {
		t.Fatalf("plan carries %q, want the pinned constant %q",
			plan.SchemaVersion, PluginPlanSchemaVersion)
	}
}

// TestPluginPlanDuplicateDeclarationIsNotACycle pins the duplicate-declaration
// conflict kind. The resolver refuses a plugin id and version declared twice,
// and that is a different fact from a dependency cycle or two providers of one
// contract: a caller has to be able to tell "you wrote it twice" from "these
// plugins depend on each other in a loop". Folding the two together would make
// the plan unable to say which mistake to fix.
func TestPluginPlanDuplicateDeclarationIsNotACycle(t *testing.T) {
	first := pbuiltin(t, "rsame")
	second := pbuiltin(t, "rsame")
	env := penv(t, "development", "rsame")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{first, second},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "rsame"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}

	var duplicate *PlanConflict
	for i := range plan.Conflicts {
		if plan.Conflicts[i].Kind == ConflictDuplicateDeclaration {
			duplicate = &plan.Conflicts[i]
			break
		}
	}
	if duplicate == nil {
		t.Fatalf("want a %q conflict, got %+v", ConflictDuplicateDeclaration, plan.Conflicts)
	}
	if duplicate.Subject != "rsame" {
		t.Fatalf("conflict subject = %q, want the duplicated plugin id", duplicate.Subject)
	}
	if duplicate.ResolverCode != plugin.DiagnosticDuplicatePluginVersion {
		t.Fatalf("conflict resolverCode = %q, want %q",
			duplicate.ResolverCode, plugin.DiagnosticDuplicatePluginVersion)
	}
	if len(duplicate.IDs) == 0 {
		t.Fatal("a duplicate declaration must name the duplicated entries")
	}
	if len(plan.Cycles) != 0 {
		t.Fatalf("a duplicate declaration is not a cycle, got %+v", plan.Cycles)
	}
	if plan.SelectionChanged {
		t.Fatal("a plan must never report a selection change")
	}
}
