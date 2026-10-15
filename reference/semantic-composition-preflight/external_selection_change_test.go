package semanticcompositionpreflight

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

// This is a synthetic selection-delta exercise, NOT a trusted Host write.
func externalSelectionFixture(t *testing.T) (ExternalSelectionChange, CurrentProjectMeaning, string) {
	t.Helper()
	_, _, _, index, reader := currentProjectMeaningFixture(t)
	current := reader.snapshot
	base := current.Source.ModelRevision
	if base == "" || base != current.Source.ModelRevision || len(index.Artifacts) == 0 {
		t.Fatal("fixture has no current managed Source identity")
	}
	before := append([]ProjectMeaningSelection(nil), current.Selected...)
	after := append([]ProjectMeaningSelection(nil), before...)
	after[0].SemanticDigest = "sha256:" + strings.Repeat("1", 64)
	return ExternalSelectionChange{
		ApplicationID:         current.Source.ApplicationID,
		BaseModelRevision:     base,
		BaseAuthorityRevision: current.Source.AuthorityRevision,
		Before:                before,
		After:                 after,
	}, current, base
}

func TestExternalSelectionChangeHasMeaningBearingDeterministicFingerprint(t *testing.T) {
	change, current, revision := externalSelectionFixture(t)
	preview, err := PreviewExternalSelectionChange(change, current, revision)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(preview.Fingerprint, "sha256:") ||
		len(preview.Fingerprint) != 71 || preview.Before[0].SemanticDigest == preview.After[0].SemanticDigest {
		t.Fatalf("selection change was not fingerprinted: %+v", preview)
	}
	different := change
	different.After = append([]ProjectMeaningSelection(nil), change.After...)
	different.After[0].SemanticDigest = "sha256:" + strings.Repeat("2", 64)
	other, err := PreviewExternalSelectionChange(different, current, revision)
	if err != nil || preview.Fingerprint == other.Fingerprint {
		t.Fatalf("new meaning digest must change selection fingerprint: %v", err)
	}

	// Input order must not change the fingerprint of the same selection set.
	extra := ProjectMeaningSelection{
		Alias: "other",
		Resource: modelref.Ref{
			Scope: "provider-domain:other",
			Subject: subject.Subject{Kind: "semantic-namespace", ID: "other"},
		},
		SemanticDigest: "sha256:" + strings.Repeat("3", 64),
	}
	current.Selected = append(current.Selected, extra)
	change.Before = append(change.Before, extra)
	change.After = append(change.After, extra)
	sorted, err := PreviewExternalSelectionChange(change, current, revision)
	if err != nil {
		t.Fatal(err)
	}
	change.Before[0], change.Before[1] = change.Before[1], change.Before[0]
	change.After[0], change.After[1] = change.After[1], change.After[0]
	reordered, err := PreviewExternalSelectionChange(change, current, revision)
	if err != nil || sorted.Fingerprint != reordered.Fingerprint {
		t.Fatalf("selection order changed meaning fingerprint: %v", err)
	}
}

func TestExternalSelectionChangeRejectsStaleAndMalformedChoices(t *testing.T) {
	cases := []struct {
		name string
		edit func(*ExternalSelectionChange, *CurrentProjectMeaning, *string)
		want string
	}{
		{"stale-managed-source", func(_ *ExternalSelectionChange, _ *CurrentProjectMeaning, rev *string) {
			*rev = "sha256:" + strings.Repeat("a", 64)
		}, "EXTERNAL_SELECTION_SOURCE_STALE"},
		{"stale-candidate-base", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.BaseModelRevision = "sha256:" + strings.Repeat("b", 64)
		}, "EXTERNAL_SELECTION_SOURCE_STALE"},
		{"stale-authority", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.BaseAuthorityRevision = "other-revision"
		}, "EXTERNAL_SELECTION_AUTHORITY_STALE"},
		{"revoked-host-provenance", func(_ *ExternalSelectionChange, current *CurrentProjectMeaning, _ *string) {
			current.Source.DecisionRef = ""
		}, "EXTERNAL_SELECTION_CURRENT_DECISION_INVALID"},
		{"other-application", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.ApplicationID = "other"
		}, "EXTERNAL_SELECTION_APPLICATION_MISMATCH"},
		{"stale-before", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.Before[0].SemanticDigest = "sha256:" + strings.Repeat("5", 64)
		}, "EXTERNAL_SELECTION_BEFORE_STALE"},
		{"no-change", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.After[0].SemanticDigest = c.Before[0].SemanticDigest
		}, "EXTERNAL_SELECTION_NO_CHANGE"},
		{"invalid-digest", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.After[0].SemanticDigest = "not-a-digest"
		}, "EXTERNAL_SELECTION_AFTER_INVALID"},
		{"blank-scope", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.After[0].Resource.Scope = ""
		}, "EXTERNAL_SELECTION_AFTER_INVALID"},
		{"wrong-kind", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.After[0].Resource.Subject.Kind = "semantic-data"
		}, "EXTERNAL_SELECTION_AFTER_INVALID"},
		{"unsupported-version", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.After[0].Resource.Version = "1.0.0"
		}, "EXTERNAL_SELECTION_AFTER_INVALID"},
		{"duplicate-alias", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			c.After = append(c.After, c.After[0])
		}, "EXTERNAL_SELECTION_AFTER_INVALID"},
		{"duplicate-namespace", func(c *ExternalSelectionChange, _ *CurrentProjectMeaning, _ *string) {
			copy := c.After[0]
			copy.Alias = "second"
			c.After = append(c.After, copy)
		}, "EXTERNAL_SELECTION_AFTER_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			change, current, revision := externalSelectionFixture(t)
			tc.edit(&change, &current, &revision)
			preview, err := PreviewExternalSelectionChange(change, current, revision)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected refusal %q, got: %+v / %v", tc.want, preview, err)
			}
			if !reflect.DeepEqual(preview, ExternalSelectionPreview{}) {
				t.Fatal("rejected change exposed a successful Preview")
			}
		})
	}
}

