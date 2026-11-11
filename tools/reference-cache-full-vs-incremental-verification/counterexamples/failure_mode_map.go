package counterexamples

import "tei/internal/failure"

// FailureModeMapping records how a counterexample scenario maps to the unified
// failure contract kinds and codes.
type FailureModeMapping struct {
	ScenarioID   string
	FailureKind  failure.Kind
	FailureCodes []string
	Notes        []string
}

// FailureModeMappings returns the recommended mapping from counterexample
// scenarios to TEI failure kinds.
func FailureModeMappings() []FailureModeMapping {
	return []FailureModeMapping{
		{
			ScenarioID:   "CE-01",
			FailureKind:  failure.TechnicalFailure,
			FailureCodes: []string{"CACHE_INTEGRITY_FAILED", "STALE_ARTIFACT"},
			Notes:        []string{"Old artifact pollution should surface as a technical failure during cache readback."},
		},
		{
			ScenarioID:   "CE-02",
			FailureKind:  failure.BusinessRejection,
			FailureCodes: []string{"INCOMPLETE_DEPENDENCY_GRAPH"},
			Notes:        []string{"Incomplete dependencies are a business-level invalid state, not a transient fault."},
		},
		{
			ScenarioID:   "CE-03",
			FailureKind:  failure.TechnicalFailure,
			FailureCodes: []string{"CACHE_CONTAMINATION"},
			Notes:        []string{"Cross-tenant contamination indicates a provider/host boundary fault."},
		},
		{
			ScenarioID:   "CE-04",
			FailureKind:  failure.BusinessRejection,
			FailureCodes: []string{"STALE_SOURCE_INSPECTION"},
			Notes:        []string{"Source inspection after change should block reuse under the source freshness rule."},
		},
		{
			ScenarioID:   "CE-05",
			FailureKind:  failure.PartialSuccess,
			FailureCodes: []string{"INTERMEDIATE_FAILURE", "PARTIAL_ARTIFACT_REUSE"},
			Notes:        []string{"Intermediate failures may leave partial effects; cache reuse must not hide them."},
		},
		{
			ScenarioID:   "CE-06",
			FailureKind:  failure.TechnicalFailure,
			FailureCodes: []string{"TOOLCHAIN_MISMATCH", "ARTIFACT_REUSE_REJECTED"},
			Notes:        []string{"Reusing a toolchain artifact produced by a different toolchain is a technical integrity failure."},
		},
	}
}
