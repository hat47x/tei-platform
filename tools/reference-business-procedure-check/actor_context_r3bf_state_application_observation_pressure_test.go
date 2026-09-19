package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3BFReportedApplied    = "state-applicator-reported-applied"
	actorContextR3BFReportedNotApplied = "state-applicator-reported-not-applied"
	actorContextR3BFReportedConflict   = "state-applicator-reported-conflict"
	actorContextR3BFReportedUnknown    = "state-applicator-reported-unknown"
)

type actorContextR3BFExecutionObservationWitnessProbe struct {
	StateApplicationCandidateIdentity   string
	AcceptedDecisionSinkRef             string
	RevisionAuthorityRef                string
	StateApplicationExecutionObservationRef string
	ReportedApplicationOutcome          string
}

type actorContextR3BFAuthorityProjectionObservationWitnessProbe struct {
	StateApplicationCandidateIdentity      string
	AcceptedDecisionSinkRef                string
	RevisionAuthorityRef                   string
	CurrentAuthorityProjectionLocusRef     string
	CurrentAuthorityObservationRef         string
	ObservedDecisionRevision               string
	ObservedStateFingerprint               string
	ObservedAuthorityProjectionFingerprint string
}

type actorContextR3BFRawExecutionObservationProbe struct {
	StateApplicationCandidateIdentity      string
	AcceptedDecisionSinkRef                string
	RevisionAuthorityRef                   string
	StateApplicationExecutionObservationRef string
	ReportedApplicationOutcome             string
}

type actorContextR3BFRawAuthorityProjectionObservationProbe struct {
	StateApplicationCandidateIdentity      string
	AcceptedDecisionSinkRef                string
	RevisionAuthorityRef                   string
	CurrentAuthorityProjectionLocusRef     string
	CurrentAuthorityObservationRef         string
	ObservedDecisionRevision               string
	ObservedStateFingerprint               string
	ObservedAuthorityProjectionFingerprint string
}

type actorContextR3BFObservationSetProbe struct {
	StateApplicationCandidateIdentity string
	ExecutionObservations             []actorContextR3BFRawExecutionObservationProbe
	AuthorityProjectionObservations   []actorContextR3BFRawAuthorityProjectionObservationProbe
	AuthorityLimits                   []string
}

func TestActorContextR3BFCandidateAloneDoesNotFabricateEvidence(t *testing.T) {
	candidate := actorContextR3BFCandidateFixture(t, actorContextR3BEDecisionApply)
	got, err := actorContextR3BFObserve(candidate, nil, nil)
	if err != nil { t.Fatal(err) }
	if len(got.ExecutionObservations)!=0 || len(got.AuthorityProjectionObservations)!=0 {
		t.Fatalf("R3BF fabricated observations from candidate alone: %+v",got)
	}
}

func TestActorContextR3BFRejectCandidateIsNotObservationEntry(t *testing.T) {
	candidate := actorContextR3BFCandidateFixture(t, actorContextR3BEDecisionReject)
	exec := actorContextR3BFExactExecutionWitness(candidate, actorContextR3BFReportedNotApplied)
	if _,err:=actorContextR3BFObserve(candidate,[]actorContextR3BFExecutionObservationWitnessProbe{exec},nil); err==nil {
		t.Fatal("R3BF accepted execution observation for reject candidate")
	}
}

