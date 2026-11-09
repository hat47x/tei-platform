package changeanalysis

import (
	"testing"

	"tei/internal/core"
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

// 同じmajorでもDatabaseが異なれば別系統として扱い、履歴や別Databaseの
// Releaseを取り違えないことを確認する。
func TestDatabaseReleaseContractCandidatesSameMajorSeparateDatabases(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "line_a", Contracts: []database.Contract{
				{ID: "line_a.order", Version: database.ContractVersion{Major: 5, Minor: 0}},
			}},
			{ID: "line_b", Contracts: []database.Contract{
				{ID: "line_b.order", Version: database.ContractVersion{Major: 5, Minor: 0}},
			}},
		},
		DatabaseReleases: []dbrelease.Definition{
			{Database: "line_a", Version: "4.1.0", Provides: []dbrelease.ProvidedContract{
				{ID: "line_a.order", Version: database.ContractVersion{Major: 4, Minor: 1}},
			}},
			{Database: "line_a", Version: "4.2.0", Provides: []dbrelease.ProvidedContract{
				{ID: "line_a.order", Version: database.ContractVersion{Major: 4, Minor: 2}},
			}},
			{Database: "line_b", Version: "4.7.0", Provides: []dbrelease.ProvidedContract{
				{ID: "line_b.order", Version: database.ContractVersion{Major: 4, Minor: 7}},
			}},
			{Database: "line_b", Version: "4.8.0", Provides: []dbrelease.ProvidedContract{
				{ID: "line_b.order", Version: database.ContractVersion{Major: 4, Minor: 8}},
			}},
		},
	}
	got := DatabaseReleaseContractCandidates(world, "after")
	if len(got) != 2 {
		t.Fatalf("Databaseごとに最新1件ずつを期待: %+v", got)
	}
	want := map[core.CanonicalID]string{"line_a": "4.2.0", "line_b": "4.8.0"}
	for _, c := range got {
		if want[c.Database] != c.Version {
			t.Fatalf("%sの最新Releaseを取り違えています: %+v", c.Database, got)
		}
		if c.Version == "4.1.0" || c.Version == "4.7.0" {
			t.Fatalf("履歴Releaseを報告しています: %+v", c)
		}
		if c.Contract != core.CanonicalID(string(c.Database)+".order") {
			t.Fatalf("別DatabaseのContractを取り違えています: %+v", c)
		}
	}
}

// 同一Database・同一majorで明示的な系統識別子がない場合、候補はmajor単位で
// 1件に集約される。ここでは低いversionの系統が報告されない過小報告を
// 特性として固定する。系統識別子の追加はまだ行わない。
func TestDatabaseReleaseContractCandidatesSameMajorSameDatabaseCollapses(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "ambiguous", Contracts: []database.Contract{
				{ID: "ambiguous.a", Version: database.ContractVersion{Major: 5, Minor: 0}},
				{ID: "ambiguous.b", Version: database.ContractVersion{Major: 5, Minor: 0}},
			}},
		},
		DatabaseReleases: []dbrelease.Definition{
			{Database: "ambiguous", Version: "4.3.0", Provides: []dbrelease.ProvidedContract{
				{ID: "ambiguous.a", Version: database.ContractVersion{Major: 4, Minor: 3}},
			}},
			{Database: "ambiguous", Version: "4.9.0", Provides: []dbrelease.ProvidedContract{
				{ID: "ambiguous.b", Version: database.ContractVersion{Major: 4, Minor: 9}},
			}},
		},
	}
	got := DatabaseReleaseContractCandidates(world, "after")
	if len(got) != 1 || got[0].Version != "4.9.0" || got[0].Contract != "ambiguous.b" {
		t.Fatalf("同一majorの最新1件への集約を期待: %+v", got)
	}
	for _, c := range got {
		if c.Contract == "ambiguous.a" {
			t.Fatalf("系統識別子なしで低いversionの系統を報告しています: %+v", c)
		}
	}
}
