//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
)

type recordingLinuxSignerSink struct {
	events    []LinuxSignerAuditEvent
	failPhase string
	onPrepare func()
}

func (s *recordingLinuxSignerSink) RecordSignerChange(_ context.Context, event LinuxSignerAuditEvent) error {
	if event.Phase == s.failPhase {
		return errors.New("audit backend failure: confidential connection token")
	}
	s.events = append(s.events, event)
	if event.Phase == "prepared" && s.onPrepare != nil {
		s.onPrepare()
	}
	return nil
}

func newAuditedSignerFixture(sink *recordingLinuxSignerSink, permitted bool) (*LinuxSignerAuditedAdmin, *LinuxHostSignerTrust) {
	trust := NewLinuxHostSignerTrust()
	return &LinuxSignerAuditedAdmin{
		Admin: LinuxSignerAdmin{Trust: trust, Authorizer: &testSignerAuthorizer{allow: permitted}},
		Sink: sink, HostID: "host/reference",
	}, trust
}

func TestLinuxSignerAuditedAdminRecordsRegistrationAndRevocation(t *testing.T) {
	sink := &recordingLinuxSignerSink{}
	admin, trust := newAuditedSignerFixture(sink, true)
	key := make(ed25519.PublicKey, ed25519.PublicKeySize)
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: key}
	if err := admin.Allow(context.Background(), "host-admin", "op-register", req); err != nil {
		t.Fatal(err)
	}
	if len(sink.events) != 2 || sink.events[0].Phase != "prepared" || sink.events[1].Phase != "applied" {
		t.Fatalf("wrong registration phases: %+v", sink.events)
	}
	epoch, fp := admin.snapshot(req.SignerRef)
	if epoch == 0 || fp == "" || sink.events[1].NewEpoch != epoch || sink.events[1].NewKeySHA256 != fp {
		t.Fatalf("registration lacks verified key fingerprint or epoch: %+v", sink.events[1])
	}
	if sink.events[1].HostID != "host/reference" || sink.events[1].OperationID != "op-register" ||
		sink.events[1].Principal != "host-admin" || sink.events[1].SignerRef != req.SignerRef {
		t.Fatalf("Host audit context was lost: %+v", sink.events[1])
	}
	if sink.events[1].RecordedAt.IsZero() || strings.Contains(sink.events[1].NewKeySHA256, "confidential") {
		t.Fatal("invalid recorded time or leaked sink data")
	}
	changed, err := admin.Revoke(context.Background(), "host-admin", "op-revoke", req.SignerRef)
	if err != nil || !changed {
		t.Fatalf("authorized audited revocation failed: changed=%v err=%v", changed, err)
	}
	if len(sink.events) != 4 || sink.events[2].Phase != "prepared" || sink.events[3].Phase != "applied" ||
		sink.events[3].OldEpoch != epoch || sink.events[3].OldKeySHA256 != fp ||
		sink.events[3].NewEpoch != 0 || sink.events[3].NewKeySHA256 != "" {
		t.Fatalf("revocation evidence was incorrect: %+v", sink.events)
	}
	if trust.current(req.SignerRef, epoch) {
		t.Fatal("revoked signer remains authorized")
	}
	changed, err = admin.Revoke(context.Background(), "host-admin", "op-revoke-missing", req.SignerRef)
	if err != nil || changed || sink.events[len(sink.events)-1].Phase != "unchanged" {
		t.Fatalf("missing signer was reported as changed: changed=%v err=%v events=%+v", changed, err, sink.events)
	}
}

func TestLinuxSignerAuditedAdminRejectsWithoutDurablePreparation(t *testing.T) {
	sink := &recordingLinuxSignerSink{failPhase: "prepared"}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize)}
	err := admin.Allow(context.Background(), "host-admin", "op-no-preparation", req)
	if !errors.Is(err, ErrLinuxSignerAuditUnavailable) {
		t.Fatalf("missing preparation audit did not reject: %v", err)
	}
	if _, _, err := trust.resolve(req.SignerRef); err == nil {
		t.Fatal("Host changed trust without durable preparation")
	}
	other, _ := newAuditedSignerFixture(nil, true)
	if err := other.Allow(context.Background(), "host-admin", "op-missing-sink", req); err == nil {
		t.Fatal("missing audit sink authorized trust change")
	}
	if err := admin.Allow(context.Background(), "host-admin", "", req); err == nil {
		t.Fatal("blank operation identifier was authorized")
	}
	if err := admin.Allow(context.Background(), "host-admin", " bad-op ", req); err == nil {
		t.Fatal("ambiguous operation identifier was authorized")
	}
}

func TestLinuxSignerAuditedAdminReportsPostMutationAuditFailure(t *testing.T) {
	sink := &recordingLinuxSignerSink{failPhase: "applied"}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize)}
	err := admin.Allow(context.Background(), "host-admin", "op-in-doubt", req)
	if !errors.Is(err, ErrLinuxSignerAuditOutcomeUnrecorded) ||
		!errors.Is(err, ErrLinuxSignerAuditUnavailable) ||
		strings.Contains(err.Error(), "confidential connection token") {
		t.Fatalf("post-change audit error did not preserve ambiguity or redaction: %v", err)
	}
	if _, _, err := trust.resolve(req.SignerRef); err != nil {
		t.Fatalf("post-commit audit failure was falsely treated as rollback: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Phase != "prepared" {
		t.Fatalf("incorrect surviving recovery evidence: %+v", sink.events)
	}
}

func TestLinuxSignerAuditedAdminAuditsDenialWithoutMutation(t *testing.T) {
	sink := &recordingLinuxSignerSink{}
	admin, trust := newAuditedSignerFixture(sink, false)
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize)}
	if err := admin.Allow(context.Background(), "host-admin", "op-denied", req); err == nil {
		t.Fatal("Host denied registration without returning an error")
	}
	if len(sink.events) != 1 || sink.events[0].Phase != "denied" ||
		sink.events[0].ReasonCode != "authorization_denied" {
		t.Fatalf("denied operation was not audited: %+v", sink.events)
	}
	if _, _, err := trust.resolve(req.SignerRef); err == nil {
		t.Fatal("unauthorized change modified trust")
	}
}

func TestLinuxSignerAuditedAdminContextCanceledAfterPreparation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sink := &recordingLinuxSignerSink{onPrepare: cancel}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := LinuxSignerAdminRequest{SignerRef: "signed/tool", PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize)}
	err := admin.Allow(ctx, "host-admin", "op-cancel", req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not returned: %v", err)
	}
	if _, _, err := trust.resolve(req.SignerRef); err == nil {
		t.Fatal("canceled operation modified trust after preparing audit")
	}
	if len(sink.events) != 2 || sink.events[0].Phase != "prepared" || sink.events[1].Phase != "failed" ||
		sink.events[1].ReasonCode != "context_canceled" {
		t.Fatalf("canceled audit sequence was incomplete: %+v", sink.events)
	}
}