func TestActorContextR3BFExecutionAndProjectionRemainIndependent(t *testing.T) {
	candidate:=actorContextR3BFCandidateFixture(t,actorContextR3BEDecisionApply)
	exec:=actorContextR3BFExactExecutionWitness(candidate,actorContextR3BFReportedApplied)
	got,err:=actorContextR3BFObserve(candidate,[]actorContextR3BFExecutionObservationWitnessProbe{exec},nil)
	if err!=nil { t.Fatal(err) }
	if len(got.ExecutionObservations)!=1 || len(got.AuthorityProjectionObservations)!=0 {
		t.Fatalf("R3BF execution-only evidence drifted: %+v",got)
	}
	proj:=actorContextR3BFExactProjectionWitness(candidate)
	got,err=actorContextR3BFObserve(candidate,nil,[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{proj})
	if err!=nil { t.Fatal(err) }
	if len(got.ExecutionObservations)!=0 || len(got.AuthorityProjectionObservations)!=1 {
		t.Fatalf("R3BF projection-only evidence drifted: %+v",got)
	}
}

func TestActorContextR3BFWrongBindingFailsClosed(t *testing.T) {
	candidate:=actorContextR3BFCandidateFixture(t,actorContextR3BEDecisionApply)
	exec:=actorContextR3BFExactExecutionWitness(candidate,actorContextR3BFReportedApplied)
	exec.StateApplicationCandidateIdentity += ":other"
	if _,err:=actorContextR3BFObserve(candidate,[]actorContextR3BFExecutionObservationWitnessProbe{exec},nil); err==nil {
		t.Fatal("R3BF accepted wrong execution candidate binding")
	}
	proj:=actorContextR3BFExactProjectionWitness(candidate)
	proj.CurrentAuthorityProjectionLocusRef += ":other"
	if _,err:=actorContextR3BFObserve(candidate,nil,[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{proj}); err==nil {
		t.Fatal("R3BF accepted wrong projection locus binding")
	}
}

func TestActorContextR3BFRawOutcomeAndObservedFingerprintsRemainNonAuthoritative(t *testing.T) {
	candidate:=actorContextR3BFCandidateFixture(t,actorContextR3BEDecisionApply)
	for _,outcome:=range []string{
		actorContextR3BFReportedApplied,actorContextR3BFReportedNotApplied,
		actorContextR3BFReportedConflict,actorContextR3BFReportedUnknown,
	} {
		exec:=actorContextR3BFExactExecutionWitness(candidate,outcome)
		proj:=actorContextR3BFExactProjectionWitness(candidate)
		got,err:=actorContextR3BFObserve(candidate,
			[]actorContextR3BFExecutionObservationWitnessProbe{exec},
			[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{proj})
		if err!=nil { t.Fatal(err) }
		if len(got.ExecutionObservations)!=1 || got.ExecutionObservations[0].ReportedApplicationOutcome!=outcome ||
			len(got.AuthorityProjectionObservations)!=1 {
			t.Fatalf("R3BF raw evidence drifted outcome=%q: %+v",outcome,got)
		}
	}
}

func TestActorContextR3BFDuplicateNormalizesAndConflictFailsClosed(t *testing.T) {
	candidate:=actorContextR3BFCandidateFixture(t,actorContextR3BEDecisionApply)
	exec:=actorContextR3BFExactExecutionWitness(candidate,actorContextR3BFReportedApplied)
	proj:=actorContextR3BFExactProjectionWitness(candidate)
	got,err:=actorContextR3BFObserve(candidate,
		[]actorContextR3BFExecutionObservationWitnessProbe{exec,exec},
		[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{proj,proj})
	if err!=nil { t.Fatal(err) }
	if len(got.ExecutionObservations)!=1 || len(got.AuthorityProjectionObservations)!=1 {
		t.Fatalf("R3BF did not normalize duplicates: %+v",got)
	}
	conflict:=exec
	conflict.ReportedApplicationOutcome=actorContextR3BFReportedConflict
	if _,err:=actorContextR3BFObserve(candidate,
		[]actorContextR3BFExecutionObservationWitnessProbe{exec,conflict},nil); err==nil {
		t.Fatal("R3BF silently selected conflicting same-ref execution outcome")
	}
	projConflict:=proj
	projConflict.ObservedAuthorityProjectionFingerprint += ":other"
	if _,err:=actorContextR3BFObserve(candidate,nil,
		[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{proj,projConflict}); err==nil {
		t.Fatal("R3BF silently selected conflicting same-ref projection observation")
	}
}

func TestActorContextR3BFRepeatedBuildAndInputOrderAreStable(t *testing.T) {
	candidate:=actorContextR3BFCandidateFixture(t,actorContextR3BEDecisionApply)
	e1:=actorContextR3BFExactExecutionWitness(candidate,actorContextR3BFReportedApplied)
	e2:=e1
	e2.StateApplicationExecutionObservationRef="test-only:r3bf/execution-2"
	e2.ReportedApplicationOutcome=actorContextR3BFReportedUnknown
	p1:=actorContextR3BFExactProjectionWitness(candidate)
	p2:=p1
	p2.CurrentAuthorityObservationRef="test-only:r3bf/projection-2"
	p2.ObservedAuthorityProjectionFingerprint="sha256:test-only-projection-2"

	first,err:=actorContextR3BFObserve(candidate,
		[]actorContextR3BFExecutionObservationWitnessProbe{e1,e2},
		[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{p1,p2})
	if err!=nil { t.Fatal(err) }
	second,err:=actorContextR3BFObserve(candidate,
		[]actorContextR3BFExecutionObservationWitnessProbe{e2,e1},
		[]actorContextR3BFAuthorityProjectionObservationWitnessProbe{p2,p1})
	if err!=nil { t.Fatal(err) }
	if !reflect.DeepEqual(first,second) {
		t.Fatalf("R3BF input order changed normalized Evidence: first=%+v second=%+v",first,second)
	}
}

func TestActorContextR3BFShapeHasNoAppliedStateOrCurrentAuthorityShortcut(t *testing.T) {
	for _,typ:=range []reflect.Type{
		reflect.TypeOf(actorContextR3BFObservationSetProbe{}),
		reflect.TypeOf(actorContextR3BFRawExecutionObservationProbe{}),
		reflect.TypeOf(actorContextR3BFRawAuthorityProjectionObservationProbe{}),
	} {
		for _,forbidden:=range []string{
			"StateApplied","StateSynchronized","RevisionAccepted","RevisionApplied",
			"AcceptedDecision","CurrentDecision","CurrentAuthority","AppliedAuthority",
			"CurrentAuthorityAccepted","RuntimeOutcome","AuthorizationResult",
			"Retry","RetryEligible","RetryRequired","Rollback","Compensation",
			"ToolCall","Command","NextAction","CanExecute","Authorized","Allowed",
			"Confidence","Priority","Severity","Winner",
		} {
			if _,found:=typ.FieldByName(forbidden); found {
				t.Fatalf("R3BF observation introduced forbidden applied-state/authority shortcut %q",forbidden)
			}
		}
	}
}

func actorContextR3BFCandidateFixture(t *testing.T, decision string) actorContextR3BEStateApplicationCandidateProbe {
	t.Helper()
	reconciliation,result,gate,evidence:=actorContextR3BESingleQualifiedFixture(t)
	witness:=actorContextR3BEExactWitness(result,gate,decision)
	got,err:=actorContextR3BEBuildCandidates(reconciliation,result,gate,evidence,
		[]actorContextR3BEStateApplicationWitnessProbe{witness})
	if err!=nil { t.Fatal(err) }
	if len(got)!=1 { t.Fatalf("R3BF setup state-application candidate count=%d want=1",len(got)) }
	return got[0]
}

func actorContextR3BFExactExecutionWitness(
	candidate actorContextR3BEStateApplicationCandidateProbe,
	outcome string,
) actorContextR3BFExecutionObservationWitnessProbe {
	return actorContextR3BFExecutionObservationWitnessProbe{
		StateApplicationCandidateIdentity:candidate.StateApplicationCandidateIdentity,
		AcceptedDecisionSinkRef:candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:candidate.RevisionAuthorityRef,
		StateApplicationExecutionObservationRef:"test-only:r3bf/execution-1",
		ReportedApplicationOutcome:outcome,
	}
}

func actorContextR3BFExactProjectionWitness(
	candidate actorContextR3BEStateApplicationCandidateProbe,
) actorContextR3BFAuthorityProjectionObservationWitnessProbe {
	return actorContextR3BFAuthorityProjectionObservationWitnessProbe{
		StateApplicationCandidateIdentity:candidate.StateApplicationCandidateIdentity,
		AcceptedDecisionSinkRef:candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:candidate.RevisionAuthorityRef,
		CurrentAuthorityProjectionLocusRef:candidate.CurrentAuthorityProjectionLocusRef,
		CurrentAuthorityObservationRef:"test-only:r3bf/projection-1",
		ObservedDecisionRevision:candidate.ObservedDecisionRevision,
		ObservedStateFingerprint:candidate.ObservedStateFingerprint,
		ObservedAuthorityProjectionFingerprint:"sha256:test-only-authority-projection",
	}
}

func actorContextR3BFObserve(
	candidate actorContextR3BEStateApplicationCandidateProbe,
	executionWitnesses []actorContextR3BFExecutionObservationWitnessProbe,
	projectionWitnesses []actorContextR3BFAuthorityProjectionObservationWitnessProbe,
) (actorContextR3BFObservationSetProbe,error) {
	if candidate.StateApplicationCandidateIdentity=="" ||
		candidate.AcceptedDecisionSinkRef=="" ||
		candidate.RevisionAuthorityRef=="" ||
		candidate.CurrentAuthorityProjectionLocusRef=="" {
		return actorContextR3BFObservationSetProbe{},fmt.Errorf("R3BF state-application candidate context is incomplete")
	}
	if (len(executionWitnesses)>0 || len(projectionWitnesses)>0) &&
		candidate.Decision!=actorContextR3BEDecisionApply {
		return actorContextR3BFObservationSetProbe{},fmt.Errorf("R3BF only apply candidates may have post-application observations")
	}

	executions:=map[string]actorContextR3BFRawExecutionObservationProbe{}
	for _,w:=range executionWitnesses {
		if w.StateApplicationCandidateIdentity!=candidate.StateApplicationCandidateIdentity ||
			w.AcceptedDecisionSinkRef!=candidate.AcceptedDecisionSinkRef ||
			w.RevisionAuthorityRef!=candidate.RevisionAuthorityRef ||
			w.StateApplicationExecutionObservationRef=="" ||
			!actorContextR3BFValidReportedOutcome(w.ReportedApplicationOutcome) {
			return actorContextR3BFObservationSetProbe{},fmt.Errorf("R3BF execution observation is incomplete or mismatched")
		}
		raw:=actorContextR3BFRawExecutionObservationProbe(w)
		if prev,ok:=executions[w.StateApplicationExecutionObservationRef]; ok {
			if !reflect.DeepEqual(prev,raw) {
				return actorContextR3BFObservationSetProbe{},fmt.Errorf("R3BF conflicting execution observations share ref %q",w.StateApplicationExecutionObservationRef)
			}
			continue
		}
		executions[w.StateApplicationExecutionObservationRef]=raw
	}

	projections:=map[string]actorContextR3BFRawAuthorityProjectionObservationProbe{}
	for _,w:=range projectionWitnesses {
		if w.StateApplicationCandidateIdentity!=candidate.StateApplicationCandidateIdentity ||
			w.AcceptedDecisionSinkRef!=candidate.AcceptedDecisionSinkRef ||
			w.RevisionAuthorityRef!=candidate.RevisionAuthorityRef ||
			w.CurrentAuthorityProjectionLocusRef!=candidate.CurrentAuthorityProjectionLocusRef ||
			w.CurrentAuthorityObservationRef=="" ||
			w.ObservedDecisionRevision=="" ||
			w.ObservedStateFingerprint=="" ||
			w.ObservedAuthorityProjectionFingerprint=="" {
			return actorContextR3BFObservationSetProbe{},fmt.Errorf("R3BF authority-projection observation is incomplete or mismatched")
		}
		raw:=actorContextR3BFRawAuthorityProjectionObservationProbe(w)
		if prev,ok:=projections[w.CurrentAuthorityObservationRef]; ok {
			if !reflect.DeepEqual(prev,raw) {
				return actorContextR3BFObservationSetProbe{},fmt.Errorf("R3BF conflicting authority-projection observations share ref %q",w.CurrentAuthorityObservationRef)
			}
			continue
		}
		projections[w.CurrentAuthorityObservationRef]=raw
	}

	out:=actorContextR3BFObservationSetProbe{
		StateApplicationCandidateIdentity:candidate.StateApplicationCandidateIdentity,
		AuthorityLimits:actorContextR0SortedUnique([]string{
			"execution-observation-does-not-prove-authoritative-state-applied",
			"matching-revision-and-state-fingerprint-do-not-accept-or-apply-revision",
			"authority-projection-observation-does-not-prove-current-authority",
			"reported-conflict-or-unknown-does-not-create-retry-rollback-or-compensation-authority",
		}),
	}
	for _,v:=range executions { out.ExecutionObservations=append(out.ExecutionObservations,v) }
	sort.Slice(out.ExecutionObservations,func(i,j int) bool {
		return out.ExecutionObservations[i].StateApplicationExecutionObservationRef <
			out.ExecutionObservations[j].StateApplicationExecutionObservationRef
	})
	for _,v:=range projections { out.AuthorityProjectionObservations=append(out.AuthorityProjectionObservations,v) }
	sort.Slice(out.AuthorityProjectionObservations,func(i,j int) bool {
		return out.AuthorityProjectionObservations[i].CurrentAuthorityObservationRef <
			out.AuthorityProjectionObservations[j].CurrentAuthorityObservationRef
	})
	return out,nil
}

func actorContextR3BFValidReportedOutcome(value string) bool {
	switch value {
	case actorContextR3BFReportedApplied,actorContextR3BFReportedNotApplied,
		actorContextR3BFReportedConflict,actorContextR3BFReportedUnknown:
		return true
	default:
		return false
	}
}
