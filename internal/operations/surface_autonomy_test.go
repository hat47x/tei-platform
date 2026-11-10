package operations

import (
	"testing"

	"tei/internal/database"
)

// The rule text states: "autonomy: list and read are unattended because they
// have no effect". That claim is only meaningful if no effectful kind can carry
// "unattended". This test pins the invariant over the derived surface, so a
// later change that marks a write kind unattended fails here rather than
// silently telling a consumer it may run without a person.
func TestOnlyEffectFreeKindsAreUnattended(t *testing.T) {
	// Read-write access is what produces create/update/retire alongside list and
	// read, so this fixture covers every kind the derivation can emit.
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessReadWrite, true), "")

	effectFree := map[string]bool{"list": true, "read": true}
	seenUnattended := 0
	for _, op := range surface.Operations {
		if op.Autonomy != "unattended" {
			continue
		}
		seenUnattended++
		if !effectFree[op.Kind] {
			t.Fatalf("effectful kind %q was marked unattended (operation %s)", op.Kind, op.ID)
		}
	}
	if seenUnattended == 0 {
		t.Fatal("the fixture produced no unattended operation; the invariant is untested")
	}

	// The other direction: every effect-free kind that was derived is unattended.
	for _, op := range surface.Operations {
		if effectFree[op.Kind] && op.Autonomy != "unattended" {
			t.Fatalf("effect-free kind %q was not unattended (operation %s)", op.Kind, op.ID)
		}
	}
}

// The summary count must equal the number of unattended operations, so the
// count cannot drift from the list it summarizes.
func TestTheUnattendedCountMatchesTheOperations(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessReadWrite, true), "")

	count := 0
	for _, op := range surface.Operations {
		if op.Autonomy == "unattended" {
			count++
		}
	}
	if surface.Summary.Unattended != count {
		t.Fatalf("summary says %d unattended, the operations list has %d",
			surface.Summary.Unattended, count)
	}
	if count == 0 {
		t.Fatal("the fixture produced no unattended operation")
	}
}
