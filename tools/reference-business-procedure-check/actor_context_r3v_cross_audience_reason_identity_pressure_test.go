package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR3VReasonWitnessProbe struct {
	Identity         string
	Category         string
	ChangeClass      string
	DependencyKey    string
	DependencyReason string
	Audiences        []string
}

type actorContextR3VCrossAudienceComparisonProbe struct {
	SharedReasons  []actorContextR3VReasonWitnessProbe
	AudienceOnly   map[string][]actorContextR3TReevaluationReasonProbe
	AuthorityLimits []string
}

func TestActorContextR3VSharedMachineReasonIdentityMatchesAcrossHumanAndAgent(t *testing.T) {
	core := actorContextR3QCurrentBP06Core(t)
	deps := actorContextR3SCurrentBP06ReferenceEnvelope(t, repoPath())
	business := deps.ReferenceArtifacts[0]
	key := actorContextR3SReferenceKey(business)

	humanView, err := actorContextR3PSelectBoundedStopView(
		core,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	humanEdges := []actorContextR3SDependencyReasonEdgeProbe{
		{
			Category:      actorContextR3PCategoryMachineReasons,
			DependencyKey: key,
			ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
		},
	}
	humanReasons, err := actorContextR3SBuildReasonEnvelope(core, humanView, deps, humanEdges)
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
	agentEdges := []actorContextR3SDependencyReasonEdgeProbe{
		humanEdges[0],
		{
			Category:      actorContextR3PCategoryKnownAuthorityFacts,
			DependencyKey: key,
			ReasonCode:    actorContextR3SReasonAuthorizationRequirementSource,
		},
	}
	agentReasons, err := actorContextR3SBuildReasonEnvelope(core, agentView, deps, agentEdges)
	if err != nil {
		t.Fatal(err)
	}

	after := actorContextR3TCloneEnvelope(deps)
	after.ReferenceArtifacts[0].ContentFingerprint = "sha256:test-only-changed-business-procedure"
	changes := actorContextR3TChangesFromEnvelopes(deps, after)

	humanImpact := actorContextR3TExplainChanges(humanReasons, changes)
	agentImpact := actorContextR3TExplainChanges(agentReasons, changes)
	got := actorContextR3VCompareAudiences(map[string]actorContextR3TImpactExplanationProbe{
		actorContextR3PAudienceHuman: humanImpact,
		actorContextR3PAudienceAgent: agentImpact,
	})

	if len(got.SharedReasons) != 1 {
		t.Fatalf("R3V did not isolate the one shared reason: %+v", got)
	}
	shared := got.SharedReasons[0]
	if shared.Category != actorContextR3PCategoryMachineReasons ||
		shared.DependencyKey != key ||
		shared.DependencyReason != actorContextR3SReasonAuthorizationRequirementSource {
		t.Fatalf("R3V shared reason lost identity/provenance: %+v", shared)
	}
	if !reflect.DeepEqual(shared.Audiences, []string{
		actorContextR3PAudienceAgent,
		actorContextR3PAudienceHuman,
	}) {
		t.Fatalf("R3V shared reason lost audience witnesses: %+v", shared.Audiences)
	}
}

func TestActorContextR3VAgentOnlyReasonDoesNotForceHumanDisclosure(t *testing.T) {
	shared := actorContextR3TReevaluationReasonProbe{
		Category:         actorContextR3PCategoryMachineReasons,
		ChangeClass:      actorContextR3TChangeReferenceInput,
		DependencyKey:    "reference:business",
		DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
	}
	agentOnly := actorContextR3TReevaluationReasonProbe{
		Category:         actorContextR3PCategoryKnownAuthorityFacts,
		ChangeClass:      actorContextR3TChangeReferenceInput,
		DependencyKey:    "reference:business",
		DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
	}

	got := actorContextR3VCompareAudiences(map[string]actorContextR3TImpactExplanationProbe{
		actorContextR3PAudienceHuman: {Reasons: []actorContextR3TReevaluationReasonProbe{shared}},
		actorContextR3PAudienceAgent: {Reasons: []actorContextR3TReevaluationReasonProbe{shared, agentOnly}},
	})

	if len(got.SharedReasons) != 1 {
		t.Fatalf("R3V changed shared reason count: %+v", got)
	}
	if !reflect.DeepEqual(
		got.AudienceOnly[actorContextR3PAudienceAgent],
		[]actorContextR3TReevaluationReasonProbe{agentOnly},
	) {
		t.Fatalf("R3V lost Agent-only reason: %+v", got.AudienceOnly)
	}
	if len(got.AudienceOnly[actorContextR3PAudienceHuman]) != 0 {
		t.Fatalf("R3V forced Agent detail into Human disclosure: %+v", got.AudienceOnly)
	}
}

func TestActorContextR3VReasonIdentityExcludesAudienceAndPurpose(t *testing.T) {
	reason := actorContextR3TReevaluationReasonProbe{
		Category:         actorContextR3PCategorySourceRefs,
		ChangeClass:      actorContextR3TChangeLocatorRefresh,
		DependencyKey:    "semantic:application-pattern:order_entry.submit_order",
		DependencyReason: actorContextR3SReasonSourceIndexDefinitionLocator,
	}

	human := actorContextR3UAggregateImpacts(
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{reason}}},
	)
	agent := actorContextR3UAggregateImpacts(
		actorContextR3PAudienceAgent,
		actorContextR3PPurposeResolutionInspection,
		[]actorContextR3TImpactExplanationProbe{{Reasons: []actorContextR3TReevaluationReasonProbe{reason}}},
	)

	if human.Audience == agent.Audience || human.Purpose == agent.Purpose {
		t.Fatal("test setup requires different bounded projection identities")
	}
	if actorContextR3VReasonIdentity(human.Reasons[0]) != actorContextR3VReasonIdentity(agent.Reasons[0]) {
		t.Fatal("R3V included audience/purpose in reason identity")
	}
}

