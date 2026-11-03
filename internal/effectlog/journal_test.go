package effectlog_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/effectlog"
)

func TestFileRoundTripsRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "effects.jsonl")
	journal := effectlog.File{Path: path}
	written := []effectlog.Record{
		{RuleID: "sales.order.stock", Stage: "after-commit", Capability: "sales.stock.decrement", Succeeded: true, Observed: true},
		{RuleID: "sales.order.audit", Stage: "compensation", Capability: "sales.audit.undo", Succeeded: false, Detail: "timeout"},
	}
	for _, record := range written {
		if err := journal.Append(record); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	read, err := journal.Records()
	if err != nil {
		t.Fatalf("records: %v", err)
	}
	if len(read) != 2 {
		t.Fatalf("expected two records: %+v", read)
	}
	if read[0].RuleID != written[0].RuleID || !read[0].Observed {
		t.Fatalf("first record lost: %+v", read[0])
	}
	if read[1].Succeeded || read[1].Detail != "timeout" {
		t.Fatalf("second record lost: %+v", read[1])
	}
}

func TestRecordsOfAMissingFileIsEmpty(t *testing.T) {
	records, err := effectlog.File{Path: filepath.Join(t.TempDir(), "missing.jsonl")}.Records()
	if err != nil || len(records) != 0 {
		t.Fatalf("a missing journal must read as empty: records=%+v err=%v", records, err)
	}
}

func TestRecordsReportsADamagedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "effects.jsonl")
	content := `{"ruleId":"a","stage":"after-commit","succeeded":true}
not-json
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	records, err := (effectlog.File{Path: path}).Records()
	if err == nil {
		t.Fatal("a damaged line must be reported")
	}
	if len(records) != 1 {
		t.Fatalf("the records before the damage must still be returned: %+v", records)
	}
}
