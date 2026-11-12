package facetmappingcheck

import "testing"

// inverseMatrix pins the table the issue asks for: which verdict and which loss
// lets a declared inverse stay eligible. The table is the contract, so it is
// asserted directly instead of being inferred from fixture outcomes.
func TestInverseQualificationMatrix(t *testing.T) {
	base := func() Case {
		return Case{
			ID: "base",
			Structure: Structure{ID: "s", Family: "collection",
				Facets: Facets{Order: "ordered", OrderOrigin: "source", Multiplicity: "multi", Grain: "line"}},
			Representation: Representation{ID: "r", Kind: "table", ContextID: "ctx",
				Facets: Facets{Order: "ordered", OrderOrigin: "source", Multiplicity: "multi", Grain: "line"}},
			Mapping: Mapping{
				Identity:  IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"id carried"}},
				Order:     Claim{ClaimedPreserved: true, Evidence: []string{"source order rendered"}},
				Multiset:  MultisetColumn{Fanout: 1, ClaimedPreserved: true, Evidence: []string{"no join"}},
				Grain:     GrainColumn{SourceGrain: "line", TargetGrain: "line", ClaimedPreserved: true, Evidence: []string{"same grain"}},
				Coverage:  CoverageColumn{Claimed: "complete", Observed: "complete"},
				Operation: OperationColumn{Mode: "edit-inverse", InverseEligible: true},
			},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Case)
		want    string
		verdict string
	}{
		{
			name:    "preserved with a declared inverse is eligible",
			mutate:  func(*Case) {},
			want:    InverseEligible,
			verdict: VerdictPreserved,
		}, {
			name: "order loss blocks the inverse",
			mutate: func(c *Case) {
				c.Representation.Facets.Order = "unordered"
				c.Representation.Facets.OrderOrigin = "none"
			},
			want:    InverseIneligible,
			verdict: VerdictLossy,
		}, {
			name: "grain loss with a declared inverse is rejected, not merely lossy",
			mutate: func(c *Case) {
				c.Mapping.Grain = GrainColumn{SourceGrain: "line", TargetGrain: "order", Steps: 1}
			},
			want:    InverseIneligible,
			verdict: VerdictRejected,
		}, {
			name: "rejected blocks the inverse",
			mutate: func(c *Case) {
				c.Mapping.Source.Revision = "r1"
				c.Mapping.Target.Revision = "r2"
			},
			want:    InverseIneligible,
			verdict: VerdictRejected,
		}, {
			name: "unqualified blocks the inverse",
			mutate: func(c *Case) {
				c.Mapping.Coverage = CoverageColumn{Claimed: "", Observed: ""}
				c.Mapping.Quantity = QuantityColumn{SourceUnit: "JPY", TargetUnit: "USD"}
			},
			want:    InverseIneligible,
			verdict: VerdictUnqualified,
		}, {
			name: "unknown blocks the inverse",
			mutate: func(c *Case) {
				c.Mapping.Identity = IdentityColumn{Kind: "1:1", ClaimedPreserved: true}
			},
			want:    InverseIneligible,
			verdict: VerdictUnknown,
		}, {
			name: "no declared inverse stays not-claimed even when preserved",
			mutate: func(c *Case) {
				c.Mapping.Operation = OperationColumn{Mode: "read-only", InverseEligible: false}
			},
			want:    InverseNotClaimed,
			verdict: VerdictPreserved,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := base()
			c.ID = tc.name
			tc.mutate(&c)
			report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
			got := report.Results[0]
			if got.Verdict != tc.verdict {
				t.Errorf("verdict = %q, want %q (losses %v)", got.Verdict, tc.verdict, got.Losses)
			}
			if got.Inverse != tc.want {
				t.Errorf("inverse = %q, want %q", got.Inverse, tc.want)
			}
		})
	}
}

// TestApproximationIsNeverPreserved pins the planner's rule that an approximate
// conversion is reported as a loss rather than as an unqualified success.
func TestApproximationIsNeverPreserved(t *testing.T) {
	c := Case{
		ID: "approx",
		Structure: Structure{ID: "s", Family: "quantity",
			Facets: Facets{Identity: "value", Grain: "row", Precision: "exact-decimal"}},
		Representation: Representation{ID: "r", Kind: "chart", ContextID: "ctx",
			Facets: Facets{Identity: "value", Grain: "row", Precision: "float-approx"}},
		Mapping: Mapping{
			Identity: IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"id carried"}},
			Multiset: MultisetColumn{Fanout: 1, ClaimedPreserved: true, Evidence: []string{"no join"}},
			Grain:    GrainColumn{SourceGrain: "row", TargetGrain: "row", ClaimedPreserved: true, Evidence: []string{"same grain"}},
			Quantity: QuantityColumn{
				SourceUnit: "JPY", TargetUnit: "JPY",
				SourceScale: "point", TargetScale: "point",
				Precision: "exact-decimal", TargetPrecision: "float-approx",
				Approximation:    "float64",
				ClaimedPreserved: true,
				Evidence:         []string{"rendered value compares equal"},
			},
			Coverage:  CoverageColumn{Claimed: "complete", Observed: "complete"},
			Operation: OperationColumn{Mode: "read-only", InverseEligible: false},
		},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("an approximation claimed as preserved must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagApproximationClaimed)
	assertLoss(t, got, LossPrecision)
}

// TestContextBindingMismatchIsRejected pins that one representation instance
// belongs to one context, and that a different context or epoch is not reused.
func TestContextBindingMismatchIsRejected(t *testing.T) {
	c := Case{
		ID: "ctx",
		Structure: Structure{ID: "s", Family: "record",
			Facets: Facets{Identity: "subject", Grain: "row"}},
		Representation: Representation{ID: "r", Kind: "table", ContextID: "ops.queue", Epoch: "e1",
			Facets: Facets{Identity: "subject", Grain: "row"}},
		Mapping: Mapping{
			Source:    SourceColumn{Subject: "sales.order", Revision: "r1"},
			Target:    TargetColumn{Representation: "r", ContextID: "order_entry.list", Epoch: "e2", Revision: "r1"},
			Identity:  IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"id carried"}},
			Multiset:  MultisetColumn{Fanout: 1},
			Grain:     GrainColumn{SourceGrain: "row", TargetGrain: "row", ClaimedPreserved: true, Evidence: []string{"same grain"}},
			Coverage:  CoverageColumn{Claimed: "complete", Observed: "complete"},
			Operation: OperationColumn{Mode: "read-only"},
		},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("a context mismatch must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagContextMismatch)
	assertLoss(t, got, LossContextIdentity)
}

func assertLoss(t *testing.T, r Result, code string) {
	t.Helper()
	for _, l := range r.Losses {
		if l == code {
			return
		}
	}
	t.Fatalf("result %s has no %s loss: %v", r.ID, code, r.Losses)
}
