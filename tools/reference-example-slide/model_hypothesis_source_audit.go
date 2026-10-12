package exampleslide

import (
	"fmt"
	"reflect"
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// ModelHypothesisSourceAudit verifies *existing* semantic Data and Attribute
// definition anchors. It never verifies the proposed business key or FD itself.
type ModelHypothesisSourceAudit struct {
	Schema                 string                         `json:"schema"`
	CaseID                 string                         `json:"caseId"`
	ModelReviewFingerprint string                         `json:"modelReviewFingerprint"`
	HypothesesFingerprint  string                         `json:"hypothesesFingerprint"`
	Fingerprint            string                         `json:"fingerprint"`
	Findings               []ModelHypothesisSourceFinding `json:"findings"`
}

type ModelHypothesisSourceFinding struct {
	IssueID             string             `json:"issueId"`
	HypothesisKind      string             `json:"hypothesisKind"`
	HypothesisID        string             `json:"hypothesisId"`
	SubjectKind         string             `json:"subjectKind"`
	SubjectID           string             `json:"subjectId"`
	Status              string             `json:"status"`
	ArtifactPath        string             `json:"artifactPath,omitempty"`
	ArtifactFingerprint string             `json:"artifactFingerprint,omitempty"`
	SourceRange         *sourceindex.Range `json:"sourceRange,omitempty"`
}

// AuditModelHypothesisSourceEvidence reconstructs every hypothesis review,
// validates the entire Source Index against a bounded reader, and checks each
// declared Data / Attribute source anchor against exactly the parsed token.
// A matching anchor is only proof of source location, never a candidate-key
// declaration, human acceptance, or execution authorization.
func AuditModelHypothesisSourceEvidence(input Input, proposals ModelHypotheses, idx sourceindex.Index, read func(string) ([]byte, error)) (ModelHypothesisSourceAudit, error) {
	var empty ModelHypothesisSourceAudit
	review, err := ReviewModelHypotheses(input, proposals)
	if err != nil {
		return empty, err
	}
	if read == nil {
		return empty, fmt.Errorf("bounded source reader required")
	}
	if err := idx.Validate(read); err != nil {
		return empty, fmt.Errorf("Source Index validation: %w", err)
	}
	artifacts := make(map[string]sourceindex.Artifact, len(idx.Artifacts))
	for _, artifact := range idx.Artifacts {
		artifacts[artifact.Path] = artifact
	}
	out := ModelHypothesisSourceAudit{
		Schema:                 "tei.reference.example-model-hypothesis-source-audit/v0",
		CaseID:                 review.CaseID,
		ModelReviewFingerprint: review.Fingerprint,
		HypothesesFingerprint:  review.HypothesesFingerprint,
		Findings:               make([]ModelHypothesisSourceFinding, 0),
	}
	for _, issue := range review.Issues {
		// Source Index supports Data and Attribute identities, not a special
		// Source declaration for a candidate key or functional dependency.
		subjects := []sourceindex.Subject{{Kind: "semantic-data", ID: issue.Table}}
		seen := map[string]bool{}
		fieldIDs := append([]string(nil), issue.FieldIDs...)
		sort.Strings(fieldIDs)
		for _, fieldID := range fieldIDs {
			if !seen[fieldID] {
				subjects = append(subjects, sourceindex.Subject{Kind: "semantic-attribute", ID: fieldID})
				seen[fieldID] = true
			}
		}
		for _, subject := range subjects {
			finding := ModelHypothesisSourceFinding{
				IssueID: issue.ID, HypothesisKind: issue.Kind,
				HypothesisID: issue.HypothesisID,
				SubjectKind: subject.Kind, SubjectID: subject.ID,
				Status: "definition-not-indexed",
			}
			definitions := make([]sourceindex.Reference, 0)
			for _, ref := range idx.ReferencesFor(subject) {
				if ref.Role == sourceindex.RoleDefinition {
					definitions = append(definitions, ref)
				}
			}
			switch len(definitions) {
			case 0:
				// A valid Catalog without an indexed definition remains
				// unresolved: the source cannot be guessed from the table ID.
			case 1:
				ref := definitions[0]
				artifact := artifacts[ref.ArtifactPath]
				finding.ArtifactPath = artifact.Path
				finding.ArtifactFingerprint = artifact.Fingerprint
				content, err := read(artifact.Path)
				if err != nil {
					return empty, fmt.Errorf("read source %s: %w", artifact.Path, err)
				}
				if sourceindex.Fingerprint(content) != artifact.Fingerprint {
					return empty, fmt.Errorf("source changed during audit: %s", artifact.Path)
				}
				finding.Status = "exact-subject-anchor-not-confirmed"
				anchored, parseErr := syntax.ParseSemanticFileAnchored(string(content))
				if parseErr == nil && ref.Range.Start.Offset < ref.Range.End.Offset {
					matches := 0
					for _, anchor := range anchored.Anchors {
						if anchor.Kind == subject.Kind && anchor.ID == subject.ID &&
							anchor.Anchor.Start.Offset == ref.Range.Start.Offset &&
							anchor.Anchor.End.Offset == ref.Range.End.Offset {
							matches++
						}
					}
					if matches == 1 {
						finding.Status = "source-range-checked"
						sourceRange := ref.Range
						finding.SourceRange = &sourceRange
					}
				}
			default:
				finding.Status = "ambiguous-definition"
			}
			out.Findings = append(out.Findings, finding)
		}
	}
	// The audit result itself is not accepted evidence: only its bound
	// content and the Workspace snapshot can be rechecked by a later gate.
	out.Fingerprint = digestHypothesisJSON(out)
	return out, nil
}

// AuditLoadedWorkspaceModelHypothesisSourceEvidence prevents a fabricated
// Catalog with a copied revision from impersonating a loaded Workspace.
// Source bytes and Source Index come from one frozen Load snapshot, not from
// independent filesystem reads after the model has been loaded.
func AuditLoadedWorkspaceModelHypothesisSourceEvidence(root string, input Input, proposals ModelHypotheses) (ModelHypothesisSourceAudit, error) {
	var empty ModelHypothesisSourceAudit
	loaded, err := workspace.Load(root)
	if err != nil {
		return empty, fmt.Errorf("load Workspace for model hypotheses: %w", err)
	}
	catalog, err := CatalogFromWorld(loaded.World)
	if err != nil {
		return empty, fmt.Errorf("derive Workspace Catalog: %w", err)
	}
	if !reflect.DeepEqual(catalog, input.Catalog) {
		return empty, fmt.Errorf("illustrative Catalog differs from loaded Workspace semantics")
	}
	idx, read, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		return empty, fmt.Errorf("bind loaded Workspace source snapshot: %w", err)
	}
	return AuditModelHypothesisSourceEvidence(input, proposals, idx, read)
}
