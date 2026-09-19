package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type actorContextR3AMAuthorityAssertionWitnessProbe struct {
	EvidenceRef        string
	AssertionClass     string
	AssertedValue      string
	AssertionSourceRef string
}

type actorContextR3AMQualifiedAuthorityAssertionProbe struct {
	RequestIdentity         string
	RequestRevisionIdentity string
	ResponseRef             string
	EvidenceRef             string
	QualificationStatus     string
	AssertionClass          string
	AssertedValue           string
	AssertionSourceRef      string
	AssertionIdentity       string
	AuthorityLimits         []string
}

func TestActorContextR3AMQualifiedEvidenceMaterializesAssertionWithoutAcceptance(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/evidence-reevaluation",
	})
	got, err := actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{{
			EvidenceRef:        "test-only:r3am/evidence-reevaluation",
			AssertionClass:     "reevaluation-authority",
			AssertedValue:      actorContextR3YReevaluationRequired,
			AssertionSourceRef: "test-only:r3am/assertion-source",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AM assertion count=%d want=1: %+v", len(got), got)
	}
	if got[0].QualificationStatus != actorContextR3ALQualificationExplicit ||
		got[0].AssertionClass != "reevaluation-authority" ||
		got[0].AssertedValue != actorContextR3YReevaluationRequired ||
		got[0].AssertionIdentity == "" {
		t.Fatalf("R3AM lost qualified assertion identity/provenance: %+v", got[0])
	}
}

func TestActorContextR3AMUnresolvedEvidenceCannotMaterializeQualifiedAssertion(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(
		t,
		[]string{"authorization-evaluation-freshness-authority"},
		[]string{"test-only:r3am/unresolved-evidence"},
	)
	assessments, err := actorContextR3ALAssessEvidence(record, result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{{
			EvidenceRef:        "test-only:r3am/unresolved-evidence",
			AssertionClass:     "reevaluation-authority",
			AssertedValue:      actorContextR3YReevaluationRequired,
			AssertionSourceRef: "test-only:r3am/assertion-source",
		}},
	); err == nil {
		t.Fatal("R3AM accepted assertion from qualification-unresolved evidence")
	}
}

func TestActorContextR3AMUnknownEvidenceOrMissingAssertionProvenanceFailsClosed(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/current-evidence",
	})
	cases := map[string]actorContextR3AMAuthorityAssertionWitnessProbe{
		"unknown-evidence": {
			EvidenceRef:        "test-only:r3am/not-current",
			AssertionClass:     "reevaluation-authority",
			AssertedValue:      actorContextR3YReevaluationRequired,
			AssertionSourceRef: "test-only:r3am/source",
		},
		"missing-source": {
			EvidenceRef:    "test-only:r3am/current-evidence",
			AssertionClass: "reevaluation-authority",
			AssertedValue:  actorContextR3YReevaluationRequired,
		},
		"missing-class": {
			EvidenceRef:        "test-only:r3am/current-evidence",
			AssertedValue:      actorContextR3YReevaluationRequired,
			AssertionSourceRef: "test-only:r3am/source",
		},
		"missing-value": {
			EvidenceRef:        "test-only:r3am/current-evidence",
			AssertionClass:     "reevaluation-authority",
			AssertionSourceRef: "test-only:r3am/source",
		},
	}
	for name, witness := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := actorContextR3AMMaterializeAssertions(
				assessments,
				[]actorContextR3AMAuthorityAssertionWitnessProbe{witness},
			); err == nil {
				t.Fatalf("R3AM accepted incomplete/unknown assertion witness %q", name)
			}
		})
	}
}

func TestActorContextR3AMR3YVocabularyMatchDoesNotWriteCurrentAuthorityStatus(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/evidence-reevaluation",
		"test-only:r3am/evidence-reuse",
	})
	got, err := actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{
			{
				EvidenceRef:        "test-only:r3am/evidence-reevaluation",
				AssertionClass:     "reevaluation-authority",
				AssertedValue:      actorContextR3YReevaluationRequired,
				AssertionSourceRef: "test-only:r3am/source-reevaluation",
			},
			{
				EvidenceRef:        "test-only:r3am/evidence-reuse",
				AssertionClass:     "prior-observation-reuse-authority",
				AssertedValue:      actorContextR3YReusePermitted,
				AssertionSourceRef: "test-only:r3am/source-reuse",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("R3AM lost R3Y-vocabulary assertions: %+v", got)
	}
	if actorContextR3AMFindAssertion(got, "reevaluation-authority", actorContextR3YReevaluationRequired) == nil ||
		actorContextR3AMFindAssertion(got, "prior-observation-reuse-authority", actorContextR3YReusePermitted) == nil {
		t.Fatalf("R3AM did not preserve asserted values as assertions: %+v", got)
	}
}

