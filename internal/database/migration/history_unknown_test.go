package migration_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
)

func historyUnknownFixture(t *testing.T) (core.CanonicalID, migration.Definition, string) {
	t.Helper()
	db := core.MustCanonicalID("sales_core")
	def := migration.Definition{
		ID: "001", Database: db,
		Recovery: migration.RecoveryForwardFix,
		Steps:    []migration.Step{{Kind: migration.StepVerify, Description: "independent expected"}},
	}
	checksum, err := migration.Fingerprint(def)
	if err != nil {
		t.Fatal(err)
	}
	return db, def, checksum
}

// HIST-01: unknown is not a successful apply even when the checksum matches.
func TestCompareHistoryRejectsUnknownWithMatchingChecksum(t *testing.T) {
	db, def, checksum := historyUnknownFixture(t)
	history := migration.HistorySnapshot{
		Database: db, Tool: migration.ToolAtlas,
		Entries: []migration.AppliedMigration{{
			Database: db, ID: def.ID, State: migration.AppliedUnknown,
			Checksum: checksum, Tool: migration.ToolAtlas,
		}},
	}
	got, err := migration.CompareHistory(db, []migration.Definition{def}, history)
	if err == nil || !strings.Contains(err.Error(), "unqualified applied state") {
		t.Fatalf("unknown applied state was not rejected: result=%#v err=%v", got, err)
	}
	if len(got.Entries) != 0 {
		t.Fatalf("failure must not return partially qualified history: %#v", got)
	}
}

// An unknown migration absent from repository must not be called orphan-applied.
func TestCompareHistoryRejectsUnknownOrphan(t *testing.T) {
	db, def, _ := historyUnknownFixture(t)
	history := migration.HistorySnapshot{
		Database: db, Tool: migration.ToolAtlas,
		Entries: []migration.AppliedMigration{{
			Database: db, ID: "orphan", State: migration.AppliedUnknown,
			Tool: migration.ToolAtlas,
		}},
	}
	got, err := migration.CompareHistory(db, []migration.Definition{def}, history)
	if err == nil || !strings.Contains(err.Error(), "unqualified applied state") {
		t.Fatalf("unknown orphan was not rejected: result=%#v err=%v", got, err)
	}
}

func TestCompareHistoryRejectsUnrecognizedStates(t *testing.T) {
	db, def, checksum := historyUnknownFixture(t)
	for _, state := range []migration.AppliedState{"", "committed?", "partial"} {
		t.Run("state-"+string(state), func(t *testing.T) {
			history := migration.HistorySnapshot{
				Database: db, Tool: migration.ToolAtlas,
				Entries: []migration.AppliedMigration{{
					Database: db, ID: def.ID, State: state,
					Checksum: checksum, Tool: migration.ToolAtlas,
				}},
			}
			if _, err := migration.CompareHistory(db, []migration.Definition{def}, history); err == nil {
				t.Fatalf("unrecognized applied state %q must not be successful", state)
			}
		})
	}
}

// HIST-03 boundary: only a genuinely complete empty history is pending.
// Fetch errors and partial pages cannot be represented by this input alone;
// the adapter must not fabricate an empty HistorySnapshot for those failures.
func TestCompareHistoryCompleteEmptyHistoryRemainsPending(t *testing.T) {
	db, def, _ := historyUnknownFixture(t)
	history := migration.HistorySnapshot{Database: db, Tool: migration.ToolAtlas, Entries: nil}
	got, err := migration.CompareHistory(db, []migration.Definition{def}, history)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].State != migration.RepoPending {
		t.Fatalf("complete empty history must remain pending: %#v", got)
	}
}

// HIST-04: recognized failed and succeeded histories retain their old classification.
func TestCompareHistoryPreservesKnownAppliedAndFailed(t *testing.T) {
	db, def, checksum := historyUnknownFixture(t)
	for _, tc := range []struct {
		state migration.AppliedState
		want  migration.RepositoryState
	}{
		{migration.AppliedSucceeded, migration.RepoApplied},
		{migration.AppliedFailed, migration.RepoFailed},
	} {
		t.Run(string(tc.state), func(t *testing.T) {
			history := migration.HistorySnapshot{
				Database: db, Tool: migration.ToolAtlas,
				Entries: []migration.AppliedMigration{{
					Database: db, ID: def.ID, State: tc.state,
					Checksum: checksum, Tool: migration.ToolAtlas,
				}},
			}
			got, err := migration.CompareHistory(db, []migration.Definition{def}, history)
			if err != nil || len(got.Entries) != 1 || got.Entries[0].State != tc.want {
				t.Fatalf("existing classification regressed: result=%#v err=%v", got, err)
			}
		})
	}
}
