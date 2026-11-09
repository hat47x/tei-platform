package runlog_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/runlog"
)

func TestRunKeySeparatesThePlanAndTheValues(t *testing.T) {
	first := runlog.RunKey("sales.monthly_close", []string{"2026-10"})
	second := runlog.RunKey("sales.monthly_close", []string{"2026-11"})
	other := runlog.RunKey("sales.other", []string{"2026-10"})
	if first == second || first == other {
		t.Fatalf("different runs must differ: %q %q %q", first, second, other)
	}
	forged := runlog.RunKey("sales.monthly_close", []string{"2026-10,\x1f2026-11"})
	if forged == runlog.RunKey("sales.monthly_close", []string{"2026-10", "2026-11"}) {
		t.Fatal("a value must not forge a different key")
	}
}

func TestDecideStartsARunThatNeverStarted(t *testing.T) {
	decision := runlog.Decide("none", runlog.Record{}, false)
	if decision.Action != runlog.Start || decision.Attempt != 1 || decision.FenceEpoch != 1 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestDecideResumesAnUnfinishedRunWithCheckpoints(t *testing.T) {
	last := runlog.Record{Attempt: 1, FenceEpoch: 1, Status: runlog.Running}
	decision := runlog.Decide("checkpoint", last, true)
	if decision.Action != runlog.Resume || decision.Attempt != 2 || decision.FenceEpoch != 2 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestDecideRefusesAnUnfinishedRunWithoutCheckpoints(t *testing.T) {
	last := runlog.Record{Attempt: 1, FenceEpoch: 1, Status: runlog.Running}
	if decision := runlog.Decide("none", last, true); decision.Action != runlog.Refuse {
		t.Fatalf("an unfinished run without checkpoints must be refused: %+v", decision)
	}
}

func TestDecideRefusesACompletedRun(t *testing.T) {
	last := runlog.Record{Attempt: 1, FenceEpoch: 1, Status: runlog.Completed}
	if decision := runlog.Decide("checkpoint", last, true); decision.Action != runlog.Refuse {
		t.Fatalf("a completed run must be refused even with checkpoints: %+v", decision)
	}
}

func TestOwnsRejectsAStaleFence(t *testing.T) {
	last := runlog.Record{Attempt: 2, FenceEpoch: 5}
	if runlog.Owns(last, 2, 4) {
		t.Fatal("a stale fence epoch must not own the run")
	}
	if !runlog.Owns(last, 2, 5) {
		t.Fatal("the current attempt and epoch own the run")
	}
}

func TestFileKeepsTheLatestRecordPerRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	journal := runlog.File{Path: path}
	key := runlog.RunKey("sales.monthly_close", []string{"2026-10"})
	for _, record := range []runlog.Record{
		{RunKey: key, PlanID: "sales.monthly_close", Attempt: 1, FenceEpoch: 1, Status: runlog.Running},
		{RunKey: runlog.RunKey("sales.other", []string{"2026-10"}), Attempt: 1, FenceEpoch: 1, Status: runlog.Running},
		{RunKey: key, PlanID: "sales.monthly_close", Attempt: 2, FenceEpoch: 2, Status: runlog.Completed},
	} {
		if err := journal.Append(record); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	last, found, err := journal.Last(key)
	if err != nil {
		t.Fatalf("last: %v", err)
	}
	if !found || last.Attempt != 2 || last.Status != runlog.Completed {
		t.Fatalf("unexpected last record: %+v found=%v", last, found)
	}
}

func TestFileReportsADamagedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	if err := os.WriteFile(path, []byte("{\"runKey\":\"a\",\"attempt\":1}\nbroken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	records, err := (runlog.File{Path: path}).Records()
	if err == nil {
		t.Fatal("a damaged line must be reported")
	}
	if len(records) != 1 {
		t.Fatalf("the records before the damage must be returned: %+v", records)
	}
}
