package expectedstateupdate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuditLogAppendsAndReadsEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	log := AuditLog{Path: path}
	first := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)
	if err := log.Append(AuditEntry{Action: "read", RunID: "run-1", Actor: "auditor-1"}, first); err != nil {
		t.Fatal(err)
	}
	if err := log.Append(AuditEntry{Action: "retain", RunID: "run-1", Actor: "auditor-2", Detail: "kept"}, second); err != nil {
		t.Fatal(err)
	}
	entries, err := log.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries=%d", len(entries))
	}
	if entries[0].Action != "read" || entries[0].Actor != "auditor-1" || entries[0].Schema != AuditSchema {
		t.Fatalf("first entry=%+v", entries[0])
	}
	if entries[1].Action != "retain" || entries[1].Detail != "kept" {
		t.Fatalf("second entry=%+v", entries[1])
	}
	if entries[0].At != first.Format(time.RFC3339Nano) {
		t.Fatalf("first time=%q", entries[0].At)
	}
}

func TestAuditLogReadsAMissingFileAsEmpty(t *testing.T) {
	entries, err := (AuditLog{Path: filepath.Join(t.TempDir(), "missing.jsonl")}).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries=%d", len(entries))
	}
}

func TestAuditLogRejectsCorruptOrUnknownEntries(t *testing.T) {
	directory := t.TempDir()
	cases := map[string]string{
		"corrupt":        "{\"schema\":\"" + AuditSchema + "\"}\nnot json\n",
		"unknown schema": "{\"schema\":\"other\",\"at\":\"2026-11-06T00:00:00Z\",\"action\":\"read\"}\n",
		"missing field":  "{\"schema\":\"" + AuditSchema + "\",\"at\":\"2026-11-06T00:00:00Z\"}\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(directory, name+".jsonl")
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := (AuditLog{Path: path}).Read(); err == nil {
				t.Fatal("a corrupt audit log was accepted")
			}
		})
	}
}

func TestAuditLogAppendRequiresPathAndAction(t *testing.T) {
	now := time.Now().UTC()
	if err := (AuditLog{}).Append(AuditEntry{Action: "read"}, now); err == nil {
		t.Fatal("an audit entry without a path was accepted")
	}
	if err := (AuditLog{Path: filepath.Join(t.TempDir(), "a.jsonl")}).Append(AuditEntry{}, now); err == nil {
		t.Fatal("an audit entry without an action was accepted")
	}
}
