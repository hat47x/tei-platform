//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"errors"
	"strings"
	"sync"
	"testing"
)

func signerOperationRequest() LinuxSignerAdminRequest {
	return LinuxSignerAdminRequest{
		SignerRef: "signed/tool",
		PublicKey: make(ed25519.PublicKey, ed25519.PublicKeySize),
	}
}

func TestLinuxSignerOperationReservationBlocksAllowReplayAndChangedPayload(t *testing.T) {
	ctx := context.Background()
	sink := &recordingLinuxSignerSink{}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := signerOperationRequest()
	if err := admin.Allow(ctx, "host-admin", "op-repeat", req); err != nil {
		t.Fatal(err)
	}
	epoch, _ := admin.snapshot(req.SignerRef)
	if epoch == 0 {
		t.Fatal("successful registration did not advance signer generation")
	}
	if err := admin.Allow(ctx, "host-admin", "op-repeat", req); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("replayed allow was not refused: %v", err)
	}
	changedKey := append(ed25519.PublicKey(nil), req.PublicKey...)
	changedKey[0] ^= 1
	if err := admin.Allow(ctx, "host-admin", "op-repeat", LinuxSignerAdminRequest{
		SignerRef: req.SignerRef, PublicKey: changedKey,
	}); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("operation ID was reused with altered key: %v", err)
	}
	if changed, err := admin.Revoke(ctx, "host-admin", "op-repeat", req.SignerRef); changed || !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("same operation ID was reused for revoke: changed=%v err=%v", changed, err)
	}
	if !trust.current(req.SignerRef, epoch) || len(sink.events) != 2 {
		t.Fatalf("replayed operation changed trust or added misleading audit events: %+v", sink.events)
	}
	if changed, err := admin.Revoke(ctx, "host-admin", "op-new", req.SignerRef); !changed || err != nil {
		t.Fatalf("a new authorized revoke was incorrectly blocked: changed=%v err=%v", changed, err)
	}
	if changed, err := admin.Revoke(ctx, "host-admin", "op-new", req.SignerRef); changed || !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("duplicate revocation did not preserve at-most-once semantics: changed=%v err=%v", changed, err)
	}
}

func TestLinuxSignerOperationReservationFailClosedOnUnavailableLedger(t *testing.T) {
	ctx := context.Background()
	sink := &recordingLinuxSignerSink{}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := signerOperationRequest()
	if err := admin.Allow(ctx, "host-admin", "op-first", req); err != nil {
		t.Fatal(err)
	}
	oldEpoch, _ := admin.snapshot(req.SignerRef)
	ledger := admin.Operations.(*recordingLinuxSignerOperationLedger)
	ledger.fail = true
	err := admin.Allow(ctx, "host-admin", "op-ledger-down", req)
	if !errors.Is(err, ErrLinuxSignerOperationReservationUnavailable) ||
		strings.Contains(err.Error(), "confidential connection token") {
		t.Fatalf("ledger failure was not redacted and fail-closed: %v", err)
	}
	if !trust.current(req.SignerRef, oldEpoch) || len(sink.events) != 2 {
		t.Fatal("trust or audit was changed before durable operation reservation")
	}
	admin.Operations = nil
	if err := admin.Allow(ctx, "host-admin", "op-no-ledger", req); err == nil {
		t.Fatal("missing ledger permitted trust mutation")
	}
	var typedNil *recordingLinuxSignerOperationLedger
	admin.Operations = typedNil
	if err := admin.Allow(ctx, "host-admin", "op-typed-nil-ledger", req); err == nil {
		t.Fatal("typed-nil ledger permitted trust mutation")
	}
	if !trust.current(req.SignerRef, oldEpoch) {
		t.Fatal("invalid ledger dependency changed trusted signer")
	}
}

func TestLinuxSignerOperationReservationRetainsAmbiguousOutcome(t *testing.T) {
	ctx := context.Background()
	sink := &recordingLinuxSignerSink{failPhase: "applied"}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := signerOperationRequest()
	if err := admin.Allow(ctx, "host-admin", "op-uncertain", req); !errors.Is(err, ErrLinuxSignerAuditOutcomeUnrecorded) {
		t.Fatalf("expected an unrecorded result: %v", err)
	}
	epoch, _ := admin.snapshot(req.SignerRef)
	if epoch == 0 {
		t.Fatal("underlying mutation did not occur")
	}
	sink.failPhase = ""
	if err := admin.Allow(ctx, "host-admin", "op-uncertain", req); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("uncertain result was blindly retried: %v", err)
	}
	if !trust.current(req.SignerRef, epoch) || len(sink.events) != 1 || sink.events[0].Phase != "prepared" {
		t.Fatalf("the uncertain outcome was altered by replay: %+v", sink.events)
	}
}

