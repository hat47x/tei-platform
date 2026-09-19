package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const actorContextR3AJRequestClassAuthorityInformation = "authority-information"

type actorContextR3AJAuthorityInformationRequestProbe struct {
	RequestClass           string
	BusinessAddress        string
	StopStage              string
	StopClass              string
	MachineReasons         []string
	InformationNeedClasses []string
	EvidenceNeedClasses    []string
	KnownSourceRefs        []string
	DrillDownRefs          []string
	ResolutionRequested    bool
	HandoffRecipientStatus string
	AuthorityLimits        []string
}

func TestActorContextR3AJBuildsScopedAuthorityInformationRequestWithoutOwner(t *testing.T) {
	reasoning := actorContextR3AJFreshnessReasoning(t)
	stopView := actorContextR3AJFreshnessStopView(t)

	got, err := actorContextR3AJBuildInformationRequest(reasoning, stopView)
	if err != nil {
		t.Fatal(err)
	}

	if got.RequestClass != actorContextR3AJRequestClassAuthorityInformation ||
		got.BusinessAddress != "order-entry.order-handling/BP-06" ||
		got.StopStage != "authorization-evaluation-freshness-authority" {
		t.Fatalf("R3AJ lost request scope: %+v", got)
	}
	if !reflect.DeepEqual(got.InformationNeedClasses, []string{
		"authorization-evaluation-freshness-authority",
	}) {
		t.Fatalf("R3AJ lost information need class: %+v", got.InformationNeedClasses)
	}
	if !reflect.DeepEqual(got.EvidenceNeedClasses, got.InformationNeedClasses) {
		t.Fatalf("R3AJ evidence need diverged from unresolved authority class: %+v", got)
	}
	if !got.ResolutionRequested ||
		got.HandoffRecipientStatus != actorContextR3MHandoffRecipientUnresolved {
		t.Fatalf("R3AJ invented or dropped resolution recipient state: %+v", got)
	}
	if len(got.KnownSourceRefs) == 0 || len(got.DrillDownRefs) == 0 {
		t.Fatalf("R3AJ lost known evidence/drill-down context: %+v", got)
	}
}

func TestActorContextR3AJCrossChecksReasoningAndStopResolutionNeeds(t *testing.T) {
	reasoning := actorContextR3AJFreshnessReasoning(t)
	stopView := actorContextR3AJFreshnessStopView(t)

	reasoning.ResolutionNeedClasses = []string{"authorization-action-authority"}
	if _, err := actorContextR3AJBuildInformationRequest(reasoning, stopView); err == nil {
		t.Fatal("R3AJ silently selected one side of mismatched resolution need classes")
	}

	reasoning = actorContextR3AJFreshnessReasoning(t)
	stopView.Payload.ResolutionNeedClasses = []string{"authorization-action-authority"}
	if _, err := actorContextR3AJBuildInformationRequest(reasoning, stopView); err == nil {
		t.Fatal("R3AJ accepted stop projection with different resolution need classes")
	}
}

func TestActorContextR3AJPreservesUnresolvedRecipientWithoutInferringIdentity(t *testing.T) {
	reasoning := actorContextR3AJFreshnessReasoning(t)
	stopView := actorContextR3AJFreshnessStopView(t)

	got, err := actorContextR3AJBuildInformationRequest(reasoning, stopView)
	if err != nil {
		t.Fatal(err)
	}
	if got.HandoffRecipientStatus != actorContextR3MHandoffRecipientUnresolved {
		t.Fatalf("R3AJ changed unresolved recipient status: %+v", got)
	}
	for _, value := range append(
		append([]string(nil), got.InformationNeedClasses...),
		got.EvidenceNeedClasses...,
	) {
		for _, forbidden := range []string{
			"security team",
			"application owner",
			"business owner",
			"human operator",
			"administrator",
			"sales operator",
			"ORDER_EDITOR",
			"execute",
		} {
			if value == forbidden {
				t.Fatalf("R3AJ inferred recipient/candidate authority value %q", value)
			}
		}
	}
}

func TestActorContextR3AJDeniedObservationDoesNotBecomeInformationRequest(t *testing.T) {
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation",
		nil,
		[]actorContextR3MAuthorityFactProbe{{
			Kind:      "authorization-requirement",
			Value:     actorContextR3BAuthorizationRequired,
			SourceRef: "business-procedure-security-boundary",
		}},
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
	stopView, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	reasoning := actorContextR3AIReadOnlyReasoningProbe{
		Audience: actorContextR3PAudienceAgent,
		Purpose:  actorContextR3PPurposeResolutionInspection,
	}

	if _, err := actorContextR3AJBuildInformationRequest(reasoning, stopView); err == nil {
		t.Fatal("R3AJ converted exact denied runtime outcome into missing-authority information request")
	}
}

