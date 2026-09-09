package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestBusinessAuthoringMigrationPreviewMayReadLegacySourcesWithoutMakingCandidateAuthoritative(t *testing.T) {
	root := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}
	legacyO1, legacySet := currentLegacyBusinessAuthoritySourcesProbe(t, root)

	// Migration preview may read the currently authoritative legacy sources to
	// construct and compare a candidate. The candidate is not yet authoritative,
	// so this is not a multiple-source-of-truth state.
	if err := validateBusinessAuthoringAuthorityCutoverProbe(candidate, legacyO1, legacySet, false, true); err != nil {
		t.Fatalf("legacy authorityからcandidate previewを作る段階をdual authorityと誤判定しました: %v", err)
	}

	bundle := businessAuthoringBundleProbe{
		Procedures: append([]inlineO1FixtureProbe(nil), candidate.Procedures...),
		Handoffs:   candidate.Handoffs,
	}
	if _, err := deriveBusinessAuthoringBundleProbe(bundle); err != nil {
		t.Fatalf("migration preview candidate自体がnormalized Business sourceとして不正です: %v", err)
	}
}

func TestBusinessAuthoringAuthorityCutoverRejectsLegacyAndCandidateO1AsSimultaneousAuthorities(t *testing.T) {
	root := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}
	legacyO1, legacySet := currentLegacyBusinessAuthoritySourcesProbe(t, root)

	if err := validateBusinessAuthoringAuthorityCutoverProbe(candidate, legacyO1, legacySet, true, true); err == nil {
		t.Fatal("inline O1 candidateとlegacy O1 overlayを同時authorityとして受理しました")
	}

	// Same Stepで値が食い違う場合も、どちらを正とするか推測しない。
	conflicting := legacyO1
	conflicting.Steps = append([]applicationPatternTargetOverlayStepProbe(nil), legacyO1.Steps...)
	if len(conflicting.Steps) == 0 {
		t.Fatal("test setup: legacy O1 overlay Stepがありません")
	}
	conflicting.Steps[0].ApplicationPatternTargets = []string{"order_entry.order_record"}
	if err := validateBusinessAuthoringAuthorityCutoverProbe(candidate, conflicting, referenceSet{}, true, true); err == nil {
		t.Fatal("conflicting legacy O1 / inline O1をmigration mergeとして黙って解決しました")
	}
}

func TestBusinessAuthoringAuthorityCutoverRejectsLegacyAndStandaloneHandoffAsSimultaneousAuthorities(t *testing.T) {
	root := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}
	_, legacySet := currentLegacyBusinessAuthoritySourcesProbe(t, root)

	if err := validateBusinessAuthoringAuthorityCutoverProbe(candidate, applicationPatternTargetOverlayProbe{}, legacySet, true, true); err == nil {
		t.Fatal("standalone Handoff candidateとlegacy Reference Set Handoffを同時authorityとして受理しました")
	}

	// Endpointが同じでconditionだけ違う場合も、二sourceのどちらかを
	// 自動的に優先せずauthority conflictとして扱う。
	conflicting := legacySet
	conflicting.Handoffs = append([]referenceSetHandoff(nil), legacySet.Handoffs...)
	if len(conflicting.Handoffs) == 0 {
		t.Fatal("test setup: legacy Handoffがありません")
	}
	conflicting.Handoffs[0].Condition = "conflicting legacy migration condition"
	if err := validateBusinessAuthoringAuthorityCutoverProbe(candidate, applicationPatternTargetOverlayProbe{}, conflicting, true, true); err == nil {
		t.Fatal("same-endpoint Handoff authority conflictをcondition mergeで黙って解決しました")
	}
}

func TestBusinessAuthoringAuthorityCutoverAllowsCandidateOnlyAfterAcceptance(t *testing.T) {
	root := repoPath()
	candidate, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}
	legacyO1, legacySet := currentLegacyBusinessAuthoritySourcesProbe(t, root)

	if err := validateBusinessAuthoringAuthorityCutoverProbe(candidate, legacyO1, legacySet, true, false); err != nil {
		t.Fatalf("accepted candidate-only authorityをrejectしました: %v", err)
	}

	bundle := businessAuthoringBundleProbe{
		Procedures: append([]inlineO1FixtureProbe(nil), candidate.Procedures...),
		Handoffs:   candidate.Handoffs,
	}
	if _, err := deriveBusinessAuthoringBundleProbe(bundle); err != nil {
		t.Fatalf("candidate-only authorityをreload / normalizeできません: %v", err)
	}
}

func currentLegacyBusinessAuthoritySourcesProbe(t *testing.T, root string) (applicationPatternTargetOverlayProbe, referenceSet) {
	t.Helper()
	base := filepath.Join(root, "reference", "business-system-design")
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(base, "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := readReferenceSet(filepath.Join(base, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	return overlay, set
}

func validateBusinessAuthoringAuthorityCutoverProbe(
	candidate businessAuthoringMigrationResultProbe,
	legacyO1 applicationPatternTargetOverlayProbe,
	legacySet referenceSet,
	candidateAuthoritative bool,
	legacyAuthoritative bool,
) error {
	if !candidateAuthoritative || !legacyAuthoritative {
		return nil
	}

	inlineTargets := map[string]string{}
	for _, procedure := range candidate.Procedures {
		for _, step := range procedure.Steps {
			if step.ProtectedApplicationPattern == "" {
				continue
			}
			inlineTargets[procedure.Procedure.ID+"/"+step.ID] = step.ProtectedApplicationPattern
		}
	}
	if legacyO1.Procedure != "" {
		for _, step := range legacyO1.Steps {
			key := legacyO1.Procedure + "/" + step.Step
			if target, exists := inlineTargets[key]; exists {
				return fmt.Errorf("dual protected-target authority for %s: inline=%q legacy=%v", key, target, step.ApplicationPatternTargets)
			}
		}
	}

	candidateHandoffs := map[string]referenceSetHandoff{}
	for _, handoff := range candidate.Handoffs.Handoffs {
		key := businessHandoffEndpointAuthorityKeyProbe(handoff)
		candidateHandoffs[key] = handoff
	}
	for _, handoff := range legacySet.Handoffs {
		key := businessHandoffEndpointAuthorityKeyProbe(handoff)
		if candidateHandoff, exists := candidateHandoffs[key]; exists {
			return fmt.Errorf("dual Business Handoff authority for %s: candidate=%q legacy=%q", key, candidateHandoff.Condition, handoff.Condition)
		}
	}
	return nil
}

func businessHandoffEndpointAuthorityKeyProbe(handoff referenceSetHandoff) string {
	return handoff.From.Procedure + "/" + handoff.From.Step + "->" + handoff.To.Procedure + "/" + handoff.To.Step
}
