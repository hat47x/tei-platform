package hook

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

// proposingWorld declares a before-update hook that declares one read and one
// proposal.
func proposingWorld() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{{
					Phase:    "before-update",
					Reads:    []core.CanonicalID{"sales.order.quantity"},
					Produces: []core.CanonicalID{"sales.order.amount"},
				}},
			},
		}},
	}}}
}

// proposingRule proposes the values it was constructed with.
type proposingRule struct {
	produced Produced
}

func (r proposingRule) Run(context.Context, Invocation) error { return nil }

func (r proposingRule) RunProducing(context.Context, Invocation) (Produced, error) {
	return r.produced, nil
}

func proposingInputs() map[core.CanonicalID]string {
	return map[core.CanonicalID]string{"sales.order.quantity": "2"}
}

// TestRunReachableKeepsTheProposalADeclaredHookMade checks that a value a
// declared hook proposes reaches the report instead of being dropped by the
// read-only path.
func TestRunReachableKeepsTheProposalADeclaredHookMade(t *testing.T) {
	runner := Runner{
		World:          proposingWorld(),
		Implementation: proposingRule{produced: Produced{"sales.order.amount": "2000"}},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", proposingInputs(), ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if report == nil {
		t.Fatal("no report")
	}
	proposed := report.Proposed("before-update")
	if proposed["sales.order.amount"] != "2000" {
		t.Fatalf("the proposal was dropped: %+v (report.Produced=%+v)", proposed, report.Produced)
	}
	if len(report.Executed) != 1 || report.Executed[0] != "before-update" {
		t.Fatalf("executed=%v", report.Executed)
	}
}

// TestRunReachableRefusesAnUndeclaredProposal checks that the reachable path
// applies the declaration check: a hook that proposes an attribute it did not
// declare is refused, not silently accepted.
func TestRunReachableRefusesAnUndeclaredProposal(t *testing.T) {
	runner := Runner{
		World:          proposingWorld(),
		Implementation: proposingRule{produced: Produced{"sales.order.secret": "x"}},
	}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", proposingInputs(), ReferenceReach{})
	if err == nil {
		t.Fatal("an undeclared proposal was accepted")
	}
	var hookError *Error
	if !errors.As(err, &hookError) || hookError.Code != "HOOK_PRODUCE_UNDECLARED" {
		t.Fatalf("err=%v", err)
	}
	if report == nil || report.Refused != "before-update" {
		t.Fatalf("report=%+v", report)
	}
	if len(report.Produced) != 0 {
		t.Fatalf("a refused phase kept a proposal: %+v", report.Produced)
	}
}

// TestRunReachableRecordsNoProposalForAReadOnlyHook checks that a hook which
// proposes nothing adds no entry, so an empty map is not read as "proposed".
func TestRunReachableRecordsNoProposalForAReadOnlyHook(t *testing.T) {
	world := ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{{Phase: "before-read"}},
			},
		}},
	}}}
	runner := Runner{World: world, Implementation: ReferenceRule{}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Produced) != 0 {
		t.Fatalf("a read-only hook recorded a proposal: %+v", report.Produced)
	}
	if len(report.Executed) != 1 {
		t.Fatalf("executed=%v", report.Executed)
	}
}

// TestRunReachableKeepsProposalsPerPhase checks that two phases do not merge
// their proposals into one.
func TestRunReachableKeepsProposalsPerPhase(t *testing.T) {
	world := ir.World{Applications: []application.Application{{
		ID: "order_entry",
		Patterns: []application.Pattern{{
			ID:   "order_entry.order_record",
			Kind: application.PatternRecord,
			Data: "sales.order",
			Interaction: &application.Interaction{
				Hooks: []application.Hook{
					{Phase: "before-read", Produces: []core.CanonicalID{"sales.order.amount"}},
					{Phase: "before-update", Produces: []core.CanonicalID{"sales.order.amount"}},
				},
			},
		}},
	}}}
	// The same implementation answers both phases, so the report has to keep
	// them apart by phase rather than by value.
	runner := Runner{World: world, Implementation: proposingRule{produced: Produced{"sales.order.amount": "1"}}}
	report, err := runner.RunReachable(context.Background(), "order_entry", "order_entry.order_record", nil, ReferenceReach{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(report.Produced) != 2 {
		t.Fatalf("produced=%+v", report.Produced)
	}
	if _, ok := report.Produced["before-read"]; !ok {
		t.Fatalf("before-read has no proposal: %+v", report.Produced)
	}
	if _, ok := report.Produced["before-update"]; !ok {
		t.Fatalf("before-update has no proposal: %+v", report.Produced)
	}
}
