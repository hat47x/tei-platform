package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR6HKnowledgeRevisionWitnessProbe struct {
	KnowledgeKey              string                           `json:"knowledgeKey"`
	KnowledgeKind             string                           `json:"knowledgeKind"`
	MaterializationDisposition string                          `json:"materializationDisposition"`
	MaterializationReason     string                           `json:"materializationReason"`
	DependencyKeys            []string                         `json:"dependencyKeys,omitempty"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	SourceRefs                []string                         `json:"sourceRefs"`
}

type actorContextR6HRevisionProvenanceProbe struct {
	Audience        string                                      `json:"audience"`
	Purpose         string                                      `json:"purpose"`
	Knowledge       []actorContextR6HKnowledgeRevisionWitnessProbe `json:"knowledge"`
	AuthorityLimits []string                                    `json:"authorityLimits"`
}

func TestActorContextR6HOmittedKnowledgeRevisionDoesNotChangeMaterializationDisposition(t *testing.T) {
	root := repoPath()
	routing := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	origins := actorContextR6EBP06HumanOriginSnapshot(t, root, routing)

	before, err := actorContextR6HBuildRevisionProvenance(
		routing.Qualified,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	revised := actorContextR6HCloneQualified(routing.Qualified)
	actorContextR6HReplaceFirstFingerprint(
		t,
		revised,
		"semantic-capability|security.authorize",
		"test-only-capability-content-revision",
	)
	after, err := actorContextR6HBuildRevisionProvenance(
		revised,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	beforeCapability := actorContextR6HKnowledgeByKey(
		t,
		before.Knowledge,
		"semantic-capability|security.authorize",
	)
	afterCapability := actorContextR6HKnowledgeByKey(
		t,
		after.Knowledge,
		"semantic-capability|security.authorize",
	)

	if beforeCapability.MaterializationDisposition != actorContextR6GDispositionNotMaterialized ||
		afterCapability.MaterializationDisposition != actorContextR6GDispositionNotMaterialized ||
		beforeCapability.MaterializationReason != actorContextR6GReasonNoRoutedDependency ||
		afterCapability.MaterializationReason != actorContextR6GReasonNoRoutedDependency {
		t.Fatalf(
			"R6H source revision changed omitted Capability materialization semantics: before=%+v after=%+v",
			beforeCapability,
			afterCapability,
		)
	}
	if reflect.DeepEqual(
		beforeCapability.RequiredInputFingerprints,
		afterCapability.RequiredInputFingerprints,
	) {
		t.Fatalf(
			"R6H failed to expose omitted Capability revision provenance: before=%+v after=%+v",
			beforeCapability.RequiredInputFingerprints,
			afterCapability.RequiredInputFingerprints,
		)
	}
	if !reflect.DeepEqual(beforeCapability.SourceRefs, afterCapability.SourceRefs) {
		t.Fatalf(
			"R6H content revision unexpectedly changed source locator identity: before=%v after=%v",
			beforeCapability.SourceRefs,
			afterCapability.SourceRefs,
		)
	}
	if len(afterCapability.DependencyKeys) != 0 {
		t.Fatalf(
			"R6H omitted Capability revision invented a projection dependency: %+v",
			afterCapability,
		)
	}
}

func TestActorContextR6HMaterializedKnowledgeRevisionDoesNotDecideFreshnessOrDematerialize(t *testing.T) {
	root := repoPath()
	routing := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	origins := actorContextR6EBP06HumanOriginSnapshot(t, root, routing)

	before, err := actorContextR6HBuildRevisionProvenance(
		routing.Qualified,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	revised := actorContextR6HCloneQualified(routing.Qualified)
	actorContextR6HReplaceFirstFingerprint(
		t,
		revised,
		"business-procedure-reference|order-entry.order-handling",
		"test-only-procedure-content-revision",
	)
	after, err := actorContextR6HBuildRevisionProvenance(
		revised,
		origins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	beforeProcedure := actorContextR6HKnowledgeByKey(
		t,
		before.Knowledge,
		"business-procedure-reference|order-entry.order-handling",
	)
	afterProcedure := actorContextR6HKnowledgeByKey(
		t,
		after.Knowledge,
		"business-procedure-reference|order-entry.order-handling",
	)

	if beforeProcedure.MaterializationDisposition != actorContextR6GDispositionMaterialized ||
		afterProcedure.MaterializationDisposition != actorContextR6GDispositionMaterialized ||
		beforeProcedure.MaterializationReason != actorContextR6GReasonConsumedRoutedDependency ||
		afterProcedure.MaterializationReason != actorContextR6GReasonConsumedRoutedDependency {
		t.Fatalf(
			"R6H source revision changed materialized Procedure disposition: before=%+v after=%+v",
			beforeProcedure,
			afterProcedure,
		)
	}
	if reflect.DeepEqual(
		beforeProcedure.RequiredInputFingerprints,
		afterProcedure.RequiredInputFingerprints,
	) {
		t.Fatalf(
			"R6H failed to expose materialized Procedure revision provenance: before=%+v after=%+v",
			beforeProcedure.RequiredInputFingerprints,
			afterProcedure.RequiredInputFingerprints,
		)
	}
	if !reflect.DeepEqual(beforeProcedure.DependencyKeys, afterProcedure.DependencyKeys) ||
		len(afterProcedure.DependencyKeys) == 0 {
		t.Fatalf(
			"R6H source revision changed current dependency consumption identity: before=%v after=%v",
			beforeProcedure.DependencyKeys,
			afterProcedure.DependencyKeys,
		)
	}
}

func TestActorContextR6HRouterSubstitutionPreservesCurrentRevisionProvenance(t *testing.T) {
	root := repoPath()
	exact := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"exact-task-router",
		actorContextR6BSelectTaskHints,
	)
	broad := actorContextR6EBP06RoutingProbe(
		t,
		root,
		"broad-heterogeneous-router",
		actorContextR6BSelectAll,
	)
	exactOrigins := actorContextR6EBP06HumanOriginSnapshot(t, root, exact)
	broadOrigins := actorContextR6EBP06HumanOriginSnapshot(t, root, broad)

	exactRevision, err := actorContextR6HBuildRevisionProvenance(
		exact.Qualified,
		exactOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}
	broadRevision, err := actorContextR6HBuildRevisionProvenance(
		broad.Qualified,
		broadOrigins.Dependencies,
		actorContextR3PAudienceHuman,
		actorContextR3PPurposeOrientation,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(exactRevision, broadRevision) {
		t.Fatalf(
			"R6H router substitution changed current materialization revision provenance: exact=%+v broad=%+v",
			exactRevision,
			broadRevision,
		)
	}
}

func TestActorContextR6HRevisionProvenanceHasNoFreshnessOrResponseAuthority(t *testing.T) {
	typ := reflect.TypeOf(actorContextR6HKnowledgeRevisionWitnessProbe{})
	for _, forbidden := range []string{
		"Fresh",
		"Stale",
		"Current",
		"Changed",
		"ChangeClass",
		"ShouldMaterialize",
		"ShouldReevaluate",
		"AutoRefresh",
		"Rebuild",
		"Authorized",
		"CanExecute",
		"NextAction",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf(
				"R6H revision provenance introduced forbidden freshness/response shortcut %q",
				forbidden,
			)
		}
	}
}

func actorContextR6HBuildRevisionProvenance(
	qualified []actorContextR6BHeterogeneousQualified,
	dependencies []actorContextR6DDependencyOriginProbe,
	audience,
	purpose string,
) (actorContextR6HRevisionProvenanceProbe, error) {
	explanation, err := actorContextR6GBuildExplanation(
		qualified,
		dependencies,
		audience,
		purpose,
	)
	if err != nil {
		return actorContextR6HRevisionProvenanceProbe{}, err
	}

	qualifiedByKey := map[string]actorContextR6BHeterogeneousQualified{}
	for _, item := range qualified {
		qualifiedByKey[actorContextR6BQualifiedKey(item)] = item
	}

	out := make([]actorContextR6HKnowledgeRevisionWitnessProbe, 0, len(explanation.Witnesses))
	for _, witness := range explanation.Witnesses {
		item, ok := qualifiedByKey[witness.KnowledgeKey]
		if !ok {
			return actorContextR6HRevisionProvenanceProbe{}, errProcedureMeaning(
				"R6H explanation references non-qualified knowledge " + witness.KnowledgeKey,
			)
		}
		out = append(out, actorContextR6HKnowledgeRevisionWitnessProbe{
			KnowledgeKey:               witness.KnowledgeKey,
			KnowledgeKind:              witness.KnowledgeKind,
			MaterializationDisposition: witness.Disposition,
			MaterializationReason:      witness.ReasonCode,
			DependencyKeys:             append([]string(nil), witness.DependencyKeys...),
			RequiredInputFingerprints:  actorContextR6HCloneFingerprints(item.RequiredInputFingerprints),
			SourceRefs:                 append([]string(nil), witness.SourceRefs...),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].KnowledgeKey < out[j].KnowledgeKey
	})

	return actorContextR6HRevisionProvenanceProbe{
		Audience:  audience,
		Purpose:   purpose,
		Knowledge: out,
		AuthorityLimits: []string{
			"revision-provenance-does-not-decide-materialization",
			"revision-provenance-does-not-decide-freshness",
			"source-content-revision-does-not-invent-dependency-consumption",
			"router-substitution-does-not-change-current-revision-provenance",
			"r3-remains-owner-of-freshness-and-reevaluation-semantics",
		},
	}, nil
}

func actorContextR6HCloneQualified(
	values []actorContextR6BHeterogeneousQualified,
) []actorContextR6BHeterogeneousQualified {
	out := make([]actorContextR6BHeterogeneousQualified, 0, len(values))
	for _, value := range values {
		copyValue := value
		copyValue.RequiredInputFingerprints = actorContextR6HCloneFingerprints(
			value.RequiredInputFingerprints,
		)
		copyValue.SourceRefs = append([]string(nil), value.SourceRefs...)
		out = append(out, copyValue)
	}
	return out
}

func actorContextR6HCloneFingerprints(
	values []actorContextR0InputFingerprint,
) []actorContextR0InputFingerprint {
	return append([]actorContextR0InputFingerprint(nil), values...)
}

func actorContextR6HReplaceFirstFingerprint(
	t *testing.T,
	values []actorContextR6BHeterogeneousQualified,
	knowledgeKey,
	replacement string,
) {
	t.Helper()
	for i := range values {
		if actorContextR6BQualifiedKey(values[i]) != knowledgeKey {
			continue
		}
		if len(values[i].RequiredInputFingerprints) == 0 {
			t.Fatalf("R6H knowledge has no input fingerprint to revise: %+v", values[i])
		}
		values[i].RequiredInputFingerprints[0].Fingerprint = replacement
		return
	}
	t.Fatalf("R6H knowledge not found for revision: %s", knowledgeKey)
}

func actorContextR6HKnowledgeByKey(
	t *testing.T,
	values []actorContextR6HKnowledgeRevisionWitnessProbe,
	knowledgeKey string,
) actorContextR6HKnowledgeRevisionWitnessProbe {
	t.Helper()
	for _, value := range values {
		if value.KnowledgeKey == knowledgeKey {
			return value
		}
	}
	t.Fatalf("R6H revision witness not found for %q: %+v", knowledgeKey, values)
	return actorContextR6HKnowledgeRevisionWitnessProbe{}
}
