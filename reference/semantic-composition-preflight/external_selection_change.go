package semanticcompositionpreflight

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

const externalSelectionChangeSchema = "tei.reference.external-semantic-selection-change/v0"

var externalSelectionDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ExternalSelectionChange is a candidate for changing which external Semantic
// namespaces a project explicitly adopts. It is not arbitrary Evidence, a
// publisher signature, a Host decision, or an additional Accepted World.
type ExternalSelectionChange struct {
	ApplicationID         string                    `json:"applicationId"`
	BaseModelRevision     string                    `json:"baseModelRevision"`
	BaseAuthorityRevision string                    `json:"baseAuthorityRevision"`
	Before                []ProjectMeaningSelection `json:"before"`
	After                 []ProjectMeaningSelection `json:"after"`
}

// ExternalSelectionPreview is an immutable-by-convention, read-only projection.
// Its fingerprint identifies the proposed change; it does not confer authority.
type ExternalSelectionPreview struct {
	Fingerprint           string                    `json:"fingerprint"`
	BaseModelRevision     string                    `json:"baseModelRevision"`
	BaseAuthorityRevision string                    `json:"baseAuthorityRevision"`
	Before                []ProjectMeaningSelection `json:"before"`
	After                 []ProjectMeaningSelection `json:"after"`
}

// ExternalSelectionEvaluation is only the output of a synthetic decision
// exercise. In particular, an "accept" evaluation does NOT create a trusted
// CurrentProjectMeaning receipt or update a Host-owned decision.
type ExternalSelectionEvaluation struct {
	Decision    string                    `json:"decision"`
	Fingerprint string                    `json:"fingerprint"`
	Proposed    []ProjectMeaningSelection `json:"proposed,omitempty"`
}

