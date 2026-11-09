package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func runPlanWorld(plan semantic.RunPlan) ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{ID: "std", Types: []semantic.Type{{ID: "std.code", Symbol: "code", Kind: "primitive"}}},
		{
			ID: "sales",
			Data: []semantic.Data{{
				ID:         "sales.period",
				Attributes: []semantic.Attribute{{ID: "sales.period.month", TypeRef: "std.code"}},
			}},
			Capabilities: []semantic.Capability{{ID: "sales.close"}, {ID: "sales.statement"}},
			RunPlans:     []semantic.RunPlan{plan},
		}}}
}

func TestRunPlanAcceptsADeclaredPlan(t *testing.T) {
	world := runPlanWorld(semantic.RunPlan{
		ID:          "sales.monthly_close",
		Operations:  []core.CanonicalID{"sales.close"},
		Key:         []core.CanonicalID{"sales.period.month"},
		Concurrency: "singleton",
		Resume:      "checkpoint",
		Outputs:     []core.CanonicalID{"sales.period"},
		Autonomy:    "unattended",
	})
	if diagnostics := validation.Validate(world); len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics: %+v", diagnostics)
	}
}

func TestRunPlanRejectsAnEmptyPlan(t *testing.T) {
	world := runPlanWorld(semantic.RunPlan{ID: "sales.empty"})
	codes := codesOf(validation.Validate(world))
	for _, want := range []string{"RUN_PLAN_NO_OPERATIONS", "RUN_PLAN_NO_KEY"} {
		if !codes[want] {
			t.Fatalf("expected %s: %+v", want, codes)
		}
	}
}

func TestRunPlanRejectsUnknownParts(t *testing.T) {
	world := runPlanWorld(semantic.RunPlan{
		ID:          "sales.bad",
		Operations:  []core.CanonicalID{"sales.missing"},
		Key:         []core.CanonicalID{"sales.period.unknown"},
		Concurrency: "sometimes",
		Resume:      "sometimes",
		Autonomy:    "sometimes",
	})
	codes := codesOf(validation.Validate(world))
	for _, want := range []string{"RUN_PLAN_UNKNOWN_OPERATION", "RUN_PLAN_UNKNOWN_KEY", "RUN_PLAN_UNKNOWN_CONCURRENCY", "RUN_PLAN_UNKNOWN_RESUME", "RUN_PLAN_UNKNOWN_AUTONOMY"} {
		if !codes[want] {
			t.Fatalf("expected %s: %+v", want, codes)
		}
	}
}

func TestRunPlanRequiresAnOutputForCheckpoints(t *testing.T) {
	world := runPlanWorld(semantic.RunPlan{
		ID:         "sales.checkpoint",
		Operations: []core.CanonicalID{"sales.close"},
		Key:        []core.CanonicalID{"sales.period.month"},
		Resume:     "checkpoint",
	})
	if !codesOf(validation.Validate(world))["RUN_PLAN_CHECKPOINT_WITHOUT_OUTPUT"] {
		t.Fatalf("a checkpoint needs an output: %+v", validation.Validate(world))
	}
}
func TestRunPlanRejectsAnUnknownOutput(t *testing.T) {
	world := runPlanWorld(semantic.RunPlan{
		ID:         "sales.outputs",
		Operations: []core.CanonicalID{"sales.close"},
		Key:        []core.CanonicalID{"sales.period.month"},
		Outputs:    []core.CanonicalID{"sales.missing_statement"},
	})
	if !codesOf(validation.Validate(world))["RUN_PLAN_UNKNOWN_OUTPUT"] {
		t.Fatalf("an output must name existing data: %+v", validation.Validate(world))
	}
}
