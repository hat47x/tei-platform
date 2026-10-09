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
	ID                string            `json:"id"`
	Status            string            `json:"status"`
	SuggestedSubjects []subject.Subject `json:"suggestedSubjects,omitempty"`
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
	SourceEvidenceStatus                  string                                `json:"sourceEvidenceStatus"`
	ReviewBlockingAmbiguities             []DataDesignAmbiguity                  `json:"reviewBlockingAmbiguities,omitempty"`
	CandidateAmbiguityStatus              string                                `json:"candidateAmbiguityStatus"`
	BusinessEvidenceRequirements          []DataDesignCandidateReviewRequirement `json:"businessEvidenceRequirements"`
	BusinessEvidenceStatus                string                                `json:"businessEvidenceStatus"`
	ReviewStatus                          string                                `json:"reviewStatus"`
	PromotionStatus                       string                                `json:"promotionStatus"`
	PromotionBlockingReasons              []string                              `json:"promotionBlockingReasons"`
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
	if ctx.DataDesign.Fingerprint == "" ||
		dataDesignFingerprint(ctx.DataDesign) != ctx.DataDesign.Fingerprint {
		return DataDesignCandidateReviewPacket{}, fmt.Errorf(
			"data-design candidate review requires an unmodified current Data Design Context",
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
	blockingAmbiguities := selectDataDesignReviewAmbiguities(ctx.DataDesign, relevant)
	requirements := make([]DataDesignCandidateReviewRequirement, 0, len(evidenceNeeded))
	for _, requirement := range normalizedDataDesignStrings(evidenceNeeded) {
		requirements = append(requirements, DataDesignCandidateReviewRequirement{
			ID:                requirement,
			Status:            "unresolved",
			SuggestedSubjects: dataDesignEvidenceSearchSubjects(candidate, requirement),
		})
	}

	sourceEvidenceStatus := "complete"
	if len(sourceMissing) != 0 {
		sourceEvidenceStatus = "incomplete"
	}
	candidateAmbiguityStatus := "clear"
	if len(blockingAmbiguities) != 0 {
		candidateAmbiguityStatus = "blocked"
	}
	reviewStatus := "evidence-required"
	if sourceEvidenceStatus != "complete" || candidateAmbiguityStatus != "clear" {
		reviewStatus = "blocked-context-unresolved"
	}
	promotionBlockingReasons := []string{
		"business-evidence-unresolved",
		"first-class-authoring-contract-unresolved",
	}
	if sourceEvidenceStatus != "complete" {
		promotionBlockingReasons = append(
			promotionBlockingReasons,
			"candidate-source-evidence-incomplete",
		)
	}
	if candidateAmbiguityStatus != "clear" {
		promotionBlockingReasons = append(
			promotionBlockingReasons,
			"candidate-local-ambiguity-unresolved",
		)
	}
	sort.Strings(promotionBlockingReasons)

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
		SourceEvidenceStatus:             sourceEvidenceStatus,
		ReviewBlockingAmbiguities:        blockingAmbiguities,
		CandidateAmbiguityStatus:         candidateAmbiguityStatus,
		BusinessEvidenceRequirements:     requirements,
		BusinessEvidenceStatus:           "unresolved",
		ReviewStatus:                     reviewStatus,
		PromotionStatus:                  "blocked",
		PromotionBlockingReasons:         promotionBlockingReasons,
		FirstClassAuthoringContractStatus: "unresolved",
		FirstClassAuthoringContractReasons: dataDesignFirstClassAuthoringContractReasons(kind),
		ReviewSequence: []string{
			"rebuild-current-data-design-context-and-verify-fingerprint",
			"review-derived-candidate-shape-and-basis-constraint",
			"review-authoritative-source-evidence",
			"resolve-business-evidence-requirements-without-inference",
			"review-first-class-authoring-contract-gaps",
			"select-core-or-semantic-extension-authoring-route-with-evidence",
			"require-lossless-authoring-contract-before-promoting-derived-candidate",
			"route-authoritative-change-through-shared-acceptance-only-after-authoring-contract-exists",
		},
		AuthorityLimits: []string{
			"candidate-review-packet-is-derived-not-authority",
			"candidate-fingerprint-is-identity-not-acceptance",
			"data-design-context-fingerprint-is-snapshot-identity-not-source-freshness-proof",
			"evidence-requirement-fingerprint-is-review-contract-not-evidence",
			"source-evidence-proves-traceability-not-business-meaning",
			"candidate-review-blocking-ambiguity-is-candidate-local-not-global-context-certainty",
			"review-status-is-review-readiness-not-promotion-authority",
			"promotion-status-is-derived-readiness-not-acceptance-or-write-authority",
			"unresolved-business-evidence-requirement-must-not-be-inferred-as-satisfied",
			"suggested-evidence-subject-is-search-start-not-evidence-or-authority",
			"candidate-review-does-not-create-first-class-business-identity-or-functional-dependency",
			"semantic-extension-reference-is-not-current-lossless-authoring-contract",
			"candidate-review-does-not-grant-write-runtime-or-decision-authority",
			"candidate-review-does-not-replace-shared-acceptance",
		},
	}
	packet.Fingerprint = dataDesignCandidateReviewFingerprint(packet)
	return packet, nil
}