func TestActorContextR3AJMissingRecipientScopeOrTraceFailsClosed(t *testing.T) {
	cases := map[string]func(*actorContextR3PBoundedStopViewProbe){
		"recipient-status": func(view *actorContextR3PBoundedStopViewProbe) {
			view.Payload.HandoffRecipientStatus = ""
		},
		"business-address": func(view *actorContextR3PBoundedStopViewProbe) {
			view.Payload.BusinessAddress = ""
		},
		"source-trace": func(view *actorContextR3PBoundedStopViewProbe) {
			view.Payload.SourceRefs = nil
			view.DrillDownRefs = nil
		},
		"resolution-request": func(view *actorContextR3PBoundedStopViewProbe) {
			view.Payload.ResolutionRequested = false
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			reasoning := actorContextR3AJFreshnessReasoning(t)
			stopView := actorContextR3AJFreshnessStopView(t)
			mutate(&stopView)
			if _, err := actorContextR3AJBuildInformationRequest(reasoning, stopView); err == nil {
				t.Fatalf("R3AJ accepted incomplete request scope %q", name)
			}
		})
	}
}

func TestActorContextR3AJRepeatedBuildAndInputOrderAreStable(t *testing.T) {
	reasoning := actorContextR3AJFreshnessReasoning(t)
	stopView := actorContextR3AJFreshnessStopView(t)

	first, err := actorContextR3AJBuildInformationRequest(reasoning, stopView)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AJBuildInformationRequest(reasoning, stopView)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AJ repeated build changed normalized request identity: first=%+v second=%+v", first, second)
	}

	reversedReasoning := reasoning
	reversedReasoning.ResolutionNeedClasses = append(
		[]string(nil), reasoning.ResolutionNeedClasses...,
	)
	for i, j := 0, len(reversedReasoning.ResolutionNeedClasses)-1; i < j; i, j = i+1, j-1 {
		reversedReasoning.ResolutionNeedClasses[i], reversedReasoning.ResolutionNeedClasses[j] =
			reversedReasoning.ResolutionNeedClasses[j], reversedReasoning.ResolutionNeedClasses[i]
	}
	reversedView := stopView
	reversedView.Payload.ResolutionNeedClasses = append(
		[]string(nil), stopView.Payload.ResolutionNeedClasses...,
	)
	reversedView.Payload.SourceRefs = append([]string(nil), stopView.Payload.SourceRefs...)
	reversedView.DrillDownRefs = append([]string(nil), stopView.DrillDownRefs...)
	for i, j := 0, len(reversedView.Payload.SourceRefs)-1; i < j; i, j = i+1, j-1 {
		reversedView.Payload.SourceRefs[i], reversedView.Payload.SourceRefs[j] =
			reversedView.Payload.SourceRefs[j], reversedView.Payload.SourceRefs[i]
	}
	for i, j := 0, len(reversedView.DrillDownRefs)-1; i < j; i, j = i+1, j-1 {
		reversedView.DrillDownRefs[i], reversedView.DrillDownRefs[j] =
			reversedView.DrillDownRefs[j], reversedView.DrillDownRefs[i]
	}

	normalized, err := actorContextR3AJBuildInformationRequest(reversedReasoning, reversedView)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, normalized) {
		t.Fatalf("R3AJ input order changed normalized request identity: first=%+v normalized=%+v",
			first, normalized)
	}
}

