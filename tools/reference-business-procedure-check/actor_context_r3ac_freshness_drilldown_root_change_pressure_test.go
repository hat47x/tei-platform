package main

import (
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ACChangeReferenceEvidence  = "reference-evidence-content-changed"
	actorContextR3ACChangeRuntimeObservation = "runtime-observation-content-changed"
	actorContextR3ACChangeTestOnlyContext    = "test-only-context-changed"
	actorContextR3ACChangeCorroboration      = "corroboration-changed"
	actorContextR3ACChangeLocatorRefresh     = "locator-refresh-required"
	actorContextR3ACChangeRootSet            = "provenance-root-set-changed"
)

type actorContextR3ACRootChangeProbe struct {
	RootIdentity string
	RootRole     string
	ChangeClass  string
}

type actorContextR3ACAssessmentProbe struct {
	Changes         []actorContextR3ACRootChangeProbe
	AuthorityLimits []string
}

func TestActorContextR3ACClassifiesContentChangeByRootRole(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	cases := []struct {
		name string
		role string
		want string
	}{
		{"reference", actorContextR3ABRoleReferenceEvidence, actorContextR3ACChangeReferenceEvidence},
		{"runtime", actorContextR3ABRoleRuntimeObservation, actorContextR3ACChangeRuntimeObservation},
		{"test-only-context", actorContextR3ABRoleTestOnlyContext, actorContextR3ACChangeTestOnlyContext},
		{"corroboration", actorContextR3ABRoleCorroboration, actorContextR3ACChangeCorroboration},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			after := actorContextR3ACCloneRoots(base.Roots)
			idx := actorContextR3ACRootIndexByRole(after, tc.role)
			if idx < 0 {
				t.Fatalf("test setup has no root role %q", tc.role)
			}
			after[idx].Fingerprint = after[idx].Fingerprint + ":changed"

			got := actorContextR3ACClassifyRootChanges(base.Roots, after)
			if !actorContextR3ACHasChange(got.Changes, actorContextR3ACRootAddress(base.Roots[idx]), tc.want) {
				t.Fatalf("%s content change was misclassified: %+v", tc.role, got)
			}
			for _, change := range got.Changes {
				if change.ChangeClass == actorContextR3ACChangeLocatorRefresh {
					t.Fatalf("content-only %s change became locator refresh: %+v", tc.role, got)
				}
			}
		})
	}
}

func TestActorContextR3ACRuntimeObservationChangeDoesNotBecomeReferenceChange(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleRuntimeObservation)
	beforeAddress := actorContextR3ACRootAddress(base.Roots[idx])
	after[idx].Fingerprint += ":new-observation"
	after[idx].ClaimKind = "prior-authorization-observation:denied"

	if actorContextR3ACRootAddress(after[idx]) != beforeAddress {
		t.Fatalf("runtime outcome revision changed stable root address: before=%q after=%q", beforeAddress, actorContextR3ACRootAddress(after[idx]))
	}

	got := actorContextR3ACClassifyRootChanges(base.Roots, after)
	if !actorContextR3ACHasChange(got.Changes, beforeAddress, actorContextR3ACChangeRuntimeObservation) {
		t.Fatalf("runtime Observation change lost runtime plane: %+v", got)
	}
	if actorContextR3ACHasChange(got.Changes, beforeAddress, actorContextR3ACChangeRootSet) {
		t.Fatalf("same-tuple runtime outcome revision became root-set change: %+v", got)
	}
	for _, change := range got.Changes {
		if change.ChangeClass == actorContextR3ACChangeReferenceEvidence {
			t.Fatalf("runtime Observation change leaked into Reference Evidence plane: %+v", got)
		}
	}
}

func TestActorContextR3ACLocatorOnlyMoveDoesNotBecomeContentChange(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleReferenceEvidence)
	after[idx].Locator = after[idx].Locator + ".moved"

	got := actorContextR3ACClassifyRootChanges(base.Roots, after)
	identity := actorContextR3ACRootAddress(base.Roots[idx])
	if !actorContextR3ACHasChange(got.Changes, identity, actorContextR3ACChangeLocatorRefresh) {
		t.Fatalf("locator move did not produce locator refresh reason: %+v", got)
	}
	for _, change := range got.Changes {
		if change.RootIdentity == identity && change.ChangeClass != actorContextR3ACChangeLocatorRefresh {
			t.Fatalf("locator-only move became content/authority change: %+v", got)
		}
	}
}

func TestActorContextR3ACContentAndLocatorChangePreserveBothReasons(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleCorroboration)
	after[idx].Fingerprint += ":changed"
	after[idx].Locator += ".moved"

	got := actorContextR3ACClassifyRootChanges(base.Roots, after)
	identity := actorContextR3ACRootAddress(base.Roots[idx])
	if !actorContextR3ACHasChange(got.Changes, identity, actorContextR3ACChangeCorroboration) ||
		!actorContextR3ACHasChange(got.Changes, identity, actorContextR3ACChangeLocatorRefresh) {
		t.Fatalf("combined content/locator delta lost an independent reason: %+v", got)
	}
}

func TestActorContextR3ACRootSetChangeDoesNotInferAuthorityMigration(t *testing.T) {
	base := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	after := actorContextR3ACCloneRoots(base.Roots)
	removed := after[len(after)-1]
	after = after[:len(after)-1]

	got := actorContextR3ACClassifyRootChanges(base.Roots, after)
	if !actorContextR3ACHasChange(got.Changes, actorContextR3ACRootAddress(removed), actorContextR3ACChangeRootSet) {
		t.Fatalf("removed provenance root was not preserved as root-set change: %+v", got)
	}
}

