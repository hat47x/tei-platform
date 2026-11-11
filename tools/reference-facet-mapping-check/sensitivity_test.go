package facetmappingcheck

import "testing"

// TestDeclaredLossPreventsPreserved pins a structural invariant of the corpus:
// a case whose contract already declares a loss cannot come out preserved. If
// one did, either the declaration or the rule is wrong, and silently accepting
// it would let an understated contract pass as a clean projection.
func TestDeclaredLossPreventsPreserved(t *testing.T) {
	cases, _ := loadFixtures(t)
	report := Check(cases, Options{})
	declared := map[string][]string{}
	for _, c := range cases.Cases {
		declared[c.ID] = c.Mapping.Loss.Declared
	}
	for _, r := range report.Results {
		if r.Verdict != VerdictPreserved {
			continue
		}
		if len(declared[r.ID]) > 0 {
			t.Errorf("case %s is preserved while declaring losses %v", r.ID, declared[r.ID])
		}
	}
}

// TestProtectedFacetsAreNeverSilentlyPreserved covers the facets the planner
// listed explicitly. For each, a case that changes only that facet on the
// representation must stop being preserved. This is a sensitivity check on the
// rules, independent of the fixture's own verdicts.
func TestProtectedFacetsAreNeverSilentlyPreserved(t *testing.T) {
	base := func() Case {
		return Case{
			ID: "sensitivity",
			Structure: Structure{ID: "s", Family: "collection",
				Facets: Facets{
					Identity: "subject", Order: "ordered", OrderOrigin: "source",
					Multiplicity: "multi", Grain: "line",
					Presence: []string{"null", "missing"},
				}},
			Representation: Representation{ID: "r", Kind: "table", ContextID: "ctx",
				Facets: Facets{
					Identity: "subject", Order: "ordered", OrderOrigin: "source",
					Multiplicity: "multi", Grain: "line",
					Presence: []string{"null", "missing"},
				}},
			Mapping: Mapping{
				Identity:  IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"id carried"}},
				Order:     Claim{ClaimedPreserved: true, Evidence: []string{"source order rendered"}},
				Multiset:  MultisetColumn{Fanout: 1, ClaimedPreserved: true, Evidence: []string{"no join"}},
				Grain:     GrainColumn{SourceGrain: "line", TargetGrain: "line", ClaimedPreserved: true, Evidence: []string{"same grain"}},
				Presence:  Claim{ClaimedPreserved: true, Evidence: []string{"states carried"}},
				Coverage:  CoverageColumn{Claimed: "complete", Observed: "complete"},
				Operation: OperationColumn{Mode: "read-only"},
			},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Case)
		wantNot string
	}{
		{
			name: "order dropped",
			mutate: func(c *Case) {
				c.Representation.Facets.Order = "unordered"
				c.Representation.Facets.OrderOrigin = "none"
			},
			wantNot: VerdictPreserved,
		}, {
			name: "multiplicity dropped",
			mutate: func(c *Case) {
				c.Representation.Facets.Multiplicity = "unique"
			},
			wantNot: VerdictPreserved,
		}, {
			name: "grain changed",
			mutate: func(c *Case) {
				c.Mapping.Grain = GrainColumn{SourceGrain: "line", TargetGrain: "order", Steps: 1}
			},
			wantNot: VerdictPreserved,
		}, {
			name: "presence collapsed",
			mutate: func(c *Case) {
				c.Representation.Facets.Presence = []string{"null", "value"}
			},
			wantNot: VerdictPreserved,
		}, {
			name: "coverage overclaimed",
			mutate: func(c *Case) {
				c.Mapping.Coverage = CoverageColumn{Claimed: "complete", Observed: "partial"}
			},
			wantNot: VerdictPreserved,
		}, {
			name: "source revision differs",
			mutate: func(c *Case) {
				c.Mapping.Source.Revision = "r1"
				c.Mapping.Target.Revision = "r2"
			},
			wantNot: VerdictPreserved,
		}, {
			name: "unit conversion without a policy",
			mutate: func(c *Case) {
				c.Mapping.Quantity = QuantityColumn{SourceUnit: "JPY", TargetUnit: "USD"}
			},
			wantNot: VerdictPreserved,
		}, {
			name: "time axes conflated",
			mutate: func(c *Case) {
				c.Mapping.Time = TimeColumn{SourceAxes: []string{"domain"}, TargetAxes: []string{"presentation"},
					SourceClock: "station", TargetClock: "device"}
			},
			wantNot: VerdictPreserved,
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
			if got.Verdict == tc.wantNot {
				t.Fatalf("%q is preserved although a protected facet changed", tc.name)
			}
			if len(got.Losses) == 0 {
				t.Fatalf("%q is not preserved but reports no loss: %+v", tc.name, got)
			}
		})
	}
}