func TestActorContextR3AMConflictingQualifiedAssertionsRemainIndependentWithoutWinner(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/evidence-a",
		"test-only:r3am/evidence-b",
	})
	got, err := actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{
			{
				EvidenceRef:        "test-only:r3am/evidence-a",
				AssertionClass:     "reevaluation-authority",
				AssertedValue:      actorContextR3YReevaluationRequired,
				AssertionSourceRef: "test-only:r3am/source-a",
			},
			{
				EvidenceRef:        "test-only:r3am/evidence-b",
				AssertionClass:     "reevaluation-authority",
				AssertedValue:      "test-only:not-required",
				AssertionSourceRef: "test-only:r3am/source-b",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("R3AM collapsed conflicting assertions: %+v", got)
	}
	if got[0].AssertionIdentity == got[1].AssertionIdentity {
		t.Fatalf("R3AM conflicting assertions shared identity: %+v", got)
	}
}

func TestActorContextR3AMExactDuplicateAssertionNormalizesButDistinctProvenanceDoesNot(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/evidence",
	})
	base := actorContextR3AMAuthorityAssertionWitnessProbe{
		EvidenceRef:        "test-only:r3am/evidence",
		AssertionClass:     "reevaluation-authority",
		AssertedValue:      actorContextR3YReevaluationRequired,
		AssertionSourceRef: "test-only:r3am/source-a",
	}
	got, err := actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AM did not normalize exact duplicate assertion: %+v", got)
	}

	distinct := base
	distinct.AssertionSourceRef = "test-only:r3am/source-b"
	got, err = actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{base, distinct},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("R3AM collapsed distinct assertion provenance: %+v", got)
	}
}

func TestActorContextR3AMQualifiedEvidenceWithoutAssertionWitnessStaysEmpty(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/evidence",
	})
	got, err := actorContextR3AMMaterializeAssertions(assessments, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AM fabricated assertion from qualified evidence alone: %+v", got)
	}
}

func TestActorContextR3AMRepeatedAndInputOrderMaterializationAreStable(t *testing.T) {
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3am/evidence-a",
		"test-only:r3am/evidence-b",
	})
	witnesses := []actorContextR3AMAuthorityAssertionWitnessProbe{
		{
			EvidenceRef:        "test-only:r3am/evidence-b",
			AssertionClass:     "prior-observation-reuse-authority",
			AssertedValue:      actorContextR3YReusePermitted,
			AssertionSourceRef: "test-only:r3am/source-b",
		},
		{
			EvidenceRef:        "test-only:r3am/evidence-a",
			AssertionClass:     "reevaluation-authority",
			AssertedValue:      actorContextR3YReevaluationRequired,
			AssertionSourceRef: "test-only:r3am/source-a",
		},
	}
	first, err := actorContextR3AMMaterializeAssertions(assessments, witnesses)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AMMaterializeAssertions(assessments, witnesses)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AM repeated materialization changed identity: first=%+v second=%+v", first, second)
	}

	reversedAssessments := append([]actorContextR3ALEvidenceAssessmentProbe(nil), assessments...)
	for i, j := 0, len(reversedAssessments)-1; i < j; i, j = i+1, j-1 {
		reversedAssessments[i], reversedAssessments[j] = reversedAssessments[j], reversedAssessments[i]
	}
	reversedWitnesses := append([]actorContextR3AMAuthorityAssertionWitnessProbe(nil), witnesses...)
	for i, j := 0, len(reversedWitnesses)-1; i < j; i, j = i+1, j-1 {
		reversedWitnesses[i], reversedWitnesses[j] = reversedWitnesses[j], reversedWitnesses[i]
	}
	normalized, err := actorContextR3AMMaterializeAssertions(reversedAssessments, reversedWitnesses)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, normalized) {
		t.Fatalf("R3AM input order changed normalized assertions: first=%+v normalized=%+v", first, normalized)
	}
}

