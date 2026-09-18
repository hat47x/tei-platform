package main

import (
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3YReevaluationUnresolved = "reevaluation-authority-unresolved"
	actorContextR3YReevaluationRequired   = "explicit-reevaluation-required"

	actorContextR3YReuseUnresolved = "prior-observation-reuse-authority-unresolved"
	actorContextR3YReusePermitted  = "explicit-reuse-permitted"

	actorContextR3YDirectiveReevaluationRequired = "reevaluation-required"
	actorContextR3YDirectiveReusePermitted       = "prior-observation-reuse-permitted"
)

type actorContextR3YFreshnessDirectiveProbe struct {
	AuthorityKind string
	Decision      string
	SourceRef     string
}

type actorContextR3YFreshnessPressureProbe struct {
	StaticSemanticChanged   bool
	RequirementTransition   bool
	TargetAuthorityRevision bool
	ProviderPolicyRevision  bool
	LocatorOnlyChange       bool
}

type actorContextR3YAuthorityResultProbe struct {
	ContextStatus               string
	ReevaluationAuthorityStatus string
	ReuseAuthorityStatus        string
	PressureKinds               []string
	DirectiveSources            []string
	PrecedenceStatus            string
	AuthorityLimits             []string
}

func TestActorContextR3YSameTupleStaticChangeWithoutDirectiveKeepsBothAxesUnresolved(t *testing.T) {
	impact := actorContextR3XContextImpactProbe{
		Status:                     actorContextR3XStatusTupleCompatibleFreshnessUnresolved,
		ObservationTupleStatus:     actorContextR3XObservationTupleMatch,
		FreshnessAuthorityStatus:   "unresolved",
	}

	got := actorContextR3YAnalyzeFreshnessAuthority(
		impact,
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		nil,
	)

	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("R3Y inferred freshness authority from static change: %+v", got)
	}
	if !reflect.DeepEqual(got.PressureKinds, []string{"static-semantic-change"}) {
		t.Fatalf("R3Y lost static semantic pressure: %+v", got.PressureKinds)
	}
}

func TestActorContextR3YExplicitReevaluationDoesNotAuthorReuseDecision(t *testing.T) {
	got := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		[]actorContextR3YFreshnessDirectiveProbe{
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReevaluationRequired,
				SourceRef:     "test-only:freshness-authority/reevaluate",
			},
		},
	)

	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationRequired {
		t.Fatalf("explicit reevaluation directive was not preserved: %+v", got)
	}
	if got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("reevaluation directive fabricated reuse prohibition/permission: %+v", got)
	}
}

func TestActorContextR3YExplicitReuseDoesNotAuthorReevaluationNotRequired(t *testing.T) {
	got := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{},
		[]actorContextR3YFreshnessDirectiveProbe{
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReusePermitted,
				SourceRef:     "test-only:freshness-authority/reuse",
			},
		},
	)

	if got.ReuseAuthorityStatus != actorContextR3YReusePermitted {
		t.Fatalf("explicit reuse permission was not preserved: %+v", got)
	}
	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved {
		t.Fatalf("reuse permission fabricated reevaluation-not-required authority: %+v", got)
	}
}

func TestActorContextR3YBothDirectivesPreserveAxesAndLeavePrecedenceUnresolved(t *testing.T) {
	got := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		[]actorContextR3YFreshnessDirectiveProbe{
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReusePermitted,
				SourceRef:     "test-only:freshness-authority/reuse",
			},
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReevaluationRequired,
				SourceRef:     "test-only:freshness-authority/reevaluate",
			},
		},
	)

	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationRequired ||
		got.ReuseAuthorityStatus != actorContextR3YReusePermitted {
		t.Fatalf("R3Y collapsed coexisting explicit directives: %+v", got)
	}
	if got.PrecedenceStatus != "unresolved" {
		t.Fatalf("R3Y invented precedence for coexisting directives: %+v", got)
	}
	if !reflect.DeepEqual(got.DirectiveSources, []string{
		"test-only:freshness-authority/reevaluate",
		"test-only:freshness-authority/reuse",
	}) {
		t.Fatalf("R3Y lost explicit directive provenance: %+v", got.DirectiveSources)
	}
}

func TestActorContextR3YRequirementTransitionIsNotResolvedByFreshnessDirective(t *testing.T) {
	impact := actorContextR3XContextImpactProbe{
		Status:                         actorContextR3XStatusRequirementTransitionUnresolved,
		ObservationTupleStatus:         actorContextR3XObservationTupleMatch,
		RequirementApplicabilityChanged: true,
		FreshnessAuthorityStatus:       "unresolved",
	}

	got := actorContextR3YAnalyzeFreshnessAuthority(
		impact,
		actorContextR3YFreshnessPressureProbe{RequirementTransition: true},
		[]actorContextR3YFreshnessDirectiveProbe{
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReusePermitted,
				SourceRef:     "test-only:freshness-authority/reuse",
			},
		},
	)

	if got.ContextStatus != "requirement-transition-authority-unresolved" {
		t.Fatalf("R3Y freshness rule bypassed requirement transition authority: %+v", got)
	}
	if got.ReuseAuthorityStatus != actorContextR3YReusePermitted {
		t.Fatalf("R3Y should preserve explicit reuse directive without using it to solve applicability: %+v", got)
	}
}

func TestActorContextR3YProviderPolicyRevisionAloneAuthorsNeitherAxis(t *testing.T) {
	got := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{ProviderPolicyRevision: true},
		nil,
	)

	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("provider policy revision became freshness authority: %+v", got)
	}
	if !reflect.DeepEqual(got.PressureKinds, []string{"provider-policy-revision"}) {
		t.Fatalf("provider policy pressure was not retained distinctly: %+v", got.PressureKinds)
	}
}

