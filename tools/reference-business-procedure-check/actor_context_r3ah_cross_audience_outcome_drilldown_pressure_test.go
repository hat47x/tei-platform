package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type actorContextR3AHReasonTraceProbe struct {
	ReasonIdentity   string
	DependencyKey    string
	DependencyReason string
	RootAddress      string
	RootIdentity     string
	RootRole         string
	RootClaimKind    string
	CurrentLocator   string
}

type actorContextR3AHOutcomeTraceProbe struct {
	Category               string
	Status                 string
	SummaryOutcomeIdentity string
	BeforeFingerprint      string
	AfterFingerprint       string
	Reasons                []actorContextR3AHReasonTraceProbe
}

type actorContextR3AHResidualTraceProbe struct {
	ResidualReason string
	RootAddress    string
	RootIdentity   string
	RootRole       string
	RootClaimKind  string
	CurrentLocator string
}

type actorContextR3AHTraceEnvelopeProbe struct {
	Outcome         actorContextR3AHOutcomeTraceProbe
	Residuals       []actorContextR3AHResidualTraceProbe
	AuthorityLimits []string
}

func TestActorContextR3AHHumanSummaryDrillsToExactReasonDependencyRootAndLocator(t *testing.T) {
	materialized, explanation, roots, bindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceHuman,
	)

	got, err := actorContextR3AHDrillDownOutcome(
		materialized,
		explanation,
		roots,
		bindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome.Category != actorContextR3PCategoryMachineReasons ||
		got.Outcome.Status != actorContextR3WStatusUnchanged ||
		got.Outcome.SummaryOutcomeIdentity == "" ||
		len(got.Outcome.Reasons) == 0 {
		t.Fatalf("R3AH lost machine-reasons drill-down chain: %+v", got)
	}
	for _, reason := range got.Outcome.Reasons {
		if reason.ReasonIdentity == "" ||
			reason.DependencyKey == "" ||
			reason.RootAddress == "" ||
			reason.RootIdentity == "" ||
			reason.RootRole == "" ||
			reason.CurrentLocator == "" {
			t.Fatalf("R3AH produced incomplete exact reason provenance trace: %+v", reason)
		}
	}
}

func TestActorContextR3AHHumanAndAgentSharedTraceAreIdentical(t *testing.T) {
	humanMaterialized, humanExplanation, humanRoots, humanBindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceHuman,
	)
	agentMaterialized, agentExplanation, agentRoots, agentBindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceAgent,
	)

	human, err := actorContextR3AHDrillDownOutcome(
		humanMaterialized, humanExplanation, humanRoots, humanBindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := actorContextR3AHDrillDownOutcome(
		agentMaterialized, agentExplanation, agentRoots, agentBindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(human.Outcome, agent.Outcome) {
		t.Fatalf("R3AH audience presentation changed shared provenance chain: human=%+v agent=%+v",
			human.Outcome, agent.Outcome)
	}
}

func TestActorContextR3AHAgentOnlyKnownAuthorityFactsDoesNotLeakToHuman(t *testing.T) {
	humanMaterialized, humanExplanation, humanRoots, humanBindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceHuman,
	)
	agentMaterialized, agentExplanation, agentRoots, agentBindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceAgent,
	)

	if _, err := actorContextR3AHDrillDownOutcome(
		humanMaterialized, humanExplanation, humanRoots, humanBindings,
		actorContextR3PCategoryKnownAuthorityFacts,
	); err == nil {
		t.Fatal("R3AH fabricated Human trace for Agent-only known-authority-facts")
	}

	got, err := actorContextR3AHDrillDownOutcome(
		agentMaterialized, agentExplanation, agentRoots, agentBindings,
		actorContextR3PCategoryKnownAuthorityFacts,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome.Category != actorContextR3PCategoryKnownAuthorityFacts ||
		len(got.Outcome.Reasons) == 0 {
		t.Fatalf("R3AH Agent lost selected known-authority-facts trace: %+v", got)
	}
}

func TestActorContextR3AHTamperedSummaryOutcomeIdentityFailsClosed(t *testing.T) {
	materialized, explanation, roots, bindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceHuman,
	)
	for i := range materialized.CategorySummaries {
		if materialized.CategorySummaries[i].Category == actorContextR3PCategoryMachineReasons {
			materialized.CategorySummaries[i].OutcomeIdentity = "tampered-outcome-identity"
		}
	}
	if _, err := actorContextR3AHDrillDownOutcome(
		materialized, explanation, roots, bindings,
		actorContextR3PCategoryMachineReasons,
	); err == nil {
		t.Fatal("R3AH accepted tampered summary outcome identity")
	}
}

