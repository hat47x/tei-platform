package main

import (
	"reflect"
	"testing"
)

const actorContextR3AAFreshnessReason = "authorization-evaluation-freshness-authority-unresolved"

func TestActorContextR3AAUnresolvedFreshnessAuthorityBuildsDistinctOperationalStop(t *testing.T) {
	authority := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		nil,
	)
	if authority.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		authority.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("test setup changed: %+v", authority)
	}

	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation-freshness-authority",
		[]string{actorContextR3AAFreshnessReason},
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-observation-tuple",
				Value:     actorContextR3XObservationTupleMatch,
				SourceRef: "r3x:context-identity",
			},
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
	if !reflect.DeepEqual(explanation.ResolutionNeedClasses, []string{
		"authorization-evaluation-freshness-authority",
	}) {
		t.Fatalf("R3AA lost freshness resolution need: %+v", explanation)
	}
	if explanation.HandoffRecipientStatus != actorContextR3MHandoffRecipientUnresolved ||
		!explanation.ResolutionRequested {
		t.Fatalf("R3AA invented or dropped handoff state: %+v", explanation)
	}

	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: []string{actorContextR3AAFreshnessReason},
	})
	if taxonomy.StopStage != "authorization-evaluation-freshness-authority" ||
		taxonomy.StopClass != actorContextR3NClassEvaluationFreshnessAuthorityUnresolved ||
		!reflect.DeepEqual(taxonomy.ResolutionNeedClasses, []string{
			"authorization-evaluation-freshness-authority",
		}) {
		t.Fatalf("R3AA freshness stop collapsed into another taxonomy state: %+v", taxonomy)
	}
	if taxonomy.RuntimeOutcome != "" {
		t.Fatalf("R3AA invented current runtime outcome from prior Observation: %+v", taxonomy)
	}

	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	if core.StopClass != actorContextR3NClassEvaluationFreshnessAuthorityUnresolved ||
		!reflect.DeepEqual(core.MachineReasons, []string{actorContextR3AAFreshnessReason}) {
		t.Fatalf("R3AA lost freshness stop in common core: %+v", core)
	}
}

func TestActorContextR3AAHumanAndAgentKeepSameFreshnessReasonWithoutUniversalDetail(t *testing.T) {
	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation-freshness-authority",
		[]string{actorContextR3AAFreshnessReason},
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "reevaluation-authority",
				Value:     actorContextR3YReevaluationUnresolved,
				SourceRef: "r3y:freshness-authority",
			},
		},
		[]string{"plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md"},
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: []string{actorContextR3AAFreshnessReason},
	})
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}

	human, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}

	for name, view := range map[string]actorContextR3PBoundedStopViewProbe{
		"human": human,
		"agent":  agent,
	} {
		if !reflect.DeepEqual(view.Payload.MachineReasons, []string{actorContextR3AAFreshnessReason}) ||
			!reflect.DeepEqual(view.Payload.ResolutionNeedClasses, []string{
				"authorization-evaluation-freshness-authority",
			}) {
			t.Fatalf("%s projection changed shared freshness reason: %+v", name, view)
		}
	}
	if len(human.Payload.KnownAuthorityFacts) != 0 {
		t.Fatalf("Human compact view unexpectedly disclosed Agent detail: %+v", human.Payload.KnownAuthorityFacts)
	}
	if len(agent.Payload.KnownAuthorityFacts) == 0 {
		t.Fatalf("Agent resolution view lost freshness authority detail: %+v", agent.Payload)
	}
}

func TestActorContextR3AAEarlierAuthorityOrTupleMismatchDoesNotBecomeFreshnessStop(t *testing.T) {
	requirement := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: []string{
			actorContextR3AAFreshnessReason,
			"authorization-requirement-unresolved",
		},
	})
	if requirement.StopStage != "authorization-requirement-authority" ||
		requirement.StopClass != actorContextR3NClassAuthorityUnresolved {
		t.Fatalf("R3AA freshness stop displaced earlier requirement authority blocker: %+v", requirement)
	}

	mismatch := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress:  "order-entry.order-handling/BP-06",
		BlockingReasons:  []string{"authorization-evaluation-observation-mismatch"},
		EvaluationStatus: actorContextR3IEvaluationObservationMismatch,
	})
	if mismatch.StopClass != actorContextR3NClassEvaluationObservationMismatch {
		t.Fatalf("R3AA reclassified tuple mismatch as freshness authority: %+v", mismatch)
	}
}

func TestActorContextR3AAFreshnessStopHasNoRetryApprovalOrCacheShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3OStopProjectionCoreProbe{})
	for _, forbidden := range []string{
		"RetryRequired",
		"ShouldReevaluate",
		"AutoReauthorize",
		"HumanApprovalRequired",
		"ApprovalRequired",
		"CacheValid",
		"Reusable",
		"HandoffOwner",
		"NextAction",
		"Authorized",
		"Allowed",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AA common core introduced forbidden freshness/action shortcut %q", forbidden)
		}
	}
}
