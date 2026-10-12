package exampleslide

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const modelRuleDraftIntentSchema = "tei.reference.model-rule-draft-intent/v0"

// Prospective rule IDs permit a hyphen in the local part even though today's
// Canonical ID validator does not; these are not yet accepted Semantic IDs.
var proposedRuleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// ModelRuleDraftIntent names explicit, review-bound business-rule declarations.
// Claims are user assertions, not verified human signatures or business truth.
type ModelRuleDraftIntent struct {
	Schema                 string                  `json:"schema"`
	ModelReviewFingerprint string                  `json:"modelReviewFingerprint"`
	SourceAuditFingerprint string                  `json:"sourceAuditFingerprint"`
	Requests               []ModelRuleDraftRequest `json:"requests"`
}

type ModelRuleDraftRequest struct {
	IssueID         string                   `json:"issueId"`
	ProposedRuleID  string                   `json:"proposedRuleId"`
	Rationale       string                   `json:"rationale"`
	ClaimedEvidence []ModelRuleEvidenceClaim `json:"claimedEvidence,omitempty"`
}

type ModelRuleEvidenceClaim struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint"`
	ClaimedBy   string `json:"claimedBy"`
}

// A ModelRuleDeclaration is a read-only staging artifact and NEVER an
// operation accepted by Shared Acceptance. Physical PKs and row IDs are not
// inferred. The SourceBindings locate existing table/attribute declarations,
// not the proposed rule's authoritative source.
type ModelRuleDeclaration struct {
	IssueID         string                         `json:"issueId"`
	ProposedRuleID  string                         `json:"proposedRuleId"`
	Kind            string                         `json:"kind"`
	Table           string                         `json:"table"`
	KeyFields       []string                       `json:"keyFields,omitempty"`
	Determinants    []string                       `json:"determinants,omitempty"`
	Dependents      []string                       `json:"dependents,omitempty"`
	ReviewResult    string                         `json:"reviewResult"`
	ReviewReason    string                         `json:"reviewReason"`
	Status          string                         `json:"status"`
	SourceBindings  []ModelHypothesisSourceFinding `json:"sourceBindings"`
	Rationale       string                         `json:"rationale"`
	ClaimedEvidence []ModelRuleEvidenceClaim       `json:"claimedEvidence,omitempty"`
	Outstanding     []string                       `json:"outstanding"`
}

type ModelRuleDeclarationDraft struct {
	Schema                 string                 `json:"schema"`
	CaseID                 string                 `json:"caseId"`
	ModelRevision          string                 `json:"modelRevision"`
	ModelReviewFingerprint string                 `json:"modelReviewFingerprint"`
	SourceAuditFingerprint string                 `json:"sourceAuditFingerprint"`
	Fingerprint            string                 `json:"fingerprint"`
	Declarations           []ModelRuleDeclaration `json:"declarations"`
}

