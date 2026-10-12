package main

import (
	"bytes"
	"fmt"
	"testing"

	"tei/internal/sourceindex"
)

func indexForActualProcedure(t *testing.T, path string, content []byte) sourceindex.Index {
	t.Helper()
	artifact, err := sourceindex.NewArtifact(path, content)
	if err != nil {
		t.Fatal(err)
	}
	id := []byte("order-entry.order-handling")
	offset := bytes.Index(content, id)
	if offset < 0 {
		t.Fatal("source reference ID is missing")
	}
	sourceRange, err := sourceindex.RangeFromOffsets(content, offset, offset+len(id))
	if err != nil {
		t.Fatal(err)
	}
	return sourceindex.Index{
		Artifacts: []sourceindex.Artifact{artifact},
		References: []sourceindex.Reference{{
			Subject: sourceindex.Subject{
				Kind: "business-procedure-authoring",
				ID:   "order-entry.order-handling",
			},
			ArtifactPath: path,
			Role:         sourceindex.RoleAuthoring,
			Range:        sourceRange,
		}},
	}
}

func TestActualProcedureCandidateCanLocatePinnedSourceIndexAnchor(t *testing.T) {
	world, source, facts := actualProcedureCandidate(t)
	path := "procedures/order-handling.json"
	idx := indexForActualProcedure(t, path, source)
	reads := 0
	report, err := InspectIndexedProcedureSnapshot(world, source, mustJSON(t, facts), idx, path, func(got string) ([]byte, error) {
		if got != path {
			return nil, fmt.Errorf("unexpected source path: %s", got)
		}
		reads++
		return append([]byte(nil), source...), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 || report.Status != "no-findings-in-declared-scope" ||
		report.IndexedProcedure == nil ||
		report.IndexedProcedure.Status != "located" ||
		report.IndexedProcedure.Subject.ID != "order-entry.order-handling" ||
		report.IndexedProcedure.Artifact.Fingerprint != facts.ProcedureJSONFingerprint ||
		report.IndexedProcedure.Reference.Role != sourceindex.RoleAuthoring ||
		report.IndexedProcedure.AuthorityLimit == "" {
		t.Fatalf("index location does not prove Accepted Business authority: reads=%d report=%+v", reads, report)
	}
}

func TestIndexedProcedureSourceFailsClosedForMissingOrAmbiguousProvenance(t *testing.T) {
	world, source, facts := actualProcedureCandidate(t)
	path := "procedures/order-handling.json"
	input := mustJSON(t, facts)
	tests := []struct {
		name string
		edit func(*sourceindex.Index, *string)
		code string
	}{
		{
			name: "unindexed procedure",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.References = nil
			},
			code: "PROCEDURE_SOURCE_ANCHOR_UNRESOLVED",
		},
		{
			name: "definition-only role cannot stand in for authoring",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.References[0].Role = sourceindex.RoleDefinition
			},
			code: "PROCEDURE_SOURCE_ANCHOR_UNRESOLVED",
		},
		{
			name: "procedure path absent from index",
			edit: func(_ *sourceindex.Index, p *string) {
				*p = "procedures/another-source.json"
			},
			code: "PROCEDURE_SOURCE_UNINDEXED",
		},
		{
			name: "derived role is not an authoring anchor",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.References[0].Role = sourceindex.RoleDerived
			},
			code: "PROCEDURE_SOURCE_ANCHOR_UNRESOLVED",
		},
		{
			name: "duplicated authoring",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.References = append(idx.References, idx.References[0])
			},
			code: "PROCEDURE_SOURCE_ANCHOR_UNRESOLVED",
		},
		{
			name: "competing definition",
			edit: func(idx *sourceindex.Index, _ *string) {
				other := idx.References[0]
				other.Role = sourceindex.RoleDefinition
				idx.References = append(idx.References, other)
			},
			code: "PROCEDURE_SOURCE_ANCHOR_UNRESOLVED",
		},
		{
			name: "wrong subject ID",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.References[0].Subject.ID = "other.procedure"
			},
			code: "PROCEDURE_SOURCE_ANCHOR_UNRESOLVED",
		},
		{
			name: "stale index fingerprint",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.Artifacts[0].Fingerprint = sourceindex.Fingerprint([]byte("stale"))
			},
			code: "PROCEDURE_SOURCE_INDEX_INVALID",
		},
		{
			name: "stale range offset",
			edit: func(idx *sourceindex.Index, _ *string) {
				idx.References[0].Range.Start.Line++
			},
			code: "PROCEDURE_SOURCE_INDEX_INVALID",
		},
		{
			name: "missing source reader",
			edit: func(idx *sourceindex.Index, _ *string) {},
			code: "PROCEDURE_SOURCE_READER_UNAVAILABLE",
		},
		{
			name: "escaped source path",
			edit: func(_ *sourceindex.Index, p *string) {
				*p = "../outside.json"
			},
			code: "PROCEDURE_SOURCE_PATH_INVALID",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idx := indexForActualProcedure(t, path, source)
			selected := path
			tc.edit(&idx, &selected)
			reader := func(got string) ([]byte, error) {
				if got != path {
					return nil, fmt.Errorf("unexpected path: %s", got)
				}
				return source, nil
			}
			if tc.code == "PROCEDURE_SOURCE_READER_UNAVAILABLE" {
				reader = nil
			}
			report, err := InspectIndexedProcedureSnapshot(world, source, input, idx, selected, reader)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "unresolved" || codeCount(report, tc.code) != 1 ||
				report.IndexedProcedure != nil {
				t.Fatalf("unverified indexed Procedure must not look accepted: %+v", report)
			}
		})
	}
}

func TestIndexedProcedureReaderCannotSwapDifferentPinnedSource(t *testing.T) {
	world, source, facts := actualProcedureCandidate(t)
	path := "procedures/order-handling.json"
	changed := append(append([]byte(nil), source...), byte(' '))
	// The index has been rebuilt for the changed source, so index.Validate
	// alone succeeds. That source still must match the CUD facts input.
	idx := indexForActualProcedure(t, path, changed)
	report, err := InspectIndexedProcedureSnapshot(world, source, mustJSON(t, facts), idx, path, func(string) ([]byte, error) {
		return changed, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || codeCount(report, "PROCEDURE_SOURCE_BYTES_MISMATCH") != 1 ||
		report.IndexedProcedure != nil {
		t.Fatalf("different source revision silently substituted: %+v", report)
	}
}
