package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadDataDeterminantAssociationSeedLedger(t *testing.T) seedLedger {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(
		repoPath(),
		"reference",
		"business-system-design",
		"data-determinant-association-seed-ledger.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := decodeSeedLedger(content)
	if err != nil {
		t.Fatal(err)
	}
	return ledger
}

func TestDataDeterminantAssociationReferenceSeparatesOriginalPressureFromSourceBackedFacts(t *testing.T) {
	ledger := loadDataDeterminantAssociationSeedLedger(t)

	if ledger.Canonical {
		t.Fatal("association determinant pressure must not claim Canonical authority")
	}
	if ledger.Seed != "coverage-assignment-data-determinant" {
		t.Fatalf("seed=%q", ledger.Seed)
	}
	if len(ledger.Sources) != 0 || len(ledger.SourceBackedFacts) != 0 {
		t.Fatalf(
			"TEI-original CoverageAssignment pressure was mislabeled as externally source-backed: sources=%+v facts=%+v",
			ledger.Sources,
			ledger.SourceBackedFacts,
		)
	}
	if len(ledger.CopiedArtifacts) != 0 || ledger.LicenseAnalogyMade {
		t.Fatalf(
			"association determinant pressure must not imply copied artifacts or license analogy: copied=%+v analogy=%v",
			ledger.CopiedArtifacts,
			ledger.LicenseAnalogyMade,
		)
	}

	requiredDecisions := map[string]bool{
		"TD-CA-1": false,
		"TD-CA-2": false,
		"TD-CA-3": false,
		"TD-CA-4": false,
		"TD-CA-5": false,
	}
	for _, decision := range ledger.TEIOriginalDecisions {
		if _, ok := requiredDecisions[decision.ID]; !ok {
			continue
		}
		if strings.TrimSpace(decision.Text) == "" || len(decision.Refs) == 0 {
			t.Fatalf("TEI-original determinant decision is incomplete: %+v", decision)
		}
		requiredDecisions[decision.ID] = true
	}
	for id, found := range requiredDecisions {
		if !found {
			t.Fatalf("TEI-original determinant decision %s is missing", id)
		}
	}
}

func TestDataDeterminantAssociationReferenceKeepsCorePromotionUnresolved(t *testing.T) {
	ledger := loadDataDeterminantAssociationSeedLedger(t)

	requiredUnresolved := map[string]bool{
		"external-domain-validation":           false,
		"relation-semantics-contract":          false,
		"business-identity-core-syntax":        false,
		"functional-dependency-core-syntax":    false,
		"temporal-overlap-policy":              false,
		"shared-acceptance-authoring-path":      false,
	}
	for _, item := range ledger.Unresolved {
		if item.Status != "unresolved" || strings.TrimSpace(item.Note) == "" {
			t.Fatalf("unresolved determinant item was promoted or left unexplained: %+v", item)
		}
		if _, ok := requiredUnresolved[item.Subject]; ok {
			requiredUnresolved[item.Subject] = true
		}
	}
	for subject, found := range requiredUnresolved {
		if !found {
			t.Fatalf("required unresolved determinant subject %q is missing", subject)
		}
	}

	if len(ledger.Proposals) == 0 {
		t.Fatal("a Core contract candidate proposal is required without promoting it to a decision")
	}
	if ledger.Review.Author != "tei-original-reference-authoring" ||
		ledger.Review.IndependentReview != "required" {
		t.Fatalf("association determinant pressure review state drifted: %+v", ledger.Review)
	}
	if ledger.Handoff.ArisesNaturally ||
		ledger.Handoff.Candidate.From != "" ||
		ledger.Handoff.Candidate.To != "" {
		t.Fatalf("data determinant pressure invented a Business Handoff: %+v", ledger.Handoff)
	}
}

func TestDataDeterminantAssociationReferenceCarriesCrossViewIdentityQuestions(t *testing.T) {
	ledger := loadDataDeterminantAssociationSeedLedger(t)
	if len(ledger.CrossViewQuestions) < 2 {
		t.Fatalf("cross-view determinant questions are missing: %+v", ledger.CrossViewQuestions)
	}
	for _, question := range ledger.CrossViewQuestions {
		if strings.TrimSpace(question.Question) == "" {
			t.Fatalf("empty cross-view determinant question: %+v", question)
		}
		distinct := map[string]bool{}
		for _, view := range question.Views {
			distinct[view] = true
		}
		if len(distinct) < 2 {
			t.Fatalf("determinant question %s does not cross views: %+v", question.ID, question)
		}
	}
}
