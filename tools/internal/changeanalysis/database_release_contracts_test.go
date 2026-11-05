package changeanalysis

import (
	"testing"

	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
)

func TestDatabaseReleaseContractCandidatesOnlyMajorBehind(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 5, Minor: 0}},
				{ID: "sales.line", Version: database.ContractVersion{Major: 2, Minor: 1}},
			}},
		},
		DatabaseReleases: []dbrelease.Definition{
			{Database: "sales", Version: "4.1.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 1}},
			}},
			{Database: "sales", Version: "4.2.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 2}},
			}},
			{Database: "sales", Version: "5.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 5, Minor: 0}},
			}},
			{Database: "sales", Version: "6.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 6, Minor: 0}},
			}},
			{Database: "sales", Version: "4.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.legacy", Version: database.ContractVersion{Major: 1, Minor: 0}},
			}},
			{Database: "sales", Version: "4.3.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.line", Version: database.ContractVersion{Major: 2, Minor: 0}},
			}},
		},
	}
	got := DatabaseReleaseContractCandidates(world, "after")
	if len(got) != 2 {
		t.Fatalf("major が古い2件だけを期待: %+v", got)
	}
	if got[0].Version != "4.1.0" || got[0].ProvidedVersion != "4.1" || got[0].CurrentVersion != "5.0" || got[0].Reason != DatabaseReleaseContractBehindCurrent {
		t.Fatalf("4.1.0 の behind が正しくありません: %+v", got[0])
	}
	if got[1].Version != "4.2.0" || got[1].ProvidedVersion != "4.2" {
		t.Fatalf("4.2.0 の behind が正しくありません: %+v", got[1])
	}
	for _, c := range got {
		if c.Snapshot != "after" || c.Database != "sales" || c.Contract != "sales.order" {
			t.Fatalf("identity が保持されていません: %+v", c)
		}
	}
	// 同一worldでもsnapshotを分けて走査する。minorだけ古いv2.0、一致、先行、
	// 現在定義のないContractは、いずれも候補にしない。
	if before := DatabaseReleaseContractCandidates(world, "before"); len(before) != 2 || before[0].Snapshot != "before" {
		t.Fatalf("snapshot 走査が独立していません: %+v", before)
	}
	empty := DatabaseReleaseContractCandidates(ir.World{}, "after")
	if len(empty) != 0 {
		t.Fatalf("空worldで候補が出ています: %+v", empty)
	}
}
