package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseRunPlan(t *testing.T) {
	source := `namespace sales

data Period {
    attribute month type std.code
}

capability sales.close {
    input period sales.period
    outcome closed sales.period
    effect update sales.period
}

capability sales.statement {
    input period sales.period
    outcome rendered sales.period
    effect create sales.period
}

run-plan monthly_close {
    operation sales.close
    key sales.period.month
    concurrency singleton
    resume checkpoint
    output sales.period
    autonomy unattended
}
`
	namespace, err := syntax.NewParser(source).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(namespace.RunPlans) != 1 {
		t.Fatalf("expected one run plan: %+v", namespace.RunPlans)
	}
	plan := namespace.RunPlans[0]
	if plan.ID != "sales.monthly_close" {
		t.Fatalf("unexpected id: %s", plan.ID)
	}
	if len(plan.Operations) != 1 || plan.Operations[0] != "sales.close" {
		t.Fatalf("unexpected operations: %+v", plan.Operations)
	}
	if len(plan.Key) != 1 || plan.Key[0] != "sales.period.month" {
		t.Fatalf("unexpected key: %+v", plan.Key)
	}
	if plan.Concurrency != "singleton" || plan.Resume != "checkpoint" || plan.Autonomy != "unattended" {
		t.Fatalf("unexpected strategies: %+v", plan)
	}
	if len(plan.Outputs) != 1 || plan.Outputs[0] != "sales.period" {
		t.Fatalf("unexpected outputs: %+v", plan.Outputs)
	}
}
