//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func linuxFileOperationFixture(t *testing.T) (string, *LinuxFileSignerOperationLedger) {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	ledger, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		var st syscall.Stat_t
		statErr := syscall.Stat(directory, &st)
		t.Fatalf("Host file ledger directory rejected: %v (mode=%#o uid=%d euid=%d statErr=%v)", err, st.Mode&0777, st.Uid, os.Geteuid(), statErr)
	}
	t.Cleanup(func() { _ = ledger.Close() })
	return directory, ledger
}

func linuxFileOperationRequest() LinuxSignerOperationReservation {
	return LinuxSignerOperationReservation{
		HostID:      "host/reference",
		OperationID: "signed-operation-1",
		Principal:   "host-admin",
		SignerRef:   "signed/tool",
		Action:      "allow",
		KeySHA256:   linuxSignerKeyFingerprint(make(ed25519.PublicKey, ed25519.PublicKeySize)),
	}
}

func TestLinuxFileSignerOperationLedgerPersistsAcrossReopen(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	ctx := context.Background()
	request := linuxFileOperationRequest()
	created, err := ledger.ReserveSignerOperation(ctx, request)
	if err != nil || !created {
		t.Fatalf("first file reservation failed: created=%v err=%v", created, err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	created, err = restored.ReserveSignerOperation(ctx, request)
	if err != nil || created {
		t.Fatalf("reservation was not retained across ledger reopen: created=%v err=%v", created, err)
	}
	modified := request
	modified.Action, modified.KeySHA256 = "revoke", ""
	if created, err := restored.ReserveSignerOperation(ctx, modified); err != nil || created {
		t.Fatalf("reused ID accepted changed action: created=%v err=%v", created, err)
	}
	modified = request
	modified.HostID = "host/another"
	if created, err := restored.ReserveSignerOperation(ctx, modified); err != nil || !created {
		t.Fatalf("different Host ID was incorrectly rejected: created=%v err=%v", created, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 2 {
		t.Fatalf("wrong file-ledger entries: n=%d err=%v", len(entries), err)
	}
	var found bool
	for _, entry := range entries {
		if strings.Contains(entry.Name(), request.OperationID) {
			t.Fatalf("operation identifier leaked into the reservation filename: %q", entry.Name())
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("reservation readable by another user: %v", info.Mode())
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var record linuxFileSignerReservationRecord
		if err := json.Unmarshal(data, &record); err != nil || record.Version != 1 || record.ReservedAt.IsZero() {
			t.Fatalf("invalid on-disk reservation evidence: record=%+v err=%v", record, err)
		}
		if record.OperationID == request.OperationID && record.HostID == request.HostID {
			found = true
			if record.SignerRef != request.SignerRef || record.KeySHA256 != request.KeySHA256 {
				t.Fatal("operation identity lost from on-disk record")
			}
		}
	}
	if !found {
		t.Fatal("Host's original operation record was not found")
	}
}

func TestLinuxFileSignerOperationLedgerAtomicAcrossIndependentInstances(t *testing.T) {
	directory, first := linuxFileOperationFixture(t)
	second, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	const workers = 16
	request := linuxFileOperationRequest()
	var wg sync.WaitGroup
	results := make(chan struct {
		ok  bool
		err error
	}, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			ledger := first
			if index%2 == 1 {
				ledger = second
			}
			ok, err := ledger.ReserveSignerOperation(context.Background(), request)
			results <- struct {
				ok  bool
				err error
			}{ok: ok, err: err}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if result.ok {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("file ledger did not atomically reserve one operation ID: successes=%d", successes)
	}
}

func TestLinuxFileSignerOperationLedgerAuditedAdminRetainsReplay(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	admin, trust := newAuditedSignerFixture(&recordingLinuxSignerSink{}, true)
	admin.Operations = ledger
	request := signerOperationRequest()
	ctx := context.Background()
	if err := admin.Allow(ctx, "host-admin", "file-ledger-once", request); err != nil {
		t.Fatal(err)
	}
	epoch, _ := admin.snapshot(request.SignerRef)
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	other := &LinuxSignerAuditedAdmin{
		Admin:      LinuxSignerAdmin{Trust: trust, Authorizer: &testSignerAuthorizer{allow: true}},
		Sink:       &recordingLinuxSignerSink{},
		Operations: reopened,
		HostID:     admin.HostID,
	}
	if err := other.Allow(ctx, "host-admin", "file-ledger-once", request); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("new Host adapter replayed on-disk operation: %v", err)
	}
	if !trust.current(request.SignerRef, epoch) {
		t.Fatal("replay invalidated the old signature generation")
	}
}

func TestLinuxFileSignerOperationLedgerPreparedAuditFailureStillReserves(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	sink := &recordingLinuxSignerSink{failPhase: "prepared"}
	admin, trust := newAuditedSignerFixture(sink, true)
	admin.Operations = ledger
	req := signerOperationRequest()
	if err := admin.Allow(context.Background(), "host-admin", "file-prepared-failed", req); !errors.Is(err, ErrLinuxSignerAuditUnavailable) {
		t.Fatalf("expected failed prepared record: %v", err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewLinuxFileSignerOperationLedger(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	admin.Operations = reopened
	sink.failPhase = ""
	if err := admin.Allow(context.Background(), "host-admin", "file-prepared-failed", req); !errors.Is(err, ErrLinuxSignerOperationReplay) {
		t.Fatalf("prepared audit failure incorrectly released operation ID: %v", err)
	}
	if _, _, err := trust.resolve(req.SignerRef); err == nil {
		t.Fatal("replayed operation after audit failure changed trust")
	}
}

func TestLinuxFileSignerOperationLedgerRejectsUnsafeDirectoryAndClosedHandle(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	if _, err := NewLinuxFileSignerOperationLedger("relative"); err == nil {
		t.Fatal("relative ledger directory was accepted")
	}
	alias := filepath.Join(t.TempDir(), "ledger-link")
	if err := os.Symlink(directory, alias); err != nil {
		t.Fatal(err)
	}
	if other, err := NewLinuxFileSignerOperationLedger(alias); err == nil {
		_ = other.Close()
		t.Fatal("symlinked operation ledger directory was accepted")
	}
	openDirectory := t.TempDir()
	if err := os.Chmod(openDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	if other, err := NewLinuxFileSignerOperationLedger(openDirectory); err == nil {
		_ = other.Close()
		t.Fatal("world-readable operation ledger directory was accepted")
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	if ok, err := ledger.ReserveSignerOperation(context.Background(), linuxFileOperationRequest()); ok || err == nil {
		t.Fatal("closed ledger permitted a reservation")
	}
}

func TestLinuxFileSignerOperationLedgerRejectsInvalidOperationBeforeWriting(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	base := linuxFileOperationRequest()
	cases := []struct {
		name string
		edit func(*LinuxSignerOperationReservation)
	}{
		{"missing-operation-id", func(r *LinuxSignerOperationReservation) { r.OperationID = "" }},
		{"ambiguous-host-id", func(r *LinuxSignerOperationReservation) { r.HostID = " bad " }},
		{"missing-principal", func(r *LinuxSignerOperationReservation) { r.Principal = "" }},
		{"unknown-action", func(r *LinuxSignerOperationReservation) { r.Action = "rotate" }},
		{"bad-fingerprint", func(r *LinuxSignerOperationReservation) { r.KeySHA256 = "sha256:" + strings.Repeat("z", 64) }},
		{"unexpected-revoke-key", func(r *LinuxSignerOperationReservation) { r.Action = "revoke" }},
		{"blank-signer-reference", func(r *LinuxSignerOperationReservation) { r.SignerRef = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.edit(&r)
			if created, err := ledger.ReserveSignerOperation(context.Background(), r); created || err == nil {
				t.Fatalf("invalid operation modified ledger: created=%v err=%v", created, err)
			}
		})
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid requests unexpectedly left reservation files: entries=%d err=%v", len(entries), err)
	}
}

func TestLinuxFileSignerOperationLedgerRefusesCanceledReservation(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	request := linuxFileOperationRequest()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if created, err := ledger.ReserveSignerOperation(ctx, request); created || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reservation was accepted: created=%v err=%v", created, err)
	}
	if created, err := ledger.ReserveSignerOperation(context.Background(), request); err != nil || !created {
		t.Fatalf("canceled attempt wrongly reserved the operation ID: created=%v err=%v", created, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("canceled attempt changed reservation files: entries=%d err=%v", len(entries), err)
	}
}

func TestLinuxFileSignerOperationLedgerRejectsDirectoryWidenedAfterOpen(t *testing.T) {
	directory, ledger := linuxFileOperationFixture(t)
	if err := os.Chmod(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if created, err := ledger.ReserveSignerOperation(context.Background(), linuxFileOperationRequest()); created || err == nil {
		t.Fatalf("reservation succeeded after the held directory became accessible: created=%v err=%v", created, err)
	}
}
