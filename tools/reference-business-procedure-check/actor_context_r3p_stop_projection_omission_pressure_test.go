package main

import (
	"fmt"
	"reflect"
	"testing"
)

const (
	actorContextR3PCategoryStopIdentity             = "stop-identity"
	actorContextR3PCategoryMachineReasons           = "machine-reasons"
	actorContextR3PCategoryKnownAuthorityFacts      = "known-authority-facts"
	actorContextR3PCategoryResolutionNeeds          = "resolution-needs"
	actorContextR3PCategoryRuntimeOutcome           = "runtime-outcome"
	actorContextR3PCategoryDownstreamNotAttempted   = "downstream-not-attempted"
	actorContextR3PCategorySourceRefs               = "source-refs"
	actorContextR3PCategoryResolutionRequest        = "resolution-request"
	actorContextR3PCategoryHandoffRecipientStatus   = "handoff-recipient-status"
	actorContextR3PCategoryAuthorityLimits          = "authority-limits"

	actorContextR3PAudienceHuman = "human"
	actorContextR3PAudienceAgent = "agent"

	actorContextR3PPurposeOrientation         = "orientation"
	actorContextR3PPurposeResolutionInspection = "resolution-inspection"
)

var actorContextR3PAllCategories = []string{
	actorContextR3PCategoryStopIdentity,
	actorContextR3PCategoryMachineReasons,
	actorContextR3PCategoryKnownAuthorityFacts,
	actorContextR3PCategoryResolutionNeeds,
	actorContextR3PCategoryRuntimeOutcome,
	actorContextR3PCategoryDownstreamNotAttempted,
	actorContextR3PCategorySourceRefs,
	actorContextR3PCategoryResolutionRequest,
	actorContextR3PCategoryHandoffRecipientStatus,
	actorContextR3PCategoryAuthorityLimits,
}

type actorContextR3PSelectedStopPayloadProbe struct {
	BusinessAddress        string
	StopStage              string
	StopClass              string
	MachineReasons         []string
	KnownAuthorityFacts    []actorContextR3MAuthorityFactProbe
	ResolutionNeedClasses  []string
	RuntimeOutcome         string
	DownstreamNotAttempted []string
	SourceRefs             []string
	ResolutionRequested    bool
	HandoffRecipientStatus string
	AuthorityLimits        []string
}

type actorContextR3PBoundedStopViewProbe struct {
	Audience             string
	Purpose              string
	Payload              actorContextR3PSelectedStopPayloadProbe
	SelectedCategories   []string
	OmittedCategories    []string
	UnresolvedCategories []string
	DrillDownRefs        []string
}

type actorContextR3PHumanBoundedViewProbe struct {
	Bounded        actorContextR3PBoundedStopViewProbe
	DisplaySummary string
}

type actorContextR3PAgentBoundedViewProbe struct {
	Bounded actorContextR3PBoundedStopViewProbe
}

func TestActorContextR3PHumanOrientationPhysicallyOmitsDetailButMarksOmission(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	view, err := actorContextR3PBuildHumanOrientation(core)
	if err != nil {
		t.Fatal(err)
	}

	if len(core.KnownAuthorityFacts) == 0 || len(core.DownstreamNotAttempted) == 0 {
		t.Fatalf("test setup changed: R3O core lacks detail needed for omission pressure: %+v", core)
	}
	if len(view.Bounded.Payload.KnownAuthorityFacts) != 0 ||
		len(view.Bounded.Payload.DownstreamNotAttempted) != 0 ||
		len(view.Bounded.Payload.AuthorityLimits) != 0 ||
		view.Bounded.Payload.HandoffRecipientStatus != "" {
		t.Fatalf("Human compact view did not physically omit detailed categories: %+v", view.Bounded.Payload)
	}

	for _, category := range []string{
		actorContextR3PCategoryKnownAuthorityFacts,
		actorContextR3PCategoryDownstreamNotAttempted,
		actorContextR3PCategoryResolutionRequest,
		actorContextR3PCategoryHandoffRecipientStatus,
		actorContextR3PCategoryAuthorityLimits,
	} {
		if !actorContextR0Contains(view.Bounded.OmittedCategories, category) {
			t.Fatalf("Human compact view omitted detail without declaring category %q: %+v", category, view.Bounded)
		}
	}

	if !reflect.DeepEqual(view.Bounded.DrillDownRefs, core.SourceRefs) ||
		!reflect.DeepEqual(view.Bounded.Payload.SourceRefs, core.SourceRefs) {
		t.Fatalf("Human omission lost drill-down refs: view=%+v core=%+v", view.Bounded, core.SourceRefs)
	}
	if !reflect.DeepEqual(view.Bounded.UnresolvedCategories, []string{"authorization-action-authority"}) {
		t.Fatalf("Human omission hid current unresolved authority: %+v", view.Bounded.UnresolvedCategories)
	}
}

