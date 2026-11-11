package reacandidate

import (
	"sort"
	"strings"
)

// maxUnresolvedSummary bounds the aggregated unresolved list so a hostile or
// very large attempt cannot turn a review packet into an unreadable dump.
const maxUnresolvedSummary = 256

// ReviewPacket is the single reviewable output of this package: the external
// attempt, its structure candidates, the optional read-only binding, the
// optional adoption preview, the optional before/after comparison, the
// carrier Fit/Gap audit, and everything still unresolved.
//
// The packet is a review surface. It contains no acceptance, no write, and no
// authority; AcceptedMeaning is false by construction and there is no code
// path in this package that sets it true.
type ReviewPacket struct {
	SchemaVersion           string                  `json:"schemaVersion"`
	Effect                  string                  `json:"effect"`
	Attempt                 AttemptProvenance       `json:"attempt"`
	CandidateSetFingerprint string                  `json:"candidateSetFingerprint"`
	Qualification           string                  `json:"qualification"`
	Candidates              []StructureCandidate    `json:"candidates"`
	BindingAttempted        bool                    `json:"bindingAttempted"`
	Bindings                []CandidateBinding      `json:"bindings,omitempty"`
	AdoptionPreviewed       bool                    `json:"adoptionPreviewed"`
	Adoption                *AdoptionPreview        `json:"adoption,omitempty"`
	ComparedWithBefore      bool                    `json:"comparedWithBefore"`
	Comparison              *CandidateSetComparison `json:"comparison,omitempty"`
	Carrier                 CarrierAudit            `json:"carrier"`
	Unresolved              []string                `json:"unresolved"`
	Unknowns                []ExternalUnknown       `json:"unknowns,omitempty"`
	Notes                   []string                `json:"notes,omitempty"`
	ReviewSequence          []string                `json:"reviewSequence"`
	AcceptedMeaning         bool                    `json:"acceptedMeaning"`
	OwnerDecision           string                  `json:"ownerDecision"`
	AuthorityLimits         []string                `json:"authorityLimits"`
}

// BuildReviewPacket assembles the reviewable output. binding, adoption and
// comparison are optional; a nil section is reported as not attempted rather
// than silently omitted from the summary.
func BuildReviewPacket(set CandidateSet, binding *BindingResult, adoption *AdoptionPreview, comparison *CandidateSetComparison) ReviewPacket {
	packet := ReviewPacket{
		SchemaVersion:           candidateReviewSchemaVers,
		Effect:                  EffectReadOnly,
		Attempt:                 set.Attempt,
		CandidateSetFingerprint: set.Fingerprint,
		Qualification:           set.Qualification,
		Candidates:              set.Candidates,
		Carrier:                 AuditCarrierFit(set),
		Unknowns:                set.Unknowns,
		Notes:                   set.Notes,
		ReviewSequence:          reviewSequence(),
		AcceptedMeaning:         false,
		OwnerDecision:           "pending",
		AuthorityLimits:         ReviewAuthorityLimits(),
	}
	if binding != nil {
		packet.BindingAttempted = true
		packet.Bindings = binding.Bindings
	}
	if adoption != nil {
		packet.AdoptionPreviewed = true
		packet.Adoption = adoption
	}
	if comparison != nil {
		packet.ComparedWithBefore = true
		packet.Comparison = comparison
	}
	packet.Unresolved = aggregateUnresolved(set, binding)
	return packet
}

// ReviewAuthorityLimits is the fixed statement set of a review packet.
func ReviewAuthorityLimits() []string {
	return []string{
		"review-packet-is-a-read-only-external-projection",
		"structural-facts-are-separated-from-inferred-business-meaning",
		"candidate-identity-is-derived-not-acceptance",
		"source-revision-identifies-current-bytes-not-history",
		"unresolved-items-must-not-be-inferred-as-satisfied",
		"binding-is-a-read-only-comparison-not-a-mapping-decision",
		"adoption-preview-is-in-memory-and-never-written",
		"accepted-meaning-belongs-to-the-owner-outside-this-tool",
		"review-packet-does-not-replace-shared-acceptance",
		"review-packet-does-not-grant-write-runtime-or-decision-authority",
	}
}

