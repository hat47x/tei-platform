package reacandidate

import (
	"sort"
	"strings"

	"tei/internal/threeelement"
)

// AdoptionRecord is the reviewable trace of ONE candidate's proposed
// adoption: its identity, the source revision it came from, the structural
// basis, what stayed unresolved, and what meaning the projection loses.
//
// Accepted and Owner are deliberately never set here. This package has no
// accept path: the record is what an Owner decides against, not a decision.
type AdoptionRecord struct {
	CandidateID            string         `json:"candidateId"`
	Kind                   string         `json:"kind"`
	ExternalIdentity       string         `json:"externalIdentity"`
	SourceRevision         SourceRevision `json:"sourceRevision"`
	Basis                  []string       `json:"basis"`
	Unresolved             []string       `json:"unresolved"`
	MeaningLoss            []string       `json:"meaningLoss"`
	MergedFromCandidates   []string       `json:"mergedFromCandidates,omitempty"`
	BeforeModelFingerprint string         `json:"beforeModelFingerprint"`
	AfterModelFingerprint  string         `json:"afterModelFingerprint"`
	Accepted               bool           `json:"accepted"`
	Owner                  string         `json:"owner"`
}

// AdoptionPreview is a BEFORE/AFTER comparison of a three-element model with
// candidates applied. The after model is an in-memory proposal preview: it is
// never written, never accepted, and never replaces the reviewed model.
type AdoptionPreview struct {
	SchemaVersion   string                         `json:"schemaVersion"`
	Effect          string                         `json:"effect"`
	Before          threeelement.ThreeElementModel `json:"before"`
	After           threeelement.ThreeElementModel `json:"after"`
	Records         []AdoptionRecord               `json:"records"`
	Diff            threeelement.DiffResult        `json:"diff"`
	AfterValidation threeelement.ValidationReport  `json:"afterValidation"`
	AuthorityLimits []string                       `json:"authorityLimits"`
}

// AdoptionAuthorityLimits states what a preview does not establish.
func AdoptionAuthorityLimits() []string {
	return []string{
		"adoption-preview-is-an-in-memory-proposal",
		"preview-does-not-write-the-reviewed-model",
		"preview-does-not-accept-business-meaning",
		"preview-does-not-replace-owner-acceptance",
		"after-model-fingerprint-is-identity-not-authority",
		"unresolved-endpoint-is-reported-not-repaired",
		"same-external-node-grouping-is-structural-not-a-business-merge",
	}
}

// unresolvedElementID names an element the preview could not derive, so the
// three-element validation reports it instead of the preview inventing one.
func unresolvedElementID(externalNodeID string) string {
	return "unresolved:" + externalNodeID
}

// safeModelFingerprint computes the three-element model fingerprint without
// letting the shared element slices be mutated.
//
// threeelement.ModelFingerprint clears AuthorityLimits on the elements of the
// value it receives. Because a Go slice copy shares its backing array, that
// clearing is visible to the caller, which would silently strip the
// not-accepted limit this package writes onto preview elements. Copying the
// element slices first keeps the fingerprint pure. The upstream mutation is
// recorded as a known defect rather than patched here, so this branch does not
// edit the three-element package it builds on.
func safeModelFingerprint(model threeelement.ThreeElementModel) string {
	isolated := model
	isolated.DataElements = append([]threeelement.DataElement(nil), model.DataElements...)
	isolated.FunctionElements = append([]threeelement.FunctionElement(nil), model.FunctionElements...)
	isolated.BusinessElements = append([]threeelement.BusinessElement(nil), model.BusinessElements...)
	return threeelement.ModelFingerprint(isolated)
}

