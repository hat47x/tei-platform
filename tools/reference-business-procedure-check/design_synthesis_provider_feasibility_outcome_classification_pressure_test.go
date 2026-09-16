package main

import "testing"

type providerOutcomeClassificationProbe struct {
	SemanticCandidate string
	Realization       string
	Surface           string
	OutcomeClass      string
	Owner             string
	HardProvider      bool
}

type providerOutcomeContractionProbe struct {
	SemanticCandidateRetained bool
	RealizationEliminated     bool
	CompositeEliminated       bool
	Unresolved                bool
	Owner                      string
	SemanticAccepted           bool
}

const (
	providerOutcomeMissingEvidenceProbe       = "missing-evidence"
	providerOutcomeObservedIncompatibleProbe  = "observed-incompatible"
	providerOutcomeStructurallyUnsupported    = "structurally-unsupported"
	providerOutcomeOperationalRecoveryProbe   = "operational-recovery"
	providerOutcomeSupportedRealizationProbe  = "supported-realization"
)

func classifyProviderOutcomeContractionProbe(p providerOutcomeClassificationProbe) providerOutcomeContractionProbe {
	result := providerOutcomeContractionProbe{
		SemanticCandidateRetained: true,
		Owner:                      p.Owner,
	}

	switch p.OutcomeClass {
	case providerOutcomeMissingEvidenceProbe:
		result.Unresolved = true
	case providerOutcomeObservedIncompatibleProbe, providerOutcomeStructurallyUnsupported:
		result.RealizationEliminated = true
		if p.HardProvider {
			result.CompositeEliminated = true
		}
	case providerOutcomeOperationalRecoveryProbe:
		result.Unresolved = true
	case providerOutcomeSupportedRealizationProbe:
		// Feasible realization evidence does not accept semantic meaning.
	}

	return result
}

func TestProviderOutcomeMissingEvidenceStaysUnresolved(t *testing.T) {
	result := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "remove sales.archive_candidate from desired schema",
		Realization:       "postgres-current-drop-table",
		Surface:           "error",
		OutcomeClass:      providerOutcomeMissingEvidenceProbe,
		Owner:             "provider-observation",
	})

	if !result.SemanticCandidateRetained || !result.Unresolved {
		t.Fatalf("missing provider evidence did not remain unresolved: %+v", result)
	}
	if result.RealizationEliminated || result.CompositeEliminated {
		t.Fatalf("missing evidence was collapsed into incompatibility: %+v", result)
	}
}

func TestProviderOutcomeObservedUnsafeEliminatesOnlyMatchingRealization(t *testing.T) {
	result := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "remove logical table from desired state",
		Realization:       "postgres-current-plain-drop-table",
		Surface:           "error",
		OutcomeClass:      providerOutcomeObservedIncompatibleProbe,
		Owner:             "provider-safety",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("provider safety contradiction erased semantic meaning")
	}
	if !result.RealizationEliminated || result.CompositeEliminated || result.Unresolved {
		t.Fatalf("observed provider incompatibility had wrong contraction scope: %+v", result)
	}
}

func TestProviderOutcomeUnsupportedOperationIsCapabilityScoped(t *testing.T) {
	result := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "change primary key",
		Realization:       "postgres-current-change-primary-key",
		Surface:           "error",
		OutcomeClass:      providerOutcomeStructurallyUnsupported,
		Owner:             "provider-capability",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("current provider capability gap invalidated semantic meaning")
	}
	if !result.RealizationEliminated || result.Unresolved {
		t.Fatalf("structural provider unsupported result was not realization-scoped: %+v", result)
	}
}

func TestProviderOutcomeOperationalRecoveryDoesNotBecomeStructuralIncompatibility(t *testing.T) {
	result := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "restore desired index state",
		Realization:       "postgres-index-reconciliation",
		Surface:           "error",
		OutcomeClass:      providerOutcomeOperationalRecoveryProbe,
		Owner:             "runtime-operations",
	})

	if !result.SemanticCandidateRetained || !result.Unresolved {
		t.Fatalf("operational recovery was not preserved as unresolved: %+v", result)
	}
	if result.RealizationEliminated || result.CompositeEliminated {
		t.Fatalf("operational recovery was treated as structural provider incompatibility: %+v", result)
	}
}

