package effectcommit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const canary = "CANARY-SECRET-dsn-postgres://u:p@h/db"

type fakeSink struct{ got []Evidence }

func (s *fakeSink) Record(e Evidence) { s.got = append(s.got, e) }

type panicSink struct{}

func (panicSink) Record(Evidence) { panic("sink failure") }

func canaryRequest() Request {
	r := fixtureRequest()
	r.Payload = []byte(canary)
	return r
}

func runExec(r Runner, q Request) (Result, error) {
	return r.Execute(context.Background(), q, fixtureGrant(q))
}
func runLook(r Runner, q Request) (Result, error) {
	return r.Lookup(context.Background(), q, fixtureGrant(q))
}

func TestEvidenceOnePerCallAndFields(t *testing.T) {
	db := newMemoryProvider()
	sink := &fakeSink{}
	runner := Runner{Provider: db, Audit: AuditConfig{Sink: sink, Provenance: ProvenanceSynthetic}}
	req := canaryRequest()
	first, err := runExec(runner, req)
	if err != nil || first.Status != StatusCommitted {
		t.Fatal(first, err)
	}
	if _, err := runExec(runner, req); err != nil {
		t.Fatal(err)
	}
	if _, err := runLook(runner, req); err != nil {
		t.Fatal(err)
	}
	if len(sink.got) != 3 {
		t.Fatalf("want one Evidence per call, got %d", len(sink.got))
	}
	want := Evidence{Operation: "execute", Scope: req.Scope, Epochs: fixtureEpochs(), OperationID: "operation-1",
		Status: StatusCommitted, ReceiptVersion: 1, Provenance: ProvenanceSynthetic, Guarantee: GuaranteeUndeclared}
	if !reflect.DeepEqual(sink.got[0], want) {
		t.Fatalf("got %+v want %+v", sink.got[0], want)
	}
	if sink.got[1].Status != StatusReplayed || sink.got[2].Operation != "lookup" || sink.got[2].Status != StatusReplayed {
		t.Fatalf("%+v", sink.got)
	}
}

func TestEvidenceStatusesAndErrorClasses(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(*memoryProvider)
		run        func(Runner, Request) (Result, error)
		wantStatus Status
		wantClass  ErrorClass
	}{
		{"denied", func(d *memoryProvider) { d.enabled = false }, runExec, StatusDenied, ErrorClassNone},
		{"conflict", func(d *memoryProvider) { d.version = 5 }, runExec, StatusConflict, ErrorClassNone},
		{"not-recorded", func(*memoryProvider) {}, runLook, StatusNotRecorded, ErrorClassNone},
		{"execution-error", func(d *memoryProvider) { d.failCheck = errors.New(canary) }, runExec, "", ErrorClassExecution},
		{"commit-uncertain", func(d *memoryProvider) { d.commitError = errors.New(canary) }, runExec, "", ErrorClassCommitUncertain},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db := newMemoryProvider()
			c.setup(db)
			sink := &fakeSink{}
			runner := Runner{Provider: db, Audit: AuditConfig{Sink: sink}}
			_, _ = c.run(runner, canaryRequest())
			if len(sink.got) != 1 {
				t.Fatalf("evidence count %d", len(sink.got))
			}
			e := sink.got[0]
			if e.Status != c.wantStatus || e.ErrorClass != c.wantClass || e.Provenance != ProvenanceUnspecified {
				t.Fatalf("%+v", e)
			}
			if s := fmt.Sprintf("%+v %#v", e, e); strings.Contains(s, "CANARY") || strings.Contains(s, "postgres://") {
				t.Fatalf("secret leaked into Evidence: %s", s)
			}
		})
	}
}

func TestEvidenceNeverContainsPayloadOnSuccess(t *testing.T) {
	sink := &fakeSink{}
	runner := Runner{Provider: newMemoryProvider(), Audit: AuditConfig{Sink: sink}}
	if _, err := runExec(runner, canaryRequest()); err != nil {
		t.Fatal(err)
	}
	if s := fmt.Sprintf("%+v %#v", sink.got[0], sink.got[0]); strings.Contains(s, "CANARY") {
		t.Fatalf("payload leaked: %s", s)
	}
}

func TestEvidenceInvalidRequestEchoesNothingUntrusted(t *testing.T) {
	sink := &fakeSink{}
	runner := Runner{Provider: newMemoryProvider(), Audit: AuditConfig{Sink: sink}}
	req := canaryRequest()
	req.Scope.Resource = canary + "\n" // fails validText
	if _, err := runExec(runner, req); !errors.Is(err, ErrInvalidRequest) {
		t.Fatal(err)
	}
	if len(sink.got) != 1 || sink.got[0].ErrorClass != ErrorClassInvalidRequest ||
		sink.got[0].Scope != (Scope{}) || sink.got[0].OperationID != "" {
		t.Fatalf("%+v", sink.got)
	}
	if s := fmt.Sprintf("%+v", sink.got[0]); strings.Contains(s, "CANARY") {
		t.Fatalf("untrusted input leaked: %s", s)
	}
}

func TestEvidenceCarriesDeclaredGuarantee(t *testing.T) {
	sink := &fakeSink{}
	p := declaring{newMemoryProvider(), Guarantee{Level: GuaranteeAtomicWithCommit, Reason: "synthetic"}}
	runner := Runner{Provider: p, Audit: AuditConfig{Sink: sink, Provenance: ProvenanceRealPG}}
	if _, err := runExec(runner, fixtureRequest()); err != nil {
		t.Fatal(err)
	}
	if sink.got[0].Guarantee != GuaranteeAtomicWithCommit || sink.got[0].Provenance != ProvenanceRealPG {
		t.Fatalf("%+v", sink.got[0])
	}
}

func TestNilSinkIdenticalBehaviourAndSinkPanicDoesNotChangeOutcome(t *testing.T) {
	req := canaryRequest()
	a, errA := runExec(Runner{Provider: newMemoryProvider()}, req)
	b, errB := runExec(Runner{Provider: newMemoryProvider(), Audit: AuditConfig{Sink: panicSink{}}}, req)
	if errA != nil || errB != nil || a != b || a.Status != StatusCommitted {
		t.Fatalf("%+v %v / %+v %v", a, errA, b, errB)
	}
}
