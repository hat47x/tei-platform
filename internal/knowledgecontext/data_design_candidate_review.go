package knowledgecontext

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const (
	dataDesignCandidateReviewProjectionRevision = "tei.knowledge.data-design-candidate-review/v1"

	DataDesignCandidateKindBusinessIdentity      = "business-identity"
	DataDesignCandidateKindFunctionalDependency = "functional-dependency"
)

type DataDesignCandidateReviewCandidate struct {
	Kind                           string                      `json:"kind"`
	Fingerprint                    string                      `json:"fingerprint"`
	EvidenceRequirementFingerprint string                      `json:"evidenceRequirementFingerprint"`
	Relation                       core.CanonicalID            `json:"relation"`
	Terms                          []DataDesignDeterminantTerm `json:"terms"`
	TargetOccurrence               core.CanonicalID            `json:"targetOccurrence"`
	BasisConstraint                core.CanonicalID            `json:"basisConstraint"`
	Scope                          string                      `json:"scope"`
	Nature                         string                      `json:"nature"`
}

type DataDesignCandidateReviewRequirement struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type DataDesignCandidateReviewPacket struct {
	ProjectionRevision                    string                                `json:"projectionRevision"`
	Fingerprint                           string                                `json:"fingerprint"`
	Subject                               subject.Subject                       `json:"subject"`
	DataDesignProjectionRevision          string                                `json:"dataDesignProjectionRevision"`
	DataDesignFingerprint                 string                                `json:"dataDesignFingerprint"`
	DataDesignResolutionStatus            string                                `json:"dataDesignResolutionStatus"`
	DataDesignEvidenceCoverageStatus      string                                `json:"dataDesignEvidenceCoverageStatus"`
	Candidate                             DataDesignCandidateReviewCandidate     `json:"candidate"`
	SourceEvidence                        []DataDesignEvidence                    `json:"sourceEvidence,omitempty"`
	SourceEvidenceMissing                 []subject.Subject                     `json:"sourceEvidenceMissing,omitempty"`
	BusinessEvidenceRequirements          []DataDesignCandidateReviewRequirement `json:"businessEvidenceRequirements"`
	ReviewStatus                          string                                `json:"reviewStatus"`
	FirstClassAuthoringContractStatus     string                                `json:"firstClassAuthoringContractStatus"`
	FirstClassAuthoringContractReasons    []string                              `json:"firstClassAuthoringContractReasons"`
	ReviewSequence                        []string                              `json:"reviewSequence"`
	AuthorityLimits                       []string                              `json:"authorityLimits"`
}

