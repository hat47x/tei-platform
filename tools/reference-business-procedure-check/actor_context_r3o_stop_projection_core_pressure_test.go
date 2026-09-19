package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type actorContextR3OStopProjectionCoreProbe struct {
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

type actorContextR3OHumanStopViewProbe struct {
	Core           actorContextR3OStopProjectionCoreProbe
	DisplaySummary string
}

type actorContextR3OAgentStopViewProbe struct {
	Core actorContextR3OStopProjectionCoreProbe
}

func TestActorContextR3OCurrentBP06BuildsSharedCoreWithoutInventingAction(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)

	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	if core.BusinessAddress != "order-entry.order-handling/BP-06" ||
		core.StopStage != "authorization-action-authority" ||
		core.StopClass != actorContextR3NClassAuthorityUnresolved {
		t.Fatalf("R3O lost current BP-06 stop identity: %+v", core)
	}
	if !reflect.DeepEqual(core.MachineReasons, []string{"authorization-action-authority-unresolved"}) {
		t.Fatalf("R3O lost exact machine reason: %+v", core.MachineReasons)
	}
	if !reflect.DeepEqual(core.ResolutionNeedClasses, []string{"authorization-action-authority"}) {
		t.Fatalf("R3O lost resolution need: %+v", core.ResolutionNeedClasses)
	}
	for _, fact := range core.KnownAuthorityFacts {
		if fact.Value == "execute" {
			t.Fatalf("R3O invented likely action value in common core: %+v", fact)
		}
		if fact.SourceRef == "" {
			t.Fatalf("R3O lost known fact provenance: %+v", fact)
		}
	}
}

func TestActorContextR3OHumanWordingCanVaryWithoutChangingSemanticCore(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}

	concise := actorContextR3ORenderHumanStop(core, "concise")
	plain := actorContextR3ORenderHumanStop(core, "plain")

	if concise.DisplaySummary == plain.DisplaySummary {
		t.Fatalf("test setup failed: renderer variants should differ")
	}
	if !reflect.DeepEqual(concise.Core, plain.Core) {
		t.Fatalf("human wording changed semantic core: concise=%+v plain=%+v", concise.Core, plain.Core)
	}
	for _, summary := range []string{concise.DisplaySummary, plain.DisplaySummary} {
		lower := strings.ToLower(summary)
		if strings.Contains(lower, "execute") ||
			strings.Contains(lower, "security team") ||
			strings.Contains(lower, "order_editor") {
			t.Fatalf("human renderer inferred missing value/owner: %q", summary)
		}
	}
}

func TestActorContextR3OAgentViewKeepsMachineCoreWithoutHumanDirective(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}

	human := actorContextR3ORenderHumanStop(core, "concise")
	agent := actorContextR3ORenderAgentStop(core)

	if !reflect.DeepEqual(human.Core, agent.Core) {
		t.Fatalf("human and agent surfaces diverged from the same semantic core")
	}
	agentType := reflect.TypeOf(agent)
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
		if _, found := agentType.FieldByName(forbidden); found {
			t.Fatalf("R3O agent view introduced presentation/authority shortcut field %q", forbidden)
		}
	}

	coreType := reflect.TypeOf(core)
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
		if _, found := coreType.FieldByName(forbidden); found {
			t.Fatalf("R3O common core introduced presentation/authority shortcut field %q", forbidden)
		}
	}
}

func TestActorContextR3ODeniedObservationStaysRuntimeOutcomeWithoutResolutionNeed(t *testing.T) {
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
			{
				Kind:      "protected-resource",
				Value:     "order_entry.submit_order",
				SourceRef: "reference-pressure-overlay:BP-06",
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
	if core.StopClass != actorContextR3NClassEvaluatedDenied ||
		core.RuntimeOutcome != "denied" {
		t.Fatalf("R3O lost denied runtime outcome: %+v", core)
	}
	if len(core.ResolutionNeedClasses) != 0 || core.ResolutionRequested {
		t.Fatalf("R3O converted denied outcome into a resolution/authority gap: %+v", core)
	}
}

func TestActorContextR3ORejectsR3MAndR3NDisagreement(t *testing.T) {
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-action-authority",
		[]string{"authorization-action-authority-unresolved"},
		nil,
		nil,
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: []string{"authorization-resource-authority-unresolved"},
	})

	if _, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy); err == nil {
		t.Fatal("R3O silently chose between disagreeing R3M/R3N projections")
	}

	taxonomy.BusinessAddress = "order-entry.order-handling/BP-02"
	if _, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy); err == nil {
		t.Fatal("R3O accepted mismatched Business addresses")
	}
}

func TestActorContextR3ORejectsMissingResolutionNeedFromEitherProjection(t *testing.T) {
	explanation, taxonomy := actorContextR3OCurrentBP06ActionStop(t)

	explanation.ResolutionNeedClasses = nil
	if _, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy); err == nil {
		t.Fatal("R3O accepted an R3M projection that dropped the current resolution need")
	}

	explanation, taxonomy = actorContextR3OCurrentBP06ActionStop(t)
	taxonomy.ResolutionNeedClasses = nil
	if _, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy); err == nil {
		t.Fatal("R3O accepted an R3N projection that dropped the current resolution need")
	}
}

