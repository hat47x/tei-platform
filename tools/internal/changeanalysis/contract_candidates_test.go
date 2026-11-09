package changeanalysis

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
	"tei/internal/core"
	"tei/internal/database"
	dbrelease "tei/internal/database/release"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/releaseplan"
)

func TestContractCandidatesBeforeAfterExactIdentityAndRemoval(t *testing.T) {
	old := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{{ID: "sales.order", Version: database.ContractVersion{Major: 1}}}}},
		Applications: []application.Application{{
			ID:        "app",
			Databases: []application.DatabaseRequirement{{Alias: "db", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}}},
			Patterns: []application.Pattern{
				{ID: "app.look", ContractQuery: &application.ContractQueryInvocation{Contract: "sales.order", Query: "find"}},
				{ID: "app.other", ContractQuery: &application.ContractQueryInvocation{Contract: "sales.other", Query: "find"}},
			},
		}},
		HeaderDetailContractQueryBindings: []realization.HeaderDetailContractQueryBinding{
			{Application: "app", Pattern: "app.a", Target: realization.ContractQueryTarget{Database: "sales", Contract: "sales.order", Query: "find"}},
			{Application: "app", Pattern: "app.b", Target: realization.ContractQueryTarget{Database: "sales", Contract: "sales.other", Query: "find"}},
			{Application: "app", Pattern: "app.c", Target: realization.ContractQueryTarget{Database: "other", Contract: "sales.order", Query: "find"}},
		},
		ApplicationReleases: []apprelease.Definition{{
			Application: "app",
			Version:     "1.2.3",
			Databases:   []application.DatabaseRequirement{{Alias: "db", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}}},
		}},
		DatabaseReleases: []dbrelease.Definition{
			{Database: "sales", Version: "1.0.0", Provides: []dbrelease.ProvidedContract{{ID: "sales.order", Version: database.ContractVersion{Major: 1}}}},
			{Database: "other", Version: "1.0.0", Provides: []dbrelease.ProvidedContract{{ID: "sales.order", Version: database.ContractVersion{Major: 1}}}},
		},
		SolutionReleases: []releaseplan.Definition{{
			ID:      "rollout",
			Version: "2026.1.1",
			Steps: []releaseplan.Step{
				{ID: "gate", Kind: releaseplan.StepDatabaseGate, Contract: "sales.order", CompatibleMajor: 1},
				{ID: "other-gate", Kind: releaseplan.StepDatabaseGate, Contract: "sales.other", CompatibleMajor: 1},
			},
		}},
	}
	after := old
	after.Databases = []database.Database{{ID: "sales"}}
	after.HeaderDetailContractQueryBindings = nil
	changes, err := ContractCandidates(old, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "removed" {
		t.Fatalf("removed contract: %+v, %v", changes, err)
	}
	seen := map[string]int{}
	for _, c := range changes[0].Consumers {
		seen[c.Snapshot+"/"+c.Kind]++
		if c.Kind == "header-detail-binding" && (c.ID != "app/app.a" || c.Query != "find") {
			t.Fatalf("unrelated consumer: %+v", c)
		}
		if c.Kind == "application-pattern-contract-query" && (c.ID != "app/app.look" || c.Query != "find") {
			t.Fatalf("unrelated pattern invocation: %+v", c)
		}
		switch c.Kind {
		case "application-release-contract-requirement":
			if c.ID != "app@1.2.3/db" {
				t.Fatalf("unrelated application release: %+v", c)
			}
		case "database-release-contract-provided":
			if c.ID != "sales@1.0.0" {
				t.Fatalf("unrelated database release: %+v", c)
			}
		case "solution-release-contract-gate":
			if c.ID != "rollout@2026.1.1/gate" {
				t.Fatalf("unrelated solution release gate: %+v", c)
			}
		}
	}
	if seen["before/header-detail-binding"] != 1 || seen["after/header-detail-binding"] != 0 || seen["before/application-contract-requirement"] != 1 || seen["after/application-contract-requirement"] != 1 {
		t.Fatalf("old and new declarations conflated: %+v", seen)
	}
	if seen["before/application-pattern-contract-query"] != 1 || seen["after/application-pattern-contract-query"] != 1 {
		t.Fatalf("pattern contract-query declaration missing: %+v", seen)
	}
	for _, kind := range []string{"application-release-contract-requirement", "database-release-contract-provided", "solution-release-contract-gate"} {
		if seen["before/"+kind] != 1 || seen["after/"+kind] != 1 {
			t.Fatalf("release declaration %s missing from a snapshot: %+v", kind, seen)
		}
	}
	if same, err := ContractCandidates(old, old); err != nil || len(same) != 0 {
		t.Fatalf("unchanged contract produced candidate: %+v %v", same, err)
	}
	duplicate := old
	duplicate.Databases = append([]database.Database(nil), old.Databases...)
	duplicate.Databases[0].Contracts = append(append([]database.Contract(nil), old.Databases[0].Contracts...), old.Databases[0].Contracts[0])
	if _, err := ContractCandidates(duplicate, old); err == nil {
		t.Fatal("duplicate contract identity accepted")
	}
	// Contract-level candidates deliberately retain every Query under that
	// contract; an unrelated Query must not be mislabeled as the changed one.
	more := old
	more.HeaderDetailContractQueryBindings = append(append([]realization.HeaderDetailContractQueryBinding(nil), old.HeaderDetailContractQueryBindings...), realization.HeaderDetailContractQueryBinding{Application: "app", Pattern: "app.d", Target: realization.ContractQueryTarget{Database: "sales", Contract: "sales.order", Query: "other"}})
	moreAfter := more
	moreAfter.Databases = []database.Database{{ID: "sales", Contracts: []database.Contract{{ID: "sales.order", Version: database.ContractVersion{Major: 1, Minor: 1}}}}}
	candidates, err := ContractCandidates(more, moreAfter)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("changed contract: %+v, %v", candidates, err)
	}
	otherCount := 0
	for _, consumer := range candidates[0].Consumers {
		if consumer.Query == "other" {
			otherCount++
		}
	}
	if otherCount != 2 {
		t.Fatalf("same-contract other Query must remain a conservative candidate with its own identity: %+v", candidates)
	}
}

