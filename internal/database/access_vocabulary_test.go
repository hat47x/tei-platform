package database

import "testing"

// Accesses is quoted by the INVALID_DATABASE_ACCESS diagnostic, so the list and
// Valid must be the same answer (#2094 round 54).
func TestAccessesAndValidAgree(t *testing.T) {
	declared := []Access{AccessRead, AccessWrite, AccessReadWrite}
	if len(declared) != len(Accesses) {
		t.Fatalf("Accesses has %d entries but %d are declared; update both together", len(Accesses), len(declared))
	}
	for _, v := range Accesses {
		if !v.Valid() {
			t.Errorf("Accesses lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []Access{"", "Read", "read-write-nothing", "read "} {
		if bad.Valid() {
			t.Errorf("Valid() accepted %q", bad)
		}
	}
}
