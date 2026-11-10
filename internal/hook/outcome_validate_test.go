package hook

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// outcomeWorld declares a record pattern with one before-read hook.
func outcomeWorld(failure string) ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{{Phase: "before-read", Failure: failure}},
			},
		}},
	}}}
}

// namingRule reports the outcome it was constructed with.
type namingRule struct {
	outcome string
}

func (r namingRule) Run(context.Context, Invocation) error {
	return &NotSatisfied{Reason: "the rule decided so", Outcome: r.outcome}
}

// TestRunnerRefusesAnOutcomeOutsideTheVocabulary checks that a hook whose
// implementation invents an outcome name is refused, the same way an
// undeclared read or proposal is refused.
func TestRunnerRefusesAnOutcomeOutsideTheVocabulary(t *testing.T) {
	runner := Runner{World: outcomeWorld("reject"), Implementation: namingRule{outcome: "valueRejected"}}
	err := runner.Run(context.Background(), Invocation{Application: "order_entry", Pattern: "order_entry.order_record", Phase: "before-read"})
	if err == nil {
		t.Fatal("an outcome outside the vocabulary was accepted")
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_OUTCOME_UNDECLARED" {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "valueRejected") {
		t.Fatalf("the refusal does not name the outcome: %s", err)
	}
}

// TestRunnerAcceptsADeclaredOutcome checks that a name in the vocabulary still
// reaches the caller as the same refusal.
func TestRunnerAcceptsADeclaredOutcome(t *testing.T) {
	runner := Runner{World: outcomeWorld("reject"), Implementation: namingRule{outcome: OutcomeValueRejected}}
	err := runner.Run(context.Background(), Invocation{Application: "order_entry", Pattern: "order_entry.order_record", Phase: "before-read"})
	var notSatisfied *NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("a declared outcome was not reported as unsatisfied: %v", err)
	}
	if notSatisfied.Outcome != OutcomeValueRejected {
		t.Fatalf("outcome=%q", notSatisfied.Outcome)
	}
}

// TestRunnerAcceptsARefusalWithoutAnOutcome checks that naming no outcome is
// allowed: a rule that only gives a reason has not invented a name.
func TestRunnerAcceptsARefusalWithoutAnOutcome(t *testing.T) {
	runner := Runner{World: outcomeWorld("reject"), Implementation: namingRule{outcome: ""}}
	err := runner.Run(context.Background(), Invocation{Application: "order_entry", Pattern: "order_entry.order_record", Phase: "before-read"})
	var notSatisfied *NotSatisfied
	if !errors.As(err, &notSatisfied) {
		t.Fatalf("err=%v", err)
	}
}

// TestValidateOutcomeAcceptsTheVocabulary checks that every declared name passes.
func TestValidateOutcomeAcceptsTheVocabulary(t *testing.T) {
	for _, outcome := range NotSatisfiedOutcomes {
		if err := ValidateOutcome(outcome); err != nil {
			t.Fatalf("%s was refused: %v", outcome, err)
		}
	}
	if err := ValidateOutcome(""); err != nil {
		t.Fatalf("an empty outcome was refused: %v", err)
	}
}

// TestValidateOutcomeRefusesAnInventedName checks the negative direction.
func TestValidateOutcomeRefusesAnInventedName(t *testing.T) {
	for _, outcome := range []string{"valueRejected", "rejected", "ok", " satisfied"} {
		if err := ValidateOutcome(outcome); err == nil {
			t.Fatalf("%q was accepted", outcome)
		}
	}
}

// TestRunReachableRecordsTheRefusalItStoppedAt checks that a refused phase which
// named a declared outcome is still recorded with that outcome after the
// vocabulary check runs.
func TestRunReachableRecordsTheRefusalItStoppedAt(t *testing.T) {
	world := ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{{Phase: "before-update", Reads: []core.CanonicalID{"sales.order.quantity"}}},
			},
		}},
	}}}
	runner := Runner{World: world, Implementation: namingRule{outcome: OutcomeValueRejected}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record",
		map[core.CanonicalID]string{"sales.order.quantity": "0"}, ReferenceReach{})
	if err == nil {
		t.Fatal("the refusal did not stop the sequence")
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if report.Outcome != OutcomeValueRejected {
		t.Fatalf("outcome=%q", report.Outcome)
	}
}
