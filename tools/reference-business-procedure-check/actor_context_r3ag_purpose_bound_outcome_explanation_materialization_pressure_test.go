package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3AGDetailCategorySummary      = "category-outcome-summary"
	actorContextR3AGDetailCategoryFingerprints = "category-outcome-fingerprints"
	actorContextR3AGDetailCategoryReasons      = "category-outcome-reasons"
	actorContextR3AGDetailFreshnessAuthority   = "freshness-authority-status"
	actorContextR3AGDetailResolutionNeeds      = "resolution-needs"
	actorContextR3AGDetailUnselectedChanges    = "unselected-changes"
	actorContextR3AGDetailResidualRootDeltas   = "residual-root-deltas"
	actorContextR3AGDetailProvenanceDrilldown  = "provenance-drilldown"

	actorContextR3AGDispositionInline    = "inline"
	actorContextR3AGDispositionDrillDown = "drill-down"
	actorContextR3AGDispositionOmitted   = "omitted"
)

var actorContextR3AGAllDetailClasses = []string{
	actorContextR3AGDetailCategorySummary,
	actorContextR3AGDetailCategoryFingerprints,
	actorContextR3AGDetailCategoryReasons,
	actorContextR3AGDetailFreshnessAuthority,
	actorContextR3AGDetailResolutionNeeds,
	actorContextR3AGDetailUnselectedChanges,
	actorContextR3AGDetailResidualRootDeltas,
	actorContextR3AGDetailProvenanceDrilldown,
}

type actorContextR3AGOutcomeSummaryProbe struct {
	Category        string
	Status          string
	OutcomeIdentity string
}

type actorContextR3AGDetailDispositionProbe struct {
	DetailClass string
	Disposition string
}

type actorContextR3AGMaterializedExplanationProbe struct {
	Audience                    string
	Purpose                     string
	FreshnessStopReason         string
	ResolutionNeedClasses       []string
	ReevaluationAuthorityStatus string
	ReuseAuthorityStatus        string
	CategorySummaries           []actorContextR3AGOutcomeSummaryProbe
	DetailedCategoryOutcomes    []actorContextR3AFCategoryOutcomeProbe
	InlineUnselectedChanges     []actorContextR3TObservedChangeProbe
	InlineResidualRootDeltas    []actorContextR3ADResidualRootDeltaProbe
	DetailDispositions          []actorContextR3AGDetailDispositionProbe
	DrillDownRefs               []string
	DrillDownOutcomeIdentities  []string
	AuthorityLimits             []string
}

func TestActorContextR3AGHumanProfileSeparatesInlineDrillDownAndOmitted(t *testing.T) {
	view, explanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceHuman)
	got, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		actorContextR3AGDetailCategorySummary:      actorContextR3AGDispositionInline,
		actorContextR3AGDetailCategoryFingerprints: actorContextR3AGDispositionDrillDown,
		actorContextR3AGDetailCategoryReasons:      actorContextR3AGDispositionDrillDown,
		actorContextR3AGDetailFreshnessAuthority:   actorContextR3AGDispositionInline,
		actorContextR3AGDetailResolutionNeeds:      actorContextR3AGDispositionInline,
		actorContextR3AGDetailUnselectedChanges:    actorContextR3AGDispositionOmitted,
		actorContextR3AGDetailResidualRootDeltas:   actorContextR3AGDispositionDrillDown,
		actorContextR3AGDetailProvenanceDrilldown:  actorContextR3AGDispositionDrillDown,
	}
	for detail, disposition := range want {
		if gotDisposition := actorContextR3AGDispositionFor(got, detail); gotDisposition != disposition {
			t.Fatalf("R3AG Human disposition %q=%q want=%q: %+v",
				detail, gotDisposition, disposition, got.DetailDispositions)
		}
	}

	if len(got.DetailedCategoryOutcomes) != 0 ||
		len(got.InlineUnselectedChanges) != 0 ||
		len(got.InlineResidualRootDeltas) != 0 {
		t.Fatalf("R3AG Human leaked non-inline detail: %+v", got)
	}
	if len(got.CategorySummaries) == 0 ||
		len(got.DrillDownOutcomeIdentities) != len(got.CategorySummaries) {
		t.Fatalf("R3AG Human lost underlying outcome identity handles: %+v", got)
	}
	if got.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		got.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("R3AG Human presentation resolved freshness authority: %+v", got)
	}
	if len(got.DrillDownRefs) == 0 {
		t.Fatal("R3AG Human drill-down profile lost existing R3P refs")
	}
}

