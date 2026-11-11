package vocabularyprototype

import (
	"testing"
)

func TestParseRangeConstraintAcceptsClosedRange(t *testing.T) {
	c, err := ParseRangeConstraint("-0.18..0.18")
	if err != nil {
		t.Fatalf("parse closed range: %v", err)
	}
	if c.Min == nil || *c.Min != -0.18 {
		t.Fatalf("min = %v, want -0.18", c.Min)
	}
	if c.Max == nil || *c.Max != 0.18 {
		t.Fatalf("max = %v, want 0.18", c.Max)
	}
	if !c.MinInclusive || !c.MaxInclusive {
		t.Fatalf("inclusive flags = %v/%v, want true/true", c.MinInclusive, c.MaxInclusive)
	}
}

func TestParseRangeConstraintRejectsEmpty(t *testing.T) {
	if _, err := ParseRangeConstraint(""); err == nil {
		t.Fatal("empty range was accepted")
	}
}

func TestParseRangeConstraintRejectsInverted(t *testing.T) {
	if _, err := ParseRangeConstraint("0.5..0.1"); err == nil {
		t.Fatal("inverted range was accepted")
	}
}

func TestRangeConstraintContainsBoundaryValues(t *testing.T) {
	c, err := ParseRangeConstraint("0.01..0.50")
	if err != nil {
		t.Fatalf("parse range: %v", err)
	}
	if !c.Contains(0.01) {
		t.Fatal("0.01 should be contained")
	}
	if !c.Contains(0.50) {
		t.Fatal("0.50 should be contained")
	}
	if c.Contains(0.0099999) {
		t.Fatal("0.0099999 should not be contained")
	}
	if c.Contains(0.5000001) {
		t.Fatal("0.5000001 should not be contained")
	}
}
