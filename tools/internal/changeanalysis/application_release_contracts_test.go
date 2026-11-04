package changeanalysis

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/database"
	"tei/internal/ir"
)

func TestApplicationReleaseContractCandidatesOnlyMismatches(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 2}},
				{ID: "sales.line", Version: database.ContractVersion{Major: 1, Minor: 0}},
			}},
		},
		ApplicationReleases: []apprelease.Definition{
			{
				Application: "matched", Version: "1.0.0",
				Databases: []application.DatabaseRequirement{{Alias: "db", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 4}}}},
			},
			{
				Application: "behind", Version: "2.0.0",
				Databases: []application.DatabaseRequirement{{Alias: "line", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.line", CompatibleMajor: 2}}}},
			},
			{
				Application: "missing", Version: "3.0.0",
				Databases: []application.DatabaseRequirement{{Alias: "legacy", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.legacy", CompatibleMajor: 1}}}},
			},
		},
	}
	got := ApplicationReleaseContractCandidates(world, "after")
	if len(got) != 2 {
		t.Fatalf("unexpected candidates: %+v", got)
	}
	if got[0].Application != "behind" || got[0].Reason != ApplicationReleaseContractMajor || got[0].CurrentVersion != "1.0" || got[0].CompatibleMajor != 2 {
		t.Fatalf("major mismatch lost: %+v", got[0])
	}
	if got[1].Application != "missing" || got[1].Reason != ApplicationReleaseContractNotCurrent || got[1].CurrentVersion != "" {
		t.Fatalf("missing contract lost: %+v", got[1])
	}
	for _, c := range got {
		if c.Snapshot != "after" {
			t.Fatalf("snapshot not carried: %+v", c)
		}
	}
	if before := ApplicationReleaseContractCandidates(world, "before"); len(before) != 2 || before[0].Snapshot != "before" {
		t.Fatalf("snapshot scan is not independent: %+v", before)
	}
}