func TestLinuxSignerOperationReservationPersistsAfterPreparedAuditFailure(t *testing.T) {
	sink := &recordingLinuxSignerSink{failPhase: "prepared"}
	admin, trust := newAuditedSignerFixture(sink, true)
	req := signerOperationRequest()
	ctx := context.Background()
	if err := admin.Allow(ctx, "host-admin", "op-prepared-unavailable", req); !errors.Is(err, ErrLinuxSignerAuditUnavailable) {
		t.Fatalf("expected prepared audit failure: %v", err)
	}
	sink.failPhase = ""
	if err := admin.Allow(ctx, "host-admin", "op-prepared-unavailable", req); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("unreconciled reservation was incorrectly released: %v", err)
	}
	if _, _, err := trust.resolve(req.SignerRef); err == nil {
		t.Fatal("audit failure allowed trust mutation")
	}
	if err := admin.Allow(ctx, "host-admin", "op-reconciled-new", req); err != nil {
		t.Fatalf("new operation ID cannot proceed after independent remediation: %v", err)
	}
}

func TestLinuxSignerOperationReservationAfterWrapperReplacement(t *testing.T) {
	admin, trust := newAuditedSignerFixture(&recordingLinuxSignerSink{}, true)
	req := signerOperationRequest()
	ctx := context.Background()
	if err := admin.Allow(ctx, "host-admin", "op-first-wrapper", req); err != nil {
		t.Fatal(err)
	}
	oldEpoch, _ := admin.snapshot(req.SignerRef)
	// Another wrapper sharing the same Host-owned ledger cannot reissue the
	// same ID. This does NOT simulate persistence across a Host restart.
	other := &LinuxSignerAuditedAdmin{
		Admin:      LinuxSignerAdmin{Trust: trust, Authorizer: &testSignerAuthorizer{allow: true}},
		Sink:       &recordingLinuxSignerSink{},
		Operations: admin.Operations,
		HostID:     admin.HostID,
	}
	if err := other.Allow(ctx, "host-admin", "op-first-wrapper", req); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("another wrapper replayed the same operation ID: %v", err)
	}
	if !trust.current(req.SignerRef, oldEpoch) {
		t.Fatal("wrapper replacement changed the signer generation")
	}
}

func TestLinuxSignerOperationReservationConcurrentWrappers(t *testing.T) {
	ctx := context.Background()
	ledger := &recordingLinuxSignerOperationLedger{}
	trust := NewLinuxHostSignerTrust()
	req := signerOperationRequest()
	admins := [2]*LinuxSignerAuditedAdmin{}
	for i := range admins {
		admins[i] = &LinuxSignerAuditedAdmin{
			Admin:      LinuxSignerAdmin{Trust: trust, Authorizer: &testSignerAuthorizer{allow: true}},
			Sink:       &recordingLinuxSignerSink{},
			Operations: ledger,
			HostID:     "host/reference",
		}
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, len(admins))
	for _, admin := range admins {
		wg.Add(1)
		go func(a *LinuxSignerAuditedAdmin) {
			defer wg.Done()
			<-start
			results <- a.Allow(ctx, "host-admin", "op-concurrent", req)
		}(admin)
	}
	close(start)
	wg.Wait()
	close(results)
	success, duplicate := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrLinuxSignerOperationReplay):
			duplicate++
		default:
			t.Fatalf("unexpected result from concurrent Host adapters: %v", err)
		}
	}
	_, epoch, err := trust.resolve(req.SignerRef)
	if err != nil || epoch != 1 || success != 1 || duplicate != 1 {
		t.Fatalf("duplicate reservation passed concurrent wrappers: epoch=%d successes=%d duplicates=%d err=%v", epoch, success, duplicate, err)
	}
}

func TestLinuxSignerOperationReservationUnauthorizedRequestCannotSquatID(t *testing.T) {
	ctx := context.Background()
	sink := &recordingLinuxSignerSink{}
	admin, _ := newAuditedSignerFixture(sink, false)
	req := signerOperationRequest()
	if err := admin.Allow(ctx, "host-admin", "op-auth-changed", req); err == nil {
		t.Fatal("unauthorized action was accepted")
	}
	ledger := admin.Operations.(*recordingLinuxSignerOperationLedger)
	if ledger.calls != 0 {
		t.Fatal("unauthorized caller claimed an operation ID")
	}
	auth := admin.Admin.Authorizer.(*testSignerAuthorizer)
	auth.allow = true
	if err := admin.Allow(ctx, "host-admin", "op-auth-changed", req); err != nil {
		t.Fatalf("authorized operation was blocked by earlier unauthorized request: %v", err)
	}
}