func TestActorContextR3AMAssertionShapeHasNoAcceptanceOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AMAuthorityAssertionWitnessProbe{}),
		reflect.TypeOf(actorContextR3AMQualifiedAuthorityAssertionProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedAuthorityValue",
			"AcceptedDecision",
			"AcceptanceStatus",
			"Approved",
			"Rejected",
			"Sufficient",
			"Resolved",
			"ResolutionComplete",
			"ReevaluationAuthorityStatus",
			"ReuseAuthorityStatus",
			"CurrentAuthority",
			"Apply",
			"AutoApply",
			"ToolCall",
			"Command",
			"NextAction",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Retry",
			"Reauthorize",
			"Confidence",
			"Priority",
			"Severity",
			"Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AM assertion shape introduced forbidden acceptance/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AMQualifiedAssessments(
	t *testing.T,
	evidenceRefs []string,
) []actorContextR3ALEvidenceAssessmentProbe {
	t.Helper()

	record, result := actorContextR3ALCurrentCorrelation(
		t,
		[]string{"authorization-evaluation-freshness-authority"},
		evidenceRefs,
	)
	witnesses := make([]actorContextR3ALEvidenceQualificationWitnessProbe, 0, len(evidenceRefs))
	for _, evidenceRef := range evidenceRefs {
		witnesses = append(witnesses, actorContextR3ALEvidenceQualificationWitnessProbe{
			EvidenceRef:            evidenceRef,
			QualificationStatus:    actorContextR3ALQualificationExplicit,
			QualificationSourceRef: "test-only:r3am/qualification/" + evidenceRef,
		})
	}
	assessments, err := actorContextR3ALAssessEvidence(record, result, witnesses)
	if err != nil {
		t.Fatal(err)
	}
	return assessments
}

func actorContextR3AMMaterializeAssertions(
	assessments []actorContextR3ALEvidenceAssessmentProbe,
	witnesses []actorContextR3AMAuthorityAssertionWitnessProbe,
) ([]actorContextR3AMQualifiedAuthorityAssertionProbe, error) {
	assessmentByEvidence := map[string]actorContextR3ALEvidenceAssessmentProbe{}
	for _, assessment := range assessments {
		if assessment.EvidenceRef == "" ||
			assessment.RequestIdentity == "" ||
			assessment.RequestRevisionIdentity == "" ||
			assessment.ResponseRef == "" {
			return nil, fmt.Errorf("R3AM evidence assessment identity is incomplete")
		}
		if previous, found := assessmentByEvidence[assessment.EvidenceRef]; found &&
			!reflect.DeepEqual(previous, assessment) {
			return nil, fmt.Errorf("R3AM conflicting assessments for evidence %q", assessment.EvidenceRef)
		}
		assessmentByEvidence[assessment.EvidenceRef] = assessment
	}

	unique := map[string]actorContextR3AMQualifiedAuthorityAssertionProbe{}
	for _, witness := range witnesses {
		if witness.EvidenceRef == "" ||
			witness.AssertionClass == "" ||
			witness.AssertedValue == "" ||
			witness.AssertionSourceRef == "" {
			return nil, fmt.Errorf("R3AM assertion witness is incomplete")
		}
		assessment, found := assessmentByEvidence[witness.EvidenceRef]
		if !found {
			return nil, fmt.Errorf("R3AM assertion references evidence absent from current assessment")
		}
		if assessment.QualificationStatus != actorContextR3ALQualificationExplicit {
			return nil, fmt.Errorf("R3AM assertion evidence is not explicitly qualified")
		}

		identity := strings.Join([]string{
			assessment.RequestIdentity,
			assessment.RequestRevisionIdentity,
			assessment.ResponseRef,
			witness.EvidenceRef,
			witness.AssertionClass,
			witness.AssertedValue,
			witness.AssertionSourceRef,
		}, "||")
		unique[identity] = actorContextR3AMQualifiedAuthorityAssertionProbe{
			RequestIdentity:         assessment.RequestIdentity,
			RequestRevisionIdentity: assessment.RequestRevisionIdentity,
			ResponseRef:             assessment.ResponseRef,
			EvidenceRef:             witness.EvidenceRef,
			QualificationStatus:     assessment.QualificationStatus,
			AssertionClass:          witness.AssertionClass,
			AssertedValue:           witness.AssertedValue,
			AssertionSourceRef:      witness.AssertionSourceRef,
			AssertionIdentity:       identity,
			AuthorityLimits: []string{
				"qualified-evidence-assertion-does-not-mean-accepted-authority",
				"asserted-value-does-not-write-current-authority-status",
				"assertion-does-not-create-accepted-decision",
				"assertion-does-not-resolve-stop",
				"conflicting-assertions-do-not-select-winner",
				"assertion-does-not-authorize-operation",
			},
		}
	}

	out := make([]actorContextR3AMQualifiedAuthorityAssertionProbe, 0, len(unique))
	for _, assertion := range unique {
		sort.Strings(assertion.AuthorityLimits)
		out = append(out, assertion)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AssertionIdentity < out[j].AssertionIdentity
	})
	return out, nil
}

func actorContextR3AMFindAssertion(
	values []actorContextR3AMQualifiedAuthorityAssertionProbe,
	assertionClass string,
	assertedValue string,
) *actorContextR3AMQualifiedAuthorityAssertionProbe {
	for i := range values {
		if values[i].AssertionClass == assertionClass &&
			values[i].AssertedValue == assertedValue {
			return &values[i]
		}
	}
	return nil
}
