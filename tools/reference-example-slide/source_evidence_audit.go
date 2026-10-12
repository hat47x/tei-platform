package exampleslide

import (
	"fmt"
	"strings"

	"tei/internal/sourceindex"
)

// SourceEvidenceAudit inspects claimed source-index pointers without changing
// a review, confirmation, or authoritative TEI source.
type SourceEvidenceAudit struct {
	Schema            string                  `json:"schema"`
	CaseID            string                  `json:"caseId"`
	ReviewFingerprint string                  `json:"reviewFingerprint"`
	AnswerFingerprint string                  `json:"answerFingerprint"`
	Findings          []SourceEvidenceFinding `json:"findings"`
}

type SourceEvidenceFinding struct {
	Question            QuestionRef `json:"question"`
	EvidenceKind        string      `json:"evidenceKind"`
	EvidenceRef         string      `json:"evidenceRef"`
	Status              string      `json:"status"`
	ArtifactPath        string      `json:"artifactPath,omitempty"`
	ArtifactFingerprint string      `json:"artifactFingerprint,omitempty"`
}

// AuditAnswerSourceEvidence rebuilds the exact Review and answer handoff,
// then checks that each claimed source-index pointer resolves to one current
// definition anchor for the question's semantic relation. It checks *bytes*
// and locators only: it does not certify business truth, source authenticity,
// contributor identity, source-index generation or acceptance authority.
//
// The caller must provide a reader constrained to the intended Workspace.
// In particular, this function does not open filesystem paths itself.
func AuditAnswerSourceEvidence(input Input, answers AnswerInput, idx sourceindex.Index, read func(string) ([]byte, error)) (SourceEvidenceAudit, error) {
	var empty SourceEvidenceAudit
	handoff, err := BindAnswers(input, answers)
	if err != nil {
		return empty, err
	}
	if read == nil {
		return empty, fmt.Errorf("Workspace source reader is required")
	}
	// A stale or invalid index cannot produce even partially positive results.
	if err := idx.Validate(read); err != nil {
		return empty, fmt.Errorf("Source Index validation: %w", err)
	}
	artifacts := make(map[string]sourceindex.Artifact, len(idx.Artifacts))
	for _, artifact := range idx.Artifacts {
		artifacts[artifact.Path] = artifact
	}
	out := SourceEvidenceAudit{
		Schema:            "tei.reference.example-source-evidence-audit/v0",
		CaseID:            handoff.CaseID,
		ReviewFingerprint: handoff.ReviewFingerprint,
		AnswerFingerprint: handoff.Fingerprint,
	}
	for _, answer := range handoff.Answers {
		for _, evidence := range answer.Evidence {
			item := SourceEvidenceFinding{
				Question: answer.Question, EvidenceKind: evidence.Kind,
				EvidenceRef: evidence.Ref, Status: "not-checked",
			}
			if evidence.Kind != "source-index" {
				out.Findings = append(out.Findings, item)
				continue
			}
			if answer.SourceIssue == nil || answer.SourceIssue.RelationID == "" {
				item.Status = "no-derived-relation"
				out.Findings = append(out.Findings, item)
				continue
			}
			kind, id, ok := strings.Cut(evidence.Ref, ":")
			if !ok || kind != "semantic-relation" || id != answer.SourceIssue.RelationID {
				item.Status = "different-subject"
				out.Findings = append(out.Findings, item)
				continue
			}
			var definitions []sourceindex.Reference
			for _, ref := range idx.ReferencesFor(sourceindex.Subject{Kind: kind, ID: id}) {
				if ref.Role == sourceindex.RoleDefinition {
					definitions = append(definitions, ref)
				}
			}
			if len(definitions) == 0 {
				item.Status = "definition-not-indexed"
				out.Findings = append(out.Findings, item)
				continue
			}
			if len(definitions) != 1 {
				item.Status = "ambiguous-definition"
				out.Findings = append(out.Findings, item)
				continue
			}
			ref := definitions[0]
			artifact := artifacts[ref.ArtifactPath]
			item.ArtifactPath = artifact.Path
			item.ArtifactFingerprint = artifact.Fingerprint
			if evidence.Fingerprint == "" {
				item.Status = "evidence-fingerprint-missing"
			} else if evidence.Fingerprint != artifact.Fingerprint {
				item.Status = "evidence-fingerprint-mismatch"
			} else {
				content, readErr := read(artifact.Path)
				if readErr != nil {
					return empty, fmt.Errorf("reread source %s: %w", artifact.Path, readErr)
				}
				if sourceindex.Fingerprint(content) != artifact.Fingerprint {
					return empty, fmt.Errorf("source changed during evidence inspection: %s", artifact.Path)
				}
				start, end := ref.Range.Start.Offset, ref.Range.End.Offset
				if start == end || string(content[start:end]) != id {
					item.Status = "exact-subject-anchor-not-confirmed"
				} else {
					item.Status = "source-range-checked"
				}
			}
			out.Findings = append(out.Findings, item)
		}
	}
	return out, nil
}