// PreviewAdoption applies candidates to a COPY of the model and returns the
// before/after comparison with one record per candidate group.
//
// Data candidates that share one external node identity are grouped into one
// proposed DataElement, because a directed edge relates to nodes, not to
// individual observations. The grouping is structural and recorded; the
// business decision to merge or split stays with the Owner.
func PreviewAdoption(set CandidateSet, model threeelement.ThreeElementModel) (AdoptionPreview, error) {
	before := model
	after := model

	// Group data candidates by external node identity.
	groupIDs := map[string]string{}
	groupMembers := map[string][]string{}
	groupNames := map[string]string{}
	for _, candidate := range set.Candidates {
		if candidate.Kind != CandidateKindDataElement {
			continue
		}
		identity := candidate.Facts.ExternalIdentity
		groupMembers[identity] = append(groupMembers[identity], candidate.CandidateID)
		if name := candidate.Inference.SuggestedName; name != "" {
			if existing, ok := groupNames[identity]; !ok || name < existing {
				groupNames[identity] = name
			}
		}
	}
	for identity, members := range groupMembers {
		sort.Strings(members)
		groupIDs[identity] = members[0]
		groupMembers[identity] = members
	}

	beforeFingerprint := safeModelFingerprint(before)

	after.DataElements = append([]threeelement.DataElement(nil), before.DataElements...)
	for _, identity := range sortedMapKeys(groupMembers) {
		after.DataElements = append(after.DataElements, threeelement.DataElement{
			ID:   groupIDs[identity],
			Name: groupNames[identity],
			AuthorityLimits: []string{
				"rea-candidate-preview-not-accepted",
				"owner-acceptance-may-be-required",
			},
		})
	}

	after.CrossReferences = append([]threeelement.CrossReference(nil), before.CrossReferences...)
	for _, candidate := range set.Candidates {
		if candidate.Kind != CandidateKindCrossReference {
			continue
		}
		direction := candidate.Facts.Direction
		if direction == nil {
			continue
		}
		fromID, okFrom := groupIDs[direction.FromExternalID]
		if !okFrom {
			fromID = unresolvedElementID(direction.FromExternalID)
		}
		toID, okTo := groupIDs[direction.ToExternalID]
		if !okTo {
			toID = unresolvedElementID(direction.ToExternalID)
		}
		after.CrossReferences = append(after.CrossReferences, threeelement.CrossReference{
			ID:       candidate.CandidateID,
			Kind:     candidate.Inference.SuggestedKind,
			FromID:   fromID,
			FromKind: "data",
			ToID:     toID,
			ToKind:   "data",
			Role:     candidate.Facts.ObservedKind,
			Directed: true,
		})
	}
	sortDataElements(after.DataElements)
	sortCrossReferences(after.CrossReferences)
	after.ModelFingerprint = safeModelFingerprint(after)

	preview := AdoptionPreview{
		SchemaVersion:   "tei.rea.adoption-preview/v0",
		Effect:          EffectReadOnly,
		Before:          before,
		After:           after,
		Diff:            threeelement.DiffModels(before, after),
		AfterValidation: threeelement.ValidateModel(after),
		AuthorityLimits: AdoptionAuthorityLimits(),
	}
	for _, candidate := range set.Candidates {
		record := AdoptionRecord{
			CandidateID:            candidate.CandidateID,
			Kind:                   candidate.Kind,
			ExternalIdentity:       candidate.Facts.ExternalIdentity,
			SourceRevision:         candidate.SourceRevision,
			Basis:                  append([]string(nil), candidate.Inference.Basis...),
			Unresolved:             append([]string(nil), candidate.Unresolved...),
			MeaningLoss:            append([]string(nil), candidate.MeaningLoss...),
			BeforeModelFingerprint: beforeFingerprint,
			AfterModelFingerprint:  after.ModelFingerprint,
			Accepted:               false,
			Owner:                  "",
		}
		if candidate.Kind == CandidateKindDataElement {
			record.MergedFromCandidates = append([]string(nil), groupMembers[candidate.Facts.ExternalIdentity]...)
		}
		preview.Records = append(preview.Records, record)
	}
	sort.Slice(preview.Records, func(i, j int) bool {
		return preview.Records[i].CandidateID < preview.Records[j].CandidateID
	})
	return preview, nil
}

