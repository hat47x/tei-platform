package operations

import (
	"sort"
	"testing"
)

// TestSectionScopeNamesBothSidesOfTheRouteGap pins #4128. `model overview` names
// the kinds the catalog does not section, and which of those `inspect` can
// still resolve by name. It never named the complement, so a caller who just
// declared a plugin or a bundle saw its kind absent and could not tell "not
// sectioned yet" from "no route exists" — they would go looking for a command
// that is not there.
func TestSectionScopeNamesBothSidesOfTheRouteGap(t *testing.T) {
	scope := referenceOverview(t).SectionScope

	if len(scope.NotCoveredKinds) == 0 {
		t.Fatal("notCoveredKinds is empty, so the statement says nothing")
	}
	if len(scope.ReadableWithoutSection) == 0 {
		t.Fatal("readableWithoutSection is empty: the fixture lost its route list")
	}
	if len(scope.NotReadableWithoutSection) == 0 {
		t.Fatal("notReadableWithoutSection is empty: the complement is missing")
	}

	// The two lists must partition the not-covered kinds exactly.
	seen := map[string]int{}
	for _, kind := range scope.ReadableWithoutSection {
		seen[kind]++
	}
	for _, kind := range scope.NotReadableWithoutSection {
		seen[kind]++
	}
	for _, kind := range scope.NotCoveredKinds {
		if seen[kind] != 1 {
			t.Fatalf("kind %q appears %d times across the two lists, want exactly 1", kind, seen[kind])
		}
	}
	if len(seen) != len(scope.NotCoveredKinds) {
		t.Fatalf("the two lists name %d kinds, but %d are not covered",
			len(seen), len(scope.NotCoveredKinds))
	}

	// A kind is "readable" only when inspect really resolves it, so the claim
	// cannot drift away from the CLI's advertised grammar.
	for _, kind := range scope.ReadableWithoutSection {
		if !supportedInspectionKind(kind) {
			t.Fatalf("%q is listed readable but inspect does not accept it", kind)
		}
	}
	for _, kind := range scope.NotReadableWithoutSection {
		if supportedInspectionKind(kind) {
			t.Fatalf("%q is listed not readable but inspect accepts it", kind)
		}
	}

	// Both lists must be sorted, like the other lists in this report.
	if !sort.StringsAreSorted(scope.ReadableWithoutSection) {
		t.Fatalf("readableWithoutSection is not sorted: %v", scope.ReadableWithoutSection)
	}
	if !sort.StringsAreSorted(scope.NotReadableWithoutSection) {
		t.Fatalf("notReadableWithoutSection is not sorted: %v", scope.NotReadableWithoutSection)
	}
}