// 現在のApplication定義とApplication Releaseは別の宣言である。同じContract
// 要求を両方が持つことはあるが、Releaseだけが履歴として残る場合や、現在定義
// だけが次のReleaseより先に変わる場合がある。片方へ統合すると参照を見落とす
// ため、候補は2種類を独立に保持する。
func TestContractCandidatesApplicationDefinitionAndReleaseAreIndependent(t *testing.T) {
	requirement := []application.DatabaseRequirement{{
		Alias: "db", TargetDatabase: "sales",
		RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 2}},
	}}
	definition := func(w *ir.World) {
		w.Applications = []application.Application{{ID: "app", Databases: requirement}}
	}
	release := func(w *ir.World) {
		w.ApplicationReleases = []apprelease.Definition{{Application: "app", Version: "1.2.3", Databases: requirement}}
	}
	both := func(w *ir.World) {
		definition(w)
		release(w)
	}
	world := func(major int, mutate func(*ir.World)) ir.World {
		w := ir.World{Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: major}},
		}}}}
		mutate(&w)
		return w
	}
	consumers := func(t *testing.T, mutate func(*ir.World)) ([]ContractConsumerCandidate, map[string]int) {
		t.Helper()
		candidates, err := ContractCandidates(world(1, mutate), world(2, mutate))
		if err != nil || len(candidates) != 1 || candidates[0].Change != "changed" {
			t.Fatalf("changed contract candidate: %+v, %v", candidates, err)
		}
		counts := map[string]int{}
		for _, c := range candidates[0].Consumers {
			counts[c.Snapshot+"/"+c.Kind]++
		}
		return candidates[0].Consumers, counts
	}

	// 現在定義だけが要求する。
	got, counts := consumers(t, definition)
	if counts["before/application-contract-requirement"] != 1 || counts["after/application-contract-requirement"] != 1 {
		t.Fatalf("current definition declaration dropped: %+v", got)
	}
	if counts["before/application-release-contract-requirement"] != 0 || counts["after/application-release-contract-requirement"] != 0 {
		t.Fatalf("missing Release was invented as a declaration: %+v", got)
	}

	// Releaseだけが要求する。
	got, counts = consumers(t, release)
	if counts["before/application-release-contract-requirement"] != 1 || counts["after/application-release-contract-requirement"] != 1 {
		t.Fatalf("Release declaration dropped: %+v", got)
	}
	if counts["before/application-contract-requirement"] != 0 || counts["after/application-contract-requirement"] != 0 {
		t.Fatalf("missing current definition was invented as a declaration: %+v", got)
	}

	// 両方が同じ要求を宣言しても、別の宣言として両方残す。
	got, counts = consumers(t, both)
	if len(got) != 4 {
		t.Fatalf("two kinds x two snapshots expected: %+v", got)
	}
	want := map[string]string{
		"before/application-contract-requirement":         "app/db",
		"after/application-contract-requirement":          "app/db",
		"before/application-release-contract-requirement": "app@1.2.3/db",
		"after/application-release-contract-requirement":  "app@1.2.3/db",
	}
	for _, c := range got {
		if key := c.Snapshot + "/" + c.Kind; want[key] != c.ID {
			t.Fatalf("declaration identity or kind conflated: %+v", c)
		}
	}
}

