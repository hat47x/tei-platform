package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR3AIReadOnlyReasoningProbe struct {
	Audience                    string
	Purpose                     string
	FreshnessStopReason         string
	ResolutionNeedClasses       []string
	ReevaluationAuthorityStatus string
	ReuseAuthorityStatus        string
	CategoryOutcomes            []actorContextR3AFCategoryOutcomeProbe
	UnselectedChanges           []actorContextR3TObservedChangeProbe
	ResidualRootDeltas          []actorContextR3ADResidualRootDeltaProbe
	OutcomeTraces               []actorContextR3AHOutcomeTraceProbe
	ResidualTraces              []actorContextR3AHResidualTraceProbe
	DrillDownRefs               []string
	AuthorityLimits             []string
}

func TestActorContextR3AIConsumesAgentMaterializationAndTraceWithoutLosingIdentity(t *testing.T) {
	materialized, explanation, roots, bindings := actorContextR3AIRequirementFixture(
		t,
		actorContextR3AFFreshnessAuthority(),
		false,
	)
	trace, err := actorContextR3AHDrillDownOutcome(
		materialized,
		explanation,
		roots,
		bindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := actorContextR3AIConsumeReadOnly(
		materialized,
		explanation,
		[]actorContextR3AHOutcomeTraceProbe{trace.Outcome},
		nil,
	)

	if got.Audience != actorContextR3PAudienceAgent ||
		got.Purpose != actorContextR3PPurposeResolutionInspection ||
		got.FreshnessStopReason != explanation.FreshnessStopReason ||
		got.ReevaluationAuthorityStatus != explanation.ReevaluationAuthorityStatus ||
		got.ReuseAuthorityStatus != explanation.ReuseAuthorityStatus {
		t.Fatalf("R3AI changed Agent explanation identity: got=%+v explanation=%+v", got, explanation)
	}
	if !reflect.DeepEqual(
		got.CategoryOutcomes,
		actorContextR3AGNormalizeCategoryOutcomes(explanation.CategoryOutcomes),
	) {
		t.Fatalf("R3AI lost exact category outcomes: got=%+v explanation=%+v",
			got.CategoryOutcomes, explanation.CategoryOutcomes)
	}
	if len(got.OutcomeTraces) != 1 ||
		got.OutcomeTraces[0].SummaryOutcomeIdentity !=
			actorContextR3AGOutcomeIdentity(*actorContextR3AFCategory(
				explanation, actorContextR3PCategoryMachineReasons,
			)) {
		t.Fatalf("R3AI lost exact R3AH outcome trace identity: %+v", got.OutcomeTraces)
	}
}

func TestActorContextR3AIResolutionNeedDoesNotBecomeRemediationSelection(t *testing.T) {
	materialized, explanation, _, _ := actorContextR3AIRequirementFixture(
		t,
		actorContextR3AFFreshnessAuthority(),
		false,
	)
	got := actorContextR3AIConsumeReadOnly(materialized, explanation, nil, nil)

	if !reflect.DeepEqual(got.ResolutionNeedClasses, []string{
		"authorization-evaluation-freshness-authority",
	}) {
		t.Fatalf("R3AI lost diagnostic resolution need class: %+v", got)
	}
	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("R3AI resolution need fabricated authority decision: %+v", got)
	}
}

func TestActorContextR3AIUnchangedOutcomeDoesNotBecomeReuseContinueOrFreshDecision(t *testing.T) {
	materialized, explanation, _, _ := actorContextR3AIRequirementFixture(
		t,
		actorContextR3AFFreshnessAuthority(),
		false,
	)
	got := actorContextR3AIConsumeReadOnly(materialized, explanation, nil, nil)

	machine := actorContextR3AGOutcome(got.CategoryOutcomes, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusUnchanged {
		t.Fatalf("R3AI setup lost unchanged machine outcome: %+v", got)
	}
	if got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("unchanged category outcome fabricated reuse authority: %+v", got)
	}
}

