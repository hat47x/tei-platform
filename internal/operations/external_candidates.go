package operations

import (
	"fmt"
	"os"
	"strings"
	"time"

	"tei/internal/externalanalysis"
)

// ExternalCandidateRequest asks for the external structural candidates one
// saved analysis output carries.
//
// The request names a path, not bytes. The projection reads the file once and
// reports the digest of what it read, so a caller can tell afterwards whether
// the file on disk is still the file that was projected — the same discipline
// the intake uses, kept here instead of being reinvented.
type ExternalCandidateRequest struct {
	// EvidencePath is the saved external analysis output to read.
	EvidencePath string
	// ToolVersion declares which external tool version produced the bytes.
	// It is required: the framing a delivery is checked against and the
	// source.kind a request may use both belong to a version, and an
	// undeclared version is refused rather than guessed.
	ToolVersion string
	// ObservationID selects the observation to project, by the producer's own
	// node or observation identity. It is an external identity, never a TEI
	// subject.
	ObservationID string
	// RelationID selects the directed edge to project, by the producer's own
	// relation identity, or by the derived handle when the producer states
	// none.
	RelationID string
	// ExpectedEvidenceID, ExpectedReceivedDigest, and ExpectedSourceRevision
	// optionally pin a previously inspected, producer-declared Evidence attempt.
	// A mismatch refuses selection; a match is NOT proof that the underlying
	// Source is still current or that a consumer has disclosure/write authority.
	ExpectedEvidenceID     string
	ExpectedReceivedDigest string
	ExpectedSourceRevision string
	// Tool records what the calling entry observed about the external tool in
	// this environment. The projection records it and does not probe: whether
	// the tool could have run here is an observation the entry makes at the
	// process boundary, not something a read-only projection can establish.
	Tool externalanalysis.Tool
}

// ExternalCandidateView is the read-only projection of the selected external
// candidates, together with the attempt they came from.
//
// It is an external result, not TEI meaning. The AuthorityLimits list is the
// projection's own statement of what it does not grant, the way the catalog,
// inspect and trace projections state theirs.
type ExternalCandidateView struct {
	SchemaVersion string                        `json:"schemaVersion"`
	Effect        string                        `json:"effect"`
	Attempt       ExternalAttempt               `json:"attempt"`
	Observation   *externalanalysis.Observation `json:"observation,omitempty"`
	Relation      *externalanalysis.Relation    `json:"relation,omitempty"`
	// Coverage, Unknowns and Limitations are copied from the producer. They
	// describe what the producer saw, not what exists.
	Coverage    *externalanalysis.GraphCoverage `json:"coverage,omitempty"`
	Unknowns    []externalanalysis.Unknown      `json:"unknowns"`
	Limitations []string                        `json:"limitations,omitempty"`
	// Unselected counts what the Evidence carries beyond the two projected
	// candidates. A caller must not read two candidates as the whole graph.
	Unselected      ExternalUnselected `json:"unselected"`
	AuthorityLimits []string           `json:"authorityLimits"`
}

// ExternalAttempt identifies the external attempt the candidates came from: the
// producer's Evidence identity, the digest of the bytes that were read, the
// revision of the target they describe, and when they were read. Two attempts
// that name the same Evidence ID with different digests are different
// observations.
type ExternalAttempt struct {
	EvidenceID string                    `json:"evidenceId"`
	Operation  string                    `json:"operation,omitempty"`
	Provider   externalanalysis.Provider `json:"provider"`
	// SourceRevision is the digest of the exact target bytes the Evidence
	// describes, as the producer declared it. It is the revision a later
	// receipt of the same Evidence ID is compared against: a moved target is a
	// different observation even when the Evidence ID is unchanged.
	SourceRevision string                     `json:"sourceRevision,omitempty"`
	ReceivedDigest string                     `json:"receivedDigest"`
	ReadAt         string                     `json:"readAt"`
	Tool           externalanalysis.Tool      `json:"tool"`
	Envelope       *externalanalysis.Envelope `json:"envelope,omitempty"`
}

