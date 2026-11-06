package migration_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
)

// HIST-02 characterizes a content-comparison ceiling, not a safe migration
// Gate. An applied record without a checksum cannot distinguish an old
// repository definition from a changed definition with the same migration ID.
func TestHIST02MissingAppliedChecksumCannotDetectDefinitionDrift(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	original := migration.Definition{
		ID:       "m001",
		Database: db,
		Recovery: migration.RecoveryForwardFix,
		Steps:    []migration.Step{{Kind: migration.StepVerify, Description: "expected-before"}},
	}
	changed := original
	changed.Steps = []migration.Step{{Kind: migration.StepVerify, Description: "expected-after"}}

	oldChecksum, err := migration.Fingerprint(original)
	if err != nil {
		t.Fatal(err)
	}
	newChecksum, err := migration.Fingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if oldChecksum == newChecksum {
		t.Fatal("invalid fixture: authored definitions must differ")
	}

	record := migration.AppliedMigration{
		Database: db,
		ID:       original.ID,
		State:    migration.AppliedSucceeded,
		Tool:     migration.ToolAtlas,
	}
	snapshot := migration.HistorySnapshot{
		Database: db,
		Tool:     migration.ToolAtlas,
		Entries:  []migration.AppliedMigration{record},
	}

	// No provider checksum: both authored versions are currently called applied.
	for _, def := range []migration.Definition{original, changed} {
		got, err := migration.CompareHistory(db, []migration.Definition{def}, snapshot)
		if err != nil || len(got.Entries) != 1 ||
			got.Entries[0].State != migration.RepoApplied ||
			got.Entries[0].AppliedChecksum != "" {
			t.Fatalf("expected to characterize unverified applied state: %#v err=%v", got, err)
		}
	}

	// With a genuine, independently supplied old-definition checksum, the
	// changed definition can be distinguished by the content comparator.
	snapshot.Entries[0].Checksum = oldChecksum
	got, err := migration.CompareHistory(db, []migration.Definition{changed}, snapshot)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].State != migration.RepoDiverged {
		t.Fatalf("expected changed definition to diverge with old applied checksum: %#v err=%v", got, err)
	}
	snapshot.Entries[0].Checksum = newChecksum
	got, err = migration.CompareHistory(db, []migration.Definition{changed}, snapshot)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].State != migration.RepoApplied {
		t.Fatalf("expected matching applied checksum to be structurally applied: %#v err=%v", got, err)
	}

	// Independent Gate Expected for the no-checksum case:
	// definition equivalence = UNKNOWN, despite RepoApplied classification.
	// Actual Schema/History Reader qualification and Owner meaning are separate.
}
