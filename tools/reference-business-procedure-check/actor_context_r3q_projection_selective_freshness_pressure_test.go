package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

type actorContextR3QCategoryFingerprintProbe struct {
	Category    string
	Fingerprint string
}

type actorContextR3QFreshnessSnapshotProbe struct {
	Audience                     string
	Purpose                      string
	ProfileFingerprint           string
	RequiredCategoryFingerprints []actorContextR3QCategoryFingerprintProbe
}

func TestActorContextR3QOmittedKnownFactsDoNotStaleHumanButStaleAgent(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	changed := actorContextR3QCloneCore(core)
	changed.KnownAuthorityFacts = append(
		changed.KnownAuthorityFacts,
		actorContextR3MAuthorityFactProbe{
			Kind:      "test-only-extra-fact",
			Value:     "changed",
			SourceRef: "test-only:extra-fact",
		},
	)

	humanBefore := actorContextR3QSnapshotForProfile(t, core, actorContextR3PAudienceHuman, actorContextR3PPurposeOrientation)
	humanAfter := actorContextR3QSnapshotForProfile(t, changed, actorContextR3PAudienceHuman, actorContextR3PPurposeOrientation)
	if !reflect.DeepEqual(humanBefore, humanAfter) {
		t.Fatalf("omitted known-authority-facts staled Human orientation: before=%+v after=%+v", humanBefore, humanAfter)
	}

	agentBefore := actorContextR3QSnapshotForProfile(t, core, actorContextR3PAudienceAgent, actorContextR3PPurposeResolutionInspection)
	agentAfter := actorContextR3QSnapshotForProfile(t, changed, actorContextR3PAudienceAgent, actorContextR3PPurposeResolutionInspection)
	if reflect.DeepEqual(agentBefore, agentAfter) {
		t.Fatal("selected known-authority-facts change did not stale Agent resolution view")
	}
}

func TestActorContextR3QSelectedMeaningChangesStaleHumanAndAgent(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)

	for name, mutate := range map[string]func(*actorContextR3OStopProjectionCoreProbe){
		"machine-reason": func(changed *actorContextR3OStopProjectionCoreProbe) {
			changed.MachineReasons = []string{"authorization-resource-authority-unresolved"}
		},
		"resolution-need": func(changed *actorContextR3OStopProjectionCoreProbe) {
			changed.ResolutionNeedClasses = []string{"authorization-resource-authority"}
		},
		"runtime-outcome": func(changed *actorContextR3OStopProjectionCoreProbe) {
			changed.RuntimeOutcome = "denied"
		},
		"source-ref": func(changed *actorContextR3OStopProjectionCoreProbe) {
			changed.SourceRefs = append(changed.SourceRefs, "test-only:new-source-ref")
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := actorContextR3QCloneCore(core)
			mutate(&changed)

			for _, profile := range []struct {
				audience string
				purpose  string
			}{
				{actorContextR3PAudienceHuman, actorContextR3PPurposeOrientation},
				{actorContextR3PAudienceAgent, actorContextR3PPurposeResolutionInspection},
			} {
				before := actorContextR3QSnapshotForProfile(t, core, profile.audience, profile.purpose)
				after := actorContextR3QSnapshotForProfile(t, changed, profile.audience, profile.purpose)
				if reflect.DeepEqual(before, after) {
					t.Fatalf("selected %s change did not stale %s/%s projection", name, profile.audience, profile.purpose)
				}
			}
		})
	}
}

func TestActorContextR3QOmittedAuthorityLimitsDoNotStaleHuman(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	changed := actorContextR3QCloneCore(core)
	changed.AuthorityLimits = append(changed.AuthorityLimits, "test-only:new-limit")

	before := actorContextR3QSnapshotForProfile(t, core, actorContextR3PAudienceHuman, actorContextR3PPurposeOrientation)
	after := actorContextR3QSnapshotForProfile(t, changed, actorContextR3PAudienceHuman, actorContextR3PPurposeOrientation)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("omitted authority-limits staled Human orientation: before=%+v after=%+v", before, after)
	}
}

func TestActorContextR3QHumanSnapshotContainsOnlySelectedCategoryFingerprints(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := actorContextR3QBuildFreshnessSnapshot(core, view)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(snapshot.RequiredCategoryFingerprints))
	for _, item := range snapshot.RequiredCategoryFingerprints {
		got = append(got, item.Category)
		if actorContextR0Contains(view.OmittedCategories, item.Category) {
			t.Fatalf("omitted category %q leaked into required freshness dependencies", item.Category)
		}
	}
	if !reflect.DeepEqual(
		actorContextR0SortedUnique(got),
		actorContextR0SortedUnique(view.SelectedCategories),
	) {
		t.Fatalf("required category fingerprints do not match selected categories: got=%v selected=%v", got, view.SelectedCategories)
	}
}

func TestActorContextR3QProfilePartitionChangeChangesProfileFingerprint(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := actorContextR3QBuildFreshnessSnapshot(core, view)
	if err != nil {
		t.Fatal(err)
	}

	changed := view
	changed.SelectedCategories = actorContextR3QRemoveString(
		changed.SelectedCategories,
		actorContextR3PCategoryRuntimeOutcome,
	)
	changed.OmittedCategories = append(
		append([]string(nil), changed.OmittedCategories...),
		actorContextR3PCategoryRuntimeOutcome,
	)
	if err := actorContextR3PValidateCategoryPartition(changed.SelectedCategories, changed.OmittedCategories); err != nil {
		t.Fatal(err)
	}
	changedSnapshot, err := actorContextR3QBuildFreshnessSnapshot(core, changed)
	if err != nil {
		t.Fatal(err)
	}

	if baseline.ProfileFingerprint == changedSnapshot.ProfileFingerprint {
		t.Fatal("selected-but-empty -> omitted profile change did not alter ProfileFingerprint")
	}
	if baseline.ProfileFingerprint == "" || changedSnapshot.ProfileFingerprint == "" {
		t.Fatal("R3Q produced empty profile fingerprint")
	}
}