func TestActorContextR3PSelectedEmptyRuntimeOutcomeIsNotOmitted(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if view.Payload.RuntimeOutcome != "" {
		t.Fatalf("test setup changed: action-authority stop unexpectedly has runtime outcome %q", view.Payload.RuntimeOutcome)
	}
	if !actorContextR0Contains(view.SelectedCategories, actorContextR3PCategoryRuntimeOutcome) {
		t.Fatalf("empty runtime outcome was not represented as selected category: %+v", view.SelectedCategories)
	}
	if actorContextR0Contains(view.OmittedCategories, actorContextR3PCategoryRuntimeOutcome) {
		t.Fatalf("selected-but-empty runtime outcome was mislabeled as omitted")
	}
}

func TestActorContextR3PAgentResolutionViewKeepsMachineDetailWithoutPresentationDirective(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	view, err := actorContextR3PBuildAgentResolution(core)
	if err != nil {
		t.Fatal(err)
	}

	if len(view.Bounded.OmittedCategories) != 0 {
		t.Fatalf("Agent resolution view unexpectedly omitted machine core categories: %+v", view.Bounded.OmittedCategories)
	}
	if !reflect.DeepEqual(view.Bounded.Payload.KnownAuthorityFacts, core.KnownAuthorityFacts) ||
		!reflect.DeepEqual(view.Bounded.Payload.DownstreamNotAttempted, core.DownstreamNotAttempted) ||
		!reflect.DeepEqual(view.Bounded.Payload.AuthorityLimits, core.AuthorityLimits) {
		t.Fatalf("Agent resolution view lost selected machine detail: %+v", view.Bounded.Payload)
	}

	typ := reflect.TypeOf(view)
	for _, forbidden := range []string{
		"DisplaySummary",
		"NextAction",
		"SuggestedAction",
		"HandoffOwner",
		"CanExecute",
		"Authorized",
		"Allowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3P Agent bounded view introduced forbidden presentation/authority shortcut %q", forbidden)
		}
	}
}

func TestActorContextR3PDeniedKeepsSelectedEmptyResolutionNeeds(t *testing.T) {
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation",
		nil,
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-requirement",
				Value:     actorContextR3BAuthorizationRequired,
				SourceRef: "business-procedure-security-boundary",
			},
		},
		[]string{"runtime-observation:test-only-current-authorization"},
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress:    "order-entry.order-handling/BP-06",
		EvaluationStatus:   actorContextR3IEvaluationCurrentObserved,
		ObservationOutcome: "denied",
	})
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if view.Payload.RuntimeOutcome != "denied" {
		t.Fatalf("Human bounded view lost denied runtime outcome: %+v", view.Payload)
	}
	if len(view.UnresolvedCategories) != 0 ||
		len(view.Payload.ResolutionNeedClasses) != 0 {
		t.Fatalf("denied outcome was converted into unresolved resolution need: %+v", view)
	}
	if !actorContextR0Contains(view.SelectedCategories, actorContextR3PCategoryResolutionNeeds) {
		t.Fatalf("empty resolution need was not selected explicitly")
	}
}

func TestActorContextR3PSelectionPartitionFailsClosed(t *testing.T) {
	cases := map[string]struct {
		selected []string
		omitted  []string
	}{
		"overlap": {
			selected: []string{
				actorContextR3PCategoryStopIdentity,
				actorContextR3PCategoryMachineReasons,
				actorContextR3PCategoryResolutionNeeds,
				actorContextR3PCategoryRuntimeOutcome,
				actorContextR3PCategorySourceRefs,
			},
			omitted: append(
				[]string{actorContextR3PCategorySourceRefs},
				actorContextR3PHumanOmittedCategories()...,
			),
		},
		"unknown": {
			selected: append(actorContextR3PAllCategoriesCopy(), "unknown-category"),
			omitted:  nil,
		},
		"missing": {
			selected: actorContextR3PAllCategoriesCopy()[:len(actorContextR3PAllCategories)-1],
			omitted:  nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := actorContextR3PValidateCategoryPartition(tc.selected, tc.omitted); err == nil {
				t.Fatalf("R3P accepted invalid selected/omitted partition: selected=%v omitted=%v", tc.selected, tc.omitted)
			}
		})
	}
}

func actorContextR3PBuildHumanOrientation(
	core actorContextR3OStopProjectionCoreProbe,
) (actorContextR3PHumanBoundedViewProbe, error) {
	bounded, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		return actorContextR3PHumanBoundedViewProbe{}, err
	}
	summary := actorContextR3ORenderHumanStop(core, "concise").DisplaySummary
	return actorContextR3PHumanBoundedViewProbe{
		Bounded:        bounded,
		DisplaySummary: summary,
	}, nil
}

func actorContextR3PBuildAgentResolution(
	core actorContextR3OStopProjectionCoreProbe,
) (actorContextR3PAgentBoundedViewProbe, error) {
	bounded, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		return actorContextR3PAgentBoundedViewProbe{}, err
	}
	return actorContextR3PAgentBoundedViewProbe{Bounded: bounded}, nil
}