func TestActorContextR3AGAgentProfileKeepsExactOutcomeDetailInline(t *testing.T) {
	view, explanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceAgent)
	got, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}

	for _, detail := range []string{
		actorContextR3AGDetailCategorySummary,
		actorContextR3AGDetailCategoryFingerprints,
		actorContextR3AGDetailCategoryReasons,
		actorContextR3AGDetailFreshnessAuthority,
		actorContextR3AGDetailResolutionNeeds,
		actorContextR3AGDetailUnselectedChanges,
		actorContextR3AGDetailResidualRootDeltas,
	} {
		if actorContextR3AGDispositionFor(got, detail) != actorContextR3AGDispositionInline {
			t.Fatalf("R3AG Agent detail %q was not inline: %+v", detail, got.DetailDispositions)
		}
	}
	if actorContextR3AGDispositionFor(got, actorContextR3AGDetailProvenanceDrilldown) != actorContextR3AGDispositionDrillDown {
		t.Fatalf("R3AG Agent lost provenance drill-down: %+v", got.DetailDispositions)
	}
	if !reflect.DeepEqual(got.DetailedCategoryOutcomes, actorContextR3AGNormalizeCategoryOutcomes(explanation.CategoryOutcomes)) {
		t.Fatalf("R3AG Agent changed exact R3AF outcome detail: got=%+v upstream=%+v",
			got.DetailedCategoryOutcomes, explanation.CategoryOutcomes)
	}
}

func TestActorContextR3AGSharedOutcomeIdentityExcludesAudienceAndPresentation(t *testing.T) {
	humanView, humanExplanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceHuman)
	agentView, agentExplanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceAgent)

	human, err := actorContextR3AGMaterialize(humanView, humanExplanation)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR3AGMaterialize(agentView, agentExplanation)
	if err != nil {
		t.Fatal(err)
	}

	humanMachine := actorContextR3AGSummary(human, actorContextR3PCategoryMachineReasons)
	agentMachine := actorContextR3AGSummary(agent, actorContextR3PCategoryMachineReasons)
	if humanMachine == nil || agentMachine == nil || !reflect.DeepEqual(*humanMachine, *agentMachine) {
		t.Fatalf("R3AG audience/presentation changed shared machine outcome identity: human=%+v agent=%+v",
			humanMachine, agentMachine)
	}
	if actorContextR3AGSummary(human, actorContextR3PCategoryKnownAuthorityFacts) != nil {
		t.Fatalf("R3AG forced Agent-only category into Human materialization: %+v", human)
	}
	if actorContextR3AGSummary(agent, actorContextR3PCategoryKnownAuthorityFacts) == nil {
		t.Fatalf("R3AG Agent lost selected known-authority-facts summary: %+v", agent)
	}
}

func TestActorContextR3AGSelectedButEmptyResidualIsDrillDownNotOmitted(t *testing.T) {
	view, explanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceHuman)
	if len(explanation.ResidualRootDeltas) != 0 {
		t.Fatal("R3AG setup requires selected-but-empty residual detail")
	}
	got, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}
	if actorContextR3AGDispositionFor(got, actorContextR3AGDetailResidualRootDeltas) != actorContextR3AGDispositionDrillDown {
		t.Fatalf("R3AG converted selected-but-empty residual detail to omitted: %+v", got.DetailDispositions)
	}
}

