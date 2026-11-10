package database_test

import (
	"testing"

	"tei/internal/database"
)

// A diagnostic quotes Ownerships. If Valid accepted something outside the slice,
// the message would offer a value it then rejects — the mismatch #4094 is about.
func TestOwnershipsSliceAndValidAgree(t *testing.T) {
	for _, value := range database.Ownerships {
		if !value.Valid() {
			t.Errorf("Ownerships lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []database.Ownership{"", "managed", "external", "MANAGED-HERE"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in Ownerships", invalid)
		}
	}
	if len(database.Ownerships) != 2 {
		t.Fatalf("Ownerships has %d entries; update the test when the vocabulary grows", len(database.Ownerships))
	}
}