// PreviewExternalSelectionChange demonstrates a meaning-bearing selection
// delta. The caller MUST separately verify the current Source Index, Host
// provenance/revocation and external Provider contents. Passing a fabricated
// current receipt or source revision here never creates acceptance authority.
func PreviewExternalSelectionChange(
	change ExternalSelectionChange,
	current CurrentProjectMeaning,
	actualSourceRevision string,
) (ExternalSelectionPreview, error) {
	source := current.Source
	if source.ApplicationID == "" || strings.TrimSpace(source.DecisionRef) == "" ||
		strings.TrimSpace(source.AuthorityRevision) == "" ||
		!externalSelectionDigestPattern.MatchString(source.ModelRevision) {
		return ExternalSelectionPreview{}, errors.New("EXTERNAL_SELECTION_CURRENT_DECISION_INVALID")
	}
	if change.ApplicationID != source.ApplicationID || change.ApplicationID == "" {
		return ExternalSelectionPreview{}, errors.New("EXTERNAL_SELECTION_APPLICATION_MISMATCH")
	}
	if actualSourceRevision != source.ModelRevision ||
		change.BaseModelRevision != source.ModelRevision {
		return ExternalSelectionPreview{}, errors.New("EXTERNAL_SELECTION_SOURCE_STALE")
	}
	if change.BaseAuthorityRevision != source.AuthorityRevision {
		return ExternalSelectionPreview{}, errors.New("EXTERNAL_SELECTION_AUTHORITY_STALE")
	}

	currentSet, err := normalizeExternalSelectionSet(current.Selected)
	if err != nil {
		return ExternalSelectionPreview{}, fmt.Errorf("EXTERNAL_SELECTION_CURRENT_SET_INVALID: %w", err)
	}
	before, err := normalizeExternalSelectionSet(change.Before)
	if err != nil {
		return ExternalSelectionPreview{}, fmt.Errorf("EXTERNAL_SELECTION_BEFORE_INVALID: %w", err)
	}
	after, err := normalizeExternalSelectionSet(change.After)
	if err != nil {
		return ExternalSelectionPreview{}, fmt.Errorf("EXTERNAL_SELECTION_AFTER_INVALID: %w", err)
	}
	if !reflect.DeepEqual(before, currentSet) {
		return ExternalSelectionPreview{}, errors.New("EXTERNAL_SELECTION_BEFORE_STALE")
	}
	if reflect.DeepEqual(before, after) {
		return ExternalSelectionPreview{}, errors.New("EXTERNAL_SELECTION_NO_CHANGE")
	}

	// The revision, before/after choice and typed identities are semantic
	// inputs. Unlike the existing change Candidate's Evidence, these MUST
	// contribute to a distinct selection-change fingerprint.
	input := struct {
		Schema                string                    `json:"schema"`
		ApplicationID         string                    `json:"applicationId"`
		BaseModelRevision     string                    `json:"baseModelRevision"`
		BaseAuthorityRevision string                    `json:"baseAuthorityRevision"`
		Before                []ProjectMeaningSelection `json:"before"`
		After                 []ProjectMeaningSelection `json:"after"`
	}{
		Schema:                externalSelectionChangeSchema,
		ApplicationID:         change.ApplicationID,
		BaseModelRevision:     change.BaseModelRevision,
		BaseAuthorityRevision: change.BaseAuthorityRevision,
		Before:                before,
		After:                 after,
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return ExternalSelectionPreview{}, fmt.Errorf("EXTERNAL_SELECTION_FINGERPRINT_UNAVAILABLE: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return ExternalSelectionPreview{
		Fingerprint:           "sha256:" + hex.EncodeToString(sum[:]),
		BaseModelRevision:     change.BaseModelRevision,
		BaseAuthorityRevision: change.BaseAuthorityRevision,
		Before:                before,
		After:                 after,
	}, nil
}

// EvaluateExternalSelectionDecision rechecks the candidate against CURRENT
// inputs, even when the user already reviewed a Preview. The result is an
// intention for a future Shared Acceptance Host transaction, not that transaction.
func EvaluateExternalSelectionDecision(
	change ExternalSelectionChange,
	current CurrentProjectMeaning,
	actualSourceRevision, expectedFingerprint, decision string,
) (ExternalSelectionEvaluation, error) {
	preview, err := PreviewExternalSelectionChange(change, current, actualSourceRevision)
	if err != nil {
		return ExternalSelectionEvaluation{}, err
	}
	if expectedFingerprint == "" || preview.Fingerprint != expectedFingerprint {
		return ExternalSelectionEvaluation{}, errors.New("EXTERNAL_SELECTION_REVIEW_STALE")
	}
	if decision != "accept" && decision != "reject" {
		return ExternalSelectionEvaluation{}, errors.New("EXTERNAL_SELECTION_DECISION_INVALID")
	}
	out := ExternalSelectionEvaluation{Decision: decision, Fingerprint: preview.Fingerprint}
	if decision == "accept" {
		out.Proposed = append([]ProjectMeaningSelection(nil), preview.After...)
	}
	return out, nil
}

func normalizeExternalSelectionSet(input []ProjectMeaningSelection) ([]ProjectMeaningSelection, error) {
	result := make([]ProjectMeaningSelection, 0, len(input))
	aliases := map[string]bool{}
	namespaces := map[string]bool{}
	for _, selected := range input {
		ref := selected.Resource
		if selected.Alias == "" || strings.TrimSpace(selected.Alias) != selected.Alias ||
			strings.ContainsAny(selected.Alias, " /\\:\t\r\n") ||
			ref.Scope == "" || strings.TrimSpace(ref.Scope) != ref.Scope ||
			strings.ContainsAny(ref.Scope, "\t\r\n") ||
			ref.Subject.Kind != "semantic-namespace" || ref.Subject.ID == "" ||
			ref.Version != "" || !externalSelectionDigestPattern.MatchString(selected.SemanticDigest) {
			return nil, errors.New("malformed Path-import Semantic selection")
		}
		if aliases[selected.Alias] || namespaces[ref.Subject.ID] {
			return nil, errors.New("duplicate alias or Semantic namespace")
		}
		aliases[selected.Alias] = true
		namespaces[ref.Subject.ID] = true
		result = append(result, selected)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Alias < result[j].Alias })
	return result, nil
}