func dataDesignEvidenceSearchSubjects(
	candidate DataDesignCandidateReviewCandidate,
	requirement string,
) []subject.Subject {
	sourceData := core.CanonicalID("")
	for _, term := range candidate.Terms {
		if term.Kind == "relation-source-occurrence" && term.Data != "" {
			sourceData = term.Data
			break
		}
	}
	relationSubject := subject.Subject{Kind: "semantic-relation", ID: string(candidate.Relation)}
	targetSubject := subject.Subject{Kind: "semantic-data", ID: string(candidate.TargetOccurrence)}
	sourceSubject := subject.Subject{Kind: "semantic-data", ID: string(sourceData)}

	var out []subject.Subject
	switch requirement {
	case "source-occurrence-business-identifier":
		if sourceData != "" {
			out = append(out, sourceSubject)
		}
	case "relation-business-meaning":
		out = append(out, relationSubject)
	case "target-membership-exclusivity":
		out = append(out, relationSubject, targetSubject)
	case "identifier-stability", "history-version-identity", "identifier-reuse-policy":
		out = append(out, targetSubject)
	case "temporal-validity", "temporal-dependency-scope":
		out = append(out, relationSubject, targetSubject)
	default:
		out = append(out, relationSubject, targetSubject)
		if sourceData != "" {
			out = append(out, sourceSubject)
		}
	}
	return normalizeDataDesignReviewSubjects(out)
}

func normalizeDataDesignReviewSubjects(values []subject.Subject) []subject.Subject {
	seen := make(map[subject.Subject]struct{}, len(values))
	out := make([]subject.Subject, 0, len(values))
	for _, value := range values {
		if value.Kind == "" || value.ID == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func dataDesignFirstClassAuthoringContractReasons(kind string) []string {
	common := []string{
		"first-class-promotion-route-core-vs-semantic-extension-not-selected",
		"semantic-extension-route-is-not-accepted-production-authoring-contract",
		"authoritative-declaration-source-not-selected",
		"semantic-extension-namespace-authority-not-selected",
	}
	switch kind {
	case DataDesignCandidateKindBusinessIdentity:
		return append([]string{
			"current-core-has-no-first-class-business-identity-authoring-contract",
			"semantic-extension-route-cannot-yet-losslessly-author-composite-identity-components-and-scope",
		}, common...)
	case DataDesignCandidateKindFunctionalDependency:
		return append([]string{
			"current-core-has-no-first-class-functional-dependency-authoring-contract",
			"semantic-extension-route-cannot-yet-losslessly-author-determinants-dependent-occurrence-and-scope",
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
			if businessIdentityCandidateFingerprint(value) != value.Fingerprint {
				return DataDesignCandidateReviewCandidate{}, nil, nil, fmt.Errorf(
					"data-design business identity candidate fingerprint does not match current candidate content",
				)
			}
			if dataDesignCandidateEvidenceRequirementFingerprint(
				"tei.knowledge.data-design.business-identity-candidate-evidence/v1",
				value.Fingerprint,
				value.EvidenceNeeded,
			) != value.EvidenceRequirementFingerprint {
				return DataDesignCandidateReviewCandidate{}, nil, nil, fmt.Errorf(
					"data-design business identity Evidence requirement fingerprint does not match current requirement content",
				)
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
			if functionalDependencyCandidateFingerprint(value) != value.Fingerprint {
				return DataDesignCandidateReviewCandidate{}, nil, nil, fmt.Errorf(
					"data-design functional dependency candidate fingerprint does not match current candidate content",
				)
			}
			if dataDesignCandidateEvidenceRequirementFingerprint(
				"tei.knowledge.data-design.functional-dependency-candidate-evidence/v1",
				value.Fingerprint,
				value.EvidenceNeeded,
			) != value.EvidenceRequirementFingerprint {
				return DataDesignCandidateReviewCandidate{}, nil, nil, fmt.Errorf(
					"data-design functional dependency Evidence requirement fingerprint does not match current requirement content",
				)
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

func selectDataDesignReviewAmbiguities(
	ctx *DataDesignContext,
	relevant map[subject.Subject]struct{},
) []DataDesignAmbiguity {
	var out []DataDesignAmbiguity
	for _, ambiguity := range ctx.Ambiguities {
		kind := ""
		switch ambiguity.Kind {
		case "duplicate-semantic-data":
			kind = "semantic-data"
		case "duplicate-semantic-attribute":
			kind = "semantic-attribute"
		case "duplicate-semantic-relation":
			kind = "semantic-relation"
		case "duplicate-semantic-validation":
			kind = "semantic-validation"
		case "duplicate-persistence-mapping":
			kind = "persistence"
		}
		if kind == "" {
			out = append(out, ambiguity)
			continue
		}
		if _, ok := relevant[subject.Subject{Kind: kind, ID: string(ambiguity.ID)}]; ok {
			out = append(out, ambiguity)
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
