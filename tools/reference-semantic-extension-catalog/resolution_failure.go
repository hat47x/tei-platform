package main

import (
	"errors"
	"fmt"
)

const (
	resolutionFailureInvalidRequirement      = "invalid-requirement"
	resolutionFailureIncompatibleMajors      = "incompatible-major-constraints"
	resolutionFailureSelectedVersionConflict = "selected-version-conflict"
	resolutionFailureNoCompatibleCandidate   = "no-compatible-candidate"
	resolutionFailureDependencyCycle         = "dependency-cycle"
	resolutionFailureCandidatesExhausted     = "candidate-set-exhausted"
	resolutionFailureSearchBudgetExceeded    = "search-budget-exceeded"
	resolutionFailureUnknown                 = "resolution-error"
)

type SemanticResolutionCandidateFailure struct {
	Version string                     `json:"version"`
	Failure *SemanticResolutionFailure `json:"failure"`
}

type SemanticResolutionFailure struct {
	Code                 string                               `json:"code"`
	Extension            string                               `json:"extension,omitempty"`
	RequiredMajor        *int                                 `json:"requiredMajor,omitempty"`
	RequiredMajors       []int                                `json:"requiredMajors,omitempty"`
	SelectedVersion      string                               `json:"selectedVersion,omitempty"`
	Cycle                []string                             `json:"cycle,omitempty"`
	MaxCandidateAttempts int                                  `json:"maxCandidateAttempts,omitempty"`
	CandidateFailures    []SemanticResolutionCandidateFailure `json:"candidateFailures,omitempty"`
	Detail               string                               `json:"detail"`

	sentinel error
}

func (e *SemanticResolutionFailure) Error() string {
	if e == nil {
		return ""
	}
	return e.Detail
}

func (e *SemanticResolutionFailure) Is(target error) bool {
	return e != nil && e.sentinel != nil && target == e.sentinel
}

func semanticResolutionFailureFromError(err error) *SemanticResolutionFailure {
	if err == nil {
		return nil
	}
	var failure *SemanticResolutionFailure
	if errors.As(err, &failure) {
		return failure
	}
	return &SemanticResolutionFailure{
		Code:   resolutionFailureUnknown,
		Detail: err.Error(),
	}
}

func newIncompatibleMajorsFailure(
	extension string,
	majors []int,
) *SemanticResolutionFailure {
	values := append([]int(nil), majors...)
	return &SemanticResolutionFailure{
		Code:           resolutionFailureIncompatibleMajors,
		Extension:      extension,
		RequiredMajors: values,
		Detail:         fmt.Sprintf("extension %s requires incompatible majors %v", extension, values),
	}
}

func newNoCompatibleCandidateFailure(
	extension string,
	major int,
) *SemanticResolutionFailure {
	requiredMajor := major
	return &SemanticResolutionFailure{
		Code:          resolutionFailureNoCompatibleCandidate,
		Extension:     extension,
		RequiredMajor: &requiredMajor,
		Detail: fmt.Sprintf(
			"no semantic extension %s compatible with major %d",
			extension,
			major,
		),
	}
}

func newSelectedVersionConflictFailure(
	extension string,
	selectedVersion string,
	major int,
) *SemanticResolutionFailure {
	requiredMajor := major
	return &SemanticResolutionFailure{
		Code:            resolutionFailureSelectedVersionConflict,
		Extension:       extension,
		RequiredMajor:   &requiredMajor,
		SelectedVersion: selectedVersion,
		Detail: fmt.Sprintf(
			"extension %s already resolved to %s, incompatible with required major %d",
			extension,
			selectedVersion,
			major,
		),
	}
}

func newDependencyCycleFailure(
	extension string,
	cycle []string,
) *SemanticResolutionFailure {
	path := append([]string(nil), cycle...)
	return &SemanticResolutionFailure{
		Code:      resolutionFailureDependencyCycle,
		Extension: extension,
		Cycle:     path,
		Detail:    fmt.Sprintf("extension dependency cycle at %s", extension),
	}
}

func newSearchBudgetExceededFailure(max int) *SemanticResolutionFailure {
	return &SemanticResolutionFailure{
		Code:                 resolutionFailureSearchBudgetExceeded,
		MaxCandidateAttempts: max,
		Detail: fmt.Sprintf(
			"%s: maxCandidateAttempts=%d",
			errSemanticResolutionBudgetExceeded,
			max,
		),
		sentinel: errSemanticResolutionBudgetExceeded,
	}
}

func newCandidateSetExhaustedFailure(
	extension string,
	major int,
	candidates []SemanticResolutionCandidateFailure,
) *SemanticResolutionFailure {
	requiredMajor := major
	failure := &SemanticResolutionFailure{
		Code:              resolutionFailureCandidatesExhausted,
		Extension:         extension,
		RequiredMajor:     &requiredMajor,
		CandidateFailures: append([]SemanticResolutionCandidateFailure(nil), candidates...),
	}
	if len(candidates) == 0 || candidates[len(candidates)-1].Failure == nil {
		failure.Detail = fmt.Sprintf(
			"no globally compatible semantic extension %s for major %d",
			extension,
			major,
		)
		return failure
	}
	failure.Detail = fmt.Sprintf(
		"no globally compatible semantic extension %s for major %d: %s",
		extension,
		major,
		candidates[len(candidates)-1].Failure.Error(),
	)
	return failure
}