func BuildDataDesignCandidateReview(
	ctx Context,
	candidateKind, candidateFingerprint, expectedEvidenceRequirementFingerprint string,
) (DataDesignCandidateReviewPacket, error) {
	if ctx.DataDesign == nil {
		return DataDesignCandidateReviewPacket{}, fmt.Errorf(
			"data-design candidate review requires Data Design Context for %s:%s",
			ctx.Subject.Kind,
			ctx.Subject.ID,
		)
	}
	kind := strings.TrimSpace(candidateKind)
	fingerprint := strings.TrimSpace(candidateFingerprint)
	if kind == "" || fingerprint == "" {
		return DataDesignCandidateReviewPacket{}, fmt.Errorf(
			"data-design candidate review requires candidate kind and fingerprint",
		)
	}

	candidate, evidenceNeeded, relevant, err := resolveDataDesignReviewCandidate(
		ctx.DataDesign,
		kind,
		fingerprint,
	)
	if err != nil {
		return DataDesignCandidateReviewPacket{}, err
	}
	if expected := strings.TrimSpace(expectedEvidenceRequirementFingerprint); expected != "" &&
		expected != candidate.EvidenceRequirementFingerprint {
		return DataDesignCandidateReviewPacket{}, fmt.Errorf(
			"data-design candidate evidence requirement fingerprint mismatch: expected=%s current=%s",
			expected,
			candidate.EvidenceRequirementFingerprint,
		)
	}

	sourceEvidence, sourceMissing := selectDataDesignReviewEvidence(ctx.DataDesign, relevant)
	requirements := make([]DataDesignCandidateReviewRequirement, 0, len(evidenceNeeded))
	for _, requirement := range normalizedDataDesignStrings(evidenceNeeded) {
		requirements = append(requirements, DataDesignCandidateReviewRequirement{
			ID:     requirement,
			Status: "unresolved",
		})
	}

	reviewStatus := "evidence-required"
	if ctx.DataDesign.ResolutionStatus != "resolved" || len(sourceMissing) != 0 {
		reviewStatus = "blocked-context-unresolved"
	}

	packet := DataDesignCandidateReviewPacket{
		ProjectionRevision:               dataDesignCandidateReviewProjectionRevision,
		Subject:                          ctx.Subject,
		DataDesignProjectionRevision:     ctx.DataDesign.ProjectionRevision,
		DataDesignFingerprint:            ctx.DataDesign.Fingerprint,
		DataDesignResolutionStatus:       ctx.DataDesign.ResolutionStatus,
		DataDesignEvidenceCoverageStatus: ctx.DataDesign.EvidenceCoverageStatus,
		Candidate:                        candidate,
		SourceEvidence:                   sourceEvidence,
		SourceEvidenceMissing:            sourceMissing,
		BusinessEvidenceRequirements:     requirements,
		ReviewStatus:                     reviewStatus,
		FirstClassAuthoringContractStatus: "unresolved",
		FirstClassAuthoringContractReasons: dataDesignFirstClassAuthoringContractReasons(kind),
		ReviewSequence: []string{
			"verify-current-data-design-fingerprint",
			"review-derived-candidate-shape-and-basis-constraint",
			"review-authoritative-source-evidence",
			"resolve-business-evidence-requirements-without-inference",
			"decide-whether-first-class-authoring-contract-is-required",
			"route-authoritative-change-through-shared-acceptance-only-after-authoring-contract-exists",
		},
		AuthorityLimits: []string{
			"candidate-review-packet-is-derived-not-authority",
			"candidate-fingerprint-is-identity-not-acceptance",
			"evidence-requirement-fingerprint-is-review-contract-not-evidence",
			"source-evidence-proves-traceability-not-business-meaning",
			"unresolved-business-evidence-requirement-must-not-be-inferred-as-satisfied",
			"candidate-review-does-not-create-first-class-business-identity-or-functional-dependency",
			"semantic-extension-reference-is-not-current-lossless-authoring-contract",
			"candidate-review-does-not-grant-write-runtime-or-decision-authority",
			"candidate-review-does-not-replace-shared-acceptance",
		},
	}
	packet.Fingerprint = dataDesignCandidateReviewFingerprint(packet)
	return packet, nil
}

func dataDesignFirstClassAuthoringContractReasons(kind string) []string {
	common := []string{
		"semantic-extension-v0-is-reference-not-production-authoring-contract",
		"authoritative-declaration-source-and-namespace-authority-not-selected",
	}
	switch kind {
	case DataDesignCandidateKindBusinessIdentity:
		return append([]string{
			"current-core-has-no-first-class-business-identity-authoring-contract",
			"semantic-extension-v0-constraint-cannot-losslessly-encode-composite-identity-components-and-scope",
		}, common...)
	case DataDesignCandidateKindFunctionalDependency:
		return append([]string{
			"current-core-has-no-first-class-functional-dependency-authoring-contract",
			"semantic-extension-v0-constraint-cannot-losslessly-encode-determinants-dependent-occurrence-and-scope",
		}, common...)
	default:
		return append([]string(nil), common...)
	}
}

