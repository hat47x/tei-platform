package release_test

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
)

// MG-21 structural characterization only: this checks immutable release
// requirements against a provided-contract snapshot. It does not execute a
// database read/write, inspect a live client, or qualify a migration gate.
func TestMG21OldAndNewApplicationReleasesDoNotSilentlyCrossMajor(t *testing.T) {
	dbID := core.MustCanonicalID("sales_core")
	contractID := core.MustCanonicalID("sales_core.order")
	appID := core.MustCanonicalID("order_entry")

	old := apprelease.Definition{
		Application: appID,
		Version:     "1.0.0",
		Databases: []application.DatabaseRequirement{{
			Alias:          core.LocalName("sales"),
			TargetDatabase: dbID,
			Access:         database.AccessReadWrite,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 4,
			}},
		}},
	}
	newer := apprelease.Definition{
		Application: appID,
		Version:     "2.0.0",
		Databases: []application.DatabaseRequirement{{
			Alias:          core.LocalName("sales"),
			TargetDatabase: dbID,
			Access:         database.AccessReadWrite,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: contractID, CompatibleMajor: 5,
			}},
		}},
	}

	oldDB := dbrelease.StateSnapshot{
		Database: dbID, Source: dbrelease.StateFromRelease, Release: "1.8.0",
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 4, Minor: 9},
		}},
	}
	newDB := dbrelease.StateSnapshot{
		Database: dbID, Source: dbrelease.StateFromRelease, Release: "2.0.0",
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 5},
		}},
	}

	for _, tc := range []struct {
		name     string
		app      apprelease.Definition
		state    dbrelease.StateSnapshot
		expected dbrelease.CompatibilityState
	}{
		{"old-app-old-db", old, oldDB, dbrelease.Compatible},
		{"new-app-old-db", newer, oldDB, dbrelease.Blocked},
		{"old-app-new-db", old, newDB, dbrelease.Blocked},
		{"new-app-new-db", newer, newDB, dbrelease.Compatible},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := dbrelease.CheckApplication(tc.app.AsApplication(), []dbrelease.StateSnapshot{tc.state})
			if got.Overall != tc.expected || len(got.Checks) != 1 {
				t.Fatalf("overall=%s checks=%#v, want %s and exactly one check", got.Overall, got.Checks, tc.expected)
			}
			if got.Checks[0].Contract != contractID {
				t.Fatalf("contract changed: %#v", got.Checks[0])
			}
		})
	}
	if old.Databases[0].RequiredContracts[0].CompatibleMajor != 4 {
		t.Fatal("new application release changed old immutable requirement")
	}
}

// MG-21 gap characterization: CheckApplication currently checks declared
// version/readiness, not whether the real Provider can read or write.
// Both variants can be structurally compatible even when the required
// independent write-effect evidence is absent.
func TestMG21CompatibilityDoesNotDistinguishReadFromWriteEffect(t *testing.T) {
	dbID := core.MustCanonicalID("sales_core")
	contractID := core.MustCanonicalID("sales_core.order")
	state := dbrelease.StateSnapshot{
		Database: dbID, Source: dbrelease.StateFromObserved,
		DataReadiness: dbrelease.DataReady,
		Provides: []dbrelease.ProvidedContract{{
			ID: contractID, Version: database.ContractVersion{Major: 4},
		}},
	}
	for _, access := range []database.Access{database.AccessRead, database.AccessWrite} {
		t.Run(string(access), func(t *testing.T) {
			app := application.Application{
				ID: core.MustCanonicalID("order_entry"),
				Databases: []application.DatabaseRequirement{{
					Alias:          core.LocalName("sales"),
					TargetDatabase: dbID,
					Access:         access,
					RequiredContracts: []database.ContractRequirement{{
						ContractID: contractID, CompatibleMajor: 4,
					}},
				}},
			}
			got := dbrelease.CheckApplication(app, []dbrelease.StateSnapshot{state})
			if got.Overall != dbrelease.Compatible {
				t.Fatalf("declared contract should be structurally compatible, got %#v", got)
			}
		})
	}
	// This is intentionally NOT a proof of read success, write effect,
	// snapshot freshness, client inventory, or owner/host authorization.
}
