package access_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

type memoryJournal struct {
	records []access.Record
	err     error
}

func (j *memoryJournal) Append(record access.Record) error {
	if j.err != nil {
		return j.err
	}
	j.records = append(j.records, record)
	return nil
}

func TestCheckAllowsCoveredRequirements(t *testing.T) {
	journal := &memoryJournal{}
	guard := access.Guard{Granted: []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "/var"}}, Journal: journal}
	decision, err := guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/reports"}})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !decision.Allowed || len(decision.Missing) != 0 {
		t.Fatalf("the requirement is covered: %+v", decision)
	}
	if len(journal.records) != 1 || !journal.records[0].Allowed {
		t.Fatalf("the decision must be recorded: %+v", journal.records)
	}
}

func TestCheckRefusesAndNamesWhatIsMissing(t *testing.T) {
	guard := access.Guard{}
	decision, err := guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if decision.Allowed || len(decision.Missing) != 1 || decision.Missing[0] != "network:connect db:5432" {
		t.Fatalf("the missing requirement must be named: %+v", decision)
	}
}

func TestCheckWithoutAJournalStillDecides(t *testing.T) {
	guard := access.Guard{}
	if decision, err := guard.Check(core.MustCanonicalID("sales.close"), nil); err != nil || !decision.Allowed {
		t.Fatalf("no requirements means allowed: %+v err=%v", decision, err)
	}
}

func TestCheckReportsAJournalFailure(t *testing.T) {
	guard := access.Guard{Journal: &memoryJournal{err: os.ErrPermission}}
	if _, err := guard.Check(core.MustCanonicalID("sales.close"), nil); err == nil {
		t.Fatal("an unrecorded decision must be reported")
	}
}

func TestFileKeepsEveryDecision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.jsonl")
	guard := access.Guard{Granted: nil, Journal: access.File{Path: path}}
	if _, err := guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}}); err != nil {
		t.Fatalf("check: %v", err)
	}
	records, err := (access.File{Path: path}).Records()
	if err != nil {
		t.Fatalf("records: %v", err)
	}
	if len(records) != 1 || records[0].Allowed || records[0].Capability != "sales.close" {
		t.Fatalf("unexpected records: %+v", records)
	}
}