func appReleaseWithContract(app, version, alias, contract string, major int) apprelease.Definition {
	return apprelease.Definition{
		Application: core.CanonicalID(app), Version: version,
		Databases: []application.DatabaseRequirement{{
			Alias: core.LocalName(alias), TargetDatabase: "sales",
			RequiredContracts: []database.ContractRequirement{{ContractID: core.CanonicalID(contract), CompatibleMajor: major}},
		}},
	}
}

func releaseConsumerIDs(consumers []ContractConsumerCandidate, snapshot string) map[string]bool {
	out := map[string]bool{}
	for _, c := range consumers {
		if c.Snapshot == snapshot && c.Kind == "application-release-contract-requirement" {
			out[c.ID] = true
		}
	}
	return out
}

func dbReleaseProviding(version, contract string, major, minor int) dbrelease.Definition {
	return dbrelease.Definition{
		Database: "sales", Version: version,
		Provides: []dbrelease.ProvidedContract{{ID: core.CanonicalID(contract), Version: database.ContractVersion{Major: major, Minor: minor}}},
	}
}

func providedConsumerIDs(consumers []ContractConsumerCandidate, snapshot string) map[string]bool {
	out := map[string]bool{}
	for _, c := range consumers {
		if c.Snapshot == snapshot && c.Kind == "database-release-contract-provided" {
			out[c.ID] = true
		}
	}
	return out
}

func definitionConsumerIDs(consumers []ContractConsumerCandidate, snapshot string) map[string]bool {
	out := map[string]bool{}
	for _, c := range consumers {
		if c.Snapshot == snapshot && c.Kind == "application-contract-requirement" {
			out[c.ID] = true
		}
	}
	return out
}

func definitionConsumerCount(consumers []ContractConsumerCandidate, snapshot string) int {
	n := 0
	for _, c := range consumers {
		if c.Snapshot == snapshot && c.Kind == "application-contract-requirement" {
			n++
		}
	}
	return n
}

// Contractが削除されても、履歴Releaseの参照は消さない。ただし同じApplication
// major系統・aliasで同じ要求を繰り返すReleaseは、その要求を最後に宣言した
// 1件だけを残す。全履歴を並べても削除検出の情報は増えない。
func TestContractCandidatesRemovedContractKeepsLatestReleasePerMajorLine(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		ApplicationReleases: []apprelease.Definition{
			appReleaseWithContract("order_entry", "3.7.0", "db", "sales.order", 1),
			appReleaseWithContract("order_entry", "3.8.0", "db", "sales.order", 1),
			// 3.9.0は要求を落としている。3.x系統の最後の宣言は3.8.0。
			appReleaseWithContract("order_entry", "3.9.0", "db", "sales.other", 1),
			// 並行保守の2.x系統と、別Applicationの系統は独立して残す。
			appReleaseWithContract("order_entry", "2.9.0", "db", "sales.order", 1),
			appReleaseWithContract("billing", "1.4.0", "db", "sales.order", 1),
		},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales"}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "removed" {
		t.Fatalf("removed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := releaseConsumerIDs(changes[0].Consumers, snapshot)
		want := []string{"order_entry@3.8.0/db", "order_entry@2.9.0/db", "billing@1.4.0/db"}
		if len(got) != len(want) {
			t.Fatalf("%s: 系統ごとの最新宣言3件を期待: %v", snapshot, got)
		}
		for _, id := range want {
			if !got[id] {
				t.Fatalf("%s: 最新宣言 %s がありません: %v", snapshot, id, got)
			}
		}
		for _, id := range []string{"order_entry@3.7.0/db", "order_entry@3.9.0/db"} {
			if got[id] {
				t.Fatalf("%s: 履歴または非参照Releaseを報告しています: %v", snapshot, got)
			}
		}
	}
}