func actorContextR3OCurrentBP06ActionStop(
	t *testing.T,
) (actorContextR3MOperationalStopExplanationProbe, actorContextR3NStopTaxonomyProbe) {
	t.Helper()
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if requirement.Status != actorContextR3BAuthorizationRequired ||
		!reflect.DeepEqual(targets, []string{"order_entry.submit_order"}) {
		t.Fatalf("test setup changed: requirement=%+v targets=%v", requirement, targets)
	}

	actionGate := actorContextR3JDeriveActionAuthorityGate(
		requirement.Status,
		targets[0],
		"reference-pressure-overlay:BP-06",
		true,
		true,
		true,
		nil,
	)
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-action-authority",
		actionGate.BlockingReasons,
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-requirement",
				Value:     requirement.Status,
				SourceRef: requirement.RequirementSource,
			},
			{
				Kind:      "protected-resource",
				Value:     targets[0],
				SourceRef: "reference-pressure-overlay:BP-06",
			},
		},
		[]string{
			"reference/business-system-design/order-entry-business-procedure.json",
			"reference/business-system-design/order-entry-protected-target-overlay.json",
		},
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: actionGate.BlockingReasons,
	})
	return explanation, taxonomy
}

func actorContextR3OBuildCommonStopCore(
	explanation actorContextR3MOperationalStopExplanationProbe,
	taxonomy actorContextR3NStopTaxonomyProbe,
) (actorContextR3OStopProjectionCoreProbe, error) {
	if explanation.BusinessAddress == "" ||
		taxonomy.BusinessAddress == "" ||
		explanation.BusinessAddress != taxonomy.BusinessAddress {
		return actorContextR3OStopProjectionCoreProbe{}, fmt.Errorf("R3O Business address mismatch")
	}
	if taxonomy.StopStage == "" || taxonomy.StopClass == "" {
		return actorContextR3OStopProjectionCoreProbe{}, fmt.Errorf("R3O taxonomy is incomplete")
	}
	if explanation.StopStage != "" && explanation.StopStage != taxonomy.StopStage {
		return actorContextR3OStopProjectionCoreProbe{}, fmt.Errorf("R3O stop stage mismatch")
	}
	if len(explanation.BlockingReasons) > 0 &&
		!reflect.DeepEqual(
			actorContextR0SortedUnique(explanation.BlockingReasons),
			actorContextR0SortedUnique(taxonomy.MachineReasons),
		) {
		return actorContextR3OStopProjectionCoreProbe{}, fmt.Errorf("R3O machine reason mismatch")
	}
	if !actorContextR3OSameStringSet(
		explanation.ResolutionNeedClasses,
		taxonomy.ResolutionNeedClasses,
	) {
		return actorContextR3OStopProjectionCoreProbe{}, fmt.Errorf("R3O resolution need mismatch")
	}

	limits := append([]string(nil), explanation.AuthorityLimits...)
	limits = append(limits, taxonomy.AuthorityLimits...)
	limits = append(limits,
		"common-stop-core-is-derived-not-authoritative",
		"human-display-text-is-not-machine-authority",
		"agent-projection-does-not-imply-next-action",
	)

	return actorContextR3OStopProjectionCoreProbe{
		BusinessAddress:        taxonomy.BusinessAddress,
		StopStage:              taxonomy.StopStage,
		StopClass:              taxonomy.StopClass,
		MachineReasons:         actorContextR0SortedUnique(append([]string(nil), taxonomy.MachineReasons...)),
		KnownAuthorityFacts:    append([]actorContextR3MAuthorityFactProbe(nil), explanation.KnownAuthorityFacts...),
		ResolutionNeedClasses:  actorContextR0SortedUnique(append([]string(nil), taxonomy.ResolutionNeedClasses...)),
		RuntimeOutcome:         taxonomy.RuntimeOutcome,
		DownstreamNotAttempted: actorContextR0SortedUnique(append([]string(nil), taxonomy.DownstreamNotAttempted...)),
		SourceRefs:             actorContextR0SortedUnique(append([]string(nil), explanation.SourceRefs...)),
		ResolutionRequested:    explanation.ResolutionRequested,
		HandoffRecipientStatus: explanation.HandoffRecipientStatus,
		AuthorityLimits:        actorContextR0SortedUnique(limits),
	}, nil
}

func actorContextR3OSameStringSet(left []string, right []string) bool {
	l := actorContextR0SortedUnique(append([]string(nil), left...))
	r := actorContextR0SortedUnique(append([]string(nil), right...))
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if l[i] != r[i] {
			return false
		}
	}
	return true
}

func actorContextR3ORenderHumanStop(
	core actorContextR3OStopProjectionCoreProbe,
	variant string,
) actorContextR3OHumanStopViewProbe {
	summary := "The current operation is stopped."
	switch {
	case core.StopClass == actorContextR3NClassAuthorityUnresolved &&
		core.StopStage == "authorization-action-authority" &&
		variant == "concise":
		summary = "Authorization action authority is unresolved."
	case core.StopClass == actorContextR3NClassAuthorityUnresolved &&
		core.StopStage == "authorization-action-authority":
		summary = "The operation cannot continue until the authorization action authority is resolved."
	case core.StopClass == actorContextR3NClassEvaluatedDenied:
		summary = "Authorization was denied for the current evaluation."
	}
	return actorContextR3OHumanStopViewProbe{
		Core:           core,
		DisplaySummary: summary,
	}
}

func actorContextR3ORenderAgentStop(
	core actorContextR3OStopProjectionCoreProbe,
) actorContextR3OAgentStopViewProbe {
	return actorContextR3OAgentStopViewProbe{Core: core}
}
