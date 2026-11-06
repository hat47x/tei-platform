package changeanalysis

import (
	"testing"

	"tei/internal/application"
	apprelease "tei/internal/application/release"
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
