package reacandidate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CandidateSet and candidate schema versions. They version the projection
// shape only; they are not a Canonical meaning contract.
const (
	candidateSetSchemaVersion  = "tei.rea.candidate-set/v0"
	candidateReviewSchemaVers  = "tei.rea.candidate-review/v0"
	candidateCarrierSchemaVers = "tei.rea.candidate-carrier-audit/v0"
)

// EffectReadOnly marks every projection in this package. None of them
// authorizes a source write, a Canonical edit, or a business acceptance.
const EffectReadOnly = "read-only"

// Candidate kinds name the three-element analysis element a candidate would
// become IF an Owner accepts it. The kind is a suggestion, never a decision.
const (
	CandidateKindDataElement    = "data-element"
	CandidateKindCrossReference = "cross-reference"
)

// Review and resolution states. A candidate is never "accepted" here.
const (
	ReviewStatusRequiresOwner = "requires-owner-review"

	ResolutionResolved   = "resolved"
	ResolutionAmbiguous  = "ambiguous"
	ResolutionUnresolved = "unresolved"
)

// Source binding states compare a candidate's declared source against an
// independently read artifact set. provider-reported is the honest intake
// state: REA named a path and digest, and nobody has re-read those bytes yet.
const (
	SourceBindingIndependent      = "independently-bound"
	SourceBindingProviderReported = "provider-reported"
	SourceBindingConflict         = "source-digest-conflict"
	SourceBindingAbsent           = "not-present-in-read-artifact-set"
	SourceBindingUnavailable      = "source-location-unavailable"
)

// artifactRootAlias is the path REAL REA 6.8.0 reports for the analyzed root.
// Its own limitation text calls it "a path-independent alias", so it is
// carried verbatim and separately marked as not being a resolvable file.
const artifactRootAlias = "artifact-root"

// SourceRevision is the Source Provenance a candidate carries: where REA said
// the observation came from, and the digest it reported for those bytes.
//
// A revision identifies CURRENT bytes. It is not a history: the same path and
// fingerprint after an intermediate change (A→B→A) proves current equality,
// not that nothing changed in between.
type SourceRevision struct {
	ArtifactPath string   `json:"artifactPath,omitempty"`
	Fingerprint  string   `json:"fingerprint,omitempty"`
	Span         *Span    `json:"span,omitempty"`
	Binding      string   `json:"binding"`
	BindingNotes []string `json:"bindingNotes,omitempty"`
}

// Span is a source range. Offset is omitted when the external provider
// reported only line/column, so a missing offset is visible instead of
// silently zero.
type Span struct {
	Start Position  `json:"start"`
	End   *Position `json:"end,omitempty"`
}