// DraftLoadedWorkspaceModelRules creates typed key/FD proposal *descriptions*
// only after rebuilding the review and source audit from an actual Workspace
// Load. No candidate is accepted, authorized, signed, or applied to TEI source.
func DraftLoadedWorkspaceModelRules(root string, input Input, proposed ModelHypotheses, intent ModelRuleDraftIntent) (ModelRuleDeclarationDraft, error) {
	var empty ModelRuleDeclarationDraft
	review, err := ReviewModelHypotheses(input, proposed)
	if err != nil {
		return empty, err
	}
	audit, err := AuditLoadedWorkspaceModelHypothesisSourceEvidence(root, input, proposed)
	if err != nil {
		return empty, err
	}
	if audit.ModelReviewFingerprint != review.Fingerprint || audit.HypothesesFingerprint != review.HypothesesFingerprint {
		return empty, fmt.Errorf("source audit does not bind to the exact model review")
	}
	if intent.Schema != modelRuleDraftIntentSchema ||
		intent.ModelReviewFingerprint != review.Fingerprint ||
		intent.SourceAuditFingerprint != audit.Fingerprint {
		return empty, fmt.Errorf("rule intent has unsupported schema or stale review / source audit fingerprint")
	}
	if len(intent.Requests) == 0 {
		return empty, fmt.Errorf("at least one rule declaration request required")
	}
	keyByID := make(map[string]CandidateKeyHypothesis, len(proposed.CandidateKeys))
	for _, h := range proposed.CandidateKeys {
		keyByID[h.ID] = h
	}
	fdByID := make(map[string]FDHypothesis, len(proposed.FunctionalDependencies))
	for _, h := range proposed.FunctionalDependencies {
		fdByID[h.ID] = h
	}
	issues := make(map[string]ModelHypothesisIssue, len(review.Issues))
	for _, issue := range review.Issues {
		issues[issue.ID] = issue
	}
	bindings := make(map[string][]ModelHypothesisSourceFinding)
	for _, finding := range audit.Findings {
		bindings[finding.IssueID] = append(bindings[finding.IssueID], finding)
	}
	out := ModelRuleDeclarationDraft{
		Schema:                 "tei.reference.model-rule-declaration-draft/v0",
		CaseID:                 review.CaseID,
		ModelRevision:          review.ModelRevision,
		ModelReviewFingerprint: review.Fingerprint,
		SourceAuditFingerprint: audit.Fingerprint,
		Declarations:           make([]ModelRuleDeclaration, 0, len(intent.Requests)),
	}
	usedIssue, usedRule := map[string]bool{}, map[string]bool{}
	for _, req := range intent.Requests {
		issue, exists := issues[req.IssueID]
		if !exists || usedIssue[req.IssueID] {
			return empty, fmt.Errorf("unknown or repeated model issue %q", req.IssueID)
		}
		usedIssue[req.IssueID] = true
		ruleID := strings.TrimSpace(req.ProposedRuleID)
		prefix := issue.Table + "."
		if !strings.HasPrefix(ruleID, prefix) {
			return empty, fmt.Errorf("rule %q must belong to semantic Data %q", ruleID, issue.Table)
		}
		if !proposedRuleNamePattern.MatchString(strings.TrimPrefix(ruleID, prefix)) {
			return empty, fmt.Errorf("invalid proposed rule ID %q", ruleID)
		}
		if usedRule[ruleID] {
			return empty, fmt.Errorf("repeated proposed rule ID %q", ruleID)
		}
		usedRule[ruleID] = true
		rationale := strings.TrimSpace(req.Rationale)
		if rationale == "" {
			return empty, fmt.Errorf("%s: explicit rationale required", ruleID)
		}
		claims := append([]ModelRuleEvidenceClaim(nil), req.ClaimedEvidence...)
		seenClaim := map[string]bool{}
		for i := range claims {
			c := &claims[i]
			c.Kind = strings.TrimSpace(c.Kind)
			c.Ref = strings.TrimSpace(c.Ref)
			c.ClaimedBy = strings.TrimSpace(c.ClaimedBy)
			if (c.Kind != "document" && c.Kind != "human-statement") ||
				c.Ref == "" || c.ClaimedBy == "" || !validModelEvidenceFingerprint(c.Fingerprint) {
				return empty, fmt.Errorf("%s: claimedEvidence[%d] has invalid or incomplete unverified claim", ruleID, i)
			}
			key := c.Kind + "\x00" + c.Ref + "\x00" + c.Fingerprint + "\x00" + c.ClaimedBy
			if seenClaim[key] {
				return empty, fmt.Errorf("%s: repeated evidence claim", ruleID)
			}
			seenClaim[key] = true
		}
		sort.Slice(claims, func(i, j int) bool {
			return claims[i].Kind+"\x00"+claims[i].Ref+"\x00"+claims[i].Fingerprint+"\x00"+claims[i].ClaimedBy <
				claims[j].Kind+"\x00"+claims[j].Ref+"\x00"+claims[j].Fingerprint+"\x00"+claims[j].ClaimedBy
		})
		d := ModelRuleDeclaration{
			IssueID: req.IssueID, ProposedRuleID: ruleID,
			Kind: issue.Kind, Table: issue.Table, ReviewResult: issue.Result,
			ReviewReason: issue.Reason, SourceBindings: append([]ModelHypothesisSourceFinding{}, bindings[issue.ID]...),
			Rationale: rationale, ClaimedEvidence: claims,
			Outstanding: []string{
				"independent business rule evidence must be verified",
				"explicit authorized human acceptance and typed semantic mutation are not implemented",
			},
		}
		if issue.Kind == "candidate-key" {
			h, ok := keyByID[issue.HypothesisID]
			if !ok || h.Table != issue.Table {
				return empty, fmt.Errorf("candidate key subject drift")
			}
			d.KeyFields = append([]string(nil), h.Fields...)
			sort.Strings(d.KeyFields)
		} else if issue.Kind == "functional-dependency" {
			h, ok := fdByID[issue.HypothesisID]
			if !ok || h.Table != issue.Table {
				return empty, fmt.Errorf("functional dependency subject drift")
			}
			d.Determinants = append([]string(nil), h.Determinants...)
			d.Dependents = append([]string(nil), h.Dependents...)
			sort.Strings(d.Determinants)
			sort.Strings(d.Dependents)
		} else {
			return empty, fmt.Errorf("unsupported business model hypothesis kind %q", issue.Kind)
		}
		if issue.Result == "contradiction" {
			d.Status = "blocked-counterexample"
			d.Outstanding = append(d.Outstanding, "observed example counterexample must be resolved")
		} else if !allRuleSubjectBindingsChecked(issue, d.SourceBindings) {
			d.Status = "blocked-source-definition"
			d.Outstanding = append(d.Outstanding, "one or more semantic definitions lack exact verified source anchors")
		} else {
			d.Status = "unverified-business-rule"
		}
		out.Declarations = append(out.Declarations, d)
	}
	sort.Slice(out.Declarations, func(i, j int) bool {
		return out.Declarations[i].ProposedRuleID < out.Declarations[j].ProposedRuleID
	})
	out.Fingerprint = digestHypothesisJSON(out)
	return out, nil
}

func allRuleSubjectBindingsChecked(issue ModelHypothesisIssue, refs []ModelHypothesisSourceFinding) bool {
	needed := map[string]bool{"semantic-data\x00" + issue.Table: true}
	for _, attr := range issue.FieldIDs {
		needed["semantic-attribute\x00"+attr] = true
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		key := ref.SubjectKind + "\x00" + ref.SubjectID
		if !needed[key] || seen[key] || ref.Status != "source-range-checked" ||
			ref.ArtifactPath == "" || !validModelEvidenceFingerprint(ref.ArtifactFingerprint) || ref.SourceRange == nil {
			return false
		}
		seen[key] = true
	}
	return len(seen) == len(needed)
}

func validModelEvidenceFingerprint(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, c := range value[len("sha256:"):] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
