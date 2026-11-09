package changeanalysis

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
)

func appReleaseAt(app, version, contract string, major int) apprelease.Definition {
	return apprelease.Definition{
		Application: core.CanonicalID(app), Version: version,
		Databases: []application.DatabaseRequirement{{
			Alias: "db", TargetDatabase: "sales",
			RequiredContracts: []database.ContractRequirement{{ContractID: core.CanonicalID(contract), CompatibleMajor: major}},
		}},
	}
}

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

// 同じApplicationの同じmajor系統では、同じContract要求を繰り返す履歴Releaseを
// すべて列挙せず、その系統で最後に宣言したReleaseだけを観測する。
func TestApplicationReleaseContractCandidatesLatestDeclarationPerMajorLine(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 6, Minor: 0}},
			}},
		},
		ApplicationReleases: []apprelease.Definition{
			// 3.x系統は3.10.0が最新。3.8.0と3.9.0は履歴。
			appReleaseAt("order_entry", "3.8.0", "sales.order", 4),
			appReleaseAt("order_entry", "3.9.0", "sales.order", 4),
			appReleaseAt("order_entry", "3.10.0", "sales.order", 4),
			// 2.x系統は並行保守。最新の2.9.0だけを観測する。
			appReleaseAt("order_entry", "2.9.0", "sales.order", 4),
			// 別Applicationは別系統。
			appReleaseAt("billing", "1.4.0", "sales.order", 4),
			// 現行majorに一致する系統は候補にしない。
			appReleaseAt("order_entry", "4.0.0", "sales.order", 6),
			// 不正versionは順序付けできないため無視する。
			appReleaseAt("order_entry", "not-a-version", "sales.order", 4),
		},
	}
	got := ApplicationReleaseContractCandidates(world, "after")
	if len(got) != 3 {
		t.Fatalf("系統ごとの最新3件を期待: %+v", got)
	}
	// Application順: billing, order_entry 2.9.0, order_entry 3.10.0。
	want := []struct {
		application core.CanonicalID
		version     string
	}{
		{"billing", "1.4.0"},
		{"order_entry", "2.9.0"},
		{"order_entry", "3.10.0"},
	}
	for i, w := range want {
		if got[i].Application != w.application || got[i].Version != w.version {
			t.Fatalf("%d番目が系統の最新ではありません: %+v", i, got)
		}
	}
	for _, c := range got {
		switch c.Version {
		case "3.8.0", "3.9.0", "not-a-version":
			t.Fatalf("履歴Releaseまたは不正versionを報告しています: %+v", c)
		case "4.0.0":
			t.Fatalf("現行majorに一致するReleaseを報告しています: %+v", c)
		}
	}
	if before := ApplicationReleaseContractCandidates(world, "before"); len(before) != 3 || before[0].Snapshot != "before" {
		t.Fatalf("snapshot走査が独立していません: %+v", before)
	}
}

// 系統の最新Releaseで別のContract要求が落ちても、そのContractの最新宣言が
// 残る限り報告する。履歴Releaseの要求を一律に消さない。
func TestApplicationReleaseContractCandidatesKeepsLatestDeclarationPerContract(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 6, Minor: 0}},
				{ID: "sales.line", Version: database.ContractVersion{Major: 6, Minor: 0}},
			}},
		},
		ApplicationReleases: []apprelease.Definition{
			{Application: "order_entry", Version: "3.8.0", Databases: []application.DatabaseRequirement{{
				Alias: "db", TargetDatabase: "sales",
				RequiredContracts: []database.ContractRequirement{
					{ContractID: "sales.order", CompatibleMajor: 4},
					{ContractID: "sales.line", CompatibleMajor: 4},
				},
			}}},
			// 3.9.0はsales.lineをもう要求しない。sales.lineの最新宣言は3.8.0。
			{Application: "order_entry", Version: "3.9.0", Databases: []application.DatabaseRequirement{{
				Alias: "db", TargetDatabase: "sales",
				RequiredContracts: []database.ContractRequirement{
					{ContractID: "sales.order", CompatibleMajor: 4},
				},
			}}},
		},
	}
	got := ApplicationReleaseContractCandidates(world, "after")
	if len(got) != 2 {
		t.Fatalf("契約ごとの最新宣言2件を期待: %+v", got)
	}
	seen := map[core.CanonicalID]string{}
	for _, c := range got {
		seen[c.Contract] = c.Version
	}
	if seen["sales.order"] != "3.9.0" || seen["sales.line"] != "3.8.0" {
		t.Fatalf("最新宣言の選択が違います: %+v", got)
	}
}

// 同じ系統のより新しいReleaseが現行majorに一致する要求へ更新していれば、
// 古いReleaseの一致しない要求を繰り越さない。
func TestApplicationReleaseContractCandidatesNewerMatchSupersedesOlderStale(t *testing.T) {
	world := ir.World{
		Databases: []database.Database{
			{ID: "sales", Contracts: []database.Contract{
				{ID: "sales.order", Version: database.ContractVersion{Major: 6, Minor: 0}},
			}},
		},
		ApplicationReleases: []apprelease.Definition{
			appReleaseAt("order_entry", "3.8.0", "sales.order", 4),
			appReleaseAt("order_entry", "3.9.0", "sales.order", 6),
		},
	}
	if got := ApplicationReleaseContractCandidates(world, "after"); len(got) != 0 {
		t.Fatalf("古いReleaseの要求を繰り越しています: %+v", got)
	}
}
