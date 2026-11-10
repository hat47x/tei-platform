package plancheck

import (
	"regexp"
	"testing"
)

// numberedFiles builds a regex quantifier from a digit count, like
//
//	regexp.MustCompile(`^([0-9]{` + itoa(digits) + `})-.+\.md$`)
//
// itoa used to be string(rune('0' + n)), which only spells 0..9 and then turns
// into punctuation: itoa(10) is ":", so the pattern became `[0-9]{:}`. That
// compiles and matches nothing, so the uniqueness check silently stopped
// checking anything the moment a caller asked for 10 digits (#3919).
func TestItoaSpellsMultiDigitCounts(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want string
	}{
		{0, "0"},
		{4, "4"},
		{6, "6"},
		{9, "9"},
		{10, "10"},
		{23, "23"},
		{514, "514"},
	} {
		if got := itoa(tc.in); got != tc.want {
			t.Fatalf("itoa(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A quantifier built from itoa must keep matching files if the digit count
// grows. This is the shape that silently broke: a pattern that compiles but
// matches nothing makes the uniqueness assertion vacuous.
func TestDigitCountPatternStillMatches(t *testing.T) {
	for _, digits := range []int{1, 2, 4, 6, 10, 12} {
		pattern := regexp.MustCompile(`^([0-9]{` + itoa(digits) + `})-.+\.md$`)
		name := ""
		for i := 0; i < digits; i++ {
			name += "7"
		}
		name += "-entry.md"
		if !pattern.MatchString(name) {
			t.Fatalf("digits=%d pattern %q does not match %q", digits, pattern, name)
		}
		// And it must still reject a shorter number.
		if digits > 1 && pattern.MatchString("7-entry.md") {
			t.Fatalf("digits=%d pattern %q matched a 1-digit name", digits, pattern)
		}
	}
}
