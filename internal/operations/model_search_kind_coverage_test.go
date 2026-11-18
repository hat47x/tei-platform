package operations

import (
	"path/filepath"
	"testing"
)

// `model search -kind <k>` accepts every kind `inspect` resolves, but it can
// only enumerate the kinds `model catalog` lists. A kind outside that list
// answers zero hits however many declarations of that kind the Workspace holds,
// so an empty page must not be read as "this term is absent" (#5228).
//
// The expected list below is written from the two closed sets: the kinds the
// catalog traversal produces, and the kinds the inspection surface accepts. The
// remainder is what the search must name.
func TestSearchCoverageNamesTheKindsItCannotEnumerate(t *testing.T) {
	searchable := []string{
		"semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation",
		"semantic-candidate-key", "semantic-functional-dependency", "semantic-capability",
		"semantic-validation", "application", "application-pattern", "query",
	}
	want := map[string]bool{}
	for _, kind := range InspectionSubjectKinds {
		want[kind] = true
	}
	for _, kind := range searchable {
		delete(want, kind)
	}

	got := searchCoverage().NotSearchableKinds
	if len(got) != len(want) {
		t.Fatalf("notSearchableKinds has %d entries, want %d: %v", len(got), len(want), got)
	}
	for _, kind := range got {
		if !want[kind] {
			t.Fatalf("%q is reported unsearchable but the catalog enumerates it", kind)
		}
		delete(want, kind)
	}
	for kind := range want {
		t.Fatalf("%q is accepted as -kind but neither searchable nor reported", kind)
	}
}

// A search restricted to an unsearchable kind answers zero hits, and the answer
// says why. The check is the response, not the code: a caller reads the JSON.
func TestSearchRestrictedToAnUnsearchableKindNamesIt(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	kinds := NotSearchableKinds()
	if len(kinds) == 0 {
		t.Fatal("no unsearchable kind to check")
	}
	// A term that does occur in the Workspace, so an empty answer cannot be
	// explained by an absent term.
	result, err := SearchWorkspace(root, SearchQuery{Term: "order_line", Kind: kinds[0]}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || len(result.Hits) != 0 {
		t.Fatalf("an unsearchable kind returned hits: total=%d", result.Total)
	}
	found := false
	for _, kind := range result.Coverage.NotSearchableKinds {
		if kind == kinds[0] {
			found = true
		}
	}
	if !found {
		t.Fatalf("the response does not name %q as unsearchable: %v", kinds[0], result.Coverage.NotSearchableKinds)
	}
}

// The searchable kinds still return hits, so the report is a narrowing and not a
// claim that the surface sees nothing.
func TestSearchableKindsStillReturnHits(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	// `order_line` is a term the reference Workspace declares, so an empty
	// answer here would mean the search stopped seeing searchable kinds.
	result, err := SearchWorkspace(root, SearchQuery{Term: "order_line"}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total == 0 {
		t.Fatalf("the reference Workspace declares order_line but the search found none")
	}
	for _, kind := range result.Coverage.NotSearchableKinds {
		if kind == "application" || kind == "semantic" {
			t.Fatalf("%q is reported unsearchable but it returns hits", kind)
		}
	}
}