func TestExternalSelectionDecisionRequiresFreshReviewAndNeverWritesHostState(t *testing.T) {
	change, current, revision := externalSelectionFixture(t)
	original := current.Selected[0]
	preview, err := PreviewExternalSelectionChange(change, current, revision)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, expected, decision, want string
	}{
		{"missing-review", "", "accept", "EXTERNAL_SELECTION_REVIEW_STALE"},
		{"wrong-review", "sha256:" + strings.Repeat("0", 64), "accept", "EXTERNAL_SELECTION_REVIEW_STALE"},
		{"unknown-decision", preview.Fingerprint, "approve", "EXTERNAL_SELECTION_DECISION_INVALID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EvaluateExternalSelectionDecision(change, current, revision, tc.expected, tc.decision)
			if err == nil || !strings.Contains(err.Error(), tc.want) ||
				!reflect.DeepEqual(result, ExternalSelectionEvaluation{}) {
				t.Fatalf("expected %q without resulting decision: %+v / %v", tc.want, result, err)
			}
		})
	}
	rejected, err := EvaluateExternalSelectionDecision(change, current, revision, preview.Fingerprint, "reject")
	if err != nil || rejected.Decision != "reject" || len(rejected.Proposed) != 0 {
		t.Fatalf("Reject should not propose any adopted selection: %+v / %v", rejected, err)
	}
	accepted, err := EvaluateExternalSelectionDecision(change, current, revision, preview.Fingerprint, "accept")
	if err != nil || accepted.Decision != "accept" || len(accepted.Proposed) != 1 ||
		accepted.Proposed[0].SemanticDigest != change.After[0].SemanticDigest {
		t.Fatalf("synthetic accept evaluation did not keep the exact reviewed choice: %+v / %v", accepted, err)
	}
	if current.Selected[0] != original || change.Before[0] != original {
		t.Fatal("Preview/Reject/Accept evaluation mutated the caller-owned current Host state")
	}
	// Concurrent Host Authority change between review and decision is refused.
	current.Source.AuthorityRevision = "superseded"
	if result, err := EvaluateExternalSelectionDecision(change, current, revision, preview.Fingerprint, "accept"); err == nil ||
		!strings.Contains(err.Error(), "EXTERNAL_SELECTION_AUTHORITY_STALE") ||
		!reflect.DeepEqual(result, ExternalSelectionEvaluation{}) {
		t.Fatalf("concurrent Host update must refuse the reviewed choice: %+v / %v", result, err)
	}
}

func TestExternalSelectionRemovalIsAChangeButNotYetHostRevocation(t *testing.T) {
	change, current, revision := externalSelectionFixture(t)
	change.After = nil
	preview, err := PreviewExternalSelectionChange(change, current, revision)
	if err != nil {
		t.Fatal(err)
	}
	eval, err := EvaluateExternalSelectionDecision(change, current, revision, preview.Fingerprint, "accept")
	if err != nil || eval.Decision != "accept" || len(eval.Proposed) != 0 {
		t.Fatalf("removing all selected namespaces should be a reviewable proposal: %+v / %v", eval, err)
	}
	if len(current.Selected) != 1 {
		t.Fatal("a selection proposal must not revoke the real current Host receipt")
	}
}
