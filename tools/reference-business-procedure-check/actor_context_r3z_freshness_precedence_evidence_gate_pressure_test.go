package main

import (
	"reflect"
	"testing"
)

const (
	actorContextR3ZGateInsufficient            = "insufficient-evidence"
	actorContextR3ZGateSameSourceCompatibility = "same-source-compatibility-unresolved"
	actorContextR3ZGateSynthetic               = "synthetic-conflict-pressure"
	actorContextR3ZGateQualified               = "precedence-mechanism-pressure-qualified"
)

type actorContextR3ZConflictEvidenceProbe struct {
	SameAuthorizationTuple         bool
	ExplicitReevaluationDirective  bool
	ExplicitReuseBypassDirective   bool
	ExplicitReevaluationSource     string
	ExplicitReuseSource            string
	SameApplicabilityScope         bool
	RepositoryBacked               bool
	ProviderPolicyRevisionObserved bool
}

type actorContextR3ZGateResultProbe struct {
	Status          string
	BlockingReasons []string
	AuthorityLimits []string
}

func TestActorContextR3YGenericReusePermissionDoesNotProvePrecedenceConflict(t *testing.T) {
	got := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		[]actorContextR3YFreshnessDirectiveProbe{
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReevaluationRequired,
				SourceRef:     "test-only:freshness-authority/reevaluate",
			},
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReusePermitted,
				SourceRef:     "test-only:freshness-authority/reuse",
			},
		},
	)
	if got.PrecedenceStatus != "unresolved" {
		t.Fatalf("test setup changed: R3Y no longer leaves coexistence unresolved: %+v", got)
	}

	gate := actorContextR3ZEvaluateGate(actorContextR3ZConflictEvidenceProbe{
		SameAuthorizationTuple:        true,
		ExplicitReevaluationDirective: true,
		ExplicitReuseBypassDirective:  false,
		ExplicitReevaluationSource:    "test-only:freshness-authority/reevaluate",
		ExplicitReuseSource:           "test-only:freshness-authority/reuse",
		SameApplicabilityScope:        true,
		RepositoryBacked:              false,
	})
	if gate.Status != actorContextR3ZGateInsufficient {
		t.Fatalf("generic reuse permission was mistaken for bypass-capable conflict Evidence: %+v", gate)
	}
	if !actorContextR0Contains(gate.BlockingReasons, "reuse-bypass-semantics-unresolved") {
		t.Fatalf("R3Z lost the distinction between generic reuse and no-reevaluation reuse: %+v", gate)
	}
}

func TestActorContextR3ZSameSourceConflictKeepsCompatibilityUnresolved(t *testing.T) {
	got := actorContextR3ZEvaluateGate(actorContextR3ZConflictEvidenceProbe{
		SameAuthorizationTuple:        true,
		ExplicitReevaluationDirective: true,
		ExplicitReuseBypassDirective:  true,
		ExplicitReevaluationSource:    "test-only:freshness-authority/same-source",
		ExplicitReuseSource:           "test-only:freshness-authority/same-source",
		SameApplicabilityScope:        true,
		RepositoryBacked:              true,
	})

	if got.Status != actorContextR3ZGateSameSourceCompatibility {
		t.Fatalf("same-source coexistence was misclassified as cross-authority precedence conflict: %+v", got)
	}
	if !actorContextR0Contains(got.BlockingReasons, "cross-authority-provenance-unresolved") {
		t.Fatalf("R3Z lost same-source compatibility blocker: %+v", got)
	}
}

func TestActorContextR3ZSyntheticExplicitConflictDoesNotJustifyProductionPrecedence(t *testing.T) {
	got := actorContextR3ZEvaluateGate(actorContextR3ZConflictEvidenceProbe{
		SameAuthorizationTuple:        true,
		ExplicitReevaluationDirective: true,
		ExplicitReuseBypassDirective:  true,
		ExplicitReevaluationSource:    "test-only:freshness-authority/reevaluate",
		ExplicitReuseSource:           "test-only:freshness-authority/reuse-without-reevaluation",
		SameApplicabilityScope:        true,
		RepositoryBacked:              false,
	})

	if got.Status != actorContextR3ZGateSynthetic {
		t.Fatalf("synthetic explicit contradiction was misclassified: %+v", got)
	}
	if !actorContextR0Contains(got.BlockingReasons, "repository-backed-business-need-unresolved") {
		t.Fatalf("R3Z lost repository-backed Evidence blocker: %+v", got)
	}
}

func TestActorContextR3ZRepositoryBackedSameScopeConflictCanOpenMechanismPressure(t *testing.T) {
	got := actorContextR3ZEvaluateGate(actorContextR3ZConflictEvidenceProbe{
		SameAuthorizationTuple:        true,
		ExplicitReevaluationDirective: true,
		ExplicitReuseBypassDirective:  true,
		ExplicitReevaluationSource:    "accepted:test-only-example/reevaluate",
		ExplicitReuseSource:           "accepted:test-only-example/reuse-without-reevaluation",
		SameApplicabilityScope:        true,
		RepositoryBacked:              true,
	})

	if got.Status != actorContextR3ZGateQualified {
		t.Fatalf("fully explicit repository-backed conflict did not open precedence mechanism pressure: %+v", got)
	}
	if len(got.BlockingReasons) != 0 {
		t.Fatalf("qualified Gate retained blockers: %+v", got.BlockingReasons)
	}
}