func TestActorContextR3AHMissingExplicitBindingFailsClosed(t *testing.T) {
	materialized, explanation, roots, _ := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceHuman,
	)
	if _, err := actorContextR3AHDrillDownOutcome(
		materialized, explanation, roots, nil,
		actorContextR3PCategoryMachineReasons,
	); err == nil {
		t.Fatal("R3AH guessed provenance root without explicit R3AD binding")
	}
}

func TestActorContextR3AHMissingBoundRootFailsClosed(t *testing.T) {
	materialized, explanation, _, bindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceHuman,
	)
	if _, err := actorContextR3AHDrillDownOutcome(
		materialized, explanation, nil, bindings,
		actorContextR3PCategoryMachineReasons,
	); err == nil {
		t.Fatal("R3AH accepted dependency binding whose current root was missing")
	}
}

func TestActorContextR3AHPriorObservationResidualUsesSeparateRootTraceWithoutDependency(t *testing.T) {
	materialized, explanation, roots := actorContextR3AHPriorObservationFixture(t)

	got, err := actorContextR3AHDrillDownResiduals(materialized, explanation, roots)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Residuals) != 1 {
		t.Fatalf("R3AH lost prior Observation residual trace: %+v", got)
	}
	residual := got.Residuals[0]
	if residual.ResidualReason != actorContextR3ADResidualNoBinding ||
		residual.RootRole != actorContextR3ABRoleRuntimeObservation ||
		residual.RootAddress == "" ||
		residual.RootIdentity == "" ||
		residual.CurrentLocator == "" {
		t.Fatalf("R3AH prior Observation residual trace incomplete: %+v", residual)
	}
}