// CandidateSetComparison reports how two candidate sets of the same external
// subject relate. It is intentionally revision- and identity-aware: equal
// current bytes after an intermediate change do not prove continuity, and a
// renumbered external identity is not a removal plus an addition.
type CandidateSetComparison struct {
	SchemaVersion   string                     `json:"schemaVersion"`
	Effect          string                     `json:"effect"`
	Entries         []CandidateComparisonEntry `json:"entries"`
	AuthorityLimits []string                   `json:"authorityLimits"`
}

// CandidateComparisonEntry is one candidate's before/after relationship.
type CandidateComparisonEntry struct {
	CandidateID     string         `json:"candidateId"`
	Status          string         `json:"status"`
	StructuralKey   string         `json:"structuralKey,omitempty"`
	BeforeSource    SourceRevision `json:"beforeSource,omitempty"`
	AfterSource     SourceRevision `json:"afterSource,omitempty"`
	BeforeAttemptID string         `json:"beforeAttemptId,omitempty"`
	AfterAttemptID  string         `json:"afterAttemptId,omitempty"`
	Notes           []string       `json:"notes,omitempty"`
}

// Comparison statuses.
const (
	ComparisonUnchanged           = "unchanged"
	ComparisonAdded               = "added"
	ComparisonRemoved             = "removed"
	ComparisonRevisionChanged     = "revision-changed"
	ComparisonSameBytesNewAttempt = "same-current-bytes-different-attempt"
	ComparisonRenumbered          = "external-identity-renumbered"
)

// ComparisonAuthorityLimits states what a comparison does not establish.
func ComparisonAuthorityLimits() []string {
	return []string{
		"comparison-is-read-only",
		"same-current-bytes-do-not-prove-no-intermediate-revision",
		"same-current-bytes-do-not-revive-a-prior-review-or-grant",
		"revision-change-requires-fresh-owner-review",
		"external-identity-renumbering-requires-fresh-owner-review",
		"structural-key-is-a-review-hint-not-an-identity-proof",
		"comparison-does-not-accept-or-write",
	}
}

// CompareCandidateSets compares two candidate sets in three passes:
//
//  1. candidates with the same identity are compared as the SAME candidate;
//  2. the remaining candidates are matched by structural key, which catches a
//     provider that renumbered its external identities over the same
//     structure;
//  3. whatever is left is genuinely added or removed.
func CompareCandidateSets(before, after CandidateSet) CandidateSetComparison {
	comparison := CandidateSetComparison{
		SchemaVersion:   "tei.rea.candidate-set-comparison/v0",
		Effect:          EffectReadOnly,
		AuthorityLimits: ComparisonAuthorityLimits(),
	}

	beforeByID := map[string]StructureCandidate{}
	for _, candidate := range before.Candidates {
		beforeByID[candidate.CandidateID] = candidate
	}
	matchedBefore := map[string]bool{}
	for _, candidate := range after.Candidates {
		if previous, ok := beforeByID[candidate.CandidateID]; ok {
			matchedBefore[candidate.CandidateID] = true
			comparison.Entries = append(comparison.Entries,
				comparePair(previous, candidate, before.Attempt.EvidenceID, after.Attempt.EvidenceID))
		}
	}

	// Pass 2: structural-key matching for renumbered identities. Both sides
	// are grouped so several structurally identical candidates match in a
	// deterministic order rather than one of them absorbing the rest.
	beforeByKey := map[string][]string{}
	for _, candidate := range before.Candidates {
		if matchedBefore[candidate.CandidateID] {
			continue
		}
		key := structuralKey(candidate)
		beforeByKey[key] = append(beforeByKey[key], candidate.CandidateID)
	}
	for key := range beforeByKey {
		sort.Strings(beforeByKey[key])
	}
	pairings := map[string]string{} // after candidate id -> before candidate id
	for _, candidate := range after.Candidates {
		if _, ok := beforeByID[candidate.CandidateID]; ok {
			continue
		}
		key := structuralKey(candidate)
		pending := beforeByKey[key]
		if len(pending) == 0 {
			continue
		}
		pairings[candidate.CandidateID] = pending[0]
		beforeByKey[key] = pending[1:]
	}

	for _, candidate := range after.Candidates {
		if _, ok := beforeByID[candidate.CandidateID]; ok {
			continue
		}
		entry := CandidateComparisonEntry{
			CandidateID:    candidate.CandidateID,
			StructuralKey:  structuralKey(candidate),
			AfterSource:    candidate.SourceRevision,
			AfterAttemptID: after.Attempt.EvidenceID,
		}
		if partner, ok := pairings[candidate.CandidateID]; ok {
			previous := beforeByID[partner]
			entry.Status = ComparisonRenumbered
			entry.BeforeSource = previous.SourceRevision
			entry.BeforeAttemptID = before.Attempt.EvidenceID
			entry.Notes = append(entry.Notes,
				"same-structure-with-a-new-external-identity",
				"a-renumbered-candidate-requires-fresh-owner-review")
		} else {
			entry.Status = ComparisonAdded
		}
		comparison.Entries = append(comparison.Entries, entry)
	}

	// Pass 3: before candidates that matched neither by identity nor by
	// structural key are genuinely removed.
	renumberedBefore := map[string]bool{}
	for _, partner := range pairings {
		renumberedBefore[partner] = true
	}
	for _, candidate := range before.Candidates {
		if matchedBefore[candidate.CandidateID] || renumberedBefore[candidate.CandidateID] {
			continue
		}
		comparison.Entries = append(comparison.Entries, CandidateComparisonEntry{
			CandidateID:     candidate.CandidateID,
			Status:          ComparisonRemoved,
			StructuralKey:   structuralKey(candidate),
			BeforeSource:    candidate.SourceRevision,
			BeforeAttemptID: before.Attempt.EvidenceID,
		})
	}

	sort.Slice(comparison.Entries, func(i, j int) bool {
		a, b := comparison.Entries[i], comparison.Entries[j]
		if a.Status != b.Status {
			return a.Status < b.Status
		}
		return a.CandidateID < b.CandidateID
	})
	return comparison
}