func TestActorContextR3QInvalidPartitionFailsBeforeSnapshot(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	view.SelectedCategories = append(view.SelectedCategories, "unknown-category")

	if _, err := actorContextR3QBuildFreshnessSnapshot(core, view); err == nil {
		t.Fatal("R3Q accepted invalid category partition")
	}
}

func actorContextR3QCurrentBP06Core(t *testing.T) actorContextR3OStopProjectionCoreProbe {
	t.Helper()
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	return core
}

func actorContextR3QSnapshotForProfile(
	t *testing.T,
	core actorContextR3OStopProjectionCoreProbe,
	audience string,
	purpose string,
) actorContextR3QFreshnessSnapshotProbe {
	t.Helper()
	view, err := actorContextR3PSelectBoundedStopView(core, audience, purpose)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := actorContextR3QBuildFreshnessSnapshot(core, view)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func actorContextR3QBuildFreshnessSnapshot(
	core actorContextR3OStopProjectionCoreProbe,
	view actorContextR3PBoundedStopViewProbe,
) (actorContextR3QFreshnessSnapshotProbe, error) {
	if err := actorContextR3PValidateCategoryPartition(
		view.SelectedCategories,
		view.OmittedCategories,
	); err != nil {
		return actorContextR3QFreshnessSnapshotProbe{}, err
	}
	if view.Audience == "" || view.Purpose == "" {
		return actorContextR3QFreshnessSnapshotProbe{}, fmt.Errorf("R3Q projection profile identity is required")
	}

	profilePayload := struct {
		Audience   string
		Purpose    string
		Selected   []string
		Omitted    []string
	}{
		Audience: view.Audience,
		Purpose: view.Purpose,
		Selected: actorContextR0SortedUnique(append([]string(nil), view.SelectedCategories...)),
		Omitted: actorContextR0SortedUnique(append([]string(nil), view.OmittedCategories...)),
	}
	profileFingerprint, err := actorContextR3QHash(profilePayload)
	if err != nil {
		return actorContextR3QFreshnessSnapshotProbe{}, err
	}

	required := make([]actorContextR3QCategoryFingerprintProbe, 0, len(view.SelectedCategories))
	for _, category := range actorContextR0SortedUnique(view.SelectedCategories) {
		payload, err := actorContextR3QCategoryPayload(core, category)
		if err != nil {
			return actorContextR3QFreshnessSnapshotProbe{}, err
		}
		fingerprint, err := actorContextR3QHash(payload)
		if err != nil {
			return actorContextR3QFreshnessSnapshotProbe{}, err
		}
		required = append(required, actorContextR3QCategoryFingerprintProbe{
			Category: category,
			Fingerprint: fingerprint,
		})
	}

	return actorContextR3QFreshnessSnapshotProbe{
		Audience: view.Audience,
		Purpose: view.Purpose,
		ProfileFingerprint: profileFingerprint,
		RequiredCategoryFingerprints: required,
	}, nil
}

func actorContextR3QCategoryPayload(
	core actorContextR3OStopProjectionCoreProbe,
	category string,
) (any, error) {
	switch category {
	case actorContextR3PCategoryStopIdentity:
		return struct {
			BusinessAddress string
			StopStage       string
			StopClass       string
		}{core.BusinessAddress, core.StopStage, core.StopClass}, nil
	case actorContextR3PCategoryMachineReasons:
		return core.MachineReasons, nil
	case actorContextR3PCategoryKnownAuthorityFacts:
		return core.KnownAuthorityFacts, nil
	case actorContextR3PCategoryResolutionNeeds:
		return core.ResolutionNeedClasses, nil
	case actorContextR3PCategoryRuntimeOutcome:
		return core.RuntimeOutcome, nil
	case actorContextR3PCategoryDownstreamNotAttempted:
		return core.DownstreamNotAttempted, nil
	case actorContextR3PCategorySourceRefs:
		return core.SourceRefs, nil
	case actorContextR3PCategoryResolutionRequest:
		return core.ResolutionRequested, nil
	case actorContextR3PCategoryHandoffRecipientStatus:
		return core.HandoffRecipientStatus, nil
	case actorContextR3PCategoryAuthorityLimits:
		return core.AuthorityLimits, nil
	default:
		return nil, fmt.Errorf("unsupported R3Q category %q", category)
	}
}

func actorContextR3QHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", sum[:]), nil
}

func actorContextR3QCloneCore(
	core actorContextR3OStopProjectionCoreProbe,
) actorContextR3OStopProjectionCoreProbe {
	cloned := core
	cloned.MachineReasons = append([]string(nil), core.MachineReasons...)
	cloned.KnownAuthorityFacts = append([]actorContextR3MAuthorityFactProbe(nil), core.KnownAuthorityFacts...)
	cloned.ResolutionNeedClasses = append([]string(nil), core.ResolutionNeedClasses...)
	cloned.DownstreamNotAttempted = append([]string(nil), core.DownstreamNotAttempted...)
	cloned.SourceRefs = append([]string(nil), core.SourceRefs...)
	cloned.AuthorityLimits = append([]string(nil), core.AuthorityLimits...)
	return cloned
}

func actorContextR3QRemoveString(values []string, target string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}
