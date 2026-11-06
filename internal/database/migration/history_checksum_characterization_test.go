package migration_test

import (
	"crypto/sha256"
	"fmt"
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

// HIST-02 follow-up: Fingerprint deliberately excludes generated Realizations.
// A matching canonical-definition fingerprint is not an independent receipt
// of which Target SQL statements were generated or executed.
func TestHIST02CanonicalChecksumDoesNotCertifyDerivedRealizationBytes(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	first := migration.Definition{
		ID:       "m001",
		Database: db,
		Recovery: migration.RecoveryForwardFix,
		Steps:    []migration.Step{{Kind: migration.StepVerify, Description: "stable canonical definition"}},
		Realizations: []migration.Realization{{
			Statements: []string{"ALTER TABLE orders ADD COLUMN audited INTEGER"},
		}},
	}
	second := first
	second.Realizations = []migration.Realization{{
		Statements: []string{"ALTER TABLE orders ADD COLUMN audited TEXT"},
	}}
	firstDigest, err := migration.Fingerprint(first)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := migration.Fingerprint(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("fixture expects unchanged canonical checksum, got %q and %q", firstDigest, secondDigest)
	}

	history := migration.HistorySnapshot{
		Database: db,
		Tool:     migration.ToolAtlas,
		Entries: []migration.AppliedMigration{{
			Database: db,
			ID:       first.ID,
			State:    migration.AppliedSucceeded,
			Checksum: firstDigest,
			Tool:     migration.ToolAtlas,
		}},
	}
	for _, def := range []migration.Definition{first, second} {
		got, err := migration.CompareHistory(db, []migration.Definition{def}, history)
		if err != nil || len(got.Entries) != 1 || got.Entries[0].State != migration.RepoApplied {
			t.Fatalf("canonical checksum does not identify realized statements: %#v err=%v", got, err)
		}
	}
	// The source fixtures contain different generated SQL. Whether either
	// realization was actually executed is UNKNOWN, not implied by RepoApplied.
}

// HIST-02c: the repository and tool may hash different byte sequences.
// The tool digest in this fixture hashes the realized SQL bytes, while
// Fingerprint hashes the canonical Definition and excludes Realizations.
// This is a synthetic algorithm mismatch; no external tool was observed.
func TestHIST02DifferentDigestBasesLookLikeDefinitionDivergence(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	sql := "ALTER TABLE orders ADD COLUMN audited INTEGER;"
	def := migration.Definition{
		ID:           "m001",
		Database:     db,
		Recovery:     migration.RecoveryForwardFix,
		Steps:        []migration.Step{{Kind: migration.StepVerify, Description: "stable canonical definition"}},
		Realizations: []migration.Realization{{Statements: []string{sql}}},
	}
	canonicalDigest, err := migration.Fingerprint(def)
	if err != nil {
		t.Fatal(err)
	}
	toolDigest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(sql)))
	if toolDigest == canonicalDigest {
		t.Fatal("invalid fixture: distinct digest bases unexpectedly match")
	}

	history := migration.HistorySnapshot{
		Database: db,
		Tool:     migration.ToolAtlas,
		Entries: []migration.AppliedMigration{{
			Database: db,
			ID:       def.ID,
			State:    migration.AppliedSucceeded,
			Checksum: toolDigest,
			Tool:     migration.ToolAtlas,
		}},
	}
	got, err := migration.CompareHistory(db, []migration.Definition{def}, history)
	if err != nil || len(got.Entries) != 1 || got.Entries[0].State != migration.RepoDiverged {
		t.Fatalf("expected raw checksum comparison to say diverged: %#v err=%v", got, err)
	}
	// Source Definition is identical on both sides of this comparison.
	// A mismatch of algorithm / byte basis does not prove definition drift.
	// The independent Gate Expected for H1 is UNKNOWN until digest-basis
	// equivalence is established. H2/H3 also remain unqualified.
}
