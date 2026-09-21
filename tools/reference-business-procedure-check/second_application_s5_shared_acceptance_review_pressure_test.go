package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

const secondApplicationS5ExpectedProposalFingerprint = "c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d"

type secondApplicationS5ReviewPacket struct {
	ProposalFingerprint      string
	BaseModelRevision        string
	AcceptanceWitnessID      string
	Decision                 string
	ReviewStatus             string
	NewDestinationSubjects   []string
	ReusedSharedSubjects     []string
	ProvenanceSubjects       []string
	EnvironmentMutationCount int
	RuntimeAuthorityGranted  bool
	SourceAuthoringAttempted bool
	RetainedUnresolved       []string
}

func TestSecondApplicationS5BindsReviewToExactS4Proposal(t *testing.T) {
	s4 := secondApplicationS4Composition(secondApplicationS3Decision())
	got := secondApplicationS5Review(s4)

	if got.ProposalFingerprint != secondApplicationS5ExpectedProposalFingerprint {
		t.Fatalf("Proposal fingerprint drifted: got=%q want=%q", got.ProposalFingerprint, secondApplicationS5ExpectedProposalFingerprint)
	}
	if got.Decision != "pending" || got.ReviewStatus != "review-ready" {
		t.Fatalf("S5 review packet self-accepted: %+v", got)
	}
}

func TestSecondApplicationS5SeparatesSemanticDiffFromSharedReuse(t *testing.T) {
	got := secondApplicationS5Review(secondApplicationS4Composition(secondApplicationS3Decision()))

	for _, want := range []string{
		"semantic:repair_intake",
		"semantic-data:repair_intake.intake",
		"semantic-attribute:repair_intake.intake.customer_id",
		"application:repair_intake",
		"semantic-relation:repair_intake.intake.customer",
		"application-pattern:repair_intake.customer_reference",
		"datasource-alias:repair_intake/customer_master",
	} {
		if !s5Contains(got.NewDestinationSubjects, want) {
			t.Fatalf("new destination subject missing %q: %+v", want, got.NewDestinationSubjects)
		}
	}
	for _, want := range []string{
		"semantic-data:customer.customer",
		"semantic-type:customer.customer_id",
		"database-contract:master_data.customer",
		"database-contract-query:master_data.customer/find_by_id",
	} {
		if !s5Contains(got.ReusedSharedSubjects, want) {
			t.Fatalf("shared reuse subject missing %q: %+v", want, got.ReusedSharedSubjects)
		}
	}
	if got.EnvironmentMutationCount != 0 {
		t.Fatalf("S5 included Environment mutation: %+v", got)
	}
}

func TestSecondApplicationS5RequiresExplicitWitnessAndBaseBeforeEligibility(t *testing.T) {
	base := secondApplicationS5Review(secondApplicationS4Composition(secondApplicationS3Decision()))

	if status := secondApplicationS5Eligibility(base); status != "blocked-missing-base-and-witness" {
		t.Fatalf("empty acceptance inputs status=%q", status)
	}

	withWitness := base
	withWitness.AcceptanceWitnessID = "explicit-review-witness"
	if status := secondApplicationS5Eligibility(withWitness); status != "blocked-missing-base" {
		t.Fatalf("witness-only status=%q", status)
	}

	withBase := base
	withBase.BaseModelRevision = "explicit-current-model-revision"
	if status := secondApplicationS5Eligibility(withBase); status != "blocked-missing-witness" {
		t.Fatalf("base-only status=%q", status)
	}

	eligible := base
	eligible.BaseModelRevision = "explicit-current-model-revision"
	eligible.AcceptanceWitnessID = "explicit-review-witness"
	if status := secondApplicationS5Eligibility(eligible); status != "eligible-for-shared-acceptance" {
		t.Fatalf("complete review inputs status=%q", status)
	}
	if eligible.Decision != "pending" {
		t.Fatalf("eligibility automatically became Accepted Decision: %+v", eligible)
	}
}

func TestSecondApplicationS5DoesNotGrantRuntimeAuthorityOrAuthorSource(t *testing.T) {
	got := secondApplicationS5Review(secondApplicationS4Composition(secondApplicationS3Decision()))

	if got.RuntimeAuthorityGranted {
		t.Fatalf("Shared Acceptance review granted Runtime authority: %+v", got)
	}
	if got.SourceAuthoringAttempted {
		t.Fatalf("S5 review wrote authoritative source before explicit acceptance: %+v", got)
	}
	for _, want := range []string{
		"authorization-principal",
		"authorization-resource-permission",
		"development-environment-binding",
		"production-environment-binding",
		"provider-verification",
		"target-realization",
	} {
		if !s5Contains(got.RetainedUnresolved, want) {
			t.Fatalf("S5 lost unresolved item %q: %+v", want, got.RetainedUnresolved)
		}
	}
}

func secondApplicationS5Review(s4 secondApplicationS4CompositionCandidate) secondApplicationS5ReviewPacket {
	payload := s4.SemanticSource + "\n---\n" + s4.ApplicationSource + "\n---\n" + s4.PatternSource
	sum := sha256.Sum256([]byte(payload))

	return secondApplicationS5ReviewPacket{
		ProposalFingerprint: hex.EncodeToString(sum[:]),
		BaseModelRevision:   "",
		AcceptanceWitnessID: "",
		Decision:            "pending",
		ReviewStatus:        "review-ready",
		NewDestinationSubjects: []string{
			"semantic:repair_intake",
			"semantic-data:repair_intake.intake",
			"semantic-attribute:repair_intake.intake.customer_id",
			"application:repair_intake",
			"semantic-relation:repair_intake.intake.customer",
			"application-pattern:repair_intake.customer_reference",
			"datasource-alias:repair_intake/customer_master",
		},
		ReusedSharedSubjects: []string{
			"semantic-data:customer.customer",
			"semantic-type:customer.customer_id",
			"database-contract:master_data.customer",
			"database-contract-query:master_data.customer/find_by_id",
		},
		ProvenanceSubjects: []string{
			"application-pattern:order_entry.customer_reference",
		},
		EnvironmentMutationCount: 0,
		RuntimeAuthorityGranted:  false,
		SourceAuthoringAttempted: false,
		RetainedUnresolved: []string{
			"authorization-principal",
			"authorization-resource-permission",
			"development-environment-binding",
			"production-environment-binding",
			"provider-verification",
			"target-realization",
		},
	}
}

func secondApplicationS5Eligibility(packet secondApplicationS5ReviewPacket) string {
	switch {
	case packet.BaseModelRevision == "" && packet.AcceptanceWitnessID == "":
		return "blocked-missing-base-and-witness"
	case packet.BaseModelRevision == "":
		return "blocked-missing-base"
	case packet.AcceptanceWitnessID == "":
		return "blocked-missing-witness"
	default:
		return "eligible-for-shared-acceptance"
	}
}

func s5Contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