func TestActorContextR3AGHumanOmittedUnselectedDoesNotMutateUnderlyingTruth(t *testing.T) {
	view, explanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceHuman)
	explanation.UnselectedChanges = []actorContextR3TObservedChangeProbe{{
		ChangeClass:   actorContextR3TChangeReferenceInput,
		DependencyKey: "reference:test-only:r3ag-unselected",
	}}
	before := append([]actorContextR3TObservedChangeProbe(nil), explanation.UnselectedChanges...)

	got, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.InlineUnselectedChanges) != 0 ||
		actorContextR3AGDispositionFor(got, actorContextR3AGDetailUnselectedChanges) != actorContextR3AGDispositionOmitted {
		t.Fatalf("R3AG Human unselected detail was not purpose-omitted: %+v", got)
	}
	if !reflect.DeepEqual(explanation.UnselectedChanges, before) {
		t.Fatalf("R3AG materialization mutated omitted upstream truth: before=%+v after=%+v",
			before, explanation.UnselectedChanges)
	}
}

func TestActorContextR3AGHumanDrillDownWithoutExistingRefFailsClosed(t *testing.T) {
	view, explanation := actorContextR3AGRequirementExplanation(t, actorContextR3PAudienceHuman)
	view.DrillDownRefs = nil
	if _, err := actorContextR3AGMaterialize(view, explanation); err == nil {
		t.Fatal("R3AG accepted Human drill-down detail without existing drill-down refs")
	}
}

func TestActorContextR3AGMaterializationHasNoAuthorityOrActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3AGMaterializedExplanationProbe{})
	for _, forbidden := range []string{
		"Fresh", "Stale", "Safe", "Reusable", "ShouldReevaluate",
		"RetryRequired", "Reauthorize", "ApprovalRequired", "NextAction",
		"HandoffOwner", "Authorized", "Allowed", "CanExecute",
		"Confidence", "Severity", "Priority",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AG materialization introduced forbidden shortcut %q", forbidden)
		}
	}
}

func actorContextR3AGRequirementExplanation(
	t *testing.T,
	audience string,
) (actorContextR3PBoundedStopViewProbe, actorContextR3AFExplanationProbe) {
	t.Helper()
	base, reasons, bindings := actorContextR3ADFreshnessFixture(t, audience)
	afterRoots := actorContextR3ACCloneRoots(base)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AG setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"

	deltas := actorContextR3ACClassifyRootChanges(base, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)
	view := actorContextR3AEView(t, audience)
	before := actorContextR3ABFreshnessCore(t)
	categoryResult, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	return view, actorContextR3AFBuildExplanation(
		view,
		categoryResult,
		bridge,
		actorContextR3AFFreshnessAuthority(),
	)
}