func TestActorContextR3AHTraceIsStableAcrossBindingAndRootInputOrder(t *testing.T) {
	materialized, explanation, roots, bindings := actorContextR3AHRequirementFixture(
		t,
		actorContextR3PAudienceAgent,
	)
	first, err := actorContextR3AHDrillDownOutcome(
		materialized, explanation, roots, bindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}

	reversedRoots := append([]actorContextR3ABProvenanceRootProbe(nil), roots...)
	for i, j := 0, len(reversedRoots)-1; i < j; i, j = i+1, j-1 {
		reversedRoots[i], reversedRoots[j] = reversedRoots[j], reversedRoots[i]
	}
	reversedBindings := append([]actorContextR3ADRootDependencyBindingProbe(nil), bindings...)
	for i, j := 0, len(reversedBindings)-1; i < j; i, j = i+1, j-1 {
		reversedBindings[i], reversedBindings[j] = reversedBindings[j], reversedBindings[i]
	}
	second, err := actorContextR3AHDrillDownOutcome(
		materialized, explanation, reversedRoots, reversedBindings,
		actorContextR3PCategoryMachineReasons,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AH input order changed normalized trace identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AHTraceShapeHasNoAuthorityOrRemediationShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AHTraceEnvelopeProbe{}),
		reflect.TypeOf(actorContextR3AHOutcomeTraceProbe{}),
		reflect.TypeOf(actorContextR3AHReasonTraceProbe{}),
		reflect.TypeOf(actorContextR3AHResidualTraceProbe{}),
	} {
		for _, forbidden := range []string{
			"Fresh", "Stale", "Safe", "Reusable", "ShouldReevaluate",
			"Retry", "Reauthorize", "Rebuild", "ApprovalRequest",
			"NextAction", "HandoffOwner", "Authorized", "Allowed",
			"CanExecute", "Confidence", "Priority", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AH trace introduced forbidden authority/remediation shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AHRequirementFixture(
	t *testing.T,
	audience string,
) (
	actorContextR3AGMaterializedExplanationProbe,
	actorContextR3AFExplanationProbe,
	[]actorContextR3ABProvenanceRootProbe,
	[]actorContextR3ADRootDependencyBindingProbe,
) {
	t.Helper()

	roots, reasons, bindings := actorContextR3ADFreshnessFixture(t, audience)
	afterRoots := actorContextR3ACCloneRoots(roots)
	idx := actorContextR3ADRootIndexByClaim(afterRoots, "authorization-requirement-source")
	if idx < 0 {
		t.Fatal("R3AH setup lost Authorization requirement root")
	}
	afterRoots[idx].Fingerprint += ":changed"
	deltas := actorContextR3ACClassifyRootChanges(roots, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	view := actorContextR3AEView(t, audience)
	before := actorContextR3ABFreshnessCore(t)
	result, err := actorContextR3AEEvaluate(
		before,
		actorContextR3QCloneCore(before),
		view,
		bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	explanation := actorContextR3AFBuildExplanation(
		view, result, bridge, actorContextR3AFFreshnessAuthority(),
	)
	materialized, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}
	return materialized, explanation, roots, bindings
}

func actorContextR3AHPriorObservationFixture(
	t *testing.T,
) (
	actorContextR3AGMaterializedExplanationProbe,
	actorContextR3AFExplanationProbe,
	[]actorContextR3ABProvenanceRootProbe,
) {
	t.Helper()

	roots, reasons, bindings := actorContextR3ADFreshnessFixture(t, actorContextR3PAudienceAgent)
	afterRoots := actorContextR3ACCloneRoots(roots)
	idx := actorContextR3ACRootIndexByRole(afterRoots, actorContextR3ABRoleRuntimeObservation)
	if idx < 0 {
		t.Fatal("R3AH setup lost prior Authorization Observation root")
	}
	afterRoots[idx].Fingerprint += ":denied"
	afterRoots[idx].ClaimKind = "prior-authorization-observation:denied"
	deltas := actorContextR3ACClassifyRootChanges(roots, afterRoots)
	bridge := actorContextR3ADExplainBoundRootChanges(reasons, deltas.Changes, bindings)

	view := actorContextR3AEView(t, actorContextR3PAudienceAgent)
	core := actorContextR3ABFreshnessCore(t)
	result, err := actorContextR3AEEvaluate(
		core, actorContextR3QCloneCore(core), view, bridge,
	)
	if err != nil {
		t.Fatal(err)
	}
	explanation := actorContextR3AFBuildExplanation(
		view, result, bridge, actorContextR3AFFreshnessAuthority(),
	)
	materialized, err := actorContextR3AGMaterialize(view, explanation)
	if err != nil {
		t.Fatal(err)
	}
	return materialized, explanation, roots
}

func actorContextR3AHDrillDownOutcome(
	materialized actorContextR3AGMaterializedExplanationProbe,
	explanation actorContextR3AFExplanationProbe,
	roots []actorContextR3ABProvenanceRootProbe,
	bindings []actorContextR3ADRootDependencyBindingProbe,
	category string,
) (actorContextR3AHTraceEnvelopeProbe, error) {
	summary := actorContextR3AGSummary(materialized, category)
	if summary == nil {
		return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
			"R3AH category %q is not materialized for %s/%s",
			category, materialized.Audience, materialized.Purpose,
		)
	}

	var exact *actorContextR3AFCategoryOutcomeProbe
	for i := range explanation.CategoryOutcomes {
		if explanation.CategoryOutcomes[i].Category == category {
			exact = &explanation.CategoryOutcomes[i]
			break
		}
	}
	if exact == nil {
		return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
			"R3AH summary category %q has no exact R3AF outcome",
			category,
		)
	}
	identity := actorContextR3AGOutcomeIdentity(*exact)
	if summary.OutcomeIdentity != identity {
		return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
			"R3AH summary outcome identity mismatch for %q",
			category,
		)
	}

	rootByAddress := map[string]actorContextR3ABProvenanceRootProbe{}
	for _, root := range roots {
		rootByAddress[actorContextR3ACRootAddress(root)] = root
	}
	addressesByDependency := map[string][]string{}
	for _, binding := range bindings {
		if binding.DependencyKey == "" || binding.RootAddress == "" {
			continue
		}
		addressesByDependency[binding.DependencyKey] = append(
			addressesByDependency[binding.DependencyKey],
			binding.RootAddress,
		)
	}

	reasonTraces := []actorContextR3AHReasonTraceProbe{}
	for _, reason := range actorContextR3TUniqueReasons(exact.Reasons) {
		if reason.DependencyKey == "" {
			return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
				"R3AH exact reason %q has no dependency key",
				actorContextR3VReasonIdentity(reason),
			)
		}
		addresses := actorContextR0SortedUnique(addressesByDependency[reason.DependencyKey])
		if len(addresses) == 0 {
			return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
				"R3AH dependency %q has no explicit provenance root binding",
				reason.DependencyKey,
			)
		}
		for _, address := range addresses {
			root, ok := rootByAddress[address]
			if !ok {
				return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
					"R3AH bound provenance root %q is not current",
					address,
				)
			}
			if root.Locator == "" {
				return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
					"R3AH bound provenance root %q has no current locator",
					address,
				)
			}
			reasonTraces = append(reasonTraces, actorContextR3AHReasonTraceProbe{
				ReasonIdentity:   actorContextR3VReasonIdentity(reason),
				DependencyKey:    reason.DependencyKey,
				DependencyReason: reason.DependencyReason,
				RootAddress:      address,
				RootIdentity:     root.Identity,
				RootRole:         root.Role,
				RootClaimKind:    root.ClaimKind,
				CurrentLocator:   root.Locator,
			})
		}
	}
	sort.Slice(reasonTraces, func(i, j int) bool {
		if reasonTraces[i].ReasonIdentity != reasonTraces[j].ReasonIdentity {
			return reasonTraces[i].ReasonIdentity < reasonTraces[j].ReasonIdentity
		}
		if reasonTraces[i].DependencyKey != reasonTraces[j].DependencyKey {
			return reasonTraces[i].DependencyKey < reasonTraces[j].DependencyKey
		}
		return reasonTraces[i].RootAddress < reasonTraces[j].RootAddress
	})

	return actorContextR3AHTraceEnvelopeProbe{
		Outcome: actorContextR3AHOutcomeTraceProbe{
			Category:               exact.Category,
			Status:                 exact.Status,
			SummaryOutcomeIdentity: identity,
			BeforeFingerprint:      exact.BeforeFingerprint,
			AfterFingerprint:       exact.AfterFingerprint,
			Reasons:                reasonTraces,
		},
		AuthorityLimits: []string{
			"summary-handle-is-not-authority",
			"reason-dependency-is-not-authority-edge",
			"dependency-root-link-requires-explicit-r3ad-binding",
			"current-locator-is-not-root-identity",
			"traceability-does-not-select-remediation-or-execution",
		},
	}, nil
}

