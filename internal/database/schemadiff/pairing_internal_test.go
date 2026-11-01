package schemadiff

import "testing"

type namedItem struct{ name string }

func itemName(item namedItem) string { return item.name }

func pairNames(pairs []constraintPair[namedItem]) [][2]string {
	out := make([][2]string, len(pairs))
	for i, pair := range pairs {
		out[i] = [2]string{pair.desired.name, pair.actual.name}
	}
	return out
}

func TestPairByNameSortsLeftoversByName(t *testing.T) {
	pairs, extraD, extraA := pairByName(
		[]namedItem{{"z_new"}, {"a_new"}},
		[]namedItem{{"n_old"}, {"m_old"}},
		itemName,
	)
	if len(extraD) != 0 || len(extraA) != 0 {
		t.Fatalf("unexpected extras: %v %v", extraD, extraA)
	}
	want := [][2]string{{"a_new", "m_old"}, {"z_new", "n_old"}}
	got := pairNames(pairs)
	if len(got) != len(want) {
		t.Fatalf("pairs=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pairs=%v want %v", got, want)
		}
	}
}

func TestPairByNameBreaksCaseTiesByRawName(t *testing.T) {
	pairs, _, _ := pairByName(
		[]namedItem{{"a"}, {"A"}},
		[]namedItem{{"b"}, {"B"}},
		itemName,
	)
	want := [][2]string{{"A", "B"}, {"a", "b"}}
	got := pairNames(pairs)
	if len(got) != len(want) {
		t.Fatalf("pairs=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pairs=%v want %v", got, want)
		}
	}
}

func TestPairByNamePairsEqualNamesBeforeOrder(t *testing.T) {
	pairs, _, _ := pairByName(
		[]namedItem{{"a_new"}, {"b_keep"}},
		[]namedItem{{"legacy"}, {"b_keep"}},
		itemName,
	)
	want := [][2]string{{"b_keep", "b_keep"}, {"a_new", "legacy"}}
	got := pairNames(pairs)
	if len(got) != len(want) {
		t.Fatalf("pairs=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pairs=%v want %v", got, want)
		}
	}
}
