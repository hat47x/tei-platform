package hook

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
)

// phaseRule records the phase it was run for and returns the error it was built
// with.
type phaseRule struct {
	seen  []string
	err   error
	label string
}

func (r *phaseRule) Run(_ context.Context, invocation Invocation) error {
	r.seen = append(r.seen, invocation.Phase+"/"+r.label)
	return r.err
}

// byPhaseWorld declares two hooks, so a runner can be asked to use a different
// implementation for each.
func byPhaseWorld() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{
					{Phase: "before-read"},
					{Phase: "before-update"},
				},
			},
		}},
	}}}
}

// TestRunnerUsesThePhaseImplementation checks that a rule supplied for a phase
// is the one that runs, so a plugin can supply a different body per phase.
func TestRunnerUsesThePhaseImplementation(t *testing.T) {
	readRule := &phaseRule{label: "read"}
	updateRule := &phaseRule{label: "update"}
	runner := Runner{
		World: byPhaseWorld(),
		ByPhase: map[string]Implementation{
			"before-read":   readRule,
			"before-update": updateRule,
		},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Executed) != 2 {
		t.Fatalf("executed=%v", report.Executed)
	}
	if len(readRule.seen) != 1 || readRule.seen[0] != "before-read/read" {
		t.Fatalf("the read rule saw %v", readRule.seen)
	}
	if len(updateRule.seen) != 1 || updateRule.seen[0] != "before-update/update" {
		t.Fatalf("the update rule saw %v", updateRule.seen)
	}
}

// TestRunnerFallsBackToTheWholeImplementation checks that a phase without its
// own rule uses the runner's implementation rather than failing.
func TestRunnerFallsBackToTheWholeImplementation(t *testing.T) {
	shared := &phaseRule{label: "shared"}
	readRule := &phaseRule{label: "read"}
	runner := Runner{
		World:          byPhaseWorld(),
		Implementation: shared,
		ByPhase:        map[string]Implementation{"before-read": readRule},
	}
	if _, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(readRule.seen) != 1 {
		t.Fatalf("the read rule saw %v", readRule.seen)
	}
	if len(shared.seen) != 1 || shared.seen[0] != "before-update/shared" {
		t.Fatalf("the shared rule saw %v", shared.seen)
	}
}

// TestRunnerRefusesWhenNoImplementationIsAvailableForAPhase checks that a
// phase with neither its own rule nor a shared one stops rather than passing.
func TestRunnerRefusesWhenNoImplementationIsAvailableForAPhase(t *testing.T) {
	runner := Runner{
		World:   byPhaseWorld(),
		ByPhase: map[string]Implementation{"before-read": &phaseRule{label: "read"}},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err == nil {
		t.Fatal("a phase with no implementation was accepted")
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_IMPLEMENTATION_UNAVAILABLE" {
		t.Fatalf("err=%v", err)
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-read" {
		t.Fatalf("executed=%v", report.Executed)
	}
}

// TestRunnerIgnoresAnImplementationKeyedByAnUnknownPhase checks that the map is
// not a way to inject a body for a phase the order does not name.
func TestRunnerIgnoresAnImplementationKeyedByAnUnknownPhase(t *testing.T) {
	shared := &phaseRule{label: "shared"}
	runner := Runner{
		World:          byPhaseWorld(),
		Implementation: shared,
		ByPhase:        map[string]Implementation{"during-update": &phaseRule{label: "sneaky"}},
	}
	if _, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(shared.seen) != 2 {
		t.Fatalf("the shared rule saw %v", shared.seen)
	}
}

// TestRunnerUsesThePhaseImplementationForRun checks the single-phase entry point
// resolves the same way.
func TestRunnerUsesThePhaseImplementationForRun(t *testing.T) {
	readRule := &phaseRule{label: "read"}
	runner := Runner{
		World:   byPhaseWorld(),
		ByPhase: map[string]Implementation{"before-read": readRule},
	}
	if err := runner.Run(context.Background(), Invocation{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Phase:       "before-read",
	}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(readRule.seen) != 1 || readRule.seen[0] != "before-read/read" {
		t.Fatalf("the read rule saw %v", readRule.seen)
	}
}