func TestActorContextR3AIChangedOutcomeDoesNotBecomeRetryRebuildOrEscalation(t *testing.T) {
	materialized, explanation, _, _ := actorContextR3AIRequirementFixture(
		t,
		actorContextR3AFFreshnessAuthority(),
		true,
	)
	got := actorContextR3AIConsumeReadOnly(materialized, explanation, nil, nil)

	machine := actorContextR3AGOutcome(got.CategoryOutcomes, actorContextR3PCategoryMachineReasons)
	if machine == nil || machine.Status != actorContextR3WStatusChanged {
		t.Fatalf("R3AI setup lost changed machine outcome: %+v", got)
	}
	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("changed category outcome selected an authority response: %+v", got)
	}
}

func TestActorContextR3AIPriorObservationResidualDoesNotBecomeCurrentRuntimeDecision(t *testing.T) {
	materialized, explanation, roots := actorContextR3AHPriorObservationFixture(t)
	trace, err := actorContextR3AHDrillDownResiduals(materialized, explanation, roots)
	if err != nil {
		t.Fatal(err)
	}
	got := actorContextR3AIConsumeReadOnly(
		materialized,
		explanation,
		nil,
		trace.Residuals,
	)

	if len(got.CategoryOutcomes) != 0 {
		t.Fatalf("prior Observation residual fabricated category outcome: %+v", got)
	}
	if len(got.ResidualRootDeltas) != 1 ||
		got.ResidualRootDeltas[0].Reason != actorContextR3ADResidualNoBinding ||
		got.ResidualRootDeltas[0].Delta.RootRole != actorContextR3ABRoleRuntimeObservation {
		t.Fatalf("R3AI lost prior Observation no-binding residual: %+v", got)
	}
	if len(got.ResidualTraces) != 1 ||
		got.ResidualTraces[0].RootRole != actorContextR3ABRoleRuntimeObservation {
		t.Fatalf("R3AI lost prior Observation provenance trace: %+v", got.ResidualTraces)
	}
}

func TestActorContextR3AIExplicitReevaluationRequiredDoesNotAuthorizeAgentInvocation(t *testing.T) {
	authority := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		[]actorContextR3YFreshnessDirectiveProbe{{
			AuthorityKind: "evaluation-freshness-directive",
			Decision:      actorContextR3YDirectiveReevaluationRequired,
			SourceRef:     "test-only:r3ai/reevaluate",
		}},
	)
	materialized, explanation, _, _ := actorContextR3AIRequirementFixture(t, authority, false)
	got := actorContextR3AIConsumeReadOnly(materialized, explanation, nil, nil)

	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationRequired {
		t.Fatalf("R3AI lost explicit reevaluation authority fact: %+v", got)
	}
	if got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("reevaluation directive fabricated reuse authority: %+v", got)
	}
}

func TestActorContextR3AIExplicitReusePermissionDoesNotAuthorizeProtectedOperation(t *testing.T) {
	authority := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{},
		[]actorContextR3YFreshnessDirectiveProbe{{
			AuthorityKind: "evaluation-freshness-directive",
			Decision:      actorContextR3YDirectiveReusePermitted,
			SourceRef:     "test-only:r3ai/reuse",
		}},
	)
	materialized, explanation, _, _ := actorContextR3AIRequirementFixture(t, authority, false)
	got := actorContextR3AIConsumeReadOnly(materialized, explanation, nil, nil)

	if got.ReuseAuthorityStatus != actorContextR3YReusePermitted {
		t.Fatalf("R3AI lost explicit reuse authority fact: %+v", got)
	}
	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved {
		t.Fatalf("reuse permission fabricated reevaluation-not-required authority: %+v", got)
	}
}