// 変更（major更新）でも同じ規則を使う。3.x系統の3.9.0/3.9.1ではなく、
// 最後に宣言した3.10.0だけを表示する。
func TestContractCandidatesChangedContractKeepsLatestReleasePerMajorLine(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		ApplicationReleases: []apprelease.Definition{
			appReleaseWithContract("order_entry", "3.9.0", "db", "sales.order", 1),
			appReleaseWithContract("order_entry", "3.9.1", "db", "sales.order", 1),
			appReleaseWithContract("order_entry", "3.10.0", "db", "sales.order", 1),
		},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales", Contracts: []database.Contract{
		{ID: "sales.order", Version: database.ContractVersion{Major: 2}},
	}}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "changed" {
		t.Fatalf("changed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := releaseConsumerIDs(changes[0].Consumers, snapshot)
		if len(got) != 1 || !got["order_entry@3.10.0/db"] {
			t.Fatalf("%s: 系統の最新宣言1件を期待: %v", snapshot, got)
		}
	}
}

// 要求を最後に宣言したReleaseが履歴側にしかなくても、そのReleaseは残す。
// 新しいReleaseで要求が落ちたことを理由に参照を消さない。
func TestContractCandidatesHistoryOnlyReleaseDeclarationSurvives(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		ApplicationReleases: []apprelease.Definition{
			appReleaseWithContract("order_entry", "3.7.0", "db", "sales.order", 1),
			appReleaseWithContract("order_entry", "3.8.0", "db", "sales.other", 1),
			appReleaseWithContract("order_entry", "3.9.0", "db", "sales.other", 1),
		},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales"}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "removed" {
		t.Fatalf("removed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := releaseConsumerIDs(changes[0].Consumers, snapshot)
		if len(got) != 1 || !got["order_entry@3.7.0/db"] {
			t.Fatalf("%s: 履歴側の最後の宣言を残すべき: %v", snapshot, got)
		}
	}
}

// 同じReleaseが同じContractを別aliasで要求する場合、aliasは別の消費経路
// なので両方残す。
func TestContractCandidatesReleaseAliasesStayDistinct(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		ApplicationReleases: []apprelease.Definition{{
			Application: "order_entry", Version: "3.8.0",
			Databases: []application.DatabaseRequirement{
				{Alias: "read", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}},
				{Alias: "write", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}},
			},
		}},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales"}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "removed" {
		t.Fatalf("removed contract candidate: %+v, %v", changes, err)
	}
	got := releaseConsumerIDs(changes[0].Consumers, "before")
	if len(got) != 2 || !got["order_entry@3.8.0/read"] || !got["order_entry@3.8.0/write"] {
		t.Fatalf("aliasごとの宣言を保持すべき: %v", got)
	}
}

// Database Releaseの提供Contractも、同じDatabase・Release major系統で同じ
// Contractを繰り返す履歴を全件並べない。系統ごとに最後に提供したReleaseだけを
// 残し、削除されたContractでも系統被覆は失わない。
func TestContractCandidatesRemovedContractKeepsLatestDatabaseReleasePerMajorLine(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		DatabaseReleases: []dbrelease.Definition{
			dbReleaseProviding("3.9.0", "sales.order", 1, 0),
			dbReleaseProviding("4.0.0", "sales.order", 1, 0),
			dbReleaseProviding("4.1.0", "sales.order", 1, 1),
			dbReleaseProviding("4.2.0", "sales.order", 1, 2),
			dbReleaseProviding("5.0.0", "sales.order", 1, 0),
			// 6.x系統は別Contractだけを提供する。
			dbReleaseProviding("6.0.0", "sales.other", 1, 0),
		},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales"}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "removed" {
		t.Fatalf("removed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := providedConsumerIDs(changes[0].Consumers, snapshot)
		want := []string{"sales@3.9.0", "sales@4.2.0", "sales@5.0.0"}
		if len(got) != len(want) {
			t.Fatalf("%s: 系統ごとの最新提供3件を期待: %v", snapshot, got)
		}
		for _, id := range want {
			if !got[id] {
				t.Fatalf("%s: 最新提供 %s がありません: %v", snapshot, id, got)
			}
		}
		for _, id := range []string{"sales@4.0.0", "sales@4.1.0", "sales@6.0.0"} {
			if got[id] {
				t.Fatalf("%s: 履歴または非提供Releaseを報告しています: %v", snapshot, got)
			}
		}
	}
}