// ExternalUnselected counts the candidates the Evidence carries that this
// projection did not select.
type ExternalUnselected struct {
	Observations int `json:"observations"`
	Relations    int `json:"relations"`
	Unknowns     int `json:"unknowns"`
}

// ExternalCandidateError is the failure a candidate projection reports. Its
// codes separate an input the caller can fix from a delivery that is not whole
// and from bytes that are not what they claim to be.
type ExternalCandidateError struct {
	Code    string
	Message string
}

func (e *ExternalCandidateError) Error() string {
	if e == nil {
		return "external candidate projection failure"
	}
	return e.Message
}

const (
	ExternalCandidateInvalidRequest = "EXTERNAL_CANDIDATE_INVALID_REQUEST"
	ExternalCandidateReadFailed     = "EXTERNAL_CANDIDATE_READ_FAILED"
	ExternalCandidateUnusable       = "EXTERNAL_CANDIDATE_UNUSABLE"
	ExternalCandidateIncomplete     = "EXTERNAL_CANDIDATE_INCOMPLETE"
	ExternalCandidateFailed         = "EXTERNAL_CANDIDATE_FAILED"
	ExternalCandidateNotFound       = "EXTERNAL_CANDIDATE_NOT_FOUND"
	ExternalCandidateAmbiguous      = "EXTERNAL_CANDIDATE_AMBIGUOUS"
	ExternalCandidateStale          = "EXTERNAL_CANDIDATE_STALE"
)

// ExternalCandidateAuthorityLimits is the fixed set of statements this
// projection makes about its own authority. It is a constant list, not a
// per-caller decision.
func ExternalCandidateAuthorityLimits() []string {
	return []string{
		"external-candidate-is-not-accepted-meaning",
		"external-identity-is-not-a-tei-semantic-subject",
		"external-observation-is-not-a-database-observation",
		"partial-selection-is-not-the-whole-graph",
		"local-read-is-not-disclosure-permission",
	}
}

// sourceRevisionOf reports the target revision the receipt declared, or "" when
// the producer stated none. An absent declaration is not an empty revision: the
// caller sees the field missing rather than a value that matches everything.
func sourceRevisionOf(receipt externalanalysis.Receipt) string {
	if receipt.Source == nil {
		return ""
	}
	return receipt.Source.Revision
}

