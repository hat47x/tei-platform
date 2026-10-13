package main

import (
	"bytes"
	"testing"

	"tei/internal/sourceindex"
)

func TestProcedureIndexRejectsUnrelatedButValidSourceRange(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	path := "procedures/order-handling.json"
	idx := indexForActualProcedure(t, path, procedure)
	// Both byte offsets and line/column data are perfectly valid, but the
	// anchor points to schemaVersion rather than the actual procedure.id.
	start := bytes.Index(procedure, []byte("business-procedure-reference-v0"))
	if start < 0 {
		t.Fatal("schemaVersion absent")
	}
	unrelated, err := sourceindex.RangeFromOffsets(procedure, start, start+len("business-procedure-reference-v0"))
	if err != nil {
		t.Fatal(err)
	}
	idx.References[0].Range = unrelated
	if err := idx.Validate(func(string) ([]byte, error) { return procedure, nil }); err != nil {
		t.Fatalf("this regression requires an otherwise valid Source Index: %v", err)
	}
	report, err := InspectIndexedProcedureSnapshot(world, procedure, mustJSON(t, facts), idx, path, func(string) ([]byte, error) { return procedure, nil })
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || report.IndexedProcedure != nil ||
		codeCount(report, "PROCEDURE_SOURCE_ANCHOR_RANGE_MISMATCH") != 1 {
		t.Fatalf("unrelated source range must not appear to locate Procedure identity: %+v", report)
	}
}

func TestProcedureIndexRejectsDecoyIDInAnotherJSONField(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	edited := bytes.Replace(procedure, []byte(`"purpose": "`), []byte(`"purpose": "order-entry.order-handling; `), 1)
	if bytes.Equal(edited, procedure) {
		t.Fatal("purpose field not found")
	}
	facts.ProcedureJSONFingerprint = snapshotDigest(edited)
	path := "procedures/order-handling.json"
	idx := indexForActualProcedure(t, path, edited)
	decoy := bytes.LastIndex(edited, []byte("order-entry.order-handling"))
	if decoy < 0 {
		t.Fatal("decoy absent")
	}
	r, err := sourceindex.RangeFromOffsets(edited, decoy, decoy+len("order-entry.order-handling"))
	if err != nil {
		t.Fatal(err)
	}
	idx.References[0].Range = r
	report, err := InspectIndexedProcedureSnapshot(world, edited, mustJSON(t, facts), idx, path, func(string) ([]byte, error) { return edited, nil })
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || report.IndexedProcedure != nil ||
		codeCount(report, "PROCEDURE_SOURCE_ANCHOR_RANGE_MISMATCH") != 1 {
		t.Fatalf("same ID in an unrelated field must not validate the wrong range: %+v", report)
	}
}

func TestProcedureIndexAcceptsWholeFileAndPointAuthoringRanges(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	path := "procedures/order-handling.json"
	_, idStart, _, ok := procedureIDValueSpan(procedure)
	if !ok {
		t.Fatal("could not locate Procedure ID")
	}
	tests := []struct {
		name  string
		start int
		end   int
	}{
		{name: "SourceContribution full-file", start: 0, end: len(procedure)},
		{name: "Source point at ID", start: idStart, end: idStart},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idx := indexForActualProcedure(t, path, procedure)
			r, err := sourceindex.RangeFromOffsets(procedure, tc.start, tc.end)
			if err != nil {
				t.Fatal(err)
			}
			idx.References[0].Range = r
			report, err := InspectIndexedProcedureSnapshot(world, procedure, mustJSON(t, facts), idx, path, func(string) ([]byte, error) { return procedure, nil })
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "no-findings-in-declared-scope" || report.IndexedProcedure == nil {
				t.Fatalf("valid SourceContribution / point range rejected: %+v", report)
			}
		})
	}
}

func TestProcedureIndexRejectsDuplicateProcedureIDKeys(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	edited := bytes.Replace(procedure,
		[]byte(`"procedure": {`),
		[]byte(`"procedure": {"id": "order-entry.order-handling",`), 1)
	if bytes.Equal(edited, procedure) {
		t.Fatal("procedure object not found")
	}
	facts.ProcedureJSONFingerprint = snapshotDigest(edited)
	path := "procedures/order-handling.json"
	idx := indexForActualProcedure(t, path, edited)
	report, err := InspectIndexedProcedureSnapshot(world, edited, mustJSON(t, facts), idx, path, func(string) ([]byte, error) { return edited, nil })
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "unresolved" || report.IndexedProcedure != nil ||
		codeCount(report, "PROCEDURE_SOURCE_ID_ANCHOR_UNRESOLVED") != 1 {
		t.Fatalf("duplicate procedure.id JSON keys must fail closed: %+v", report)
	}
}

func TestProcedureIndexAcceptsEscapedJSONIDAtItsActualLocation(t *testing.T) {
	world, procedure, facts := actualProcedureCandidate(t)
	edited := bytes.Replace(procedure,
		[]byte(`"id": "order-entry.order-handling"`),
		[]byte(`"id": "order\u002dentry.order-handling"`), 1)
	if bytes.Equal(edited, procedure) {
		t.Fatal("procedure identity not found")
	}
	facts.ProcedureJSONFingerprint = snapshotDigest(edited)
	path := "procedures/order-handling.json"
	idx := indexForActualProcedure(t, path, procedure)
	_, start, end, ok := procedureIDValueSpan(edited)
	if !ok {
		t.Fatal("escaped ID string no longer resolves")
	}
	idx.Artifacts[0].Fingerprint = sourceindex.Fingerprint(edited)
	r, err := sourceindex.RangeFromOffsets(edited, start, end)
	if err != nil {
		t.Fatal(err)
	}
	idx.References[0].Range = r
	report, err := InspectIndexedProcedureSnapshot(world, edited, mustJSON(t, facts), idx, path, func(string) ([]byte, error) { return edited, nil })
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "no-findings-in-declared-scope" || report.IndexedProcedure == nil {
		t.Fatalf("same semantic ID encoded with JSON escapes must resolve: %+v", report)
	}
}