// Contractの変更でも同じ規則を使う。
func TestContractCandidatesChangedContractKeepsLatestDatabaseReleasePerMajorLine(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		DatabaseReleases: []dbrelease.Definition{
			dbReleaseProviding("4.1.0", "sales.order", 1, 1),
			dbReleaseProviding("4.2.0", "sales.order", 1, 2),
		},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales", Contracts: []database.Contract{
		{ID: "sales.order", Version: database.ContractVersion{Major: 2}},
	}}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "changed" {
		t.Fatalf("changed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := providedConsumerIDs(changes[0].Consumers, snapshot)
		if len(got) != 1 || !got["sales@4.2.0"] {
			t.Fatalf("%s: 系統の最新提供1件を期待: %v", snapshot, got)
		}
	}
}

// 提供を最後に宣言したReleaseが履歴側にしかなくても、そのReleaseは残す。
func TestContractCandidatesHistoryOnlyDatabaseReleaseProviderSurvives(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		DatabaseReleases: []dbrelease.Definition{
			dbReleaseProviding("4.0.0", "sales.order", 1, 0),
			dbReleaseProviding("4.1.0", "sales.other", 1, 0),
			dbReleaseProviding("4.2.0", "sales.other", 1, 0),
		},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales"}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "removed" {
		t.Fatalf("removed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := providedConsumerIDs(changes[0].Consumers, snapshot)
		if len(got) != 1 || !got["sales@4.0.0"] {
			t.Fatalf("%s: 履歴側の最後の提供を残すべき: %v", snapshot, got)
		}
	}
}

// 現在のApplication定義はApplicationごとに1つで履歴を持たない。同じaliasで
// 同じContractを重複宣言した場合、識別子が同一の候補行は情報を増やさないため
// 1件に畳む。
func TestContractCandidatesCurrentApplicationDuplicateDeclarationCollapses(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		Applications: []application.Application{{
			ID: "app",
			Databases: []application.DatabaseRequirement{{
				Alias: "db", TargetDatabase: "sales",
				RequiredContracts: []database.ContractRequirement{
					{ContractID: "sales.order", CompatibleMajor: 1},
					{ContractID: "sales.order", CompatibleMajor: 1},
				},
			}},
		}},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales", Contracts: []database.Contract{
		{ID: "sales.order", Version: database.ContractVersion{Major: 2}},
	}}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "changed" {
		t.Fatalf("changed contract candidate: %+v, %v", changes, err)
	}
	for _, snapshot := range []string{"before", "after"} {
		got := definitionConsumerIDs(changes[0].Consumers, snapshot)
		if definitionConsumerCount(changes[0].Consumers, snapshot) != 1 || len(got) != 1 || !got["app/db"] {
			t.Fatalf("%s: 同一宣言を1件に畳むべき: count=%d %v", snapshot, definitionConsumerCount(changes[0].Consumers, snapshot), got)
		}
	}
}

// 別aliasは別の消費経路なので、現在のApplication定義でも両方残す。
func TestContractCandidatesCurrentApplicationAliasesStayDistinct(t *testing.T) {
	before := ir.World{
		Databases: []database.Database{{ID: "sales", Contracts: []database.Contract{
			{ID: "sales.order", Version: database.ContractVersion{Major: 1}},
		}}},
		Applications: []application.Application{{
			ID: "app",
			Databases: []application.DatabaseRequirement{
				{Alias: "read", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}},
				{Alias: "write", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}},
			},
		}},
	}
	after := before
	after.Databases = []database.Database{{ID: "sales", Contracts: []database.Contract{
		{ID: "sales.order", Version: database.ContractVersion{Major: 2}},
	}}}
	changes, err := ContractCandidates(before, after)
	if err != nil || len(changes) != 1 || changes[0].Change != "changed" {
		t.Fatalf("changed contract candidate: %+v, %v", changes, err)
	}
	got := definitionConsumerIDs(changes[0].Consumers, "before")
	if len(got) != 2 || !got["app/read"] || !got["app/write"] {
		t.Fatalf("aliasごとの宣言を保持すべき: %v", got)
	}
}
