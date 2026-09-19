package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3ATReportedApplied    = "writer-reported-applied"
	actorContextR3ATReportedNotApplied = "writer-reported-not-applied"
	actorContextR3ATReportedConflict   = "writer-reported-conflict"
	actorContextR3ATReportedUnknown    = "writer-reported-unknown"
)

type actorContextR3ATWriterObservationProbe struct {
	WriteAttemptCandidateIdentity    string
	AcceptedDecisionSinkRef          string
	IdempotencyIdentity              string
	WriterObservationRef             string
	ReportedOutcome                  string
	PostWriteRevisionObservationRef  string
	PostWriteObservedDecisionRevision string
}

type actorContextR3ATNormalizedObservationProbe struct {
	WriteAttemptCandidateIdentity    string
	AcceptedDecisionSinkRef          string
	IdempotencyIdentity              string
	WriterObservationRef             string
	ReportedOutcome                  string
	PostWriteRevisionObservationRef  string
	PostWriteObservedDecisionRevision string
	ObservationIdentity              string
	AuthorityLimits                  []string
}

func TestActorContextR3ATExactObservationBindsToWriteAttemptCandidate(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)
	input := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedApplied)

	got, err := actorContextR3ATNormalizeObservations(attempt, []actorContextR3ATWriterObservationProbe{input})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AT observation count=%d want=1: %+v", len(got), got)
	}
	value := got[0]
	if value.WriteAttemptCandidateIdentity != attempt.WriteAttemptCandidateIdentity ||
		value.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
		value.IdempotencyIdentity != attempt.IdempotencyIdentity ||
		value.ReportedOutcome != actorContextR3ATReportedApplied ||
		value.ObservationIdentity == "" {
		t.Fatalf("R3AT observation binding drifted: %+v", value)
	}
}

func TestActorContextR3ATWrongAttemptSinkOrIdempotencyIdentityFailsClosed(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)
	base := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedApplied)
	cases := map[string]func(*actorContextR3ATWriterObservationProbe){
		"attempt": func(v *actorContextR3ATWriterObservationProbe) {
			v.WriteAttemptCandidateIdentity += ":other"
		},
		"sink": func(v *actorContextR3ATWriterObservationProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"idempotency": func(v *actorContextR3ATWriterObservationProbe) {
			v.IdempotencyIdentity += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3ATNormalizeObservations(
				attempt, []actorContextR3ATWriterObservationProbe{value},
			); err == nil {
				t.Fatalf("R3AT accepted mismatched %s", name)
			}
		})
	}
}

func TestActorContextR3ATMissingObservationAttributionFailsClosed(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)
	base := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedApplied)
	cases := map[string]func(*actorContextR3ATWriterObservationProbe){
		"writer": func(v *actorContextR3ATWriterObservationProbe) {
			v.WriterObservationRef = ""
		},
		"revision": func(v *actorContextR3ATWriterObservationProbe) {
			v.PostWriteRevisionObservationRef = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3ATNormalizeObservations(
				attempt, []actorContextR3ATWriterObservationProbe{value},
			); err == nil {
				t.Fatalf("R3AT accepted unattributed %s observation", name)
			}
		})
	}
}

func TestActorContextR3ATRawOutcomeClassesRemainObservations(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)
	for _, outcome := range []string{
		actorContextR3ATReportedApplied,
		actorContextR3ATReportedNotApplied,
		actorContextR3ATReportedConflict,
		actorContextR3ATReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			got, err := actorContextR3ATNormalizeObservations(
				attempt,
				[]actorContextR3ATWriterObservationProbe{
					actorContextR3ATExactObservation(attempt, outcome),
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].ReportedOutcome != outcome {
				t.Fatalf("R3AT changed raw writer outcome %q: %+v", outcome, got)
			}
		})
	}
}

