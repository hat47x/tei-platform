package migration_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
)

func TestCompareHistorySeparatesPendingDivergedAndOrphan(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	m1 := migration.Definition{ID: "001", Database: db, Recovery: migration.RecoveryForwardFix, Steps: []migration.Step{{Kind: migration.StepVerify, Description: "v1"}}}
	m2 := migration.Definition{ID: "002", Database: db, Recovery: migration.RecoveryForwardFix, Steps: []migration.Step{{Kind: migration.StepVerify, Description: "v2"}}}
	fp1, _ := migration.Fingerprint(m1)
	h := migration.HistorySnapshot{Database: db, Tool: migration.ToolAtlas, Entries: []migration.AppliedMigration{
		{Database: db, ID: "001", State: migration.AppliedSucceeded, Checksum: fp1, Tool: migration.ToolAtlas},
		{Database: db, ID: "003", State: migration.AppliedSucceeded, Checksum: "sha256:orphan", Tool: migration.ToolAtlas},
	}}
	got, err := migration.CompareHistory(db, []migration.Definition{m1, m2}, h)
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]migration.RepositoryState{}
	for _, e := range got.Entries {
		states[e.ID] = e.State
	}
	if states["001"] != migration.RepoApplied || states["002"] != migration.RepoPending || states["003"] != migration.RepoOrphanApplied {
		t.Fatalf("states=%#v", states)
	}

	h.Entries[0].Checksum = "sha256:edited-after-apply"
	got, err = migration.CompareHistory(db, []migration.Definition{m1, m2}, h)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entries[0].State != migration.RepoDiverged {
		t.Fatalf("entry=%#v", got.Entries[0])
	}
}

func TestFingerprintIgnoresDerivedRealizations(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	a := migration.Definition{ID: "001", Database: db, Recovery: migration.RecoveryForwardFix, Steps: []migration.Step{{Kind: migration.StepVerify, Description: "same"}}}
	b := a
	b.Realizations = []migration.Realization{{Statements: []string{"generated sql changed"}}}
	fa, _ := migration.Fingerprint(a)
	fb, _ := migration.Fingerprint(b)
	if fa != fb {
		t.Fatalf("derived realization changed fingerprint: %s != %s", fa, fb)
	}
}
