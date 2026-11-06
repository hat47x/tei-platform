package changeanalysis

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/releaseplan"
)

func TestGateCandidatesOnlyUnreferencedAndUnsatisfied(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4}},
				{ID: "sales.other", Version: database.ContractVersion{Major: 2}},
			}},
		},
		ApplicationReleases: []apprelease.Definition{{
			Application: "app", Version: "1.0.0",
			Databases: []application.DatabaseRequirement{{
				Alias: "db", TargetDatabase: "sales",
				RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 4}},
			}},
		}},
		DatabaseReleases: []dbrelease.Definition{
			{Database: "sales", Version: "1.0.0", Provides: []dbrelease.ProvidedContract{{ID: "sales.legacy", Version: database.ContractVersion{Major: 1}}}},
		},
		SolutionReleases: []releaseplan.Definition{{
			ID: "rollout", Version: "2026.1.1",
			Steps: []releaseplan.Step{
				{ID: "referenced_but_missing", Kind: releaseplan.StepDatabaseGate, Database: "sales", Contract: "sales.order", CompatibleMajor: 4},
				{ID: "unreferenced_current", Kind: releaseplan.StepDatabaseGate, Database: "sales", Contract: "sales.other", CompatibleMajor: 2},
				{ID: "unreferenced_release", Kind: releaseplan.StepDatabaseGate, Database: "sales", Contract: "sales.legacy", CompatibleMajor: 1},
				{ID: "unreferenced_missing", Kind: releaseplan.StepDatabaseGate, Database: "sales", Contract: "sales.missing", CompatibleMajor: 9},
				{ID: "unreferenced_wrong_major", Kind: releaseplan.StepDatabaseGate, Database: "sales", Contract: "sales.legacy", CompatibleMajor: 2},
				{ID: "wrong_database", Kind: releaseplan.StepDatabaseGate, Database: "other", Contract: "sales.legacy", CompatibleMajor: 1},
			},
		}},
	}
	got := GateCandidates(world, "after")
	byStep := map[string]UnreferencedUnsatisfiableGate{}
	for _, gate := range got {
		byStep[string(gate.Step)] = gate
	}
	if len(got) != 3 {
		t.Fatalf("unexpected gate candidates: %+v", got)
	}
	for _, step := range []string{"unreferenced_missing", "unreferenced_wrong_major", "wrong_database"} {
		if _, ok := byStep[step]; !ok {
			t.Fatalf("unsatisfiable gate %s missing: %+v", step, got)
		}
	}
	for _, step := range []string{"referenced_but_missing", "unreferenced_current", "unreferenced_release"} {
		if _, ok := byStep[step]; ok {
			t.Fatalf("excluded gate %s was reported: %+v", step, got)
		}
	}
	if byStep["unreferenced_wrong_major"].Snapshot != "after" || byStep["unreferenced_missing"].CompatibleMajor != 9 {
		t.Fatalf("candidate fields lost: %+v", got)
	}
	if again := GateCandidates(world, "before"); len(again) != len(got) || again[0].Snapshot != "before" {
		t.Fatalf("snapshot is not carried: %+v", again)
	}
}