func resolveDataDesignReviewCandidate(
	ctx *DataDesignContext,
	kind, fingerprint string,
) (
	DataDesignCandidateReviewCandidate,
	[]string,
	map[subject.Subject]struct{},
	error,
) {
	switch kind {
	case DataDesignCandidateKindBusinessIdentity:
		for _, value := range ctx.BusinessIdentityCandidates {
			if value.Fingerprint != fingerprint {
				continue
			}
			relevant := dataDesignReviewSubjects(
				value.Relation,
				value.BasisConstraint,
				value.TargetOccurrence,
				value.Components,
			)
			return DataDesignCandidateReviewCandidate{
				Kind:                           kind,
				Fingerprint:                    value.Fingerprint,
				EvidenceRequirementFingerprint: value.EvidenceRequirementFingerprint,
				Relation:                       value.Relation,
				Terms:                          append([]DataDesignDeterminantTerm(nil), value.Components...),
				TargetOccurrence:               value.TargetOccurrence,
				BasisConstraint:                value.BasisConstraint,
				Scope:                          value.Scope,
				Nature:                         value.Nature,
			}, append([]string(nil), value.EvidenceNeeded...), relevant, nil
		}
	case DataDesignCandidateKindFunctionalDependency:
		for _, value := range ctx.FunctionalDependencyCandidates {
			if value.Fingerprint != fingerprint {
				continue
			}
			relevant := dataDesignReviewSubjects(
				value.Relation,
				value.BasisConstraint,
				value.DependentOccurrence,
				value.Determinants,
			)
			return DataDesignCandidateReviewCandidate{
				Kind:                           kind,
				Fingerprint:                    value.Fingerprint,
				EvidenceRequirementFingerprint: value.EvidenceRequirementFingerprint,
				Relation:                       value.Relation,
				Terms:                          append([]DataDesignDeterminantTerm(nil), value.Determinants...),
				TargetOccurrence:               value.DependentOccurrence,
				BasisConstraint:                value.BasisConstraint,
				Scope:                          value.Scope,
				Nature:                         value.Nature,
			}, append([]string(nil), value.EvidenceNeeded...), relevant, nil
		}
	default:
		return DataDesignCandidateReviewCandidate{}, nil, nil, fmt.Errorf(
			"unsupported data-design candidate kind %q",
			kind,
		)
	}
	return DataDesignCandidateReviewCandidate{}, nil, nil, fmt.Errorf(
		"data-design candidate %s %s is not present in current Context",
		kind,
		fingerprint,
	)
}

func dataDesignReviewSubjects(
	relation, basisConstraint, target core.CanonicalID,
	terms []DataDesignDeterminantTerm,
) map[subject.Subject]struct{} {
	out := map[subject.Subject]struct{}{
		{Kind: "semantic-relation", ID: string(relation)}:       {},
		{Kind: "semantic-validation", ID: string(basisConstraint)}: {},
		{Kind: "semantic-data", ID: string(target)}:             {},
	}
	for _, term := range terms {
		if term.Data != "" {
			out[subject.Subject{Kind: "semantic-data", ID: string(term.Data)}] = struct{}{}
		}
		if term.Attribute != "" {
			out[subject.Subject{Kind: "semantic-attribute", ID: string(term.Attribute)}] = struct{}{}
		}
	}
	return out
}

func selectDataDesignReviewEvidence(
	ctx *DataDesignContext,
	relevant map[subject.Subject]struct{},
) ([]DataDesignEvidence, []subject.Subject) {
	var evidence []DataDesignEvidence
	for _, item := range ctx.Evidence {
		if _, ok := relevant[item.Subject]; ok {
			evidence = append(evidence, item)
		}
	}
	var missing []subject.Subject
	for item := range relevant {
		found := false
		for _, evidenceItem := range evidence {
			if evidenceItem.Subject == item {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, item)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Kind != missing[j].Kind {
			return missing[i].Kind < missing[j].Kind
		}
		return missing[i].ID < missing[j].ID
	})
	return evidence, missing
}

func dataDesignCandidateReviewFingerprint(value DataDesignCandidateReviewPacket) string {
	value.Fingerprint = ""
	encoded, err := json.Marshal(value)
	if err != nil {
		panic("Data Design candidate review contains non-JSON state: " + err.Error())
	}
	return sourceindex.Fingerprint(encoded)
}