func TestActorContextR3AJRequestShapeHasNoOwnerCandidateActionOrExecutionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3AJAuthorityInformationRequestProbe{})
	for _, forbidden := range []string{
		"Recipient",
		"RecipientID",
		"HandoffOwner",
		"SuggestedOwner",
		"CandidateValue",
		"ProposedAuthorityValue",
		"ToolCall",
		"InvocationPlan",
		"Command",
		"SuggestedAction",
		"NextAction",
		"Remediation",
		"Retry",
		"Reauthorize",
		"Rebuild",
		"ApprovalRequest",
		"CanExecute",
		"Authorized",
		"Allowed",
		"Continue",
		"Confidence",
		"Priority",
		"Severity",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AJ information request introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func actorContextR3AJFreshnessReasoning(t *testing.T) actorContextR3AIReadOnlyReasoningProbe {
	t.Helper()

	materialized, explanation, _, _ := actorContextR3AIRequirementFixture(
		t,
		actorContextR3AFFreshnessAuthority(),
		false,
	)
	return actorContextR3AIConsumeReadOnly(materialized, explanation, nil, nil)
}

func actorContextR3AJFreshnessStopView(t *testing.T) actorContextR3PBoundedStopViewProbe {
	t.Helper()

	authority := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		nil,
	)
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation-freshness-authority",
		[]string{actorContextR3AAFreshnessReason},
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "reevaluation-authority",
				Value:     authority.ReevaluationAuthorityStatus,
				SourceRef: "r3y:freshness-authority",
			},
			{
				Kind:      "prior-observation-reuse-authority",
				Value:     authority.ReuseAuthorityStatus,
				SourceRef: "r3y:freshness-authority",
			},
		},
		[]string{
			"plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md",
			"plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md",
			"plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md",
		},
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: []string{actorContextR3AAFreshnessReason},
	})
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func actorContextR3AJBuildInformationRequest(
	reasoning actorContextR3AIReadOnlyReasoningProbe,
	stopView actorContextR3PBoundedStopViewProbe,
) (actorContextR3AJAuthorityInformationRequestProbe, error) {
	if reasoning.Audience != actorContextR3PAudienceAgent ||
		reasoning.Purpose != actorContextR3PPurposeResolutionInspection ||
		stopView.Audience != actorContextR3PAudienceAgent ||
		stopView.Purpose != actorContextR3PPurposeResolutionInspection {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ requires Agent resolution-inspection inputs",
		)
	}
	if !stopView.Payload.ResolutionRequested {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ stop projection does not request external resolution information",
		)
	}
	if stopView.Payload.BusinessAddress == "" ||
		stopView.Payload.StopStage == "" ||
		stopView.Payload.StopClass == "" {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ request scope is incomplete",
		)
	}
	if stopView.Payload.HandoffRecipientStatus == "" {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ handoff recipient status is missing",
		)
	}
	reasoningNeeds := actorContextR0SortedUnique(append(
		[]string(nil), reasoning.ResolutionNeedClasses...,
	))
	stopNeeds := actorContextR0SortedUnique(append(
		[]string(nil), stopView.Payload.ResolutionNeedClasses...,
	))
	if len(reasoningNeeds) == 0 || !reflect.DeepEqual(reasoningNeeds, stopNeeds) {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ resolution need classes disagree",
		)
	}
	if reasoning.FreshnessStopReason != "" &&
		!actorContextR0Contains(stopView.Payload.MachineReasons, reasoning.FreshnessStopReason) {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ stop reason is not present in the authority-preserving stop projection",
		)
	}
	knownRefs := actorContextR0SortedUnique(append(
		[]string(nil), stopView.Payload.SourceRefs...,
	))
	drillRefs := actorContextR0SortedUnique(append(
		[]string(nil), stopView.DrillDownRefs...,
	))
	if len(knownRefs) == 0 || len(drillRefs) == 0 {
		return actorContextR3AJAuthorityInformationRequestProbe{}, fmt.Errorf(
			"R3AJ request has no known source/drill-down context",
		)
	}

	out := actorContextR3AJAuthorityInformationRequestProbe{
		RequestClass:           actorContextR3AJRequestClassAuthorityInformation,
		BusinessAddress:        stopView.Payload.BusinessAddress,
		StopStage:              stopView.Payload.StopStage,
		StopClass:              stopView.Payload.StopClass,
		MachineReasons:         actorContextR0SortedUnique(append([]string(nil), stopView.Payload.MachineReasons...)),
		InformationNeedClasses: reasoningNeeds,
		EvidenceNeedClasses:    append([]string(nil), reasoningNeeds...),
		KnownSourceRefs:        knownRefs,
		DrillDownRefs:          drillRefs,
		ResolutionRequested:    true,
		HandoffRecipientStatus: stopView.Payload.HandoffRecipientStatus,
		AuthorityLimits: []string{
			"information-request-does-not-select-remediation",
			"information-need-class-does-not-supply-missing-authority-value",
			"evidence-need-class-does-not-supply-candidate-value",
			"recipient-status-does-not-select-recipient-identity",
			"known-source-ref-does-not-resolve-authority",
			"request-scope-does-not-select-routing-destination",
			"information-request-does-not-authorize-tool-invocation-or-command",
		},
	}
	sort.Strings(out.AuthorityLimits)
	return out, nil
}
