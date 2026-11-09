package access_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

// TestTheRecordIsTheDecision pins that the journal keeps exactly what the caller
// was told. They used to be two structs with the same fields, converted by hand;
// a field added to one and not the other would have been dropped at the
// conversion and nothing would have reported it.
func TestTheRecordIsTheDecision(t *testing.T) {
	if reflect.TypeOf(access.Decision{}) != reflect.TypeOf(access.Record{}) {
		t.Fatal("the decision and the record must be one type, not two that happen to agree today")
	}
}

// TestTheJournalWritesWhatTheCallerReads compares the decision the caller gets
// with the line the journal writes. Two types with the same fields can still
// serialize differently, so the check is on the bytes, not on the shape.
func TestTheJournalWritesWhatTheCallerReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.jsonl")
	guard := access.Guard{
		Granted: []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}},
		Journal: access.File{Path: path},
	}
	decision, err := guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{
		{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/out"},
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	records, err := (access.File{Path: path}).Records()
	if err != nil || len(records) != 1 {
		t.Fatalf("the decision must be recorded: %+v err=%v", records, err)
	}
	fromDecision, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("encode decision: %v", err)
	}
	fromRecord, err := json.Marshal(records[0])
	if err != nil {
		t.Fatalf("encode record: %v", err)
	}
	if string(fromDecision) != string(fromRecord) {
		t.Fatalf("the caller and the journal disagree:\n decision: %s\n record:   %s", fromDecision, fromRecord)
	}
}
