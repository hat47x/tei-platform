package changeanalysis

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/realization"
)

func TestContractCandidatesBeforeAfterExactIdentityAndRemoval(t *testing.T) {
	old := ir.World{
		Databases:    []database.Database{{ID: "sales", Contracts: []database.Contract{{ID: "sales.order", Version: database.ContractVersion{Major: 1}}}}},
		Applications: []application.Application{{ID: "app", Databases: []application.DatabaseRequirement{{Alias: "db", TargetDatabase: "sales", RequiredContracts: []database.ContractRequirement{{ContractID: "sales.order", CompatibleMajor: 1}}}}}},
		HeaderDetailContractQueryBindings: []realization.HeaderDetailContractQueryBinding{
			{Application: "app", Pattern: "app.a", Target: realization.ContractQueryTarget{Database: "sales", Contract: "sales.order", Query: "find"}},
			{Application: "app", Pattern: "app.b", Target: realization.ContractQueryTarget{Database: "sales", Contract: "sales.other", Query: "find"}},
			{Application: "app", Pattern: "app.c", Target: realization.ContractQueryTarget{Database: "other", Contract: "sales.order", Query: "find"}},
		},
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
	}
	if seen["before/header-detail-binding"] != 1 || seen["after/header-detail-binding"] != 0 || seen["before/application-contract-requirement"] != 1 || seen["after/application-contract-requirement"] != 1 {
		t.Fatalf("old and new declarations conflated: %+v", seen)
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
