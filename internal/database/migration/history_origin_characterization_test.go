package migration_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
)

// HIST-03d: CompareHistory classifies repository definitions against the
// entries it receives. It does NOT certify which actual DB, reader, tool
// instance, capture time, or observation scope produced those entries.
// This test characterizes that boundary; RepoApplied here is NOT a Gate pass.
func TestHIST03dCompareHistoryCannotQualifyObservationOrigin(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	def := migration.Definition{
		ID:       "m001",
		Database: db,
		Recovery: migration.RecoveryForwardFix,
		Steps:    []migration.Step{{Kind: migration.StepVerify, Description: "fixed fixture"}},
	}
	checksum, err := migration.Fingerprint(def)
	if err != nil {
		t.Fatal(err)
	}
	base := migration.HistorySnapshot{
		Database:   db,
		Tool:       migration.ToolAtlas,
		CapturedAt: "2026-11-06T00:00:00Z",
		Entries: []migration.AppliedMigration{{
			Database: db,
			ID:       def.ID,
			State:    migration.AppliedSucceeded,
			Checksum: checksum,
			Tool:     migration.ToolAtlas,
		}},
	}
	cases := []struct {
		name   string
		change func(*migration.HistorySnapshot)
	}{
		{"no-database-or-tool-source", func(h *migration.HistorySnapshot) {
			h.Database = ""
			h.Tool = ""
			h.CapturedAt = ""
			h.Entries[0].Database = ""
			h.Entries[0].Tool = ""
		}},
		{"no-capture-time", func(h *migration.HistorySnapshot) {
			h.CapturedAt = ""
		}},
		{"entry-tool-not-qualified", func(h *migration.HistorySnapshot) {
			h.Entries[0].Tool = ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := base
			h.Entries = append([]migration.AppliedMigration(nil), base.Entries...)
			tc.change(&h)
			got, err := migration.CompareHistory(db, []migration.Definition{def}, h)
			if err != nil || len(got.Entries) != 1 || got.Entries[0].State != migration.RepoApplied {
				t.Fatalf("expected current content-only classification, got=%#v err=%v", got, err)
			}
			// The independent expected Gate result is UNKNOWN until the
			// source/target/reader, completeness, and freshness are qualified
			// outside CompareHistory. That Gate is not implemented here.
		})
	}
}

// HIST-03c: finishing every page of a filtered query does not establish
// completeness of the required History scope. An applied orphan hidden by a
// "repository IDs only" filter is invisible to CompareHistory. The independent
// Gate expected for the filtered read is UNKNOWN, not "all applied".
func TestHIST03cFilteredHistoryHidesOrphanDespiteCompletePages(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	def := migration.Definition{
		ID:       "m001",
		Database: db,
		Recovery: migration.RecoveryForwardFix,
		Steps:    []migration.Step{{Kind: migration.StepVerify, Description: "fixed source fixture"}},
	}
	checksum, err := migration.Fingerprint(def)
	if err != nil {
		t.Fatal(err)
	}
	// Fixed source truth, independent of the comparator: both m001 and the
	// unexpected m999 were applied in the same target database.
	source := []migration.AppliedMigration{
		{Database: db, ID: "m001", State: migration.AppliedSucceeded, Checksum: checksum, Tool: migration.ToolAtlas},
		{Database: db, ID: "m999", State: migration.AppliedSucceeded, Checksum: "sha256:other", Tool: migration.ToolAtlas},
	}
	full := migration.HistorySnapshot{
		Database: db, Tool: migration.ToolAtlas,
		CapturedAt: "2026-11-06T00:00:00Z",
		Entries:    source,
	}
	// The reader used an unjustified filter: ID IN (repository IDs).
	filtered := full
	filtered.Entries = append([]migration.AppliedMigration(nil), source[:1]...)

	all, err := migration.CompareHistory(db, []migration.Definition{def}, full)
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Entries) != 2 || all.Entries[0].State != migration.RepoApplied ||
		all.Entries[1].State != migration.RepoOrphanApplied {
		t.Fatalf("full source did not reveal unexpected applied migration: %#v", all)
	}
	partial, err := migration.CompareHistory(db, []migration.Definition{def}, filtered)
	if err != nil {
		t.Fatal(err)
	}
	if len(partial.Entries) != 1 || partial.Entries[0].State != migration.RepoApplied {
		t.Fatalf("expected content-only comparator to be blind to omitted orphan: %#v", partial)
	}
	// Both calls can be "all pages retrieved" within their own filters.
	// This test characterizes the information loss, not a validated Reader.
}
