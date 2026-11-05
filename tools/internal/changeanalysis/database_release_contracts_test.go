package changeanalysis

import (
	"testing"

	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
)

func TestDatabaseReleaseContractCandidatesLatestPerMajorLine(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 6, Minor: 0}},
			}},
			{ID: "inventory", Contracts: []database.Contract{
				{ID: "inventory.line", Version: database.ContractVersion{Major: 3, Minor: 1}},
			}},
			{ID: "newdb", Contracts: []database.Contract{
				{ID: "newdb.item", Version: database.ContractVersion{Major: 3, Minor: 0}},
			}},
		},
		DatabaseReleases: []dbrelease.Definition{
			// salesの4.x系統は4.10.0が最新。4.1.0と4.9.0は履歴。
			{Database: "sales", Version: "4.1.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 1}},
			}},
			{Database: "sales", Version: "4.9.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 9}},
			}},
			{Database: "sales", Version: "4.10.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 10}},
			}},
			// salesの5.x系統は並行して保守されている。
			{Database: "sales", Version: "5.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 5, Minor: 0}},
			}},
			{Database: "sales", Version: "not-a-version", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 4, Minor: 0}},
			}},
			{Database: "inventory", Version: "2.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "inventory.line", Version: database.ContractVersion{Major: 2, Minor: 0}},
			}},
			// newdbは4.xが先行、3.xが一致なので候補にしない。
			{Database: "newdb", Version: "4.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "newdb.item", Version: database.ContractVersion{Major: 4, Minor: 0}},
			}},
			{Database: "newdb", Version: "3.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "newdb.item", Version: database.ContractVersion{Major: 3, Minor: 0}},
			}},
			// 現在定義にないContractだけを提供する系統は候補にしない。
			{Database: "sales", Version: "3.0.0", Provides: []dbrelease.ProvidedContract{
				{ID: "sales.legacy", Version: database.ContractVersion{Major: 1, Minor: 0}},
			}},
		},
	}
	got := DatabaseReleaseContractCandidates(world, "after")
	if len(got) != 3 {
		t.Fatalf("系統ごとの最新behind 3件を期待: %+v", got)
	}
	// Database順: inventory, sales 4.10.0, sales 5.0.0。
	if got[0].Database != "inventory" || got[0].Version != "2.0.0" || got[0].ProvidedVersion != "2.0" || got[0].CurrentVersion != "3.1" {
		t.Fatalf("inventoryの最新behindが正しくありません: %+v", got[0])
	}
	if got[1].Database != "sales" || got[1].Version != "4.10.0" || got[1].ProvidedVersion != "4.10" || got[1].CurrentVersion != "6.0" {
		t.Fatalf("sales 4.x系統の最新behindが正しくありません: %+v", got[1])
	}
	if got[2].Database != "sales" || got[2].Version != "5.0.0" || got[2].ProvidedVersion != "5.0" || got[2].CurrentVersion != "6.0" {
		t.Fatalf("sales 5.x系統の最新behindが正しくありません: %+v", got[2])
	}
	for _, c := range got {
		if c.Snapshot != "after" || c.Contract == "" {
			t.Fatalf("identity が保持されていません: %+v", c)
		}
		if c.Database == "sales" && (c.Version == "4.1.0" || c.Version == "4.9.0" || c.Version == "3.0.0" || c.Version == "not-a-version") {
			t.Fatalf("履歴Releaseまたは不正versionが候補になっています: %+v", c)
		}
		if c.Database == "newdb" {
			t.Fatalf("先行方向または一致が候補になっています: %+v", c)
		}
	}
	if before := DatabaseReleaseContractCandidates(world, "before"); len(before) != 3 || before[0].Snapshot != "before" {
		t.Fatalf("snapshot 走査が独立していません: %+v", before)
	}
	if empty := DatabaseReleaseContractCandidates(ir.World{}, "after"); len(empty) != 0 {
		t.Fatalf("空worldで候補が出ています: %+v", empty)
	}
}
