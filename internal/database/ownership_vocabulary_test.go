package database

import "testing"

// Ownerships is quoted by INVALID_DATABASE_OWNERSHIP, so the list and Valid must
// be one answer (#4094).
func TestOwnershipsAndValidAgree(t *testing.T) {
	declared := []Ownership{OwnershipManagedHere, OwnershipExternalObserved}
	if len(declared) != len(Ownerships) {
		t.Fatalf("Ownerships has %d entries but %d are declared", len(Ownerships), len(declared))
	}
	for _, v := range Ownerships {
		if !v.Valid() {
			t.Errorf("Ownerships lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []Ownership{"", "managed-elsewhere", "ManagedHere"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted %q", bad)
		}
	}
}
