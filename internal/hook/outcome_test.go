package hook

import (
	"context"
	"testing"

	"tei/internal/command"
	"tei/internal/core"
)

// TestNotSatisfiedOutcomesMapToCommandNames checks that every outcome a hook may
// report maps to the name command.FailureOutcome reports for the same refusal,
// so the two vocabularies cannot drift apart.
func TestNotSatisfiedOutcomesMapToCommandNames(t *testing.T) {
	for _, outcome := range NotSatisfiedOutcomes {
		name, ok := CommandOutcomeFor(outcome)
		if outcome == OutcomeUnmet {
			if ok {
				t.Fatalf("%s was mapped to a command outcome %q", outcome, name)
			}
			continue
		}
		if !ok {
			t.Fatalf("%s has no command outcome", outcome)
		}
		if name != outcome {
			t.Fatalf("%s maps to %q, but command.FailureOutcome spells it %q", outcome, name, outcome)
		}
	}
}

// TestCommandOutcomeForNamesTheReferenceRefusals checks the mapping for each
// refusal the reference store decides.
func TestCommandOutcomeForNamesTheReferenceRefusals(t *testing.T) {
	cases := map[string]string{
		"not-found":      "not-found",
		"already-exists": "already-exists",
		"value-rejected": "value-rejected",
		"rule-rejected":  "rule-rejected",
		"undeclared":     "undeclared",
	}
	for hookOutcome, want := range cases {
		got, ok := CommandOutcomeFor(hookOutcome)
		if !ok || got != want {
			t.Fatalf("%s -> (%q, %t), want %q", hookOutcome, got, ok, want)
		}
	}
}

// TestCommandOutcomeForRefusesAnUnknownName checks that a name outside the
// declared vocabulary is not mapped: an invented outcome must not become a
// command refusal.
func TestCommandOutcomeForRefusesAnUnknownName(t *testing.T) {
	if name, ok := CommandOutcomeFor("valueRejected"); ok {
		t.Fatalf("a name outside the vocabulary mapped to %q", name)
	}
	if name, ok := CommandOutcomeFor(""); ok {
		t.Fatalf("an empty name mapped to %q", name)
	}
}

// TestReferenceRuleUsesTheDeclaredOutcomes checks that the reference rule reports
// names from the declared vocabulary, not ad-hoc strings.
func TestReferenceRuleUsesTheDeclaredOutcomes(t *testing.T) {
	rule := ReferenceRule{Required: []core.CanonicalID{"sales.order.status"}}
	err := rule.Run(context.Background(), Invocation{})
	outcome, ok := OutcomeOf(err)
	if !ok || outcome != OutcomeUnmet {
		t.Fatalf("outcome=(%q, %t), want %q", outcome, ok, OutcomeUnmet)
	}
	if _, mapped := CommandOutcomeFor(outcome); mapped {
		t.Fatalf("%s was mapped to a command outcome", outcome)
	}

	rule = ReferenceRule{Positive: []core.CanonicalID{"sales.order.quantity"}}
	err = rule.Run(context.Background(), Invocation{Reads: map[core.CanonicalID]string{"sales.order.quantity": "0"}})
	outcome, ok = OutcomeOf(err)
	if !ok || outcome != OutcomeValueRejected {
		t.Fatalf("outcome=(%q, %t), want %q", outcome, ok, OutcomeValueRejected)
	}
	if mapped, ok := CommandOutcomeFor(outcome); !ok || mapped != "value-rejected" {
		t.Fatalf("%s mapped to (%q, %t)", outcome, mapped, ok)
	}
}

// TestOutcomeOfIgnoresAnExecutionFailure checks that an execution failure is not
// reported as a business outcome.
func TestOutcomeOfIgnoresAnExecutionFailure(t *testing.T) {
	err := &Error{Code: "HOOK_REJECTED", Message: "the rule failed"}
	if outcome, ok := OutcomeOf(err); ok {
		t.Fatalf("an execution failure reported outcome %q", outcome)
	}
	if outcome, ok := OutcomeOf(nil); ok {
		t.Fatalf("nil reported outcome %q", outcome)
	}
}

// TestOutcomeOfReportsNoOutcomeWhenTheHookNamedNone checks that an unsatisfied
// hook without an outcome name is not given one here.
func TestOutcomeOfReportsNoOutcomeWhenTheHookNamedNone(t *testing.T) {
	err := &NotSatisfied{Reason: "the record is not ready"}
	if outcome, ok := OutcomeOf(err); ok {
		t.Fatalf("an unnamed outcome was reported as %q", outcome)
	}
}

// TestNotSatisfiedOutcomesHasNoDuplicates keeps the vocabulary a set.
func TestNotSatisfiedOutcomesHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, outcome := range NotSatisfiedOutcomes {
		if seen[outcome] {
			t.Fatalf("duplicate outcome %q", outcome)
		}
		seen[outcome] = true
	}
	if !seen[OutcomeUnmet] || !seen[OutcomeValueRejected] {
		t.Fatal("the vocabulary does not hold the names the reference rule reports")
	}
}

// TestEveryVocabularyNameIsReachableFromFailureOutcome checks the other
// direction: a name in the vocabulary that is not OutcomeUnmet must be one that
// command.FailureOutcome can return.
func TestEveryVocabularyNameIsReachableFromFailureOutcome(t *testing.T) {
	reachable := map[string]bool{}
	for _, code := range []command.ReferenceFailureCode{
		command.ReferenceRecordMissing,
		command.ReferenceRecordExists,
		command.ReferenceValueRejected,
		command.ReferenceRuleRejected,
		command.ReferenceUndeclared,
	} {
		if name, ok := command.FailureOutcome(&command.ReferenceFailure{Code: code}); ok {
			reachable[name] = true
		}
	}
	for _, outcome := range NotSatisfiedOutcomes {
		if outcome == OutcomeUnmet {
			continue
		}
		if !reachable[outcome] {
			t.Fatalf("%s is not a name command.FailureOutcome returns", outcome)
		}
	}
}
