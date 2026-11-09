package expectedstateupdate

import (
	"testing"
	"time"
)

func TestRetentionKeepsNewestRunsAndDropsExpired(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	runs := []RetainedRun{
		{RunID: "old", RecordedAt: now.Add(-48 * time.Hour)},
		{RunID: "newest", RecordedAt: now.Add(-1 * time.Hour)},
		{RunID: "middle", RecordedAt: now.Add(-2 * time.Hour)},
		{RunID: "third", RecordedAt: now.Add(-3 * time.Hour)},
	}
	policy := RetentionPolicy{MaxRuns: 3, MaxAge: 24 * time.Hour}
	kept, err := policy.Retain(runs, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"newest", "middle", "third"}
	if len(kept) != len(want) {
		t.Fatalf("kept=%d, want %d", len(kept), len(want))
	}
	for i, id := range want {
		if kept[i].RunID != id {
			t.Fatalf("kept[%d]=%s, want %s", i, kept[i].RunID, id)
		}
	}
}

func TestRetentionAppliesTheRunLimitAfterAge(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	runs := []RetainedRun{
		{RunID: "a", RecordedAt: now.Add(-1 * time.Hour)},
		{RunID: "b", RecordedAt: now.Add(-2 * time.Hour)},
		{RunID: "c", RecordedAt: now.Add(-3 * time.Hour)},
	}
	policy := RetentionPolicy{MaxRuns: 2, MaxAge: 10 * time.Hour}
	kept, err := policy.Retain(runs, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 2 || kept[0].RunID != "a" || kept[1].RunID != "b" {
		t.Fatalf("kept=%+v", kept)
	}
}

func TestRetentionRejectsInvalidPolicyOrUnknownTime(t *testing.T) {
	now := time.Now().UTC()
	for name, policy := range map[string]RetentionPolicy{
		"no run limit": {MaxRuns: 0, MaxAge: time.Hour},
		"no age limit": {MaxRuns: 1, MaxAge: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := policy.Retain(nil, now); err == nil {
				t.Fatal("an incomplete policy was accepted")
			}
		})
	}
	policy := RetentionPolicy{MaxRuns: 1, MaxAge: time.Hour}
	if _, err := policy.Retain([]RetainedRun{{RunID: "missing-time"}}, now); err == nil {
		t.Fatal("a run without a recorded time was accepted")
	}
}