func TestActorContextR3AIRepeatedConsumptionAndInputOrderAreIdentityStable(t *testing.T) {
	materialized, explanation, roots, bindings := actorContextR3AIRequirementFixture(
		t,
		actorContextR3AFFreshnessAuthority(),
		false,
	)
	trace, err := actorContextR3AHDrillDownOutcome(
		materialized, explanation, roots, bindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}

	first := actorContextR3AIConsumeReadOnly(
		materialized, explanation,
		[]actorContextR3AHOutcomeTraceProbe{trace.Outcome}, nil,
	)
	second := actorContextR3AIConsumeReadOnly(
		materialized, explanation,
		[]actorContextR3AHOutcomeTraceProbe{trace.Outcome}, nil,
	)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated R3AI consumption changed normalized identity: first=%+v second=%+v", first, second)
	}

	reversed := materialized
	reversed.DetailedCategoryOutcomes = append(
		[]actorContextR3AFCategoryOutcomeProbe(nil),
		materialized.DetailedCategoryOutcomes...,
	)
	for i, j := 0, len(reversed.DetailedCategoryOutcomes)-1; i < j; i, j = i+1, j-1 {
		reversed.DetailedCategoryOutcomes[i], reversed.DetailedCategoryOutcomes[j] =
			reversed.DetailedCategoryOutcomes[j], reversed.DetailedCategoryOutcomes[i]
	}
	reversed.InlineUnselectedChanges = append(
		[]actorContextR3TObservedChangeProbe(nil),
		materialized.InlineUnselectedChanges...,
	)
	for i, j := 0, len(reversed.InlineUnselectedChanges)-1; i < j; i, j = i+1, j-1 {
		reversed.InlineUnselectedChanges[i], reversed.InlineUnselectedChanges[j] =
			reversed.InlineUnselectedChanges[j], reversed.InlineUnselectedChanges[i]
	}

	normalized := actorContextR3AIConsumeReadOnly(
		reversed, explanation,
		[]actorContextR3AHOutcomeTraceProbe{trace.Outcome}, nil,
	)
	if !reflect.DeepEqual(first, normalized) {
		t.Fatalf("R3AI input order changed normalized reasoning identity: first=%+v normalized=%+v",
			first, normalized)
	}
}

func TestActorContextR3AIReadOnlyShapeHasNoRemediationOrExecutionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3AIReadOnlyReasoningProbe{})
	for _, forbidden := range []string{
		"ToolCall",
		"InvocationPlan",
		"Command",
		"SuggestedAction",
		"NextAction",
		"Remediation",
		"Retry",
		"RetryRequired",
		"Reauthorize",
		"AutoReauthorize",
		"Rebuild",
		"AutoRebuild",
		"Escalate",
		"ApprovalRequest",
		"ApprovalRequired",
		"HandoffOwner",
		"CanExecute",
		"Authorized",
		"Allowed",
		"Continue",
		"Reusable",
		"Safe",
		"Fresh",
		"Confidence",
		"Severity",
		"Priority",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AI read-only reasoning introduced forbidden action/authority shortcut %q", forbidden)
		}
	}
}

func actorContextR3AIRequirementFixture(
	t *testing.T,
	authority actorContextR3YAuthorityResultProbe,
	changeMeaning bool,
) (
	actorContextR3AGMaterializedExplanationProbe,
	actorContextR3AFExplanationProbe,
	[]actorContextR3ABProvenanceRootProbe,
	[]actorContextR3ADRootDependencyBindingProbe,
) {
	t.Helper()

	roots, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AI setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(roots, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	view := actorContextR3AEView(t, actorContextR3PAudienceAgent)
	before := actorContextR3ABFreshnessCore(t)
	after := actorContextR3QCloneCore(before)
	if changeMeaning {
		after.MachineReasons = actorContextR0SortedUnique(append(
			after.MachineReasons,
			"test-only:r3ai-changed-machine-meaning",
		))
	}
	result, err := actorContextR3AEEvaluate(before, after, view, bridge)
	if err != nil {
		t.Fatal(err)
	}
	explanation := actorContextR3AFBuildExplanation(view, result, bridge, authority)
	materialized, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}
	return materialized, explanation, roots, bindings
}