func actorContextR3AGMaterialize(
	view actorContextR3PBoundedStopViewProbe,
	explanation actorContextR3AFExplanationProbe,
) (actorContextR3AGMaterializedExplanationProbe, error) {
	if view.Audience != explanation.Audience || view.Purpose != explanation.Purpose {
		return actorContextR3AGMaterializedExplanationProbe{}, fmt.Errorf(
			"R3AG audience/purpose mismatch: view=%s/%s explanation=%s/%s",
			view.Audience, view.Purpose, explanation.Audience, explanation.Purpose,
		)
	}

	out := actorContextR3AGMaterializedExplanationProbe{
		Audience:                    view.Audience,
		Purpose:                     view.Purpose,
		FreshnessStopReason:         explanation.FreshnessStopReason,
		ResolutionNeedClasses:       actorContextR0SortedUnique(append([]string(nil), explanation.ResolutionNeedClasses...)),
		ReevaluationAuthorityStatus: explanation.ReevaluationAuthorityStatus,
		ReuseAuthorityStatus:        explanation.ReuseAuthorityStatus,
		DrillDownRefs:               actorContextR0SortedUnique(append([]string(nil), view.DrillDownRefs...)),
		AuthorityLimits: []string{
			"presentation-selection-does-not-change-underlying-truth",
			"drill-down-does-not-mean-omitted-or-resolved",
			"omitted-from-materialization-does-not-mean-absent-upstream",
			"more-inline-detail-does-not-create-stronger-authority",
			"materialization-does-not-select-response-action",
		},
	}

	for _, category := range explanation.CategoryOutcomes {
		identity := actorContextR3AGOutcomeIdentity(category)
		out.CategorySummaries = append(out.CategorySummaries, actorContextR3AGOutcomeSummaryProbe{
			Category:        category.Category,
			Status:          category.Status,
			OutcomeIdentity: identity,
		})
		out.DrillDownOutcomeIdentities = append(out.DrillDownOutcomeIdentities, identity)
	}

	switch {
	case view.Audience == actorContextR3PAudienceHuman &&
		view.Purpose == actorContextR3PPurposeOrientation:
		out.DetailDispositions = actorContextR3AGHumanDetailProfile()
		if len(explanation.CategoryOutcomes) > 0 && len(out.DrillDownRefs) == 0 {
			return actorContextR3AGMaterializedExplanationProbe{}, fmt.Errorf(
				"R3AG Human drill-down detail requires existing provenance refs",
			)
		}
	case view.Audience == actorContextR3PAudienceAgent &&
		view.Purpose == actorContextR3PPurposeResolutionInspection:
		out.DetailDispositions = actorContextR3AGAgentDetailProfile()
		out.DetailedCategoryOutcomes = actorContextR3AGNormalizeCategoryOutcomes(explanation.CategoryOutcomes)
		out.InlineUnselectedChanges = actorContextR3TUniqueChanges(explanation.UnselectedChanges)
		out.InlineResidualRootDeltas = actorContextR3AGNormalizeResiduals(explanation.ResidualRootDeltas)
	default:
		return actorContextR3AGMaterializedExplanationProbe{}, fmt.Errorf(
			"unsupported R3AG audience/purpose %q/%q", view.Audience, view.Purpose,
		)
	}

	if err := actorContextR3AGValidateDetailPartition(out.DetailDispositions); err != nil {
		return actorContextR3AGMaterializedExplanationProbe{}, err
	}
	sort.Slice(out.CategorySummaries, func(i, j int) bool {
		return out.CategorySummaries[i].OutcomeIdentity < out.CategorySummaries[j].OutcomeIdentity
	})
	out.DrillDownOutcomeIdentities = actorContextR0SortedUnique(out.DrillDownOutcomeIdentities)
	sort.Slice(out.DetailDispositions, func(i, j int) bool {
		return out.DetailDispositions[i].DetailClass < out.DetailDispositions[j].DetailClass
	})
	sort.Strings(out.AuthorityLimits)
	return out, nil
}

func actorContextR3AGHumanDetailProfile() []actorContextR3AGDetailDispositionProbe {
	return []actorContextR3AGDetailDispositionProbe{
		{actorContextR3AGDetailCategorySummary, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailCategoryFingerprints, actorContextR3AGDispositionDrillDown},
		{actorContextR3AGDetailCategoryReasons, actorContextR3AGDispositionDrillDown},
		{actorContextR3AGDetailFreshnessAuthority, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailResolutionNeeds, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailUnselectedChanges, actorContextR3AGDispositionOmitted},
		{actorContextR3AGDetailResidualRootDeltas, actorContextR3AGDispositionDrillDown},
		{actorContextR3AGDetailProvenanceDrilldown, actorContextR3AGDispositionDrillDown},
	}
}

