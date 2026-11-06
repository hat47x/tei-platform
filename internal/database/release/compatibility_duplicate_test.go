package release_test

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/release"
)

func mg29Application(requiredMajor int) application.Application {
	return application.Application{
		ID: core.MustCanonicalID("order_entry"),
		Databases: []application.DatabaseRequirement{{
			Alias:          core.LocalName("sales"),
			TargetDatabase: core.MustCanonicalID("sales_core"),
			Access:         database.AccessReadWrite,
			RequiredContracts: []database.ContractRequirement{{
				ContractID:      core.MustCanonicalID("sales_core.order"),
				CompatibleMajor: requiredMajor,
			}},
		}},
	}
}

func mg29State(providedMajor int) release.StateSnapshot {
	return release.StateSnapshot{
		Database:      core.MustCanonicalID("sales_core"),
		Source:        release.StateFromObserved,
		DataReadiness: release.DataReady,
		Provides: []release.ProvidedContract{{
			ID:      core.MustCanonicalID("sales_core.order"),
			Version: database.ContractVersion{Major: providedMajor},
		}},
	}
}

// MG-29a: the same observations must not change the answer when reordered.
func TestMG29RejectsDuplicateDatabaseSnapshotsRegardlessOfOrder(t *testing.T) {
	app := mg29Application(5)
	oldState, newState := mg29State(4), mg29State(5)
	oldState.ObservedAt = "2026-11-05T09:00:00Z"
	newState.ObservedAt = "2026-11-05T10:00:00Z"
	for _, tc := range []struct {
		name   string
		states []release.StateSnapshot
	}{
		{"old-then-new", []release.StateSnapshot{oldState, newState}},
		{"new-then-old", []release.StateSnapshot{newState, oldState}},
		{"identical-duplicates", []release.StateSnapshot{newState, newState}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := release.CheckApplication(app, tc.states)
			if got.Overall != release.Unknown || len(got.Checks) != 1 || got.Checks[0].State != release.Unknown {
				t.Fatalf("ambiguous database state must fail closed: %#v", got)
			}
			if !strings.Contains(got.Checks[0].Reason, "multiple database state") {
				t.Fatalf("missing diagnostic for ambiguous database state: %#v", got.Checks[0])
			}
		})
	}
}

// MG-29b: a single snapshot must not silently choose the last contract version.
func TestMG29RejectsDuplicateProvidedContractsRegardlessOfOrder(t *testing.T) {
	app := mg29Application(5)
	oldContract := release.ProvidedContract{
		ID:      core.MustCanonicalID("sales_core.order"),
		Version: database.ContractVersion{Major: 4},
	}
	newContract := release.ProvidedContract{
		ID:      core.MustCanonicalID("sales_core.order"),
		Version: database.ContractVersion{Major: 5},
	}
	for _, tc := range []struct {
		name     string
		provides []release.ProvidedContract
	}{
		{"old-then-new", []release.ProvidedContract{oldContract, newContract}},
		{"new-then-old", []release.ProvidedContract{newContract, oldContract}},
		{"identical-duplicates", []release.ProvidedContract{newContract, newContract}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := mg29State(5)
			state.Provides = tc.provides
			got := release.CheckApplication(app, []release.StateSnapshot{state})
			if got.Overall != release.Unknown || len(got.Checks) != 1 || got.Checks[0].State != release.Unknown {
				t.Fatalf("ambiguous provided contract must fail closed: %#v", got)
			}
			if !strings.Contains(got.Checks[0].Reason, "multiple provided contracts") {
				t.Fatalf("missing diagnostic for duplicate provided contract: %#v", got.Checks[0])
			}
		})
	}
}

// Duplicate evidence for one DB must not overwrite an independent DB's result.
func TestMG29DuplicateDatabaseDoesNotCorruptOtherDatabaseCheck(t *testing.T) {
	app := mg29Application(5)
	app.Databases = append(app.Databases, application.DatabaseRequirement{
		Alias:          core.LocalName("master"),
		TargetDatabase: core.MustCanonicalID("master_data"),
		Access:         database.AccessRead,
		RequiredContracts: []database.ContractRequirement{{
			ContractID:      core.MustCanonicalID("master_data.customer"),
			CompatibleMajor: 2,
		}},
	})
	master := release.StateSnapshot{
		Database:      core.MustCanonicalID("master_data"),
		Source:        release.StateFromRelease,
		DataReadiness: release.DataReady,
		Provides: []release.ProvidedContract{{
			ID:      core.MustCanonicalID("master_data.customer"),
			Version: database.ContractVersion{Major: 2},
		}},
	}
	got := release.CheckApplication(app, []release.StateSnapshot{mg29State(4), master, mg29State(5)})
	if got.Overall != release.Unknown || len(got.Checks) != 2 {
		t.Fatalf("unexpected overall/number of checks: %#v", got)
	}
	states := map[core.CanonicalID]release.CompatibilityState{}
	for _, check := range got.Checks {
		states[check.Database] = check.State
	}
	if states[core.MustCanonicalID("sales_core")] != release.Unknown || states[core.MustCanonicalID("master_data")] != release.Compatible {
		t.Fatalf("duplicate DB affected independent database: %#v", got.Checks)
	}
}

func TestMG29SingleStatePreservesPriorCompatibilityRules(t *testing.T) {
	app := mg29Application(5)
	if got := release.CheckApplication(app, []release.StateSnapshot{mg29State(5)}); got.Overall != release.Compatible {
		t.Fatalf("single matching state should be compatible: %#v", got)
	}
	if got := release.CheckApplication(app, []release.StateSnapshot{mg29State(4)}); got.Overall != release.Blocked {
		t.Fatalf("different major version should be blocked: %#v", got)
	}
	if got := release.CheckApplication(app, nil); got.Overall != release.Unknown {
		t.Fatalf("missing required database should be unknown: %#v", got)
	}
	pending := mg29State(5)
	pending.DataReadiness = release.DataPreparing
	if got := release.CheckApplication(app, []release.StateSnapshot{pending}); got.Overall != release.Blocked {
		t.Fatalf("data not ready should be blocked: %#v", got)
	}
}
