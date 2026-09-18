package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ABRoleContextIdentity      = "authorization-context-identity-boundary"
	actorContextR3ABRoleFreshnessAuthority   = "evaluation-freshness-authority-boundary"
	actorContextR3ABRolePrecedenceEvidence   = "freshness-precedence-evidence-gate"
)

type actorContextR3ABDrillDownRootProbe struct {
	StopReason          string
	ResolutionNeedClass string
	SourceRef           string
	RootRole            string
}

type actorContextR3ABDrillDownProbe struct {
	Audience        string
	Purpose         string
	Roots           []actorContextR3ABDrillDownRootProbe
	AuthorityLimits []string
}

func TestActorContextR3ABFreshnessStopRootsPreserveDistinctRoles(t *testing.T) {
	core := actorContextR3ABFreshnessCore(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	got, err := actorContextR3ABBuildDrillDown(view)
	if err != nil {
		t.Fatal(err)
	}

	wantRoles := []string{
		actorContextR3ABRoleContextIdentity,
		actorContextR3ABRoleFreshnessAuthority,
		actorContextR3ABRolePrecedenceEvidence,
	}
	gotRoles := []string{}
	for _, root := range got.Roots {
		gotRoles = append(gotRoles, root.RootRole)
		if root.StopReason != actorContextR3AAFreshnessReason ||
			root.ResolutionNeedClass != "authorization-evaluation-freshness-authority" {
			t.Fatalf("R3AB root lost stop identity: %+v", root)
		}
	}
	sort.Strings(gotRoles)
	sort.Strings(wantRoles)
	if !reflect.DeepEqual(gotRoles, wantRoles) {
		t.Fatalf("R3AB flattened or lost provenance roles: got=%v want=%v", gotRoles, wantRoles)
	}
}

func TestActorContextR3ABHumanAndAgentReachSameRoots(t *testing.T) {
	core := actorContextR3ABFreshnessCore(t)
	humanView, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	agentView, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}

	human, err := actorContextR3ABBuildDrillDown(humanView)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR3ABBuildDrillDown(agentView)
	if err != nil {
		t.Fatal(err)
	}

	if human.Audience == agent.Audience || human.Purpose == agent.Purpose {
		t.Fatal("test setup requires different bounded projections")
	}
	if !reflect.DeepEqual(human.Roots, agent.Roots) {
		t.Fatalf("Human / Agent freshness stop reached different provenance roots: human=%+v agent=%+v", human.Roots, agent.Roots)
	}
}

func TestActorContextR3ABHumanWordingDoesNotChangeDrillDownIdentity(t *testing.T) {
	core := actorContextR3ABFreshnessCore(t)
	concise := actorContextR3ORenderHumanStop(core, "concise")
	plain := actorContextR3ORenderHumanStop(core, "plain")
	if concise.DisplaySummary == plain.DisplaySummary {
		t.Fatal("test setup requires different Human wording")
	}

	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := actorContextR3ABBuildDrillDown(view)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3ABBuildDrillDown(view)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Roots, second.Roots) {
		t.Fatalf("Human presentation wording changed provenance roots: first=%+v second=%+v", first.Roots, second.Roots)
	}
}

func TestActorContextR3ABUnknownSourceRoleFailsClosed(t *testing.T) {
	core := actorContextR3ABFreshnessCore(t)
	core.SourceRefs = append(core.SourceRefs, "test-only:unknown-freshness-source")
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actorContextR3ABBuildDrillDown(view); err == nil {
		t.Fatal("R3AB guessed a provenance role for unknown source ref")
	}
}

func TestActorContextR3ABDrillDownHasNoAuthorityDecisionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ABDrillDownProbe{})
	for _, forbidden := range []string{
		"Truth",
		"EffectiveDecision",
		"ReevaluationRequired",
		"ReusePermitted",
		"PrecedenceWinner",
		"HandoffOwner",
		"NextAction",
		"Authorized",
		"Allowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AB drill-down introduced forbidden authority/action shortcut %q", forbidden)
		}
	}
}

func actorContextR3ABFreshnessCore(t *testing.T) actorContextR3OStopProjectionCoreProbe {
	t.Helper()
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
				Value:     actorContextR3YReevaluationUnresolved,
				SourceRef: "r3y:freshness-authority",
			},
			{
				Kind:      "prior-observation-reuse-authority",
				Value:     actorContextR3YReuseUnresolved,
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
	return core
}

func actorContextR3ABBuildDrillDown(
	view actorContextR3PBoundedStopViewProbe,
) (actorContextR3ABDrillDownProbe, error) {
	if !actorContextR0Contains(view.SelectedCategories, actorContextR3PCategorySourceRefs) {
		return actorContextR3ABDrillDownProbe{}, fmt.Errorf("R3AB source-refs category is not selected")
	}
	if !reflect.DeepEqual(view.Payload.MachineReasons, []string{actorContextR3AAFreshnessReason}) ||
		!reflect.DeepEqual(view.Payload.ResolutionNeedClasses, []string{
			"authorization-evaluation-freshness-authority",
		}) {
		return actorContextR3ABDrillDownProbe{}, fmt.Errorf("R3AB unexpected freshness stop identity")
	}

	roleByRef := map[string]string{
		"plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md":
			actorContextR3ABRoleContextIdentity,
		"plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md":
			actorContextR3ABRoleFreshnessAuthority,
		"plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md":
			actorContextR3ABRolePrecedenceEvidence,
	}

	result := actorContextR3ABDrillDownProbe{
		Audience: view.Audience,
		Purpose:  view.Purpose,
		AuthorityLimits: []string{
			"context-identity-boundary-is-not-freshness-directive-authority",
			"freshness-authority-boundary-plan-is-not-current-production-directive",
			"precedence-evidence-gate-is-not-authorization-decision-authority",
			"same-provenance-roots-do-not-imply-same-disclosure",
			"drill-down-does-not-select-handoff-owner-or-next-action",
		},
	}
	for _, ref := range view.Payload.SourceRefs {
		role, ok := roleByRef[ref]
		if !ok {
			return actorContextR3ABDrillDownProbe{}, fmt.Errorf("R3AB unknown source role for %q", ref)
		}
		result.Roots = append(result.Roots, actorContextR3ABDrillDownRootProbe{
			StopReason:          actorContextR3AAFreshnessReason,
			ResolutionNeedClass: "authorization-evaluation-freshness-authority",
			SourceRef:           ref,
			RootRole:            role,
		})
	}
	sort.Slice(result.Roots, func(i, j int) bool {
		if result.Roots[i].RootRole != result.Roots[j].RootRole {
			return result.Roots[i].RootRole < result.Roots[j].RootRole
		}
		return result.Roots[i].SourceRef < result.Roots[j].SourceRef
	})
	return result, nil
}