func TestActorContextR3VDifferentDependencyKeyIsDifferentReasonIdentity(t *testing.T) {
	left := actorContextR3TReevaluationReasonProbe{
		Category:         actorContextR3PCategoryMachineReasons,
		ChangeClass:      actorContextR3TChangeReferenceInput,
		DependencyKey:    "reference:business-a",
		DependencyReason: actorContextR3SReasonAuthorizationRequirementSource,
	}
	right := left
	right.DependencyKey = "reference:business-b"

	if actorContextR3VReasonIdentity(left) == actorContextR3VReasonIdentity(right) {
		t.Fatal("R3V collapsed distinct dependency provenance into one reason identity")
	}
}

func TestActorContextR3VComparisonHasNoUniversalOrAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3VCrossAudienceComparisonProbe{})
	for _, forbidden := range []string{
		"UniversalPacket",
		"PreferredAudience",
		"Winner",
		"StrongerAuthority",
		"PrimaryReason",
		"NextAction",
		"Approved",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3V comparison introduced forbidden universal/authority shortcut %q", forbidden)
		}
	}
}

func actorContextR3VCompareAudiences(
	impacts map[string]actorContextR3TImpactExplanationProbe,
) actorContextR3VCrossAudienceComparisonProbe {
	type witnessed struct {
		reason    actorContextR3TReevaluationReasonProbe
		audiences map[string]bool
	}
	byIdentity := map[string]*witnessed{}
	audienceCount := len(impacts)

	for audience, impact := range impacts {
		for _, reason := range actorContextR3TUniqueReasons(impact.Reasons) {
			identity := actorContextR3VReasonIdentity(reason)
			item, ok := byIdentity[identity]
			if !ok {
				item = &witnessed{
					reason:    reason,
					audiences: map[string]bool{},
				}
				byIdentity[identity] = item
			}
			item.audiences[audience] = true
		}
	}

	result := actorContextR3VCrossAudienceComparisonProbe{
		AudienceOnly: map[string][]actorContextR3TReevaluationReasonProbe{},
		AuthorityLimits: []string{
			"shared-reason-does-not-create-stronger-authority",
			"audience-agreement-does-not-create-universal-packet",
			"presentation-does-not-enter-reason-identity",
			"audience-only-reason-does-not-force-disclosure",
		},
	}

	for _, item := range byIdentity {
		audiences := make([]string, 0, len(item.audiences))
		for audience := range item.audiences {
			audiences = append(audiences, audience)
		}
		sort.Strings(audiences)

		if len(audiences) == audienceCount && audienceCount > 1 {
			result.SharedReasons = append(result.SharedReasons, actorContextR3VReasonWitnessProbe{
				Identity:         actorContextR3VReasonIdentity(item.reason),
				Category:         item.reason.Category,
				ChangeClass:      item.reason.ChangeClass,
				DependencyKey:    item.reason.DependencyKey,
				DependencyReason: item.reason.DependencyReason,
				Audiences:        audiences,
			})
			continue
		}
		for _, audience := range audiences {
			result.AudienceOnly[audience] = append(result.AudienceOnly[audience], item.reason)
		}
	}

	sort.Slice(result.SharedReasons, func(i, j int) bool {
		return result.SharedReasons[i].Identity < result.SharedReasons[j].Identity
	})
	for audience, reasons := range result.AudienceOnly {
		result.AudienceOnly[audience] = actorContextR3TUniqueReasons(reasons)
	}
	return result
}

func actorContextR3VReasonIdentity(reason actorContextR3TReevaluationReasonProbe) string {
	return reason.Category + "|" +
		reason.ChangeClass + "|" +
		reason.DependencyKey + "|" +
		reason.DependencyReason
}
