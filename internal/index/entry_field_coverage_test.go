package index

import (
	"reflect"
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/realization"
)

// Round 45 added worldFieldCoverage, which checks that index.Build reads each
// ir.World field. That guard covers one step:
//
//	ir.World --[checked]--> index.Build --[NOT checked]--> reference graph
//
// A field marked "read" can still lose its substructure in the entry type the
// index builds. Persistence is the live example: index.Build reads
// w.Persistence, but PersistenceIndexEntry keeps 4 of the source's 10 fields, so
// column mappings never become edges (#4001).
//
// entryFieldCoverage closes that second step: for each index entry type that
// mirrors a World object, every source field either appears in the entry or is
// listed as intentionally dropped.

// entryFieldCoverage records, per source type, which of its fields the index
// entry type keeps. A dropped field is a hole in what the reference graph can
// ever show, so it needs a reason.
var entryFieldCoverage = []struct {
	// source is the World-side type the index copies from.
	source reflect.Type
	// entry is the index-side type it copies into.
	entry reflect.Type
	// renamed maps a source field name to the entry field that carries it,
	// for entries that keep the value under another name.
	renamed map[string]string
	// dropped maps a source field name to why the entry does not keep it.
	dropped map[string]string
}{
	{
		source: reflect.TypeOf(realization.CustomGoBinding{}),
		entry:  reflect.TypeOf(BindingIndexEntry{}),
		// Both source fields are kept, so nothing is dropped.
	},
	{
		source: reflect.TypeOf(persistence.Mapping{}),
		entry:  reflect.TypeOf(PersistenceIndexEntry{}),
		// The mapping's own ID is stored as Mapping. Columns is kept: the column
		// correspondence is what #4001 needed, so it is no longer a dropped field.
		renamed: map[string]string{"ID": "Mapping"},
		dropped: map[string]string{
			"RelationStorage":   "#4001: not carried by the entry type",
			"PrimaryKey":        "#4001: not carried by the entry type",
			"UniqueConstraints": "#4001: not carried by the entry type",
			"Indexes":           "#4001: not carried by the entry type",
			"ForeignKeys":       "#4001: not carried by the entry type",
		},
	},
}

// TestEntryTypesKeepOrExplainEverySourceField fails when an index entry type
// silently stops carrying a field of the World object it mirrors. The reference
// graph can only ever show what reached the entry, so a field missing from both
// the entry and the dropped list is a hole nobody decided on.
func TestEntryTypesKeepOrExplainEverySourceField(t *testing.T) {
	for _, row := range entryFieldCoverage {
		sourceName := row.source.Name()
		for i := 0; i < row.source.NumField(); i++ {
			field := row.source.Field(i).Name
			if _, kept := row.entry.FieldByName(field); kept {
				continue
			}
			if target, ok := row.renamed[field]; ok {
				if _, keptElsewhere := row.entry.FieldByName(target); !keptElsewhere {
					t.Errorf("%s.%s is recorded as renamed to %s.%s, which does not exist",
						sourceName, field, row.entry.Name(), target)
				}
				continue
			}
			if _, explained := row.dropped[field]; explained {
				continue
			}
			t.Errorf("%s.%s neither appears in %s nor is listed as dropped; say why the entry does not keep it",
				sourceName, field, row.entry.Name())
		}
		// A stale row would hide a later rename.
		for field := range row.dropped {
			if _, exists := row.source.FieldByName(field); !exists {
				t.Errorf("%s no longer has field %s; remove the dropped row", sourceName, field)
			}
			if _, kept := row.entry.FieldByName(field); kept {
				t.Errorf("%s.%s is listed as dropped but %s keeps it", sourceName, field, row.entry.Name())
			}
		}
	}
}

// TestEntryTypesArePresent guards against a rename that would make the table
// above silently match nothing.
func TestEntryTypesArePresent(t *testing.T) {
	for _, row := range entryFieldCoverage {
		if row.source.NumField() == 0 || row.entry.NumField() == 0 {
			t.Errorf("%s -> %s: one side has no fields", row.source, row.entry)
		}
	}
}