// Position is a 1-based line and column as reported by the provider.
type Position struct {
	Offset int `json:"offset,omitempty"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// StructuralFacts are what the external attempt structurally supports: the
// external identities, the observed direction, the observed state/authority,
// and the evidence coverage REA itself declared. Nothing here is a business
// claim, and nothing here is inferred.
type StructuralFacts struct {
	ExternalOccurrenceID string     `json:"externalOccurrenceId"`
	ExternalIdentity     string     `json:"externalIdentity"`
	ObservedKind         string     `json:"observedKind,omitempty"`
	Direction            *Direction `json:"direction,omitempty"`
	ObservedState        string     `json:"observedState,omitempty"`
	ObservedAuthority    string     `json:"observedAuthority,omitempty"`
	EvidenceCoverage     string     `json:"evidenceCoverage,omitempty"`
}

// Direction is the observed from→to order of a directed relation, in external
// identities. A→B and B→A are different facts and never merged.
type Direction struct {
	FromExternalID string `json:"fromExternalId"`
	ToExternalID   string `json:"toExternalId"`
}

// CandidateInference is what this tool SUGGESTS, separated from facts. Every
// entry names its structural basis; everything it could not support is listed
// under Unresolved instead of being guessed.
type CandidateInference struct {
	TargetElement string   `json:"targetElement"`
	SuggestedKind string   `json:"suggestedKind,omitempty"`
	SuggestedName string   `json:"suggestedName,omitempty"`
	Basis         []string `json:"basis"`
	Unresolved    []string `json:"unresolved"`
}

// StructureCandidate is one reviewable structure candidate. Its identity is
// derived from the external occurrence, so re-analysis of the same occurrence
// keeps the identity and a changed source is reported as a revision change of
// that candidate.
type StructureCandidate struct {
	CandidateID    string             `json:"candidateId"`
	Kind           string             `json:"kind"`
	Facts          StructuralFacts    `json:"facts"`
	Inference      CandidateInference `json:"inference"`
	SourceRevision SourceRevision     `json:"sourceRevision"`
	MeaningLoss    []string           `json:"meaningLoss"`
	Unresolved     []string           `json:"unresolved"`
	Notes          []string           `json:"notes,omitempty"`
	ReviewStatus   string             `json:"reviewStatus"`
	Resolution     string             `json:"resolution"`
}

// AttemptProvenance records which external attempt produced a candidate set.
// It is descriptive: it grants neither freshness nor acceptance.
type AttemptProvenance struct {
	EvidenceID       string             `json:"evidenceId"`
	Operation        string             `json:"operation"`
	Provider         ProviderDescriptor `json:"provider"`
	SubjectDigest    string             `json:"subjectDigest"`
	GraphCoverage    *CoverageRecord    `json:"graphCoverage,omitempty"`
	SemanticCoverage *CoverageRecord    `json:"semanticCoverage,omitempty"`
	Limitations      []string           `json:"limitations,omitempty"`
}

// CandidateSet is the complete reviewable output of one intake. It is a
// candidate-only projection: no Canonical subject, no accepted meaning, no
// write authority.
type CandidateSet struct {
	SchemaVersion   string               `json:"schemaVersion"`
	Effect          string               `json:"effect"`
	Attempt         AttemptProvenance    `json:"attempt"`
	Candidates      []StructureCandidate `json:"candidates"`
	Unknowns        []ExternalUnknown    `json:"unknowns,omitempty"`
	Qualification   string               `json:"qualification"`
	Notes           []string             `json:"notes,omitempty"`
	AuthorityLimits []string             `json:"authorityLimits"`
	Fingerprint     string               `json:"fingerprint"`
}

// CandidateAuthorityLimits is the fixed set of statements a candidate set
// makes about its own authority.
func CandidateAuthorityLimits() []string {
	return []string{
		"candidate-set-is-a-read-only-external-projection",
		"candidate-identity-is-derived-not-acceptance",
		"structural-facts-are-not-business-meaning",
		"inference-basis-is-structural-not-semantic-authority",
		"unresolved-items-must-not-be-inferred-as-satisfied",
		"source-revision-identifies-current-bytes-not-history",
		"candidate-set-does-not-create-canonical-subjects",
		"candidate-set-does-not-grant-write-runtime-or-decision-authority",
		"candidate-set-does-not-replace-owner-acceptance",
	}
}

// candidateIdentityInput is hashed to derive a candidate identity. The
// attempt evidence id and the source revision are deliberately excluded:
//
//   - the attempt id, so two attempts over the same occurrence describe the
//     same candidate, and the attempt is recorded separately as provenance;
//   - the source revision, so a re-analysis of the SAME occurrence over
//     CHANGED bytes keeps the candidate identity and the revision change is
//     reported as a change of that candidate, not as a removal plus an
//     addition.
//
// The source path is excluded for the same reason: a moved occurrence is a
// revision change of one candidate, not two candidates.
type candidateIdentityInput struct {
	Kind               string `json:"kind"`
	ExternalIdentity   string `json:"externalIdentity"`
	ExternalOccurrence string `json:"externalOccurrenceId"`
	From               string `json:"from,omitempty"`
	To                 string `json:"to,omitempty"`
}

// candidateID derives the stable identity of one candidate.
func candidateID(kind, externalIdentity, externalOccurrence, from, to string) string {
	encoded, err := json.Marshal(candidateIdentityInput{
		Kind:               kind,
		ExternalIdentity:   externalIdentity,
		ExternalOccurrence: externalOccurrence,
		From:               from,
		To:                 to,
	})
	if err != nil {
		// The input holds strings only; a marshal failure is impossible.
		panic("reacandidate: candidate identity input is not JSON: " + err.Error())
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// structuralKey is the comparison key used to recognise the SAME observed
// structure across two attempts. It deliberately excludes the external
// identities, so a re-analysis that renumbers REA's ids while observing the
// same structure is reported as a renumbering rather than as a removal plus
// an addition.
//
// A shared key is a candidate for Owner review, never proof of identity: two
// distinct structures can produce the same key when the provider reported no
// span, so callers must treat it as a hint.
func structuralKey(candidate StructureCandidate) string {
	parts := []string{
		candidate.Kind,
		candidate.SourceRevision.ArtifactPath,
		candidate.SourceRevision.Fingerprint,
		candidate.Facts.ObservedKind,
	}
	if span := candidate.SourceRevision.Span; span != nil {
		endLine, endColumn := span.Start.Line, span.Start.Column
		if span.End != nil {
			endLine, endColumn = span.End.Line, span.End.Column
		}
		parts = append(parts, fmt.Sprintf("span:%d:%d-%d:%d",
			span.Start.Line, span.Start.Column, endLine, endColumn))
	}
	return strings.Join(parts, "\x1f")
}

// SetFingerprint returns the content fingerprint of a candidate set. The
// attempt provenance is excluded so that two attempts over the same current
// candidates compare equal on content; the attempt is compared separately.
func SetFingerprint(set CandidateSet) string {
	projection := set
	projection.Fingerprint = ""
	projection.Attempt = AttemptProvenance{}
	projection.AuthorityLimits = nil
	projection.Notes = nil
	encoded, err := json.Marshal(projection)
	if err != nil {
		panic("reacandidate: candidate set is not JSON: " + err.Error())
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// sortCandidates orders candidates deterministically for review output.
func sortCandidates(candidates []StructureCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Facts.ExternalIdentity != b.Facts.ExternalIdentity {
			return a.Facts.ExternalIdentity < b.Facts.ExternalIdentity
		}
		return a.Facts.ExternalOccurrenceID < b.Facts.ExternalOccurrenceID
	})
}

// sortStrings returns a sorted copy; nil stays nil so JSON stays stable.
func sortStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

// uniqueStrings returns the sorted set of nonempty values.
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