func actorContextR3PSelectBoundedStopView(
	core actorContextR3OStopProjectionCoreProbe,
	audience string,
	purpose string,
) (actorContextR3PBoundedStopViewProbe, error) {
	var selected []string
	var omitted []string

	switch {
	case audience == actorContextR3PAudienceHuman &&
		purpose == actorContextR3PPurposeOrientation:
		selected = []string{
			actorContextR3PCategoryStopIdentity,
			actorContextR3PCategoryMachineReasons,
			actorContextR3PCategoryResolutionNeeds,
			actorContextR3PCategoryRuntimeOutcome,
			actorContextR3PCategorySourceRefs,
		}
		omitted = actorContextR3PHumanOmittedCategories()
	case audience == actorContextR3PAudienceAgent &&
		purpose == actorContextR3PPurposeResolutionInspection:
		selected = actorContextR3PAllCategoriesCopy()
	default:
		return actorContextR3PBoundedStopViewProbe{}, fmt.Errorf(
			"unsupported R3P audience/purpose %q/%q",
			audience,
			purpose,
		)
	}

	if err := actorContextR3PValidateCategoryPartition(selected, omitted); err != nil {
		return actorContextR3PBoundedStopViewProbe{}, err
	}
	if len(omitted) > 0 && len(core.SourceRefs) == 0 {
		return actorContextR3PBoundedStopViewProbe{}, fmt.Errorf("bounded R3P omission requires drill-down refs")
	}

	payload := actorContextR3PProjectSelectedCategories(core, selected)
	return actorContextR3PBoundedStopViewProbe{
		Audience:             audience,
		Purpose:              purpose,
		Payload:              payload,
		SelectedCategories:   actorContextR0SortedUnique(selected),
		OmittedCategories:    actorContextR0SortedUnique(omitted),
		UnresolvedCategories: actorContextR0SortedUnique(append([]string(nil), core.ResolutionNeedClasses...)),
		DrillDownRefs:        actorContextR0SortedUnique(append([]string(nil), core.SourceRefs...)),
	}, nil
}

func actorContextR3PProjectSelectedCategories(
	core actorContextR3OStopProjectionCoreProbe,
	selected []string,
) actorContextR3PSelectedStopPayloadProbe {
	has := func(category string) bool {
		return actorContextR0Contains(selected, category)
	}
	var result actorContextR3PSelectedStopPayloadProbe

	if has(actorContextR3PCategoryStopIdentity) {
		result.BusinessAddress = core.BusinessAddress
		result.StopStage = core.StopStage
		result.StopClass = core.StopClass
	}
	if has(actorContextR3PCategoryMachineReasons) {
		result.MachineReasons = append([]string(nil), core.MachineReasons...)
	}
	if has(actorContextR3PCategoryKnownAuthorityFacts) {
		result.KnownAuthorityFacts = append([]actorContextR3MAuthorityFactProbe(nil), core.KnownAuthorityFacts...)
	}
	if has(actorContextR3PCategoryResolutionNeeds) {
		result.ResolutionNeedClasses = append([]string(nil), core.ResolutionNeedClasses...)
	}
	if has(actorContextR3PCategoryRuntimeOutcome) {
		result.RuntimeOutcome = core.RuntimeOutcome
	}
	if has(actorContextR3PCategoryDownstreamNotAttempted) {
		result.DownstreamNotAttempted = append([]string(nil), core.DownstreamNotAttempted...)
	}
	if has(actorContextR3PCategorySourceRefs) {
		result.SourceRefs = append([]string(nil), core.SourceRefs...)
	}
	if has(actorContextR3PCategoryResolutionRequest) {
		result.ResolutionRequested = core.ResolutionRequested
	}
	if has(actorContextR3PCategoryHandoffRecipientStatus) {
		result.HandoffRecipientStatus = core.HandoffRecipientStatus
	}
	if has(actorContextR3PCategoryAuthorityLimits) {
		result.AuthorityLimits = append([]string(nil), core.AuthorityLimits...)
	}
	return result
}

func actorContextR3PValidateCategoryPartition(
	selected []string,
	omitted []string,
) error {
	known := map[string]bool{}
	for _, category := range actorContextR3PAllCategories {
		known[category] = true
	}
	seen := map[string]string{}
	for _, item := range []struct {
		name   string
		values []string
	}{
		{name: "selected", values: selected},
		{name: "omitted", values: omitted},
	} {
		for _, category := range item.values {
			if !known[category] {
				return fmt.Errorf("unknown R3P category %q", category)
			}
			if previous, ok := seen[category]; ok {
				return fmt.Errorf("R3P category %q appears in both/duplicate partitions %q/%q", category, previous, item.name)
			}
			seen[category] = item.name
		}
	}
	if len(seen) != len(known) {
		return fmt.Errorf("R3P category partition is incomplete: got=%d want=%d", len(seen), len(known))
	}
	return nil
}

func actorContextR3PAllCategoriesCopy() []string {
	return append([]string(nil), actorContextR3PAllCategories...)
}

func actorContextR3PHumanOmittedCategories() []string {
	return []string{
		actorContextR3PCategoryKnownAuthorityFacts,
		actorContextR3PCategoryDownstreamNotAttempted,
		actorContextR3PCategoryResolutionRequest,
		actorContextR3PCategoryHandoffRecipientStatus,
		actorContextR3PCategoryAuthorityLimits,
	}
}
