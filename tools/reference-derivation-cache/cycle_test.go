package derivationcache

import (
	"testing"
)

func TestReuseCycleRejectsABAHistory(t *testing.T) {
	report := ReuseCycle(CycleInput{
		CacheRequested: true,
		CachedSource:   "sha256:a",
		CurrentSource:  "sha256:a",
		SourceHistory:  []string{"sha256:a", "sha256:b", "sha256:a"},
	})
	if report.Accepted || report.Code != CodeSnapshotIntegrityUnproven || !report.ValueNull || !report.ABA {
		t.Fatalf("report=%+v want SNAPSHOT_INTEGRITY_UNPROVEN with ABA=true", report)
	}
}

func TestReuseCycleAllowsIdenticalHistory(t *testing.T) {
	report := ReuseCycle(CycleInput{
		CacheRequested: true,
		CachedSource:   "sha256:a",
		CurrentSource:  "sha256:a",
		SourceHistory:  []string{"sha256:a", "sha256:a"},
	})
	if !report.Accepted || report.Code != CodeCacheReused || report.ValueNull || report.ABA {
		t.Fatalf("report=%+v want CACHE_REUSED without ABA", report)
	}
}

func TestReuseCycleRejectsSourceChange(t *testing.T) {
	report := ReuseCycle(CycleInput{
		CacheRequested: true,
		CachedSource:   "sha256:a",
		CurrentSource:  "sha256:b",
	})
	if report.Accepted || report.Code != CodeStaleSourceTime || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_SOURCE_TIME with unknown value", report)
	}
}
