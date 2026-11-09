package webapicontract

import (
	"context"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/semantic"
)

// TestCompletionReadSliceQualificationBoundaries characterizes an existing API
// boundary. EvaluateCompletion trusts caller-supplied coverage and predicate
// results; neither a successful page nor equal row counts proves the Owner's
// whole target set or the inputs needed to evaluate its predicate.
//
// The rowsets below are synthetic, not results of a real PostgreSQL read.
// This test does not introduce a second production qualifier or claim that the
// Function -> Query -> Provider -> Completion handoff is implemented.
func TestCompletionReadSliceQualificationBoundaries(t *testing.T) {
	fixture, err := os.ReadFile("fixtures/postgres_scoped_orders.sql")
	if err != nil {
		t.Fatal(err)
	}
	// Anchor independent expectations to the already checked-in public SQL
	// fixture, rather than deriving Expected from the supplied read slice.
	for _, want := range []string{
		"('tenant-a', 3, 11, '2026-10-01T08:00:00Z', 'OPEN'",
		"('tenant-a', 9, 12, '2026-10-02T08:00:00Z', 'READY'",
	} {
		if !strings.Contains(string(fixture), want) {
			t.Fatalf("independent tenant A baseline changed: missing %q", want)
		}
	}

	const a3 = "tenant-a/order/3"
	const a9 = "tenant-a/order/9"
	ownerTargetIDs := []string{a3, a9}
	capability := semantic.Capability{ID: "order_entry.recent_orders", EmptyPolicy: semantic.EmptyPolicyRequiresEvidence}
	complete := semantic.CompletionEvidence{
		Targets:  []semantic.TargetResult{{ID: a3, Holds: true}, {ID: a9, Holds: true}},
		Coverage: semantic.CoverageComplete,
	}
	t.Run("Q00 complete same-scope targets qualify locally", func(t *testing.T) {
		decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, complete)
		if decision.State != semantic.CompletionHolds {
			t.Fatalf("complete finite target evaluation: %+v", decision)
		}
	})

	t.Run("Q05 equal row count with repeated occurrence is not complete", func(t *testing.T) {
		// A JOIN or another projection can repeat order 3 while omitting 9.
		// A count-only test is intentionally incapable of distinguishing them.
		observedIDs := []string{a3, a3}
		if len(observedIDs) != len(ownerTargetIDs) {
			t.Fatal("negative control no longer has the same row count")
		}
		gotIDs := append([]string(nil), observedIDs...)
		wantIDs := append([]string(nil), ownerTargetIDs...)
		sort.Strings(gotIDs)
		sort.Strings(wantIDs)
		if reflect.DeepEqual(gotIDs, wantIDs) {
			t.Fatal("negative control must have a missing target despite equal row count")
		}

		evidence := semantic.CompletionEvidence{
			Targets: []semantic.TargetResult{
				{ID: a3, Holds: true}, {ID: a3, Holds: true},
			},
			Coverage: semantic.CoverageComplete, // Deliberately false caller claim.
		}
		if decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, evidence); decision.State != semantic.CompletionHolds {
			t.Fatalf("caller-supplied complete is accepted locally, not independently verified: %+v", decision)
		}
		evidence.Coverage = semantic.CoveragePartial
		if decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, evidence); decision.State != semantic.CompletionUnverified {
			t.Fatalf("the correctly reported incomplete target set must not hold: %+v", decision)
		}
	})

	// Exercise the actual reference ReadPlan builder. The page and selected
	// columns are real plan properties; returned rows are synthetic below.
	pub := scopedPublication(t)
	plan, err := AuthorizeBrowseRead(context.Background(), fixtureAuthorization(t), pub,
		url.Values{"$select": {"orderId"}, "$orderby": {"orderId desc"}, "$top": {"1"}},
		TrustedReadContext{
			PrincipalID:    "subject-order-editor",
			RowScopeID:     "scope-tenant-a",
			PolicyRevision: "e2e-policy-1",
		})
	if err != nil {
		t.Fatalf("cannot construct the existing authorized projection: %v", err)
	}
	if plan.Limit != 1 || !reflect.DeepEqual(plan.Selected, []string{"orderId"}) {
		t.Fatalf("expected an order-ID-only top-one plan, got %+v", plan)
	}

	t.Run("Q06 fulfilled top-one page is not whole population", func(t *testing.T) {
		observed := []semantic.TargetResult{{ID: a9, Holds: true}}
		if len(observed) != 1 || len(ownerTargetIDs) != 2 {
			t.Fatal("fixed one-page versus two-target contrast changed")
		}
		evidence := semantic.CompletionEvidence{Targets: observed, Coverage: semantic.CoveragePartial}
		if decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, evidence); decision.State != semantic.CompletionUnverified {
			t.Fatalf("partial page is not universal completion: %+v", decision)
		}
		evidence.Coverage = semantic.CoverageComplete // A caller cannot infer this from top=1.
		if decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, evidence); decision.State != semantic.CompletionHolds {
			t.Fatalf("the local evaluator does not validate the page witness: %+v", decision)
		}
	})

	t.Run("Q07 predicate field not projected cannot support reported truth", func(t *testing.T) {
		// Owner predicate for this case: status == READY. The plan exposes only
		// orderId. Independently, the SQL fixture has order 3 OPEN, 9 READY.
		for _, field := range plan.Selected {
			if field == "status" {
				t.Fatal("negative control unexpectedly exposes predicate input")
			}
		}
		madeUp := complete // Same IDs, but Holds=true was not computed from status.
		if decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, madeUp); decision.State != semantic.CompletionHolds {
			t.Fatalf("local evaluator trusts ungrounded predicate truth: %+v", decision)
		}
		withPredicate := semantic.CompletionEvidence{
			Targets:  []semantic.TargetResult{{ID: a3, Holds: false}, {ID: a9, Holds: true}},
			Coverage: semantic.CoverageComplete,
		}
		decision := semantic.EvaluateCompletion(capability, semantic.EmptyPolicyUniversal, withPredicate)
		if decision.State != semantic.CompletionFails || !reflect.DeepEqual(decision.Violations, []string{a3}) {
			t.Fatalf("independently known predicate violation was not detected: %+v", decision)
		}
	})
}