func actorContextR3AGAgentDetailProfile() []actorContextR3AGDetailDispositionProbe {
	return []actorContextR3AGDetailDispositionProbe{
		{actorContextR3AGDetailCategorySummary, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailCategoryFingerprints, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailCategoryReasons, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailFreshnessAuthority, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailResolutionNeeds, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailUnselectedChanges, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailResidualRootDeltas, actorContextR3AGDispositionInline},
		{actorContextR3AGDetailProvenanceDrilldown, actorContextR3AGDispositionDrillDown},
	}
}

func actorContextR3AGOutcomeIdentity(outcome actorContextR3AFCategoryOutcomeProbe) string {
	reasons := actorContextR3TUniqueReasons(outcome.Reasons)
	reasonIDs := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		reasonIDs = append(reasonIDs, actorContextR3VReasonIdentity(reason))
	}
	sort.Strings(reasonIDs)
	return strings.Join([]string{
		outcome.Category,
		outcome.Status,
		outcome.BeforeFingerprint,
		outcome.AfterFingerprint,
		strings.Join(reasonIDs, "||"),
	}, "|")
}

func actorContextR3AGNormalizeCategoryOutcomes(
	values []actorContextR3AFCategoryOutcomeProbe,
) []actorContextR3AFCategoryOutcomeProbe {
	out := append([]actorContextR3AFCategoryOutcomeProbe(nil), values...)
	for i := range out {
		out[i].Reasons = actorContextR3TUniqueReasons(out[i].Reasons)
	}
	sort.Slice(out, func(i, j int) bool {
		return actorContextR3AGOutcomeIdentity(out[i]) < actorContextR3AGOutcomeIdentity(out[j])
	})
	return out
}

func actorContextR3AGNormalizeResiduals(
	values []actorContextR3ADResidualRootDeltaProbe,
) []actorContextR3ADResidualRootDeltaProbe {
	out := append([]actorContextR3ADResidualRootDeltaProbe(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Delta.RootIdentity != out[j].Delta.RootIdentity {
			return out[i].Delta.RootIdentity < out[j].Delta.RootIdentity
		}
		if out[i].Delta.RootRole != out[j].Delta.RootRole {
			return out[i].Delta.RootRole < out[j].Delta.RootRole
		}
		if out[i].Delta.ChangeClass != out[j].Delta.ChangeClass {
			return out[i].Delta.ChangeClass < out[j].Delta.ChangeClass
		}
		return out[i].Reason < out[j].Reason
	})
	return out
}

func actorContextR3AGValidateDetailPartition(values []actorContextR3AGDetailDispositionProbe) error {
	known := map[string]bool{}
	for _, detail := range actorContextR3AGAllDetailClasses {
		known[detail] = true
	}
	seen := map[string]bool{}
	for _, value := range values {
		if !known[value.DetailClass] {
			return fmt.Errorf("unknown R3AG detail class %q", value.DetailClass)
		}
		if seen[value.DetailClass] {
			return fmt.Errorf("duplicate R3AG detail class %q", value.DetailClass)
		}
		switch value.Disposition {
		case actorContextR3AGDispositionInline,
			actorContextR3AGDispositionDrillDown,
			actorContextR3AGDispositionOmitted:
		default:
			return fmt.Errorf("invalid R3AG disposition %q", value.Disposition)
		}
		seen[value.DetailClass] = true
	}
	if len(seen) != len(known) {
		return fmt.Errorf("incomplete R3AG detail partition: got=%d want=%d", len(seen), len(known))
	}
	return nil
}

func actorContextR3AGDispositionFor(
	materialized actorContextR3AGMaterializedExplanationProbe,
	detailClass string,
) string {
	for _, value := range materialized.DetailDispositions {
		if value.DetailClass == detailClass {
			return value.Disposition
		}
	}
	return ""
}

func actorContextR3AGSummary(
	materialized actorContextR3AGMaterializedExplanationProbe,
	category string,
) *actorContextR3AGOutcomeSummaryProbe {
	for i := range materialized.CategorySummaries {
		if materialized.CategorySummaries[i].Category == category {
			return &materialized.CategorySummaries[i]
		}
	}
	return nil
}
