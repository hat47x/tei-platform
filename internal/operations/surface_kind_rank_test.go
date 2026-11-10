package operations

import (
	"testing"

	"tei/internal/database"
)

// operationKind rank assigns a position to four kinds and falls back to a
// catch-all for anything else. That fallback is how an unregistered kind
// silently sorts last instead of failing review, which is the gap recorded in
// the operation-surface decision brief (D-2). These tests pin the relation
// between the ranking vocabulary and the kinds the derivation actually emits.
const unknownKindRank = 5

// Every kind the derivation emits must have its own rank. A new kind that
// reaches the surface without a rank would land on the catch-all.
func TestEveryDerivedKindHasItsOwnRank(t *testing.T) {
	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessReadWrite, true), "")

	kinds := map[string]bool{}
	for _, op := range surface.Operations {
		kinds[op.Kind] = true
	}
	if len(kinds) == 0 {
		t.Fatal("the fixture produced no operations; the invariant is untested")
	}
	for kind := range kinds {
		if operationKindRank(kind) == unknownKindRank {
			t.Fatalf("derived kind %q has no rank and falls through to the catch-all", kind)
		}
	}
}

// The four ranked kinds are exactly the kinds the derivation can emit today.
// "retire" is ranked even though it is only ever a gap, so the vocabulary is
// the emitted set plus retire. A kind added here without being emitted, or vice
// versa, has to be a deliberate change to both sides.
func TestTheRankedVocabularyMatchesTheDerivation(t *testing.T) {
	ranked := []string{"list", "read", "create", "update", "retire"}
	for _, kind := range ranked {
		if operationKindRank(kind) == unknownKindRank {
			t.Fatalf("ranked kind %q falls through to the catch-all", kind)
		}
	}

	surface := DeriveOperationSurface(surfaceTestWorld(database.AccessReadWrite, true), "")
	for _, op := range surface.Operations {
		if op.Kind == "retire" {
			t.Fatalf("retire is ranked but must never be emitted as an operation: %s", op.ID)
		}
	}
}

// The catch-all is reachable only for a kind outside the vocabulary, and a
// ranking must put the catch-all after every ranked kind. This is what makes an
// unknown kind visible at the end of the list rather than mixed in.
func TestAnUnknownKindSortsAfterEveryRankedKind(t *testing.T) {
	for _, kind := range []string{"list", "read", "create", "update", "retire"} {
		if operationKindRank(kind) >= operationKindRank("something-new") {
			t.Fatalf("ranked kind %q does not sort before an unknown kind", kind)
		}
	}
}