func TestActorContextR3ACHumanAndAgentClassifySharedRootDeltaIdentically(t *testing.T) {
	human := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	agent := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)
	if !reflect.DeepEqual(human.Roots, agent.Roots) {
		t.Fatal("test setup requires identical shared provenance roots")
	}

	after := actorContextR3ACCloneRoots(human.Roots)
	idx := actorContextR3ACRootIndexByRole(after, actorContextR3ABRoleRuntimeObservation)
	after[idx].Fingerprint += ":changed"

	humanAssessment := actorContextR3ACClassifyRootChanges(human.Roots, after)
	agentAssessment := actorContextR3ACClassifyRootChanges(agent.Roots, after)
	if !reflect.DeepEqual(humanAssessment, agentAssessment) {
		t.Fatalf("audience-specific inline disclosure changed root delta classification: human=%+v agent=%+v",
			humanAssessment, agentAssessment)
	}
}

func TestActorContextR3ACAssessmentHasNoAutomaticFreshnessActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ACAssessmentProbe{})
	for _, forbidden := range []string{
		"Stale",
		"Fresh",
		"ShouldReevaluate",
		"ReevaluationRequired",
		"ReusePermitted",
		"RetryRequired",
		"CacheValid",
		"AutoRefresh",
		"AutoReauthorize",
		"HandoffOwner",
		"NextAction",
		"Authorized",
		"Allowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AC root-change assessment introduced forbidden automatic action %q", forbidden)
		}
	}
}

func actorContextR3ACClassifyRootChanges(
	before []actorContextR3ABProvenanceRootProbe,
	after []actorContextR3ABProvenanceRootProbe,
) actorContextR3ACAssessmentProbe {
	result := actorContextR3ACAssessmentProbe{
		AuthorityLimits: []string{
			"reference-change-does-not-author-freshness-directive",
			"runtime-observation-change-does-not-equal-reference-change",
			"test-only-context-change-does-not-equal-production-authority-change",
			"corroboration-change-does-not-equal-authority-change",
			"locator-refresh-does-not-equal-content-change",
			"root-set-change-does-not-imply-authority-migration",
			"root-change-classification-does-not-select-response-action",
		},
	}

	beforeByID := actorContextR3ACRootsByIdentity(before)
	afterByID := actorContextR3ACRootsByIdentity(after)
	all := map[string]bool{}
	for identity := range beforeByID {
		all[identity] = true
	}
	for identity := range afterByID {
		all[identity] = true
	}

	for identity := range all {
		left, leftOK := beforeByID[identity]
		right, rightOK := afterByID[identity]
		if !leftOK || !rightOK {
			role := left.Role
			if !leftOK {
				role = right.Role
			}
			result.Changes = append(result.Changes, actorContextR3ACRootChangeProbe{
				RootIdentity: identity,
				RootRole:     role,
				ChangeClass:  actorContextR3ACChangeRootSet,
			})
			continue
		}

		if left.Fingerprint != right.Fingerprint || left.ClaimKind != right.ClaimKind {
			if changeClass := actorContextR3ACContentChangeClass(left.Role); changeClass != "" {
				result.Changes = append(result.Changes, actorContextR3ACRootChangeProbe{
					RootIdentity: identity,
					RootRole:     left.Role,
					ChangeClass:  changeClass,
				})
			}
		}
		if left.Locator != right.Locator {
			result.Changes = append(result.Changes, actorContextR3ACRootChangeProbe{
				RootIdentity: identity,
				RootRole:     left.Role,
				ChangeClass:  actorContextR3ACChangeLocatorRefresh,
			})
		}
	}

	sort.Slice(result.Changes, func(i, j int) bool {
		if result.Changes[i].RootIdentity != result.Changes[j].RootIdentity {
			return result.Changes[i].RootIdentity < result.Changes[j].RootIdentity
		}
		return result.Changes[i].ChangeClass < result.Changes[j].ChangeClass
	})
	return result
}

func actorContextR3ACContentChangeClass(role string) string {
	switch role {
	case actorContextR3ABRoleReferenceEvidence:
		return actorContextR3ACChangeReferenceEvidence
	case actorContextR3ABRoleRuntimeObservation:
		return actorContextR3ACChangeRuntimeObservation
	case actorContextR3ABRoleTestOnlyContext:
		return actorContextR3ACChangeTestOnlyContext
	case actorContextR3ABRoleCorroboration:
		return actorContextR3ACChangeCorroboration
	default:
		return ""
	}
}

func actorContextR3ACRootAddress(
	value actorContextR3ABProvenanceRootProbe,
) string {
	return value.Identity + "|" + value.Role
}

func actorContextR3ACRootsByIdentity(
	values []actorContextR3ABProvenanceRootProbe,
) map[string]actorContextR3ABProvenanceRootProbe {
	out := make(map[string]actorContextR3ABProvenanceRootProbe, len(values))
	for _, value := range values {
		out[actorContextR3ACRootAddress(value)] = value
	}
	return out
}

func actorContextR3ACCloneRoots(
	values []actorContextR3ABProvenanceRootProbe,
) []actorContextR3ABProvenanceRootProbe {
	return append([]actorContextR3ABProvenanceRootProbe(nil), values...)
}

func actorContextR3ACRootIndexByRole(
	values []actorContextR3ABProvenanceRootProbe,
	role string,
) int {
	for i := range values {
		if values[i].Role == role {
			return i
		}
	}
	return -1
}

func actorContextR3ACHasChange(
	values []actorContextR3ACRootChangeProbe,
	identity string,
	changeClass string,
) bool {
	for _, value := range values {
		if value.RootIdentity == identity && value.ChangeClass == changeClass {
			return true
		}
	}
	return false
}
