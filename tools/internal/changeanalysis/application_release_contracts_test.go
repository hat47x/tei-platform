package changeanalysis

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/database"
	"tei/internal/ir"
)

func TestApplicationReleaseContractCandidatesOnlyMajorMismatch(t *testing.T) {
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
	if len(got) != 1 {
		t.Fatalf("major不一致だけを期待: %+v", got)
	}
	if got[0].Application != "behind" || got[0].CurrentVersion != "1.0" || got[0].CompatibleMajor != 2 || got[0].Snapshot != "after" {
		t.Fatalf("major mismatch lost: %+v", got[0])
	}
	// 現在定義にない要求は APPLICATION_RELEASE_CONTRACT_NOT_CURRENT が扱う。
	for _, c := range got {
		if c.Application == "missing" {
			t.Fatalf("existing missing-contract warning is duplicated: %+v", c)
		}
	}
	if before := ApplicationReleaseContractCandidates(world, "before"); len(before) != 1 || before[0].Snapshot != "before" {
		t.Fatalf("snapshot scan is not independent: %+v", before)
	}
	if empty := ApplicationReleaseContractCandidates(ir.World{}, "after"); len(empty) != 0 {
		t.Fatalf("empty world produced candidates: %+v", empty)
	}
}
