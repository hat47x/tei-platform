package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ALQualificationUnresolved = "qualification-unresolved"
	actorContextR3ALQualificationExplicit   = "explicit-evidence-qualified"
)

type actorContextR3ALEvidenceQualificationWitnessProbe struct {
	EvidenceRef            string
	QualificationStatus    string
	QualificationSourceRef string
}

type actorContextR3ALEvidenceAssessmentProbe struct {
	RequestIdentity             string
	RequestRevisionIdentity     string
	ResponseRef                 string
	EvidenceRef                 string
	ProvidedInformationClasses  []string
	MatchingEvidenceNeedClasses []string
	QualificationStatus         string
	QualificationSourceRefs     []string
	AuthorityLimits             []string
}

func TestActorContextR3ALObservedEvidenceStartsQualificationUnresolved(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, []string{
		"authorization-evaluation-freshness-authority",
	}, []string{
		"test-only:r3al/evidence-1",
	})

	got, err := actorContextR3ALAssessEvidence(record, result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AL assessment count=%d want=1: %+v", len(got), got)
	}
	assessment := got[0]
	if assessment.EvidenceRef != "test-only:r3al/evidence-1" ||
		assessment.QualificationStatus != actorContextR3ALQualificationUnresolved {
		t.Fatalf("R3AL observed evidence was treated as qualified: %+v", assessment)
	}
	if !reflect.DeepEqual(assessment.MatchingEvidenceNeedClasses, []string{
		"authorization-evaluation-freshness-authority",
	}) {
		t.Fatalf("R3AL lost evidence need class overlap: %+v", assessment)
	}
}

func TestActorContextR3ALClassOverlapDoesNotQualifyEvidence(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, []string{
		"authorization-evaluation-freshness-authority",
	}, []string{
		"test-only:r3al/evidence-overlap",
	})
	got, err := actorContextR3ALAssessEvidence(record, result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 ||
		len(got[0].MatchingEvidenceNeedClasses) != 1 ||
		got[0].QualificationStatus != actorContextR3ALQualificationUnresolved {
		t.Fatalf("R3AL class overlap became evidence qualification: %+v", got)
	}
}

func TestActorContextR3ALExplicitWitnessQualifiesOnlyExactEvidenceRef(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, []string{
		"authorization-evaluation-freshness-authority",
	}, []string{
		"test-only:r3al/evidence-a",
		"test-only:r3al/evidence-b",
	})
	got, err := actorContextR3ALAssessEvidence(
		record,
		result,
		[]actorContextR3ALEvidenceQualificationWitnessProbe{{
			EvidenceRef:            "test-only:r3al/evidence-a",
			QualificationStatus:    actorContextR3ALQualificationExplicit,
			QualificationSourceRef: "test-only:r3al/qualifier-a",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("R3AL assessment count=%d want=2: %+v", len(got), got)
	}

	a := actorContextR3ALFindAssessment(got, "test-only:r3al/evidence-a")
	b := actorContextR3ALFindAssessment(got, "test-only:r3al/evidence-b")
	if a == nil || a.QualificationStatus != actorContextR3ALQualificationExplicit ||
		!reflect.DeepEqual(a.QualificationSourceRefs, []string{"test-only:r3al/qualifier-a"}) {
		t.Fatalf("R3AL lost explicit qualification witness: %+v", a)
	}
	if b == nil || b.QualificationStatus != actorContextR3ALQualificationUnresolved ||
		len(b.QualificationSourceRefs) != 0 {
		t.Fatalf("R3AL propagated qualification to unrelated evidence: %+v", b)
	}
}

func TestActorContextR3ALQualificationWitnessNeedsProvenance(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, nil, []string{
		"test-only:r3al/evidence",
	})
	if _, err := actorContextR3ALAssessEvidence(
		record,
		result,
		[]actorContextR3ALEvidenceQualificationWitnessProbe{{
			EvidenceRef:         "test-only:r3al/evidence",
			QualificationStatus: actorContextR3ALQualificationExplicit,
		}},
	); err == nil {
		t.Fatal("R3AL accepted qualification witness without provenance")
	}
}

func TestActorContextR3ALUnknownEvidenceQualificationFailsClosed(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, nil, []string{
		"test-only:r3al/current-evidence",
	})
	if _, err := actorContextR3ALAssessEvidence(
		record,
		result,
		[]actorContextR3ALEvidenceQualificationWitnessProbe{{
			EvidenceRef:            "test-only:r3al/not-in-response",
			QualificationStatus:    actorContextR3ALQualificationExplicit,
			QualificationSourceRef: "test-only:r3al/qualifier",
		}},
	); err == nil {
		t.Fatal("R3AL accepted qualification witness for evidence absent from current response")
	}
}

