package numbering

import "testing"

// Overflow says what a numbering sequence does when it runs out. The choice
// changes whether a document is refused or wraps, so both values are named here
// (#4032).
func TestOverflowValuesAreCovered(t *testing.T) {
	values := []Overflow{OverflowStop, OverflowWrap}
	want := []string{"stop", "wrap"}
	if len(values) != len(want) {
		t.Fatalf("an overflow value was dropped: %d values", len(values))
	}
	for i, value := range values {
		if string(value) != want[i] {
			t.Fatalf("values[%d]=%q want %q", i, value, want[i])
		}
	}
	if OverflowStop == OverflowWrap {
		t.Fatal("stop and wrap share a value")
	}
}
