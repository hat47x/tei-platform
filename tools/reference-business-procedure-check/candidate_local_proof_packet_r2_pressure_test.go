package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/driver"
)

func TestCandidateLocalProofPacketKeepsR2ObservedFactSeparateFromHandoffAuthority(t *testing.T) {
	results := deriveSecondCorpusProofResultsPacketProbe(t)

	commandCandidate, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R2-M1-command-means-completion",
		RequiredProofIDs: []string{"P5-COMMAND-COMPLETION"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if commandCandidate.Elimination == nil || !reflect.DeepEqual(commandCandidate.Elimination.ProofIDs, []string{"P5-COMMAND-COMPLETION"}) {
		t.Fatalf("Command acceptance to physical completion contradiction was not contracted: %+v", commandCandidate)
	}

	observedThenContinue, err := buildCandidateLocalProofPacketProbe(results, candidateProofScopePacketProbe{
		CandidateID:      "R2-M2-observed-then-continue",
		RequiredProofIDs: []string{"P5-PHYSICAL-OBSERVED", "P6-HANDOFF-AUTHORITY"},
		DimensionByProof: map[string]string{
			"P6-HANDOFF-AUTHORITY": "responsibility-continuation",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if observedThenContinue.Elimination != nil {
		t.Fatalf("supported physical fact plus unresolved Handoff was incorrectly eliminated: %+v", observedThenContinue)
	}
	if len(observedThenContinue.SupportedProofs) != 1 || observedThenContinue.SupportedProofs[0].ID != "P5-PHYSICAL-OBSERVED" {
		t.Fatalf("positive physical Observation support was lost: %+v", observedThenContinue.SupportedProofs)
	}
	if len(observedThenContinue.UnresolvedDimensions) != 1 || observedThenContinue.UnresolvedDimensions[0].MemberProofs[0].ProofID != "P6-HANDOFF-AUTHORITY" {
		t.Fatalf("responsibility Handoff gap was not preserved: %+v", observedThenContinue.UnresolvedDimensions)
	}
	if !containsStringProofProbe(observedThenContinue.UnresolvedDimensions[0].MissingAuthorities, "accepted-business-handoff-source") {
		t.Fatalf("accepted Business Handoff source gap was lost: %+v", observedThenContinue.UnresolvedDimensions[0])
	}
}

func deriveSecondCorpusProofResultsPacketProbe(t *testing.T) []crossViewProofResultProbe {
	t.Helper()
	commandPressure := physicalWorldCompletionPressure{
		BusinessFact:    "manufacturing physically completed",
		CommandAccepted: true,
	}
	if diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", commandPressure); !hasDiagnosticCodePacketProbe(diagnostics, "BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION") {
		t.Fatalf("test setup: Command acceptance no longer pressures physical completion: %+v", diagnostics)
	}

	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-corpus.physical-state.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) { return true, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	observedPressure := physicalWorldCompletionPressure{
		BusinessFact:      "finished goods physically present",
		CommandAccepted:   true,
		ObservationResult: &execution,
	}
	if diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", observedPressure); len(diagnostics) != 0 {
		t.Fatalf("test setup: positive external Observation is not supported: %+v", diagnostics)
	}

	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	withoutExplicitHandoff := set
	withoutExplicitHandoff.Handoffs = nil
	model, err := deriveNormalizedBusinessModelProbe(withoutExplicitHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 0 {
		t.Fatalf("test setup: Observation or adjacency synthesized Handoff: %+v", model.Handoffs)
	}

	return []crossViewProofResultProbe{
		{
			ID:            "P5-COMMAND-COMPLETION",
			Family:        "system-result-actuality",
			Question:      "Command acceptance establishes manufacturing physical completion",
			Result:        proofContradictionProbe,
			EvidencePaths: []string{"driver-boundary:command-accepted/physical-completion-requires-observation"},
			Reason:        "Command acceptance alone cannot establish physical completion",
		},
		{
			ID:            "P5-PHYSICAL-OBSERVED",
			Family:        "system-result-actuality",
			Question:      "finished goods physically present has been explicitly observed",
			Result:        proofSupportedProbe,
			EvidencePaths: []string{"runtime-observation:second-corpus.physical-state.read/external-observed/result:true"},
			Reason:        "explicit external Observation supports only the observed physical fact",
		},
		{
			ID:                 "P6-HANDOFF-AUTHORITY",
			Family:             "responsibility-handoff",
			Question:           "Business responsibility continuation after the observed fact is defined by accepted Handoff authority",
			Result:             proofUnresolvedProbe,
			EvidencePaths:      []string{"runtime-observation:second-corpus.physical-state.read/external-observed/result:true"},
			MissingAuthorities: []string{"accepted-business-handoff-source", "accepted-handoff-endpoints", "accepted-handoff-condition"},
			Reason:             "positive physical Observation does not author Business Handoff from/to/condition",
		},
	}
}

func hasDiagnosticCodePacketProbe(diagnostics []diagnostic, code string) bool {
	for _, item := range diagnostics {
		if item.Code == code {
			return true
		}
	}
	return false
}