func TestActorContextR3ALRequestRevisionMismatchFailsClosed(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, nil, []string{
		"test-only:r3al/evidence",
	})
	result.RequestRevisionIdentity += ":stale"
	if _, err := actorContextR3ALAssessEvidence(record, result, nil); err == nil {
		t.Fatal("R3AL accepted qualification against stale request revision")
	}
}

func TestActorContextR3ALNoEvidenceRefsProducesNoAssessmentWithoutResolution(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, []string{
		"authorization-evaluation-freshness-authority",
	}, nil)
	got, err := actorContextR3ALAssessEvidence(record, result, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AL fabricated assessment without EvidenceRef: %+v", got)
	}
}

func TestActorContextR3ALConflictingQualificationWitnessesFailClosed(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, nil, []string{
		"test-only:r3al/evidence-conflict",
	})
	_, err := actorContextR3ALAssessEvidence(
		record,
		result,
		[]actorContextR3ALEvidenceQualificationWitnessProbe{
			{
				EvidenceRef:            "test-only:r3al/evidence-conflict",
				QualificationStatus:    actorContextR3ALQualificationExplicit,
				QualificationSourceRef: "test-only:r3al/qualifier-a",
			},
			{
				EvidenceRef:            "test-only:r3al/evidence-conflict",
				QualificationStatus:    actorContextR3ALQualificationUnresolved,
				QualificationSourceRef: "test-only:r3al/qualifier-b",
			},
		},
	)
	if err == nil {
		t.Fatal("R3AL silently selected among conflicting qualification witnesses")
	}
}

func TestActorContextR3ALRepeatedAndInputOrderAssessmentAreStable(t *testing.T) {
	record, result := actorContextR3ALCurrentCorrelation(t, []string{
		"authorization-evaluation-freshness-authority",
		"test-only:other-class",
	}, []string{
		"test-only:r3al/evidence-a",
		"test-only:r3al/evidence-b",
	})
	witnesses := []actorContextR3ALEvidenceQualificationWitnessProbe{
		{
			EvidenceRef:            "test-only:r3al/evidence-b",
			QualificationStatus:    actorContextR3ALQualificationExplicit,
			QualificationSourceRef: "test-only:r3al/qualifier-b",
		},
	}

	first, err := actorContextR3ALAssessEvidence(record, result, witnesses)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3ALAssessEvidence(record, result, witnesses)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AL repeated assessment changed identity: first=%+v second=%+v", first, second)
	}

	reversed := result
	reversed.ProvidedInformationClasses = actorContextR3AKReverseStrings(result.ProvidedInformationClasses)
	reversed.EvidenceRefs = actorContextR3AKReverseStrings(result.EvidenceRefs)
	reversedWitnesses := append([]actorContextR3ALEvidenceQualificationWitnessProbe(nil), witnesses...)
	for i, j := 0, len(reversedWitnesses)-1; i < j; i, j = i+1, j-1 {
		reversedWitnesses[i], reversedWitnesses[j] = reversedWitnesses[j], reversedWitnesses[i]
	}

	normalized, err := actorContextR3ALAssessEvidence(record, reversed, reversedWitnesses)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, normalized) {
		t.Fatalf("R3AL input order changed normalized assessment: first=%+v normalized=%+v",
			first, normalized)
	}
}

