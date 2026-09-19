package main

import "testing"

type providerFeasibilityScopeProbe struct {
	SemanticCandidate  string
	RealizationVariant string
	Provider           string
	HardProvider       bool
	Result             string
	Reason             string
}

type providerContractionResultProbe struct {
	SemanticCandidateRetained bool
	RealizationEliminated     bool
	CompositeEliminated       bool
	Unresolved                bool
}

const (
	providerSupportedProbe     = "supported"
	providerContradictionProbe = "contradiction"
	providerUnresolvedProbe    = "unresolved"
)

func contractProviderFeasibilityProbe(p providerFeasibilityScopeProbe) providerContractionResultProbe {
	result := providerContractionResultProbe{SemanticCandidateRetained: true}
	switch p.Result {
	case providerContradictionProbe:
		result.RealizationEliminated = true
		if p.HardProvider {
			result.CompositeEliminated = true
		}
	case providerUnresolvedProbe:
		result.Unresolved = true
	}
	return result
}

func TestProviderFeasibilityMissingObservationDoesNotInvalidateSemanticCandidate(t *testing.T) {
	result := contractProviderFeasibilityProbe(providerFeasibilityScopeProbe{
		SemanticCandidate:  "remove sales.archive_candidate from desired schema",
		RealizationVariant: "postgres-current-drop-table",
		Provider:           "postgres",
		Result:             providerUnresolvedProbe,
		Reason:             "partition-observation-unavailable",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("missing provider observation invalidated semantic candidate")
	}
	if result.RealizationEliminated || result.CompositeEliminated {
		t.Fatalf("missing evidence was treated as contradiction: %+v", result)
	}
	if !result.Unresolved {
		t.Fatalf("missing provider observation did not remain unresolved: %+v", result)
	}
}

func TestProviderContradictionEliminatesOnlyMatchingRealizationVariant(t *testing.T) {
	result := contractProviderFeasibilityProbe(providerFeasibilityScopeProbe{
		SemanticCandidate:  "change primary key",
		RealizationVariant: "postgres-current-change-primary-key",
		Provider:           "postgres",
		Result:             providerContradictionProbe,
		Reason:             "current renderer cannot safely realize constraint identity",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("provider-specific contradiction invalidated semantic meaning")
	}
	if !result.RealizationEliminated {
		t.Fatalf("matching realization variant was not eliminated: %+v", result)
	}
	if result.CompositeEliminated {
		t.Fatalf("optional provider contradiction eliminated composite design: %+v", result)
	}
}

func TestHardProviderConstraintMayEliminateCompositeWithoutRewritingMeaning(t *testing.T) {
	result := contractProviderFeasibilityProbe(providerFeasibilityScopeProbe{
		SemanticCandidate:  "change primary key",
		RealizationVariant: "postgres-current-change-primary-key",
		Provider:           "postgres",
		HardProvider:       true,
		Result:             providerContradictionProbe,
		Reason:             "accepted target requires current postgres profile",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("hard provider incompatibility erased semantic subject")
	}
	if !result.RealizationEliminated || !result.CompositeEliminated {
		t.Fatalf("hard provider constraint did not eliminate the composite candidate: %+v", result)
	}
}

func TestProviderSupportDoesNotImplySemanticAcceptance(t *testing.T) {
	result := contractProviderFeasibilityProbe(providerFeasibilityScopeProbe{
		SemanticCandidate:  "remove ordinary table",
		RealizationVariant: "postgres-current-drop-table",
		Provider:           "postgres",
		Result:             providerSupportedProbe,
		Reason:             "complete observation shows no partition relation",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("supported realization unexpectedly removed semantic candidate")
	}
	if result.RealizationEliminated || result.CompositeEliminated || result.Unresolved {
		t.Fatalf("supported provider realization produced contraction side effects: %+v", result)
	}
}