func actorContextR3AHDrillDownResiduals(
	materialized actorContextR3AGMaterializedExplanationProbe,
	explanation actorContextR3AFExplanationProbe,
	roots []actorContextR3ABProvenanceRootProbe,
) (actorContextR3AHTraceEnvelopeProbe, error) {
	if materialized.Audience != explanation.Audience ||
		materialized.Purpose != explanation.Purpose {
		return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
			"R3AH residual materialization/explanation profile mismatch",
		)
	}

	rootByAddress := map[string]actorContextR3ABProvenanceRootProbe{}
	for _, root := range roots {
		rootByAddress[actorContextR3ACRootAddress(root)] = root
	}
	out := actorContextR3AHTraceEnvelopeProbe{
		AuthorityLimits: []string{
			"residual-root-trace-does-not-fabricate-projection-dependency",
			"residual-provenance-does-not-select-remediation",
		},
	}
	for _, residual := range explanation.ResidualRootDeltas {
		root, ok := rootByAddress[residual.Delta.RootIdentity]
		if !ok {
			return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
				"R3AH residual root %q is not current",
				residual.Delta.RootIdentity,
			)
		}
		if root.Locator == "" {
			return actorContextR3AHTraceEnvelopeProbe{}, fmt.Errorf(
				"R3AH residual root %q has no current locator",
				residual.Delta.RootIdentity,
			)
		}
		out.Residuals = append(out.Residuals, actorContextR3AHResidualTraceProbe{
			ResidualReason: residual.Reason,
			RootAddress:    residual.Delta.RootIdentity,
			RootIdentity:   root.Identity,
			RootRole:       root.Role,
			RootClaimKind:  root.ClaimKind,
			CurrentLocator: root.Locator,
		})
	}
	sort.Slice(out.Residuals, func(i, j int) bool {
		if out.Residuals[i].RootAddress != out.Residuals[j].RootAddress {
			return out.Residuals[i].RootAddress < out.Residuals[j].RootAddress
		}
		return out.Residuals[i].ResidualReason < out.Residuals[j].ResidualReason
	})
	sort.Strings(out.AuthorityLimits)
	return out, nil
}