func TestActorContextR3ALAssessmentShapeHasNoAcceptanceOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3ALEvidenceQualificationWitnessProbe{}),
		reflect.TypeOf(actorContextR3ALEvidenceAssessmentProbe{}),
	} {
		for _, forbidden := range []string{
			"AuthorityValue",
			"AcceptedAuthorityValue",
			"AcceptedDecision",
			"Approved",
			"Sufficient",
			"Resolved",
			"ResolutionComplete",
			"Fresh",
			"Safe",
			"Reusable",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Apply",
			"AutoApply",
			"ToolCall",
			"Command",
			"NextAction",
			"Retry",
			"Reauthorize",
			"Confidence",
			"Priority",
			"Severity",
			"Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AL evidence qualification introduced forbidden acceptance/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3ALCurrentCorrelation(
	t *testing.T,
	providedInformationClasses []string,
	evidenceRefs []string,
) (actorContextR3AKRequestRecordProbe, actorContextR3AKCorrelationResultProbe) {
	t.Helper()

	record, err := actorContextR3AKPrepareRequest(actorContextR3AKCurrentRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	response := actorContextR3AKResponseWitnessProbe{
		RequestIdentity:            record.RequestIdentity,
		RequestRevisionIdentity:    record.RequestRevisionIdentity,
		ResponseRef:                "test-only:r3al/current-response",
		ProvidedInformationClasses: append([]string(nil), providedInformationClasses...),
		EvidenceRefs:               append([]string(nil), evidenceRefs...),
	}
	result, err := actorContextR3AKCorrelateResponse(record, response)
	if err != nil {
		t.Fatal(err)
	}
	return record, result
}

func actorContextR3ALAssessEvidence(
	record actorContextR3AKRequestRecordProbe,
	result actorContextR3AKCorrelationResultProbe,
	witnesses []actorContextR3ALEvidenceQualificationWitnessProbe,
) ([]actorContextR3ALEvidenceAssessmentProbe, error) {
	if record.RequestIdentity == "" || record.RequestRevisionIdentity == "" {
		return nil, fmt.Errorf("R3AL request identity is incomplete")
	}
	if result.RequestIdentity != record.RequestIdentity ||
		result.RequestRevisionIdentity != record.RequestRevisionIdentity {
		return nil, fmt.Errorf("R3AL correlation does not match current request revision")
	}
	if result.CorrelationStatus != actorContextR3AKCorrelationMatchedCurrent ||
		result.LifecycleState != actorContextR3AKLifecycleResponseCorrelated {
		return nil, fmt.Errorf("R3AL requires current-revision correlated response")
	}

	evidenceRefs := actorContextR0SortedUnique(append([]string(nil), result.EvidenceRefs...))
	evidenceSet := map[string]bool{}
	for _, ref := range evidenceRefs {
		if ref == "" {
			return nil, fmt.Errorf("R3AL evidence ref must not be empty")
		}
		evidenceSet[ref] = true
	}

	witnessByEvidence := map[string]actorContextR3ALEvidenceQualificationWitnessProbe{}
	for _, witness := range witnesses {
		if witness.EvidenceRef == "" ||
			witness.QualificationStatus == "" ||
			witness.QualificationSourceRef == "" {
			return nil, fmt.Errorf("R3AL qualification witness is incomplete")
		}
		if !evidenceSet[witness.EvidenceRef] {
			return nil, fmt.Errorf("R3AL qualification witness references evidence absent from response")
		}
		if witness.QualificationStatus != actorContextR3ALQualificationExplicit &&
			witness.QualificationStatus != actorContextR3ALQualificationUnresolved {
			return nil, fmt.Errorf("R3AL unknown qualification status %q", witness.QualificationStatus)
		}
		if previous, found := witnessByEvidence[witness.EvidenceRef]; found {
			if previous.QualificationStatus != witness.QualificationStatus {
				return nil, fmt.Errorf("R3AL conflicting qualification witnesses for %q", witness.EvidenceRef)
			}
			if previous.QualificationSourceRef != witness.QualificationSourceRef {
				return nil, fmt.Errorf("R3AL multiple qualification sources need explicit aggregation authority")
			}
			continue
		}
		witnessByEvidence[witness.EvidenceRef] = witness
	}

	provided := actorContextR0SortedUnique(
		append([]string(nil), result.ProvidedInformationClasses...),
	)
	needed := actorContextR0SortedUnique(
		append([]string(nil), record.Request.EvidenceNeedClasses...),
	)
	matches := actorContextR3ALIntersection(needed, provided)

	out := make([]actorContextR3ALEvidenceAssessmentProbe, 0, len(evidenceRefs))
	for _, evidenceRef := range evidenceRefs {
		status := actorContextR3ALQualificationUnresolved
		var qualificationRefs []string
		if witness, found := witnessByEvidence[evidenceRef]; found {
			status = witness.QualificationStatus
			qualificationRefs = []string{witness.QualificationSourceRef}
		}

		assessment := actorContextR3ALEvidenceAssessmentProbe{
			RequestIdentity:             record.RequestIdentity,
			RequestRevisionIdentity:     record.RequestRevisionIdentity,
			ResponseRef:                 result.ResponseRef,
			EvidenceRef:                 evidenceRef,
			ProvidedInformationClasses:  append([]string(nil), provided...),
			MatchingEvidenceNeedClasses: append([]string(nil), matches...),
			QualificationStatus:         status,
			QualificationSourceRefs:     actorContextR0SortedUnique(qualificationRefs),
			AuthorityLimits: []string{
				"observed-evidence-does-not-mean-qualified-evidence",
				"matching-information-class-does-not-qualify-evidence",
				"evidence-qualification-does-not-accept-authority",
				"qualified-evidence-does-not-resolve-stop",
				"qualification-does-not-authorize-operation",
				"qualification-does-not-select-accepted-decision",
			},
		}
		sort.Strings(assessment.AuthorityLimits)
		out = append(out, assessment)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].EvidenceRef < out[j].EvidenceRef
	})
	return out, nil
}

func actorContextR3ALIntersection(left []string, right []string) []string {
	rightSet := map[string]bool{}
	for _, value := range right {
		rightSet[value] = true
	}
	var out []string
	for _, value := range left {
		if rightSet[value] {
			out = append(out, value)
		}
	}
	return actorContextR0SortedUnique(out)
}

func actorContextR3ALFindAssessment(
	values []actorContextR3ALEvidenceAssessmentProbe,
	evidenceRef string,
) *actorContextR3ALEvidenceAssessmentProbe {
	for i := range values {
		if values[i].EvidenceRef == evidenceRef {
			return &values[i]
		}
	}
	return nil
}
