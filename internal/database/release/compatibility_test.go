package release_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/release"
)

func TestCompatibilityEvaluatesMultipleDatabasesAsSet(t *testing.T) {
	app := application.Application{ID: core.MustCanonicalID("order_entry"), Databases: []application.DatabaseRequirement{
		{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("sales_core.order"), CompatibleMajor: 4}}},
		{Alias: core.LocalName("master"), TargetDatabase: core.MustCanonicalID("master_data"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("master_data.customer"), CompatibleMajor: 2}}},
	}}
	states := []release.StateSnapshot{
		{Database: core.MustCanonicalID("sales_core"), Source: release.StateFromRelease, DataReadiness: release.DataReady, Provides: []release.ProvidedContract{{ID: core.MustCanonicalID("sales_core.order"), Version: database.ContractVersion{Major: 4, Minor: 2}}}},
		{Database: core.MustCanonicalID("master_data"), Source: release.StateFromObserved, DataReadiness: release.DataReady, Provides: []release.ProvidedContract{{ID: core.MustCanonicalID("master_data.customer"), Version: database.ContractVersion{Major: 2, Minor: 4}}}},
	}
	got := release.CheckApplication(app, states)
	if got.Overall != release.Compatible || len(got.Checks) != 2 {
		t.Fatalf("got=%#v", got)
	}
	states[1].DataReadiness = release.DataPreparing
	got = release.CheckApplication(app, states)
	if got.Overall != release.Blocked {
		t.Fatalf("got=%#v", got)
	}
}

func TestCheckApplicationDoesNotTreatNextMajorAsCaretCompatible(t *testing.T) {
	app := application.Application{ID: core.MustCanonicalID("a"), Databases: []application.DatabaseRequirement{{Alias: core.LocalName("db"), TargetDatabase: core.MustCanonicalID("d"), RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("d.c"), CompatibleMajor: 4}}}}}
	states := []release.StateSnapshot{{Database: core.MustCanonicalID("d"), Source: release.StateFromRelease, Release: "5.0.0", DataReadiness: release.DataReady, Provides: []release.ProvidedContract{{ID: core.MustCanonicalID("d.c"), Version: database.ContractVersion{Major: 5}}}}}
	got := release.CheckApplication(app, states)
	if got.Overall != release.Blocked {
		t.Fatalf("expected next major to be blocked, got %+v", got)
	}
}