func TestActorContextR3ATPostWriteRevisionIsIndependentFromReportedOutcome(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)

	changed := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedApplied)
	changed.PostWriteObservedDecisionRevision += ":changed"
	gotChanged, err := actorContextR3ATNormalizeObservations(
		attempt, []actorContextR3ATWriterObservationProbe{changed},
	)
	if err != nil {
		t.Fatal(err)
	}

	unchanged := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedApplied)
	gotUnchanged, err := actorContextR3ATNormalizeObservations(
		attempt, []actorContextR3ATWriterObservationProbe{unchanged},
	)
	if err != nil {
		t.Fatal(err)
	}

	if gotChanged[0].ReportedOutcome != gotUnchanged[0].ReportedOutcome ||
		gotChanged[0].PostWriteObservedDecisionRevision == gotUnchanged[0].PostWriteObservedDecisionRevision {
		t.Fatalf("R3AT collapsed raw outcome and post-write revision axes: changed=%+v unchanged=%+v",
			gotChanged[0], gotUnchanged[0])
	}
}

func TestActorContextR3ATDuplicateNormalizesAndConflictingObservationFailsClosed(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)
	base := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedApplied)

	got, err := actorContextR3ATNormalizeObservations(
		attempt, []actorContextR3ATWriterObservationProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AT did not normalize exact duplicate: %+v", got)
	}

	cases := map[string]func(*actorContextR3ATWriterObservationProbe){
		"outcome": func(v *actorContextR3ATWriterObservationProbe) {
			v.ReportedOutcome = actorContextR3ATReportedConflict
		},
		"post-revision": func(v *actorContextR3ATWriterObservationProbe) {
			v.PostWriteObservedDecisionRevision += ":other"
		},
		"writer-ref": func(v *actorContextR3ATWriterObservationProbe) {
			v.WriterObservationRef += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			other := base
			mutate(&other)
			if _, err := actorContextR3ATNormalizeObservations(
				attempt, []actorContextR3ATWriterObservationProbe{base, other},
			); err == nil {
				t.Fatalf("R3AT silently aggregated conflicting %s", name)
			}
		})
	}
}

func TestActorContextR3ATRepeatedBuildAndInputOrderAreStable(t *testing.T) {
	attempt := actorContextR3ATWriteAttemptFixture(t)
	base := actorContextR3ATExactObservation(attempt, actorContextR3ATReportedUnknown)
	inputA := []actorContextR3ATWriterObservationProbe{base, base}
	inputB := []actorContextR3ATWriterObservationProbe{base, base}
	inputB[0], inputB[1] = inputB[1], inputB[0]

	first, err := actorContextR3ATNormalizeObservations(attempt, inputA)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3ATNormalizeObservations(attempt, inputB)
	if err != nil {
		t.Fatal(err)
	}
	third, err := actorContextR3ATNormalizeObservations(attempt, inputA)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, third) {
		t.Fatalf("R3AT repeated/order normalization drifted: first=%+v second=%+v third=%+v",
			first, second, third)
	}
}

func TestActorContextR3ATShapeHasNoAcceptedOutcomeAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3ATWriterObservationProbe{}),
		reflect.TypeOf(actorContextR3ATNormalizedObservationProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedWriteOutcome",
			"OutcomeAccepted",
			"CommitAccepted",
			"Committed",
			"CommitSucceeded",
			"AcceptedDecision",
			"NewDecisionRevision",
			"CurrentAuthority",
			"AppliedAuthority",
			"RuntimeOutcome",
			"AuthorizationResult",
			"Retry",
			"RetryRequired",
			"Reauthorize",
			"ToolCall",
			"Command",
			"NextAction",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Confidence",
			"Priority",
			"Severity",
			"Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AT observation introduced forbidden accepted-outcome/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3ATWriteAttemptFixture(t *testing.T) actorContextR3ASWriteAttemptCandidateProbe {
	t.Helper()
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	witness := actorContextR3ASExactWitness(candidate, evidence[0], revision)
	got, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		evidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AT setup attempt candidate count=%d want=1", len(got))
	}
	return got[0]
}