func TestProviderOutcomeHardProviderConstraintEliminatesCompositeOnlyForActualIncompatibility(t *testing.T) {
	incompatible := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "change primary key",
		Realization:       "postgres-current-change-primary-key",
		Surface:           "error",
		OutcomeClass:      providerOutcomeStructurallyUnsupported,
		Owner:             "provider-capability",
		HardProvider:      true,
	})
	if !incompatible.SemanticCandidateRetained || !incompatible.RealizationEliminated || !incompatible.CompositeEliminated {
		t.Fatalf("hard provider constraint did not eliminate incompatible composite: %+v", incompatible)
	}

	operational := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "restore desired index state",
		Realization:       "postgres-index-reconciliation",
		Surface:           "error",
		OutcomeClass:      providerOutcomeOperationalRecoveryProbe,
		Owner:             "runtime-operations",
		HardProvider:      true,
	})
	if operational.CompositeEliminated || operational.RealizationEliminated || !operational.Unresolved {
		t.Fatalf("hard provider requirement converted operational recovery into incompatibility: %+v", operational)
	}
}

func TestProviderOutcomeSupportedRealizationDoesNotAcceptSemanticMeaning(t *testing.T) {
	result := classifyProviderOutcomeContractionProbe(providerOutcomeClassificationProbe{
		SemanticCandidate: "remove ordinary table",
		Realization:       "postgres-current-drop-table",
		Surface:           "success",
		OutcomeClass:      providerOutcomeSupportedRealizationProbe,
		Owner:             "provider-binding",
	})

	if !result.SemanticCandidateRetained {
		t.Fatal("supported realization unexpectedly removed semantic candidate")
	}
	if result.SemanticAccepted {
		t.Fatal("provider support was promoted to semantic acceptance")
	}
}

func TestProviderOutcomeGenericErrorSurfaceIsInsufficientForContraction(t *testing.T) {
	cases := []providerOutcomeClassificationProbe{
		{
			SemanticCandidate: "remove table",
			Realization:       "postgres-drop-table",
			Surface:           "error",
			OutcomeClass:      providerOutcomeMissingEvidenceProbe,
			Owner:             "provider-observation",
		},
		{
			SemanticCandidate: "remove table",
			Realization:       "postgres-drop-table",
			Surface:           "error",
			OutcomeClass:      providerOutcomeObservedIncompatibleProbe,
			Owner:             "provider-safety",
		},
		{
			SemanticCandidate: "change primary key",
			Realization:       "postgres-change-primary-key",
			Surface:           "error",
			OutcomeClass:      providerOutcomeStructurallyUnsupported,
			Owner:             "provider-capability",
		},
		{
			SemanticCandidate: "restore index state",
			Realization:       "postgres-index-reconciliation",
			Surface:           "error",
			OutcomeClass:      providerOutcomeOperationalRecoveryProbe,
			Owner:             "runtime-operations",
		},
	}

	results := make([]providerOutcomeContractionProbe, 0, len(cases))
	for _, tc := range cases {
		results = append(results, classifyProviderOutcomeContractionProbe(tc))
	}

	if !results[0].Unresolved || results[0].RealizationEliminated {
		t.Fatalf("missing evidence classification lost: %+v", results[0])
	}
	if !results[1].RealizationEliminated || results[1].Unresolved {
		t.Fatalf("observed incompatibility classification lost: %+v", results[1])
	}
	if !results[2].RealizationEliminated || results[2].Owner != "provider-capability" {
		t.Fatalf("structural unsupported classification lost: %+v", results[2])
	}
	if !results[3].Unresolved || results[3].RealizationEliminated || results[3].Owner != "runtime-operations" {
		t.Fatalf("operational recovery classification lost: %+v", results[3])
	}
}