func comparePair(before, after StructureCandidate, beforeAttempt, afterAttempt string) CandidateComparisonEntry {
	entry := CandidateComparisonEntry{
		CandidateID:     after.CandidateID,
		StructuralKey:   structuralKey(after),
		BeforeSource:    before.SourceRevision,
		AfterSource:     after.SourceRevision,
		BeforeAttemptID: beforeAttempt,
		AfterAttemptID:  afterAttempt,
	}
	sameBytes := before.SourceRevision.ArtifactPath == after.SourceRevision.ArtifactPath &&
		before.SourceRevision.Fingerprint == after.SourceRevision.Fingerprint
	switch {
	case !sameBytes:
		entry.Status = ComparisonRevisionChanged
		entry.Notes = append(entry.Notes,
			"the-analyzed-bytes-differ-between-the-two-sets",
			"a-returning-fingerprint-does-not-prove-no-intermediate-revision",
			"fresh-owner-review-is-required")
	case beforeAttempt == afterAttempt:
		entry.Status = ComparisonUnchanged
	default:
		entry.Status = ComparisonSameBytesNewAttempt
		entry.Notes = append(entry.Notes,
			"the-current-bytes-are-identical-in-a-different-attempt",
			"identical-current-bytes-are-not-evidence-that-nothing-changed-in-between",
			"no-prior-review-grant-or-acceptance-is-revived")
	}
	return entry
}

func sortedMapKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortDataElements(elements []threeelement.DataElement) {
	sort.Slice(elements, func(i, j int) bool { return elements[i].ID < elements[j].ID })
}

func sortCrossReferences(refs []threeelement.CrossReference) {
	sort.Slice(refs, func(i, j int) bool {
		a, b := refs[i], refs[j]
		if a.FromID != b.FromID {
			return a.FromID < b.FromID
		}
		if a.ToID != b.ToID {
			return a.ToID < b.ToID
		}
		return a.ID < b.ID
	})
}

// describeKind renders a candidate kind for review prose.
func describeKind(kind string) string {
	switch kind {
	case CandidateKindDataElement:
		return "data element"
	case CandidateKindCrossReference:
		return "cross-reference"
	}
	return strings.TrimSpace(kind)
}
