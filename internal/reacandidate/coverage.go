package reacandidate

import (
	"sort"
	"strings"
)

// Three-element coverage statuses.
const (
	ElementSupported   = "supported"
	ElementUnsupported = "unsupported"
)

// ElementCoverage is one row of "can this attempt support a candidate for this
// element of the three-element model?".
//
// It is a gap statement, not a candidate. The three-element model has Data,
// Function and Business; an external static analyzer of JavaScript sources can
// observe assets, modules and directed relations, and that is data-shaped
// evidence. Naming a Function or a Business meaning from it would be inference
// the attempt does not support, so those rows say so instead of producing a
// plausible-looking element.
type ElementCoverage struct {
	Element     string   `json:"element"`
	Status      string   `json:"status"`
	Candidates  int      `json:"candidates"`
	SupportedBy []string `json:"supportedBy,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	WouldNeed   []string `json:"wouldNeed,omitempty"`
}

// ThreeElementCoverage is the per-element answer for one attempt.
type ThreeElementCoverage struct {
	SchemaVersion   string            `json:"schemaVersion"`
	Effect          string            `json:"effect"`
	Elements        []ElementCoverage `json:"elements"`
	AuthorityLimits []string          `json:"authorityLimits"`
}

// CoverageAuthorityLimits states what the coverage table does not establish.
func CoverageAuthorityLimits() []string {
	return []string{
		"coverage-is-a-gap-statement-not-a-candidate",
		"unsupported-element-is-not-an-absence-of-that-element-in-the-system",
		"unsupported-element-is-not-a-failure-of-the-external-attempt",
		"business-meaning-is-never-derivable-from-external-structure-alone",
		"coverage-does-not-accept-or-authorize-anything",
	}
}

// EvaluateThreeElementCoverage reports, per element, whether the attempt's
// facts support a candidate. Data is supported by observations and directed
// relations; Function and Business stay unsupported with the reason and what
// evidence would be needed, so a reviewer is never left to guess whether the
// silence was deliberate.
func EvaluateThreeElementCoverage(set CandidateSet) ThreeElementCoverage {
	dataCount := countCandidatesOfKind(set, CandidateKindDataElement)
	relationCount := countCandidatesOfKind(set, CandidateKindCrossReference)

	coverage := ThreeElementCoverage{
		SchemaVersion:   "tei.rea.three-element-coverage/v0",
		Effect:          EffectReadOnly,
		AuthorityLimits: CoverageAuthorityLimits(),
	}

	data := ElementCoverage{Element: "data", Status: ElementUnsupported, Candidates: dataCount + relationCount}
	if dataCount > 0 {
		data.Status = ElementSupported
		data.SupportedBy = append(data.SupportedBy, "source-backed-node-observations")
	}
	if relationCount > 0 {
		if data.Status != ElementSupported {
			data.Status = ElementSupported
		}
		data.SupportedBy = append(data.SupportedBy, "directed-external-relations")
	}
	if data.Status == ElementUnsupported {
		data.Reason = "the attempt reported no observed node and no directed relation"
	}
	data.WouldNeed = sortStrings(data.WouldNeed)
	coverage.Elements = append(coverage.Elements, data)

	function := ElementCoverage{
		Element:   "function",
		Status:    ElementUnsupported,
		Reason:    "the external facts name assets, modules and directed relations; none of them is an operation with declared inputs, outputs or effects",
		WouldNeed: []string{"an observed operation or entry point with declared inputs and outputs", "an observed effect or side effect the operation is responsible for"},
	}
	// A relation label such as "exposes" or "loads" is a static relationship,
	// not an operation. Record that it was considered so the reviewer sees the
	// decision rather than an unexplained blank.
	if labels := observedRelationLabels(set); len(labels) > 0 {
		function.Reason = function.Reason + " (relation labels observed: " + strings.Join(labels, ", ") + ")"
	}
	function.WouldNeed = sortStrings(function.WouldNeed)
	coverage.Elements = append(coverage.Elements, function)

	business := ElementCoverage{
		Element:   "business",
		Status:    ElementUnsupported,
		Reason:    "business meaning is an Owner decision; no external structural fact can stand in for it",
		WouldNeed: []string{"an Owner acceptance of the business meaning behind the candidate", "a declared actor, activity or rule that the system owner names"},
	}
	business.WouldNeed = sortStrings(business.WouldNeed)
	coverage.Elements = append(coverage.Elements, business)

	sort.Slice(coverage.Elements, func(i, j int) bool {
		return elementRank(coverage.Elements[i].Element) < elementRank(coverage.Elements[j].Element)
	})
	return coverage
}

func elementRank(element string) int {
	switch element {
	case "data":
		return 0
	case "function":
		return 1
	case "business":
		return 2
	}
	return 3
}

// countCandidatesOfKind counts the candidates of one kind. It is a production
// helper (the package tests use their own slice-returning version).
func countCandidatesOfKind(set CandidateSet, kind string) int {
	count := 0
	for _, candidate := range set.Candidates {
		if candidate.Kind == kind {
			count++
		}
	}
	return count
}

// observedRelationLabels returns the distinct nonempty relation labels the
// attempt reported, so the Function row can name what was considered.
func observedRelationLabels(set CandidateSet) []string {
	var labels []string
	for _, candidate := range set.Candidates {
		if candidate.Kind != CandidateKindCrossReference {
			continue
		}
		labels = append(labels, candidate.Facts.ObservedKind)
	}
	return uniqueStrings(labels)
}