func actorContextR3ATExactObservation(
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	outcome string,
) actorContextR3ATWriterObservationProbe {
	return actorContextR3ATWriterObservationProbe{
		WriteAttemptCandidateIdentity:     attempt.WriteAttemptCandidateIdentity,
		AcceptedDecisionSinkRef:           attempt.AcceptedDecisionSinkRef,
		IdempotencyIdentity:               attempt.IdempotencyIdentity,
		WriterObservationRef:              "test-only:r3at/writer-observation",
		ReportedOutcome:                   outcome,
		PostWriteRevisionObservationRef:   "test-only:r3at/post-write-revision-observation",
		PostWriteObservedDecisionRevision: attempt.ObservedDecisionRevision,
	}
}

func actorContextR3ATNormalizeObservations(
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	input []actorContextR3ATWriterObservationProbe,
) ([]actorContextR3ATNormalizedObservationProbe, error) {
	if attempt.WriteAttemptCandidateIdentity == "" ||
		attempt.AcceptedDecisionSinkRef == "" ||
		attempt.IdempotencyIdentity == "" {
		return nil, fmt.Errorf("R3AT write-attempt candidate is incomplete")
	}
	if len(input) == 0 {
		return nil, nil
	}

	var normalized *actorContextR3ATNormalizedObservationProbe
	for _, value := range input {
		if value.WriteAttemptCandidateIdentity == "" ||
			value.AcceptedDecisionSinkRef == "" ||
			value.IdempotencyIdentity == "" ||
			value.WriterObservationRef == "" ||
			value.ReportedOutcome == "" ||
			value.PostWriteRevisionObservationRef == "" ||
			value.PostWriteObservedDecisionRevision == "" {
			return nil, fmt.Errorf("R3AT writer observation is incomplete")
		}
		if value.WriteAttemptCandidateIdentity != attempt.WriteAttemptCandidateIdentity ||
			value.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
			value.IdempotencyIdentity != attempt.IdempotencyIdentity {
			return nil, fmt.Errorf("R3AT writer observation does not bind exact write-attempt candidate")
		}
		if !actorContextR3ATKnownRawOutcome(value.ReportedOutcome) {
			return nil, fmt.Errorf("R3AT unknown raw writer outcome %q", value.ReportedOutcome)
		}

		identity := strings.Join([]string{
			value.WriteAttemptCandidateIdentity,
			value.AcceptedDecisionSinkRef,
			value.IdempotencyIdentity,
			value.WriterObservationRef,
			value.ReportedOutcome,
			value.PostWriteRevisionObservationRef,
			value.PostWriteObservedDecisionRevision,
		}, "||")
		current := actorContextR3ATNormalizedObservationProbe{
			WriteAttemptCandidateIdentity:     value.WriteAttemptCandidateIdentity,
			AcceptedDecisionSinkRef:           value.AcceptedDecisionSinkRef,
			IdempotencyIdentity:               value.IdempotencyIdentity,
			WriterObservationRef:              value.WriterObservationRef,
			ReportedOutcome:                   value.ReportedOutcome,
			PostWriteRevisionObservationRef:   value.PostWriteRevisionObservationRef,
			PostWriteObservedDecisionRevision: value.PostWriteObservedDecisionRevision,
			ObservationIdentity:               identity,
			AuthorityLimits: []string{
				"writer-reported-outcome-does-not-mean-accepted-write-outcome",
				"writer-reported-applied-does-not-mean-accepted-decision-committed",
				"post-write-revision-observation-does-not-mean-revision-accepted",
				"idempotency-identity-echo-does-not-prove-deduplication",
				"raw-attempt-observation-does-not-apply-current-authority",
				"raw-conflict-observation-does-not-authorize-retry",
			},
		}
		sort.Strings(current.AuthorityLimits)

		if normalized != nil {
			if !reflect.DeepEqual(*normalized, current) {
				return nil, fmt.Errorf("R3AT conflicting observations require separate aggregation/acceptance authority")
			}
			continue
		}
		normalized = &current
	}
	if normalized == nil {
		return nil, nil
	}
	return []actorContextR3ATNormalizedObservationProbe{*normalized}, nil
}

func actorContextR3ATKnownRawOutcome(value string) bool {
	switch value {
	case actorContextR3ATReportedApplied,
		actorContextR3ATReportedNotApplied,
		actorContextR3ATReportedConflict,
		actorContextR3ATReportedUnknown:
		return true
	default:
		return false
	}
}
