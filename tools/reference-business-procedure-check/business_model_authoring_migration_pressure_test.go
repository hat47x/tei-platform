package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

type businessAuthoringMigrationResultProbe struct {
	Procedures []inlineO1FixtureProbe        `json:"procedures"`
	Handoffs   standaloneHandoffSourceProbe `json:"handoffs"`
}

func TestBusinessAuthoringMigrationPreservesCurrentNormalizedMeaning(t *testing.T) {
	root := repoPath()
	result, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}
	migrated, err := deriveBusinessAuthoringBundleProbe(businessAuthoringBundleProbe(result))
	if err != nil {
		t.Fatal(err)
	}

	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)
	current, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(migrated, current) {
		t.Fatalf("authoring migrationでBusiness semantic meaningが変化しました: migrated=%+v current=%+v", migrated, current)
	}
}

func TestBusinessAuthoringMigrationMovesO1IntoProcedureAndHandoffOutOfReferenceManifest(t *testing.T) {
	root := repoPath()
	result, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}

	primary, ok := inlineO1ProcedureByIDMigrationProbe(result.Procedures, "order-entry.order-handling")
	if !ok {
		t.Fatal("migrated primary Procedureがありません")
	}
	bp06, ok := inlineO1StepByIDProbe(primary, "BP-06")
	if !ok {
		t.Fatal("migrated BP-06がありません")
	}
	if bp06.ProtectedApplicationPattern != "order_entry.submit_order" {
		t.Fatalf("O1 targetをinline Procedure sourceへ移せません: %+v", bp06)
	}

	if len(result.Handoffs.Handoffs) != 1 {
		t.Fatalf("standalone Handoff migration result=%d want=1", len(result.Handoffs.Handoffs))
	}
	if result.Handoffs.Handoffs[0].From.Procedure != "order-entry.order-handling" || result.Handoffs.Handoffs[0].To.Procedure != "order-entry.follow-up" {
		t.Fatalf("standalone Handoff meaningを失いました: %+v", result.Handoffs.Handoffs[0])
	}
}

func TestBusinessAuthoringMigrationDoesNotCopyReferenceArtifactLocatorsIntoNewSemanticSources(t *testing.T) {
	root := repoPath()
	result, err := migrateCurrentReferenceAuthoringProbe(t, root)
	if err != nil {
		t.Fatal(err)
	}

	for _, procedure := range result.Procedures {
		typ := reflect.TypeOf(procedure)
		for _, forbidden := range []string{"Path", "Artifact", "Locator", "ReferenceSet"} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("migrated Procedure sourceへReference locator field %qをコピーしました", forbidden)
			}
		}
	}
	handoffType := reflect.TypeOf(result.Handoffs)
	for _, forbidden := range []string{"Path", "Artifact", "Locator", "Procedures"} {
		if _, found := handoffType.FieldByName(forbidden); found {
			t.Fatalf("migrated Handoff sourceへReference assembly field %qをコピーしました", forbidden)
		}
	}
}

func TestBusinessAuthoringMigrationFailsClosedWhenLegacyO1NeedsMultipleTargets(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(overlay.Steps) == 0 {
		t.Fatal("test setup: O1 overlay Stepがありません")
	}
	overlay.Steps = append([]applicationPatternTargetOverlayStepProbe(nil), overlay.Steps...)
	overlay.Steps[0].ApplicationPatternTargets = []string{"order_entry.submit_order", "order_entry.order_record"}

	if _, err := migrateProcedureWithLegacyO1OverlayProbe(primary, overlay); err == nil {
		t.Fatal("current singular O1 candidateで表現できないlegacy multi-targetを黙ってmigrationしました")
	}
}

func TestBusinessAuthoringMigrationLeavesLegacyInputsUnchanged(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"))
	if err != nil {
		t.Fatal(err)
	}
	beforeProcedure := primary
	beforeOverlay := overlay

	if _, err := migrateProcedureWithLegacyO1OverlayProbe(primary, overlay); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(primary, beforeProcedure) || !reflect.DeepEqual(overlay, beforeOverlay) {
		t.Fatal("migration previewがlegacy authoritative inputsをin-place mutationしました")
	}
}

func migrateCurrentReferenceAuthoringProbe(t *testing.T, root string) (businessAuthoringMigrationResultProbe, error) {
	t.Helper()
	base := filepath.Join(root, "reference", "business-system-design")
	primary, err := readFixture(filepath.Join(base, "order-entry-business-procedure.json"))
	if err != nil {
		return businessAuthoringMigrationResultProbe{}, err
	}
	followUp, err := readFixture(filepath.Join(base, "order-entry-follow-up-procedure.json"))
	if err != nil {
		return businessAuthoringMigrationResultProbe{}, err
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(filepath.Join(base, "order-entry-protected-target-overlay.json"))
	if err != nil {
		return businessAuthoringMigrationResultProbe{}, err
	}
	set, err := readReferenceSet(filepath.Join(base, "order-entry-procedure-reference-set.json"))
	if err != nil {
		return businessAuthoringMigrationResultProbe{}, err
	}

	migratedPrimary, err := migrateProcedureWithLegacyO1OverlayProbe(primary, overlay)
	if err != nil {
		return businessAuthoringMigrationResultProbe{}, err
	}
	migratedFollowUp := inlineO1CandidateFromFixtureProbe(followUp, nil)

	return businessAuthoringMigrationResultProbe{
		Procedures: []inlineO1FixtureProbe{migratedPrimary, migratedFollowUp},
		Handoffs: standaloneHandoffSourceProbe{
			SchemaVersion: standaloneHandoffSchemaVersion,
			ArtifactRole:  standaloneHandoffArtifactRole,
			Canonical:     false,
			Handoffs:      append([]referenceSetHandoff(nil), set.Handoffs...),
		},
	}, nil
}

func migrateProcedureWithLegacyO1OverlayProbe(base fixture, overlay applicationPatternTargetOverlayProbe) (inlineO1FixtureProbe, error) {
	if overlay.Procedure != base.Procedure.ID {
		return inlineO1FixtureProbe{}, fmt.Errorf("legacy O1 overlay Procedure %q does not match %q", overlay.Procedure, base.Procedure.ID)
	}
	targets := map[string]string{}
	seen := map[string]struct{}{}
	for _, item := range overlay.Steps {
		if _, duplicate := seen[item.Step]; duplicate {
			return inlineO1FixtureProbe{}, fmt.Errorf("duplicate legacy O1 Step %q", item.Step)
		}
		seen[item.Step] = struct{}{}
		if len(item.ApplicationPatternTargets) != 1 {
			return inlineO1FixtureProbe{}, fmt.Errorf("legacy O1 Step %q has %d targets; singular candidate requires exactly one", item.Step, len(item.ApplicationPatternTargets))
		}
		targets[item.Step] = item.ApplicationPatternTargets[0]
	}
	return inlineO1CandidateFromFixtureProbe(base, targets), nil
}

func inlineO1ProcedureByIDMigrationProbe(values []inlineO1FixtureProbe, id string) (inlineO1FixtureProbe, bool) {
	for _, value := range values {
		if value.Procedure.ID == id {
			return value, true
		}
	}
	return inlineO1FixtureProbe{}, false
}