// ExternalCandidates reads one saved external analysis output and projects the
// selected observation and directed edge, with the attempt they came from.
//
// It reuses the intake's receipt and candidate reading rather than parsing the
// producer's bytes a second time: there is one reader of the Evidence format,
// and this projection is a view over it.
func ExternalCandidates(request ExternalCandidateRequest, at time.Time) (ExternalCandidateView, error) {
	if strings.TrimSpace(request.EvidencePath) == "" {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateInvalidRequest, Message: "evidence path must be nonempty"}
	}
	if strings.TrimSpace(request.ToolVersion) == "" {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateInvalidRequest, Message: "tool version must be declared: the framing and the accepted source kinds belong to a version"}
	}
	if strings.TrimSpace(request.ObservationID) == "" {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateInvalidRequest, Message: "observation id must be nonempty"}
	}
	if strings.TrimSpace(request.RelationID) == "" {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateInvalidRequest, Message: "relation id must be nonempty"}
	}
	raw, err := os.ReadFile(request.EvidencePath)
	if err != nil {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateReadFailed, Message: fmt.Sprintf("read the saved external analysis output: %v", err)}
	}
	tool := request.Tool
	if strings.TrimSpace(tool.Name) == "" {
		tool.Name = "rea"
	}
	tool.Version = request.ToolVersion
	receipt := externalanalysis.Receive(raw, tool, at)

	switch receipt.State {
	case externalanalysis.StateComplete, externalanalysis.StateToolAbsent:
		// A whole receipt from an environment without the tool is still a
		// whole receipt; the attempt records the caveat.
	case externalanalysis.StatePartial:
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateIncomplete, Message: receipt.Reason}
	case externalanalysis.StateFailed:
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateFailed, Message: receipt.Reason}
	default:
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateUnusable, Message: receipt.Reason}
	}
	if receipt.Evidence == nil {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateUnusable, Message: "the received bytes carry no Evidence record"}
	}
	candidates, err := externalanalysis.ReadCandidates(receipt.EvidenceBody())
	if err != nil {
		return ExternalCandidateView{}, &ExternalCandidateError{Code: ExternalCandidateUnusable, Message: err.Error()}
	}

	view := ExternalCandidateView{
		SchemaVersion: "tei.operations.external-candidates/v0",
		Effect:        "read-only",
		Attempt: ExternalAttempt{
			EvidenceID:     receipt.Evidence.ID,
			Operation:      receipt.Evidence.Operation,
			Provider:       receipt.Evidence.Provider,
			SourceRevision: sourceRevisionOf(receipt),
			ReceivedDigest: receipt.Raw.Fingerprint,
			ReadAt:         receipt.AcquiredAt,
			Tool:           receipt.Tool,
			Envelope:       receipt.Envelope,
		},
		Coverage:        candidates.Coverage,
		Unknowns:        candidates.Unknowns,
		Limitations:     candidates.Limitations,
		AuthorityLimits: ExternalCandidateAuthorityLimits(),
	}
	if view.Unknowns == nil {
		view.Unknowns = []externalanalysis.Unknown{}
	}
	// Candidate IDs are producer-owned and may not be unique. Derived
	// relation handles can also collide for parallel edges with identical
	// from/to/kind. Never let iteration order silently choose the last match.
	observationMatches := 0
	// Only the producer's declared SourceRevision is available in this
	// read-only Operation. Its agreement with the pin does not re-read the
	// original Source, revalidate Workspace Root, or reinstate an old Grant.
	if (request.ExpectedEvidenceID != "" && request.ExpectedEvidenceID != view.Attempt.EvidenceID) ||
		(request.ExpectedReceivedDigest != "" && request.ExpectedReceivedDigest != view.Attempt.ReceivedDigest) ||
		(request.ExpectedSourceRevision != "" && request.ExpectedSourceRevision != view.Attempt.SourceRevision) {
		view.Unselected.Observations = len(candidates.Observations)
		view.Unselected.Relations = len(candidates.Relations)
		view.Unselected.Unknowns = len(candidates.Unknowns)
		return view, &ExternalCandidateError{
			Code:    ExternalCandidateStale,
			Message: "the received external Evidence no longer matches the pinned evidence identity, raw digest, or declared source revision",
		}
	}
	for _, observation := range candidates.Observations {
		if observation.NodeID == request.ObservationID || observation.ObservationID == request.ObservationID {
			observationMatches++
			if observationMatches == 1 {
				selected := observation
				view.Observation = &selected
			}
			continue
		}
		view.Unselected.Observations++
	}
	relationMatches := 0
	for _, relation := range candidates.Relations {
		if relation.RelationID == request.RelationID {
			relationMatches++
			if relationMatches == 1 {
				selected := relation
				view.Relation = &selected
			}
			continue
		}
		view.Unselected.Relations++
	}
	view.Unselected.Unknowns = len(candidates.Unknowns)
	if observationMatches > 1 || relationMatches > 1 {
		// None of the matching occurrences is a qualified unique selection.
		// Keep the attempt and coverage for diagnosis, without returning any
		// arbitrary candidate or claiming that only one was projected.
		view.Observation = nil
		view.Relation = nil
		view.Unselected.Observations = len(candidates.Observations)
		view.Unselected.Relations = len(candidates.Relations)
		return view, &ExternalCandidateError{
			Code:    ExternalCandidateAmbiguous,
			Message: fmt.Sprintf("external candidate selection is ambiguous: observation %q matches %d occurrences, relation %q matches %d occurrences", request.ObservationID, observationMatches, request.RelationID, relationMatches),
		}
	}
	if view.Observation == nil {
		return view, &ExternalCandidateError{Code: ExternalCandidateNotFound, Message: fmt.Sprintf("observation %q is not in Evidence %s", request.ObservationID, candidates.EvidenceID)}
	}
	if view.Relation == nil {
		return view, &ExternalCandidateError{Code: ExternalCandidateNotFound, Message: fmt.Sprintf("relation %q is not in Evidence %s", request.RelationID, candidates.EvidenceID)}
	}
	return view, nil
}
