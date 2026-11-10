package operations

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/stdlib"
)

// worldWithStd builds a World whose std namespace carries the prelude's own
// types plus any extra the caller adds, which is what the loader produces after
// a Workspace writes `namespace std`.
func worldWithStd(extra ...string) ir.World {
	prelude := stdlib.SemanticNamespace()
	ns := semantic.Namespace{ID: prelude.ID}
	ns.Types = append(ns.Types, prelude.Types...)
	for _, id := range extra {
		ns.Types = append(ns.Types, semantic.Type{
			ID: core.MustCanonicalID(id), Symbol: id, Kind: semantic.TypePrimitive,
		})
	}
	return ir.World{Semantics: []semantic.Namespace{ns}}
}

// A Workspace may declare its own type inside the std namespace. That type is
// the caller's, so it must not be marked as the in-process prelude: the contract
// defines prelude:true as "not something the Workspace declared" (#3851).
func TestStandardNamespaceIsNotAPreludeMark(t *testing.T) {
	catalog := CatalogWorld(worldWithStd("std.email"), sourceindex.Index{}, "", "w", "s")

	byID := map[string]bool{}
	for _, entry := range catalog.Entries {
		if entry.Subject.ID == "std" || entry.Subject.ID == "std.text" || entry.Subject.ID == "std.email" {
			byID[entry.Subject.ID] = entry.Prelude
		}
	}
	if !byID["std"] {
		t.Fatal("the std namespace itself is the prelude")
	}
	if !byID["std.text"] {
		t.Fatal("std.text is declared by the prelude")
	}
	if byID["std.email"] {
		t.Fatal("std.email was declared by the Workspace, so it is not the prelude")
	}
}

// A type the Workspace writes inside std is still a declaration it authored, so
// it has to be counted. Counting used to skip the whole std namespace by name,
// which made such a Workspace report total:0 (#3851).
func TestDeclarationCountsIncludeTypesAddedToStd(t *testing.T) {
	empty := CountDeclarations(worldWithStd())
	if empty.Total != 0 {
		t.Fatalf("a prelude-only World counted %d declarations", empty.Total)
	}

	added := CountDeclarations(worldWithStd("std.email"))
	if added.Types != 1 {
		t.Fatalf("types=%d, want 1 (the added std.email)", added.Types)
	}
	if added.Total != 2 {
		// namespace + type
		t.Fatalf("total=%d, want 2", added.Total)
	}
}

// isPreludeType must answer for exactly the prelude's identities, so a name that
// merely starts with "std." is not enough.
func TestIsPreludeTypeUsesIdentity(t *testing.T) {
	if !isPreludeType("std.text") {
		t.Fatal("std.text is a prelude type")
	}
	if isPreludeType("std.email") {
		t.Fatal("std.email is not a prelude type")
	}
}
