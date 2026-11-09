package operations

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// TestCatalogEntriesMarkTheStandardPrelude pins #3347: the catalog always
// lists the standard semantic prelude, so `entries` alone could not show
// whether a Workspace declared anything. `prelude` marks those entries and
// leaves them out of the Workspace-authored count (#2822).
func TestCatalogEntriesMarkTheStandardPrelude(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{
			ID:    "std",
			Types: []semantic.Type{{ID: "std.text", Symbol: "text"}},
		},
		{
			ID:    "sales",
			Data:  []semantic.Data{{ID: "sales.order", Symbol: "Order"}},
			Types: []semantic.Type{{ID: "sales.order_id", Symbol: "OrderId"}},
		},
	}}
	got := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")

	var prelude, owned int
	for _, entry := range got.Entries {
		if entry.Subject.ID == "std" || len(entry.Subject.ID) > 4 && entry.Subject.ID[:4] == "std." {
			if !entry.Prelude {
				t.Fatalf("prelude subject %v is not marked: %+v", entry.Subject, entry)
			}
			prelude++
			continue
		}
		if entry.Prelude {
			t.Fatalf("source-authored subject %v is marked as prelude: %+v", entry.Subject, entry)
		}
		owned++
	}
	if prelude == 0 {
		t.Fatal("the prelude is missing from the catalog")
	}
	// `sales` and `sales.order_id` are the Workspace's own; `sales.order` adds
	// one more, so the count that matters is that prelude entries are excluded
	// from it rather than a fixed number.
	if owned == 0 {
		t.Fatal("source-authored subjects were lost")
	}
	if prelude+owned != len(got.Entries) {
		t.Fatalf("entries are unaccounted for: prelude=%d owned=%d total=%d", prelude, owned, len(got.Entries))
	}
	if got.DeclarationCounts.Total == 0 {
		t.Fatal("declared elements were counted as zero")
	}
}

// TestCatalogEmptyWorkspaceIsPreludeOnly is the empty-Workspace end: every
// entry is prelude, and the declaration count is zero, so a reader can now
// distinguish the two facts without matching IDs by hand.
func TestCatalogEmptyWorkspaceIsPreludeOnly(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID:    "std",
		Types: []semantic.Type{{ID: "std.text", Symbol: "text"}},
	}}}
	got := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")
	if len(got.Entries) == 0 {
		t.Fatal("the prelude must still be listed")
	}
	for _, entry := range got.Entries {
		if !entry.Prelude {
			t.Fatalf("unexpected non-prelude entry %+v", entry)
		}
	}
	if got.DeclarationCounts.Total != 0 {
		t.Fatalf("the prelude leaked into the declaration count: %+v", got.DeclarationCounts)
	}
}