func actorContextR3AIConsumeReadOnly(
	materialized actorContextR3AGMaterializedExplanationProbe,
	explanation actorContextR3AFExplanationProbe,
	outcomeTraces []actorContextR3AHOutcomeTraceProbe,
	residualTraces []actorContextR3AHResidualTraceProbe,
) actorContextR3AIReadOnlyReasoningProbe {
	out := actorContextR3AIReadOnlyReasoningProbe{
		Audience:                    materialized.Audience,
		Purpose:                     materialized.Purpose,
		FreshnessStopReason:         materialized.FreshnessStopReason,
		ResolutionNeedClasses:       actorContextR0SortedUnique(append([]string(nil), materialized.ResolutionNeedClasses...)),
		ReevaluationAuthorityStatus: materialized.ReevaluationAuthorityStatus,
		ReuseAuthorityStatus:        materialized.ReuseAuthorityStatus,
		CategoryOutcomes: actorContextR3AGNormalizeCategoryOutcomes(
			materialized.DetailedCategoryOutcomes,
		),
		UnselectedChanges: actorContextR3TUniqueChanges(
			materialized.InlineUnselectedChanges,
		),
		ResidualRootDeltas: actorContextR3AGNormalizeResiduals(
			materialized.InlineResidualRootDeltas,
		),
		OutcomeTraces:  actorContextR3AINormalizeOutcomeTraces(outcomeTraces),
		ResidualTraces: actorContextR3AINormalizeResidualTraces(residualTraces),
		DrillDownRefs:  actorContextR0SortedUnique(append([]string(nil), materialized.DrillDownRefs...)),
		AuthorityLimits: actorContextR0SortedUnique(append(
			append([]string(nil), explanation.AuthorityLimits...),
			"read-only-reasoning-does-not-select-remediation",
			"resolution-need-does-not-select-owner-tool-or-workflow",
			"reevaluation-authority-does-not-author-agent-invocation",
			"reuse-authority-does-not-authorize-protected-operation",
			"category-outcome-does-not-select-response-policy",
			"provenance-detail-does-not-strengthen-authority",
			"repeated-consumption-does-not-strengthen-authority",
		)),
	}
	return out
}

func actorContextR3AINormalizeOutcomeTraces(
	values []actorContextR3AHOutcomeTraceProbe,
) []actorContextR3AHOutcomeTraceProbe {
	out := append([]actorContextR3AHOutcomeTraceProbe(nil), values...)
	for i := range out {
		out[i].Reasons = append([]actorContextR3AHReasonTraceProbe(nil), out[i].Reasons...)
		sort.Slice(out[i].Reasons, func(left, right int) bool {
			if out[i].Reasons[left].ReasonIdentity != out[i].Reasons[right].ReasonIdentity {
				return out[i].Reasons[left].ReasonIdentity < out[i].Reasons[right].ReasonIdentity
			}
			if out[i].Reasons[left].DependencyKey != out[i].Reasons[right].DependencyKey {
				return out[i].Reasons[left].DependencyKey < out[i].Reasons[right].DependencyKey
			}
			return out[i].Reasons[left].RootAddress < out[i].Reasons[right].RootAddress
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].SummaryOutcomeIdentity < out[j].SummaryOutcomeIdentity
	})
	return out
}

func actorContextR3AINormalizeResidualTraces(
	values []actorContextR3AHResidualTraceProbe,
) []actorContextR3AHResidualTraceProbe {
	out := append([]actorContextR3AHResidualTraceProbe(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].RootAddress != out[j].RootAddress {
			return out[i].RootAddress < out[j].RootAddress
		}
		if out[i].ResidualReason != out[j].ResidualReason {
			return out[i].ResidualReason < out[j].ResidualReason
		}
		return out[i].CurrentLocator < out[j].CurrentLocator
	})
	return out
}