func reviewSequence() []string {
	return []string{
		"verify-the-attempt-provenance-against-the-external-run",
		"review-each-candidate-structural-fact-against-its-named-source-revision",
		"decide-the-canonical-mapping-for-each-candidate-or-leave-it-unresolved",
		"resolve-or-explicitly-defer-every-unresolved-item",
		"decide-same-name-parallel-relation-and-direction-conflict-items",
		"accept-or-reject-through-the-existing-acceptance-route",
	}
}

// aggregateUnresolved collects every unresolved statement the packet carries,
// deduplicated and bounded.
func aggregateUnresolved(set CandidateSet, binding *BindingResult) []string {
	seen := map[string]bool{}
	var out []string
	add := func(values []string) {
		for _, value := range values {
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			out = append(out, value)
		}
	}
	for _, candidate := range set.Candidates {
		add(candidate.Unresolved)
		add(candidate.Inference.Unresolved)
	}
	if binding != nil {
		for _, item := range binding.Bindings {
			add(item.ResolutionNotes)
			if item.SourceBinding == SourceBindingConflict {
				add([]string{"source-digest-conflict-requires-fresh-analysis"})
			}
		}
	}
	for _, unknown := range set.Unknowns {
		if unknown.UnknownID != "" {
			add([]string{"provider-unknown:" + unknown.UnknownID})
		}
	}
	if set.Qualification != "complete" {
		add([]string{"attempt-qualification:" + set.Qualification})
	}
	sort.Strings(out)
	if len(out) > maxUnresolvedSummary {
		out = append(out[:maxUnresolvedSummary], "unresolved-summary-truncated")
	}
	return out
}

// Summary renders the packet's headline facts for a human reviewer. It never
// includes source bytes.
func (p ReviewPacket) Summary() []string {
	lines := []string{
		"attempt: " + p.Attempt.EvidenceID + " (" + p.Attempt.Provider.ID + " " + p.Attempt.Provider.Version + ")",
		"qualification: " + p.Qualification,
		"candidates: " + itoa(len(p.Candidates)),
		"unresolved: " + itoa(len(p.Unresolved)),
		"accepted-meaning: false",
		"owner-decision: " + p.OwnerDecision,
	}
	if p.BindingAttempted {
		resolved, ambiguous, unresolved := 0, 0, 0
		for _, binding := range p.Bindings {
			switch binding.Resolution {
			case ResolutionResolved:
				resolved++
			case ResolutionAmbiguous:
				ambiguous++
			default:
				unresolved++
			}
		}
		lines = append(lines, "binding: resolved="+itoa(resolved)+" ambiguous="+itoa(ambiguous)+" unresolved="+itoa(unresolved))
	} else {
		lines = append(lines, "binding: not-attempted")
	}
	if p.AdoptionPreviewed {
		lines = append(lines, "adoption-preview: in-memory only, not written")
	}
	for _, fact := range p.Carrier.MissingFacts() {
		lines = append(lines, "carrier-gap: "+fact)
	}
	return lines
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// CandidateLine renders one candidate for a review listing.
func CandidateLine(candidate StructureCandidate) string {
	var b strings.Builder
	b.WriteString(candidate.Kind)
	b.WriteString(" ")
	b.WriteString(candidate.CandidateID)
	b.WriteString(" external=")
	b.WriteString(candidate.Facts.ExternalIdentity)
	if candidate.Facts.Direction != nil {
		b.WriteString(" ")
		b.WriteString(candidate.Facts.Direction.FromExternalID)
		b.WriteString("->")
		b.WriteString(candidate.Facts.Direction.ToExternalID)
	}
	if candidate.SourceRevision.ArtifactPath != "" {
		b.WriteString(" source=")
		b.WriteString(candidate.SourceRevision.ArtifactPath)
	}
	b.WriteString(" binding=")
	b.WriteString(candidate.SourceRevision.Binding)
	b.WriteString(" resolution=")
	b.WriteString(candidate.Resolution)
	return b.String()
}