func TestActorContextR3ZTupleOrScopeMismatchDoesNotQualifyConflict(t *testing.T) {
	cases := []struct {
		name        string
		evidence    actorContextR3ZConflictEvidenceProbe
		wantBlocker string
	}{
		{
			name: "tuple-mismatch",
			evidence: actorContextR3ZConflictEvidenceProbe{
				SameAuthorizationTuple:        false,
				ExplicitReevaluationDirective: true,
				ExplicitReuseBypassDirective:  true,
				ExplicitReevaluationSource:    "accepted:a",
				ExplicitReuseSource:           "accepted:b",
				SameApplicabilityScope:        true,
				RepositoryBacked:              true,
			},
			wantBlocker: "authorization-context-tuple-mismatch",
		},
		{
			name: "scope-mismatch",
			evidence: actorContextR3ZConflictEvidenceProbe{
				SameAuthorizationTuple:        true,
				ExplicitReevaluationDirective: true,
				ExplicitReuseBypassDirective:  true,
				ExplicitReevaluationSource:    "accepted:a",
				ExplicitReuseSource:           "accepted:b",
				SameApplicabilityScope:        false,
				RepositoryBacked:              true,
			},
			wantBlocker: "applicability-scope-mismatch",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := actorContextR3ZEvaluateGate(tc.evidence)
			if got.Status != actorContextR3ZGateInsufficient {
				t.Fatalf("%s incorrectly qualified precedence conflict: %+v", tc.name, got)
			}
			if !actorContextR0Contains(got.BlockingReasons, tc.wantBlocker) {
				t.Fatalf("%s lost blocker %q: %+v", tc.name, tc.wantBlocker, got)
			}
		})
	}
}

func TestActorContextR3ZProviderPolicyRevisionDoesNotFillMissingFreshnessDirective(t *testing.T) {
	got := actorContextR3ZEvaluateGate(actorContextR3ZConflictEvidenceProbe{
		SameAuthorizationTuple:         true,
		ExplicitReevaluationDirective:  false,
		ExplicitReuseBypassDirective:   false,
		SameApplicabilityScope:         true,
		RepositoryBacked:               true,
		ProviderPolicyRevisionObserved: true,
	})

	if got.Status != actorContextR3ZGateInsufficient {
		t.Fatalf("provider policy revision became precedence authority: %+v", got)
	}
	if !actorContextR0Contains(got.BlockingReasons, "explicit-reevaluation-directive-unresolved") ||
		!actorContextR0Contains(got.BlockingReasons, "explicit-reuse-bypass-directive-unresolved") {
		t.Fatalf("provider revision filled missing explicit directives: %+v", got)
	}
}

func TestActorContextR3ZGateShapeHasNoPrecedenceWinnerShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ZGateResultProbe{})
	for _, forbidden := range []string{
		"Winner",
		"Priority",
		"EffectiveDirective",
		"PreferredDirective",
		"PrecedenceRule",
		"ShouldReevaluate",
		"Reusable",
		"CacheValid",
		"Authorized",
		"Allowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3Z Evidence Gate introduced forbidden precedence shortcut %q", forbidden)
		}
	}
}

func actorContextR3ZEvaluateGate(
	evidence actorContextR3ZConflictEvidenceProbe,
) actorContextR3ZGateResultProbe {
	result := actorContextR3ZGateResultProbe{
		AuthorityLimits: []string{
			"generic-reuse-permission-is-not-reevaluation-bypass-authority",
			"same-source-coexistence-does-not-imply-cross-authority-precedence",
			"provider-policy-revision-is-not-freshness-directive-authority",
			"synthetic-conflict-does-not-justify-production-precedence",
			"qualified-conflict-does-not-select-precedence-winner",
			"scope-specificity-does-not-select-winner",
			"source-order-does-not-select-winner",
		},
	}

	blockers := []string{}
	if !evidence.SameAuthorizationTuple {
		blockers = append(blockers, "authorization-context-tuple-mismatch")
	}
	if !evidence.ExplicitReevaluationDirective ||
		evidence.ExplicitReevaluationSource == "" {
		blockers = append(blockers, "explicit-reevaluation-directive-unresolved")
	}
	if !evidence.ExplicitReuseBypassDirective {
		blockers = append(blockers, "reuse-bypass-semantics-unresolved")
		blockers = append(blockers, "explicit-reuse-bypass-directive-unresolved")
	} else if evidence.ExplicitReuseSource == "" {
		blockers = append(blockers, "explicit-reuse-bypass-directive-unresolved")
	}
	if !evidence.SameApplicabilityScope {
		blockers = append(blockers, "applicability-scope-mismatch")
	}

	blockers = actorContextR0SortedUnique(blockers)
	if len(blockers) > 0 {
		result.Status = actorContextR3ZGateInsufficient
		result.BlockingReasons = blockers
		return result
	}

	if evidence.ExplicitReevaluationSource == evidence.ExplicitReuseSource {
		result.Status = actorContextR3ZGateSameSourceCompatibility
		result.BlockingReasons = []string{"cross-authority-provenance-unresolved"}
		return result
	}

	if !evidence.RepositoryBacked {
		result.Status = actorContextR3ZGateSynthetic
		result.BlockingReasons = []string{"repository-backed-business-need-unresolved"}
		return result
	}

	result.Status = actorContextR3ZGateQualified
	return result
}
