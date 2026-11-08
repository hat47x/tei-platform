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

func TestCompareHistoryRejectsUnconfirmedStates(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	m := migration.Definition{
		ID: "001", Database: db, Recovery: migration.RecoveryForwardFix,
		Steps: []migration.Step{{Kind: migration.StepVerify, Description: "v1"}},
	}
	fp, err := migration.Fingerprint(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		state migration.AppliedState
	}{
		{name: "unknown", state: migration.AppliedUnknown},
		{name: "omitted", state: ""},
		{name: "unrecognized", state: migration.AppliedState("unexpected")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := migration.HistorySnapshot{
				Database: db, Tool: migration.ToolAtlas,
				Entries: []migration.AppliedMigration{{
					Database: db, ID: "001", State: tc.state,
					Checksum: fp, Tool: migration.ToolAtlas,
				}},
			}
			if got, err := migration.CompareHistory(db, []migration.Definition{m}, h); err == nil {
				t.Fatalf("unconfirmed history state %q was accepted: %#v", tc.state, got)
			}
		})
	}
}

func TestCompareHistoryStillReportsConfirmedFailure(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	m := migration.Definition{
		ID: "001", Database: db, Recovery: migration.RecoveryForwardFix,
		Steps: []migration.Step{{Kind: migration.StepVerify, Description: "v1"}},
	}
	fp, err := migration.Fingerprint(m)
	if err != nil {
		t.Fatal(err)
	}
	h := migration.HistorySnapshot{
		Database: db, Tool: migration.ToolAtlas,
		Entries: []migration.AppliedMigration{{
			Database: db, ID: "001", State: migration.AppliedFailed,
			Checksum: fp, Tool: migration.ToolAtlas,
		}},
	}
	got, err := migration.CompareHistory(db, []migration.Definition{m}, h)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].State != migration.RepoFailed {
		t.Fatalf("confirmed failure must remain failed: %#v", got.Entries)
	}
}

func TestCompareHistoryRejectsUnknownOrphan(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	h := migration.HistorySnapshot{
		Database: db, Tool: migration.ToolAtlas,
		Entries: []migration.AppliedMigration{{
			Database: db, ID: "orphan-001",
			State: migration.AppliedUnknown, Tool: migration.ToolAtlas,
		}},
	}
	if got, err := migration.CompareHistory(db, nil, h); err == nil {
		t.Fatalf("orphan with unknown application status was certified: %#v", got.Entries)
	}
	h.Entries[0].State = migration.AppliedSucceeded
	got, err := migration.CompareHistory(db, nil, h)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 || got.Entries[0].State != migration.RepoOrphanApplied {
		t.Fatalf("confirmed orphan must retain its classification: %#v", got.Entries)
	}
}
