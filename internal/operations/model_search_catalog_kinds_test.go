package operations

import (
	"path/filepath"
	"testing"
)

// `catalogEnumeratesKind` is a hand-kept list of the kinds `model catalog`
// produces. If the catalog starts producing a kind this list does not name, the
// search would report that kind as unsearchable — telling a caller "this term is
// absent" when the truth is "we never looked" (#5228).
//
// The expectation is the catalog traversal itself, not a copy of it: the
// reference Workspace is walked and every kind it yields must be searchable.
//
// The reverse direction is NOT checked here, and this test does not pretend to:
// `catalogEnumeratesKind` also names `semantic-candidate-key` and
// `semantic-functional-dependency`, which the reference Workspace does not
// declare, so no fixture in this package produces them. Claiming those are
// reachable is a claim about the traversal in model_catalog.go, verified by
// reading that file, not by this test.
func TestCatalogKindsAreSearchable(t *testing.T) {
	catalog, err := CatalogWorkspace(filepath.Join("..", "..", "reference", "workspace"), "")
	if err != nil {
		t.Fatal(err)
	}
	produced := map[string]bool{}
	for _, entry := range catalog.Entries {
		produced[entry.Subject.Kind] = true
	}
	if len(produced) == 0 {
		t.Fatal("the reference Workspace produced no catalog entries")
	}

	for kind := range produced {
		if !catalogEnumeratesKind(kind) {
			t.Fatalf("the catalog produces %q but the search reports it as unsearchable", kind)
		}
		if !SupportedInspectionKind(kind) {
			t.Fatalf("the catalog produces %q, which is not an accepted -kind value", kind)
		}
		if containsKind(NotSearchableKinds(), kind) {
			t.Fatalf("%q is producible but reported unsearchable", kind)
		}
	}
}

// The kinds the reference Workspace exercises are exactly the ones a search
// over that Workspace can reach, so an empty answer for any other kind is not a
// statement about the Workspace's content.
func TestUnsearchableKindsAreNotInTheReferenceCatalog(t *testing.T) {
	catalog, err := CatalogWorkspace(filepath.Join("..", "..", "reference", "workspace"), "")
	if err != nil {
		t.Fatal(err)
	}
	produced := map[string]bool{}
	for _, entry := range catalog.Entries {
		produced[entry.Subject.Kind] = true
	}
	for _, kind := range NotSearchableKinds() {
		if produced[kind] {
			t.Fatalf("%q is reported unsearchable but the catalog produced entries of that kind", kind)
		}
	}
}

func containsKind(kinds []string, kind string) bool {
	for _, candidate := range kinds {
		if candidate == kind {
			return true
		}
	}
	return false
}
