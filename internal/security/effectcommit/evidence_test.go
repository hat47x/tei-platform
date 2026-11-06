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

func TestEvidenceCarriesProvisionalGrantProvenance(t *testing.T) {
	if ProvenanceProvisionalGrant != "provisional-grant" {
		t.Fatalf("provenance mark changed: %q", ProvenanceProvisionalGrant)
	}
	sink := &fakeSink{}
	runner := Runner{Provider: newMemoryProvider(), Audit: AuditConfig{Sink: sink, Provenance: ProvenanceProvisionalGrant}}
	if _, err := runExec(runner, fixtureRequest()); err != nil {
		t.Fatal(err)
	}
	if len(sink.got) != 1 || sink.got[0].Provenance != ProvenanceProvisionalGrant {
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

func TestEvidenceGuaranteeReasonIsSanitizedAndBounded(t *testing.T) {
	long := strings.Repeat("あ", MaxGuaranteeReasonRunes+50)
	for name, tc := range map[string]struct{ in, want string }{
		"control-chars-removed": {"line1\nline2\x00\x1b[31mred\r\t", "line1line2[31mred"},
		"truncated":             {long, strings.Repeat("あ", MaxGuaranteeReasonRunes)},
		"plain-kept":            {"provisional D10: x", "provisional D10: x"},
	} {
		sink := &fakeSink{}
		p := declaring{newMemoryProvider(), Guarantee{Level: GuaranteeAtomicWithCommit, Reason: tc.in}}
		runner := Runner{Provider: p, Audit: AuditConfig{Sink: sink}}
		if _, err := runExec(runner, fixtureRequest()); err != nil {
			t.Fatal(err)
		}
		if got := sink.got[0].GuaranteeReason; got != tc.want {
			t.Fatalf("%s: reason=%q want %q", name, got, tc.want)
		}
	}
}

func TestEvidenceCarriesDeclaredGuarantee(t *testing.T) {
	sink := &fakeSink{}
	p := declaring{newMemoryProvider(), Guarantee{Level: GuaranteeAtomicWithCommit, Reason: "synthetic"}}
	runner := Runner{Provider: p, Audit: AuditConfig{Sink: sink, Provenance: ProvenanceRealPG}}
	if _, err := runExec(runner, fixtureRequest()); err != nil {
		t.Fatal(err)
	}
	if sink.got[0].Guarantee != GuaranteeAtomicWithCommit || sink.got[0].Provenance != ProvenanceRealPG ||
		sink.got[0].GuaranteeReason != "synthetic" {
		t.Fatalf("%+v", sink.got[0])
	}
}

// A Provider whose Guarantee() panics must not change the outcome of a call
// that already committed, and the Evidence must still be recorded (as
// undeclared). Before the fix the panic escaped Execute after the commit.
func TestPanickingGuaranteeCannotChangeOutcome(t *testing.T) {
	req := canaryRequest()
	plain, errPlain := runExec(Runner{Provider: newMemoryProvider()}, req)
	sink := &fakeSink{}
	var got Result
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("a panicking Guarantee() escaped Execute after the effect committed: %v", r)
			}
		}()
		got, err = runExec(Runner{Provider: panickingDeclarer{newMemoryProvider()}, Audit: AuditConfig{Sink: sink}}, req)
	}()
	if errPlain != nil || err != nil || got != plain || got.Status != StatusCommitted {
		t.Fatalf("outcome changed: %+v %v vs %+v %v", got, err, plain, errPlain)
	}
	if len(sink.got) != 1 || sink.got[0].Guarantee != GuaranteeUndeclared || sink.got[0].Status != StatusCommitted {
		t.Fatalf("want one undeclared committed Evidence, got %+v", sink.got)
	}
}

// NotAtomic is carried in Evidence too, and it is the Provider's own claim.
func TestEvidenceCarriesNotAtomicDeclaration(t *testing.T) {
	sink := &fakeSink{}
	p := &remoteEffectProvider{}
	_, err := runExec(Runner{Provider: p, Audit: AuditConfig{Sink: sink, Provenance: ProvenanceSynthetic}}, fixtureRequest())
	if err == nil {
		t.Fatal("the fake remote Provider's Begin is expected to fail")
	}
	if len(sink.got) != 1 || sink.got[0].Guarantee != GuaranteeNotAtomic || sink.got[0].ErrorClass != ErrorClassExecution {
		t.Fatalf("%+v", sink.got)
	}
}

// Scope fields and the OperationID that pass validation are echoed verbatim:
// Evidence is only as secret-free as the identifiers a caller chooses. This
// test pins that documented limit so it cannot silently change.
func TestEvidenceEchoesValidatedIdentifiersVerbatim(t *testing.T) {
	sink := &fakeSink{}
	req := fixtureRequest()
	req.OperationID = "operation-with-visible-text"
	if _, err := runExec(Runner{Provider: newMemoryProvider(), Audit: AuditConfig{Sink: sink}}, req); err != nil {
		t.Fatal(err)
	}
	if sink.got[0].OperationID != "operation-with-visible-text" {
		t.Fatalf("validated identifiers are expected to be echoed: %+v", sink.got[0])
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
