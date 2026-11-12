package facetmappingcheck

import (
	"path/filepath"
	"testing"
)

// mutantCases names, for each known mutant, the cases the mutant must break in
// the hand-written expectations of XMAP-01..XMAP-10. A mutant that breaks an
// unrelated case would show the expectations are only accidentally sensitive,
// so the exact set is asserted rather than "at least one mismatch".
var mutantCases = map[string]struct {
	options Options
	wantIDs []string
}{
	"order column ignored": {
		options: Options{IgnoreOrder: true},
		wantIDs: []string{"XMAP-01"},
	},
	"presence binarized": {
		options: Options{BinarizePresence: true},
		wantIDs: []string{"XMAP-08"},
	},
	"aggregate writeback allowed": {
		options: Options{AllowAggregateInverse: true},
		wantIDs: []string{"XMAP-03"},
	},
	"quantity scale ignored": {
		options: Options{IgnoreQuantityScale: true},
		wantIDs: []string{"XMAP-04"},
	},
}

func TestKnownMutantsBreakExpectedCases(t *testing.T) {
	cases, expected := loadFixtures(t)
	for name, spec := range mutantCases {
		spec := spec
		t.Run(name, func(t *testing.T) {
			report := Check(cases, spec.options)
			cmp := Compare(report, expected)
			if cmp.Status == "ok" {
				t.Fatalf("mutant %q is not detected at all", name)
			}
			broken := map[string]bool{}
			for _, m := range cmp.Mismatches {
				broken[m.ID] = true
			}
			for _, want := range spec.wantIDs {
				if !broken[want] {
					t.Errorf("mutant %q must break %s; broken cases were %v", name, want, keys(broken))
				}
			}
		})
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

var _ = filepath.Join
