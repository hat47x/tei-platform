package main

import "testing"

const (
	secondApplicationS5FreshnessBaseRevision = "bfda34c27f9768ddd744f9acfbb6991d9c104bb4"
	secondApplicationS5CustomerFingerprint = "sha256:d0897ed9ac8ce7e01200b05285e445fa57fa402abd0725542c7cf7ebfa0ad23b"
	secondApplicationS5ContractFingerprint = "sha256:bfbc0ed04d258fa05e254dc6b0c6a9779069e98d4f93511d0f062631ddb5fc15"
	secondApplicationS5OriginPatternFingerprint = "sha256:ae17bb1078c0a27ceeb20a5ece20431d98bee403c28fe7bc468643691263ef21"
)

func TestSecondApplicationS5FreshnessRequalifiesCurrentSharedAuthority(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}

	if secondApplicationS5FreshnessBaseRevision == "" {
		t.Fatal("review base revision is empty")
	}

	if got.SemanticCandidate.Qualification != "applicable" ||
		got.SemanticCandidate.ReuseRelation != "linked-reuse" {
		t.Fatalf("Customer Semantic is not fresh/applicable: %+v", got.SemanticCandidate)
	}
	if got.CustomerIDTypeCandidate.Qualification != "applicable" ||
		got.CustomerIDTypeCandidate.ReuseRelation != "linked-reuse" {
		t.Fatalf("Customer ID Type is not fresh/applicable: %+v", got.CustomerIDTypeCandidate)
	}
	if got.ContractCandidate.Qualification != "applicable" ||
		got.ContractCandidate.ReuseRelation != "conditional-linked-reuse" {
		t.Fatalf("Customer Contract is not fresh/applicable: %+v", got.ContractCandidate)
	}
	if got.ContractQueryCandidate.Qualification != "applicable" ||
		got.ContractQueryCandidate.ReuseRelation != "conditional-linked-reuse" {
		t.Fatalf("Customer Contract Query is not fresh/applicable: %+v", got.ContractQueryCandidate)
	}
	if got.PatternProvenanceCandidate.Qualification != "applicable-as-provenance" ||
		got.PatternProvenanceCandidate.ReuseRelation != "adopted-local-variation" {
		t.Fatalf("origin Pattern provenance qualification drifted: %+v", got.PatternProvenanceCandidate)
	}
}

func TestSecondApplicationS5FreshnessBindsExactCurrentSourceFingerprints(t *testing.T) {
	got, err := deriveSecondApplicationS1S2Packet(repoPath())
	if err != nil {
		t.Fatal(err)
	}

	if got.SemanticCandidate.AuthoritySource != "semantic/customer/customer.tei" ||
		got.SemanticCandidate.AuthorityFingerprint != secondApplicationS5CustomerFingerprint {
		t.Fatalf("Customer Semantic source/fingerprint drifted: %+v", got.SemanticCandidate)
	}
	if got.CustomerIDTypeCandidate.AuthoritySource != "semantic/customer/customer.tei" ||
		got.CustomerIDTypeCandidate.AuthorityFingerprint != secondApplicationS5CustomerFingerprint {
		t.Fatalf("Customer ID Type source/fingerprint drifted: %+v", got.CustomerIDTypeCandidate)
	}
	if got.ContractCandidate.AuthoritySource != "databases/master-data/contracts/customer.tei" ||
		got.ContractCandidate.AuthorityFingerprint != secondApplicationS5ContractFingerprint {
		t.Fatalf("Customer Contract source/fingerprint drifted: %+v", got.ContractCandidate)
	}
	if got.ContractQueryCandidate.AuthoritySource != "databases/master-data/contracts/customer.tei" ||
		got.ContractQueryCandidate.AuthorityFingerprint != secondApplicationS5ContractFingerprint {
		t.Fatalf("Customer Contract Query source/fingerprint drifted: %+v", got.ContractQueryCandidate)
	}
	if got.PatternProvenanceCandidate.AuthoritySource != "applications/order-entry/patterns/customer-reference.tei" ||
		got.PatternProvenanceCandidate.AuthorityFingerprint != secondApplicationS5OriginPatternFingerprint {
		t.Fatalf("origin Pattern provenance source/fingerprint drifted: %+v", got.PatternProvenanceCandidate)
	}
}

func TestSecondApplicationS5FreshnessPreservesExactProposalIdentity(t *testing.T) {
	s4 := secondApplicationS4Composition(secondApplicationS3Decision())
	got := secondApplicationS5Review(s4)

	if got.ProposalFingerprint != secondApplicationS5ExpectedProposalFingerprint {
		t.Fatalf("S4 Proposal changed after freshness capture: got=%q want=%q",
			got.ProposalFingerprint, secondApplicationS5ExpectedProposalFingerprint)
	}
}

func TestSecondApplicationS5FreshnessDoesNotSubstituteForAcceptanceWitness(t *testing.T) {
	got := secondApplicationS5Review(secondApplicationS4Composition(secondApplicationS3Decision()))
	got.BaseModelRevision = secondApplicationS5FreshnessBaseRevision

	if got.AcceptanceWitnessID != "" {
		t.Fatalf("freshness capture invented an Acceptance witness: %+v", got)
	}
	if status := secondApplicationS5Eligibility(got); status != "blocked-missing-witness" {
		t.Fatalf("freshness-only eligibility=%q", status)
	}
	if got.Decision != "pending" || got.SourceAuthoringAttempted {
		t.Fatalf("freshness capture crossed Shared Acceptance/S6 boundary: %+v", got)
	}
}