func TestActorContextR3YLocatorOnlyChangeCreatesNoAuthorizationFreshnessPressure(t *testing.T) {
	got := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{LocatorOnlyChange: true},
		nil,
	)

	if len(got.PressureKinds) != 0 {
		t.Fatalf("locator-only change leaked into Authorization freshness pressure: %+v", got)
	}
	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("locator-only change changed Authorization freshness authority: %+v", got)
	}
}

func TestActorContextR3YTupleMismatchMakesPriorReuseComparisonNotApplicable(t *testing.T) {
	impact := actorContextR3XContextImpactProbe{
		Status:                 actorContextR3XStatusObservationContextMismatch,
		ObservationTupleStatus: actorContextR3XObservationTupleMismatch,
		ChangedTupleComponents: []string{"action"},
	}

	got := actorContextR3YAnalyzeFreshnessAuthority(
		impact,
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		[]actorContextR3YFreshnessDirectiveProbe{
			{
				AuthorityKind: "evaluation-freshness-directive",
				Decision:      actorContextR3YDirectiveReusePermitted,
				SourceRef:     "test-only:freshness-authority/reuse",
			},
		},
	)

	if got.ContextStatus != "prior-observation-context-not-applicable" {
		t.Fatalf("R3Y continued same-context reuse comparison after tuple mismatch: %+v", got)
	}
	if got.ReuseAuthorityStatus != "not-applicable" {
		t.Fatalf("R3Y represented tuple mismatch as reuse prohibition/permission: %+v", got)
	}
}

func TestActorContextR3YShapeHasNoCacheOrAutomaticDecisionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3YAuthorityResultProbe{})
	for _, forbidden := range []string{
		"Reusable",
		"CacheValid",
		"Valid",
		"ShouldReevaluate",
		"ReevaluationRequired",
		"AutoReauthorize",
		"TTL",
		"ExpiresAt",
		"Winner",
		"Preferred",
		"Authorized",
		"Allowed",
		"CanExecute",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3Y authority result introduced forbidden shortcut %q", forbidden)
		}
	}
}

func actorContextR3YTupleMatchImpact() actorContextR3XContextImpactProbe {
	return actorContextR3XContextImpactProbe{
		Status:                   actorContextR3XStatusTupleCompatibleFreshnessUnresolved,
		ObservationTupleStatus:   actorContextR3XObservationTupleMatch,
		FreshnessAuthorityStatus: "unresolved",
	}
}

func actorContextR3YAnalyzeFreshnessAuthority(
	contextImpact actorContextR3XContextImpactProbe,
	pressure actorContextR3YFreshnessPressureProbe,
	directives []actorContextR3YFreshnessDirectiveProbe,
) actorContextR3YAuthorityResultProbe {
	result := actorContextR3YAuthorityResultProbe{
		ContextStatus:               "tuple-compatible",
		ReevaluationAuthorityStatus: actorContextR3YReevaluationUnresolved,
		ReuseAuthorityStatus:        actorContextR3YReuseUnresolved,
		PrecedenceStatus:            "not-applicable",
		AuthorityLimits: []string{
			"tuple-match-does-not-prove-freshness",
			"change-pressure-does-not-author-reevaluation",
			"change-pressure-does-not-author-reuse",
			"reevaluation-directive-does-not-imply-reuse-prohibition",
			"reuse-permission-does-not-imply-reevaluation-not-required",
			"provider-policy-revision-is-not-freshness-authority",
			"freshness-authority-does-not-resolve-requirement-applicability",
		},
	}

	if contextImpact.ObservationTupleStatus == actorContextR3XObservationTupleMismatch {
		result.ContextStatus = "prior-observation-context-not-applicable"
		result.ReuseAuthorityStatus = "not-applicable"
		result.ReevaluationAuthorityStatus = "not-applicable"
		return result
	}

	if contextImpact.Status == actorContextR3XStatusRequirementTransitionUnresolved ||
		pressure.RequirementTransition {
		result.ContextStatus = "requirement-transition-authority-unresolved"
	}

	pressures := []string{}
	if pressure.StaticSemanticChanged {
		pressures = append(pressures, "static-semantic-change")
	}
	if pressure.TargetAuthorityRevision {
		pressures = append(pressures, "target-authority-revision")
	}
	if pressure.ProviderPolicyRevision {
		pressures = append(pressures, "provider-policy-revision")
	}
	result.PressureKinds = actorContextR0SortedUnique(pressures)

	reevaluationCount := 0
	reuseCount := 0
	for _, directive := range directives {
		if directive.AuthorityKind != "evaluation-freshness-directive" ||
			directive.SourceRef == "" {
			continue
		}
		switch directive.Decision {
		case actorContextR3YDirectiveReevaluationRequired:
			result.ReevaluationAuthorityStatus = actorContextR3YReevaluationRequired
			reevaluationCount++
			result.DirectiveSources = append(result.DirectiveSources, directive.SourceRef)
		case actorContextR3YDirectiveReusePermitted:
			result.ReuseAuthorityStatus = actorContextR3YReusePermitted
			reuseCount++
			result.DirectiveSources = append(result.DirectiveSources, directive.SourceRef)
		}
	}
	result.DirectiveSources = actorContextR0SortedUnique(result.DirectiveSources)

	if reevaluationCount > 0 && reuseCount > 0 {
		result.PrecedenceStatus = "unresolved"
	} else {
		result.PrecedenceStatus = "single-axis-or-none"
	}

	sort.Strings(result.PressureKinds)
	return result
}
