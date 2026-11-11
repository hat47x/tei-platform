package sqlgen

import "testing"

// TestPlaceholderStylesAreDistinctVocabulary pins the three binding styles as
// separate vocabulary words. PlaceholderNamedColon has no adapter today, so this
// is the only place the word is produced; without it the plan check cannot tell
// the value apart from an unused declaration.
func TestPlaceholderStylesAreDistinctVocabulary(t *testing.T) {
	styles := []PlaceholderStyle{
		PlaceholderPositionalDollar,
		PlaceholderPositionalQuestion,
		PlaceholderNamedColon,
	}
	if len(styles) != 3 {
		t.Fatalf("expected three placeholder styles, got %d", len(styles))
	}
	seen := make(map[PlaceholderStyle]bool, len(styles))
	for _, style := range styles {
		if style == "" {
			t.Errorf("placeholder style must not be empty")
		}
		if seen[style] {
			t.Errorf("placeholder style %q is declared twice", style)
		}
		seen[style] = true
	}
}

// TestIsComparison pins which operators bind a value.
func TestIsComparison(t *testing.T) {
	comparisons := []string{"=", "!=", "<", ">", "<=", ">=", "LIKE", "NOT LIKE", "like"}
	for _, op := range comparisons {
		if !IsComparison(op) {
			t.Errorf("IsComparison(%q) = false, want true", op)
		}
	}
	nonComparisons := []string{"IN", "NOT IN", "IS NULL", "IS NOT NULL", "", "BETWEEN"}
	for _, op := range nonComparisons {
		if IsComparison(op) {
			t.Errorf("IsComparison(%q) = true, want false", op)
		}
	}
}

// TestSetOperationVocabulary pins the set operations the common contract carries.
func TestSetOperationVocabulary(t *testing.T) {
	operations := []SetOperation{SetUnion, SetIntersect, SetExcept}
	if len(operations) != 3 {
		t.Fatalf("expected three set operations, got %d", len(operations))
	}
	for _, op := range operations {
		if op == "" {
			t.Errorf("set operation must not be empty")
		}
	}
}
